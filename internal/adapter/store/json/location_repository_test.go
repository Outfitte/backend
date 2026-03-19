package json_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestNewLocationRepositoryShouldImplementLocationRepository(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	require.Implements(t, (*ports.LocationRepository)(nil), r)
}

func TestGetShouldReturnNotFoundWhenLocationDoesNotExist(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())

	_, err := r.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetShouldReturnErrorWhenContextIsCancelledForLocation(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Get(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetShouldReturnLocationWhenFound(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	var loc domain.Location
	loc.ID = "42"
	loc.OwnerID = "owner1"
	loc.Label = "Wardrobe"
	require.NoError(t, r.Save(t.Context(), loc))

	got, err := r.Get(t.Context(), "42")
	require.NoError(t, err)
	require.Equal(t, loc, got)
}

func TestSaveShouldReturnErrorWhenContextIsCancelledForLocation(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Save(ctx, domain.Location{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnErrorWhenContextIsCancelledForLocation(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := r.Delete(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnNotFoundWhenLocationDoesNotExist(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())

	err := r.Delete(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteShouldRemoveLocationWhenFound(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	var loc domain.Location
	loc.ID = "42"
	require.NoError(t, r.Save(t.Context(), loc))

	require.NoError(t, r.Delete(t.Context(), "42"))

	_, err := r.Get(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestListByOwnerShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.ListByOwner(ctx, "owner1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestListByOwnerShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "locations.json"), []byte("not json"), 0o644))
	r := json.NewLocationRepository(dir)

	_, err := r.ListByOwner(t.Context(), "owner1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListByOwnerShouldReturnEmptyWhenNoLocationsExist(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())

	locs, err := r.ListByOwner(t.Context(), "owner1")
	require.NoError(t, err)
	require.Empty(t, locs)
}

func TestListByOwnerShouldReturnEmptyWhenNoLocationsMatchOwner(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	var loc domain.Location
	loc.ID = "1"
	loc.OwnerID = "owner2"
	require.NoError(t, r.Save(t.Context(), loc))

	locs, err := r.ListByOwner(t.Context(), "owner1")
	require.NoError(t, err)
	require.Empty(t, locs)
}

func TestListByOwnerShouldReturnOnlyLocationsForOwner(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	var loc1, loc2, loc3 domain.Location
	loc1.ID = "1"
	loc1.OwnerID = "owner1"
	loc2.ID = "2"
	loc2.OwnerID = "owner1"
	loc3.ID = "3"
	loc3.OwnerID = "owner2"
	require.NoError(t, r.Save(t.Context(), loc1))
	require.NoError(t, r.Save(t.Context(), loc2))
	require.NoError(t, r.Save(t.Context(), loc3))

	locs, err := r.ListByOwner(t.Context(), "owner1")
	require.NoError(t, err)
	require.Len(t, locs, 2)
}

func TestHasChildrenShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.HasChildren(ctx, "42")
	require.ErrorIs(t, err, context.Canceled)
}

func TestHasChildrenShouldReturnIOErrorWhenStorageIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "locations.json"), []byte("not json"), 0o644))
	r := json.NewLocationRepository(dir)

	_, err := r.HasChildren(t.Context(), "42")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestHasChildrenShouldReturnFalseWhenNoChildrenExist(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())

	has, err := r.HasChildren(t.Context(), "parent1")
	require.NoError(t, err)
	require.False(t, has)
}

func TestHasChildrenShouldReturnTrueWhenChildExists(t *testing.T) {
	r := json.NewLocationRepository(t.TempDir())
	parentID := "parent1"
	var child domain.Location
	child.ID = "child1"
	child.OwnerID = "owner1"
	child.ParentID = &parentID
	require.NoError(t, r.Save(t.Context(), child))

	has, err := r.HasChildren(t.Context(), "parent1")
	require.NoError(t, err)
	require.True(t, has)
}
