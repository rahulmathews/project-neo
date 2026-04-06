package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"project-neo/shared/model"
	"project-neo/workers/internal/store"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// Client implements internal.Connector for WhatsApp via whatsmeow.
type Client struct {
	number  string
	wac     *whatsmeow.Client
	handler *Handler
	logger  *slog.Logger
	wg      *sync.WaitGroup
}

// NewClient creates a whatsmeow client for the given phone number.
// srcs are the group_sources rows this number should listen to.
func NewClient(
	ctx context.Context,
	number string,
	srcs []*model.GroupSource,
	bunDB *bun.DB,
	sqlDB *sql.DB,
	logger *slog.Logger,
) (*Client, error) {
	// Build JID → group_id and JID → source_id maps from group_sources.
	jidMap := make(map[string]uuid.UUID, len(srcs))
	srcMap := make(map[string]uuid.UUID, len(srcs))
	for _, s := range srcs {
		jidMap[s.SourceIdentifier] = s.GroupID
		srcMap[s.SourceIdentifier] = s.ID
	}

	// Initialise the whatsmeow Postgres session store.
	container := sqlstore.NewWithDB(sqlDB, "postgres", waLog.Noop)

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get whatsmeow device: %w", err)
	}

	wac := whatsmeow.NewClient(deviceStore, waLog.Noop)
	// Anti-detection: disable automatic missed-message requests on reconnect.
	wac.AutomaticMessageRerequestFromPhone = false

	msgWriter := store.NewMessageWriter(bunDB, logger)
	srcReader := store.NewGroupSourceReader(bunDB, logger)

	wg := &sync.WaitGroup{}

	c := &Client{
		number:  number,
		wac:     wac,
		logger:  logger,
		wg:      wg,
		handler: NewHandler(jidMap, srcMap, msgWriter, srcReader, logger, wg),
	}

	// Register event handler.
	wac.AddEventHandler(func(evt interface{}) {
		if msg, ok := evt.(*events.Message); ok {
			c.handler.Handle(msg)
		}
	})

	if err := c.connect(ctx); err != nil {
		return nil, err
	}

	// Anti-detection: set presence to unavailable immediately after connect.
	_ = wac.SendPresence(ctx, types.PresenceUnavailable)

	logger.Info("whatsapp connector started", "number", number, "groups", len(srcs))
	return c, nil
}

// connect handles first-time QR flow or silent session resume.
func (c *Client) connect(ctx context.Context) error {
	if c.wac.Store.ID == nil {
		// No stored session — need QR scan.
		qrChan, _ := c.wac.GetQRChannel(ctx)
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
				if evt.Event == "code" {
					fmt.Printf("\n=== WhatsApp QR for %s ===\n%s\n=========================\n\n", c.number, evt.Code)
				} else if evt.Event == "success" {
					return nil
				} else if evt.Event == "timeout" || evt.Event == "error" {
					return fmt.Errorf("QR scan failed (event: %s): restart the service and scan again", evt.Event)
				}
			case <-timeout:
				return fmt.Errorf("QR scan timeout for %s: restart the service and scan within 60 seconds", c.number)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	// Existing session — reconnect silently.
	if err := c.wac.Connect(); err != nil {
		return fmt.Errorf("whatsmeow reconnect: %w", err)
	}
	return nil
}

// Start is a no-op — connection is established in NewClient.
func (c *Client) Start(_ context.Context) error {
	return nil
}

// Stop disconnects the WhatsApp client and waits for all in-flight handlers to finish.
func (c *Client) Stop() {
	c.wac.Disconnect()
	c.wg.Wait()
}
