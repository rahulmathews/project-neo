package postgres

import (
	"context"
	"fmt"
	"time"

	"project-neo/shared/model"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type GroupSourceStore struct {
	db *bun.DB
}

func NewGroupSourceStore(db *bun.DB) *GroupSourceStore {
	return &GroupSourceStore{db: db}
}

// ListActive returns all active group_sources for a given source type.
func (s *GroupSourceStore) ListActive(ctx context.Context, sourceType model.SourceType) ([]*model.GroupSource, error) {
	var sources []*model.GroupSource
	err := s.db.NewSelect().
		Model(&sources).
		Where("gs.source_type = ?", sourceType).
		Where("gs.is_active = true").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active group sources: %w", err)
	}
	return sources, nil
}

// UpdateLastParsedAt sets the last_parsed_at timestamp for a group_source row.
func (s *GroupSourceStore) UpdateLastParsedAt(ctx context.Context, id uuid.UUID, t time.Time) error {
	_, err := s.db.NewUpdate().
		Model((*model.GroupSource)(nil)).
		Set("last_parsed_at = ?", t).
		Set("updated_at = now()").
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update last_parsed_at: %w", err)
	}
	return nil
}
