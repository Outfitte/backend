package sqlstore

import (
	"database/sql"
	"fmt"

	"github.com/outfitte/outfitte/internal/config"
	"github.com/outfitte/outfitte/internal/domain"
	_ "modernc.org/sqlite"
)

const sqlitePragmas = "PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"

// Open opens a database connection for the given DBConfig, applying
// any driver-specific setup (e.g. PRAGMAs for SQLite).
func Open(cfg config.DBConfig) (*sql.DB, error) {
	switch cfg.Driver {
	case "sqlite":
		return openSQLite(cfg.DSN)
	case "postgres":
		return nil, domain.ErrUnsupportedDriver
	default:
		return nil, domain.ErrUnsupportedDriver
	}
}

func openSQLite(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	if _, err := db.Exec(sqlitePragmas); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return db, nil
}
