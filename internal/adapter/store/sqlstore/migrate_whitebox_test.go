package sqlstore

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

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
