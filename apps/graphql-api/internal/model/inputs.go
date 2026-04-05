package model

import (
	"time"

	"github.com/google/uuid"
)

type Pagination struct {
	Limit  int
	Offset int
}

func NewPagination(limit, offset *int) Pagination {
	p := Pagination{Limit: 20, Offset: 0}
	if limit != nil && *limit > 0 && *limit <= 100 {
		p.Limit = *limit
	}
	if offset != nil && *offset >= 0 {
		p.Offset = *offset
	}
	return p
}

type UpsertUserInput struct {
	Name      string
	Phone     *string
	Role      *UserRole
	AvatarURL *string
}

type CreateRideInput struct {
	GroupID               uuid.UUID
	Type                  RideType
	FromLocationContextID *uuid.UUID
	ToLocationContextID   *uuid.UUID
	FromLocationText      *string
	ToLocationText        *string
	DepartureTime         *time.Time
	IsImmediate           bool
	Cost                  *float64
	Currency              *string
	Distance              *float64
	SeatsAvailable        *int
}

type UpdateRideInput struct {
	DepartureTime  *time.Time
	Cost           *float64
	SeatsAvailable *int
}

type CreateGroupInput struct {
	Name        string
	Description *string
}

type UpsertLocationContextInput struct {
	GroupID       uuid.UUID
	LocationAlias string
	LocationName  string
	LocationID    uuid.UUID
}
