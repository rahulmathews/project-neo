package parser

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"project-neo/shared/model"
	sharedpostgres "project-neo/shared/postgres"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// writeRide assembles a model.Ride from the parsed result and inserts it.
// On success it marks the message SUCCESS and returns nil.
// On insert failure it returns the error — the caller owns markFailed.
func writeRide(
	ctx context.Context,
	db *bun.DB,
	msg *model.Message,
	parsed *ParsedRide,
	fromLocationID *uuid.UUID,
	toLocationID *uuid.UUID,
	logger *slog.Logger,
) error {
	var exists bool
	if err := db.NewSelect().
		ColumnExpr("EXISTS (SELECT 1 FROM rides r2 JOIN messages m ON m.id = r2.message_id WHERE m.group_id = ? AND m.content_hash = ?)", msg.GroupID, msg.ContentHash).
		Scan(ctx, &exists); err != nil {
		logger.Warn("writer: duplicate check failed, proceeding", "msg_id", msg.ID, "error", err)
	} else if exists {
		logger.Info("writer: skipping duplicate ride (same content hash in group)", "msg_id", msg.ID)
		markSuccess(ctx, db, msg.ID, logger)
		return nil
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
		return fmt.Errorf("ride insert: %w", err)
	}

	markSuccess(ctx, db, msg.ID, logger)
	logger.Info("parser: ride created", "ride_id", ride.ID, "msg_id", msg.ID, "type", ride.Type)
	return nil
}

func incrementRetryCount(ctx context.Context, db *bun.DB, msgID uuid.UUID, reason string, logger *slog.Logger) { //nolint:unused // used in Task 5 retry loop
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("retry_count = retry_count + 1").
		Set("parse_error = ?", reason).
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: increment retry_count", "msg_id", msgID, "error", err)
	}
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
