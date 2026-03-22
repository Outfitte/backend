package sqlstore

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// ── batchLoadPhotos ───────────────────────────────────────────────────────────

func TestBatchLoadPhotosShouldReturnErrIOWhenQueryFails(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.Close()

	repo := &ItemRepository{db: db}
	var item domain.Item
	item.ID = "item-1"
	err = repo.batchLoadPhotos(t.Context(), []domain.Item{item})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestBatchLoadPhotosShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openTestDB(t)

	// Insert a photo with non-integer position to cause scan failure into *int.
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO item_photos (id, item_id, media_key, position, created_at)
		VALUES ('ph-batch-scan', 'item-x', 'key.jpg', 'not-an-int', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	repo := &ItemRepository{db: db}
	var item domain.Item
	item.ID = "item-x"
	err = repo.batchLoadPhotos(t.Context(), []domain.Item{item})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── queryItemsByOwner rows.Err ────────────────────────────────────────────────

func TestQueryItemsByOwnerRowsErrInternal(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &ItemRepository{db: db}
	_, err := repo.queryItemsByOwner(t.Context(), "owner-1", ports.ItemListFilter{})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── batchLoadPhotos rows.Err ──────────────────────────────────────────────────

func TestBatchLoadPhotosRowsErrInternal(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &ItemRepository{db: db}
	var item domain.Item
	item.ID = "item-1"
	err := repo.batchLoadPhotos(t.Context(), []domain.Item{item})
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── ListPhotoKeys ─────────────────────────────────────────────────────────────

func TestListPhotoKeysShouldReturnErrIOWhenScanFails(t *testing.T) {
	db := openFakeDB(t, "fake-scan-err")
	repo := &ItemRepository{db: db}
	_, err := repo.ListPhotoKeys(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListPhotoKeysRowsErrInternal(t *testing.T) {
	db := openFakeDB(t, "fake-rows-err")
	repo := &ItemRepository{db: db}
	_, err := repo.ListPhotoKeys(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── ItemRepository.Save tx.Commit error ──────────────────────────────────────

func TestItemRepositorySaveShouldReturnErrIOWhenCommitFails(t *testing.T) {
	db := openFakeDB(t, "fake-commit-err")
	repo := &ItemRepository{db: db}
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Test"
	item.CreatedAt = time.Now()
	item.Metadata = domain.ItemMetadata{}
	err := repo.Save(t.Context(), item)
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── ItemRepository.Save ───────────────────────────────────────────────────────

func TestItemRepositorySaveShouldReturnErrIOWhenUpsertFails(t *testing.T) {
	db := openTestDB(t)
	db.SetMaxOpenConns(1)
	_, err := db.ExecContext(t.Context(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	repo := &ItemRepository{db: db}

	var item domain.Item
	item.ID = "item-repo-fk-fail"
	item.OwnerID = "nonexistent-user"
	item.Name = "Test"
	item.CreatedAt = time.Now()
	item.Metadata = domain.ItemMetadata{}

	err = repo.Save(t.Context(), item)
	require.ErrorIs(t, err, domain.ErrIO)
}
