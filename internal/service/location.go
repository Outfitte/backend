package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// LocationService manages wardrobe locations.
type LocationService struct {
	locations ports.LocationRepository
	items     ports.ItemRepository
	shares    shareAccessChecker
}

// NewLocationService constructs a LocationService backed by the given repositories.
func NewLocationService(locations ports.LocationRepository, items ports.ItemRepository, shares shareAccessChecker) *LocationService {
	return &LocationService{locations: locations, items: items, shares: shares}
}

func (s *LocationService) getOwnedLocation(ctx context.Context, callerID, locationID string) (domain.Location, error) {
	loc, err := s.locations.Get(ctx, locationID)
	if err != nil {
		return domain.Location{}, err
	}
	if loc.OwnerID != callerID {
		return domain.Location{}, domain.ErrForbidden
	}
	return loc, nil
}

func (s *LocationService) validateParent(ctx context.Context, callerID string, parentID *string) error {
	if parentID == nil {
		return nil
	}
	parent, err := s.locations.Get(ctx, *parentID)
	if err != nil {
		return err
	}
	if parent.OwnerID != callerID {
		return domain.ErrForbidden
	}
	return nil
}

func (s *LocationService) getSharedLocation(ctx context.Context, callerID string, loc domain.Location) (domain.Location, error) {
	ok, err := s.shares.HasReadAccess(ctx, callerID, domain.ShareTargetLocation, loc.ID)
	if err != nil {
		return domain.Location{}, err
	}
	if !ok {
		return domain.Location{}, domain.ErrForbidden
	}
	return loc, nil
}

// GetByID returns the location identified by locationID if it belongs to callerID,
// or if callerID has shared read access to it.
func (s *LocationService) GetByID(ctx context.Context, callerID, locationID string) (domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return domain.Location{}, err
	}
	loc, err := s.locations.Get(ctx, locationID)
	if err != nil {
		return domain.Location{}, err
	}
	if loc.OwnerID == callerID {
		return loc, nil
	}
	return s.getSharedLocation(ctx, callerID, loc)
}

// ListByOwner returns all locations belonging to callerID.
func (s *LocationService) ListByOwner(ctx context.Context, callerID string) ([]domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.locations.ListByOwner(ctx, callerID)
}

// Update changes the label of the location identified by locationID if it belongs to callerID.
func (s *LocationService) Update(ctx context.Context, callerID, locationID, label string) (domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return domain.Location{}, err
	}
	loc, err := s.getOwnedLocation(ctx, callerID, locationID)
	if err != nil {
		return domain.Location{}, err
	}
	loc.Label = label
	if err := s.locations.Save(ctx, loc); err != nil {
		return domain.Location{}, err
	}
	return loc, nil
}

// Delete removes the location identified by locationID if it belongs to callerID
// and has no children or items assigned.
func (s *LocationService) Delete(ctx context.Context, callerID, locationID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := s.getOwnedLocation(ctx, callerID, locationID); err != nil {
		return err
	}
	if err := s.checkNoChildLocations(ctx, locationID); err != nil {
		return err
	}
	if err := s.checkNoAssignedItems(ctx, locationID); err != nil {
		return err
	}
	if err := s.locations.Delete(ctx, locationID); err != nil {
		return err
	}
	return cleanUpShares(ctx, s.shares, domain.ShareTargetLocation, locationID)
}

func (s *LocationService) checkNoChildLocations(ctx context.Context, locationID string) error {
	hasChildren, err := s.locations.HasChildren(ctx, locationID)
	if err != nil {
		return err
	}
	if hasChildren {
		return domain.ErrConflict
	}
	return nil
}

func (s *LocationService) checkNoAssignedItems(ctx context.Context, locationID string) error {
	count, err := s.items.CountByLocation(ctx, locationID)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrConflict
	}
	return nil
}

// Move reparents the location identified by locationID to newParentID (nil = root).
func (s *LocationService) Move(ctx context.Context, callerID, locationID string, newParentID *string) (domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return domain.Location{}, err
	}
	loc, err := s.getOwnedLocation(ctx, callerID, locationID)
	if err != nil {
		return domain.Location{}, err
	}
	if err := s.validateParent(ctx, callerID, newParentID); err != nil {
		return domain.Location{}, err
	}
	if newParentID != nil {
		if err := s.checkNoCircularReference(ctx, locationID, *newParentID); err != nil {
			return domain.Location{}, err
		}
	}
	loc.ParentID = newParentID
	if err := s.locations.Save(ctx, loc); err != nil {
		return domain.Location{}, err
	}
	return loc, nil
}

func (s *LocationService) checkNoCircularReference(ctx context.Context, locationID, candidateID string) error {
	current := candidateID
	for current != "" {
		if current == locationID {
			return domain.ErrConflict
		}
		l, err := s.locations.Get(ctx, current)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				break
			}
			return err
		}
		if l.ParentID == nil {
			break
		}
		current = *l.ParentID
	}
	return nil
}

func (s *LocationService) Create(ctx context.Context, callerID, label string, parentID *string) (domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return domain.Location{}, err
	}
	if err := s.validateParent(ctx, callerID, parentID); err != nil {
		return domain.Location{}, err
	}
	var loc domain.Location
	loc.ID = uuid.NewString()
	loc.OwnerID = callerID
	loc.Label = label
	loc.ParentID = parentID
	loc.CreatedAt = time.Now().UTC()
	if err := s.locations.Save(ctx, loc); err != nil {
		return domain.Location{}, err
	}
	return loc, nil
}
