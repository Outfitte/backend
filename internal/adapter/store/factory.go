package store

import (
	"context"
	"io"

	storejson "github.com/outfitte/backend/internal/adapter/store/json"
	"github.com/outfitte/backend/internal/adapter/store/sqlstore"
	"github.com/outfitte/backend/internal/config"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// nopCloser is an io.Closer that does nothing on Close.
type nopCloser struct{}

func (nopCloser) Close() error { return nil }

// NewRepositories constructs a ports.Repositories and its associated io.Closer
// based on the driver specified in cfg. The caller must defer Close() on the
// returned Closer to release any held resources.
func NewRepositories(ctx context.Context, cfg config.Config) (ports.Repositories, io.Closer, error) {
	if err := ctx.Err(); err != nil {
		return ports.Repositories{}, nil, err
	}
	switch cfg.DB.Driver {
	case "json":
		return storejson.NewRepositories(cfg.DB.DSN), nopCloser{}, nil
	case "sqlite":
		return openSQLiteRepositories(ctx, cfg)
	case "postgres":
		return ports.Repositories{}, nil, domain.ErrUnsupportedDriver
	default:
		return ports.Repositories{}, nil, domain.ErrUnsupportedDriver
	}
}

func openSQLiteRepositories(ctx context.Context, cfg config.Config) (ports.Repositories, io.Closer, error) {
	db, err := sqlstore.Open(ctx, cfg.DB)
	if err != nil {
		return ports.Repositories{}, nil, err
	}
	if err := sqlstore.RunMigrations(ctx, db); err != nil {
		db.Close()
		return ports.Repositories{}, nil, err
	}
	return sqlstore.NewRepositories(db), db, nil
}

