package repository

import (
	"context"

	"project-neo/shared/model"

	"github.com/google/uuid"
)

type GroupRepository interface {
	List(ctx context.Context) ([]*model.Group, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Group, error)
	Create(ctx context.Context, input model.CreateGroupInput) (*model.Group, error)
	ListLocationContexts(ctx context.Context, groupID uuid.UUID) ([]*model.LocationContext, error)
	GetLocationContextByID(ctx context.Context, id uuid.UUID) (*model.LocationContext, error)
	UpsertLocationContext(ctx context.Context, input model.UpsertLocationContextInput) (*model.LocationContext, error)
}
