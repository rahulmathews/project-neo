package internal

import (
	"context"
	"log/slog"
	"os"
	"time"

	sharedpostgres "project-neo/shared/postgres"
	"project-neo/workers/whatsapp"

	"github.com/uptrace/bun"
)

// Connector is implemented by each platform-specific worker (WhatsApp, Telegram, etc.).
type Connector interface {
	// Run blocks while the connector is live and returns when the connection
	// ends: nil when the account was unlinked (re-pair needed), an error for
	// failures, ctx.Err() on shutdown.
	Run(ctx context.Context) error
	Stop() // blocks until all in-flight handlers complete
}

const (
	supervisorBaseBackoff = 5 * time.Second
	supervisorMaxBackoff  = 5 * time.Minute
)

// Supervisor owns the lifecycle of the platform connectors. Construction and
// connection failures are retried with backoff instead of crashing the
// service: a fresh deployment with no WhatsApp session must stay up (and
// healthy) while an operator pairs via the QR code in the logs.
type Supervisor struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// StartConnectorSupervisor launches the WhatsApp connector loop in the
// background. onStatus receives coarse connector state transitions for the
// health endpoint ("connecting", "awaiting_pairing", "connected", ...).
func StartConnectorSupervisor(ctx context.Context, bunDB *bun.DB, logger *slog.Logger, onStatus func(string)) *Supervisor {
	ctx, cancel := context.WithCancel(ctx)
	s := &Supervisor{cancel: cancel, done: make(chan struct{})}
	go s.superviseWhatsApp(ctx, bunDB, logger, onStatus)
	return s
}

// Stop shuts the supervised connector down and waits for it to finish.
func (s *Supervisor) Stop() {
	s.cancel()
	<-s.done
}

func (s *Supervisor) superviseWhatsApp(ctx context.Context, bunDB *bun.DB, logger *slog.Logger, onStatus func(string)) {
	defer close(s.done)

	groupStore := sharedpostgres.NewGroupStore(bunDB)
	groupSourceStore := sharedpostgres.NewGroupSourceStore(bunDB)

	sessionPath := os.Getenv("WHATSAPP_SESSION_PATH")
	if sessionPath == "" {
		sessionPath = "whatsapp.db"
	}

	backoff := supervisorBaseBackoff
	for {
		// The client is rebuilt every iteration: whatsmeow clears its session
		// store when the phone unlinks the device, so a rebuild after logout
		// lands in the QR pairing flow automatically.
		runErr := runWhatsAppOnce(ctx, groupStore, groupSourceStore, bunDB, sessionPath, logger, onStatus)
		if ctx.Err() != nil {
			return
		}
		if runErr != nil {
			backoff = min(backoff*2, supervisorMaxBackoff)
			logger.Warn("whatsapp connector failed", "error", runErr, "retry_in", backoff)
			onStatus("reconnecting: " + runErr.Error())
		} else {
			backoff = supervisorBaseBackoff
			logger.Info("whatsapp connector stopped, restarting", "retry_in", backoff)
			onStatus("restarting")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
	}
}

// runWhatsAppOnce builds, runs, and tears down one WhatsApp client lifetime.
func runWhatsAppOnce(
	ctx context.Context,
	groupStore *sharedpostgres.GroupStore,
	groupSourceStore *sharedpostgres.GroupSourceStore,
	bunDB *bun.DB,
	sessionPath string,
	logger *slog.Logger,
	onStatus func(string),
) error {
	c, err := whatsapp.NewClient(ctx, groupStore, groupSourceStore, bunDB, sessionPath, logger, onStatus)
	if err != nil {
		onStatus("error: " + err.Error())
		return err
	}
	defer c.Stop()
	return c.Run(ctx)
}
