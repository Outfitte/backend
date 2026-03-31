package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/adapter/store/sqlstore"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func seedUserForItem(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES (?, ?, 'hash', 'member', '2025-01-01T00:00:00Z')`,
		id, id+"@example.com")
	require.NoError(t, err)
}

func newItemRepo(t *testing.T) (*sqlstore.ItemRepository, *sql.DB) {
	t.Helper()
	db := openMigratedDB(t)
	return sqlstore.NewItemRepository(db), db
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestItemRepositoryGetShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.Get(ctx, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemRepositoryGetShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newItemRepo(t)

	_, err := repo.Get(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemRepositoryGetShouldReturnItemWithPhotosWhenRowExists(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-1")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-1', 'user-1', 'Blue Jeans', '2025-06-01T10:00:00Z', '{}')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO item_photos (id, item_id, media_key, position, created_at)
		VALUES ('photo-1', 'item-1', 'key-a.jpg', 0, '2025-06-01T10:00:00Z')`)
	require.NoError(t, err)

	item, err := repo.Get(t.Context(), "item-1")
	require.NoError(t, err)
	require.Equal(t, "item-1", item.GetID())
	require.Equal(t, "Blue Jeans", item.Name)
	require.Len(t, item.Photos, 1)
	require.Equal(t, "key-a.jpg", item.Photos[0].MediaKey)
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestItemRepositorySaveShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var item domain.Item
	item.ID = "item-1"
	err := repo.Save(ctx, item)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemRepositorySaveShouldPersistItemWithoutTouchingPhotos(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-save")

	var item domain.Item
	item.ID = "item-save-1"
	item.OwnerID = "user-save"
	item.Name = "Sneakers"
	item.CreatedAt = time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{Fields: map[string]string{"size": "42"}}

	require.NoError(t, repo.Save(t.Context(), item))

	got, err := repo.Get(t.Context(), "item-save-1")
	require.NoError(t, err)
	require.Equal(t, "Sneakers", got.Name)
	require.Equal(t, "42", got.Metadata.Fields["size"])
	require.Empty(t, got.Photos)
}

func TestItemRepositorySaveShouldNotReplaceExistingPhotosWithFKEnforced(t *testing.T) {
	db := openMigratedDB(t)
	_, err := db.ExecContext(t.Context(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	repo := sqlstore.NewItemRepository(db)
	seedUserForItem(t, db, "user-fk")

	var item domain.Item
	item.ID = "item-fk"
	item.OwnerID = "user-fk"
	item.Name = "FK Test"
	item.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}
	require.NoError(t, repo.Save(t.Context(), item))

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO item_photos (id, item_id, media_key, position, created_at)
		VALUES ('photo-fk', 'item-fk', 'fk.jpg', 0, '2025-06-01T10:00:00Z')`)
	require.NoError(t, err)

	item.Name = "FK Test Updated"
	require.NoError(t, repo.Save(t.Context(), item))

	got, err := repo.Get(t.Context(), "item-fk")
	require.NoError(t, err)
	require.Equal(t, "FK Test Updated", got.Name)
	require.Len(t, got.Photos, 1, "photos must survive re-save with FK enforcement on")
	require.Equal(t, "fk.jpg", got.Photos[0].MediaKey)
}

func TestItemRepositorySaveShouldNotReplaceExistingPhotos(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-noreplace")

	var item domain.Item
	item.ID = "item-noreplace"
	item.OwnerID = "user-noreplace"
	item.Name = "Jacket"
	item.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}

	require.NoError(t, repo.Save(t.Context(), item))

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO item_photos (id, item_id, media_key, position, created_at)
		VALUES ('photo-keep', 'item-noreplace', 'keep.jpg', 0, '2025-06-01T10:00:00Z')`)
	require.NoError(t, err)

	item.Name = "Jacket Updated"
	require.NoError(t, repo.Save(t.Context(), item))

	got, err := repo.Get(t.Context(), "item-noreplace")
	require.NoError(t, err)
	require.Equal(t, "Jacket Updated", got.Name)
	require.Len(t, got.Photos, 1, "Save must not delete existing photos")
	require.Equal(t, "keep.jpg", got.Photos[0].MediaKey)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestItemRepositoryDeleteShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.Delete(ctx, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemRepositoryDeleteShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	repo, _ := newItemRepo(t)

	err := repo.Delete(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemRepositoryDeleteShouldRemoveItemWhenExists(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-del")

	var item domain.Item
	item.ID = "item-del-1"
	item.OwnerID = "user-del"
	item.Name = "To Delete"
	item.CreatedAt = time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}
	require.NoError(t, repo.Save(t.Context(), item))

	require.NoError(t, repo.Delete(t.Context(), "item-del-1"))

	_, err := repo.Get(t.Context(), "item-del-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// ── ListByOwner ───────────────────────────────────────────────────────────────

func TestItemRepositoryListByOwnerShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListByOwner(ctx, "user-1", ports.ItemListFilter{Status: ports.ItemStatusActive})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemRepositoryListByOwnerShouldReturnEmptySliceWhenNoItemsExist(t *testing.T) {
	repo, _ := newItemRepo(t)

	items, err := repo.ListByOwner(t.Context(), "user-nobody", ports.ItemListFilter{Status: ports.ItemStatusActive})
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestItemRepositoryListByOwnerShouldReturnOnlyOwnerItems(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-a")
	seedUserForItem(t, db, "user-b")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES
			('item-a1', 'user-a', 'Item A1', '2025-01-01T00:00:00Z', '{}'),
			('item-b1', 'user-b', 'Item B1', '2025-01-01T00:00:00Z', '{}')`)
	require.NoError(t, err)

	items, err := repo.ListByOwner(t.Context(), "user-a", ports.ItemListFilter{Status: ports.ItemStatusAll})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "item-a1", items[0].GetID())
}

func TestItemRepositoryListByOwnerShouldReturnActiveItemsWhenStatusIsActive(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-active")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata, archived_at)
		VALUES
			('item-active', 'user-active', 'Active', '2025-01-01T00:00:00Z', '{}', NULL),
			('item-archived', 'user-active', 'Archived', '2025-01-01T00:00:00Z', '{}', '2025-06-01T00:00:00Z')`)
	require.NoError(t, err)

	items, err := repo.ListByOwner(t.Context(), "user-active", ports.ItemListFilter{Status: ports.ItemStatusActive})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "item-active", items[0].GetID())
}

func TestItemRepositoryListByOwnerShouldReturnArchivedItemsWhenStatusIsArchived(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-archived")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata, archived_at)
		VALUES
			('item-act', 'user-archived', 'Active', '2025-01-01T00:00:00Z', '{}', NULL),
			('item-arc', 'user-archived', 'Archived', '2025-01-01T00:00:00Z', '{}', '2025-06-01T00:00:00Z')`)
	require.NoError(t, err)

	items, err := repo.ListByOwner(t.Context(), "user-archived", ports.ItemListFilter{Status: ports.ItemStatusArchived})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "item-arc", items[0].GetID())
}

func TestItemRepositoryListByOwnerShouldBatchLoadPhotos(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-batch")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES
			('item-b1', 'user-batch', 'Item 1', '2025-01-01T00:00:00Z', '{}'),
			('item-b2', 'user-batch', 'Item 2', '2025-01-02T00:00:00Z', '{}')`)
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO item_photos (id, item_id, media_key, position, created_at)
		VALUES
			('ph-1', 'item-b1', 'a.jpg', 0, '2025-06-01T10:00:00Z'),
			('ph-2', 'item-b2', 'b.jpg', 0, '2025-06-01T11:00:00Z')`)
	require.NoError(t, err)

	items, err := repo.ListByOwner(t.Context(), "user-batch", ports.ItemListFilter{Status: ports.ItemStatusAll})
	require.NoError(t, err)
	require.Len(t, items, 2)

	byID := make(map[string]domain.Item)
	for _, it := range items {
		byID[it.GetID()] = it
	}
	require.Len(t, byID["item-b1"].Photos, 1)
	require.Equal(t, "a.jpg", byID["item-b1"].Photos[0].MediaKey)
	require.Len(t, byID["item-b2"].Photos, 1)
	require.Equal(t, "b.jpg", byID["item-b2"].Photos[0].MediaKey)
}

// ── CountByLocation ───────────────────────────────────────────────────────────

func TestItemRepositoryCountByLocationShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.CountByLocation(ctx, "loc-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemRepositoryCountByLocationShouldReturnCount(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-count")

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO locations (id, owner_id, label, created_at)
		VALUES ('loc-1', 'user-count', 'Closet', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, location_id, created_at, metadata)
		VALUES
			('item-c1', 'user-count', 'Item 1', 'loc-1', '2025-01-01T00:00:00Z', '{}'),
			('item-c2', 'user-count', 'Item 2', 'loc-1', '2025-01-02T00:00:00Z', '{}'),
			('item-c3', 'user-count', 'Item 3', NULL, '2025-01-03T00:00:00Z', '{}')`)
	require.NoError(t, err)

	count, err := repo.CountByLocation(t.Context(), "loc-1")
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

// ── SavePhoto ─────────────────────────────────────────────────────────────────

func TestItemRepositorySavePhotoShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.SavePhoto(ctx, "item-1", "photo-1", "key.jpg", 0)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemRepositorySavePhotoShouldInsertPhotoRecord(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-photo")

	var item domain.Item
	item.ID = "item-photo"
	item.OwnerID = "user-photo"
	item.Name = "Photo Item"
	item.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}
	require.NoError(t, repo.Save(t.Context(), item))

	err := repo.SavePhoto(t.Context(), "item-photo", "photo-id-1", "key-a.jpg", 0)
	require.NoError(t, err)

	got, err := repo.Get(t.Context(), "item-photo")
	require.NoError(t, err)
	require.Len(t, got.Photos, 1)
	require.Equal(t, "key-a.jpg", got.Photos[0].MediaKey)
	require.Equal(t, "photo-id-1", got.Photos[0].ID)
}

// ── DeletePhoto ───────────────────────────────────────────────────────────────

func TestItemRepositoryDeletePhotoShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := repo.DeletePhoto(ctx, "item-1", "key.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemRepositoryDeletePhotoShouldRemovePhotoByItemIDAndMediaKey(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-dphoto")

	var item domain.Item
	item.ID = "item-dphoto"
	item.OwnerID = "user-dphoto"
	item.Name = "Del Photo Item"
	item.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}
	require.NoError(t, repo.Save(t.Context(), item))
	require.NoError(t, repo.SavePhoto(t.Context(), "item-dphoto", "photo-keep", "keep.jpg", 0))
	require.NoError(t, repo.SavePhoto(t.Context(), "item-dphoto", "photo-del", "delete.jpg", 1))

	err := repo.DeletePhoto(t.Context(), "item-dphoto", "delete.jpg")
	require.NoError(t, err)

	got, err := repo.Get(t.Context(), "item-dphoto")
	require.NoError(t, err)
	require.Len(t, got.Photos, 1)
	require.Equal(t, "keep.jpg", got.Photos[0].MediaKey)
}

// ── ListPhotoKeys ─────────────────────────────────────────────────────────────

func TestItemRepositoryListPhotoKeysShouldReturnErrWhenContextCancelled(t *testing.T) {
	repo, _ := newItemRepo(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := repo.ListPhotoKeys(ctx, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemRepositoryListPhotoKeysShouldReturnOrderedKeys(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-keys")

	var item domain.Item
	item.ID = "item-keys"
	item.OwnerID = "user-keys"
	item.Name = "Keys Item"
	item.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}
	require.NoError(t, repo.Save(t.Context(), item))
	require.NoError(t, repo.SavePhoto(t.Context(), "item-keys", "photo-b", "key-b.jpg", 1))
	require.NoError(t, repo.SavePhoto(t.Context(), "item-keys", "photo-a", "key-a.jpg", 0))

	keys, err := repo.ListPhotoKeys(t.Context(), "item-keys")
	require.NoError(t, err)
	require.Equal(t, []string{"key-a.jpg", "key-b.jpg"}, keys)
}

func TestItemRepositoryListPhotoKeysShouldReturnEmptyWhenNoPhotos(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-nokeys")

	var item domain.Item
	item.ID = "item-nokeys"
	item.OwnerID = "user-nokeys"
	item.Name = "No Photos"
	item.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}
	require.NoError(t, repo.Save(t.Context(), item))

	keys, err := repo.ListPhotoKeys(t.Context(), "item-nokeys")
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestItemRepositoryGetShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemRepository(db)
	db.Close()

	_, err := repo.Get(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositoryDeleteShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemRepository(db)
	db.Close()

	err := repo.Delete(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── DB closed error paths ─────────────────────────────────────────────────────

func TestItemRepositorySaveShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemRepository(db)
	db.Close()

	var item domain.Item
	item.ID = "item-1"
	err := repo.Save(t.Context(), item)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositoryListByOwnerShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemRepository(db)
	db.Close()

	_, err := repo.ListByOwner(t.Context(), "user-1", ports.ItemListFilter{Status: ports.ItemStatusAll})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositoryListByOwnerShouldReturnErrIOWhenMetadataIsInvalid(t *testing.T) {
	repo, db := newItemRepo(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-bad-meta', 'owner-x', 'Bad', '2025-01-01T00:00:00Z', 'not-json')`)
	require.NoError(t, err)

	_, err = repo.ListByOwner(t.Context(), "owner-x", ports.ItemListFilter{Status: ports.ItemStatusAll})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositoryListByOwnerShouldReturnErrIOWhenPhotoCreatedAtIsInvalid(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-badphoto")

	var item domain.Item
	item.ID = "item-badphoto"
	item.OwnerID = "user-badphoto"
	item.Name = "Bad Photo"
	item.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}
	require.NoError(t, repo.Save(t.Context(), item))

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO item_photos (id, item_id, media_key, position, created_at)
		VALUES ('photo-bad', 'item-badphoto', 'bad.jpg', 0, 'not-a-date')`)
	require.NoError(t, err)

	_, err = repo.ListByOwner(t.Context(), "user-badphoto", ports.ItemListFilter{Status: ports.ItemStatusAll})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositoryCountByLocationShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemRepository(db)
	db.Close()

	_, err := repo.CountByLocation(t.Context(), "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositorySavePhotoShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemRepository(db)
	db.Close()

	err := repo.SavePhoto(t.Context(), "item-1", "photo-1", "key.jpg", 0)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositoryDeletePhotoShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemRepository(db)
	db.Close()

	err := repo.DeletePhoto(t.Context(), "item-1", "key.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositoryListPhotoKeysShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	repo := sqlstore.NewItemRepository(db)
	db.Close()

	_, err := repo.ListPhotoKeys(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

// ── SellerURL and PurchaseCurrency ────────────────────────────────────────────

func TestItemRepositorySaveShouldPersistSellerURLAndPurchaseCurrency(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-seller")

	sellerURL := "https://example.com/jacket"
	currency := "USD"
	var item domain.Item
	item.ID = "item-seller-1"
	item.OwnerID = "user-seller"
	item.Name = "Jacket"
	item.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}
	item.SellerURL = &sellerURL
	item.PurchaseCurrency = &currency

	require.NoError(t, repo.Save(t.Context(), item))

	got, err := repo.Get(t.Context(), "item-seller-1")
	require.NoError(t, err)
	require.NotNil(t, got.SellerURL)
	require.Equal(t, "https://example.com/jacket", *got.SellerURL)
	require.NotNil(t, got.PurchaseCurrency)
	require.Equal(t, "USD", *got.PurchaseCurrency)
}

func TestItemRepositorySaveShouldPersistNilSellerURLAndPurchaseCurrency(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-nourl")

	var item domain.Item
	item.ID = "item-nourl"
	item.OwnerID = "user-nourl"
	item.Name = "Plain Shirt"
	item.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}

	require.NoError(t, repo.Save(t.Context(), item))

	got, err := repo.Get(t.Context(), "item-nourl")
	require.NoError(t, err)
	require.Nil(t, got.SellerURL)
	require.Nil(t, got.PurchaseCurrency)
}

func TestItemRepositoryListByOwnerShouldReturnSellerURLAndPurchaseCurrency(t *testing.T) {
	repo, db := newItemRepo(t)
	seedUserForItem(t, db, "user-list-seller")

	sellerURL := "https://shop.example.com/item"
	currency := "EUR"
	var item domain.Item
	item.ID = "item-list-seller"
	item.OwnerID = "user-list-seller"
	item.Name = "Shoes"
	item.CreatedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}
	item.SellerURL = &sellerURL
	item.PurchaseCurrency = &currency

	require.NoError(t, repo.Save(t.Context(), item))

	items, err := repo.ListByOwner(t.Context(), "user-list-seller", ports.ItemListFilter{Status: ports.ItemStatusAll})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].SellerURL)
	require.Equal(t, "https://shop.example.com/item", *items[0].SellerURL)
	require.NotNil(t, items[0].PurchaseCurrency)
	require.Equal(t, "EUR", *items[0].PurchaseCurrency)
}

// ── Test doubles ──────────────────────────────────────────────────────────────

type stubDB struct {
	err error
}

func (s *stubDB) BeginTx(_ context.Context, _ *sql.TxOptions) (*sql.Tx, error) {
	return nil, s.err
}

func (s *stubDB) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, s.err
}

func (s *stubDB) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	return nil, s.err
}

func (s *stubDB) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	panic("QueryRowContext not expected in this test")
}

// ── Tests using test doubles ──────────────────────────────────────────────────

func TestItemRepositorySaveShouldReturnErrIOWhenBeginTxFails(t *testing.T) {
	stub := &stubDB{err: errors.New("begin failed")}
	repo := sqlstore.NewItemRepository(stub)

	var item domain.Item
	item.ID = "item-1"
	err := repo.Save(t.Context(), item)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositorySavePhotoShouldReturnErrIOWhenExecContextFails(t *testing.T) {
	stub := &stubDB{err: errors.New("exec failed")}
	repo := sqlstore.NewItemRepository(stub)

	err := repo.SavePhoto(t.Context(), "item-1", "photo-1", "key.jpg", 0)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositoryDeletePhotoShouldReturnErrIOWhenExecContextFails(t *testing.T) {
	stub := &stubDB{err: errors.New("exec failed")}
	repo := sqlstore.NewItemRepository(stub)

	err := repo.DeletePhoto(t.Context(), "item-1", "key.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositoryListByOwnerShouldReturnErrIOWhenQueryContextFails(t *testing.T) {
	stub := &stubDB{err: errors.New("query failed")}
	repo := sqlstore.NewItemRepository(stub)

	_, err := repo.ListByOwner(t.Context(), "user-1", ports.ItemListFilter{Status: ports.ItemStatusAll})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemRepositoryListPhotoKeysShouldReturnErrIOWhenQueryContextFails(t *testing.T) {
	stub := &stubDB{err: errors.New("query failed")}
	repo := sqlstore.NewItemRepository(stub)

	_, err := repo.ListPhotoKeys(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}
