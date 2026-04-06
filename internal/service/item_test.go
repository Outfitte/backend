package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
	"github.com/stretchr/testify/require"
)

// mockItemRepo is an in-memory ports.ItemRepository for tests.
type mockItemRepo struct {
	items          []domain.Item
	getErr         error
	saveErr        error
	deleteErr      error
	listByOwnerErr error
	savePhotoErr   error
	deletePhotoErr error
	photoKeys      []string
	listPhotoErr   error

	countByLocationErr    error
	countByLocationResult int

	// tracking
	savedPhotoItemID   string
	savedPhotoPhotoID  string
	savedPhotoKey      string
	savedPhotoPosition int
	deletedPhotoItemID string
	deletedPhotoKey    string
}

func (m *mockItemRepo) Get(_ context.Context, id string) (domain.Item, error) {
	if m.getErr != nil {
		return domain.Item{}, m.getErr
	}
	for _, item := range m.items {
		if item.GetID() == id {
			return item, nil
		}
	}
	return domain.Item{}, domain.ErrNotFound
}

func (m *mockItemRepo) Save(_ context.Context, item domain.Item) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	for i, existing := range m.items {
		if existing.GetID() == item.GetID() {
			m.items[i] = item
			return nil
		}
	}
	m.items = append(m.items, item)
	return nil
}

func (m *mockItemRepo) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, item := range m.items {
		if item.GetID() == id {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *mockItemRepo) ListByOwner(_ context.Context, ownerID string, _ ports.ItemListFilter) ([]domain.Item, error) {
	if m.listByOwnerErr != nil {
		return nil, m.listByOwnerErr
	}
	result := make([]domain.Item, 0)
	for _, item := range m.items {
		if item.OwnerID == ownerID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (m *mockItemRepo) CountByLocation(_ context.Context, _ string) (int, error) {
	if m.countByLocationErr != nil {
		return 0, m.countByLocationErr
	}
	return m.countByLocationResult, nil
}

func (m *mockItemRepo) SavePhoto(_ context.Context, itemID, photoID, mediaKey string, position int) error {
	if m.savePhotoErr != nil {
		return m.savePhotoErr
	}
	m.savedPhotoItemID = itemID
	m.savedPhotoPhotoID = photoID
	m.savedPhotoKey = mediaKey
	m.savedPhotoPosition = position
	return nil
}

func (m *mockItemRepo) DeletePhoto(_ context.Context, itemID, mediaKey string) error {
	if m.deletePhotoErr != nil {
		return m.deletePhotoErr
	}
	m.deletedPhotoItemID = itemID
	m.deletedPhotoKey = mediaKey
	return nil
}

func (m *mockItemRepo) ListPhotoKeys(_ context.Context, _ string) ([]string, error) {
	if m.listPhotoErr != nil {
		return nil, m.listPhotoErr
	}
	return m.photoKeys, nil
}

// mockLocationRepo is an in-memory ports.LocationRepository for tests.
type mockLocationRepo struct {
	locations []domain.Location
	getErr    error
	getErrFor map[string]error
	saveErr   error
	deleteErr error

	listByOwnerErr  error
	hasChildrenErr  error
	hasChildrenResult bool
}

func (m *mockLocationRepo) Get(_ context.Context, id string) (domain.Location, error) {
	if err, ok := m.getErrFor[id]; ok {
		return domain.Location{}, err
	}
	if m.getErr != nil {
		return domain.Location{}, m.getErr
	}
	for _, loc := range m.locations {
		if loc.GetID() == id {
			return loc, nil
		}
	}
	return domain.Location{}, domain.ErrNotFound
}

func (m *mockLocationRepo) Save(_ context.Context, loc domain.Location) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	for i, existing := range m.locations {
		if existing.GetID() == loc.GetID() {
			m.locations[i] = loc
			return nil
		}
	}
	m.locations = append(m.locations, loc)
	return nil
}

func (m *mockLocationRepo) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, loc := range m.locations {
		if loc.GetID() == id {
			m.locations = append(m.locations[:i], m.locations[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *mockLocationRepo) ListByOwner(_ context.Context, ownerID string) ([]domain.Location, error) {
	if m.listByOwnerErr != nil {
		return nil, m.listByOwnerErr
	}
	var result []domain.Location
	for _, loc := range m.locations {
		if loc.OwnerID == ownerID {
			result = append(result, loc)
		}
	}
	return result, nil
}

func (m *mockLocationRepo) HasChildren(_ context.Context, _ string) (bool, error) {
	if m.hasChildrenErr != nil {
		return false, m.hasChildrenErr
	}
	return m.hasChildrenResult, nil
}

// mockMediaProvider is an in-memory MediaProvider for tests.
type mockMediaProvider struct {
	uploadErr   error
	uploadedKey string
	deleteErr   error
	deletedKeys []string
}

func (m *mockMediaProvider) Upload(_ context.Context, key string, _ io.Reader) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
	m.uploadedKey = key
	return nil
}

func (m *mockMediaProvider) Delete(_ context.Context, key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedKeys = append(m.deletedKeys, key)
	return nil
}

func (m *mockMediaProvider) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockMediaProvider) GetURL(_ context.Context, _ string) (string, error) {
	return "", nil
}

// mockShareAccessChecker is a test double for the shareAccessChecker interface.
type mockShareAccessChecker struct {
	hasAccess bool
	err       error

	deleteByTargetErr   error
	deleteByTargetCalls int
	deletedTargetType   domain.ShareTargetType
	deletedTargetID     string
}

func (m *mockShareAccessChecker) HasReadAccess(_ context.Context, _ string, _ domain.ShareTargetType, _ string) (bool, error) {
	return m.hasAccess, m.err
}

func (m *mockShareAccessChecker) DeleteByTarget(_ context.Context, targetType domain.ShareTargetType, targetID string) error {
	m.deleteByTargetCalls++
	m.deletedTargetType = targetType
	m.deletedTargetID = targetID
	return m.deleteByTargetErr
}

// ── NewItemService ────────────────────────────────────────────────────────────

func TestNewItemServiceShouldUseProvidedLoggerWhenGiven(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{}, log)
	items, err := svc.ListByOwner(t.Context(), "owner-1", ports.ItemListFilter{})
	require.NoError(t, err)
	require.Empty(t, items)
}

// ── AssignLocation ────────────────────────────────────────────────────────────

func TestItemServiceAssignLocationShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.AssignLocation(ctx, "owner-1", "item-1", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceAssignLocationShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", nil)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceAssignLocationShouldReturnErrorWhenItemStoreGetFails(t *testing.T) {
	repo := &mockItemRepo{getErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceAssignLocationShouldReturnErrForbiddenWhenCallerIsNotItemOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.AssignLocation(t.Context(), "owner-2", "item-1", nil)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceAssignLocationShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	locID := "loc-1"
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(itemRepo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", &locID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceAssignLocationShouldReturnErrForbiddenWhenLocationBelongsToDifferentOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"

	locID := "loc-1"
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	locRepo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewItemService(itemRepo, &mockMediaProvider{}, locRepo, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", &locID)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceAssignLocationShouldReturnErrorWhenItemStoreSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceAssignLocationShouldAssignLocationIDWhenLocationBelongsToCaller(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	locID := "loc-1"
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	locRepo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewItemService(itemRepo, &mockMediaProvider{}, locRepo, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", &locID)
	require.NoError(t, err)

	saved, err := itemRepo.Get(t.Context(), "item-1")
	require.NoError(t, err)
	require.NotNil(t, saved.LocationID)
	require.Equal(t, "loc-1", *saved.LocationID)
}

func TestItemServiceAssignLocationShouldClearLocationIDWhenLocationIDIsNil(t *testing.T) {
	existingLocID := "old-loc-1"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.LocationID = &existingLocID

	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(itemRepo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", nil)
	require.NoError(t, err)

	saved, err := itemRepo.Get(t.Context(), "item-1")
	require.NoError(t, err)
	require.Nil(t, saved.LocationID)
}

func TestItemServiceCreateShouldReturnErrValidationWhenNameIsEmpty(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: ""})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceCreateShouldReturnErrNotFoundWhenCategoryIDIsUnknown(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	catID := "unknown-cat-id"
	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket", CategoryID: &catID})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceCreateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Create(ctx, "owner-1", CreateItemInput{Name: "Jacket"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceCreateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	repo := &mockItemRepo{saveErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket"})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceCreateShouldReturnErrValidationWhenMetadataKeyIsInvalid(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{
		Name:     "Jacket",
		Metadata: domain.ItemMetadata{Fields: map[string]string{"bad!key": "value"}},
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceCreateShouldReturnErrValidationWhenMetadataExceedsMaxFields(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	fields := make(map[string]string, 51)
	for i := range 51 {
		fields["field"+string(rune('a'+i%26))+string(rune('0'+i/26))] = "v"
	}
	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{
		Name:     "Jacket",
		Metadata: domain.ItemMetadata{Fields: fields},
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceCreateShouldReturnErrValidationWhenPurchasePriceSetWithoutCurrency(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	price := "10.00"
	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket", PurchasePrice: &price})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceCreateShouldReturnErrValidationWhenPurchaseCurrencySetWithoutPrice(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	currency := "USD"
	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket", PurchaseCurrency: &currency})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceCreateShouldReturnErrValidationWhenPurchasePriceIsInvalid(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	price := "-5.00"
	currency := "USD"
	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket", PurchasePrice: &price, PurchaseCurrency: &currency})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceCreateShouldReturnErrValidationWhenPurchaseCurrencyIsInvalid(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	price := "10.00"
	currency := "US"
	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket", PurchasePrice: &price, PurchaseCurrency: &currency})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceCreateShouldReturnErrFutureDateNotAllowedWhenPurchaseDateIsInFuture(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	future := time.Now().Add(24 * time.Hour)
	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket", PurchaseDate: &future})
	require.ErrorIs(t, err, domain.ErrFutureDateNotAllowed)
}

func TestItemServiceCreateShouldNormalizePurchaseCurrencyToUppercase(t *testing.T) {
	repo := &mockItemRepo{}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	price := "10.00"
	currency := "usd"
	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket", PurchasePrice: &price, PurchaseCurrency: &currency})
	require.NoError(t, err)
	require.Equal(t, "USD", *repo.items[0].PurchaseCurrency)
}

func TestItemServiceCreateShouldSetSellerURL(t *testing.T) {
	repo := &mockItemRepo{}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	url := "https://example.com/item"
	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket", SellerURL: &url})
	require.NoError(t, err)
	require.Equal(t, &url, repo.items[0].SellerURL)
}

func TestItemServiceCreateShouldCreateItemWithCallerAsOwner(t *testing.T) {
	repo := &mockItemRepo{}
	catSvc := NewCategoryService()
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, catSvc, &mockShareAccessChecker{})

	cats, err := catSvc.ListAll(t.Context())
	require.NoError(t, err)

	brand := "Patagonia"
	catID := cats[0].ID
	color := "Black"
	input := CreateItemInput{
		Name:       "Jacket",
		Brand:      &brand,
		CategoryID: &catID,
		Color:      &color,
		Metadata:   domain.ItemMetadata{Fields: map[string]string{"size": "M"}},
		PhotoKeys:  []string{"photo-1.jpg"},
	}

	item, err := svc.Create(t.Context(), "owner-1", input)
	require.NoError(t, err)
	require.NotEmpty(t, item.GetID())
	require.Equal(t, "owner-1", item.OwnerID)
	require.Equal(t, "Jacket", item.Name)
	require.Equal(t, &brand, item.Brand)
	require.Equal(t, &catID, item.CategoryID)
	require.Equal(t, &color, item.Color)
	require.Equal(t, "M", item.Metadata.Fields["size"])
	require.Len(t, item.Photos, 1)
	require.Equal(t, "photo-1.jpg", item.Photos[0].MediaKey)
	require.False(t, item.CreatedAt.IsZero())
	require.Len(t, repo.items, 1)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestItemServiceGetByIDShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.GetByID(ctx, "owner-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceGetByIDShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	_, err := svc.GetByID(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceGetByIDShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	repo := &mockItemRepo{getErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	_, err := svc.GetByID(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceGetByIDShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	_, err := svc.GetByID(t.Context(), "owner-2", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceGetByIDShouldReturnItemWhenCallerIsOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	got, err := svc.GetByID(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Equal(t, item, got)
}

func TestItemServiceGetByIDShouldReturnErrorWhenShareCheckerFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	shareChecker := &mockShareAccessChecker{err: domain.ErrIO}
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), shareChecker)

	_, err := svc.GetByID(t.Context(), "caller-2", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceGetByIDShouldReturnErrForbiddenWhenCallerHasNoShare(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	shareChecker := &mockShareAccessChecker{hasAccess: false}
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), shareChecker)

	_, err := svc.GetByID(t.Context(), "caller-2", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

// TestItemServiceGetByIDShouldReturnItemWhenShareCheckerGrantsAccess covers the
// direct-item-share, location-share, and ancestor-location-share paths as a group:
// all three resolve to HasReadAccess returning true, which is the contract
// ItemService depends on. The per-path logic is tested in share_test.go.
func TestItemServiceGetByIDShouldReturnItemWhenShareCheckerGrantsAccess(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	shareChecker := &mockShareAccessChecker{hasAccess: true}
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), shareChecker)

	got, err := svc.GetByID(t.Context(), "caller-2", "item-1")
	require.NoError(t, err)
	require.Equal(t, item, got)
}

// ── ListByOwner ───────────────────────────────────────────────────────────────

func TestItemServiceListByOwnerShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListByOwner(ctx, "owner-1", ports.ItemListFilter{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceListByOwnerShouldReturnErrorWhenListFails(t *testing.T) {
	repo := &mockItemRepo{listByOwnerErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	_, err := svc.ListByOwner(t.Context(), "owner-1", ports.ItemListFilter{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceListByOwnerShouldReturnEmptySliceWhenCallerHasNoItems(t *testing.T) {
	var other domain.Item
	other.ID = "item-1"
	other.OwnerID = "owner-2"

	repo := &mockItemRepo{items: []domain.Item{other}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	got, err := svc.ListByOwner(t.Context(), "owner-1", ports.ItemListFilter{})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
}

func TestItemServiceListByOwnerShouldReturnOnlyCallerItems(t *testing.T) {
	var item1 domain.Item
	item1.ID = "item-1"
	item1.OwnerID = "owner-1"

	var item2 domain.Item
	item2.ID = "item-2"
	item2.OwnerID = "owner-2"

	var item3 domain.Item
	item3.ID = "item-3"
	item3.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item1, item2, item3}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	got, err := svc.ListByOwner(t.Context(), "owner-1", ports.ItemListFilter{})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, []domain.Item{item1, item3}, got)
}

// nullableVal returns a Nullable[T] with the given value (three-state: update).
func nullableVal[T any](v T) domain.Nullable[T] {
	p := &v
	return &p
}

// nullableNil returns a Nullable[T] pointing to nil (three-state: clear).
func nullableNil[T any]() domain.Nullable[T] {
	var p *T
	return &p
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestItemServiceUpdateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Update(ctx, "owner-1", "item-1", UpdateItemInput{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceUpdateShouldReturnErrValidationWhenNameIsEmpty(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	emptyName := ""
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &emptyName})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceUpdateShouldReturnErrNotFoundWhenCategoryIDIsUnknown(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	catID := "unknown-cat-id"
	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name, CategoryID: nullableVal(catID)})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceUpdateShouldReturnErrValidationWhenMetadataKeyIsInvalid(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	meta := domain.ItemMetadata{Fields: map[string]string{"bad!key": "value"}}
	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:     &name,
		Metadata: &meta,
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceUpdateShouldReturnErrValidationWhenMetadataExceedsMaxFields(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	fields := make(map[string]string, 51)
	for i := range 51 {
		fields["field"+string(rune('a'+i%26))+string(rune('0'+i/26))] = "v"
	}
	meta := domain.ItemMetadata{Fields: fields}
	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:     &name,
		Metadata: &meta,
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceUpdateShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceUpdateShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	repo := &mockItemRepo{getErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUpdateShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-2", "item-1", UpdateItemInput{Name: &name})
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceUpdateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Updated"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUpdateShouldUpdateItemWhenCallerIsOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Old Name"

	catSvc := NewCategoryService()
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, catSvc, &mockShareAccessChecker{})

	cats, err := catSvc.ListAll(t.Context())
	require.NoError(t, err)

	name := "New Name"
	brand := "New Brand"
	catID := cats[0].ID
	color := "Blue"
	meta := domain.ItemMetadata{Fields: map[string]string{"size": "L"}}
	input := UpdateItemInput{
		Name:       &name,
		Brand:      nullableVal(brand),
		CategoryID: nullableVal(catID),
		Color:      nullableVal(color),
		Metadata:   &meta,
	}

	got, err := svc.Update(t.Context(), "owner-1", "item-1", input)
	require.NoError(t, err)
	require.Equal(t, "item-1", got.GetID())
	require.Equal(t, "owner-1", got.OwnerID)
	require.Equal(t, "New Name", got.Name)
	require.Equal(t, &brand, got.Brand)
	require.Equal(t, &catID, got.CategoryID)
	require.Equal(t, &color, got.Color)
	require.Equal(t, "L", got.Metadata.Fields["size"])
}

func TestItemServiceUpdateShouldPreserveBrandWhenAbsentFromInput(t *testing.T) {
	existingBrand := "Nike"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.Brand = &existingBrand

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name})
	require.NoError(t, err)
	require.NotNil(t, got.Brand)
	require.Equal(t, "Nike", *got.Brand)
}

func TestItemServiceUpdateShouldClearBrandWhenNullableNilInInput(t *testing.T) {
	existingBrand := "Nike"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.Brand = &existingBrand

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name, Brand: nullableNil[string]()})
	require.NoError(t, err)
	require.Nil(t, got.Brand)
}

func TestItemServiceUpdateShouldPreserveMetadataWhenAbsentFromInput(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.Metadata = domain.ItemMetadata{Fields: map[string]string{"color": "red", "size": "M"}}

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name})
	require.NoError(t, err)
	require.Equal(t, "red", got.Metadata.Fields["color"])
	require.Equal(t, "M", got.Metadata.Fields["size"])
}

func TestItemServiceUpdateShouldPreserveExistingMetadataKeysNotInPatch(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.Metadata = domain.ItemMetadata{Fields: map[string]string{"color": "red", "size": "M"}}

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	meta := domain.ItemMetadata{Fields: map[string]string{"size": "L"}}
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:     &name,
		Metadata: &meta,
	})
	require.NoError(t, err)
	require.Equal(t, "L", got.Metadata.Fields["size"])
	require.Equal(t, "red", got.Metadata.Fields["color"])
}

func TestItemServiceUpdateShouldDeleteMetadataKeyWhenValueIsEmptyString(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.Metadata = domain.ItemMetadata{Fields: map[string]string{"color": "red", "size": "M"}}

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	meta := domain.ItemMetadata{Fields: map[string]string{"color": ""}}
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:     &name,
		Metadata: &meta,
	})
	require.NoError(t, err)
	require.NotContains(t, got.Metadata.Fields, "color")
	require.Equal(t, "M", got.Metadata.Fields["size"])
}

func TestItemServiceUpdateShouldReturnErrValidationWhenMergedMetadataExceeds50Fields(t *testing.T) {
	existingFields := make(map[string]string, 50)
	for i := range 50 {
		existingFields["field"+string(rune('a'+i%26))+string(rune('0'+i/26))] = "v"
	}
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.Metadata = domain.ItemMetadata{Fields: existingFields}

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	// Adding a new key would push the merged count to 51.
	name := "Jacket"
	meta := domain.ItemMetadata{Fields: map[string]string{"brand new key": "v"}}
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:     &name,
		Metadata: &meta,
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceUpdateShouldReturnErrValidationWhenPatchPriceSetButNoCurrencyAndExistingHasNone(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name, PurchasePrice: nullableVal("10.00")})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceUpdateShouldPreservePurchaseFieldsWhenAbsentFromInput(t *testing.T) {
	existingPrice := "10.00"
	existingCurrency := "USD"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.PurchasePrice = &existingPrice
	item.PurchaseCurrency = &existingCurrency

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name})
	require.NoError(t, err)
	require.NotNil(t, got.PurchasePrice)
	require.Equal(t, "10.00", *got.PurchasePrice)
	require.NotNil(t, got.PurchaseCurrency)
	require.Equal(t, "USD", *got.PurchaseCurrency)
}

func TestItemServiceUpdateShouldClearPurchaseFieldsWhenNullableNilInInput(t *testing.T) {
	existingPrice := "10.00"
	existingCurrency := "USD"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.PurchasePrice = &existingPrice
	item.PurchaseCurrency = &existingCurrency

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:             &name,
		PurchasePrice:    nullableNil[string](),
		PurchaseCurrency: nullableNil[string](),
	})
	require.NoError(t, err)
	require.Nil(t, got.PurchasePrice)
	require.Nil(t, got.PurchaseCurrency)
}

func TestItemServiceUpdateShouldReturnErrValidationWhenPatchPriceIsInvalid(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:             &name,
		PurchasePrice:    nullableVal("-5.00"),
		PurchaseCurrency: nullableVal("USD"),
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceUpdateShouldReturnErrValidationWhenPatchCurrencyIsInvalid(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:             &name,
		PurchasePrice:    nullableVal("10.00"),
		PurchaseCurrency: nullableVal("US"),
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceUpdateShouldReturnErrFutureDateNotAllowedWhenPatchDateIsInFuture(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	future := time.Now().Add(24 * time.Hour)
	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name, PurchaseDate: nullableVal(future)})
	require.ErrorIs(t, err, domain.ErrFutureDateNotAllowed)
}

func TestItemServiceUpdateShouldSucceedWhenSettingCurrencyOnItemWithExistingPrice(t *testing.T) {
	existingPrice := "10.00"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.PurchasePrice = &existingPrice

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	// Setting currency while price is preserved from existing — results in both set, which is valid.
	name := "Jacket"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name, PurchaseCurrency: nullableVal("EUR")})
	require.NoError(t, err)
	require.NotNil(t, got.PurchasePrice)
	require.Equal(t, "10.00", *got.PurchasePrice)
	require.NotNil(t, got.PurchaseCurrency)
	require.Equal(t, "EUR", *got.PurchaseCurrency)
}

func TestItemServiceUpdateShouldReturnErrValidationWhenClearingCurrencyPreservesPrice(t *testing.T) {
	existingPrice := "10.00"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.PurchasePrice = &existingPrice

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	// Clearing currency but price is preserved → resulting state has price without currency → invalid.
	name := "Jacket"
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name, PurchaseCurrency: nullableNil[string]()})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceUpdateShouldNormalizePurchaseCurrencyToUppercase(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:             &name,
		PurchasePrice:    nullableVal("10.00"),
		PurchaseCurrency: nullableVal("eur"),
	})
	require.NoError(t, err)
	require.Equal(t, "EUR", *got.PurchaseCurrency)
}

func TestItemServiceUpdateShouldSetSellerURL(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	url := "https://example.com/item"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name, SellerURL: nullableVal(url)})
	require.NoError(t, err)
	require.Equal(t, &url, got.SellerURL)
}

func TestItemServiceUpdateShouldPreserveSellerURLWhenAbsentFromInput(t *testing.T) {
	existingURL := "https://example.com/item"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.SellerURL = &existingURL

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name})
	require.NoError(t, err)
	require.NotNil(t, got.SellerURL)
	require.Equal(t, existingURL, *got.SellerURL)
}

func TestItemServiceUpdateShouldSetLocationIDWhenNullableValInInput(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	repo := &mockItemRepo{items: []domain.Item{item}}
	locRepo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewItemService(repo, &mockMediaProvider{}, locRepo, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name, LocationID: nullableVal("loc-1")})
	require.NoError(t, err)
	require.NotNil(t, got.LocationID)
	require.Equal(t, "loc-1", *got.LocationID)
}

func TestItemServiceUpdateShouldClearSellerURLWhenNullableNilInInput(t *testing.T) {
	existingURL := "https://example.com/item"
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"
	item.SellerURL = &existingURL

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	name := "Jacket"
	got, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: &name, SellerURL: nullableNil[string]()})
	require.NoError(t, err)
	require.Nil(t, got.SellerURL)
}

// ── UploadPhoto ───────────────────────────────────────────────────────────────

func TestItemServiceUploadPhotoShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.UploadPhoto(ctx, "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceUploadPhotoShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceUploadPhotoShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	repo := &mockItemRepo{getErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUploadPhotoShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.UploadPhoto(t.Context(), "owner-2", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceUploadPhotoShouldReturnErrorWhenUploadFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}}
	media := &mockMediaProvider{uploadErr: domain.ErrIO}
	svc := NewItemService(repo, media, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUploadPhotoShouldReturnErrorWhenSavePhotoFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}, savePhotoErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUploadPhotoShouldCallSavePhotoWithCorrectArgsWhenSuccessful(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{{ID: "photo-existing", MediaKey: "existing.jpg", Position: 0}}

	repo := &mockItemRepo{items: []domain.Item{item}}
	media := &mockMediaProvider{}
	svc := NewItemService(repo, media, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "new.jpg")
	require.NoError(t, err)

	// Key must follow the pattern items/<itemID>/<uuid>/<filename>
	require.NotEmpty(t, media.uploadedKey)
	require.Equal(t, "items/item-1/", media.uploadedKey[:13])
	require.Equal(t, "/new.jpg", media.uploadedKey[len(media.uploadedKey)-8:])

	// SavePhoto must be called with the correct item ID, a non-empty photo ID,
	// the uploaded key, and the position after existing photos.
	require.Equal(t, "item-1", repo.savedPhotoItemID)
	require.NotEmpty(t, repo.savedPhotoPhotoID)
	require.Equal(t, media.uploadedKey, repo.savedPhotoKey)
	require.Equal(t, 1, repo.savedPhotoPosition)
}

// ── DeletePhoto ───────────────────────────────────────────────────────────────

func TestItemServiceDeletePhotoShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.DeletePhoto(ctx, "owner-1", "item-1", "photo.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceDeletePhotoShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	repo := &mockItemRepo{getErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeletePhotoShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "photo.jpg")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceDeletePhotoShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.DeletePhoto(t.Context(), "owner-2", "item-1", "photo.jpg")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceDeletePhotoShouldReturnErrNotFoundWhenPhotoKeyIsNotInItem(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{{ID: "photo-other", MediaKey: "other.jpg", Position: 0}}

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "missing.jpg")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceDeletePhotoShouldReturnErrorWhenMediaDeleteFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{{ID: "photo-p", MediaKey: "photo.jpg", Position: 0}}

	repo := &mockItemRepo{items: []domain.Item{item}}
	media := &mockMediaProvider{deleteErr: domain.ErrIO}
	svc := NewItemService(repo, media, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeletePhotoShouldReturnErrorWhenDeletePhotoFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{{ID: "photo-p", MediaKey: "photo.jpg", Position: 0}}

	repo := &mockItemRepo{items: []domain.Item{item}, deletePhotoErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeletePhotoShouldDeleteMediaAndCallDeletePhotoWhenSuccessful(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{
		{ID: "p1", MediaKey: "keep.jpg", Position: 0},
		{ID: "p2", MediaKey: "remove.jpg", Position: 1},
		{ID: "p3", MediaKey: "also-keep.jpg", Position: 2},
	}

	repo := &mockItemRepo{items: []domain.Item{item}}
	media := &mockMediaProvider{}
	svc := NewItemService(repo, media, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "remove.jpg")
	require.NoError(t, err)

	require.Equal(t, []string{"remove.jpg"}, media.deletedKeys)
	require.Equal(t, "item-1", repo.deletedPhotoItemID)
	require.Equal(t, "remove.jpg", repo.deletedPhotoKey)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestItemServiceDeleteShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Delete(ctx, "owner-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceDeleteShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceDeleteShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-2", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceDeleteShouldReturnErrorWhenListPhotoKeysFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}, listPhotoErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeleteShouldReturnErrorWhenMediaDeleteFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}, photoKeys: []string{"photo-1.jpg"}}
	media := &mockMediaProvider{deleteErr: domain.ErrIO}
	svc := NewItemService(repo, media, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeleteShouldReturnErrorWhenStoreDeleteFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}, deleteErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeleteShouldDeleteMediaKeysAndItemWhenCallerIsOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{
		items:     []domain.Item{item},
		photoKeys: []string{"photo-1.jpg", "photo-2.jpg"},
	}
	media := &mockMediaProvider{}
	svc := NewItemService(repo, media, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Equal(t, []string{"photo-1.jpg", "photo-2.jpg"}, media.deletedKeys)
	require.Empty(t, repo.items)
}

func TestItemServiceDeleteShouldNotFailWhenShareCleanupFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}}
	shares := &mockShareAccessChecker{deleteByTargetErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), shares)

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
}

func TestItemServiceDeleteShouldCleanUpSharesAfterDeletingItemWhenSuccessful(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	repo := &mockItemRepo{items: []domain.Item{item}}
	shares := &mockShareAccessChecker{}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), shares)

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Equal(t, 1, shares.deleteByTargetCalls)
	require.Equal(t, domain.ShareTargetItem, shares.deletedTargetType)
	require.Equal(t, "item-1", shares.deletedTargetID)
}

// ── makeItemPhotos ────────────────────────────────────────────────────────────

func TestMakeItemPhotosShouldReturnEmptySliceWhenNoKeysGiven(t *testing.T) {
	photos := makeItemPhotos(nil)
	require.Empty(t, photos)
}

func TestMakeItemPhotosShouldAssignSequentialPositionsWhenKeysGiven(t *testing.T) {
	photos := makeItemPhotos([]string{"a.jpg", "b.jpg", "c.jpg"})
	require.Len(t, photos, 3)
	require.Equal(t, 0, photos[0].Position)
	require.Equal(t, 1, photos[1].Position)
	require.Equal(t, 2, photos[2].Position)
}

func TestMakeItemPhotosShouldSetMediaKeyFromInputWhenKeysGiven(t *testing.T) {
	photos := makeItemPhotos([]string{"x.jpg", "y.jpg"})
	require.Equal(t, "x.jpg", photos[0].MediaKey)
	require.Equal(t, "y.jpg", photos[1].MediaKey)
}

func TestMakeItemPhotosShouldAssignNonEmptyIDAndCreatedAtWhenKeysGiven(t *testing.T) {
	photos := makeItemPhotos([]string{"z.jpg"})
	require.NotEmpty(t, photos[0].ID)
	require.False(t, photos[0].CreatedAt.IsZero())
}

func TestMakeItemPhotosShouldAssignUniqueIDsWhenMultipleKeysGiven(t *testing.T) {
	photos := makeItemPhotos([]string{"a.jpg", "b.jpg", "c.jpg"})
	ids := map[string]bool{photos[0].ID: true, photos[1].ID: true, photos[2].ID: true}
	require.Len(t, ids, 3)
}

func TestMakeItemPhotosShouldShareCreatedAtWhenMultipleKeysGiven(t *testing.T) {
	photos := makeItemPhotos([]string{"a.jpg", "b.jpg", "c.jpg"})
	require.Equal(t, photos[0].CreatedAt, photos[1].CreatedAt)
	require.Equal(t, photos[1].CreatedAt, photos[2].CreatedAt)
}

// ── Archive ───────────────────────────────────────────────────────────────────

func TestItemServiceArchiveShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Archive(ctx, "owner-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceArchiveShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Archive(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceArchiveShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Archive(t.Context(), "other-owner", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceArchiveShouldReturnErrAlreadyArchivedWhenItemIsAlreadyArchived(t *testing.T) {
	now := time.Now().UTC()
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.ArchivedAt = &now
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Archive(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrAlreadyArchived)
}

func TestItemServiceArchiveShouldReturnErrorWhenSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	repo := &mockItemRepo{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Archive(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceArchiveShouldSetArchivedAtWhenItemIsNotArchived(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Archive(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.NotNil(t, repo.items[0].ArchivedAt)
}

// ── Unarchive ─────────────────────────────────────────────────────────────────

func TestItemServiceUnarchiveShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Unarchive(ctx, "owner-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceUnarchiveShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Unarchive(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceUnarchiveShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Unarchive(t.Context(), "other-owner", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceUnarchiveShouldReturnErrNotArchivedWhenItemIsNotArchived(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Unarchive(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotArchived)
}

func TestItemServiceUnarchiveShouldReturnErrorWhenSaveFails(t *testing.T) {
	now := time.Now().UTC()
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.ArchivedAt = &now
	repo := &mockItemRepo{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Unarchive(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUnarchiveShouldClearArchivedAtAndDisposalReasonWhenArchived(t *testing.T) {
	now := time.Now().UTC()
	reason := domain.DisposalDonated
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.ArchivedAt = &now
	item.DisposalReason = &reason
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Unarchive(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Nil(t, repo.items[0].ArchivedAt)
	require.Nil(t, repo.items[0].DisposalReason)
}

// ── Dispose ───────────────────────────────────────────────────────────────────

func TestItemServiceDisposeShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Dispose(ctx, "owner-1", "item-1", domain.DisposalDonated)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceDisposeShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemRepo{}, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Dispose(t.Context(), "owner-1", "item-1", domain.DisposalDonated)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceDisposeShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Dispose(t.Context(), "other-owner", "item-1", domain.DisposalDonated)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceDisposeShouldReturnErrorWhenSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	repo := &mockItemRepo{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Dispose(t.Context(), "owner-1", "item-1", domain.DisposalSold)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDisposeShouldSetDisposalReasonAndArchivedAtWhenItemIsNotArchived(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Dispose(t.Context(), "owner-1", "item-1", domain.DisposalDonated)
	require.NoError(t, err)
	require.NotNil(t, repo.items[0].ArchivedAt)
	require.Equal(t, domain.DisposalDonated, *repo.items[0].DisposalReason)
}

func TestItemServiceDisposeShouldSetDisposalReasonWithoutChangingArchivedAtWhenAlreadyArchived(t *testing.T) {
	archivedAt := time.Now().Add(-24 * time.Hour).UTC()
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.ArchivedAt = &archivedAt
	repo := &mockItemRepo{items: []domain.Item{item}}
	svc := NewItemService(repo, &mockMediaProvider{}, &mockLocationRepo{}, NewCategoryService(), &mockShareAccessChecker{})

	err := svc.Dispose(t.Context(), "owner-1", "item-1", domain.DisposalSold)
	require.NoError(t, err)
	require.Equal(t, archivedAt, *repo.items[0].ArchivedAt)
	require.Equal(t, domain.DisposalSold, *repo.items[0].DisposalReason)
}
