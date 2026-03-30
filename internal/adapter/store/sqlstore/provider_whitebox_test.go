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

// ── buildItem ─────────────────────────────────────────────────────────────────

func TestGetItemShouldReturnErrIOWhenMetadataIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-bad-meta', 'owner-1', 'Bad Item', '2025-01-01T00:00:00Z', 'not-json')`)
	require.NoError(t, err)

	_, err = getItem(t.Context(), db, "item-bad-meta")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetItemShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-bad-ts', 'owner-1', 'Bad Item', 'not-a-date', '{}')`)
	require.NoError(t, err)

	_, err = getItem(t.Context(), db, "item-bad-ts")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetItemShouldReturnErrIOWhenPurchaseDateIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, purchase_date, metadata)
		VALUES ('item-bad-pd', 'owner-1', 'Bad Item', '2025-01-01T00:00:00Z', 'not-a-date', '{}')`)
	require.NoError(t, err)

	_, err = getItem(t.Context(), db, "item-bad-pd")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── getItem ───────────────────────────────────────────────────────────────────

func TestGetItemContextCancelledInternal(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := getItem(ctx, db, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetItemShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openTestDB(t)
	db.Close()
	_, err := getItem(t.Context(), db, "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetItemShouldReturnItemWithAllOptionalFieldsWhenSet(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, brand, category_id, color,
		                   location_id, purchase_price, purchase_date, created_at, metadata)
		VALUES ('item-all', 'owner-1', 'Full Item', 'Nike', 'cat-1', 'Red',
		        'loc-1', '99.99', '2024-01-15T00:00:00Z', '2025-01-01T00:00:00Z', '{}')`)
	require.NoError(t, err)

	item, err := getItem(t.Context(), db, "item-all")
	require.NoError(t, err)
	require.NotNil(t, item.Brand)
	require.Equal(t, "Nike", *item.Brand)
	require.NotNil(t, item.CategoryID)
	require.Equal(t, "cat-1", *item.CategoryID)
	require.NotNil(t, item.Color)
	require.Equal(t, "Red", *item.Color)
	require.NotNil(t, item.LocationID)
	require.Equal(t, "loc-1", *item.LocationID)
	require.NotNil(t, item.PurchasePrice)
	require.Equal(t, "99.99", *item.PurchasePrice)
	require.NotNil(t, item.PurchaseDate)
}

func TestGetItemShouldReturnErrIOWhenLoadPhotosFails(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-bad-photo-ts', 'owner-1', 'Item', '2025-01-01T00:00:00Z', '{}')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO item_photos (id, item_id, media_key, position, created_at)
		VALUES ('photo-bad', 'item-bad-photo-ts', 'key.jpg', 0, 'not-a-date')`)
	require.NoError(t, err)

	_, err = getItem(t.Context(), db, "item-bad-photo-ts")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── deleteItem ────────────────────────────────────────────────────────────────

func TestDeleteItemContextCancelledInternal(t *testing.T) {
	db := openTestDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := deleteItem(ctx, db, "item-1")
	require.ErrorIs(t, err, context.Canceled)
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

// ── upsertItemRow with purchaseDate ──────────────────────────────────────────

func TestUpsertItemRowShouldSucceedWhenPurchaseDateIsSet(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-pd', 'pd@test.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	pd := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	var item domain.Item
	item.ID = "item-pd"
	item.OwnerID = "user-pd"
	item.Name = "Test"
	item.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	item.PurchaseDate = &pd
	item.Metadata = domain.ItemMetadata{}

	require.NoError(t, upsertItemRow(t.Context(), tx, item))
}

// ── upsertItemRow FK violation ────────────────────────────────────────────────

func TestUpsertItemRowShouldReturnErrIOWhenFKFails(t *testing.T) {
	db := openTestDB(t)
	db.SetMaxOpenConns(1)
	_, err := db.ExecContext(t.Context(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	var item domain.Item
	item.ID = "item-fk-fail"
	item.OwnerID = "nonexistent-user"
	item.Name = "Test"
	item.CreatedAt = time.Now()
	item.Metadata = domain.ItemMetadata{}

	err = upsertItemRow(t.Context(), tx, item)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── upsertItemRow with archivedAt and disposalReason ─────────────────────────

func TestUpsertItemRowShouldSucceedWhenArchivedAtIsSet(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-arc', 'arc@test.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	archivedAt := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	var item domain.Item
	item.ID = "item-arc"
	item.OwnerID = "user-arc"
	item.Name = "Archived"
	item.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	item.ArchivedAt = &archivedAt
	item.Metadata = domain.ItemMetadata{}

	require.NoError(t, upsertItemRow(t.Context(), tx, item))
}

func TestUpsertItemRowShouldSucceedWhenDisposalReasonIsSet(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-disp', 'disp@test.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	reason := domain.DisposalReason("donated")
	var item domain.Item
	item.ID = "item-disp"
	item.OwnerID = "user-disp"
	item.Name = "Donated"
	item.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	item.DisposalReason = &reason
	item.Metadata = domain.ItemMetadata{}

	require.NoError(t, upsertItemRow(t.Context(), tx, item))
}

func TestGetItemShouldReturnDisposalReasonWhenSet(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, archived_at, disposal_reason, metadata)
		VALUES ('item-disposal', 'owner-1', 'Item', '2025-01-01T00:00:00Z', '2025-06-01T00:00:00Z', 'donated', '{}')`)
	require.NoError(t, err)

	item, err := getItem(t.Context(), db, "item-disposal")
	require.NoError(t, err)
	require.NotNil(t, item.DisposalReason)
	require.Equal(t, domain.DisposalReason("donated"), *item.DisposalReason)
}

// ── buildItem bad archivedAt ──────────────────────────────────────────────────

func TestGetItemShouldReturnErrIOWhenArchivedAtIsInvalid(t *testing.T) {
	db := openTestDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, archived_at, metadata)
		VALUES ('item-bad-arc', 'owner-1', 'Bad Item', '2025-01-01T00:00:00Z', 'not-a-date', '{}')`)
	require.NoError(t, err)

	_, err = getItem(t.Context(), db, "item-bad-arc")
	require.ErrorIs(t, err, domain.ErrIO)
}
