package ports

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
)

// WearLogRepository is the storage port for WearLog persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type WearLogRepository interface {
	// Get retrieves a single wear log by ID.
	// Returns domain.ErrNotFound if no wear log with that ID exists.
	Get(ctx context.Context, id string) (domain.WearLog, error)

	// Save creates or updates the wear log.
	Save(ctx context.Context, log domain.WearLog) error

	// Delete removes the wear log identified by id.
	// Returns domain.ErrNotFound if no wear log with that ID exists.
	Delete(ctx context.Context, id string) error

	// ListByItem returns all wear logs for itemID, ordered by WornOn descending.
	ListByItem(ctx context.Context, itemID string) ([]domain.WearLog, error)

	// LatestByItem returns the most recent wear log for itemID, or nil if none exist.
	LatestByItem(ctx context.Context, itemID string) (*domain.WearLog, error)

	// CountByItem returns the number of wear logs for itemID.
	CountByItem(ctx context.Context, itemID string) (int, error)
}
