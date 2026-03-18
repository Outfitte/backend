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
	Brand         *string
	CategoryID    *string
	Color         *string
	Metadata      domain.ItemMetadata
	PhotoKeys     []string
	LocationID    *string
	PurchasePrice *string
	PurchaseDate  *time.Time
}

// UpdateItemInput holds the fields that can be updated on an existing Item.
type UpdateItemInput struct {
	Name          string
	Brand         *string
	CategoryID    *string
	Color         *string
	Metadata      domain.ItemMetadata
	PhotoKeys     []string
	LocationID    *string
	PurchasePrice *string
	PurchaseDate  *time.Time
}

// ItemService manages wardrobe items.
type ItemService struct {
	items     ports.StorageProvider[domain.Item]
	locations ports.StorageProvider[domain.Location]
	media     ports.MediaProvider
}

// NewItemService constructs an ItemService backed by the given storage and media providers.
func NewItemService(items ports.StorageProvider[domain.Item], media ports.MediaProvider, locations ports.StorageProvider[domain.Location]) *ItemService {
	return &ItemService{items: items, locations: locations, media: media}
}

func (s *ItemService) AssignLocation(ctx context.Context, callerID, itemID string, locationID *string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := s.getOwnedItem(ctx, callerID, itemID)
	if err != nil {
		return err
	}
	if err := s.validateLocationOwnership(ctx, callerID, locationID); err != nil {
		return err
	}
	item.LocationID = locationID
	return s.items.Save(ctx, item)
}

func (s *ItemService) getOwnedItem(ctx context.Context, callerID, itemID string) (domain.Item, error) {
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.OwnerID != callerID {
		return domain.Item{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *ItemService) validateLocationOwnership(ctx context.Context, callerID string, locationID *string) error {
	if locationID == nil {
		return nil
	}
	loc, err := s.locations.Get(ctx, *locationID)
	if err != nil {
		return err
	}
	if loc.OwnerID != callerID {
		return domain.ErrForbidden
	}
	return nil
}

func (s *ItemService) Create(ctx context.Context, callerID string, input CreateItemInput) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	if err := domain.ValidateMetadata(input.Metadata); err != nil {
		return domain.Item{}, err
	}
	var item domain.Item
	item.ID = uuid.NewString()
	item.OwnerID = callerID
	item.Name = input.Name
	item.Brand = input.Brand
	item.CategoryID = input.CategoryID
	item.Color = input.Color
	item.Metadata = input.Metadata
	item.Photos = makeItemPhotos(input.PhotoKeys)
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
	if err := domain.ValidateMetadata(input.Metadata); err != nil {
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
	item.Metadata = input.Metadata
	item.Photos = makeItemPhotos(input.PhotoKeys)
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
	item.Photos = append(item.Photos, domain.ItemPhoto{
		ID:        uuid.NewString(),
		MediaKey:  key,
		Position:  len(item.Photos),
		CreatedAt: time.Now().UTC(),
	})
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
	for _, p := range item.Photos {
		if p.MediaKey == photoKey {
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
	filtered := make([]domain.ItemPhoto, 0, len(item.Photos))
	for _, p := range item.Photos {
		if p.MediaKey != photoKey {
			filtered = append(filtered, p)
		}
	}
	item.Photos = filtered
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
	for _, p := range item.Photos {
		if err := s.media.Delete(ctx, p.MediaKey); err != nil {
			return err
		}
	}
	return s.items.Delete(ctx, itemID)
}

// makeItemPhotos converts a slice of media keys into ItemPhoto structs,
// assigning sequential positions starting from 0 and a fresh UUID for each.
func makeItemPhotos(keys []string) []domain.ItemPhoto {
	photos := make([]domain.ItemPhoto, len(keys))
	now := time.Now().UTC()
	for i, key := range keys {
		photos[i] = domain.ItemPhoto{
			ID:        uuid.NewString(),
			MediaKey:  key,
			Position:  i,
			CreatedAt: now,
		}
	}
	return photos
}
