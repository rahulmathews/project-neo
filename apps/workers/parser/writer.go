package parser

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"project-neo/shared/model"
	sharedpostgres "project-neo/shared/postgres"
	"project-neo/workers/internal/metrics"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// writeRide upserts a canonical ride keyed by semantic fingerprint, then
// links the source message to it via messages.ride_id. Two messages with the
// same fingerprint converge on the same canonical ride row.
func writeRide(
	ctx context.Context,
	db *bun.DB,
	msg *model.Message,
	parsed *ParsedRide,
	fromLocationID *uuid.UUID,
	toLocationID *uuid.UUID,
	m *metrics.Parser,
	logger *slog.Logger,
) error {
	fingerprint := rideSemanticFingerprint(ctx, db, msg, parsed)

	ride := &model.Ride{
		ID:                  uuid.New(),
		MessageID:           &msg.ID,
		GroupID:             msg.GroupID,
		Type:                parsed.RideType,
		FromLocationID:      fromLocationID,
		ToLocationID:        toLocationID,
		FromLocationText:    parsed.FromLocationText,
		ToLocationText:      parsed.ToLocationText,
		DepartureTime:       parsed.DepartureTime,
		IsImmediate:         parsed.IsImmediate,
		Cost:                parsed.Cost,
		Currency:            currencyOrDefault(parsed.Currency),
		Distance:            parsed.Distance,
		SeatsAvailable:      parsed.SeatsAvailable,
		Status:              model.RideStatusAvailable,
		SemanticFingerprint: &fingerprint,
		FingerprintVersion:  rideFingerprintVersion,
	}

	rideStore := sharedpostgres.NewRideStore(db)
	canonicalRide, inserted, err := rideStore.UpsertCanonicalRide(ctx, ride)
	if err != nil {
		logger.Error("writer: upsert canonical ride", "msg_id", msg.ID, "error", err)
		return fmt.Errorf("ride upsert: %w", err)
	}

	if err := linkMessageToRide(ctx, db, msg.ID, canonicalRide.ID); err != nil {
		logger.Error("writer: link message to ride", "msg_id", msg.ID, "ride_id", canonicalRide.ID, "error", err)
		return fmt.Errorf("link message to ride: %w", err)
	}

	markSuccess(ctx, db, msg.ID, m, logger)
	logger.Info("parser: ride linked", "ride_id", canonicalRide.ID, "msg_id", msg.ID, "type", canonicalRide.Type, "canonical_inserted", inserted)
	return nil
}

// linkMessageToRide sets messages.ride_id linking this message to its canonical ride.
func linkMessageToRide(ctx context.Context, db *bun.DB, msgID, rideID uuid.UUID) error {
	_, err := db.NewUpdate().
		TableExpr("messages").
		Set("ride_id = ?", rideID).
		Where("id = ?", msgID).
		Exec(ctx)
	return err
}

func incrementRetryCount(ctx context.Context, db *bun.DB, msgID uuid.UUID, reason string, logger *slog.Logger) {
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("retry_count = retry_count + 1").
		Set("parse_error = ?", reason).
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: increment retry_count", "msg_id", msgID, "error", err)
	}
}

func markSuccess(ctx context.Context, db *bun.DB, msgID uuid.UUID, m *metrics.Parser, logger *slog.Logger) {
	now := time.Now()
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("parse_status = ?", model.ParseStatusSuccess).
		Set("parsed_at = ?", now).
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: mark success", "msg_id", msgID, "error", err)
		return
	}
	m.Messages.WithLabelValues("success").Inc()
}

func markFailed(ctx context.Context, db *bun.DB, msgID uuid.UUID, reason string, m *metrics.Parser, logger *slog.Logger) {
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("parse_status = ?", model.ParseStatusFailed).
		Set("parse_error = ?", reason).
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: mark failed", "msg_id", msgID, "error", err)
		return
	}
	m.Messages.WithLabelValues("failed").Inc()
}

func markSkipped(ctx context.Context, db *bun.DB, msgID uuid.UUID, m *metrics.Parser, logger *slog.Logger) {
	if _, err := db.NewUpdate().
		TableExpr("messages").
		Set("parse_status = ?", model.ParseStatusSkipped).
		Set("parse_error = ?", "not a ride message").
		Where("id = ?", msgID).
		Exec(ctx); err != nil {
		logger.Error("writer: mark skipped", "msg_id", msgID, "error", err)
		return
	}
	m.Messages.WithLabelValues("skipped").Inc()
}

func currencyOrDefault(c *string) string {
	if c == nil || *c == "" {
		return "USD"
	}
	return *c
}
