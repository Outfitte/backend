package ports

import (
	"context"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
)

// OutfitLogRepository is the storage port for OutfitLog persistence.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type OutfitLogRepository interface {
	// Get retrieves a single outfit log by ID. WearLogIDs is populated from the
	// outfit_log_wear_logs join table.
	// Returns domain.ErrNotFound if no outfit log with that ID exists.
	Get(ctx context.Context, id string) (domain.OutfitLog, error)

	// Save upserts the outfit log row only (not wear log links). Callers are
	// responsible for transactional consistency when calling LinkWearLog after
	// Save to attach wear log links.
	Save(ctx context.Context, log domain.OutfitLog) error

	// Delete removes the outfit log row identified by id.
	// Join table rows cascade via FK.
	// Returns domain.ErrNotFound if no outfit log with that ID exists.
	Delete(ctx context.Context, id string) error

	// ListByOutfit returns all logs for outfitID, ordered by worn_on descending.
	ListByOutfit(ctx context.Context, outfitID string) ([]domain.OutfitLog, error)

	// ListByOwnerDateRange returns logs belonging to ownerID (the user/owner ID,
	// not an outfit ID) where worn_on is between from and to (inclusive), ordered
	// by worn_on ascending. Each log includes its linked wear log IDs.
	// Callers must ensure from is not after to; behaviour is undefined otherwise.
	ListByOwnerDateRange(ctx context.Context, ownerID string, from, to time.Time) ([]domain.OutfitLog, error)

	// LinkWearLog inserts a row into the outfit_log_wear_logs join table.
	LinkWearLog(ctx context.Context, outfitLogID, wearLogID string) error

	// LinkedWearLogIDs returns the wear log IDs linked to the given outfit log.
	// Use this when only the IDs are needed and the full log row is not required.
	LinkedWearLogIDs(ctx context.Context, outfitLogID string) ([]string, error)
}
