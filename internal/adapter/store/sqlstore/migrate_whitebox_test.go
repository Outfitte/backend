package sqlstore

import (
	"database/sql"
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
