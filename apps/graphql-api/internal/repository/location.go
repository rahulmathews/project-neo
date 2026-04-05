package repository

import (
	"context"

	"github.com/google/uuid"
	"project-neo/graphql-api/internal/model"
)

type LocationRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Location, error)
	Search(ctx context.Context, query string) ([]*model.Location, error)
}
