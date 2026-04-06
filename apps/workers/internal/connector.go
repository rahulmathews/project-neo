package internal

import (
	"context"
	"database/sql"
	"log/slog"

	"project-neo/shared/model"
	"project-neo/workers/internal/store"
	"project-neo/workers/whatsapp"

	"github.com/uptrace/bun"
)

// Connector is implemented by each platform-specific worker (WhatsApp, Telegram, etc.).
type Connector interface {
	Start(ctx context.Context) error
	Stop() // blocks until all in-flight handlers complete
}

// NewConnectors reads all active WHATSAPP group_sources from the DB, groups them by
// source_identifier (phone number), and returns one Connector per unique number.
func NewConnectors(ctx context.Context, bunDB *bun.DB, sqlDB *sql.DB, logger *slog.Logger) ([]Connector, error) {
	reader := store.NewGroupSourceReader(bunDB, logger)
	sources, err := reader.ListActiveWhatsApp(ctx)
	if err != nil {
		return nil, err
	}

	// Group sources by phone number — one whatsmeow client per number.
	byNumber := make(map[string][]*model.GroupSource)
	for _, s := range sources {
		byNumber[s.SourceIdentifier] = append(byNumber[s.SourceIdentifier], s)
	}

	connectors := make([]Connector, 0, len(byNumber))
	for number, srcs := range byNumber {
		c, err := whatsapp.NewClient(ctx, number, srcs, bunDB, sqlDB, logger)
		if err != nil {
			return nil, err
		}
		connectors = append(connectors, c)
	}
	return connectors, nil
}
