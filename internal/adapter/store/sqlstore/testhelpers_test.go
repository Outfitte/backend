package sqlstore_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/outfitte/outfitte/internal/adapter/store/sqlstore"
)

func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	require.NoError(t, sqlstore.RunMigrations(t.Context(), db))
	return db
}
