package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// Provider is a SQL-backed implementation of ports.StorageProvider[T].
type Provider[T ports.Entity] struct {
	db *sql.DB
}

// NewProvider creates a Provider backed by the given *sql.DB.
func NewProvider[T ports.Entity](db *sql.DB) *Provider[T] {
	return &Provider[T]{db: db}
}

// Get retrieves the entity with the given id.
// Returns domain.ErrNotFound if no row matches.
func (p *Provider[T]) Get(ctx context.Context, id string) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	switch any(&zero).(type) {
	case *domain.Item:
		item, err := getItem(ctx, p.db, id)
		if err != nil {
			return zero, err
		}
		return any(item).(T), nil
	default:
		return zero, fmt.Errorf("unsupported entity type %T", zero)
	}
}

// List returns all stored entities.
func (p *Provider[T]) List(ctx context.Context) ([]T, error) {
	return nil, errors.New("not implemented")
}

// Save creates or replaces the entity.
func (p *Provider[T]) Save(ctx context.Context, entity T) error {
	return errors.New("not implemented")
}

// Delete removes the entity.
func (p *Provider[T]) Delete(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func getItem(ctx context.Context, db *sql.DB, id string) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}

	const q = `
		SELECT id, owner_id, name, brand, category_id, color,
		       location_id, purchase_price, purchase_date, created_at, metadata
		FROM items WHERE id = ?`

	var (
		itemID        string
		ownerID       string
		name          string
		brand         sql.NullString
		categoryID    sql.NullString
		color         sql.NullString
		locationID    sql.NullString
		purchasePrice sql.NullString
		purchaseDate  sql.NullString
		createdAt     string
		metadataRaw   string
	)

	err := db.QueryRowContext(ctx, q, id).Scan(
		&itemID, &ownerID, &name, &brand, &categoryID, &color,
		&locationID, &purchasePrice, &purchaseDate, &createdAt, &metadataRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Item{}, fmt.Errorf("%w: id %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return domain.Item{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return domain.Item{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	var metadata domain.ItemMetadata
	if err := json.Unmarshal([]byte(metadataRaw), &metadata); err != nil {
		return domain.Item{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	item := domain.Item{}
	item.ID = itemID
	item.OwnerID = ownerID
	item.Name = name
	item.Metadata = metadata
	item.CreatedAt = parsedCreatedAt
	// PhotoKeys is populated via a separate item_photos query; left nil here
	// until the photo-loading sub-issue is implemented.

	if brand.Valid {
		item.Brand = &brand.String
	}
	if categoryID.Valid {
		item.CategoryID = &categoryID.String
	}
	if color.Valid {
		item.Color = &color.String
	}
	if locationID.Valid {
		item.LocationID = &locationID.String
	}
	if purchasePrice.Valid {
		item.PurchasePrice = &purchasePrice.String
	}
	if purchaseDate.Valid {
		t, err := time.Parse(time.RFC3339, purchaseDate.String)
		if err != nil {
			return domain.Item{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		item.PurchaseDate = &t
	}

	return item, nil
}
