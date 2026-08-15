package parser

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"project-neo/shared/model"
	sharedpostgres "project-neo/shared/postgres"
	"project-neo/workers/internal/metrics"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/uptrace/bun"
)

const defaultMaxConcurrentParses = 8

// StartListener opens a persistent LISTEN connection on 'messages_inserted' and
// dispatches each notification to the extractor pipeline. Blocks until ctx is
// cancelled. onReady is called after LISTEN succeeds.
func StartListener(ctx context.Context, databaseURL string, bunDB *bun.DB, provider LLMProvider, m *metrics.Parser, logger *slog.Logger, onReady func()) error {
	msgStore := sharedpostgres.NewMessageStore(bunDB)
	// Cap parse fan-out: a burst of group traffic must not spawn unbounded
	// goroutines (each potentially holding a 30s LLM call). A full semaphore
	// delays dispatch; the recovery ticker backstops anything left PENDING.
	sem := make(chan struct{}, maxConcurrentParses(logger))

	listener := pq.NewListener(
		databaseURL, 10*time.Second, time.Minute,
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
		return fmt.Errorf("listen messages_inserted: %w", err)
	}
	if onReady != nil {
		onReady()
	}
	logger.Info("message parser listener started")

	for {
		select {
		case <-ctx.Done():
			return nil
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
			select {
			case <-ctx.Done():
				return nil
			case sem <- struct{}{}:
			}
			go func() {
				defer func() { <-sem }()
				handleNotification(ctx, id, msgStore, bunDB, provider, m, logger)
			}()
		}
	}
}

func maxConcurrentParses(logger *slog.Logger) int {
	raw := os.Getenv("PARSER_MAX_CONCURRENT")
	if raw == "" {
		return defaultMaxConcurrentParses
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		logger.Warn("parser listener: invalid PARSER_MAX_CONCURRENT, using default",
			"value", raw, "default", defaultMaxConcurrentParses)
		return defaultMaxConcurrentParses
	}
	return n
}

func handleNotification(ctx context.Context, id uuid.UUID, msgStore *sharedpostgres.MessageStore, db *bun.DB, provider LLMProvider, m *metrics.Parser, logger *slog.Logger) {
	// Each notification runs in its own goroutine: an unrecovered panic here
	// would kill the whole workers service, not just this message.
	defer func() {
		if r := recover(); r != nil {
			logger.Error("parser: panic recovered",
				"msg_id", id, "panic", r, "stack", string(debug.Stack()))
		}
	}()
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
	Process(ctx, msg, db, provider, m, logger)
}
