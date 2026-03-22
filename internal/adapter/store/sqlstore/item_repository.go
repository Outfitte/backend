package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

var _ ports.ItemRepository = (*ItemRepository)(nil)

// ItemRepository is a SQL-backed implementation of ports.ItemRepository.
type ItemRepository struct {
	db *sql.DB
}

// NewItemRepository creates an ItemRepository backed by the given *sql.DB.
func NewItemRepository(db *sql.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// Get retrieves a single item by ID, including its photo keys.
// Returns domain.ErrNotFound if no item with that ID exists.
func (r *ItemRepository) Get(ctx context.Context, id string) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	return getItem(ctx, r.db, id)
}

// Save upserts the item row. It does NOT touch photo records.
func (r *ItemRepository) Save(ctx context.Context, item domain.Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer tx.Rollback() //nolint:errcheck
	if err := upsertItemRow(ctx, tx, item); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// Delete removes the item row identified by id.
// Returns domain.ErrNotFound if no item with that ID exists.
func (r *ItemRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return deleteItem(ctx, r.db, id)
}

// ListByOwner returns all items belonging to ownerID that match filter.
// Each returned item includes its photo keys loaded in a single batch query.
func (r *ItemRepository) ListByOwner(ctx context.Context, ownerID string, filter ports.ItemListFilter) ([]domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items, err := r.queryItemsByOwner(ctx, ownerID, filter)
	if err != nil {
		return nil, err
	}
	if err := r.batchLoadPhotos(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ItemRepository) queryItemsByOwner(ctx context.Context, ownerID string, filter ports.ItemListFilter) ([]domain.Item, error) {
	q := `
		SELECT id, owner_id, name, brand, category_id, color,
		       location_id, purchase_price, purchase_date, created_at, metadata
		FROM items
		WHERE owner_id = ?`

	switch filter.Status {
	case ports.ItemStatusArchived:
		q += ` AND archived_at IS NOT NULL`
	case ports.ItemStatusAll:
		// no filter
	default: // ItemStatusActive is the default
		q += ` AND archived_at IS NULL`
	}

	rows, err := r.db.QueryContext(ctx, q, ownerID)
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
	return items, nil
}

func (r *ItemRepository) batchLoadPhotos(ctx context.Context, items []domain.Item) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	q := fmt.Sprintf(`
		SELECT item_id, id, media_key, position, created_at
		FROM item_photos
		WHERE item_id IN (%s)
		ORDER BY position ASC`, placeholders)

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	photosByItemID := make(map[string][]domain.ItemPhoto)
	for rows.Next() {
		var (
			itemID    string
			id        string
			mediaKey  string
			position  int
			createdAt string
		)
		if err := rows.Scan(&itemID, &id, &mediaKey, &position, &createdAt); err != nil {
			return fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		photosByItemID[itemID] = append(photosByItemID[itemID], domain.ItemPhoto{
			ID:        id,
			MediaKey:  mediaKey,
			Position:  position,
			CreatedAt: parsedCreatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	for i, it := range items {
		if photos, ok := photosByItemID[it.ID]; ok {
			items[i].Photos = photos
		} else {
			items[i].Photos = []domain.ItemPhoto{}
		}
	}
	return nil
}

// CountByLocation returns the number of items assigned to locationID.
func (r *ItemRepository) CountByLocation(ctx context.Context, locationID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	const q = `SELECT COUNT(*) FROM items WHERE location_id = ?`
	var count int
	if err := r.db.QueryRowContext(ctx, q, locationID).Scan(&count); err != nil {
		return 0, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return count, nil
}

// SavePhoto inserts a photo record linking photoID and mediaKey to itemID at the given position.
func (r *ItemRepository) SavePhoto(ctx context.Context, itemID, photoID, mediaKey string, position int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		INSERT INTO item_photos (id, item_id, media_key, position, created_at)
		VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, q, photoID, itemID, mediaKey, position, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// DeletePhoto removes the photo record identified by itemID + mediaKey.
func (r *ItemRepository) DeletePhoto(ctx context.Context, itemID, mediaKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `DELETE FROM item_photos WHERE item_id = ? AND media_key = ?`
	_, err := r.db.ExecContext(ctx, q, itemID, mediaKey)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}

// ListPhotoKeys returns the ordered media keys for all photos of itemID.
func (r *ItemRepository) ListPhotoKeys(ctx context.Context, itemID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `SELECT media_key FROM item_photos WHERE item_id = ? ORDER BY position ASC`
	rows, err := r.db.QueryContext(ctx, q, itemID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer rows.Close()

	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return keys, nil
}
