package service_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/outfitte/backend/internal/adapter/media/local"
	jsonstore "github.com/outfitte/backend/internal/adapter/store/json"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/service"
	"github.com/stretchr/testify/require"
)

// noopShareChecker always denies access, satisfying the shareAccessChecker
// interface for tests that never exercise the shared-access path.
type noopShareChecker struct{}

func (n *noopShareChecker) HasReadAccess(_ context.Context, _ string, _ domain.ShareTargetType, _ string) (bool, error) {
	return false, nil
}

func (n *noopShareChecker) DeleteByTarget(_ context.Context, _ domain.ShareTargetType, _ string) error {
	return nil
}

func TestItemServiceShouldCompleteFullCycleWhenUploadGetThenDelete(t *testing.T) {
	itemRepo := jsonstore.NewItemRepository(t.TempDir())
	media := local.NewProvider(t.TempDir())
	locRepo := jsonstore.NewLocationRepository(t.TempDir())
	svc := service.NewItemService(itemRepo, media, locRepo, service.NewCategoryService(), &noopShareChecker{})

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
