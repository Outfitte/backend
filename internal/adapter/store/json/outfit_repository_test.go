package json_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/adapter/store/json"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

func TestNewOutfitRepositoryShouldImplementOutfitRepository(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	require.Implements(t, (*ports.OutfitRepository)(nil), r)
}

func TestOutfitGetShouldReturnNotFoundWhenOutfitDoesNotExist(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())

	_, err := r.Get(t.Context(), "o1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitGetShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Get(ctx, "o1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitSaveShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Save(ctx, domain.Outfit{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitDeleteShouldReturnNotFoundWhenOutfitDoesNotExist(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())

	err := r.Delete(t.Context(), "o1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitDeleteShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Delete(ctx, "o1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitListByOwnerShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListByOwner(ctx, "u1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitListByOwnerShouldReturnErrIOWhenDataFileIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "outfits.json"), []byte("not json"), 0o644))
	r := json.NewOutfitRepository(dir)

	_, err := r.ListByOwner(t.Context(), "u1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestOutfitSaveItemShouldReturnNotFoundWhenOutfitDoesNotExist(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())

	err := r.SaveItem(t.Context(), "o1", "i1", 0)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitSaveItemShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.SaveItem(ctx, "o1", "i1", 0)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitDeleteItemShouldReturnNotFoundWhenOutfitDoesNotExist(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())

	err := r.DeleteItem(t.Context(), "o1", "i1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitDeleteItemShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.DeleteItem(ctx, "o1", "i1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitListItemIDsShouldReturnNotFoundWhenOutfitDoesNotExist(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())

	_, err := r.ListItemIDs(t.Context(), "o1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitListItemIDsShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListItemIDs(ctx, "o1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitSavePhotoShouldReturnNotFoundWhenOutfitDoesNotExist(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())

	err := r.SavePhoto(t.Context(), "o1", "p1", "key1", 0)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitSavePhotoShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.SavePhoto(ctx, "o1", "p1", "key1", 0)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitDeletePhotoShouldReturnNotFoundWhenOutfitDoesNotExist(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())

	err := r.DeletePhoto(t.Context(), "o1", "key1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitDeletePhotoShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.DeletePhoto(ctx, "o1", "key1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestOutfitGetShouldReturnOutfitWhenFound(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	outfit.OwnerID = "u1"
	require.NoError(t, r.Save(t.Context(), outfit))

	got, err := r.Get(t.Context(), "o1")
	require.NoError(t, err)
	require.Equal(t, outfit, got)
}

func TestOutfitDeleteShouldRemoveOutfitWhenFound(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))

	require.NoError(t, r.Delete(t.Context(), "o1"))

	_, err := r.Get(t.Context(), "o1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestOutfitListByOwnerShouldReturnEmptyWhenNoOutfitsExist(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())

	outfits, err := r.ListByOwner(t.Context(), "u1")
	require.NoError(t, err)
	require.Empty(t, outfits)
}

func TestOutfitListByOwnerShouldFilterByOwner(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var o1, o2, o3 domain.Outfit
	o1.ID = "o1"
	o1.OwnerID = "u1"
	o2.ID = "o2"
	o2.OwnerID = "u1"
	o3.ID = "o3"
	o3.OwnerID = "u2"
	require.NoError(t, r.Save(t.Context(), o1))
	require.NoError(t, r.Save(t.Context(), o2))
	require.NoError(t, r.Save(t.Context(), o3))

	outfits, err := r.ListByOwner(t.Context(), "u1")
	require.NoError(t, err)
	require.Len(t, outfits, 2)
}

func TestOutfitSaveItemShouldAddItemToOutfit(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))

	require.NoError(t, r.SaveItem(t.Context(), "o1", "i1", 0))

	ids, err := r.ListItemIDs(t.Context(), "o1")
	require.NoError(t, err)
	require.Equal(t, []string{"i1"}, ids)
}

func TestOutfitSaveItemShouldUpdatePositionWhenItemAlreadyExists(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))
	require.NoError(t, r.SaveItem(t.Context(), "o1", "i1", 0))

	require.NoError(t, r.SaveItem(t.Context(), "o1", "i1", 5))

	got, err := r.Get(t.Context(), "o1")
	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	require.Equal(t, 5, got.Items[0].Position)
}

func TestOutfitDeleteItemShouldRemoveItemFromOutfit(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))
	require.NoError(t, r.SaveItem(t.Context(), "o1", "i1", 0))

	require.NoError(t, r.DeleteItem(t.Context(), "o1", "i1"))

	ids, err := r.ListItemIDs(t.Context(), "o1")
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestOutfitDeleteItemShouldKeepOtherItemsWhenDeletingOne(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))
	require.NoError(t, r.SaveItem(t.Context(), "o1", "i1", 0))
	require.NoError(t, r.SaveItem(t.Context(), "o1", "i2", 1))

	require.NoError(t, r.DeleteItem(t.Context(), "o1", "i1"))

	ids, err := r.ListItemIDs(t.Context(), "o1")
	require.NoError(t, err)
	require.Equal(t, []string{"i2"}, ids)
}

func TestOutfitListItemIDsShouldReturnIDsOrderedByPosition(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))
	require.NoError(t, r.SaveItem(t.Context(), "o1", "iA", 2))
	require.NoError(t, r.SaveItem(t.Context(), "o1", "iB", 0))
	require.NoError(t, r.SaveItem(t.Context(), "o1", "iC", 1))

	ids, err := r.ListItemIDs(t.Context(), "o1")
	require.NoError(t, err)
	require.Equal(t, []string{"iB", "iC", "iA"}, ids)
}

func TestOutfitSavePhotoShouldAddPhotoToOutfit(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))

	require.NoError(t, r.SavePhoto(t.Context(), "o1", "p1", "key1", 0))

	got, err := r.Get(t.Context(), "o1")
	require.NoError(t, err)
	require.Len(t, got.Photos, 1)
	require.Equal(t, "key1", got.Photos[0].MediaKey)
}

func TestOutfitSavePhotoShouldUpdatePositionWhenMediaKeyAlreadyExists(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))
	require.NoError(t, r.SavePhoto(t.Context(), "o1", "p1", "key1", 0))

	require.NoError(t, r.SavePhoto(t.Context(), "o1", "p1", "key1", 5))

	got, err := r.Get(t.Context(), "o1")
	require.NoError(t, err)
	require.Len(t, got.Photos, 1)
	require.Equal(t, 5, got.Photos[0].Position)
}

func TestOutfitDeletePhotoShouldRemovePhotoFromOutfit(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))
	require.NoError(t, r.SavePhoto(t.Context(), "o1", "p1", "key1", 0))

	require.NoError(t, r.DeletePhoto(t.Context(), "o1", "key1"))

	got, err := r.Get(t.Context(), "o1")
	require.NoError(t, err)
	require.Empty(t, got.Photos)
}

func TestOutfitSavePhotoShouldUpdatePhotoIDWhenMediaKeyAlreadyExists(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))
	require.NoError(t, r.SavePhoto(t.Context(), "o1", "p1", "key1", 0))

	require.NoError(t, r.SavePhoto(t.Context(), "o1", "p2", "key1", 5))

	got, err := r.Get(t.Context(), "o1")
	require.NoError(t, err)
	require.Len(t, got.Photos, 1)
	require.Equal(t, "p2", got.Photos[0].ID)
	require.Equal(t, 5, got.Photos[0].Position)
}

func TestOutfitDeletePhotoShouldKeepOtherPhotosWhenDeletingOne(t *testing.T) {
	r := json.NewOutfitRepository(t.TempDir())
	var outfit domain.Outfit
	outfit.ID = "o1"
	require.NoError(t, r.Save(t.Context(), outfit))
	require.NoError(t, r.SavePhoto(t.Context(), "o1", "p1", "key1", 0))
	require.NoError(t, r.SavePhoto(t.Context(), "o1", "p2", "key2", 1))

	require.NoError(t, r.DeletePhoto(t.Context(), "o1", "key1"))

	got, err := r.Get(t.Context(), "o1")
	require.NoError(t, err)
	require.Len(t, got.Photos, 1)
	require.Equal(t, "key2", got.Photos[0].MediaKey)
}
