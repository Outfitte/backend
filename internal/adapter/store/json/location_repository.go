package json

import (
	"context"

	"github.com/outfitte/outfitte/internal/domain"
)

// LocationRepository is a JSON file-backed implementation of ports.LocationRepository.
type LocationRepository struct {
	provider *Provider[domain.Location]
}

// NewLocationRepository creates a LocationRepository that stores locations in root/locations.json.
func NewLocationRepository(root string) *LocationRepository {
	return &LocationRepository{
		provider: NewProvider[domain.Location](root, "locations.json"),
	}
}

func (r *LocationRepository) Get(ctx context.Context, id string) (domain.Location, error) {
	return r.provider.Get(ctx, id)
}

func (r *LocationRepository) Save(ctx context.Context, location domain.Location) error {
	return r.provider.Save(ctx, location)
}

func (r *LocationRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.provider.Delete(ctx, id)
}

func (r *LocationRepository) ListByOwner(ctx context.Context, ownerID string) ([]domain.Location, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	result := []domain.Location{}
	for _, loc := range all {
		if loc.OwnerID == ownerID {
			result = append(result, loc)
		}
	}
	return result, nil
}

func (r *LocationRepository) HasChildren(ctx context.Context, locationID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if _, err := r.provider.Get(ctx, locationID); err != nil {
		return false, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return false, err
	}
	for _, loc := range all {
		if loc.ParentID != nil && *loc.ParentID == locationID {
			return true, nil
		}
	}
	return false, nil
}
