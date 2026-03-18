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
