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

func (s *LocationService) Create(ctx context.Context, callerID, label string, parentID *string) (domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return domain.Location{}, err
	}
	if parentID != nil {
		parent, err := s.locations.Get(ctx, *parentID)
		if err != nil {
			return domain.Location{}, err
		}
		if parent.OwnerID != callerID {
			return domain.Location{}, domain.ErrForbidden
		}
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
