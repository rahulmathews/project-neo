package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"project-neo/shared/model"
)

type matchRepository struct {
	db *bun.DB
}

func NewMatchRepository(db *bun.DB) *matchRepository {
	return &matchRepository{db: db}
}

func (r *matchRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Match, error) {
	match := new(model.Match)
	err := r.db.NewSelect().Model(match).Where("m.id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get match: %w", err)
	}
	return match, nil
}

func (r *matchRepository) ListByUser(ctx context.Context, userID uuid.UUID, p model.Pagination) ([]*model.Match, error) {
	var matches []*model.Match
	err := r.db.NewSelect().
		Model(&matches).
		Where("m.rider_id = ? OR m.driver_id = ?", userID, userID).
		OrderExpr("m.matched_at DESC").
		Limit(p.Limit).
		Offset(p.Offset).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list matches by user: %w", err)
	}
	return matches, nil
}

func (r *matchRepository) Create(ctx context.Context, rideID, riderID, driverID uuid.UUID) (*model.Match, error) {
	match := &model.Match{
		ID:       uuid.New(),
		RideID:   rideID,
		RiderID:  riderID,
		DriverID: driverID,
		Status:   model.MatchStatusPending,
	}
	_, err := r.db.NewInsert().Model(match).Returning("*").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("create match: %w", err)
	}
	return match, nil
}

func (r *matchRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.MatchStatus) (*model.Match, error) {
	match := new(model.Match)
	q := r.db.NewUpdate().
		Model(match).
		Set("status = ?", status).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id)

	switch status {
	case model.MatchStatusAccepted:
		now := time.Now()
		q = q.Set("accepted_at = ?", now)
	case model.MatchStatusCompleted:
		now := time.Now()
		q = q.Set("completed_at = ?", now)
	case model.MatchStatusCancelled, model.MatchStatusRejected:
		now := time.Now()
		q = q.Set("cancelled_at = ?", now)
	}

	_, err := q.Returning("*").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update match status: %w", err)
	}
	return match, nil
}
