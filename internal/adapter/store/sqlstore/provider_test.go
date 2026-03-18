package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/outfitte/outfitte/internal/adapter/store/sqlstore"
	"github.com/outfitte/outfitte/internal/domain"
)

func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	require.NoError(t, sqlstore.RunMigrations(t.Context(), db))
	return db
}

func TestGetItemShouldReturnErrWhenContextCancelled(t *testing.T) {
	db := openMigratedDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	p := sqlstore.NewProvider[domain.Item](db)
	_, err := p.Get(ctx, "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetItemShouldReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	db := openMigratedDB(t)
	p := sqlstore.NewProvider[domain.Item](db)
	_, err := p.Get(t.Context(), "nonexistent-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetItemShouldReturnItemWhenRowExists(t *testing.T) {
	db := openMigratedDB(t)

	// Seed a user (required by FK) then an item row.
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-1', 'a@b.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, brand, category_id, color, location_id,
		                   purchase_price, purchase_date, created_at, metadata)
		VALUES ('item-1', 'user-1', 'Blue Jeans', 'Levi''s', NULL, 'Blue', NULL,
		        NULL, NULL, '2025-06-01T10:00:00Z', '{"Fields":{"fit":"slim"}}')`)
	require.NoError(t, err)

	p := sqlstore.NewProvider[domain.Item](db)
	item, err := p.Get(t.Context(), "item-1")
	require.NoError(t, err)

	require.Equal(t, "item-1", item.GetID())
	require.Equal(t, "user-1", item.OwnerID)
	require.Equal(t, "Blue Jeans", item.Name)
	require.NotNil(t, item.Brand)
	require.Equal(t, "Levi's", *item.Brand)
	require.Equal(t, "slim", item.Metadata.Fields["fit"])
}

func TestGetItemShouldReturnItemWithPurchaseDateWhenSet(t *testing.T) {
	db := openMigratedDB(t)

	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-2', 'b@b.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, purchase_price, purchase_date, created_at, metadata)
		VALUES ('item-2', 'user-2', 'Jacket', '120.00', '2024-03-15T00:00:00Z',
		        '2025-06-01T10:00:00Z', '{}')`)
	require.NoError(t, err)

	p := sqlstore.NewProvider[domain.Item](db)
	item, err := p.Get(t.Context(), "item-2")
	require.NoError(t, err)

	require.NotNil(t, item.PurchaseDate)
	require.Equal(t, time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), *item.PurchaseDate)
	require.NotNil(t, item.PurchasePrice)
	require.Equal(t, "120.00", *item.PurchasePrice)
}

func TestGetShouldReturnErrWhenEntityTypeIsUnsupported(t *testing.T) {
	db := openMigratedDB(t)
	p := sqlstore.NewProvider[domain.User](db)
	_, err := p.Get(t.Context(), "user-1")
	require.Error(t, err)
}

func TestListItemsShouldReturnErrWhenContextCancelled(t *testing.T) {
	db := openMigratedDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	p := sqlstore.NewProvider[domain.Item](db)
	_, err := p.List(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestListItemsShouldReturnErrIOWhenMetadataIsInvalid(t *testing.T) {
	db := openMigratedDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-bad-meta', 'owner-1', 'Bad Item', '2025-01-01T00:00:00Z', 'not-json')`)
	require.NoError(t, err)

	p := sqlstore.NewProvider[domain.Item](db)
	_, err = p.List(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListShouldReturnErrWhenEntityTypeIsUnsupported(t *testing.T) {
	db := openMigratedDB(t)
	p := sqlstore.NewProvider[domain.User](db)
	_, err := p.List(t.Context())
	require.Error(t, err)
}

func TestListItemsShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	db.Close()
	p := sqlstore.NewProvider[domain.Item](db)
	_, err := p.List(t.Context())
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSaveItemShouldReturnErrWhenContextCancelled(t *testing.T) {
	db := openMigratedDB(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	p := sqlstore.NewProvider[domain.Item](db)
	var item domain.Item
	item.ID = "item-1"
	err := p.Save(ctx, item)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSaveItemShouldReturnErrWhenEntityTypeUnsupported(t *testing.T) {
	db := openMigratedDB(t)
	p := sqlstore.NewProvider[domain.User](db)
	err := p.Save(t.Context(), domain.User{})
	require.Error(t, err)
}

func TestSaveItemShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	db.Close()
	p := sqlstore.NewProvider[domain.Item](db)
	var item domain.Item
	item.ID = "item-1"
	err := p.Save(t.Context(), item)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestSaveItemShouldPersistNewItem(t *testing.T) {
	db := openMigratedDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-save', 'save@b.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	brand := "Nike"
	item := domain.Item{}
	item.ID = "item-save-1"
	item.OwnerID = "user-save"
	item.Name = "Sneakers"
	item.Brand = &brand
	item.CreatedAt = time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{Fields: map[string]string{"size": "42"}}

	p := sqlstore.NewProvider[domain.Item](db)
	require.NoError(t, p.Save(t.Context(), item))

	got, err := p.Get(t.Context(), "item-save-1")
	require.NoError(t, err)
	require.Equal(t, "item-save-1", got.GetID())
	require.Equal(t, "user-save", got.OwnerID)
	require.Equal(t, "Sneakers", got.Name)
	require.NotNil(t, got.Brand)
	require.Equal(t, "Nike", *got.Brand)
	require.Equal(t, "42", got.Metadata.Fields["size"])
}

func TestSaveItemShouldReplaceExistingItem(t *testing.T) {
	db := openMigratedDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-replace', 'rep@b.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	item := domain.Item{}
	item.ID = "item-replace-1"
	item.OwnerID = "user-replace"
	item.Name = "Old Name"
	item.CreatedAt = time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	item.Metadata = domain.ItemMetadata{}

	p := sqlstore.NewProvider[domain.Item](db)
	require.NoError(t, p.Save(t.Context(), item))

	item.Name = "New Name"
	require.NoError(t, p.Save(t.Context(), item))

	got, err := p.Get(t.Context(), "item-replace-1")
	require.NoError(t, err)
	require.Equal(t, "New Name", got.Name)
}

func TestListItemsShouldReturnAllItemsWhenRowsExist(t *testing.T) {
	db := openMigratedDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES ('user-list', 'list@b.com', 'hash', 'member', '2025-01-01T00:00:00Z')`)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES
			('item-a', 'user-list', 'Item A', '2025-06-01T10:00:00Z', '{"Fields":{"color":"red"}}'),
			('item-b', 'user-list', 'Item B', '2025-06-02T10:00:00Z', '{}')`)
	require.NoError(t, err)

	p := sqlstore.NewProvider[domain.Item](db)
	items, err := p.List(t.Context())
	require.NoError(t, err)
	require.Len(t, items, 2)

	ids := []string{items[0].GetID(), items[1].GetID()}
	require.ElementsMatch(t, []string{"item-a", "item-b"}, ids)
}

func TestListItemsShouldReturnEmptySliceWhenNoRowsExist(t *testing.T) {
	db := openMigratedDB(t)
	p := sqlstore.NewProvider[domain.Item](db)
	items, err := p.List(t.Context())
	require.NoError(t, err)
	require.NotNil(t, items)
	require.Empty(t, items)
}

func TestDeleteShouldReturnErrWhenNotImplemented(t *testing.T) {
	db := openMigratedDB(t)
	p := sqlstore.NewProvider[domain.Item](db)
	err := p.Delete(t.Context(), "item-1")
	require.Error(t, err)
}

func TestGetItemShouldReturnErrIOWhenDBIsClosed(t *testing.T) {
	db := openMigratedDB(t)
	db.Close()
	p := sqlstore.NewProvider[domain.Item](db)
	_, err := p.Get(t.Context(), "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetItemShouldReturnErrIOWhenCreatedAtIsInvalid(t *testing.T) {
	db := openMigratedDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-bad-ts', 'owner-1', 'Bad Item', 'not-a-date', '{}')`)
	require.NoError(t, err)

	p := sqlstore.NewProvider[domain.Item](db)
	_, err = p.Get(t.Context(), "item-bad-ts")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetItemShouldReturnErrIOWhenMetadataIsInvalid(t *testing.T) {
	db := openMigratedDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, metadata)
		VALUES ('item-bad-meta', 'owner-1', 'Bad Item', '2025-01-01T00:00:00Z', 'not-json')`)
	require.NoError(t, err)

	p := sqlstore.NewProvider[domain.Item](db)
	_, err = p.Get(t.Context(), "item-bad-meta")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestGetItemShouldReturnItemWithAllOptionalFieldsWhenSet(t *testing.T) {
	db := openMigratedDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, category_id, color, location_id, created_at, metadata)
		VALUES ('item-full', 'owner-1', 'Full Item', 'cat-1', 'Red', 'loc-1',
		        '2025-01-01T00:00:00Z', '{}')`)
	require.NoError(t, err)

	p := sqlstore.NewProvider[domain.Item](db)
	item, err := p.Get(t.Context(), "item-full")
	require.NoError(t, err)

	require.NotNil(t, item.CategoryID)
	require.Equal(t, "cat-1", *item.CategoryID)
	require.NotNil(t, item.LocationID)
	require.Equal(t, "loc-1", *item.LocationID)
}

func TestGetItemShouldReturnErrIOWhenPurchaseDateIsInvalid(t *testing.T) {
	db := openMigratedDB(t)
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO items (id, owner_id, name, created_at, purchase_date, metadata)
		VALUES ('item-bad-pd', 'owner-1', 'Bad Item', '2025-01-01T00:00:00Z', 'not-a-date', '{}')`)
	require.NoError(t, err)

	p := sqlstore.NewProvider[domain.Item](db)
	_, err = p.Get(t.Context(), "item-bad-pd")
	require.ErrorIs(t, err, domain.ErrIO)
}
