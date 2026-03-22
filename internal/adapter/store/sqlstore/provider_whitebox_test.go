package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/outfitte/outfitte/internal/domain"
)

// openTestDB opens a migrated in-memory SQLite DB for whitebox tests.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	require.NoError(t, RunMigrations(t.Context(), db))
	return db
}

// ── listItems ─────────────────────────────────────────────────────────────────

func TestListItemsContextCancelledInternal(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := listItems(ctx, db)
	require.ErrorIs(t, err, context.Canceled)
}

// ── getItem ───────────────────────────────────────────────────────────────────

func TestGetItemContextCancelledInternal(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := getItem(ctx, db, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

// ── deleteItem ────────────────────────────────────────────────────────────────

func TestDeleteItemContextCancelledInternal(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := deleteItem(ctx, db, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

// ── saveItem ──────────────────────────────────────────────────────────────────

func TestSaveItemContextCancelledInternal(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := saveItem(ctx, db, domain.Item{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestSaveItemShouldReturnErrIOWhenUpsertFails(t *testing.T) {
	db := openTestDB(t)
	db.SetMaxOpenConns(1)
	_, err := db.ExecContext(t.Context(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	var item domain.Item
	item.ID = "item-fk-fail"
	item.OwnerID = "nonexistent-user"
	item.Name = "Test"
	item.CreatedAt = time.Now()
	item.Metadata = domain.ItemMetadata{}

	err = saveItem(t.Context(), db, item)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── upsertItemRow ─────────────────────────────────────────────────────────────

func TestUpsertItemRowContextCancelledInternal(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err = upsertItemRow(ctx, tx, domain.Item{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestUpsertItemRowShouldReturnErrIOWhenExecFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	var item domain.Item
	item.ID = "item-1"
	err = upsertItemRow(t.Context(), tx, item)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── replacePhotos ─────────────────────────────────────────────────────────────

func TestReplacePhotosContextCancelledInternal(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var item1 domain.Item
	item1.ID = "item-1"
	err = replacePhotos(ctx, tx, item1)
	require.ErrorIs(t, err, context.Canceled)
}

func TestReplacePhotosShouldReturnErrIOWhenDeleteFails(t *testing.T) {
	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	var item1 domain.Item
	item1.ID = "item-1"
	err = replacePhotos(t.Context(), tx, item1)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── scanItem ──────────────────────────────────────────────────────────────────

func TestScanItemShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)
	rows, err := db.QueryContext(t.Context(), "SELECT 1")
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())
	_, err = scanItem(rows)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── loadPhotos ────────────────────────────────────────────────────────────────

func TestLoadPhotosContextCancelledInternal(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := loadPhotos(ctx, db, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLoadPhotosShouldReturnErrIOWhenQueryFails(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	require.NoError(t, RunMigrations(t.Context(), db))
	db.Close()

	_, err = loadPhotos(t.Context(), db, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLoadPhotosShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)

	// Insert a photo with a non-integer position to cause scan failure into *int.
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO item_photos (id, item_id, media_key, position, created_at)
		VALUES ('ph-scan', 'item-scan', 'key.jpg', 'not-an-int', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	var err2 error
	_, err2 = loadPhotos(t.Context(), db, "item-scan")
	require.ErrorIs(t, err2, domain.ErrIO)
}

// ── listItems rows.Err ────────────────────────────────────────────────────────

func TestListItemsRowsErrInternal(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	_, err := listItems(t.Context(), db)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── loadPhotos rows.Err ───────────────────────────────────────────────────────

func TestLoadPhotosRowsErrInternal(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	_, err := loadPhotos(t.Context(), db, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── deleteItem RowsAffected error ─────────────────────────────────────────────

func TestDeleteItemShouldReturnErrIOWhenRowsAffectedFails(t *testing.T) {
	db := openFakeDB(t, "fake-rows-aff-err")
	err := deleteItem(t.Context(), db, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── saveItem tx.Commit error ──────────────────────────────────────────────────

func TestSaveItemShouldReturnErrIOWhenCommitFails(t *testing.T) {
	db := openFakeDB(t, "fake-commit-err")
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Test"
	item.CreatedAt = time.Now()
	item.Metadata = domain.ItemMetadata{}
	err := saveItem(t.Context(), db, item)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── upsertItemRow json.Marshal error ──────────────────────────────────────────

func TestUpsertItemRowShouldReturnErrIOWhenMarshalFails(t *testing.T) {
	old := jsonMarshalFn
	jsonMarshalFn = func(_ any) ([]byte, error) { return nil, errors.New("injected marshal failure") }
	t.Cleanup(func() { jsonMarshalFn = old })

	db := openTestDB(t)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	var item domain.Item
	err = upsertItemRow(t.Context(), tx, item)
	require.ErrorIs(t, err, domain.ErrIO)
}
