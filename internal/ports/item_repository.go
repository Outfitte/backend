package ports

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
)

// ItemStatus values for ItemListFilter.Status.
const (
	ItemStatusActive   = "active"
	ItemStatusArchived = "archived"
	ItemStatusAll      = "all"
)

// ItemListFilter controls which items are returned by ListByOwner.
// Status accepts ItemStatusActive, ItemStatusArchived, or ItemStatusAll; defaults to ItemStatusActive.
type ItemListFilter struct {
	Status string
}

// ItemRepository is the storage port for Item persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
//
// Photo methods are kept separate from Save so that callers can manage photos
// independently without re-saving the full item.
type ItemRepository interface {
	// Get retrieves a single item by ID, including its photo keys.
	// Returns domain.ErrNotFound if no item with that ID exists.
	Get(ctx context.Context, id string) (domain.Item, error)

	// Save upserts the item row. It does NOT touch photo records; use
	// SavePhoto / DeletePhoto for photo management.
	Save(ctx context.Context, item domain.Item) error

	// Delete removes the item row identified by id.
	// Returns domain.ErrNotFound if no item with that ID exists.
	Delete(ctx context.Context, id string) error

	// ListByOwner returns all items belonging to ownerID that match filter.
	// Each returned item includes its photo keys. Implementations must avoid
	// N+1 queries when loading photos.
	ListByOwner(ctx context.Context, ownerID string, filter ItemListFilter) ([]domain.Item, error)

	// CountByLocation returns the number of items assigned to locationID.
	CountByLocation(ctx context.Context, locationID string) (int, error)

	// SavePhoto inserts a photo record linking photoID and mediaKey to itemID
	// at the given position.
	SavePhoto(ctx context.Context, itemID, photoID, mediaKey string, position int) error

	// DeletePhoto removes the photo record identified by itemID + mediaKey.
	DeletePhoto(ctx context.Context, itemID, mediaKey string) error

	// ListPhotoKeys returns the ordered media keys for all photos of itemID.
	ListPhotoKeys(ctx context.Context, itemID string) ([]string, error)
}
