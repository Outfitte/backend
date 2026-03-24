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
	// Returns domain.ErrNotFound if no outfit with that ID exists.
	Delete(ctx context.Context, id string) error

	// ListByOwner returns all outfits belonging to ownerID, each including
	// its items and photos. Implementations must avoid N+1 queries.
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Outfit, error)

	// SaveItem upserts the outfit-item link for outfitID + itemID, setting
	// position to the provided value. If the link already exists the position
	// is updated in place.
	SaveItem(ctx context.Context, outfitID, itemID string, position int) error

	// DeleteItem removes the outfit-item link for outfitID + itemID.
	DeleteItem(ctx context.Context, outfitID, itemID string) error

	// ListItemIDs returns the ordered item IDs for outfitID. Use this when
	// only IDs are needed; prefer Get when the full outfit is required.
	ListItemIDs(ctx context.Context, outfitID string) ([]string, error)

	// SavePhoto upserts a photo record linking photoID and mediaKey to outfitID
	// at the given position. If a record for (outfitID, mediaKey) already exists
	// the position is updated in place.
	SavePhoto(ctx context.Context, outfitID, photoID, mediaKey string, position int) error

	// DeletePhoto removes the photo record identified by outfitID + mediaKey.
	// mediaKey is used as the deletion key (rather than photoID) because it is
	// the stable identifier that callers receive from the media provider and
	// can reliably supply without a prior Get round-trip.
	DeletePhoto(ctx context.Context, outfitID, mediaKey string) error
}
