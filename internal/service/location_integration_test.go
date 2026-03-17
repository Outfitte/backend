package service_test

import (
	"testing"

	jsonstore "github.com/outfitte/outfitte/internal/adapter/store/json"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/service"
	"github.com/stretchr/testify/require"
)

func TestLocationServiceShouldCompleteFullCycleWhenCreateUpdateListMoveDelete(t *testing.T) {
	locStore := jsonstore.NewProvider[domain.Location](t.TempDir(), "locations.json")
	itemStore := jsonstore.NewProvider[domain.Item](t.TempDir(), "items.json")
	svc := service.NewLocationService(locStore, itemStore)

	ctx := t.Context()

	// Create a root location.
	root, err := svc.Create(ctx, "owner-1", "Wardrobe", nil)
	require.NoError(t, err)
	require.NotEmpty(t, root.GetID())
	require.Equal(t, "owner-1", root.OwnerID)
	require.Equal(t, "Wardrobe", root.Label)
	require.Nil(t, root.ParentID)

	// Create a child location under root.
	rootID := root.GetID()
	child, err := svc.Create(ctx, "owner-1", "Shelf", &rootID)
	require.NoError(t, err)
	require.NotEmpty(t, child.GetID())
	require.NotEqual(t, root.GetID(), child.GetID())
	require.NotNil(t, child.ParentID)
	require.Equal(t, rootID, *child.ParentID)

	// Update the child label.
	updated, err := svc.Update(ctx, "owner-1", child.GetID(), "Top Shelf")
	require.NoError(t, err)
	require.Equal(t, child.GetID(), updated.GetID())
	require.Equal(t, "Top Shelf", updated.Label)

	// Verify the update is persisted via GetByID.
	got, err := svc.GetByID(ctx, "owner-1", child.GetID())
	require.NoError(t, err)
	require.Equal(t, "Top Shelf", got.Label)

	// List — both locations must be present.
	locs, err := svc.ListByOwner(ctx, "owner-1")
	require.NoError(t, err)
	require.Len(t, locs, 2)

	// Move child to root (nil parent).
	moved, err := svc.Move(ctx, "owner-1", child.GetID(), nil)
	require.NoError(t, err)
	require.Equal(t, child.GetID(), moved.GetID())
	require.Nil(t, moved.ParentID)

	// Verify move is persisted.
	got, err = svc.GetByID(ctx, "owner-1", child.GetID())
	require.NoError(t, err)
	require.Nil(t, got.ParentID)

	// Delete child first (no children, no items).
	err = svc.Delete(ctx, "owner-1", child.GetID())
	require.NoError(t, err)

	// Delete root.
	err = svc.Delete(ctx, "owner-1", root.GetID())
	require.NoError(t, err)

	// Store must be empty.
	remaining, err := svc.ListByOwner(ctx, "owner-1")
	require.NoError(t, err)
	require.Empty(t, remaining)
}
