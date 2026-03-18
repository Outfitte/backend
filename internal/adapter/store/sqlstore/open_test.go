package sqlstore_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/adapter/store/sqlstore"
	"github.com/outfitte/outfitte/internal/config"
	"github.com/outfitte/outfitte/internal/domain"
)

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
