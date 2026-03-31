package ports

import (
	"context"
	"time"

	"github.com/outfitte/backend/internal/domain"
)

// OutfitLogRepository is the storage port for OutfitLog persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type OutfitLogRepository interface {
	// Get retrieves a single outfit log by ID. The returned log includes its
	// linked wear log IDs.
	// Returns domain.ErrNotFound if no outfit log with that ID exists.
	Get(ctx context.Context, id string) (domain.OutfitLog, error)

	// Save persists the outfit log (not its wear log links). Callers are
	// responsible for consistency when calling LinkWearLog after Save to attach
	// wear log associations.
	Save(ctx context.Context, log domain.OutfitLog) error

	// Delete removes the outfit log identified by id, including all of its wear
	// log associations.
	// Returns domain.ErrNotFound if no outfit log with that ID exists.
	Delete(ctx context.Context, id string) error

	// ListByOutfit returns all logs for outfitID, sorted by WornOn descending.
	ListByOutfit(ctx context.Context, outfitID string) ([]domain.OutfitLog, error)

	// ListByOwnerDateRange returns logs belonging to ownerID (the user/owner ID,
	// not an outfit ID) where WornOn is within [from, to] inclusive, sorted by
	// WornOn ascending. Each log includes its linked wear log IDs.
	// Callers must ensure from is not after to; behaviour is undefined otherwise.
	ListByOwnerDateRange(ctx context.Context, ownerID string, from, to time.Time) ([]domain.OutfitLog, error)

	// LinkWearLog associates a wear log with an outfit log.
	LinkWearLog(ctx context.Context, outfitLogID, wearLogID string) error

	// LinkedWearLogIDs returns the wear log IDs associated with the given outfit log.
	// Use this when only the IDs are needed and the full log is not required.
	LinkedWearLogIDs(ctx context.Context, outfitLogID string) ([]string, error)
}
