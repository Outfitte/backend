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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var zero T
	switch any(&zero).(type) {
	case *domain.Item:
		items, err := listItems(ctx, p.db)
		if err != nil {
			return nil, err
		}
		result := make([]T, len(items))
		for i, item := range items {
			result[i] = any(item).(T)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported entity type %T", zero)
	}
}

func listItems(ctx context.Context, db *sql.DB) ([]domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	const q = `
		SELECT id, owner_id, name, brand, category_id, color,
		       location_id, purchase_price, purchase_date, created_at, metadata
		FROM items`

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	items := []domain.Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	for i, item := range items {
		photos, err := loadPhotos(ctx, db, item.ID)
		if err != nil {
			return nil, err
		}
		items[i].Photos = photos
	}
	return items, nil
}

func scanItem(rows *sql.Rows) (domain.Item, error) {
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

	if err := rows.Scan(
		&itemID, &ownerID, &name, &brand, &categoryID, &color,
		&locationID, &purchasePrice, &purchaseDate, &createdAt, &metadataRaw,
	); err != nil {
		return domain.Item{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return buildItem(itemID, ownerID, name, brand, categoryID, color, locationID, purchasePrice, purchaseDate, createdAt, metadataRaw)
}

func buildItem(
	itemID, ownerID, name string,
	brand, categoryID, color, locationID, purchasePrice, purchaseDate sql.NullString,
	createdAt, metadataRaw string,
) (domain.Item, error) {
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

func loadPhotos(ctx context.Context, db *sql.DB, itemID string) ([]domain.ItemPhoto, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	const q = `
		SELECT id, media_key, position, created_at
		FROM item_photos
		WHERE item_id = ?
		ORDER BY position ASC`

	rows, err := db.QueryContext(ctx, q, itemID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	photos := []domain.ItemPhoto{}
	for rows.Next() {
		var (
			id        string
			mediaKey  string
			position  int
			createdAt string
		)
		if err := rows.Scan(&id, &mediaKey, &position, &createdAt); err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		photos = append(photos, domain.ItemPhoto{
			ID:        id,
			MediaKey:  mediaKey,
			Position:  position,
			CreatedAt: parsedCreatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return photos, nil
}

// Save creates or replaces the entity.
func (p *Provider[T]) Save(ctx context.Context, entity T) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var zero T
	switch any(&zero).(type) {
	case *domain.Item:
		return saveItem(ctx, p.db, any(entity).(domain.Item))
	default:
		return fmt.Errorf("unsupported entity type %T", zero)
	}
}

// Delete removes the entity.
func (p *Provider[T]) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var zero T
	switch any(&zero).(type) {
	case *domain.Item:
		return deleteItem(ctx, p.db, id)
	default:
		return fmt.Errorf("unsupported entity type %T", zero)
	}
}

func deleteItem(ctx context.Context, db *sql.DB, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	const q = `DELETE FROM items WHERE id = ?`

	result, err := db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	if n == 0 {
		return fmt.Errorf("%w: id %s", domain.ErrNotFound, id)
	}
	return nil
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

	item, err := buildItem(itemID, ownerID, name, brand, categoryID, color, locationID, purchasePrice, purchaseDate, createdAt, metadataRaw)
	if err != nil {
		return domain.Item{}, err
	}
	photos, err := loadPhotos(ctx, db, id)
	if err != nil {
		return domain.Item{}, err
	}
	item.Photos = photos
	return item, nil
}

func saveItem(ctx context.Context, db *sql.DB, item domain.Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := upsertItemRow(ctx, tx, item); err != nil {
		return err
	}
	if err := replacePhotos(ctx, tx, item); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

func upsertItemRow(ctx context.Context, tx *sql.Tx, item domain.Item) error {
	metadataRaw, err := json.Marshal(item.Metadata)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	var purchaseDate *string
	if item.PurchaseDate != nil {
		s := item.PurchaseDate.UTC().Format(time.RFC3339)
		purchaseDate = &s
	}

	const q = `
		INSERT OR REPLACE INTO items
			(id, owner_id, name, brand, category_id, color, location_id,
			 purchase_price, purchase_date, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.ExecContext(ctx, q,
		item.ID,
		item.OwnerID,
		item.Name,
		item.Brand,
		item.CategoryID,
		item.Color,
		item.LocationID,
		item.PurchasePrice,
		purchaseDate,
		item.CreatedAt.UTC().Format(time.RFC3339),
		string(metadataRaw),
	)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

func replacePhotos(ctx context.Context, tx *sql.Tx, item domain.Item) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM item_photos WHERE item_id = ?`, item.ID); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	const q = `
		INSERT INTO item_photos (id, item_id, media_key, position, created_at)
		VALUES (?, ?, ?, ?, ?)`

	for _, photo := range item.Photos {
		if _, err := tx.ExecContext(ctx, q,
			photo.ID,
			item.ID,
			photo.MediaKey,
			photo.Position,
			photo.CreatedAt.UTC().Format(time.RFC3339),
		); err != nil {
			return fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
	}
	return nil
}
