package parser

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"project-neo/shared/model"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// resolveLocation looks up locationText in the location_contexts table for
// the given group. Returns lc.ID (the location_contexts PK) if found, nil otherwise.
// A nil result means the raw text will be stored as-is on the ride row.
func resolveLocation(ctx context.Context, db *bun.DB, locationText *string, groupID uuid.UUID, logger *slog.Logger) *uuid.UUID {
	if locationText == nil || *locationText == "" {
		return nil
	}

	var lc model.LocationContext
	err := db.NewSelect().
		Model(&lc).
		Where("group_id = ?", groupID).
		Where("LOWER(location_alias) = LOWER(?)", *locationText).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			logger.Warn("location resolve error", "text", *locationText, "group_id", groupID, "error", err)
		}
		return nil
	}

	return &lc.ID
}
