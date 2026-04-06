package postgres

import (
	"context"
	"fmt"

	"project-neo/shared/model"

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
