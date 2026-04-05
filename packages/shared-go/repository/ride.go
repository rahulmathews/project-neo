package repository

import (
	"context"

	"github.com/google/uuid"
	"project-neo/shared/model"
)

type RideRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Ride, error)
	List(ctx context.Context, filter model.RideFilter) ([]*model.Ride, error)
	ListByUser(ctx context.Context, userID uuid.UUID, p model.Pagination) ([]*model.Ride, error)
	Create(ctx context.Context, userID uuid.UUID, input model.CreateRideInput) (*model.Ride, error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input model.UpdateRideInput) (*model.Ride, error)
	Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.Ride, error)
	SetStatus(ctx context.Context, id uuid.UUID, status model.RideStatus) (*model.Ride, error)
}
