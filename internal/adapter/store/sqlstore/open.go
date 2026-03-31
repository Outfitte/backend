package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/outfitte/backend/internal/config"
	"github.com/outfitte/backend/internal/domain"
	_ "modernc.org/sqlite"
)

const sqlitePragmas = "PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"

// sqlOpenFn is the function used to open a database connection.
// Exposed as a variable so whitebox tests can inject a failing implementation.
var sqlOpenFn = sql.Open

// Open opens a database connection for the given DBConfig, applying
// any driver-specific setup (e.g. PRAGMAs for SQLite).
func Open(ctx context.Context, cfg config.DBConfig) (*sql.DB, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch cfg.Driver {
	case "sqlite":
		return openSQLite(ctx, cfg.DSN)
	case "postgres":
		return nil, domain.ErrUnsupportedDriver
	default:
		return nil, domain.ErrUnsupportedDriver
	}
}

func openSQLite(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sqlOpenFn("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	if _, err := db.ExecContext(ctx, sqlitePragmas); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return db, nil
}
