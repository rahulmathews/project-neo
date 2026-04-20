package parser

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"project-neo/shared/model"

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
func Process(ctx context.Context, msg *model.Message, db *bun.DB, logger *slog.Logger) {
	groupName := fetchGroupName(ctx, db, msg)

	backoff := baseBackoff
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Step 1: extract
		parsed, done := extract(ctx, db, msg, groupName, attempt, &backoff, logger)
		if done && parsed == nil {
			return
		}
		if parsed == nil {
			continue
		}

		// Step 2: resolve locations (best-effort — never blocks progress)
		fromID := resolveLocation(ctx, db, parsed.FromLocationText, msg.GroupID, logger)
		toID := resolveLocation(ctx, db, parsed.ToLocationText, msg.GroupID, logger)

		// Step 3: write ride
		if err := writeRide(ctx, db, msg, parsed, fromID, toID, logger); err != nil {
			logger.Warn("parser: write ride failed", "msg_id", msg.ID, "attempt", attempt, "error", err)
			incrementRetryCount(ctx, db, msg.ID, err.Error(), logger)
			if attempt < maxRetries {
				sleep(ctx, backoff)
				backoff *= backoffFactor
				continue
			}
			markFailed(ctx, db, msg.ID, err.Error(), logger)
			return
		}

		return // success
	}
}

// extract attempts regex extraction, falling back to Haiku.
// Returns (parsed, done): if done=true+parsed=nil → caller should return;
// if done=false+parsed=nil → caller should continue to next attempt.
func extract(
	ctx context.Context,
	db *bun.DB,
	msg *model.Message,
	groupName string,
	attempt int,
	backoff *time.Duration,
	logger *slog.Logger,
) (*ParsedRide, bool) {
	parsed, hit := extractWithRegex(msg.Content)
	if hit {
		return parsed, false
	}

	var err error
	parsed, err = extractWithHaiku(ctx, msg.Content, groupName, logger)
	if err == nil {
		return parsed, false
	}

	if errors.Is(err, ErrNotARide) {
		logger.Info("parser: skipped (not a ride)", "msg_id", msg.ID)
		markSkipped(ctx, db, msg.ID, logger)
		return nil, true
	}

	logger.Warn("parser: extraction failed", "msg_id", msg.ID, "attempt", attempt, "error", err)
	incrementRetryCount(ctx, db, msg.ID, err.Error(), logger)

	if attempt < maxRetries {
		sleep(ctx, *backoff)
		*backoff *= backoffFactor
		return nil, false
	}

	markFailed(ctx, db, msg.ID, err.Error(), logger)
	return nil, true
}

// sleep blocks for d or until ctx is cancelled.
func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// fetchGroupName queries the group name for Haiku context. Returns empty string on error.
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
