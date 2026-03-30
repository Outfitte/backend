package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

// CreateItemInput holds the fields required to create a new Item.
type CreateItemInput struct {
	Name             string
	Brand            *string
	CategoryID       *string
	Color            *string
	Metadata         domain.ItemMetadata
	PhotoKeys        []string
	LocationID       *string
	PurchasePrice    *string
	PurchaseCurrency *string
	PurchaseDate     *time.Time
	SellerURL        *string
}

// UpdateItemInput holds the fields that can be updated on an existing Item.
type UpdateItemInput struct {
	Name             string
	Brand            *string
	CategoryID       *string
	Color            *string
	Metadata         domain.ItemMetadata
	PhotoKeys        []string
	LocationID       *string
	PurchasePrice    *string
	PurchaseCurrency *string
	PurchaseDate     *time.Time
	SellerURL        *string
}

// categoryGetter is a narrow interface used by ItemService to validate that a
// category ID refers to a known category.
type categoryGetter interface {
	GetByID(ctx context.Context, id string) (domain.Category, error)
}

// ItemService manages wardrobe items.
type ItemService struct {
	items      ports.ItemRepository
	locations  ports.LocationRepository
	media      ports.MediaProvider
	categories categoryGetter
}

// NewItemService constructs an ItemService backed by the given repositories and media provider.
func NewItemService(items ports.ItemRepository, media ports.MediaProvider, locations ports.LocationRepository, categories categoryGetter) *ItemService {
	return &ItemService{items: items, locations: locations, media: media, categories: categories}
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
	if err := s.validateNameAndCategory(ctx, input.Name, input.CategoryID); err != nil {
		return domain.Item{}, err
	}
	if err := domain.ValidateMetadata(input.Metadata); err != nil {
		return domain.Item{}, err
	}
	if err := domain.ValidatePurchasePair(input.PurchasePrice, input.PurchaseCurrency); err != nil {
		return domain.Item{}, err
	}
	if input.PurchasePrice != nil {
		if err := domain.ValidatePurchasePrice(*input.PurchasePrice); err != nil {
			return domain.Item{}, err
		}
	}
	if input.PurchaseCurrency != nil {
		if err := domain.ValidatePurchaseCurrency(*input.PurchaseCurrency); err != nil {
			return domain.Item{}, err
		}
		upper := strings.ToUpper(*input.PurchaseCurrency)
		input.PurchaseCurrency = &upper
	}
	if input.PurchaseDate != nil {
		if err := domain.ValidatePurchaseDate(*input.PurchaseDate); err != nil {
			return domain.Item{}, err
		}
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
	item.PurchaseCurrency = input.PurchaseCurrency
	item.PurchaseDate = input.PurchaseDate
	item.SellerURL = input.SellerURL
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

func (s *ItemService) ListByOwner(ctx context.Context, callerID string, filter ports.ItemListFilter) ([]domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.items.ListByOwner(ctx, callerID, filter)
}

func (s *ItemService) Update(ctx context.Context, callerID, itemID string, input UpdateItemInput) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	if err := s.validateNameAndCategory(ctx, input.Name, input.CategoryID); err != nil {
		return domain.Item{}, err
	}
	for k := range input.Metadata.Fields {
		if err := domain.ValidateMetadataKey(k); err != nil {
			return domain.Item{}, err
		}
	}
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return domain.Item{}, err
	}
	if item.OwnerID != callerID {
		return domain.Item{}, domain.ErrForbidden
	}
	merged, err := s.mergeMetadata(item.Metadata, input.Metadata)
	if err != nil {
		return domain.Item{}, err
	}
	// Compute merged purchase fields: nil in patch means keep existing.
	resultPrice := coalesce(input.PurchasePrice, item.PurchasePrice)
	resultCurrency := coalesce(input.PurchaseCurrency, item.PurchaseCurrency)
	resultDate := coalesceTime(input.PurchaseDate, item.PurchaseDate)
	if err := domain.ValidatePurchasePair(resultPrice, resultCurrency); err != nil {
		return domain.Item{}, err
	}
	if input.PurchasePrice != nil {
		if err := domain.ValidatePurchasePrice(*input.PurchasePrice); err != nil {
			return domain.Item{}, err
		}
	}
	if input.PurchaseCurrency != nil {
		if err := domain.ValidatePurchaseCurrency(*input.PurchaseCurrency); err != nil {
			return domain.Item{}, err
		}
		upper := strings.ToUpper(*input.PurchaseCurrency)
		resultCurrency = &upper
	}
	if input.PurchaseDate != nil {
		if err := domain.ValidatePurchaseDate(*input.PurchaseDate); err != nil {
			return domain.Item{}, err
		}
	}
	item.Name = input.Name
	item.Brand = input.Brand
	item.CategoryID = input.CategoryID
	item.Color = input.Color
	item.Metadata = merged
	item.Photos = makeItemPhotos(input.PhotoKeys)
	item.LocationID = input.LocationID
	item.PurchasePrice = resultPrice
	item.PurchaseCurrency = resultCurrency
	item.PurchaseDate = resultDate
	item.SellerURL = input.SellerURL
	if err := s.items.Save(ctx, item); err != nil {
		return domain.Item{}, err
	}
	return item, nil
}

// coalesce returns a if non-nil, otherwise b.
func coalesce(a, b *string) *string {
	if a != nil {
		return a
	}
	return b
}

// coalesceTime returns a if non-nil, otherwise b.
func coalesceTime(a, b *time.Time) *time.Time {
	if a != nil {
		return a
	}
	return b
}

// mergeMetadata applies patch semantics: keys with empty values are deleted,
// other keys overwrite existing ones, and keys absent from the patch are preserved.
// Returns ErrValidation if the merged result exceeds the maximum field count.
func (s *ItemService) mergeMetadata(existing, patch domain.ItemMetadata) (domain.ItemMetadata, error) {
	merged := make(map[string]string, len(existing.Fields))
	for k, v := range existing.Fields {
		merged[k] = v
	}
	for k, v := range patch.Fields {
		if v == "" {
			delete(merged, k)
		} else {
			merged[k] = v
		}
	}
	if len(merged) > 50 {
		return domain.ItemMetadata{}, fmt.Errorf("%w: metadata exceeds maximum of 50 fields", domain.ErrValidation)
	}
	return domain.ItemMetadata{Fields: merged}, nil
}

func (s *ItemService) UploadPhoto(ctx context.Context, callerID, itemID string, r io.Reader, filename string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := s.getOwnedItem(ctx, callerID, itemID)
	if err != nil {
		return err
	}
	key := "items/" + itemID + "/" + uuid.NewString() + "/" + filename
	if err := s.media.Upload(ctx, key, r); err != nil {
		return err
	}
	photoID := uuid.NewString()
	position := len(item.Photos)
	return s.items.SavePhoto(ctx, itemID, photoID, key, position)
}

func (s *ItemService) DeletePhoto(ctx context.Context, callerID, itemID, photoKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := s.getOwnedItem(ctx, callerID, itemID)
	if err != nil {
		return err
	}
	if !s.itemHasPhoto(item, photoKey) {
		return domain.ErrNotFound
	}
	if err := s.media.Delete(ctx, photoKey); err != nil {
		return err
	}
	return s.items.DeletePhoto(ctx, itemID, photoKey)
}

func (s *ItemService) itemHasPhoto(item domain.Item, photoKey string) bool {
	for _, p := range item.Photos {
		if p.MediaKey == photoKey {
			return true
		}
	}
	return false
}

func (s *ItemService) Delete(ctx context.Context, callerID, itemID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := s.getOwnedItem(ctx, callerID, itemID); err != nil {
		return err
	}
	keys, err := s.items.ListPhotoKeys(ctx, itemID)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := s.media.Delete(ctx, key); err != nil {
			return err
		}
	}
	return s.items.Delete(ctx, itemID)
}

func (s *ItemService) Archive(ctx context.Context, callerID, itemID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := s.getOwnedItem(ctx, callerID, itemID)
	if err != nil {
		return err
	}
	if item.ArchivedAt != nil {
		return domain.ErrAlreadyArchived
	}
	now := time.Now().UTC()
	item.ArchivedAt = &now
	return s.items.Save(ctx, item)
}

func (s *ItemService) Unarchive(ctx context.Context, callerID, itemID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := s.getOwnedItem(ctx, callerID, itemID)
	if err != nil {
		return err
	}
	if item.ArchivedAt == nil {
		return domain.ErrNotArchived
	}
	item.ArchivedAt = nil
	item.DisposalReason = nil
	return s.items.Save(ctx, item)
}

func (s *ItemService) Dispose(ctx context.Context, callerID, itemID string, reason domain.DisposalReason) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := s.getOwnedItem(ctx, callerID, itemID)
	if err != nil {
		return err
	}
	if item.ArchivedAt == nil {
		now := time.Now().UTC()
		item.ArchivedAt = &now
	}
	item.DisposalReason = &reason
	return s.items.Save(ctx, item)
}

// validateNameAndCategory checks that name is non-empty and, if categoryID is
// provided, that the referenced category exists.
func (s *ItemService) validateNameAndCategory(ctx context.Context, name string, categoryID *string) error {
	if name == "" {
		return fmt.Errorf("%w: name must not be empty", domain.ErrValidation)
	}
	if categoryID != nil {
		if _, err := s.categories.GetByID(ctx, *categoryID); err != nil {
			return err
		}
	}
	return nil
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
