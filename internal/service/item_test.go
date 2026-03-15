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

func TestItemServiceCreateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Create(ctx, "owner-1", CreateItemInput{Name: "Jacket"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceCreateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	store := &mockItemStore{saveErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{})

	_, err := svc.Create(t.Context(), "owner-1", CreateItemInput{Name: "Jacket"})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceCreateShouldCreateItemWithCallerAsOwner(t *testing.T) {
	store := &mockItemStore{}
	svc := NewItemService(store, &mockMediaProvider{})

	input := CreateItemInput{
		Name:       "Jacket",
		Brand:      "Patagonia",
		CategoryID: "cat-1",
		Color:      "Black",
		Size:       "M",
		PhotoKeys:  []string{"photo-1.jpg"},
	}

	item, err := svc.Create(t.Context(), "owner-1", input)
	require.NoError(t, err)
	require.NotEmpty(t, item.GetID())
	require.Equal(t, "owner-1", item.OwnerID)
	require.Equal(t, "Jacket", item.Name)
	require.Equal(t, "Patagonia", item.Brand)
	require.Equal(t, "cat-1", item.CategoryID)
	require.Equal(t, "Black", item.Color)
	require.Equal(t, "M", item.Size)
	require.Equal(t, []string{"photo-1.jpg"}, item.PhotoKeys)
	require.False(t, item.CreatedAt.IsZero())
	require.Len(t, store.items, 1)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestItemServiceGetByIDShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.GetByID(ctx, "owner-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceGetByIDShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{})

	_, err := svc.GetByID(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceGetByIDShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockItemStore{getErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{})

	_, err := svc.GetByID(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceGetByIDShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{})

	_, err := svc.GetByID(t.Context(), "owner-2", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceGetByIDShouldReturnItemWhenCallerIsOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Jacket"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{})

	got, err := svc.GetByID(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Equal(t, item, got)
}

// ── ListByOwner ───────────────────────────────────────────────────────────────

func TestItemServiceListByOwnerShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListByOwner(ctx, "owner-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceListByOwnerShouldReturnErrorWhenListFails(t *testing.T) {
	store := &mockItemStore{listErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{})

	_, err := svc.ListByOwner(t.Context(), "owner-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceListByOwnerShouldReturnEmptySliceWhenCallerHasNoItems(t *testing.T) {
	var other domain.Item
	other.ID = "item-1"
	other.OwnerID = "owner-2"

	store := &mockItemStore{items: []domain.Item{other}}
	svc := NewItemService(store, &mockMediaProvider{})

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
	svc := NewItemService(store, &mockMediaProvider{})

	got, err := svc.ListByOwner(t.Context(), "owner-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, []domain.Item{item1, item3}, got)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestItemServiceUpdateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Update(ctx, "owner-1", "item-1", UpdateItemInput{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceUpdateShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{})

	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceUpdateShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockItemStore{getErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{})

	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUpdateShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{})

	_, err := svc.Update(t.Context(), "owner-2", "item-1", UpdateItemInput{})
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceUpdateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{})

	_, err := svc.Update(t.Context(), "owner-1", "item-1", UpdateItemInput{Name: "Updated"})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUpdateShouldUpdateItemWhenCallerIsOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.Name = "Old Name"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{})

	input := UpdateItemInput{
		Name:       "New Name",
		Brand:      "New Brand",
		CategoryID: "cat-2",
		Color:      "Blue",
		Size:       "L",
		PhotoKeys:  []string{"new-photo.jpg"},
	}

	got, err := svc.Update(t.Context(), "owner-1", "item-1", input)
	require.NoError(t, err)
	require.Equal(t, "item-1", got.GetID())
	require.Equal(t, "owner-1", got.OwnerID)
	require.Equal(t, "New Name", got.Name)
	require.Equal(t, "New Brand", got.Brand)
	require.Equal(t, "cat-2", got.CategoryID)
	require.Equal(t, "Blue", got.Color)
	require.Equal(t, "L", got.Size)
	require.Equal(t, []string{"new-photo.jpg"}, got.PhotoKeys)
}

// ── UploadPhoto ───────────────────────────────────────────────────────────────

func TestItemServiceUploadPhotoShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.UploadPhoto(ctx, "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceUploadPhotoShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceUploadPhotoShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockItemStore{getErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUploadPhotoShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{})

	err := svc.UploadPhoto(t.Context(), "owner-2", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceUploadPhotoShouldReturnErrorWhenUploadFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	media := &mockMediaProvider{uploadErr: domain.ErrIO}
	svc := NewItemService(store, media)

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUploadPhotoShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}, saveErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{})

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "photo.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceUploadPhotoShouldAppendKeyAndSaveItemWhenSuccessful(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.PhotoKeys = []string{"existing.jpg"}

	store := &mockItemStore{items: []domain.Item{item}}
	media := &mockMediaProvider{}
	svc := NewItemService(store, media)

	err := svc.UploadPhoto(t.Context(), "owner-1", "item-1", nil, "new.jpg")
	require.NoError(t, err)

	// Key must follow the pattern items/<itemID>/<uuid>/<filename>
	require.NotEmpty(t, media.uploadedKey)
	require.Equal(t, "items/item-1/", media.uploadedKey[:13])
	require.Equal(t, "/new.jpg", media.uploadedKey[len(media.uploadedKey)-8:])

	// Item must be updated in the store
	saved, err := store.Get(t.Context(), "item-1")
	require.NoError(t, err)
	require.Len(t, saved.PhotoKeys, 2)
	require.Equal(t, "existing.jpg", saved.PhotoKeys[0])
	require.Equal(t, media.uploadedKey, saved.PhotoKeys[1])
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestItemServiceDeleteShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Delete(ctx, "owner-1", "item-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemServiceDeleteShouldReturnErrNotFoundWhenItemDoesNotExist(t *testing.T) {
	svc := NewItemService(&mockItemStore{}, &mockMediaProvider{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemServiceDeleteShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}}
	svc := NewItemService(store, &mockMediaProvider{})

	err := svc.Delete(t.Context(), "owner-2", "item-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestItemServiceDeleteShouldReturnErrorWhenMediaDeleteFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.PhotoKeys = []string{"photo-1.jpg"}

	store := &mockItemStore{items: []domain.Item{item}}
	media := &mockMediaProvider{deleteErr: domain.ErrIO}
	svc := NewItemService(store, media)

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeleteShouldReturnErrorWhenStoreDeleteFails(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"

	store := &mockItemStore{items: []domain.Item{item}, deleteErr: domain.ErrIO}
	svc := NewItemService(store, &mockMediaProvider{})

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestItemServiceDeleteShouldDeleteMediaKeysAndItemWhenCallerIsOwner(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.PhotoKeys = []string{"photo-1.jpg", "photo-2.jpg"}

	store := &mockItemStore{items: []domain.Item{item}}
	media := &mockMediaProvider{}
	svc := NewItemService(store, media)

	err := svc.Delete(t.Context(), "owner-1", "item-1")
	require.NoError(t, err)
	require.Equal(t, []string{"photo-1.jpg", "photo-2.jpg"}, media.deletedKeys)
	require.Empty(t, store.items)
}
