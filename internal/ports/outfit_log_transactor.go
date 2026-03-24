package ports

import (
	"context"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
)

// OutfitLogTransactor is the port for atomic outfit log operations.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type OutfitLogTransactor interface {
	// CreateOutfitLog atomically creates the outfit log, creates one wear log
	// per item in wearLogs, creates the associations linking them, and returns
	// the persisted outfit log with its WearLogIDs populated.
	CreateOutfitLog(ctx context.Context, log domain.OutfitLog, wearLogs []domain.WearLog) (domain.OutfitLog, error)

	// DeleteOutfitLog atomically deletes the outfit log, all of its wear log
	// associations, and all linked wear logs.
	DeleteOutfitLog(ctx context.Context, outfitLogID string) error

	// UpdateOutfitLogDate atomically updates the outfit log's worn_on date
	// and propagates the new date to all linked wear logs.
	UpdateOutfitLogDate(ctx context.Context, outfitLogID string, newDate time.Time) error
}
