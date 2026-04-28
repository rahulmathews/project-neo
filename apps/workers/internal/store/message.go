package store

import (
	"context"
	"log/slog"
	"time"

	"project-neo/shared/model"
	sharedpostgres "project-neo/shared/postgres"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// MessageWriter handles workers-specific message insertion logic:
// content hash computation, nil-ID deduplication, and DB writes.
type MessageWriter struct {
	store  *sharedpostgres.MessageStore
	logger *slog.Logger
}

func NewMessageWriter(db *bun.DB, logger *slog.Logger) *MessageWriter {
	return &MessageWriter{
		store:  sharedpostgres.NewMessageStore(db),
		logger: logger,
	}
}

// Write inserts a message into the database. Returns true if the message was stored,
// false if it was a duplicate (skipped).
func (w *MessageWriter) Write(
	ctx context.Context,
	groupID uuid.UUID,
	sourceID uuid.UUID, // group_sources.id — used to identify which source row to update
	sourceMessageID *string,
	senderIdentifier *string,
	content string,
	timestamp time.Time,
) (stored bool, err error) {
	hash := model.ComputeContentHash(content)

	// For messages with no WhatsApp message ID, check exact hash+timestamp match.
	if sourceMessageID == nil {
		exists, err := w.store.ExistsByHash(ctx, groupID, hash, timestamp)
		if err != nil {
			return false, err
		}
		if exists {
			return false, nil // duplicate, skip
		}
	}

	msg := &model.Message{
		ID:               uuid.New(),
		GroupID:          groupID,
		SourceMessageID:  sourceMessageID,
		SenderIdentifier: senderIdentifier,
		Content:          content,
		ContentHash:      hash,
		Timestamp:        timestamp,
		ParseStatus:      model.ParseStatusPending,
	}

	inserted, err := w.store.Insert(ctx, msg)
	if err != nil {
		return false, err
	}
	return inserted, nil
}
