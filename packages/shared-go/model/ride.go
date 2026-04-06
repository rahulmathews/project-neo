package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type RideType string

const (
	RideTypeNeedRide      RideType = "NEED_RIDE"
	RideTypeRideAvailable RideType = "RIDE_AVAILABLE"
)

type RideStatus string

const (
	RideStatusAvailable RideStatus = "AVAILABLE"
	RideStatusMatched   RideStatus = "MATCHED"
	RideStatusCompleted RideStatus = "COMPLETED"
	RideStatusCancelled RideStatus = "CANCELLED"
	RideStatusExpired   RideStatus = "EXPIRED"
)

type Ride struct {
	bun.BaseModel `bun:"table:rides,alias:r"`

	ID               uuid.UUID  `bun:"id,pk,type:uuid"`
	MessageID        *uuid.UUID `bun:"message_id,type:uuid"`
	GroupID          uuid.UUID  `bun:"group_id,type:uuid"`
	Type             RideType   `bun:"type"`
	FromLocationID   *uuid.UUID `bun:"from_location_id,type:uuid"`
	ToLocationID     *uuid.UUID `bun:"to_location_id,type:uuid"`
	FromLocationText *string    `bun:"from_location_text"`
	ToLocationText   *string    `bun:"to_location_text"`
	DepartureTime    *time.Time `bun:"departure_time"`
	IsImmediate      bool       `bun:"is_immediate"`
	Cost             *float64   `bun:"cost"`
	Currency         string     `bun:"currency"`
	Distance         *float64   `bun:"distance"`
	SeatsAvailable   *int       `bun:"seats_available"`
	Status           RideStatus `bun:"status"`
	PosterUserID     *uuid.UUID `bun:"poster_user_id,type:uuid"`
	CreatedAt        time.Time  `bun:"created_at,nullzero"`
	UpdatedAt        time.Time  `bun:"updated_at,nullzero"`
}

type RideFilter struct {
	GroupID uuid.UUID
	Type    *RideType
	Status  *RideStatus
	Pagination
}
