package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	migrateSQLite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/outfitte/outfitte/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const dbName = "outfitte"

// RunMigrations applies all pending migrations to the given database.
func RunMigrations(ctx context.Context, db *sql.DB) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	src, err := newMigrationSource(migrationsFS, "migrations")
	if err != nil {
		return err
	}

	m, err := newMigrateRunner(src, db)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return nil
}

func newMigrationSource(fsys fs.FS, dir string) (source.Driver, error) {
	src, err := iofs.New(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return src, nil
}

func newMigrateRunner(src source.Driver, db *sql.DB) (*migrate.Migrate, error) {
	dbDriver, err := migrateSQLite.WithInstance(db, &migrateSQLite.Config{DatabaseName: dbName})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	m, err := migrate.NewWithInstance("iofs", src, dbName, dbDriver)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return m, nil
}
