package parser

import (
	"context"
	"log/slog"
	"time"

	"project-neo/shared/model"

	"github.com/uptrace/bun"
)

const (
	// recoveryStaleness is the minimum age of a stale PENDING message before
	// recovery re-queues it. Avoids racing with messages mid-first-attempt.
	recoveryStaleness = 2 * time.Minute

	// recoveryMaxConcurrent caps goroutines spawned during startup recovery
	// to avoid hammering the LLM provider API.
	recoveryMaxConcurrent = 5
)

// StartRecovery queries for stale PENDING messages (retry_count > 0, older than
// recoveryStaleness) and re-processes each one in a goroutine. Runs once at startup.
func StartRecovery(ctx context.Context, db *bun.DB, provider LLMProvider, logger *slog.Logger) {
	var msgs []*model.Message
	cutoff := time.Now().Add(-recoveryStaleness)

	if err := db.NewSelect().
		Model(&msgs).
		Where("parse_status = ?", model.ParseStatusPending).
		Where("retry_count > 0").
		Where("created_at < ?", cutoff).
		Scan(ctx); err != nil {
		logger.Error("recovery: query failed", "error", err)
		return
	}

	if len(msgs) == 0 {
		logger.Info("recovery: no stale messages to recover")
		return
	}

	logger.Info("recovery: re-queuing stale messages", "count", len(msgs))

	sem := make(chan struct{}, recoveryMaxConcurrent)
	for _, msg := range msgs {
		sem <- struct{}{}
		go func(m *model.Message) {
			defer func() { <-sem }()
			Process(ctx, m, db, provider, logger)
		}(msg)
	}
}
