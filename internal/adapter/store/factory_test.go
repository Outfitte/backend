package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	store "github.com/outfitte/outfitte/internal/adapter/store"
	"github.com/outfitte/outfitte/internal/config"
	"github.com/outfitte/outfitte/internal/domain"
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
	require.NoError(t, closer.Close())
}
