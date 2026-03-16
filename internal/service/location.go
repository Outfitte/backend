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
}

// NewLocationService constructs a LocationService backed by the given storage provider.
func NewLocationService(locations ports.StorageProvider[domain.Location]) *LocationService {
	return &LocationService{locations: locations}
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
	loc, err := s.locations.Get(ctx, locationID)
	if err != nil {
		return domain.Location{}, err
	}
	if loc.OwnerID != callerID {
		return domain.Location{}, domain.ErrForbidden
	}
	return loc, nil
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
