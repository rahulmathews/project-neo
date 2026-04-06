package postgres

import (
	"context"
	"fmt"

	"project-neo/shared/model"
	"project-neo/shared/repository"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type groupRepository struct {
	db *bun.DB
}

func NewGroupRepository(db *bun.DB) repository.GroupRepository {
	return &groupRepository{db: db}
}

func (r *groupRepository) List(ctx context.Context) ([]*model.Group, error) {
	var groups []*model.Group
	err := r.db.NewSelect().Model(&groups).Where("g.is_active = true").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	return groups, nil
}

func (r *groupRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Group, error) {
	group := new(model.Group)
	err := r.db.NewSelect().Model(group).Where("g.id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}
	return group, nil
}

func (r *groupRepository) Create(ctx context.Context, input model.CreateGroupInput) (*model.Group, error) {
	group := &model.Group{
		ID:          uuid.New(),
		Name:        input.Name,
		Description: input.Description,
		IsActive:    true,
	}
	_, err := r.db.NewInsert().Model(group).Returning("*").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return group, nil
}

func (r *groupRepository) ListLocationContexts(ctx context.Context, groupID uuid.UUID) ([]*model.LocationContext, error) {
	var lcs []*model.LocationContext
	err := r.db.NewSelect().Model(&lcs).Where("lc.group_id = ?", groupID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list location contexts: %w", err)
	}
	return lcs, nil
}

func (r *groupRepository) GetLocationContextByID(ctx context.Context, id uuid.UUID) (*model.LocationContext, error) {
	lc := new(model.LocationContext)
	err := r.db.NewSelect().Model(lc).Where("lc.id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get location context: %w", err)
	}
	return lc, nil
}

func (r *groupRepository) UpsertLocationContext(ctx context.Context, input model.UpsertLocationContextInput) (*model.LocationContext, error) {
	lc := &model.LocationContext{
		ID:            uuid.New(),
		GroupID:       input.GroupID,
		LocationAlias: input.LocationAlias,
		LocationName:  input.LocationName,
		LocationID:    &input.LocationID,
	}
	_, err := r.db.NewInsert().
		Model(lc).
		On("CONFLICT (group_id, location_alias) DO UPDATE").
		Set("location_name = EXCLUDED.location_name").
		Set("location_id = EXCLUDED.location_id").
		Set("updated_at = now()").
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("upsert location context: %w", err)
	}
	return lc, nil
}

// GroupStore is a concrete store for direct group upsert operations (used by workers).
type GroupStore struct {
	db *bun.DB
}

func NewGroupStore(db *bun.DB) *GroupStore {
	return &GroupStore{db: db}
}

// UpsertGroup inserts a group by name, or returns the existing ID if one already exists.
// Uses DO UPDATE SET name = EXCLUDED.name (no-op) so RETURNING id works on both paths.
func (s *GroupStore) UpsertGroup(ctx context.Context, name string) (uuid.UUID, error) {
	group := &model.Group{
		ID:       uuid.New(),
		Name:     name,
		IsActive: true,
	}
	_, err := s.db.NewInsert().
		Model(group).
		On("CONFLICT (name) DO UPDATE").
		Set("name = EXCLUDED.name").
		Returning("id").
		Exec(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert group: %w", err)
	}
	return group.ID, nil
}
