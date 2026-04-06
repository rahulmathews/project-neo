package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ParseStatus string

const (
	ParseStatusPending ParseStatus = "PENDING"
	ParseStatusSuccess ParseStatus = "SUCCESS"
	ParseStatusFailed  ParseStatus = "FAILED"
	ParseStatusSkipped ParseStatus = "SKIPPED"
)

type Message struct {
	bun.BaseModel `bun:"table:messages,alias:msg"`

	ID               uuid.UUID   `bun:"id,pk,type:uuid"`
	GroupID          uuid.UUID   `bun:"group_id,type:uuid"`
	SourceMessageID  *string     `bun:"source_message_id"`
	SenderIdentifier *string     `bun:"sender_identifier"`
	Content          string      `bun:"content"`
	ContentHash      string      `bun:"content_hash"`
	Timestamp        time.Time   `bun:"timestamp"`
	ParsedAt         *time.Time  `bun:"parsed_at"`
	ParseStatus      ParseStatus `bun:"parse_status"`
	ParseError       *string     `bun:"parse_error"`
	CreatedAt        time.Time   `bun:"created_at,nullzero"`
}

// ComputeContentHash returns the SHA-256 hex digest of the trimmed message content.
// Used for cross-group deduplication in the parser phase.
func ComputeContentHash(content string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(h[:])
}
