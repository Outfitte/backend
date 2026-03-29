package json

import (
	"context"
	"sync"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

var _ ports.OutfitLogTransactor = (*OutfitLogTransactor)(nil)

// OutfitLogTransactor is a JSON file-backed implementation of ports.OutfitLogTransactor.
// Atomicity is achieved via an in-process mutex that serializes all transactional operations,
// preventing concurrent transactional calls from interleaving with each other.
//
// NOTE: This implementation does not provide true atomicity — if a step fails partway through,
// prior steps are not rolled back. This is an accepted limitation for single-instance personal use.
type OutfitLogTransactor struct {
	outfitLogs *OutfitLogRepository
	wearLogs   *WearLogRepository
	mu         sync.Mutex
}

// NewOutfitLogTransactor creates an OutfitLogTransactor using the given repositories.
func NewOutfitLogTransactor(outfitLogs *OutfitLogRepository, wearLogs *WearLogRepository) *OutfitLogTransactor {
	return &OutfitLogTransactor{
		outfitLogs: outfitLogs,
		wearLogs:   wearLogs,
	}
}

func (t *OutfitLogTransactor) CreateOutfitLog(ctx context.Context, log domain.OutfitLog, wearLogs []domain.WearLog) (domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return domain.OutfitLog{}, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.outfitLogs.Save(ctx, log); err != nil {
		return domain.OutfitLog{}, err
	}

	for _, wl := range wearLogs {
		if err := t.saveAndLinkWearLog(ctx, log.ID, wl); err != nil {
			return domain.OutfitLog{}, err
		}
	}

	return t.outfitLogs.Get(ctx, log.ID)
}

func (t *OutfitLogTransactor) saveAndLinkWearLog(ctx context.Context, outfitLogID string, wl domain.WearLog) error {
	if err := t.wearLogs.Save(ctx, wl); err != nil {
		return err
	}
	return t.outfitLogs.LinkWearLog(ctx, outfitLogID, wl.ID)
}

func (t *OutfitLogTransactor) DeleteOutfitLog(ctx context.Context, outfitLogID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	wearLogIDs, err := t.outfitLogs.LinkedWearLogIDs(ctx, outfitLogID)
	if err != nil {
		return err
	}

	for _, id := range wearLogIDs {
		if err := t.wearLogs.Delete(ctx, id); err != nil {
			return err
		}
	}

	return t.outfitLogs.Delete(ctx, outfitLogID)
}

func (t *OutfitLogTransactor) UpdateOutfitLogDate(ctx context.Context, outfitLogID string, newDate time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	ol, err := t.outfitLogs.Get(ctx, outfitLogID)
	if err != nil {
		return err
	}
	ol.WornOn = newDate

	if err := t.outfitLogs.Save(ctx, ol); err != nil {
		return err
	}

	// ol.WearLogIDs is safe to use here: OutfitLogRepository.Save preserves the
	// stored WearLogIDs, so ol still reflects the correct set from the Get above.
	for _, id := range ol.WearLogIDs {
		if err := t.updateWearLogDate(ctx, id, newDate); err != nil {
			return err
		}
	}

	return nil
}

func (t *OutfitLogTransactor) updateWearLogDate(ctx context.Context, wearLogID string, newDate time.Time) error {
	wl, err := t.wearLogs.Get(ctx, wearLogID)
	if err != nil {
		return err
	}
	wl.WornOn = newDate
	return t.wearLogs.Save(ctx, wl)
}
