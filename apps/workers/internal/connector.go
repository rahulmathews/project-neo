package internal

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	sharedpostgres "project-neo/shared/postgres"
	"project-neo/workers/whatsapp"

	"github.com/uptrace/bun"
)

// Connector is implemented by each platform-specific worker (WhatsApp, Telegram, etc.).
type Connector interface {
	// Name identifies the connector in health output and logs.
	Name() string
	// Run blocks while the connector is live and returns when the connection
	// ends: nil when the account was unlinked (re-pair needed), an error for
	// failures, ctx.Err() on shutdown.
	Run(ctx context.Context) error
	Stop() // blocks until all in-flight handlers complete
}

var _ Connector = (*whatsapp.Client)(nil)

const (
	supervisorBaseBackoff = 5 * time.Second
	supervisorMaxBackoff  = 5 * time.Minute
)

// Registration describes how to construct one supervised connector. Adding a
// platform means adding one entry to registrations(): a Build func returning a
// type satisfying Connector that ingests via store.MessageWriter — the parser
// is source-agnostic past the messages insert.
type Registration struct {
	Name    string
	Enabled func() bool
	Build   func(ctx context.Context, onStatus func(string)) (Connector, error)
}

// Supervisor owns the lifecycle of all registered platform connectors.
// Construction and connection failures are retried with backoff instead of
// crashing the service: a fresh deployment with no WhatsApp session must stay
// up (and healthy) while an operator pairs via the QR code in the logs.
type Supervisor struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// StartConnectorSupervisor launches one supervised loop per enabled
// registration. onStatus receives (connector name, coarse state) transitions
// for the health endpoint ("connecting", "awaiting_pairing", "connected", ...).
func StartConnectorSupervisor(ctx context.Context, bunDB *bun.DB, logger *slog.Logger, onStatus func(name, status string)) *Supervisor {
	ctx, cancel := context.WithCancel(ctx)
	s := &Supervisor{cancel: cancel}
	for _, reg := range registrations(bunDB, logger) {
		if !reg.Enabled() {
			logger.Info("connector disabled", "connector", reg.Name)
			onStatus(reg.Name, "disabled")
			continue
		}
		s.wg.Add(1)
		go func(reg Registration) {
			defer s.wg.Done()
			supervise(ctx, reg, logger, func(status string) { onStatus(reg.Name, status) })
		}(reg)
	}
	return s
}

// Stop shuts all supervised connectors down and waits for them to finish.
func (s *Supervisor) Stop() {
	s.cancel()
	s.wg.Wait()
}

// registrations is the connector registry.
func registrations(bunDB *bun.DB, logger *slog.Logger) []Registration {
	groupStore := sharedpostgres.NewGroupStore(bunDB)
	groupSourceStore := sharedpostgres.NewGroupSourceStore(bunDB)
	return []Registration{
		{
			Name: "whatsapp",
			Enabled: func() bool {
				return !strings.EqualFold(os.Getenv("WHATSAPP_ENABLED"), "false")
			},
			Build: func(ctx context.Context, onStatus func(string)) (Connector, error) {
				sessionPath := os.Getenv("WHATSAPP_SESSION_PATH")
				if sessionPath == "" {
					sessionPath = "whatsapp.db"
				}
				return whatsapp.NewClient(ctx, groupStore, groupSourceStore, bunDB, sessionPath, logger, onStatus)
			},
		},
		// Telegram slot: gate on TELEGRAM_ENABLED + TELEGRAM_BOT_TOKEN and build
		// a telegram.Client here once implemented — see docs/connectors.md.
	}
}

// supervise runs one connector's build/run loop with backoff, forever.
func supervise(ctx context.Context, reg Registration, logger *slog.Logger, onStatus func(string)) {
	backoff := supervisorBaseBackoff
	for {
		runErr := runOnce(ctx, reg, onStatus)
		if ctx.Err() != nil {
			return
		}
		if runErr != nil {
			backoff = min(backoff*2, supervisorMaxBackoff)
			logger.Warn("connector failed", "connector", reg.Name, "error", runErr, "retry_in", backoff)
			onStatus("reconnecting: " + runErr.Error())
		} else {
			backoff = supervisorBaseBackoff
			logger.Info("connector stopped, restarting", "connector", reg.Name, "retry_in", backoff)
			onStatus("restarting")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
	}
}

// runOnce builds, runs, and tears down one connector lifetime. The connector
// is rebuilt every iteration — e.g. whatsmeow clears its session store when
// the phone unlinks the device, so a rebuild after logout lands in the QR
// pairing flow automatically.
func runOnce(ctx context.Context, reg Registration, onStatus func(string)) error {
	c, err := reg.Build(ctx, onStatus)
	if err != nil {
		onStatus("error: " + err.Error())
		return err
	}
	defer c.Stop()
	return c.Run(ctx)
}
