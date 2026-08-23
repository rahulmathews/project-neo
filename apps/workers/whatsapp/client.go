package whatsapp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"project-neo/shared/model"
	sharedpostgres "project-neo/shared/postgres"
	"project-neo/workers/internal/store"

	"github.com/google/uuid"
	"github.com/mdp/qrterminal/v3"
	"github.com/uptrace/bun"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

// Client implements internal.Connector for WhatsApp via whatsmeow.
type Client struct {
	wac              *whatsmeow.Client
	logger           *slog.Logger
	wg               *sync.WaitGroup
	container        *sqlstore.Container
	groupStore       *sharedpostgres.GroupStore
	groupSourceStore *sharedpostgres.GroupSourceStore
	bunDB            *bun.DB
	onStatus         func(string)

	mu      sync.RWMutex
	handler *Handler

	loggedOut  chan struct{}
	logoutOnce sync.Once
}

// NewClient constructs the whatsmeow client without touching the network:
// it opens the SQLite session store, loads the device, and registers the
// event handler. Connecting (QR pairing or silent resume) happens in Run.
func NewClient(
	ctx context.Context,
	groupStore *sharedpostgres.GroupStore,
	groupSourceStore *sharedpostgres.GroupSourceStore,
	bunDB *bun.DB,
	sessionPath string,
	logger *slog.Logger,
	onStatus func(string),
) (c *Client, err error) {
	// Open a dedicated SQLite database for the whatsmeow session.
	sqliteDB, err := sql.Open("sqlite", "file:"+sessionPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open whatsapp session db: %w", err)
	}

	container := sqlstore.NewWithDB(sqliteDB, "sqlite3", waLog.Noop)
	// Close container (which also closes sqliteDB) on any error return; on success it is owned by Client.Stop().
	defer func() {
		if err != nil {
			_ = container.Close()
		}
	}()
	if err = container.Upgrade(ctx); err != nil {
		return nil, fmt.Errorf("whatsmeow db upgrade: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get whatsmeow device: %w", err)
	}

	wac := whatsmeow.NewClient(deviceStore, waLog.Noop)
	// Anti-detection: disable automatic missed-message requests on reconnect.
	wac.AutomaticMessageRerequestFromPhone = false

	c = &Client{
		wac:              wac,
		logger:           logger,
		wg:               &sync.WaitGroup{},
		container:        container,
		groupStore:       groupStore,
		groupSourceStore: groupSourceStore,
		bunDB:            bunDB,
		onStatus:         onStatus,
		loggedOut:        make(chan struct{}),
	}

	// Registered once for the client's lifetime. Message events are dropped
	// until Run has synced groups and installed the handler.
	wac.AddEventHandler(func(evt any) {
		switch e := evt.(type) {
		case *events.Message:
			c.mu.RLock()
			h := c.handler
			c.mu.RUnlock()
			if h != nil {
				h.Handle(ctx, e)
			}
		case *events.LoggedOut:
			c.logger.Warn("whatsapp account unlinked", "reason", e.Reason)
			c.logoutOnce.Do(func() { close(c.loggedOut) })
		}
	})

	return c, nil
}

// Name identifies this connector in health output and logs.
func (c *Client) Name() string { return "whatsapp" }

// Run connects (QR pairing on first run, silent resume thereafter), syncs the
// account's joined groups into the database, installs the message handler,
// and blocks until ctx is cancelled or the account is unlinked. It returns
// nil on unlink — the caller should rebuild the client, which re-enters the
// QR pairing flow because whatsmeow clears its session store on logout.
func (c *Client) Run(ctx context.Context) error {
	c.setStatus("connecting")
	if err := c.connect(ctx); err != nil {
		return err
	}

	// Anti-detection: set presence to unavailable immediately after connect.
	if presenceErr := c.wac.SendPresence(ctx, types.PresenceUnavailable); presenceErr != nil {
		c.logger.Warn("failed to set presence unavailable", "error", presenceErr)
	}

	// Discover all joined groups and sync to the database. Runs on every
	// (re)connect, so groups joined while offline are picked up.
	jidMap, srcMap, err := c.syncGroups(ctx)
	if err != nil {
		return fmt.Errorf("sync groups: %w", err)
	}

	msgWriter := store.NewMessageWriter(c.bunDB, c.logger)
	srcReader := store.NewGroupSourceReader(c.bunDB, c.logger)
	c.setHandler(NewHandler(jidMap, srcMap, msgWriter, srcReader, c.logger, c.wg))

	c.setStatus("connected")
	c.logger.Info("whatsapp connector started", "groups", len(jidMap))

	// Periodic group resync: whatsmeow reconnects internally without
	// re-entering Run, so groups joined, renamed, or disabled mid-session
	// would otherwise be missed until a supervisor-level restart.
	resync := time.NewTicker(groupSyncInterval())
	defer resync.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.loggedOut:
			c.setStatus("unlinked")
			return nil
		case <-resync.C:
			jidMap, srcMap, err := c.syncGroups(ctx)
			if err != nil {
				c.logger.Warn("whatsapp group resync failed", "error", err)
				continue
			}
			c.setHandler(NewHandler(jidMap, srcMap, msgWriter, srcReader, c.logger, c.wg))
		}
	}
}

// groupSyncInterval reads WHATSAPP_GROUP_SYNC_INTERVAL (default 10m).
func groupSyncInterval() time.Duration {
	const def = 10 * time.Minute
	raw := os.Getenv("WHATSAPP_GROUP_SYNC_INTERVAL")
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return def
	}
	return d
}

// syncGroups fetches all WhatsApp groups the account is joined to, upserts
// them into the database, and returns JID→groupID and JID→sourceID maps.
func (c *Client) syncGroups(ctx context.Context) (jidMap, srcMap map[string]uuid.UUID, err error) {
	groups, err := c.wac.GetJoinedGroups(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get joined groups: %w", err)
	}

	jidMap = make(map[string]uuid.UUID, len(groups))
	srcMap = make(map[string]uuid.UUID, len(groups))

	for _, g := range groups {
		jid := g.JID.String()

		groupID, upsertErr := c.groupStore.UpsertGroup(ctx, g.Name)
		if upsertErr != nil {
			c.logger.Warn("failed to upsert group", "jid", jid, "name", g.Name, "error", upsertErr)
			continue
		}

		sourceID, upsertErr := c.groupSourceStore.UpsertGroupSource(ctx, groupID, model.SourceTypeWhatsApp, jid)
		if upsertErr != nil {
			c.logger.Warn("failed to upsert group source", "jid", jid, "group_id", groupID, "error", upsertErr)
			continue
		}

		jidMap[jid] = groupID
		srcMap[jid] = sourceID
	}

	// Honor the operator kill-switch: only sources still active in the DB are
	// monitored (the upserts above never re-activate a disabled source).
	active, err := c.groupSourceStore.ListActive(ctx, model.SourceTypeWhatsApp)
	if err != nil {
		return nil, nil, fmt.Errorf("list active sources: %w", err)
	}
	activeByJID := make(map[string]struct{}, len(active))
	for _, src := range active {
		activeByJID[src.SourceIdentifier] = struct{}{}
	}
	for jid := range jidMap {
		if _, ok := activeByJID[jid]; !ok {
			c.logger.Info("skipping disabled group source", "jid", jid)
			delete(jidMap, jid)
			delete(srcMap, jid)
		}
	}

	c.logger.Info("synced whatsapp groups", "count", len(jidMap))
	return jidMap, srcMap, nil
}

// connect handles first-time QR pairing or silent session resume.
func (c *Client) connect(ctx context.Context) error {
	if c.wac.Store.ID != nil {
		// Existing session — reconnect silently.
		if err := c.wac.Connect(); err != nil {
			return fmt.Errorf("whatsmeow reconnect: %w", err)
		}
		return nil
	}
	return c.waitForPairing(ctx)
}

// waitForPairing runs QR pairing sessions until the operator links the
// account. There is no overall deadline: a fresh QR code is printed to the
// logs every time the previous one expires, so `docker compose up -d` stays
// healthy on a brand-new machine and pairing can happen whenever convenient.
func (c *Client) waitForPairing(ctx context.Context) error {
	c.setStatus("awaiting_pairing")
	for {
		paired, err := c.qrSession(ctx)
		if err != nil {
			return err
		}
		if paired {
			return nil
		}
		// QR window expired — brief pause, then request a fresh code batch.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

// qrSession runs one QR pairing window. Returns (true, nil) once linked and
// (false, nil) when a window that showed at least one code expired normally.
// A window that ends before any code was shown returns an error — that is a
// connection-level failure, and the caller's backoff should pace the retries
// rather than hammering the WhatsApp servers every few seconds.
func (c *Client) qrSession(ctx context.Context) (bool, error) {
	qrChan, err := c.wac.GetQRChannel(ctx)
	if err != nil {
		// Pairing may have completed in a race with the previous window.
		if c.wac.Store.ID != nil {
			if connErr := c.wac.Connect(); connErr != nil {
				return false, fmt.Errorf("whatsmeow connect: %w", connErr)
			}
			return true, nil
		}
		return false, fmt.Errorf("get QR channel: %w", err)
	}
	if err := c.wac.Connect(); err != nil {
		return false, fmt.Errorf("whatsmeow connect: %w", err)
	}
	sawCode := false
	for {
		select {
		case evt, ok := <-qrChan:
			if !ok {
				// Closed without an explicit terminal event — whatsmeow also
				// closes the channel on internal failures, so trust the
				// store, not the close.
				if c.wac.Store.ID != nil {
					return true, nil
				}
				c.wac.Disconnect()
				if !sawCode {
					return false, fmt.Errorf("QR channel closed before any code was issued")
				}
				return false, nil
			}
			switch evt.Event {
			case "code":
				sawCode = true
				fmt.Println("\n=== Scan with WhatsApp → Linked Devices → Link a Device ===")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("=============================================================")
				fmt.Println("(no deadline — a fresh code is printed when this one expires)")
				fmt.Println()
			case "success":
				return true, nil
			default:
				// "timeout", "error", "err-unexpected-state", ... — window over.
				c.wac.Disconnect()
				if !sawCode {
					return false, fmt.Errorf("QR window ended before any code was issued (event %q): %w",
						evt.Event, orErr(evt.Error))
				}
				c.logger.Info("whatsapp QR window ended, issuing a fresh code", "event", evt.Event)
				return false, nil
			}
		case <-ctx.Done():
			c.wac.Disconnect()
			return false, ctx.Err()
		}
	}
}

// orErr substitutes a placeholder for nil so %w always has an error to wrap.
func orErr(err error) error {
	if err == nil {
		return errNoDetail
	}
	return err
}

var errNoDetail = errors.New("no further detail")

func (c *Client) setHandler(h *Handler) {
	c.mu.Lock()
	c.handler = h
	c.mu.Unlock()
}

func (c *Client) setStatus(status string) {
	if c.onStatus != nil {
		c.onStatus(status)
	}
}

// Stop disconnects the WhatsApp client, waits for all in-flight handlers to
// finish, and closes the SQLite session database.
func (c *Client) Stop() {
	c.wac.Disconnect()
	c.wg.Wait()
	if err := c.container.Close(); err != nil {
		c.logger.Error("failed to close whatsapp session db", "error", err)
	}
}
