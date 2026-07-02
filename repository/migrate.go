package repository

import (
	"embed"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies every not-yet-applied migration from the embedded
// migrations/ directory using goose. Applied versions are tracked in goose's
// goose_db_version table, so it is safe to run on every startup.
func Migrate(db *sqlx.DB) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db.DB, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
