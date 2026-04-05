package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Location struct {
	bun.BaseModel `bun:"table:locations,alias:loc"`

	ID        uuid.UUID `bun:"id,pk,type:uuid"`
	Name      string    `bun:"name"`
	Latitude  float64   `bun:"latitude"`
	Longitude float64   `bun:"longitude"`
	Address   *string   `bun:"address"`
	City      *string   `bun:"city"`
	State     *string   `bun:"state"`
	Country   *string   `bun:"country"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type LocationContext struct {
	bun.BaseModel `bun:"table:location_contexts,alias:lc"`

	ID            uuid.UUID  `bun:"id,pk,type:uuid"`
	GroupID       uuid.UUID  `bun:"group_id,type:uuid"`
	LocationAlias string     `bun:"location_alias"`
	LocationName  string     `bun:"location_name"`
	LocationID    *uuid.UUID `bun:"location_id,type:uuid"`
	CreatedAt     time.Time  `bun:"created_at"`
	UpdatedAt     time.Time  `bun:"updated_at"`
}
