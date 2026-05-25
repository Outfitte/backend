package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"

	store "github.com/outfitte/backend/internal/adapter/store"
	"github.com/outfitte/backend/internal/config"
	"github.com/outfitte/backend/internal/domain"
)

func TestNewRepositoriesShouldReturnErrUnsupportedDriverWhenDriverIsPostgres(t *testing.T) {
	cfg := config.Config{DB: config.DBConfig{Driver: "postgres", DSN: "postgres://x"}}
	_, _, err := store.NewRepositories(t.Context(), cfg)
	require.ErrorIs(t, err, domain.ErrUnsupportedDriver)
}

func TestNewRepositoriesShouldReturnErrUnsupportedDriverWhenDriverIsUnknown(t *testing.T) {
	cfg := config.Config{DB: config.DBConfig{Driver: "mysql"}}
	_, _, err := store.NewRepositories(t.Context(), cfg)
	require.ErrorIs(t, err, domain.ErrUnsupportedDriver)
}

func TestNewRepositoriesShouldReturnErrWhenContextCancelledForSQLite(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	cfg := config.Config{DB: config.DBConfig{Driver: "sqlite", DSN: ":memory:"}}
	_, _, err := store.NewRepositories(ctx, cfg)
	require.ErrorIs(t, err, context.Canceled)
}

func TestNewRepositoriesShouldReturnErrWhenSQLiteMigrationFails(t *testing.T) {
	// Pre-seed a temp SQLite file with a conflicting table so that the first
	// migration (CREATE TABLE users) fails, exercising the RunMigrations error path.
	tmpFile := filepath.Join(t.TempDir(), "conflict.db")
	db, err := sql.Open("sqlite", tmpFile)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE users (id TEXT NOT NULL PRIMARY KEY)")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	cfg := config.Config{DB: config.DBConfig{Driver: "sqlite", DSN: tmpFile}}
	_, _, err = store.NewRepositories(t.Context(), cfg)
	require.Error(t, err)
}

func TestNewRepositoriesShouldReturnRepositoriesAndCloserWhenDriverIsSQLite(t *testing.T) {
	cfg := config.Config{DB: config.DBConfig{Driver: "sqlite", DSN: ":memory:"}}
	repos, closer, err := store.NewRepositories(t.Context(), cfg)
	require.NoError(t, err)
	require.NotNil(t, closer)
	require.NotNil(t, repos.Items)
	require.NotNil(t, repos.Users)
	require.NotNil(t, repos.Sessions)
	require.NotNil(t, repos.Locations)
	require.NotNil(t, repos.WearLogs)
	require.NotNil(t, repos.AppSettings)
	require.NotNil(t, repos.Shares)
	require.NotNil(t, repos.ItemTransfers)
	require.NotNil(t, repos.ItemTransferTransactor)
	require.NoError(t, closer.Close())
}

func TestNewRepositoriesShouldReturnErrWhenSQLiteOpenFails(t *testing.T) {
	cfg := config.Config{DB: config.DBConfig{Driver: "sqlite", DSN: "/nonexistent/path/test.db"}}
	_, _, err := store.NewRepositories(t.Context(), cfg)
	require.Error(t, err)
}

func TestNewRepositoriesShouldReturnRepositoriesAndNopCloserWhenDriverIsJSON(t *testing.T) {
	cfg := config.Config{DB: config.DBConfig{Driver: "json", DSN: t.TempDir()}}
	repos, closer, err := store.NewRepositories(t.Context(), cfg)
	require.NoError(t, err)
	require.NotNil(t, closer)
	require.NotNil(t, repos.Items)
	require.NotNil(t, repos.Users)
	require.NotNil(t, repos.Sessions)
	require.NotNil(t, repos.Locations)
	require.NotNil(t, repos.WearLogs)
	require.NotNil(t, repos.AppSettings)
	require.NotNil(t, repos.Shares)
	require.NotNil(t, repos.ItemTransfers)
	require.NotNil(t, repos.ItemTransferTransactor)
	require.NoError(t, closer.Close())
}
