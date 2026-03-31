package sqlstore_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/adapter/store/sqlstore"
	"github.com/outfitte/backend/internal/config"
	"github.com/outfitte/backend/internal/domain"
)

func TestOpenShouldReturnErrWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	cfg := config.DBConfig{Driver: "sqlite", DSN: ":memory:"}
	_, err := sqlstore.Open(ctx, cfg)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOpenShouldReturnErrIOWhenSQLiteDSNPathIsNotCreatable(t *testing.T) {
	// sql.Open is lazy; ExecContext triggers the actual connection attempt.
	// A DSN pointing to a non-existent directory causes ExecContext to fail.
	cfg := config.DBConfig{Driver: "sqlite", DSN: "/nonexistent_xyz_dir/test.db"}
	_, err := sqlstore.Open(t.Context(), cfg)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOpenShouldFailWhenGivenUnknownDriver(t *testing.T) {
	cfg := config.DBConfig{Driver: "unknown", DSN: ":memory:"}
	_, err := sqlstore.Open(t.Context(), cfg)
	require.ErrorIs(t, err, domain.ErrUnsupportedDriver)
}

func TestOpenShouldFailWhenGivenPostgresDriver(t *testing.T) {
	cfg := config.DBConfig{Driver: "postgres", DSN: "host=localhost"}
	_, err := sqlstore.Open(t.Context(), cfg)
	require.ErrorIs(t, err, domain.ErrUnsupportedDriver)
}

func TestOpenShouldSucceedWhenGivenSQLiteDriver(t *testing.T) {
	cfg := config.DBConfig{Driver: "sqlite", DSN: ":memory:"}
	db, err := sqlstore.Open(t.Context(), cfg)
	require.NoError(t, err)
	require.NotNil(t, db)
	t.Cleanup(func() { db.Close() })
}

func TestOpenShouldApplyWALModeWhenGivenSQLiteDriver(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "test.db")
	cfg := config.DBConfig{Driver: "sqlite", DSN: dsn}
	db, err := sqlstore.Open(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	var mode string
	require.NoError(t, db.QueryRowContext(t.Context(), "PRAGMA journal_mode;").Scan(&mode))
	require.Equal(t, "wal", mode)
}

func TestOpenShouldEnableForeignKeysWhenGivenSQLiteDriver(t *testing.T) {
	cfg := config.DBConfig{Driver: "sqlite", DSN: ":memory:"}
	db, err := sqlstore.Open(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	var fkEnabled int
	require.NoError(t, db.QueryRowContext(t.Context(), "PRAGMA foreign_keys;").Scan(&fkEnabled))
	require.Equal(t, 1, fkEnabled)
}
