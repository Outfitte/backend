package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// CreateOutfitInput holds the fields required to create a new Outfit.
type CreateOutfitInput struct {
	Name  *string
	Notes *string
}

// UpdateOutfitInput holds the fields that can be updated on an existing Outfit.
// A nil outer pointer (Nullable) means the field was absent — preserve the existing value.
type UpdateOutfitInput struct {
	Name  *string              // two-state: nil = preserve, non-nil = update
	Notes domain.Nullable[string]
}

// OutfitService manages outfits, item linking, and photo management.
type OutfitService struct {
	outfits    ports.OutfitRepository
	items      ports.ItemRepository
	media      ports.MediaProvider
	outfitLogs ports.OutfitLogRepository
	shares     shareAccessChecker
}

// NewOutfitService constructs an OutfitService backed by the given repositories and media provider.
func NewOutfitService(outfits ports.OutfitRepository, items ports.ItemRepository, media ports.MediaProvider, outfitLogs ports.OutfitLogRepository, shares shareAccessChecker) *OutfitService {
	return &OutfitService{outfits: outfits, items: items, media: media, outfitLogs: outfitLogs, shares: shares}
}

func (s *OutfitService) Create(ctx context.Context, callerID string, input CreateOutfitInput) (domain.Outfit, error) {
	if err := ctx.Err(); err != nil {
		return domain.Outfit{}, err
	}
	var outfit domain.Outfit
	outfit.ID = uuid.NewString()
	outfit.OwnerID = callerID
	outfit.Name = input.Name
	outfit.Notes = input.Notes
	outfit.CreatedAt = time.Now().UTC()
	if err := s.outfits.Save(ctx, outfit); err != nil {
		return domain.Outfit{}, err
	}
	return outfit, nil
}

func (s *OutfitService) GetByID(ctx context.Context, callerID, outfitID string) (domain.Outfit, error) {
	if err := ctx.Err(); err != nil {
		return domain.Outfit{}, err
	}
	outfit, err := s.outfits.Get(ctx, outfitID)
	if err != nil {
		return domain.Outfit{}, err
	}
	if outfit.OwnerID == callerID {
		return outfit, nil
	}
	return s.getSharedOutfit(ctx, callerID, outfit)
}

func (s *OutfitService) getSharedOutfit(ctx context.Context, callerID string, outfit domain.Outfit) (domain.Outfit, error) {
	ok, err := s.shares.HasReadAccess(ctx, callerID, domain.ShareTargetOutfit, outfit.ID)
	if err != nil {
		return domain.Outfit{}, err
	}
	if !ok {
		return domain.Outfit{}, domain.ErrForbidden
	}
	return outfit, nil
}

func (s *OutfitService) ListByOwner(ctx context.Context, callerID string) ([]domain.Outfit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.outfits.ListByOwner(ctx, callerID)
}

func (s *OutfitService) Update(ctx context.Context, callerID, outfitID string, input UpdateOutfitInput) (domain.Outfit, error) {
	if err := ctx.Err(); err != nil {
		return domain.Outfit{}, err
	}
	outfit, err := s.getOwnedOutfit(ctx, callerID, outfitID)
	if err != nil {
		return domain.Outfit{}, err
	}
	if input.Name != nil {
		outfit.Name = input.Name
	}
	if input.Notes != nil {
		outfit.Notes = *input.Notes
	}
	if err := s.outfits.Save(ctx, outfit); err != nil {
		return domain.Outfit{}, err
	}
	return outfit, nil
}

func (s *OutfitService) Delete(ctx context.Context, callerID, outfitID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	outfit, err := s.getOwnedOutfit(ctx, callerID, outfitID)
	if err != nil {
		return err
	}
	if err := s.deleteOutfitPhotos(ctx, outfit); err != nil {
		return err
	}
	if err := s.outfits.Delete(ctx, outfitID); err != nil {
		return err
	}
	s.cleanUpShares(ctx, domain.ShareTargetOutfit, outfitID)
	return nil
}

func (s *OutfitService) cleanUpShares(ctx context.Context, targetType domain.ShareTargetType, targetID string) {
	if err := s.shares.DeleteByTarget(ctx, targetType, targetID); err != nil {
		slog.Default().ErrorContext(ctx, "failed to clean up shares after delete", "error", err, "target_type", targetType, "target_id", targetID)
	}
}

func (s *OutfitService) AddItem(ctx context.Context, callerID, outfitID, itemID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := s.getOwnedOutfit(ctx, callerID, outfitID); err != nil {
		return err
	}
	if err := s.validateItemOwnership(ctx, callerID, itemID); err != nil {
		return err
	}
	position, err := s.nextItemPosition(ctx, outfitID)
	if err != nil {
		return err
	}
	return s.outfits.SaveItem(ctx, outfitID, itemID, position)
}

func (s *OutfitService) RemoveItem(ctx context.Context, callerID, outfitID, itemID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := s.getOwnedOutfit(ctx, callerID, outfitID); err != nil {
		return err
	}
	return s.outfits.DeleteItem(ctx, outfitID, itemID)
}

func (s *OutfitService) UploadPhoto(ctx context.Context, callerID, outfitID string, r io.Reader, filename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	outfit, err := s.getOwnedOutfit(ctx, callerID, outfitID)
	if err != nil {
		return err
	}
	key := "outfits/" + outfitID + "/" + uuid.NewString() + "/" + filename
	if err := s.media.Upload(ctx, key, r); err != nil {
		return err
	}
	photoID := uuid.NewString()
	position := len(outfit.Photos)
	return s.outfits.SavePhoto(ctx, outfitID, photoID, key, position)
}

func (s *OutfitService) DeletePhoto(ctx context.Context, callerID, outfitID, mediaKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	outfit, err := s.getOwnedOutfit(ctx, callerID, outfitID)
	if err != nil {
		return err
	}
	if !s.outfitHasPhoto(outfit, mediaKey) {
		return domain.ErrNotFound
	}
	if err := s.media.Delete(ctx, mediaKey); err != nil {
		return err
	}
	return s.outfits.DeletePhoto(ctx, outfitID, mediaKey)
}

func (s *OutfitService) getOwnedOutfit(ctx context.Context, callerID, outfitID string) (domain.Outfit, error) {
	outfit, err := s.outfits.Get(ctx, outfitID)
	if err != nil {
		return domain.Outfit{}, err
	}
	if outfit.OwnerID != callerID {
		return domain.Outfit{}, domain.ErrForbidden
	}
	return outfit, nil
}

func (s *OutfitService) deleteOutfitPhotos(ctx context.Context, outfit domain.Outfit) error {
	for _, photo := range outfit.Photos {
		if err := s.media.Delete(ctx, photo.MediaKey); err != nil {
			return err
		}
	}
	return nil
}

func (s *OutfitService) validateItemOwnership(ctx context.Context, callerID, itemID string) error {
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return err
	}
	if item.OwnerID != callerID {
		return domain.ErrForbidden
	}
	return nil
}

func (s *OutfitService) nextItemPosition(ctx context.Context, outfitID string) (int, error) {
	ids, err := s.outfits.ListItemIDs(ctx, outfitID)
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

func (s *OutfitService) outfitHasPhoto(outfit domain.Outfit, mediaKey string) bool {
	for _, p := range outfit.Photos {
		if p.MediaKey == mediaKey {
			return true
		}
	}
	return false
}

// ListByDateRange returns outfits owned by callerID that have at least one outfit log
// in [from, to]. Returns domain.ErrValidation if from is after to.
func (s *OutfitService) ListByDateRange(ctx context.Context, callerID string, from, to time.Time) ([]domain.Outfit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if from.After(to) {
		return nil, domain.ErrValidation
	}
	logs, err := s.outfitLogs.ListByOwnerDateRange(ctx, callerID, from, to)
	if err != nil {
		return nil, err
	}
	return s.outfitsForLogs(ctx, logs)
}

// outfitsForLogs fetches the unique outfits referenced by logs.
// TODO: replace per-outfit Get calls with a batch fetch once OutfitRepository
// exposes a ListByIDs method.
func (s *OutfitService) outfitsForLogs(ctx context.Context, logs []domain.OutfitLog) ([]domain.Outfit, error) {
	seen := make(map[string]struct{}, len(logs))
	outfits := make([]domain.Outfit, 0, len(logs))
	for _, l := range logs {
		if _, ok := seen[l.OutfitID]; ok {
			continue
		}
		seen[l.OutfitID] = struct{}{}
		o, err := s.outfits.Get(ctx, l.OutfitID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return nil, err
		}
		outfits = append(outfits, o)
	}
	return outfits, nil
}
