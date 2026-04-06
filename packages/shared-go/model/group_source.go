package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type SourceType string

const (
	SourceTypeWhatsApp SourceType = "WHATSAPP"
	SourceTypeTelegram SourceType = "TELEGRAM"
	SourceTypeDiscord  SourceType = "DISCORD"
	SourceTypeSlack    SourceType = "SLACK"
	SourceTypeManual   SourceType = "MANUAL"
)

type GroupSource struct {
	bun.BaseModel `bun:"table:group_sources,alias:gs"`

	ID               uuid.UUID  `bun:"id,pk,type:uuid"`
	GroupID          uuid.UUID  `bun:"group_id,type:uuid"`
	SourceType       SourceType `bun:"source_type"`
	SourceIdentifier string     `bun:"source_identifier"`
	LastParsedAt     *time.Time `bun:"last_parsed_at"`
	ParseFrequency   int        `bun:"parse_frequency"`
	IsActive         bool       `bun:"is_active"`
	CreatedAt        time.Time  `bun:"created_at"`
	UpdatedAt        time.Time  `bun:"updated_at"`
}
