package parser

import (
	"context"
	"log/slog"
	"time"

	"project-neo/shared/model"
	sharedpostgres "project-neo/shared/postgres"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// writeRide assembles a model.Ride from the parsed result, inserts it, and
// marks the message as SUCCESS. On any error, marks FAILED.
func writeRide(
	ctx context.Context,
	db *bun.DB,
	msg *model.Message,
	parsed *ParsedRide,
	fromLocationID *uuid.UUID,
	toLocationID *uuid.UUID,
	logger *slog.Logger,
) {
	// Skip if a ride already exists for this content in this group (duplicate message).
	var exists bool
	if err := db.NewSelect().
		TableExpr("rides AS r").
		ColumnExpr("EXISTS (SELECT 1 FROM rides r2 JOIN messages m ON m.id = r2.message_id WHERE m.group_id = ? AND m.content_hash = ?)", msg.GroupID, msg.ContentHash).
		Scan(ctx, &exists); err != nil {
		logger.Warn("writer: duplicate check failed, proceeding", "msg_id", msg.ID, "error", err)
	} else if exists {
		logger.Info("writer: skipping duplicate ride (same content hash in group)", "msg_id", msg.ID)
		markSuccess(ctx, db, msg.ID, logger)
		return
	}

	ride := &model.Ride{
		ID:               uuid.New(),
		MessageID:        &msg.ID,
		GroupID:          msg.GroupID,
		Type:             parsed.RideType,
		FromLocationID:   fromLocationID,
		ToLocationID:     toLocationID,
		FromLocationText: parsed.FromLocationText,
		ToLocationText:   parsed.ToLocationText,
		DepartureTime:    parsed.DepartureTime,
		IsImmediate:      parsed.IsImmediate,
		Cost:             parsed.Cost,
		Currency:         currencyOrDefault(parsed.Currency),
		Distance:         parsed.Distance,
		SeatsAvailable:   parsed.SeatsAvailable,
		Status:           model.RideStatusAvailable,
	}

	rideStore := sharedpostgres.NewRideStore(db)
	if err := rideStore.InsertRide(ctx, ride); err != nil {
		logger.Error("writer: insert ride", "msg_id", msg.ID, "error", err)
		markFailed(ctx, db, msg.ID, "ride insert failed: "+err.Error(), logger)
		return
	}

	markSuccess(ctx, db, msg.ID, logger)
	logger.Info("parser: ride created", "ride_id", ride.ID, "msg_id", msg.ID, "type", ride.Type)
}

func markSuccess(ctx context.Context, db *bun.DB, msgID uuid.UUID, logger *slog.Logger) {
	now := time.Now()
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("parse_status = ?", model.ParseStatusSuccess).
		Set("parsed_at = ?", now).
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: mark success", "msg_id", msgID, "error", err)
	}
}

func markFailed(ctx context.Context, db *bun.DB, msgID uuid.UUID, reason string, logger *slog.Logger) {
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("parse_status = ?", model.ParseStatusFailed).
		Set("parse_error = ?", reason).
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: mark failed", "msg_id", msgID, "error", err)
	}
}

func markSkipped(ctx context.Context, db *bun.DB, msgID uuid.UUID, logger *slog.Logger) {
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("parse_status = ?", model.ParseStatusSkipped).
		Set("parse_error = ?", "not a ride message").
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: mark skipped", "msg_id", msgID, "error", err)
	}
}

func currencyOrDefault(c *string) string {
	if c == nil || *c == "" {
		return "USD"
	}
	return *c
}
