package parser

import (
	"context"
	"log/slog"
	"os"
	"time"

	"project-neo/shared/model"
	"project-neo/workers/internal/metrics"

	"github.com/uptrace/bun"
)

const (
	// recoveryStaleness is the minimum age of a stale PENDING message before
	// recovery re-queues it. Must exceed a worst-case in-flight retry ladder
	// (3 attempts with 5s/15s/45s backoff) so a live Process call is never
	// raced; the fingerprint upsert keeps double-processing convergent anyway.
	recoveryStaleness = 5 * time.Minute

	// recoveryMaxConcurrent caps goroutines spawned during a recovery sweep
	// to avoid hammering the LLM provider API.
	recoveryMaxConcurrent = 5

	defaultRecoveryInterval = 5 * time.Minute
)

// StartRecovery sweeps stale PENDING messages immediately and then on a
// ticker (PARSER_RECOVERY_INTERVAL, default 5m). It also backstops rows a
// shutdown left PENDING mid-retry. Blocks until ctx is cancelled.
func StartRecovery(ctx context.Context, db *bun.DB, provider LLMProvider, m *metrics.Parser, logger *slog.Logger) {
	interval := recoveryInterval(logger)
	sweepStalePending(ctx, db, provider, m, logger)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweepStalePending(ctx, db, provider, m, logger)
		}
	}
}

func recoveryInterval(logger *slog.Logger) time.Duration {
	raw := os.Getenv("PARSER_RECOVERY_INTERVAL")
	if raw == "" {
		return defaultRecoveryInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		logger.Warn("recovery: invalid PARSER_RECOVERY_INTERVAL, using default",
			"value", raw, "default", defaultRecoveryInterval)
		return defaultRecoveryInterval
	}
	return d
}

// sweepStalePending re-processes PENDING messages older than recoveryStaleness.
func sweepStalePending(ctx context.Context, db *bun.DB, provider LLMProvider, m *metrics.Parser, logger *slog.Logger) {
	var msgs []*model.Message
	cutoff := time.Now().Add(-recoveryStaleness)

	if err := db.NewSelect().
		Model(&msgs).
		Where("parse_status = ?", model.ParseStatusPending).
		Where("created_at < ?", cutoff).
		Scan(ctx); err != nil {
		if ctx.Err() == nil {
			logger.Error("recovery: query failed", "error", err)
		}
		return
	}

	if len(msgs) == 0 {
		return
	}

	logger.Info("recovery: re-queuing stale messages", "count", len(msgs))

	sem := make(chan struct{}, recoveryMaxConcurrent)
	for _, msg := range msgs {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}
		go func(msg *model.Message) {
			defer func() { <-sem }()
			Process(ctx, msg, db, provider, m, logger)
		}(msg)
	}
}
