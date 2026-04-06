package internal

import (
	"context"
	"database/sql"
	"log/slog"

	"project-neo/workers/internal/store"
	"project-neo/workers/whatsapp"

	"github.com/uptrace/bun"
)

// Connector is implemented by each platform-specific worker (WhatsApp, Telegram, etc.).
type Connector interface {
	Start(ctx context.Context) error
	Stop() // blocks until all in-flight handlers complete
}

// NewConnectors reads all active WHATSAPP group_sources from the DB and returns
// one Connector that listens to all of them. If no sources are configured,
// an empty slice is returned with no error.
func NewConnectors(ctx context.Context, bunDB *bun.DB, sqlDB *sql.DB, logger *slog.Logger) ([]Connector, error) {
	reader := store.NewGroupSourceReader(bunDB, logger)
	sources, err := reader.ListActiveWhatsApp(ctx)
	if err != nil {
		return nil, err
	}

	if len(sources) == 0 {
		logger.Info("no active WhatsApp sources found — connector not started")
		return nil, nil
	}

	c, err := whatsapp.NewClient(ctx, sources, bunDB, sqlDB, logger)
	if err != nil {
		return nil, err
	}
	return []Connector{c}, nil
}
