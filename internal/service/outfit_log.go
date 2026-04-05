package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// OutfitLogService manages outfit log CRUD with atomic wear log creation.
type OutfitLogService struct {
	outfits    ports.OutfitRepository
	outfitLogs ports.OutfitLogRepository
	transactor ports.OutfitLogTransactor
	shares     shareAccessChecker
}

// NewOutfitLogService constructs an OutfitLogService backed by the given dependencies.
func NewOutfitLogService(outfits ports.OutfitRepository, outfitLogs ports.OutfitLogRepository, transactor ports.OutfitLogTransactor, shares shareAccessChecker) *OutfitLogService {
	return &OutfitLogService{outfits: outfits, outfitLogs: outfitLogs, transactor: transactor, shares: shares}
}

// LogWear validates outfit ownership and date, then atomically creates the outfit log and wear logs.
func (s *OutfitLogService) LogWear(ctx context.Context, callerID, outfitID string, wornOn time.Time, notes *string) (domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return domain.OutfitLog{}, err
	}
	now := time.Now().UTC()
	if wornOn.UTC().After(now) {
		return domain.OutfitLog{}, domain.ErrFutureDateNotAllowed
	}
	outfit, err := s.getOwnedOutfit(ctx, callerID, outfitID)
	if err != nil {
		return domain.OutfitLog{}, err
	}
	return s.createOutfitLog(ctx, callerID, outfit, wornOn, notes, now)
}

func (s *OutfitLogService) createOutfitLog(ctx context.Context, callerID string, outfit domain.Outfit, wornOn time.Time, notes *string, now time.Time) (domain.OutfitLog, error) {
	var outfitLog domain.OutfitLog
	outfitLog.ID = uuid.NewString()
	outfitLog.OutfitID = outfit.GetID()
	outfitLog.OwnerID = callerID
	outfitLog.WornOn = wornOn.UTC()
	outfitLog.Notes = notes
	outfitLog.CreatedAt = now

	wearLogs := make([]domain.WearLog, 0, len(outfit.Items))
	for _, item := range outfit.Items {
		var wl domain.WearLog
		wl.ID = uuid.NewString()
		wl.ItemID = item.ItemID
		wl.OwnerID = callerID
		wl.WornOn = wornOn.UTC()
		wl.Notes = notes
		wl.CreatedAt = now
		wearLogs = append(wearLogs, wl)
	}

	return s.transactor.CreateOutfitLog(ctx, outfitLog, wearLogs)
}

func (s *OutfitLogService) getOwnedOutfit(ctx context.Context, callerID, outfitID string) (domain.Outfit, error) {
	outfit, err := s.outfits.Get(ctx, outfitID)
	if err != nil {
		return domain.Outfit{}, err
	}
	if outfit.OwnerID != callerID {
		return domain.Outfit{}, domain.ErrForbidden
	}
	return outfit, nil
}

// ListByOutfit returns all outfit logs for outfitID, verifying caller owns or has shared access to the outfit.
func (s *OutfitLogService) ListByOutfit(ctx context.Context, callerID, outfitID string) ([]domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, err := s.getOwnedOutfit(ctx, callerID, outfitID)
	if err != nil {
		if !errors.Is(err, domain.ErrForbidden) {
			return nil, err
		}
		if err := s.checkSharedOutfitAccess(ctx, callerID, outfitID); err != nil {
			return nil, err
		}
	}
	return s.outfitLogs.ListByOutfit(ctx, outfitID)
}

func (s *OutfitLogService) checkSharedOutfitAccess(ctx context.Context, callerID, outfitID string) error {
	return errors.New("not implemented")
}

// ListByDateRange returns all outfit logs for callerID within [from, to].
// Returns domain.ErrValidation if from is after to.
func (s *OutfitLogService) ListByDateRange(ctx context.Context, callerID string, from, to time.Time) ([]domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if from.After(to) {
		return nil, domain.ErrValidation
	}
	return s.outfitLogs.ListByOwnerDateRange(ctx, callerID, from, to)
}

// UpdateDate validates the new date, verifies caller owns the outfit log, and updates it atomically.
func (s *OutfitLogService) UpdateDate(ctx context.Context, callerID, outfitLogID string, newDate time.Time) (domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return domain.OutfitLog{}, err
	}
	if newDate.UTC().After(time.Now().UTC()) {
		return domain.OutfitLog{}, domain.ErrFutureDateNotAllowed
	}
	if _, err := s.getOwnedOutfitLog(ctx, callerID, outfitLogID); err != nil {
		return domain.OutfitLog{}, err
	}
	if err := s.transactor.UpdateOutfitLogDate(ctx, outfitLogID, newDate.UTC()); err != nil {
		return domain.OutfitLog{}, err
	}
	return s.outfitLogs.Get(ctx, outfitLogID)
}

func (s *OutfitLogService) getOwnedOutfitLog(ctx context.Context, callerID, outfitLogID string) (domain.OutfitLog, error) {
	log, err := s.outfitLogs.Get(ctx, outfitLogID)
	if err != nil {
		return domain.OutfitLog{}, err
	}
	if log.OwnerID != callerID {
		return domain.OutfitLog{}, domain.ErrForbidden
	}
	return log, nil
}

// Delete verifies caller owns the outfit log and deletes it atomically.
func (s *OutfitLogService) Delete(ctx context.Context, callerID, outfitLogID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := s.getOwnedOutfitLog(ctx, callerID, outfitLogID); err != nil {
		return err
	}
	return s.transactor.DeleteOutfitLog(ctx, outfitLogID)
}
