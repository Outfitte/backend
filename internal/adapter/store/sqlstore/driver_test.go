package sqlstore_test

import (
	"database/sql"
	"testing"

	migrateSQLite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestDriverNameShouldMatchWhenGivenModerncSQLite verifies that the
// migrate/database/sqlite package registers under the same driver name
// ("sqlite") as modernc.org/sqlite, confirming they are wire-compatible.
func TestDriverNameShouldMatchWhenGivenModerncSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = migrateSQLite.WithInstance(db, &migrateSQLite.Config{
		DatabaseName: "test",
	})
	require.NoError(t, err)
}
