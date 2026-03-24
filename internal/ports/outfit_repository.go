package ports

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
)

// OutfitRepository is the storage port for Outfit persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
//
// Get and ListByOwner must eagerly load outfit items and photos to avoid N+1 queries.
// Item and photo methods are kept separate from Save so callers can manage them
// independently without re-saving the full outfit.
type OutfitRepository interface {
	// Get retrieves a single outfit by ID, including its items and photos.
	// Returns domain.ErrNotFound if no outfit with that ID exists.
	Get(ctx context.Context, id string) (domain.Outfit, error)

	// Save upserts the outfit row. It does NOT touch item or photo records; use
	// SaveItem / DeleteItem and SavePhoto / DeletePhoto for those.
	Save(ctx context.Context, outfit domain.Outfit) error

	// Delete removes the outfit row identified by id.
	// Items and photos are removed via FK cascade.
	Delete(ctx context.Context, id string) error

	// ListByOwner returns all outfits belonging to ownerID, each including
	// its items and photos. Implementations must avoid N+1 queries.
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Outfit, error)

	// SaveItem inserts the outfit-item link for outfitID + itemID at position.
	SaveItem(ctx context.Context, outfitID, itemID string, position int) error

	// DeleteItem removes the outfit-item link for outfitID + itemID.
	DeleteItem(ctx context.Context, outfitID, itemID string) error

	// ListItemIDs returns the ordered item IDs for outfitID.
	ListItemIDs(ctx context.Context, outfitID string) ([]string, error)

	// SavePhoto inserts a photo record linking photoID and mediaKey to outfitID
	// at the given position.
	SavePhoto(ctx context.Context, outfitID, photoID, mediaKey string, position int) error

	// DeletePhoto removes the photo record identified by outfitID + mediaKey.
	DeletePhoto(ctx context.Context, outfitID, mediaKey string) error
}
