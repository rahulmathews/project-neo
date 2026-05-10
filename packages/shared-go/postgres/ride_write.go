package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"project-neo/shared/model"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// RideStore is a concrete store for ride inserts (used by workers).
type RideStore struct {
	db *bun.DB
}

func NewRideStore(db *bun.DB) *RideStore {
	return &RideStore{db: db}
}

// InsertRide inserts a new ride row. The insert fires ride_added_trigger →
// pg_notify('rides_added', ...) → graphql-api subscribers notified automatically.
func (s *RideStore) InsertRide(ctx context.Context, ride *model.Ride) error {
	_, err := s.db.NewInsert().Model(ride).Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert ride: %w", err)
	}
	return nil
}

// UpsertCanonicalRide inserts a canonical ride, or returns the existing ride with
// the same semantic fingerprint. The caller should always record a RideOccurrence.
func (s *RideStore) UpsertCanonicalRide(ctx context.Context, ride *model.Ride) (*model.Ride, bool, error) {
	if ride.SemanticFingerprint == nil || *ride.SemanticFingerprint == "" {
		if err := s.InsertRide(ctx, ride); err != nil {
			return nil, false, err
		}
		return ride, true, nil
	}

	res, err := s.db.NewInsert().
		Model(ride).
		On("CONFLICT (semantic_fingerprint) WHERE semantic_fingerprint IS NOT NULL DO NOTHING").
		Exec(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("upsert canonical ride: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, false, fmt.Errorf("canonical ride rows affected: %w", err)
	}
	if rowsAffected > 0 {
		return ride, true, nil
	}

	existing, err := s.GetBySemanticFingerprint(ctx, *ride.SemanticFingerprint)
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		return nil, false, fmt.Errorf("canonical ride not found after conflict: %s", *ride.SemanticFingerprint)
	}
	return existing, false, nil
}

func (s *RideStore) GetBySemanticFingerprint(ctx context.Context, fingerprint string) (*model.Ride, error) {
	ride := new(model.Ride)
	err := s.db.NewSelect().
		Model(ride).
		Where("semantic_fingerprint = ?", fingerprint).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get canonical ride: %w", err)
	}
	return ride, nil
}

// InsertRideOccurrence records every message/group occurrence for a canonical ride.
func (s *RideStore) InsertRideOccurrence(ctx context.Context, occurrence *model.RideOccurrence) (bool, error) {
	if occurrence.ID == uuid.Nil {
		occurrence.ID = uuid.New()
	}
	res, err := s.db.NewInsert().
		Model(occurrence).
		On("CONFLICT (message_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("insert ride occurrence: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("ride occurrence rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}
