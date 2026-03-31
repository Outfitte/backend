package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/outfitte/backend/internal/domain"
)

func getItem(ctx context.Context, db itemDB, id string) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}

	const q = `
		SELECT id, owner_id, name, brand, category_id, color,
		       location_id, purchase_price, purchase_date, created_at, metadata,
		       archived_at, disposal_reason, seller_url, purchase_currency
		FROM items WHERE id = ?`

	var (
		itemID           string
		ownerID          string
		name             string
		brand            sql.NullString
		categoryID       sql.NullString
		color            sql.NullString
		locationID       sql.NullString
		purchasePrice    sql.NullString
		purchaseDate     sql.NullString
		createdAt        string
		metadataRaw      string
		archivedAt       sql.NullString
		disposalReason   sql.NullString
		sellerURL        sql.NullString
		purchaseCurrency sql.NullString
	)

	err := db.QueryRowContext(ctx, q, id).Scan(
		&itemID, &ownerID, &name, &brand, &categoryID, &color,
		&locationID, &purchasePrice, &purchaseDate, &createdAt, &metadataRaw,
		&archivedAt, &disposalReason, &sellerURL, &purchaseCurrency,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Item{}, fmt.Errorf("%w: id %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return domain.Item{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	item, err := buildItem(itemID, ownerID, name, brand, categoryID, color, locationID, purchasePrice, purchaseDate, createdAt, metadataRaw, archivedAt, disposalReason, sellerURL, purchaseCurrency)
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

func scanItem(rows *sql.Rows) (domain.Item, error) {
	var (
		itemID           string
		ownerID          string
		name             string
		brand            sql.NullString
		categoryID       sql.NullString
		color            sql.NullString
		locationID       sql.NullString
		purchasePrice    sql.NullString
		purchaseDate     sql.NullString
		createdAt        string
		metadataRaw      string
		archivedAt       sql.NullString
		disposalReason   sql.NullString
		sellerURL        sql.NullString
		purchaseCurrency sql.NullString
	)

	if err := rows.Scan(
		&itemID, &ownerID, &name, &brand, &categoryID, &color,
		&locationID, &purchasePrice, &purchaseDate, &createdAt, &metadataRaw,
		&archivedAt, &disposalReason, &sellerURL, &purchaseCurrency,
	); err != nil {
		return domain.Item{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return buildItem(itemID, ownerID, name, brand, categoryID, color, locationID, purchasePrice, purchaseDate, createdAt, metadataRaw, archivedAt, disposalReason, sellerURL, purchaseCurrency)
}

func buildItem(
	itemID, ownerID, name string,
	brand, categoryID, color, locationID, purchasePrice, purchaseDate sql.NullString,
	createdAt, metadataRaw string,
	archivedAt, disposalReason, sellerURL, purchaseCurrency sql.NullString,
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
	if archivedAt.Valid {
		t, err := time.Parse(time.RFC3339, archivedAt.String)
		if err != nil {
			return domain.Item{}, fmt.Errorf("%w: %w", domain.ErrIO, err)
		}
		item.ArchivedAt = &t
	}
	if disposalReason.Valid {
		r := domain.DisposalReason(disposalReason.String)
		item.DisposalReason = &r
	}
	if sellerURL.Valid {
		item.SellerURL = &sellerURL.String
	}
	if purchaseCurrency.Valid {
		item.PurchaseCurrency = &purchaseCurrency.String
	}

	return item, nil
}

func loadPhotos(ctx context.Context, db itemDB, itemID string) ([]domain.ItemPhoto, error) {
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

func deleteItem(ctx context.Context, db itemDB, id string) error {
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

// jsonMarshalFn is the function used to marshal item metadata.
// Exposed as a variable so whitebox tests can inject a failing implementation.
var jsonMarshalFn = json.Marshal

func upsertItemRow(ctx context.Context, tx *sql.Tx, item domain.Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	metadataRaw, err := jsonMarshalFn(item.Metadata)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	var purchaseDate *string
	if item.PurchaseDate != nil {
		s := item.PurchaseDate.UTC().Format(time.RFC3339)
		purchaseDate = &s
	}

	var archivedAt *string
	if item.ArchivedAt != nil {
		s := item.ArchivedAt.UTC().Format(time.RFC3339)
		archivedAt = &s
	}

	var disposalReason *string
	if item.DisposalReason != nil {
		s := string(*item.DisposalReason)
		disposalReason = &s
	}

	const q = `
		INSERT INTO items
			(id, owner_id, name, brand, category_id, color, location_id,
			 purchase_price, purchase_date, created_at, metadata,
			 archived_at, disposal_reason, seller_url, purchase_currency)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			owner_id          = excluded.owner_id,
			name              = excluded.name,
			brand             = excluded.brand,
			category_id       = excluded.category_id,
			color             = excluded.color,
			location_id       = excluded.location_id,
			purchase_price    = excluded.purchase_price,
			purchase_date     = excluded.purchase_date,
			created_at        = excluded.created_at,
			metadata          = excluded.metadata,
			archived_at       = excluded.archived_at,
			disposal_reason   = excluded.disposal_reason,
			seller_url        = excluded.seller_url,
			purchase_currency = excluded.purchase_currency`

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
		archivedAt,
		disposalReason,
		item.SellerURL,
		item.PurchaseCurrency,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	return nil
}
