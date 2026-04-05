package model

import "github.com/google/uuid"

// Stub types — replaced with full implementations in a later task.
// These exist so gqlgen can resolve type references during code generation.

type UserRole string
type RideType string
type RideStatus string
type MatchStatus string

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
	DepartureTime         *string
	IsImmediate           bool
	Cost                  *float64
	Currency              *string
	Distance              *float64
	SeatsAvailable        *int
}

type UpdateRideInput struct {
	DepartureTime  *string
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
