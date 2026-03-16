package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// LocationService manages wardrobe locations.
type LocationService struct {
	locations ports.StorageProvider[domain.Location]
	items     ports.StorageProvider[domain.Item]
}

// NewLocationService constructs a LocationService backed by the given storage and items providers.
func NewLocationService(locations ports.StorageProvider[domain.Location], items ports.StorageProvider[domain.Item]) *LocationService {
	return &LocationService{locations: locations, items: items}
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

// GetByID returns the location identified by locationID if it belongs to callerID.
func (s *LocationService) GetByID(ctx context.Context, callerID, locationID string) (domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return domain.Location{}, err
	}
	return s.getOwnedLocation(ctx, callerID, locationID)
}

// ListByOwner returns all locations belonging to callerID.
func (s *LocationService) ListByOwner(ctx context.Context, callerID string) ([]domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	locs, err := s.locations.List(ctx)
	if err != nil {
		return nil, err
	}
	var result []domain.Location
	for _, loc := range locs {
		if loc.OwnerID == callerID {
			result = append(result, loc)
		}
	}
	return result, nil
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
	return s.locations.Delete(ctx, locationID)
}

func (s *LocationService) checkNoChildLocations(ctx context.Context, locationID string) error {
	locs, err := s.locations.List(ctx)
	if err != nil {
		return err
	}
	for _, loc := range locs {
		if loc.ParentID != nil && *loc.ParentID == locationID {
			return domain.ErrConflict
		}
	}
	return nil
}

func (s *LocationService) checkNoAssignedItems(ctx context.Context, locationID string) error {
	items, err := s.items.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.LocationID != nil && *item.LocationID == locationID {
			return domain.ErrConflict
		}
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
