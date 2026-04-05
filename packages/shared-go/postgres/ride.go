package postgres

import (
	"context"
	"fmt"
	"time"

	"project-neo/shared/model"
	"project-neo/shared/repository"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type rideRepository struct {
	db *bun.DB
}

func NewRideRepository(db *bun.DB) repository.RideRepository {
	return &rideRepository{db: db}
}

func (r *rideRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Ride, error) {
	ride := new(model.Ride)
	err := r.db.NewSelect().Model(ride).Where("r.id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get ride: %w", err)
	}
	return ride, nil
}

func (r *rideRepository) List(ctx context.Context, filter model.RideFilter) ([]*model.Ride, error) {
	var rides []*model.Ride
	q := r.db.NewSelect().Model(&rides).Where("r.group_id = ?", filter.GroupID)
	if filter.Type != nil {
		q = q.Where("r.type = ?", *filter.Type)
	}
	if filter.Status != nil {
		q = q.Where("r.status = ?", *filter.Status)
	}
	q = q.OrderExpr("r.created_at DESC").Limit(filter.Limit).Offset(filter.Offset)
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list rides: %w", err)
	}
	return rides, nil
}

func (r *rideRepository) ListByUser(ctx context.Context, userID uuid.UUID, p model.Pagination) ([]*model.Ride, error) {
	var rides []*model.Ride
	err := r.db.NewSelect().
		Model(&rides).
		Where("r.poster_user_id = ?", userID).
		OrderExpr("r.created_at DESC").
		Limit(p.Limit).
		Offset(p.Offset).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rides by user: %w", err)
	}
	return rides, nil
}

func (r *rideRepository) Create(ctx context.Context, userID uuid.UUID, input model.CreateRideInput) (*model.Ride, error) {
	currency := "USD"
	if input.Currency != nil {
		currency = *input.Currency
	}
	ride := &model.Ride{
		ID:               uuid.New(),
		GroupID:          input.GroupID,
		Type:             input.Type,
		FromLocationID:   input.FromLocationContextID,
		ToLocationID:     input.ToLocationContextID,
		FromLocationText: input.FromLocationText,
		ToLocationText:   input.ToLocationText,
		DepartureTime:    input.DepartureTime,
		IsImmediate:      input.IsImmediate,
		Cost:             input.Cost,
		Currency:         currency,
		Distance:         input.Distance,
		SeatsAvailable:   input.SeatsAvailable,
		Status:           model.RideStatusAvailable,
		PosterUserID:     &userID,
	}
	_, err := r.db.NewInsert().Model(ride).Returning("*").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("create ride: %w", err)
	}
	return ride, nil
}

func (r *rideRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input model.UpdateRideInput) (*model.Ride, error) {
	ride, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ride.PosterUserID == nil || *ride.PosterUserID != userID {
		return nil, fmt.Errorf("forbidden")
	}
	if ride.Status != model.RideStatusAvailable {
		return nil, fmt.Errorf("ride is not available for editing")
	}
	q := r.db.NewUpdate().Model(ride).Where("r.id = ?", id).Set("updated_at = now()")
	if input.DepartureTime != nil {
		ride.DepartureTime = input.DepartureTime
		q = q.Set("departure_time = ?", input.DepartureTime)
	}
	if input.Cost != nil {
		ride.Cost = input.Cost
		q = q.Set("cost = ?", input.Cost)
	}
	if input.SeatsAvailable != nil {
		ride.SeatsAvailable = input.SeatsAvailable
		q = q.Set("seats_available = ?", input.SeatsAvailable)
	}
	_, err = q.Returning("*").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update ride: %w", err)
	}
	return ride, nil
}

func (r *rideRepository) Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.Ride, error) {
	ride, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ride.PosterUserID == nil || *ride.PosterUserID != userID {
		return nil, fmt.Errorf("forbidden")
	}
	return r.SetStatus(ctx, id, model.RideStatusCancelled)
}

func (r *rideRepository) SetStatus(ctx context.Context, id uuid.UUID, status model.RideStatus) (*model.Ride, error) {
	ride := new(model.Ride)
	_, err := r.db.NewUpdate().
		Model(ride).
		Set("status = ?", status).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("set ride status: %w", err)
	}
	return ride, nil
}
