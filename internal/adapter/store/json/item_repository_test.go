package json_test

import (
	"context"
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestNewItemRepositoryShouldImplementItemRepository(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	require.Implements(t, (*ports.ItemRepository)(nil), r)
}

func TestItemGetShouldReturnNotFoundWhenItemDoesNotExist(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())

	_, err := r.Get(t.Context(), "i1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemGetShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Get(ctx, "i1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemSaveShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Save(ctx, domain.Item{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemDeleteShouldReturnNotFoundWhenItemDoesNotExist(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())

	err := r.Delete(t.Context(), "i1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemDeleteShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Delete(ctx, "i1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemGetShouldReturnItemWhenFound(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	var item domain.Item
	item.ID = "i1"
	item.OwnerID = "o1"
	require.NoError(t, r.Save(t.Context(), item))

	got, err := r.Get(t.Context(), "i1")
	require.NoError(t, err)
	require.Equal(t, item, got)
}

func TestItemDeleteShouldRemoveItemWhenFound(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	var item domain.Item
	item.ID = "i1"
	require.NoError(t, r.Save(t.Context(), item))

	require.NoError(t, r.Delete(t.Context(), "i1"))

	_, err := r.Get(t.Context(), "i1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemListByOwnerShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListByOwner(ctx, "o1", ports.ItemListFilter{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemListByOwnerShouldReturnEmptyWhenNoItemsExist(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())

	items, err := r.ListByOwner(t.Context(), "o1", ports.ItemListFilter{})
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestItemListByOwnerShouldFilterByOwner(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	var i1, i2, i3 domain.Item
	i1.ID = "i1"
	i1.OwnerID = "o1"
	i2.ID = "i2"
	i2.OwnerID = "o1"
	i3.ID = "i3"
	i3.OwnerID = "o2"
	require.NoError(t, r.Save(t.Context(), i1))
	require.NoError(t, r.Save(t.Context(), i2))
	require.NoError(t, r.Save(t.Context(), i3))

	items, err := r.ListByOwner(t.Context(), "o1", ports.ItemListFilter{Status: ports.ItemStatusActive})
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func TestItemListByOwnerAllStatusShouldReturnAllOwnerItems(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	var i1, i2 domain.Item
	i1.ID = "i1"
	i1.OwnerID = "o1"
	i2.ID = "i2"
	i2.OwnerID = "o1"
	require.NoError(t, r.Save(t.Context(), i1))
	require.NoError(t, r.Save(t.Context(), i2))

	items, err := r.ListByOwner(t.Context(), "o1", ports.ItemListFilter{Status: ports.ItemStatusAll})
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func TestItemListByOwnerArchivedStatusShouldReturnEmpty(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	var i1 domain.Item
	i1.ID = "i1"
	i1.OwnerID = "o1"
	require.NoError(t, r.Save(t.Context(), i1))

	items, err := r.ListByOwner(t.Context(), "o1", ports.ItemListFilter{Status: ports.ItemStatusArchived})
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestItemCountByLocationShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.CountByLocation(ctx, "loc1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemCountByLocationShouldReturnZeroWhenNoItems(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())

	count, err := r.CountByLocation(t.Context(), "loc1")
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestItemCountByLocationShouldReturnCorrectCount(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	loc := "loc1"
	var i1, i2, i3 domain.Item
	i1.ID = "i1"
	i1.LocationID = &loc
	i2.ID = "i2"
	i2.LocationID = &loc
	i3.ID = "i3"
	require.NoError(t, r.Save(t.Context(), i1))
	require.NoError(t, r.Save(t.Context(), i2))
	require.NoError(t, r.Save(t.Context(), i3))

	count, err := r.CountByLocation(t.Context(), "loc1")
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestItemSavePhotoShouldReturnNotFoundWhenItemDoesNotExist(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())

	err := r.SavePhoto(t.Context(), "i1", "p1", "key1", 0)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemSavePhotoShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.SavePhoto(ctx, "i1", "p1", "key1", 0)
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemSavePhotoShouldAddPhotoToItem(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	var item domain.Item
	item.ID = "i1"
	require.NoError(t, r.Save(t.Context(), item))

	require.NoError(t, r.SavePhoto(t.Context(), "i1", "p1", "key1", 0))

	keys, err := r.ListPhotoKeys(t.Context(), "i1")
	require.NoError(t, err)
	require.Equal(t, []string{"key1"}, keys)
}

func TestItemDeletePhotoShouldReturnNotFoundWhenItemDoesNotExist(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())

	err := r.DeletePhoto(t.Context(), "i1", "key1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemDeletePhotoShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.DeletePhoto(ctx, "i1", "key1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemDeletePhotoShouldRemovePhoto(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	var item domain.Item
	item.ID = "i1"
	require.NoError(t, r.Save(t.Context(), item))
	require.NoError(t, r.SavePhoto(t.Context(), "i1", "p1", "key1", 0))

	require.NoError(t, r.DeletePhoto(t.Context(), "i1", "key1"))

	keys, err := r.ListPhotoKeys(t.Context(), "i1")
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestItemListPhotoKeysShouldReturnNotFoundWhenItemDoesNotExist(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())

	_, err := r.ListPhotoKeys(t.Context(), "i1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestItemListPhotoKeysShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListPhotoKeys(ctx, "i1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestItemListPhotoKeysShouldReturnKeysOrderedByPosition(t *testing.T) {
	r := json.NewItemRepository(t.TempDir())
	var item domain.Item
	item.ID = "i1"
	require.NoError(t, r.Save(t.Context(), item))
	require.NoError(t, r.SavePhoto(t.Context(), "i1", "p1", "keyA", 2))
	require.NoError(t, r.SavePhoto(t.Context(), "i1", "p2", "keyB", 0))
	require.NoError(t, r.SavePhoto(t.Context(), "i1", "p3", "keyC", 1))

	keys, err := r.ListPhotoKeys(t.Context(), "i1")
	require.NoError(t, err)
	require.Equal(t, []string{"keyB", "keyC", "keyA"}, keys)
}
