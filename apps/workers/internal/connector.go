package internal

import (
	"context"
	"log/slog"
	"os"

	sharedpostgres "project-neo/shared/postgres"
	"project-neo/workers/whatsapp"

	"github.com/uptrace/bun"
)

// Connector is implemented by each platform-specific worker (WhatsApp, Telegram, etc.).
type Connector interface {
	Start(ctx context.Context) error
	Stop() // blocks until all in-flight handlers complete
}

// NewConnectors creates all platform connectors. The WhatsApp connector is always
// started — it handles QR pairing on first run and silent session resume thereafter.
func NewConnectors(ctx context.Context, bunDB *bun.DB, logger *slog.Logger) ([]Connector, error) {
	groupStore := sharedpostgres.NewGroupStore(bunDB)
	groupSourceStore := sharedpostgres.NewGroupSourceStore(bunDB)

	sessionPath := os.Getenv("WHATSAPP_SESSION_PATH")
	if sessionPath == "" {
		sessionPath = "whatsapp.db"
	}

	c, err := whatsapp.NewClient(ctx, groupStore, groupSourceStore, bunDB, sessionPath, logger)
	if err != nil {
		return nil, err
	}
	return []Connector{c}, nil
}
