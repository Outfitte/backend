package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// WearLogService manages wear log entries for wardrobe items.
type WearLogService struct {
	wearLogs ports.WearLogRepository
	items    ports.ItemRepository
}

// NewWearLogService constructs a WearLogService backed by the given repositories.
func NewWearLogService(wearLogs ports.WearLogRepository, items ports.ItemRepository) *WearLogService {
	return &WearLogService{wearLogs: wearLogs, items: items}
}

// LogWear records a wear event for itemID on the given date.
func (s *WearLogService) LogWear(ctx context.Context, callerID, itemID string, wornOn time.Time, notes *string) (domain.WearLog, error) {
	if err := ctx.Err(); err != nil {
		return domain.WearLog{}, err
	}
	item, err := s.getOwnedItem(ctx, callerID, itemID)
	if err != nil {
		return domain.WearLog{}, err
	}
	if wornOn.After(time.Now()) {
		return domain.WearLog{}, domain.ErrFutureDateNotAllowed
	}
	log, err := s.saveNewWearLog(ctx, callerID, itemID, wornOn, notes)
	if err != nil {
		return domain.WearLog{}, err
	}
	if err := s.updateItemWearStats(ctx, item, wornOn); err != nil {
		return domain.WearLog{}, err
	}
	return log, nil
}

func (s *WearLogService) saveNewWearLog(ctx context.Context, callerID, itemID string, wornOn time.Time, notes *string) (domain.WearLog, error) {
	var log domain.WearLog
	log.ID = uuid.NewString()
	log.ItemID = itemID
	log.OwnerID = callerID
	log.WornOn = wornOn
	log.Notes = notes
	log.CreatedAt = time.Now().UTC()
	if err := s.wearLogs.Save(ctx, log); err != nil {
		return domain.WearLog{}, err
	}
	return log, nil
}

func (s *WearLogService) updateItemWearStats(ctx context.Context, item domain.Item, wornOn time.Time) error {
	item.WearCount++
	item.LastWornAt = &wornOn
	return s.items.Save(ctx, item)
}

// ListByItem returns all wear logs for itemID, ordered by worn_on descending.
func (s *WearLogService) ListByItem(ctx context.Context, callerID, itemID string) ([]domain.WearLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := s.getOwnedItem(ctx, callerID, itemID); err != nil {
		return nil, err
	}
	return s.wearLogs.ListByItem(ctx, itemID)
}

// DeleteWearLog removes the wear log identified by logID and recomputes the item's wear stats.
func (s *WearLogService) DeleteWearLog(ctx context.Context, callerID, logID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	log, err := s.getOwnedWearLog(ctx, callerID, logID)
	if err != nil {
		return err
	}
	if err := s.wearLogs.Delete(ctx, logID); err != nil {
		return err
	}
	return s.recomputeItemWearStats(ctx, log.ItemID)
}

func (s *WearLogService) recomputeItemWearStats(ctx context.Context, itemID string) error {
	latest, err := s.wearLogs.LatestByItem(ctx, itemID)
	if err != nil {
		return err
	}
	count, err := s.wearLogs.CountByItem(ctx, itemID)
	if err != nil {
		return err
	}
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return err
	}
	item.WearCount = count
	if latest != nil {
		item.LastWornAt = &latest.WornOn
	} else {
		item.LastWornAt = nil
	}
	return s.items.Save(ctx, item)
}

// getOwnedItem fetches the item and verifies it belongs to callerID.
func (s *WearLogService) getOwnedItem(ctx context.Context, callerID, itemID string) (domain.Item, error) {
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.OwnerID != callerID {
		return domain.Item{}, domain.ErrForbidden
	}
	return item, nil
}

// getOwnedWearLog fetches the wear log and verifies it belongs to callerID.
func (s *WearLogService) getOwnedWearLog(ctx context.Context, callerID, logID string) (domain.WearLog, error) {
	log, err := s.wearLogs.Get(ctx, logID)
	if err != nil {
		return domain.WearLog{}, err
	}
	if log.OwnerID != callerID {
		return domain.WearLog{}, domain.ErrForbidden
	}
	return log, nil
}
