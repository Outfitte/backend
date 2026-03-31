package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/outfitte/backend/internal/adapter/store/sqlstore"
	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestRunMigrationsShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	err = sqlstore.RunMigrations(t.Context(), db)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRunMigrationsShouldReturnErrWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	err = sqlstore.RunMigrations(ctx, db)
	require.ErrorIs(t, err, context.Canceled)
}

func TestRunMigrationsShouldSucceedWhenGivenFreshDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	err = sqlstore.RunMigrations(t.Context(), db)
	require.NoError(t, err)
}

func TestRunMigrationsShouldSucceedWhenMigrationsAlreadyApplied(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	require.NoError(t, sqlstore.RunMigrations(t.Context(), db))

	err = sqlstore.RunMigrations(t.Context(), db)
	require.NoError(t, err)
}

func TestRunMigrationsShouldCreateTokenHashIndex(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	require.NoError(t, sqlstore.RunMigrations(t.Context(), db))

	var count int
	err = db.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_sessions_token_hash'`,
	).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
