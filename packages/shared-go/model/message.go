package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
	"unicode"

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
	GroupSourceID    *uuid.UUID  `bun:"group_source_id,type:uuid"`
	SourceMessageID  *string     `bun:"source_message_id"`
	SenderIdentifier *string     `bun:"sender_identifier"`
	Content          string      `bun:"content"`
	ContentHash      string      `bun:"content_hash"`
	Timestamp        time.Time   `bun:"timestamp"`
	ParsedAt         *time.Time  `bun:"parsed_at"`
	ParseStatus      ParseStatus `bun:"parse_status"`
	ParseError       *string     `bun:"parse_error"`
	RetryCount       int         `bun:"retry_count"`
	RideID           *uuid.UUID  `bun:"ride_id,type:uuid"`
	CreatedAt        time.Time   `bun:"created_at,nullzero"`
}

// NormalizeMessageContent returns the canonical content stored in messages history.
func NormalizeMessageContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.Map(normalizeMessageRune, content)

	lines := strings.Split(content, "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			normalized = append(normalized, line)
		}
	}

	return strings.Join(normalized, "\n")
}

func normalizeMessageRune(r rune) rune {
	switch r {
	case '\u00a0', '\t', '\v', '\f':
		return ' '
	case '\u200b', '\u200c', '\u200d', '\ufeff':
		return -1
	}
	if unicode.IsControl(r) && r != '\n' {
		return -1
	}
	return r
}

// ComputeContentHash returns the SHA-256 hex digest of normalized message content.
// Used for history deduplication and parser-phase duplicate ride detection.
func ComputeContentHash(content string) string {
	h := sha256.Sum256([]byte(NormalizeMessageContent(content)))
	return hex.EncodeToString(h[:])
}
