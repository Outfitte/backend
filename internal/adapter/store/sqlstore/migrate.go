package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migrateSQLite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/outfitte/outfitte/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies all pending migrations to the given database.
func RunMigrations(ctx context.Context, db *sql.DB, driverName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	dbDriver, err := migrateSQLite.WithInstance(db, &migrateSQLite.Config{
		DatabaseName: driverName,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	m, err := migrate.NewWithInstance("iofs", src, driverName, dbDriver)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return nil
}
