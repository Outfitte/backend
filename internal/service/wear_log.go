package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// WearLogService manages wear log entries for wardrobe items.
type WearLogService struct {
	wearLogs ports.WearLogRepository
	items    ports.ItemRepository
	shares   shareAccessChecker
}

// NewWearLogService constructs a WearLogService backed by the given repositories.
func NewWearLogService(wearLogs ports.WearLogRepository, items ports.ItemRepository, shares shareAccessChecker) *WearLogService {
	return &WearLogService{wearLogs: wearLogs, items: items, shares: shares}
}

// LogWear records a wear event for itemID on the given date.
func (s *WearLogService) LogWear(ctx context.Context, callerID, itemID string, wornOn time.Time, notes *string) (domain.WearLog, error) {
	if err := ctx.Err(); err != nil {
		return domain.WearLog{}, err
	}
	if _, err := s.getOwnedItem(ctx, callerID, itemID); err != nil {
		return domain.WearLog{}, err
	}
	if wornOn.UTC().After(time.Now().UTC()) {
		return domain.WearLog{}, domain.ErrFutureDateNotAllowed
	}
	return s.saveNewWearLog(ctx, callerID, itemID, wornOn, notes)
}

func (s *WearLogService) saveNewWearLog(ctx context.Context, callerID, itemID string, wornOn time.Time, notes *string) (domain.WearLog, error) {
	var log domain.WearLog
	log.ID = uuid.NewString()
	log.ItemID = itemID
	log.OwnerID = callerID
	log.WornOn = wornOn.UTC()
	log.Notes = notes
	log.CreatedAt = time.Now().UTC()
	if err := s.wearLogs.Save(ctx, log); err != nil {
		return domain.WearLog{}, err
	}
	return log, nil
}

// ListByItem returns all wear logs for itemID, ordered by worn_on descending.
func (s *WearLogService) ListByItem(ctx context.Context, callerID, itemID string) ([]domain.WearLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.verifyItemAccess(ctx, callerID, itemID); err != nil {
		return nil, err
	}
	return s.wearLogs.ListByItem(ctx, itemID)
}

func (s *WearLogService) verifyItemAccess(ctx context.Context, callerID, itemID string) error {
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return err
	}
	if item.OwnerID == callerID {
		return nil
	}
	ok, err := s.shares.HasReadAccess(ctx, callerID, domain.ShareTargetItem, itemID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrForbidden
	}
	return nil
}

// DeleteWearLog removes the wear log identified by logID.
func (s *WearLogService) DeleteWearLog(ctx context.Context, callerID, logID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := s.getOwnedWearLog(ctx, callerID, logID); err != nil {
		return err
	}
	return s.wearLogs.Delete(ctx, logID)
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
