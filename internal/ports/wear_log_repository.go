package ports

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
)

// WearLogRepository is the storage port for WearLog persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type WearLogRepository interface {
	// Get retrieves a single wear log by ID.
	// Returns domain.ErrNotFound if no wear log with that ID exists.
	Get(ctx context.Context, id string) (domain.WearLog, error)

	// Save upserts the wear log row.
	Save(ctx context.Context, log domain.WearLog) error

	// Delete removes the wear log row identified by id.
	// Returns domain.ErrNotFound if no wear log with that ID exists.
	Delete(ctx context.Context, id string) error

	// ListByItem returns all wear logs for itemID, ordered by worn_on descending.
	ListByItem(ctx context.Context, itemID string) ([]domain.WearLog, error)
}
