package postgres

import (
	"database/sql"
	"fmt"

	"project-neo/shared/model"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// NewDB creates and returns a configured Bun DB instance.
func NewDB(dsn string) (*bun.DB, error) {
	db := bun.NewDB(
		sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn))),
		pgdialect.New(),
	)

	// Register models for Bun's query builder
	db.RegisterModel(
		(*model.User)(nil),
		(*model.Group)(nil),
		(*model.Location)(nil),
		(*model.LocationContext)(nil),
		(*model.Ride)(nil),
		(*model.Match)(nil),
		(*model.Message)(nil),
		(*model.GroupSource)(nil),
	)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	return db, nil
}
