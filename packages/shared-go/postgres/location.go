package postgres

import (
	"context"
	"fmt"

	"project-neo/shared/model"
	"project-neo/shared/repository"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type locationRepository struct {
	db *bun.DB
}

func NewLocationRepository(db *bun.DB) repository.LocationRepository {
	return &locationRepository{db: db}
}

func (r *locationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Location, error) {
	loc := new(model.Location)
	err := r.db.NewSelect().Model(loc).Where("loc.id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get location: %w", err)
	}
	return loc, nil
}

func (r *locationRepository) Search(ctx context.Context, query string) ([]*model.Location, error) {
	var locs []*model.Location
	err := r.db.NewSelect().
		Model(&locs).
		Where("loc.name ILIKE ?", "%"+query+"%").
		Limit(20).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("search locations: %w", err)
	}
	return locs, nil
}
