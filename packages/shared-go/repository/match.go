package repository

import (
	"context"

	"github.com/google/uuid"
	"project-neo/shared/model"
)

type MatchRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Match, error)
	ListByUser(ctx context.Context, userID uuid.UUID, p model.Pagination) ([]*model.Match, error)
	Create(ctx context.Context, rideID, riderID, driverID uuid.UUID) (*model.Match, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.MatchStatus) (*model.Match, error)
}
