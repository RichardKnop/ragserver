package ragserver

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
)

func Migrate(db *sql.DB) error {
	// Run db migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{SchemaName: "public"})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://db/migrations",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrations up: %w", err)
	}

	return nil
}
