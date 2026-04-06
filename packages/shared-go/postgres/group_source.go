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
		Where("gs.is_active = ?", true).
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

// UpsertGroupSource inserts a group_source row, or updates group_id and is_active if the
// (source_type, source_identifier) pair already exists.
func (s *GroupSourceStore) UpsertGroupSource(ctx context.Context, groupID uuid.UUID, sourceType model.SourceType, sourceIdentifier string) (uuid.UUID, error) {
	src := &model.GroupSource{
		ID:               uuid.New(),
		GroupID:          groupID,
		SourceType:       sourceType,
		SourceIdentifier: sourceIdentifier,
		IsActive:         true,
	}
	if err := s.db.NewInsert().
		Model(src).
		On("CONFLICT (source_type, source_identifier) DO UPDATE").
		Set("group_id = EXCLUDED.group_id").
		Set("is_active = true").
		Set("updated_at = now()").
		Returning("id").
		Scan(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("upsert group source: %w", err)
	}
	return src.ID, nil
}
