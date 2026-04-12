package whatsapp

import (
	"context"
	"database/sql"
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
	wac      *whatsmeow.Client
	handler  *Handler
	logger   *slog.Logger
	wg       *sync.WaitGroup
	sqliteDB *sql.DB
}

// NewClient creates a whatsmeow client, connects to WhatsApp (QR on first run, silent
// resume on subsequent runs), discovers all joined groups, syncs them to the database,
// and starts listening for messages.
func NewClient(
	ctx context.Context,
	groupStore *sharedpostgres.GroupStore,
	groupSourceStore *sharedpostgres.GroupSourceStore,
	bunDB *bun.DB,
	sessionPath string,
	logger *slog.Logger,
) (c *Client, err error) {
	// Open a dedicated SQLite database for the whatsmeow session.
	sqliteDB, err := sql.Open("sqlite", "file:"+sessionPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open whatsapp session db: %w", err)
	}
	// Close sqliteDB on any error return; on success it is owned by Client.Stop().
	defer func() {
		if err != nil {
			_ = sqliteDB.Close()
		}
	}()

	container := sqlstore.NewWithDB(sqliteDB, "sqlite3", waLog.Noop)
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

	wg := &sync.WaitGroup{}
	c = &Client{wac: wac, logger: logger, wg: wg, sqliteDB: sqliteDB}

	// Connect first — QR flow or silent session resume.
	if err = c.connect(ctx); err != nil {
		return nil, err
	}

	// Anti-detection: set presence to unavailable immediately after connect.
	if presenceErr := wac.SendPresence(ctx, types.PresenceUnavailable); presenceErr != nil {
		logger.Warn("failed to set presence unavailable", "error", presenceErr)
	}

	// Discover all joined groups and sync to the database.
	jidMap, srcMap, err := c.syncGroups(ctx, groupStore, groupSourceStore)
	if err != nil {
		return nil, fmt.Errorf("sync groups: %w", err)
	}

	msgWriter := store.NewMessageWriter(bunDB, logger)
	srcReader := store.NewGroupSourceReader(bunDB, logger)

	c.handler = NewHandler(jidMap, srcMap, msgWriter, srcReader, logger, wg)

	// Register event handler — ctx is the application root context.
	wac.AddEventHandler(func(evt any) {
		if msg, ok := evt.(*events.Message); ok {
			c.handler.Handle(ctx, msg)
		}
	})

	logger.Info("whatsapp connector started", "groups", len(jidMap))
	return c, nil
}

// syncGroups fetches all WhatsApp groups the account is joined to, upserts them into
// the database, and returns JID→groupID and JID→sourceID maps.
func (c *Client) syncGroups(
	ctx context.Context,
	groupStore *sharedpostgres.GroupStore,
	groupSourceStore *sharedpostgres.GroupSourceStore,
) (jidMap map[string]uuid.UUID, srcMap map[string]uuid.UUID, err error) {
	groups, err := c.wac.GetJoinedGroups(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get joined groups: %w", err)
	}

	jidMap = make(map[string]uuid.UUID, len(groups))
	srcMap = make(map[string]uuid.UUID, len(groups))

	for _, g := range groups {
		jid := g.JID.String()

		groupID, upsertErr := groupStore.UpsertGroup(ctx, g.Name)
		if upsertErr != nil {
			c.logger.Warn("failed to upsert group", "jid", jid, "name", g.Name, "error", upsertErr)
			continue
		}

		sourceID, upsertErr := groupSourceStore.UpsertGroupSource(ctx, groupID, model.SourceTypeWhatsApp, jid)
		if upsertErr != nil {
			c.logger.Warn("failed to upsert group source", "jid", jid, "group_id", groupID, "error", upsertErr)
			continue
		}

		jidMap[jid] = groupID
		srcMap[jid] = sourceID
	}

	c.logger.Info("synced whatsapp groups", "count", len(jidMap))
	return jidMap, srcMap, nil
}

// connect handles first-time QR flow or silent session resume.
func (c *Client) connect(ctx context.Context) error {
	if c.wac.Store.ID != nil {
		// Existing session — reconnect silently.
		if err := c.wac.Connect(); err != nil {
			return fmt.Errorf("whatsmeow reconnect: %w", err)
		}
		return nil
	}
	return c.connectWithQR(ctx)
}

// connectWithQR handles the first-time pairing flow by printing a QR code.
func (c *Client) connectWithQR(ctx context.Context) error {
	qrChan, err := c.wac.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("get QR channel: %w", err)
	}
	if err := c.wac.Connect(); err != nil {
		return fmt.Errorf("whatsmeow connect: %w", err)
	}
	timeout := time.After(60 * time.Second)
	for {
		select {
		case evt, ok := <-qrChan:
			if !ok {
				return nil // connected successfully
			}
			switch evt.Event {
			case "code":
				fmt.Println("\n=== Scan with WhatsApp → Linked Devices → Link a Device ===")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("=============================================================")
				fmt.Println()
			case "success":
				return nil
			case "timeout", "error":
				return fmt.Errorf("QR scan failed (event: %s): restart the service and scan again", evt.Event)
			}
		case <-timeout:
			return fmt.Errorf("QR scan timeout: restart the service and scan within 60 seconds")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Start is a no-op — connection is established in NewClient.
func (c *Client) Start(_ context.Context) error {
	return nil
}

// Stop disconnects the WhatsApp client, waits for all in-flight handlers to finish,
// and closes the SQLite session database.
func (c *Client) Stop() {
	c.wac.Disconnect()
	c.wg.Wait()
	if err := c.sqliteDB.Close(); err != nil {
		c.logger.Error("failed to close whatsapp session db", "error", err)
	}
}
