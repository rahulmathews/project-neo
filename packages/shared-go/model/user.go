package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UserRole string

const (
	UserRoleRider  UserRole = "RIDER"
	UserRoleDriver UserRole = "DRIVER"
	UserRoleBoth   UserRole = "BOTH"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID        uuid.UUID `bun:"id,pk,type:uuid"`
	Email     *string   `bun:"email"`
	Phone     *string   `bun:"phone"`
	Name      string    `bun:"name"`
	Role      UserRole  `bun:"role"`
	AvatarURL *string   `bun:"avatar_url"`
	CreatedAt time.Time `bun:"created_at,nullzero"`
	UpdatedAt time.Time `bun:"updated_at,nullzero"`
}
