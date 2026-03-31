package ports

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
)

// LocationRepository is the storage port for Location persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type LocationRepository interface {
	// Get retrieves a single location by ID.
	// Returns domain.ErrNotFound if no location with that ID exists.
	Get(ctx context.Context, id string) (domain.Location, error)

	// Save creates or updates the location.
	Save(ctx context.Context, location domain.Location) error

	// Delete removes the location identified by id.
	// Returns domain.ErrNotFound if no location with that ID exists.
	Delete(ctx context.Context, id string) error

	// ListByOwner returns all locations belonging to ownerID.
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Location, error)

	// HasChildren reports whether locationID has any child locations.
	HasChildren(ctx context.Context, locationID string) (bool, error)
}
