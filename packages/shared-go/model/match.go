package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MatchStatus string

const (
	MatchStatusPending   MatchStatus = "PENDING"
	MatchStatusAccepted  MatchStatus = "ACCEPTED"
	MatchStatusRejected  MatchStatus = "REJECTED"
	MatchStatusCompleted MatchStatus = "COMPLETED"
	MatchStatusCancelled MatchStatus = "CANCELLED"
)

type Match struct {
	bun.BaseModel `bun:"table:matches,alias:m"`

	ID          uuid.UUID   `bun:"id,pk,type:uuid"`
	RideID      uuid.UUID   `bun:"ride_id,type:uuid"`
	RiderID     uuid.UUID   `bun:"rider_id,type:uuid"`
	DriverID    uuid.UUID   `bun:"driver_id,type:uuid"`
	Status      MatchStatus `bun:"status"`
	MatchedAt   time.Time   `bun:"matched_at"`
	AcceptedAt  *time.Time  `bun:"accepted_at"`
	CompletedAt *time.Time  `bun:"completed_at"`
	CancelledAt *time.Time  `bun:"cancelled_at"`
	CreatedAt   time.Time   `bun:"created_at"`
	UpdatedAt   time.Time   `bun:"updated_at"`
}
