package service_test

import (
	"io"
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/media/local"
	jsonstore "github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/service"
	"github.com/stretchr/testify/require"
)

func TestItemServiceShouldCompleteFullCycleWhenUploadGetThenDelete(t *testing.T) {
	store := jsonstore.NewProvider[domain.Item](t.TempDir(), "items.json")
	media := local.NewProvider(t.TempDir())
	locStore := jsonstore.NewProvider[domain.Location](t.TempDir(), "locations.json")
	svc := service.NewItemService(store, media, locStore, service.NewCategoryService())

	ctx := t.Context()

	// Create an item to attach photos to.
	item, err := svc.Create(ctx, "owner-1", service.CreateItemInput{Name: "Jacket"})
	require.NoError(t, err)

	// Upload a photo.
	const content = "fake image data"
	err = svc.UploadPhoto(ctx, "owner-1", item.GetID(), strings.NewReader(content), "photo.jpg")
	require.NoError(t, err)

	// Get the item — verify the photo key was recorded.
	got, err := svc.GetByID(ctx, "owner-1", item.GetID())
	require.NoError(t, err)
	require.Len(t, got.Photos, 1)
	photoKey := got.Photos[0].MediaKey

	// Download the photo via the media provider and verify its content.
	rc, err := media.Download(ctx, photoKey)
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	require.NoError(t, rc.Close())
	require.NoError(t, err)
	require.Equal(t, content, string(data))

	// Delete the photo via the service.
	err = svc.DeletePhoto(ctx, "owner-1", item.GetID(), photoKey)
	require.NoError(t, err)

	// Verify the photo key is removed from the item.
	got, err = svc.GetByID(ctx, "owner-1", item.GetID())
	require.NoError(t, err)
	require.Empty(t, got.Photos)

	// Verify the file is gone from the media provider.
	_, err = media.Download(ctx, photoKey)
	require.ErrorIs(t, err, domain.ErrNotFound)
}
