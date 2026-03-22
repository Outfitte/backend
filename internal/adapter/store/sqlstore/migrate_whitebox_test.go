package sqlstore

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	migrate "github.com/golang-migrate/migrate/v4"
	migrateDB "github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestRunMigrationsShouldReturnErrIOWhenSourceCreationFails(t *testing.T) {
	old := migrationsEmbedDir
	migrationsEmbedDir = "nonexistent"
	t.Cleanup(func() { migrationsEmbedDir = old })

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	err = RunMigrations(t.Context(), db)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestNewMigrateRunnerShouldReturnErrIOWhenNewWithInstanceFails(t *testing.T) {
	old := migrateNewWithInstance
	migrateNewWithInstance = func(_ string, _ source.Driver, _ string, _ migrateDB.Driver) (*migrate.Migrate, error) {
		return nil, errors.New("injected failure")
	}
	t.Cleanup(func() { migrateNewWithInstance = old })

	src, err := newMigrationSource(migrationsFS, "migrations")
	require.NoError(t, err)

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = newMigrateRunner(src, db)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestNewMigrationSourceShouldReturnErrIOWhenDirMissing(t *testing.T) {
	_, err := newMigrationSource(migrationsFS, "nonexistent")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestNewMigrateRunnerShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	src, err := newMigrationSource(migrationsFS, "migrations")
	require.NoError(t, err)

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	_, err = newMigrateRunner(src, db)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestMigrateDownShouldDropTokenHashIndex(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	src, err := newMigrationSource(migrationsFS, "migrations")
	require.NoError(t, err)

	m, err := newMigrateRunner(src, db)
	require.NoError(t, err)

	require.NoError(t, m.Up())

	var count int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_sessions_token_hash'`,
	).Scan(&count))
	require.Equal(t, 1, count, "index should exist after up migration")

	require.NoError(t, m.Steps(-1))

	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_sessions_token_hash'`,
	).Scan(&count))
	require.Equal(t, 0, count, "index should be gone after down migration")
}

func TestRunMigrationsShouldReturnErrIOWhenMigrationsAreDirty(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	require.NoError(t, RunMigrations(t.Context(), db))

	// Mark the latest migration as dirty so that the next Up() returns ErrDirty.
	_, err = db.ExecContext(t.Context(), "UPDATE schema_migrations SET dirty = 1")
	require.NoError(t, err)

	err = RunMigrations(t.Context(), db)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestTokenHashIndexIsUsedForLookup(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	src, err := newMigrationSource(migrationsFS, "migrations")
	require.NoError(t, err)
	m, err := newMigrateRunner(src, db)
	require.NoError(t, err)
	require.NoError(t, m.Up())

	rows, err := db.Query(
		`EXPLAIN QUERY PLAN SELECT id, user_id, token_hash, expires_at, created_at FROM sessions WHERE token_hash = ?`,
		"somehash",
	)
	require.NoError(t, err)
	defer rows.Close()

	var usesIndex bool
	for rows.Next() {
		var id, parent, notused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notused, &detail))
		if strings.Contains(detail, "idx_sessions_token_hash") {
			usesIndex = true
		}
	}
	require.NoError(t, rows.Err())
	require.True(t, usesIndex, "query plan should use idx_sessions_token_hash index")
}
