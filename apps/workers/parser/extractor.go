package parser

import (
	"context"
	"errors"
	"log/slog"

	"project-neo/shared/model"

	"github.com/uptrace/bun"
)

// Process runs the full extraction pipeline for a single PENDING message.
// It is safe to call concurrently — each call operates on its own message ID.
func Process(ctx context.Context, msg *model.Message, db *bun.DB, logger *slog.Logger) {
	// Fetch group name for Haiku context (best-effort — empty string on failure)
	groupName := fetchGroupName(ctx, db, msg)

	// Step 1: try regex extraction
	parsed, hit := extractWithRegex(msg.Content)
	if !hit {
		// Step 2: regex miss → try Haiku
		var err error
		parsed, err = extractWithHaiku(ctx, msg.Content, groupName)
		if err != nil {
			if errors.Is(err, ErrNotARide) {
				logger.Info("parser: skipped (not a ride)", "msg_id", msg.ID)
				markSkipped(ctx, db, msg.ID, logger)
				return
			}
			logger.Warn("parser: extraction failed", "msg_id", msg.ID, "error", err)
			markFailed(ctx, db, msg.ID, err.Error(), logger)
			return
		}
	}

	// Step 3: resolve locations
	fromID := resolveLocation(ctx, db, parsed.FromLocationText, msg.GroupID, logger)
	toID := resolveLocation(ctx, db, parsed.ToLocationText, msg.GroupID, logger)

	// Step 4: write ride + update message status
	writeRide(ctx, db, msg, parsed, fromID, toID, logger) //nolint:errcheck,gosec // error handling added in Task 5 retry loop
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
