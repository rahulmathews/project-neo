package repository

import (
	"context"

	"project-neo/shared/model"

	"github.com/google/uuid"
)

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Upsert(ctx context.Context, id uuid.UUID, input model.UpsertUserInput) (*model.User, error)
}
