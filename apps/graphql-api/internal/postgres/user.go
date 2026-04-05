package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"project-neo/graphql-api/internal/model"
)

type userRepository struct {
	db *bun.DB
}

func NewUserRepository(db *bun.DB) *userRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := new(model.User)
	err := r.db.NewSelect().Model(user).Where("u.id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (r *userRepository) Upsert(ctx context.Context, id uuid.UUID, input model.UpsertUserInput) (*model.User, error) {
	role := model.UserRoleBoth
	if input.Role != nil {
		role = *input.Role
	}

	user := &model.User{
		ID:        id,
		Name:      input.Name,
		Phone:     input.Phone,
		Role:      role,
		AvatarURL: input.AvatarURL,
	}

	_, err := r.db.NewInsert().
		Model(user).
		On("CONFLICT (id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("phone = EXCLUDED.phone").
		Set("role = EXCLUDED.role").
		Set("avatar_url = EXCLUDED.avatar_url").
		Set("updated_at = now()").
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return user, nil
}
