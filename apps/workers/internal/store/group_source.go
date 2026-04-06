package store

import (
	"context"
	"log/slog"
	"time"

	"project-neo/shared/model"
	sharedpostgres "project-neo/shared/postgres"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// GroupSourceReader reads and updates group_source rows for the workers service.
type GroupSourceReader struct {
	store  *sharedpostgres.GroupSourceStore
	logger *slog.Logger
}

func NewGroupSourceReader(db *bun.DB, logger *slog.Logger) *GroupSourceReader {
	return &GroupSourceReader{
		store:  sharedpostgres.NewGroupSourceStore(db),
		logger: logger,
	}
}

// ListActiveWhatsApp returns all active group_sources with source_type = WHATSAPP.
func (r *GroupSourceReader) ListActiveWhatsApp(ctx context.Context) ([]*model.GroupSource, error) {
	return r.store.ListActive(ctx, model.SourceTypeWhatsApp)
}

// TouchLastParsedAt updates last_parsed_at for the given source. Failures are logged
// and swallowed — this is best-effort and must not interrupt message processing.
func (r *GroupSourceReader) TouchLastParsedAt(ctx context.Context, id uuid.UUID) {
	if err := r.store.UpdateLastParsedAt(ctx, id, time.Now()); err != nil {
		r.logger.Warn("failed to update last_parsed_at", "source_id", id, "error", err)
	}
}
