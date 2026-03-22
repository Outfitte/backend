package json

import (
	"context"
	"sort"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// ItemRepository is a JSON file-backed implementation of ports.ItemRepository.
type ItemRepository struct {
	provider *Provider[domain.Item]
}

// NewItemRepository creates an ItemRepository that stores items in root/items.json.
func NewItemRepository(root string) *ItemRepository {
	return &ItemRepository{
		provider: NewProvider[domain.Item](root, "items.json"),
	}
}

func (r *ItemRepository) Get(ctx context.Context, id string) (domain.Item, error) {
	return r.provider.Get(ctx, id)
}

func (r *ItemRepository) Save(ctx context.Context, item domain.Item) error {
	return r.provider.Save(ctx, item)
}

func (r *ItemRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.provider.Delete(ctx, id)
}

// List returns all stored items. It is provided so that ItemRepository satisfies
// ports.StorageProvider[domain.Item] alongside ports.ItemRepository, allowing a
// single adapter instance to be wired to services that require either interface.
func (r *ItemRepository) List(ctx context.Context) ([]domain.Item, error) {
	return r.provider.List(ctx)
}

func (r *ItemRepository) ListByOwner(ctx context.Context, ownerID string, filter ports.ItemListFilter) ([]domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// domain.Item has no archived field until M2-012 is implemented.
	// All stored items are therefore implicitly active.
	// ItemStatusArchived returns empty; ItemStatusActive and ItemStatusAll both return all owner items.
	if filter.Status == ports.ItemStatusArchived {
		return []domain.Item{}, nil
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	result := []domain.Item{}
	for _, item := range all {
		if item.OwnerID == ownerID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *ItemRepository) CountByLocation(ctx context.Context, locationID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, item := range all {
		if item.LocationID != nil && *item.LocationID == locationID {
			count++
		}
	}
	return count, nil
}

func (r *ItemRepository) SavePhoto(ctx context.Context, itemID, photoID, mediaKey string, position int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := r.provider.Get(ctx, itemID)
	if err != nil {
		return err
	}
	item.Photos = append(item.Photos, domain.ItemPhoto{
		ID:        photoID,
		MediaKey:  mediaKey,
		Position:  position,
		CreatedAt: time.Now(),
	})
	return r.provider.Save(ctx, item)
}

func (r *ItemRepository) DeletePhoto(ctx context.Context, itemID, mediaKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := r.provider.Get(ctx, itemID)
	if err != nil {
		return err
	}
	filtered := make([]domain.ItemPhoto, 0, len(item.Photos))
	for _, p := range item.Photos {
		if p.MediaKey != mediaKey {
			filtered = append(filtered, p)
		}
	}
	item.Photos = filtered
	return r.provider.Save(ctx, item)
}

func (r *ItemRepository) ListPhotoKeys(ctx context.Context, itemID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	item, err := r.provider.Get(ctx, itemID)
	if err != nil {
		return nil, err
	}
	photos := make([]domain.ItemPhoto, len(item.Photos))
	copy(photos, item.Photos)
	sort.Slice(photos, func(i, j int) bool {
		return photos[i].Position < photos[j].Position
	})
	keys := make([]string, len(photos))
	for i, p := range photos {
		keys[i] = p.MediaKey
	}
	return keys, nil
}
