package sqlstore_test

import (
	"database/sql"
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/store/sqlstore"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestRunMigrationsShouldReturnErrorWhenDBIsClosed(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	err = sqlstore.RunMigrations(db, "sqlite")
	require.Error(t, err)
}

func TestRunMigrationsShouldSucceedWhenGivenFreshDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	err = sqlstore.RunMigrations(db, "sqlite")
	require.NoError(t, err)
}

func TestRunMigrationsShouldSucceedWhenMigrationsAlreadyApplied(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	require.NoError(t, sqlstore.RunMigrations(db, "sqlite"))

	err = sqlstore.RunMigrations(db, "sqlite")
	require.NoError(t, err)
}
