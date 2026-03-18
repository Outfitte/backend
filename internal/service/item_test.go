package service

import (
	"context"
	"io"
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// mockItemStore is an in-memory StorageProvider[domain.Item] for tests.
type mockItemStore struct {
	items     []domain.Item
	getErr    error
	listErr   error
	saveErr   error
	deleteErr error
}

func (m *mockItemStore) Get(_ context.Context, id string) (domain.Item, error) {
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

func (m *mockItemStore) List(_ context.Context) ([]domain.Item, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.items, nil
}

func (m *mockItemStore) Save(_ context.Context, item domain.Item) error {
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

func (m *mockItemStore) Delete(_ context.Context, id string) error {
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

// mockMediaProvider is an in-memory MediaProvider for tests.
type mockMediaProvider struct {
	uploadErr    error
	uploadedKey  string
	deleteErr    error
	deletedKeys  []string
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

// ── Create ────────────────────────────────────────────────────────────────────

// ── AssignLocation ────────────────────────────────────────────────────────────

func TestItemServiceAssignLocationShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.AssignLocation(ctx, "owner-1", "item-1", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceAssignLocationShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", nil)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceAssignLocationShouldReturnErrorWhenItemStoreGetFails(t *testing.T) {
	store := &mockItemStore{getErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceAssignLocationShouldReturnErrForbiddenWhenCallerIsNotItemOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.AssignLocation(t.Context(), "owner-2", "item-1", nil)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceAssignLocationShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	locID := "loc-1"
	itemStore := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(itemStore, &mockMediaProvider{}, &mockLocationStore{})

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
	itemStore := &mockItemStore{items: []domain.Item{item}}
	locStore := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewItemService(itemStore, &mockMediaProvider{}, locStore)

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", &locID)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceAssignLocationShouldReturnErrorWhenItemStoreSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

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
	itemStore := &mockItemStore{items: []domain.Item{item}}
	locStore := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewItemService(itemStore, &mockMediaProvider{}, locStore)

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", &locID)
	require.NoError(t, err)

	saved, err := itemStore.Get(t.Context(), "item-1")
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

	itemStore := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(itemStore, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.AssignLocation(t.Context(), "owner-1", "item-1", nil)
	require.NoError(t, err)

	saved, err := itemStore.Get(t.Context(), "item-1")
	require.NoError(t, err)
	require.Nil(t, saved.LocationID)
}

func TestItemServiceCreateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Create(ctx, "owner-1", CreateItemInput{Name: "Jacket"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceCreateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	store := &mockItemStore{saveErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket"})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceCreateShouldReturnErrValidationWhenMetadataKeyIsInvalid(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{
		Name:     "Jacket",
		Metadata: domain.ItemMetadata{Fields: map[string]string{"bad!key": "value"}},
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceCreateShouldReturnErrValidationWhenMetadataExceedsMaxFields(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})

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

func TestItemServiceCreateShouldCreateItemWithCallerAsOwner(t *testing.T) {
	store := &mockItemStore{}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	brand := "Patagonia"
	catID := "cat-1"
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
	require.Len(t, store.items, 1)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestItemServiceGetByIDShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.GetByID(ctx, "owner-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceGetByIDShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.GetByID(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceGetByIDShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockItemStore{getErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.GetByID(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceGetByIDShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.GetByID(t.Context(), "owner-2", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceGetByIDShouldReturnItemWhenCallerIsOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	got, err := svc.GetByID(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Equal(t, item, got)
}

// ── ListByOwner ───────────────────────────────────────────────────────────────

func TestItemServiceListByOwnerShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListByOwner(ctx, "owner-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceListByOwnerShouldReturnErrorWhenListFails(t *testing.T) {
	store := &mockItemStore{listErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.ListByOwner(t.Context(), "owner-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceListByOwnerShouldReturnEmptySliceWhenCallerHasNoItems(t *testing.T) {
	var other domain.Item
	other.ID = "item-1"
	other.OwnerID = "owner-2"

	store := &mockItemStore{items: []domain.Item{other}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	got, err := svc.ListByOwner(t.Context(), "owner-1")
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

	store := &mockItemStore{items: []domain.Item{item1, item2, item3}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	got, err := svc.ListByOwner(t.Context(), "owner-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, []domain.Item{item1, item3}, got)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestItemServiceUpdateShouldReturnErrValidationWhenMetadataKeyIsInvalid(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:     "Jacket",
		Metadata: domain.ItemMetadata{Fields: map[string]string{"bad!key": "value"}},
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceUpdateShouldReturnErrValidationWhenMetadataExceedsMaxFields(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	fields := make(map[string]string, 51)
	for i := range 51 {
		fields["field"+string(rune('a'+i%26))+string(rune('0'+i/26))] = "v"
	}
	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{
		Name:     "Jacket",
		Metadata: domain.ItemMetadata{Fields: fields},
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestItemServiceUpdateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Update(ctx, "owner-1", "item-1", UpdateItemInput{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceUpdateShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceUpdateShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockItemStore{getErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUpdateShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.Update(t.Context(), "owner-2", "item-1", UpdateItemInput{})
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceUpdateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: "Updated"})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUpdateShouldUpdateItemWhenCallerIsOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Old Name"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	brand := "New Brand"
	catID := "cat-2"
	color := "Blue"
	input := UpdateItemInput{
		Name:       "New Name",
		Brand:      &brand,
		CategoryID: &catID,
		Color:      &color,
		Metadata:   domain.ItemMetadata{Fields: map[string]string{"size": "L"}},
		PhotoKeys:  []string{"new-photo.jpg"},
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
	require.Len(t, got.Photos, 1)
	require.Equal(t, "new-photo.jpg", got.Photos[0].MediaKey)
}

// ── UploadPhoto ───────────────────────────────────────────────────────────────

func TestItemServiceUploadPhotoShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.UploadPhoto(ctx, "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceUploadPhotoShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceUploadPhotoShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockItemStore{getErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUploadPhotoShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.UploadPhoto(t.Context(), "owner-2", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceUploadPhotoShouldReturnErrorWhenUploadFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	media := &mockMediaProvider{uploadErr: domain.ErrIO}
	svc := NewItemService(store, media, &mockLocationStore{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUploadPhotoShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUploadPhotoShouldAppendKeyAndSaveItemWhenSuccessful(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{{ID: "photo-existing", MediaKey: "existing.jpg", Position: 0}}

	store := &mockItemStore{items: []domain.Item{item}}
	media := &mockMediaProvider{}
	svc := NewItemService(store, media, &mockLocationStore{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "new.jpg")
	require.NoError(t, err)

	// Key must follow the pattern items/<itemID>/<uuid>/<filename>
	require.NotEmpty(t, media.uploadedKey)
	require.Equal(t, "items/item-1/", media.uploadedKey[:13])
	require.Equal(t, "/new.jpg", media.uploadedKey[len(media.uploadedKey)-8:])

	// Item must be updated in the store
	saved, err := store.Get(t.Context(), "item-1")
	require.NoError(t, err)
	require.Len(t, saved.Photos, 2)
	require.Equal(t, "existing.jpg", saved.Photos[0].MediaKey)
	require.Equal(t, media.uploadedKey, saved.Photos[1].MediaKey)
}

// ── DeletePhoto ───────────────────────────────────────────────────────────────

func TestItemServiceDeletePhotoShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.DeletePhoto(ctx, "owner-1", "item-1", "photo.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceDeletePhotoShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockItemStore{getErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeletePhotoShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "photo.jpg")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceDeletePhotoShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.DeletePhoto(t.Context(), "owner-2", "item-1", "photo.jpg")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceDeletePhotoShouldReturnErrNotFoundWhenPhotoKeyIsNotInItem(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{{ID: "photo-other", MediaKey: "other.jpg", Position: 0}}

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "missing.jpg")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceDeletePhotoShouldReturnErrorWhenMediaDeleteFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{{ID: "photo-p", MediaKey: "photo.jpg", Position: 0}}

	store := &mockItemStore{items: []domain.Item{item}}
	media := &mockMediaProvider{deleteErr: domain.ErrIO}
	svc := NewItemService(store, media, &mockLocationStore{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeletePhotoShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{{ID: "photo-p", MediaKey: "photo.jpg", Position: 0}}

	store := &mockItemStore{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeletePhotoShouldRemoveKeyAndSaveItemWhenSuccessful(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{
		{ID: "p1", MediaKey: "keep.jpg", Position: 0},
		{ID: "p2", MediaKey: "remove.jpg", Position: 1},
		{ID: "p3", MediaKey: "also-keep.jpg", Position: 2},
	}

	store := &mockItemStore{items: []domain.Item{item}}
	media := &mockMediaProvider{}
	svc := NewItemService(store, media, &mockLocationStore{})

	err := svc.DeletePhoto(t.Context(), "owner-1", "item-1", "remove.jpg")
	require.NoError(t, err)

	require.Equal(t, []string{"remove.jpg"}, media.deletedKeys)

	saved, err := store.Get(t.Context(), "item-1")
	require.NoError(t, err)
	require.Len(t, saved.Photos, 2)
	require.Equal(t, "keep.jpg", saved.Photos[0].MediaKey)
	require.Equal(t, "also-keep.jpg", saved.Photos[1].MediaKey)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestItemServiceDeleteShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Delete(ctx, "owner-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceDeleteShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceDeleteShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.Delete(t.Context(), "owner-2", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceDeleteShouldReturnErrorWhenMediaDeleteFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{{ID: "p1", MediaKey: "photo-1.jpg", Position: 0}}

	store := &mockItemStore{items: []domain.Item{item}}
	media := &mockMediaProvider{deleteErr: domain.ErrIO}
	svc := NewItemService(store, media, &mockLocationStore{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeleteShouldReturnErrorWhenStoreDeleteFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}, deleteErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{}, &mockLocationStore{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeleteShouldDeleteMediaKeysAndItemWhenCallerIsOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Photos = []domain.ItemPhoto{
		{ID: "p1", MediaKey: "photo-1.jpg", Position: 0},
		{ID: "p2", MediaKey: "photo-2.jpg", Position: 1},
	}

	store := &mockItemStore{items: []domain.Item{item}}
	media := &mockMediaProvider{}
	svc := NewItemService(store, media, &mockLocationStore{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Equal(t, []string{"photo-1.jpg", "photo-2.jpg"}, media.deletedKeys)
	require.Empty(t, store.items)
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
