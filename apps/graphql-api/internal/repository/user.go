package repository

import (
	"context"

	"github.com/google/uuid"
	"project-neo/graphql-api/internal/model"
)

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Upsert(ctx context.Context, id uuid.UUID, input model.UpsertUserInput) (*model.User, error)
}
