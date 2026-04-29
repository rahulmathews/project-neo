package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"project-neo/shared/model"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MessageStore struct {
	db *bun.DB
}

func NewMessageStore(db *bun.DB) *MessageStore {
	return &MessageStore{db: db}
}

// Insert writes a message row. If source_message_id is set and a row with the same
// (group_id, source_message_id) already exists, the insert is silently skipped.
// Returns true if a new row was inserted, false if a conflict caused a no-op.
func (s *MessageStore) Insert(ctx context.Context, msg *model.Message) (bool, error) {
	res, err := s.db.NewInsert().
		Model(msg).
		On("CONFLICT ON CONSTRAINT messages_group_id_source_message_id_key DO NOTHING").
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("insert message: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("insert message rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

// ExistsByHash checks whether a message with the same group, content hash, and exact
// timestamp already exists. Used to deduplicate messages with a nil source_message_id.
func (s *MessageStore) ExistsByHash(ctx context.Context, groupID uuid.UUID, contentHash string, timestamp time.Time) (bool, error) {
	exists, err := s.db.NewSelect().
		Model((*model.Message)(nil)).
		Where("group_id = ?", groupID).
		Where("content_hash = ?", contentHash).
		Where("timestamp = ?", timestamp).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check message exists by hash: %w", err)
	}
	return exists, nil
}

// GetByID fetches a message by primary key. Returns (nil, nil) if not found.
func (s *MessageStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Message, error) {
	msg := new(model.Message)
	err := s.db.NewSelect().
		Model(msg).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get message by id: %w", err)
	}
	return msg, nil
}
