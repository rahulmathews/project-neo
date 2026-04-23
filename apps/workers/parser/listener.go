package parser

import (
	"context"
	"log/slog"
	"time"

	"project-neo/shared/model"
	sharedpostgres "project-neo/shared/postgres"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/uptrace/bun"
)

// StartListener opens a persistent LISTEN connection on 'messages_inserted' and
// dispatches each notification to the extractor pipeline. Blocks until ctx is cancelled.
func StartListener(ctx context.Context, databaseURL string, bunDB *bun.DB, provider LLMProvider, logger *slog.Logger) {
	msgStore := sharedpostgres.NewMessageStore(bunDB)

	listener := pq.NewListener(databaseURL, 10*time.Second, time.Minute,
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				logger.Error("parser pg listener event", "event", ev, "error", err)
			}
		},
	)
	defer func() {
		if err := listener.Close(); err != nil {
			logger.Error("parser listener close", "error", err)
		}
	}()

	if err := listener.Listen("messages_inserted"); err != nil {
		logger.Error("parser listener: listen failed", "error", err)
		return
	}
	logger.Info("message parser listener started")

	for {
		select {
		case <-ctx.Done():
			return
		case n := <-listener.Notify:
			if n == nil {
				// nil means the connection was re-established after a drop — safe to continue
				continue
			}
			id, err := uuid.Parse(n.Extra)
			if err != nil {
				logger.Warn("parser listener: invalid uuid payload", "payload", n.Extra)
				continue
			}
			go handleNotification(ctx, id, msgStore, bunDB, provider, logger)
		}
	}
}

func handleNotification(ctx context.Context, id uuid.UUID, msgStore *sharedpostgres.MessageStore, db *bun.DB, provider LLMProvider, logger *slog.Logger) {
	msg, err := msgStore.GetByID(ctx, id)
	if err != nil {
		logger.Error("parser listener: fetch message", "id", id, "error", err)
		return
	}
	if msg == nil {
		return // message not found — skip
	}
	if msg.ParseStatus != model.ParseStatusPending {
		return // already handled (defensive check)
	}
	Process(ctx, msg, db, provider, logger)
}
