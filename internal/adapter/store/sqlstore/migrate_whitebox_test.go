package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

	require.NoError(t, m.Steps(2))

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

func itemColumnNames(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(items)`)
	if err != nil {
		return nil, fmt.Errorf("query table_info: %w", err)
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("scan table_info row: %w", err)
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return cols, nil
}

func TestItemColumnNamesShouldReturnErrWhenQueryFails(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	_, err = itemColumnNames(t.Context(), db)
	require.Error(t, err)
}

func TestItemColumnNamesShouldReturnErrWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	_, err := itemColumnNames(t.Context(), db)
	require.Error(t, err)
}

func TestItemColumnNamesShouldReturnErrWhenRowsErrFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	_, err := itemColumnNames(t.Context(), db)
	require.Error(t, err)
}

func TestMigration003UpShouldRemoveWearCountAndLastWornAtFromItems(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	src, err := newMigrationSource(migrationsFS, "migrations")
	require.NoError(t, err)

	m, err := newMigrateRunner(src, db)
	require.NoError(t, err)

	require.NoError(t, m.Up())

	cols, err := itemColumnNames(t.Context(), db)
	require.NoError(t, err)
	require.NotContains(t, cols, "wear_count")
	require.NotContains(t, cols, "last_worn_at")
}

func TestMigration003DownShouldRestoreWearCountAndLastWornAtToItems(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	src, err := newMigrationSource(migrationsFS, "migrations")
	require.NoError(t, err)

	m, err := newMigrateRunner(src, db)
	require.NoError(t, err)

	require.NoError(t, m.Steps(3))
	require.NoError(t, m.Steps(-1))

	cols, err := itemColumnNames(t.Context(), db)
	require.NoError(t, err)
	require.Contains(t, cols, "wear_count")
	require.Contains(t, cols, "last_worn_at")
}

func TestMigration004DownShouldDropAllOutfitTables(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	src, err := newMigrationSource(migrationsFS, "migrations")
	require.NoError(t, err)

	m, err := newMigrateRunner(src, db)
	require.NoError(t, err)

	require.NoError(t, m.Steps(4))
	require.NoError(t, m.Steps(-1))

	for _, tbl := range []string{"outfit_log_wear_logs", "outfit_logs", "outfit_photos", "outfit_items", "outfits"} {
		var count int
		require.NoError(t, db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, tbl,
		).Scan(&count))
		require.Equal(t, 0, count, "table %s should be dropped after down migration", tbl)
	}
}

func TestMigration004UpShouldCreateAllOutfitTables(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	src, err := newMigrationSource(migrationsFS, "migrations")
	require.NoError(t, err)

	m, err := newMigrateRunner(src, db)
	require.NoError(t, err)

	require.NoError(t, m.Up())

	for _, tbl := range []string{"outfits", "outfit_items", "outfit_photos", "outfit_logs", "outfit_log_wear_logs"} {
		var count int
		require.NoError(t, db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, tbl,
		).Scan(&count))
		require.Equal(t, 1, count, "table %s should exist after up migration", tbl)
	}
}

func TestMigration004UpShouldCreateAllOutfitIndexes(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	src, err := newMigrationSource(migrationsFS, "migrations")
	require.NoError(t, err)

	m, err := newMigrateRunner(src, db)
	require.NoError(t, err)

	require.NoError(t, m.Up())

	for _, idx := range []string{
		"idx_outfits_owner_id",
		"idx_outfit_items_item_id",
		"idx_outfit_photos_outfit_id",
		"idx_outfit_logs_outfit_id",
		"idx_outfit_logs_owner_worn",
		"idx_outfit_log_wear_logs_wear_log_id",
	} {
		var count int
		require.NoError(t, db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, idx,
		).Scan(&count))
		require.Equal(t, 1, count, "index %s should exist after up migration", idx)
	}
}

func TestMigration004UpShouldCascadeDeleteOutfitChildRowsWhenOutfitDeleted(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	src, err := newMigrationSource(migrationsFS, "migrations")
	require.NoError(t, err)

	m, err := newMigrateRunner(src, db)
	require.NoError(t, err)

	require.NoError(t, m.Up())

	_, err = db.ExecContext(t.Context(), `PRAGMA foreign_keys = ON`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(),
		`INSERT INTO users (id, email, password_hash, created_at) VALUES ('u1', 'a@b.com', 'h', '2024-01-01')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(),
		`INSERT INTO items (id, owner_id, name, created_at) VALUES ('i1', 'u1', 'shirt', '2024-01-01')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(),
		`INSERT INTO wear_logs (id, item_id, owner_id, worn_on, created_at) VALUES ('w1', 'i1', 'u1', '2024-01-01', '2024-01-01')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(),
		`INSERT INTO outfits (id, owner_id, created_at) VALUES ('o1', 'u1', '2024-01-01')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(),
		`INSERT INTO outfit_items (outfit_id, item_id) VALUES ('o1', 'i1')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(),
		`INSERT INTO outfit_photos (id, outfit_id, media_key, created_at) VALUES ('p1', 'o1', 'key', '2024-01-01')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(),
		`INSERT INTO outfit_logs (id, outfit_id, owner_id, worn_on, created_at) VALUES ('ol1', 'o1', 'u1', '2024-01-01', '2024-01-01')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(),
		`INSERT INTO outfit_log_wear_logs (outfit_log_id, wear_log_id) VALUES ('ol1', 'w1')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `DELETE FROM outfits WHERE id = 'o1'`)
	require.NoError(t, err)

	for _, query := range []string{
		`SELECT COUNT(*) FROM outfit_items WHERE outfit_id = 'o1'`,
		`SELECT COUNT(*) FROM outfit_photos WHERE outfit_id = 'o1'`,
		`SELECT COUNT(*) FROM outfit_logs WHERE outfit_id = 'o1'`,
		`SELECT COUNT(*) FROM outfit_log_wear_logs WHERE outfit_log_id = 'ol1'`,
	} {
		var count int
		require.NoError(t, db.QueryRowContext(t.Context(), query).Scan(&count))
		require.Equal(t, 0, count, "expected 0 rows for: %s", query)
	}
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
