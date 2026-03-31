package json

import (
	"context"
	"sort"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

var _ ports.OutfitRepository = (*OutfitRepository)(nil)

// OutfitRepository is a JSON file-backed implementation of ports.OutfitRepository.
type OutfitRepository struct {
	provider *Provider[domain.Outfit]
}

// NewOutfitRepository creates an OutfitRepository that stores outfits in root/outfits.json.
func NewOutfitRepository(root string) *OutfitRepository {
	return &OutfitRepository{
		provider: NewProvider[domain.Outfit](root, "outfits.json"),
	}
}

func (r *OutfitRepository) Get(ctx context.Context, id string) (domain.Outfit, error) {
	return r.provider.Get(ctx, id)
}

func (r *OutfitRepository) Save(ctx context.Context, outfit domain.Outfit) error {
	return r.provider.Save(ctx, outfit)
}

func (r *OutfitRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.provider.Delete(ctx, id)
}

func (r *OutfitRepository) ListByOwner(ctx context.Context, ownerID string) ([]domain.Outfit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	result := []domain.Outfit{}
	for _, o := range all {
		if o.OwnerID == ownerID {
			result = append(result, o)
		}
	}
	return result, nil
}

func (r *OutfitRepository) SaveItem(ctx context.Context, outfitID, itemID string, position int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	outfit, err := r.provider.Get(ctx, outfitID)
	if err != nil {
		return err
	}
	outfit.Items = upsertOutfitItem(outfit.Items, domain.OutfitItem{
		OutfitID: outfitID,
		ItemID:   itemID,
		Position: position,
	})
	return r.provider.Save(ctx, outfit)
}

func (r *OutfitRepository) DeleteItem(ctx context.Context, outfitID, itemID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	outfit, err := r.provider.Get(ctx, outfitID)
	if err != nil {
		return err
	}
	filtered := make([]domain.OutfitItem, 0, len(outfit.Items))
	for _, oi := range outfit.Items {
		if oi.ItemID != itemID {
			filtered = append(filtered, oi)
		}
	}
	outfit.Items = filtered
	return r.provider.Save(ctx, outfit)
}

func (r *OutfitRepository) ListItemIDs(ctx context.Context, outfitID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	outfit, err := r.provider.Get(ctx, outfitID)
	if err != nil {
		return nil, err
	}
	return sortedItemIDs(outfit.Items), nil
}

func (r *OutfitRepository) SavePhoto(ctx context.Context, outfitID, photoID, mediaKey string, position int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	outfit, err := r.provider.Get(ctx, outfitID)
	if err != nil {
		return err
	}
	outfit.Photos = upsertOutfitPhoto(outfit.Photos, domain.OutfitPhoto{
		ID:        photoID,
		MediaKey:  mediaKey,
		Position:  position,
		CreatedAt: time.Now(),
	})
	return r.provider.Save(ctx, outfit)
}

func (r *OutfitRepository) DeletePhoto(ctx context.Context, outfitID, mediaKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	outfit, err := r.provider.Get(ctx, outfitID)
	if err != nil {
		return err
	}
	filtered := make([]domain.OutfitPhoto, 0, len(outfit.Photos))
	for _, p := range outfit.Photos {
		if p.MediaKey != mediaKey {
			filtered = append(filtered, p)
		}
	}
	outfit.Photos = filtered
	return r.provider.Save(ctx, outfit)
}

func upsertOutfitItem(items []domain.OutfitItem, item domain.OutfitItem) []domain.OutfitItem {
	for i, oi := range items {
		if oi.ItemID == item.ItemID {
			items[i] = item
			return items
		}
	}
	return append(items, item)
}

func upsertOutfitPhoto(photos []domain.OutfitPhoto, photo domain.OutfitPhoto) []domain.OutfitPhoto {
	for i, p := range photos {
		if p.MediaKey == photo.MediaKey {
			photo.CreatedAt = p.CreatedAt
			photos[i] = photo
			return photos
		}
	}
	return append(photos, photo)
}

func sortedItemIDs(items []domain.OutfitItem) []string {
	sorted := make([]domain.OutfitItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})
	ids := make([]string, len(sorted))
	for i, oi := range sorted {
		ids[i] = oi.ItemID
	}
	return ids
}
