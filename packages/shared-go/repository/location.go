package repository

import (
	"context"

	"project-neo/shared/model"

	"github.com/google/uuid"
)

type LocationRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Location, error)
	Search(ctx context.Context, query string) ([]*model.Location, error)
}
