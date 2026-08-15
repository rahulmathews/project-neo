package parser

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"project-neo/shared/model"
	"project-neo/workers/internal/metrics"

	"github.com/uptrace/bun"
)

const (
	maxRetries    = 3
	baseBackoff   = 5 * time.Second
	backoffFactor = 3
)

// Process runs the full extraction pipeline for a single PENDING message,
// retrying up to maxRetries times on transient errors with exponential backoff.
// It is safe to call concurrently — each call operates on its own message.
func Process(ctx context.Context, msg *model.Message, db *bun.DB, provider LLMProvider, m *metrics.Parser, logger *slog.Logger) {
	groupName := fetchGroupName(ctx, db, msg)

	backoff := baseBackoff
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Step 1: extract
		parsed, done := extract(ctx, db, msg, groupName, attempt, &backoff, provider, m, logger)
		if done && parsed == nil {
			return
		}
		if parsed == nil {
			m.Retries.Inc()
			continue
		}
		parsed.FromLocationText = cleanOptionalLocationText(parsed.FromLocationText)
		parsed.ToLocationText = cleanOptionalLocationText(parsed.ToLocationText)

		// Step 2: resolve locations (best-effort — never blocks progress)
		fromID := resolveLocation(ctx, db, parsed.FromLocationText, msg.GroupID, logger)
		toID := resolveLocation(ctx, db, parsed.ToLocationText, msg.GroupID, logger)

		// Step 3: write ride
		if err := writeRide(ctx, db, msg, parsed, fromID, toID, m, logger); err != nil {
			if ctx.Err() != nil {
				return // shutdown mid-write — row stays PENDING for recovery
			}
			logger.Warn("parser: write ride failed", "msg_id", msg.ID, "attempt", attempt, "error", err)
			incrementRetryCount(ctx, db, msg.ID, err.Error(), logger)
			if attempt < maxRetries {
				m.Retries.Inc()
				if !sleep(ctx, backoff) {
					return
				}
				backoff *= backoffFactor
				continue
			}
			markFailed(ctx, db, msg.ID, err.Error(), m, logger)
			return
		}

		return // success
	}
}

// extract attempts regex extraction, falling back to the LLM provider.
// Returns (parsed, done): if done=true+parsed=nil → caller should return;
// if done=false+parsed=nil → caller should continue to next attempt.
func extract(
	ctx context.Context,
	db *bun.DB,
	msg *model.Message,
	groupName string,
	attempt int,
	backoff *time.Duration,
	provider LLMProvider,
	m *metrics.Parser,
	logger *slog.Logger,
) (*ParsedRide, bool) {
	regexStart := time.Now()
	parsed, hit := extractWithRegex(msg.Content, msg.Timestamp)
	m.ExtractDuration.WithLabelValues("regex").Observe(time.Since(regexStart).Seconds())
	if hit {
		m.Extractor.WithLabelValues("regex", "matched").Inc()
		return parsed, false
	}
	m.Extractor.WithLabelValues("regex", "miss").Inc()
	if isClearlyNonRide(msg.Content) {
		m.Extractor.WithLabelValues("regex", "not_a_ride").Inc()
		logger.Info("parser: skipped (blocked non-ride topic)", "msg_id", msg.ID)
		markSkipped(ctx, db, msg.ID, m, logger)
		return nil, true
	}

	llmStart := time.Now()
	var err error
	parsed, err = provider.Extract(ctx, msg.Content, groupName)
	m.ExtractDuration.WithLabelValues("llm").Observe(time.Since(llmStart).Seconds())
	if ctx.Err() != nil {
		// Shutdown mid-extraction: a cancelled LLM call must not be recorded
		// as an outage or failure — leave the row PENDING for recovery.
		return nil, true
	}
	if err == nil {
		m.Extractor.WithLabelValues("llm", "success").Inc()
		return parsed, false
	}

	if errors.Is(err, ErrNotARide) {
		m.Extractor.WithLabelValues("llm", "not_a_ride").Inc()
		logger.Info("parser: skipped (not a ride)", "msg_id", msg.ID)
		markSkipped(ctx, db, msg.ID, m, logger)
		return nil, true
	}

	if errors.Is(err, ErrLLMDisabled) {
		// Regex-only mode: a miss here is a real parse failure to review, not an
		// outage. The parse_error must point at the pattern gap, not at Ollama.
		m.Extractor.WithLabelValues("llm", "disabled").Inc()
		logger.Info("parser: no regex pattern matched (regex-only mode)", "msg_id", msg.ID)
		markFailed(ctx, db, msg.ID, "no regex pattern matched (regex-only mode)", m, logger)
		return nil, true
	}

	if errors.Is(err, ErrLLMUnavailable) {
		// Retrying cannot help while the provider is down or disabled — fail
		// fast with a clear parse_error instead of burning the retry budget.
		m.Extractor.WithLabelValues("llm", "unavailable").Inc()
		logger.Warn("parser: llm unavailable, failing fast", "msg_id", msg.ID, "error", err)
		markFailed(ctx, db, msg.ID,
			"llm unavailable — start Ollama or set OLLAMA_ENABLED=false ("+err.Error()+")",
			m, logger)
		return nil, true
	}

	m.Extractor.WithLabelValues("llm", "error").Inc()
	logger.Warn("parser: extraction failed", "msg_id", msg.ID, "attempt", attempt, "error", err)
	incrementRetryCount(ctx, db, msg.ID, err.Error(), logger)

	if attempt < maxRetries {
		if !sleep(ctx, *backoff) {
			return nil, true
		}
		*backoff *= backoffFactor
		return nil, false
	}

	markFailed(ctx, db, msg.ID, err.Error(), m, logger)
	return nil, true
}

// sleep blocks for d or until ctx is cancelled; reports whether the full
// backoff elapsed. A false return means shutdown — callers must bail out and
// leave the row PENDING for the recovery sweep instead of burning attempts.
func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// fetchGroupName queries the group name for LLM provider context. Returns empty string on error.
func fetchGroupName(ctx context.Context, db *bun.DB, msg *model.Message) string {
	var g model.Group
	if err := db.NewSelect().
		Model(&g).
		Column("name").
		Where("id = ?", msg.GroupID).
		Scan(ctx); err != nil {
		return ""
	}
	return g.Name
}
