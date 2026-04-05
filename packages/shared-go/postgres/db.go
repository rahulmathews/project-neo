package postgres

import (
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"project-neo/shared/model"
)

// NewDB creates and returns a configured Bun DB instance.
func NewDB(dsn string) (*bun.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Register models for Bun's query builder
	db.RegisterModel(
		(*model.User)(nil),
		(*model.Group)(nil),
		(*model.Location)(nil),
		(*model.LocationContext)(nil),
		(*model.Ride)(nil),
		(*model.Match)(nil),
	)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	return db, nil
}
