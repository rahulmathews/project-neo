package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Group struct {
	bun.BaseModel `bun:"table:groups,alias:g"`

	ID          uuid.UUID `bun:"id,pk,type:uuid"`
	Name        string    `bun:"name"`
	Description *string   `bun:"description"`
	IsActive    bool      `bun:"is_active"`
	CreatedAt   time.Time `bun:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at"`
}
