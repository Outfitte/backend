package service

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// CreateItemInput holds the fields required to create a new Item.
type CreateItemInput struct {
	Name          string
	Brand         string
	CategoryID    string
	Color         string
	Size          string
	PhotoKeys     []string
	LocationID    *string
	PurchasePrice *string
	PurchaseDate  *time.Time
}

// UpdateItemInput holds the fields that can be updated on an existing Item.
type UpdateItemInput struct {
	Name          string
	Brand         string
	CategoryID    string
	Color         string
	Size          string
	PhotoKeys     []string
	LocationID    *string
	PurchasePrice *string
	PurchaseDate  *time.Time
}

// ItemService manages wardrobe items.
type ItemService struct {
	items ports.StorageProvider[domain.Item]
	media ports.MediaProvider
}

// NewItemService constructs an ItemService backed by the given storage and media providers.
func NewItemService(items ports.StorageProvider[domain.Item], media ports.MediaProvider) *ItemService {
	return &ItemService{items: items, media: media}
}

func (s *ItemService) Create(ctx context.Context, callerID string, input CreateItemInput) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	var item domain.Item
	item.ID = uuid.NewString()
	item.OwnerID = callerID
	item.Name = input.Name
	item.Brand = input.Brand
	item.CategoryID = input.CategoryID
	item.Color = input.Color
	item.Size = input.Size
	item.PhotoKeys = input.PhotoKeys
	item.LocationID = input.LocationID
	item.PurchasePrice = input.PurchasePrice
	item.PurchaseDate = input.PurchaseDate
	item.CreatedAt = time.Now().UTC()
	if err := s.items.Save(ctx, item); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func (s *ItemService) GetByID(ctx context.Context, callerID, itemID string) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.OwnerID != callerID {
		return domain.Item{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *ItemService) ListByOwner(ctx context.Context, callerID string) ([]domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := s.items.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Item, 0)
	for _, item := range all {
		if item.OwnerID == callerID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *ItemService) Update(ctx context.Context, callerID, itemID string, input UpdateItemInput) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.OwnerID != callerID {
		return domain.Item{}, domain.ErrForbidden
	}
	item.Name = input.Name
	item.Brand = input.Brand
	item.CategoryID = input.CategoryID
	item.Color = input.Color
	item.Size = input.Size
	item.PhotoKeys = input.PhotoKeys
	item.LocationID = input.LocationID
	item.PurchasePrice = input.PurchasePrice
	item.PurchaseDate = input.PurchaseDate
	if err := s.items.Save(ctx, item); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

func (s *ItemService) UploadPhoto(ctx context.Context, callerID, itemID string, r io.Reader, filename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return err
	}
	if item.OwnerID != callerID {
		return domain.ErrForbidden
	}
	key := "items/" + itemID + "/" + uuid.NewString() + "/" + filename
	if err := s.media.Upload(ctx, key, r); err != nil {
		return err
	}
	item.PhotoKeys = append(item.PhotoKeys, key)
	return s.items.Save(ctx, item)
}

func (s *ItemService) DeletePhoto(ctx context.Context, callerID, itemID, photoKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return err
	}
	if item.OwnerID != callerID {
		return domain.ErrForbidden
	}
	found := false
	for _, k := range item.PhotoKeys {
		if k == photoKey {
			found = true
			break
		}
	}
	if !found {
		return domain.ErrNotFound
	}
	if err := s.media.Delete(ctx, photoKey); err != nil {
		return err
	}
	filtered := make([]string, 0, len(item.PhotoKeys))
	for _, k := range item.PhotoKeys {
		if k != photoKey {
			filtered = append(filtered, k)
		}
	}
	item.PhotoKeys = filtered
	return s.items.Save(ctx, item)
}

func (s *ItemService) Delete(ctx context.Context, callerID, itemID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return err
	}
	if item.OwnerID != callerID {
		return domain.ErrForbidden
	}
	for _, key := range item.PhotoKeys {
		if err := s.media.Delete(ctx, key); err != nil {
			return err
		}
	}
	return s.items.Delete(ctx, itemID)
}
