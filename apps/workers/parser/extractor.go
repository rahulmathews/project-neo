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

		// Step 2: resolve locations (best-effort — never blocks progress)
		fromID := resolveLocation(ctx, db, parsed.FromLocationText, msg.GroupID, logger)
		toID := resolveLocation(ctx, db, parsed.ToLocationText, msg.GroupID, logger)

		// Step 3: write ride
		if err := writeRide(ctx, db, msg, parsed, fromID, toID, m, logger); err != nil {
			logger.Warn("parser: write ride failed", "msg_id", msg.ID, "attempt", attempt, "error", err)
			incrementRetryCount(ctx, db, msg.ID, err.Error(), logger)
			if attempt < maxRetries {
				m.Retries.Inc()
				sleep(ctx, backoff)
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
	parsed, hit := extractWithRegex(msg.Content)
	m.ExtractDuration.WithLabelValues("regex").Observe(time.Since(regexStart).Seconds())
	if hit {
		m.Extractor.WithLabelValues("regex", "matched").Inc()
		return parsed, false
	}
	m.Extractor.WithLabelValues("regex", "miss").Inc()

	llmStart := time.Now()
	var err error
	parsed, err = provider.Extract(ctx, msg.Content, groupName)
	m.ExtractDuration.WithLabelValues("llm").Observe(time.Since(llmStart).Seconds())
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

	m.Extractor.WithLabelValues("llm", "error").Inc()
	logger.Warn("parser: extraction failed", "msg_id", msg.ID, "attempt", attempt, "error", err)
	incrementRetryCount(ctx, db, msg.ID, err.Error(), logger)

	if attempt < maxRetries {
		sleep(ctx, *backoff)
		*backoff *= backoffFactor
		return nil, false
	}

	markFailed(ctx, db, msg.ID, err.Error(), m, logger)
	return nil, true
}

// sleep blocks for d or until ctx is cancelled.
func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
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
