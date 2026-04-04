package service

import (
	"context"
	"testing"

	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

// ── Move ──────────────────────────────────────────────────────────────────────

func TestLocationServiceMoveShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Move(ctx, "owner-1", "loc-1", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceMoveShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	repo := &mockLocationRepo{locations: []domain.Location{loc}, saveErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceMoveShouldReturnErrorWhenGetFailsDuringCircularCheck(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	grandParentID := "grandparent-1"
	var parent domain.Location
	parent.ID = "parent-1"
	parent.OwnerID = "owner-1"
	parent.ParentID = &grandParentID

	newParentID := "parent-1"
	repo := &mockLocationRepo{
		locations: []domain.Location{loc, parent},
		getErrFor: map[string]error{"grandparent-1": domain.ErrIO},
	}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", &newParentID)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceMoveShouldReturnErrConflictWhenNewParentIsADescendant(t *testing.T) {
	locID := "loc-1"
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	var child domain.Location
	child.ID = "child-1"
	child.OwnerID = "owner-1"
	child.ParentID = &locID

	newParentID := "child-1"
	repo := &mockLocationRepo{locations: []domain.Location{loc, child}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", &newParentID)
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestLocationServiceMoveShouldReturnErrForbiddenWhenNewParentBelongsToDifferentOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	var other domain.Location
	other.ID = "parent-2"
	other.OwnerID = "owner-2"

	newParentID := "parent-2"
	repo := &mockLocationRepo{locations: []domain.Location{loc, other}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", &newParentID)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceMoveShouldReturnErrNotFoundWhenNewParentDoesNotExist(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	newParentID := "nonexistent-parent"
	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", &newParentID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceMoveShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"

	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", nil)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceMoveShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", nil)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceMoveShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	repo := &mockLocationRepo{getErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceMoveShouldMakeLocationRootWhenNewParentIDIsNil(t *testing.T) {
	parentID := "parent-1"
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.ParentID = &parentID

	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	got, err := svc.Move(t.Context(), "owner-1", "loc-1", nil)
	require.NoError(t, err)
	require.Equal(t, "loc-1", got.GetID())
	require.Nil(t, got.ParentID)
	require.Nil(t, repo.locations[0].ParentID)
}

func TestLocationServiceMoveShouldSucceedWhenAncestorInChainIsMissing(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	ghostParentID := "ghost-id"
	var parent domain.Location
	parent.ID = "parent-1"
	parent.OwnerID = "owner-1"
	parent.ParentID = &ghostParentID // ghost-id does not exist in store

	newParentID := "parent-1"
	repo := &mockLocationRepo{locations: []domain.Location{loc, parent}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	got, err := svc.Move(t.Context(), "owner-1", "loc-1", &newParentID)
	require.NoError(t, err)
	require.NotNil(t, got.ParentID)
	require.Equal(t, "parent-1", *got.ParentID)
}

func TestLocationServiceMoveShouldReparentLocationWhenNewParentExists(t *testing.T) {
	oldParentID := "parent-1"
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.ParentID = &oldParentID

	var newParent domain.Location
	newParent.ID = "parent-2"
	newParent.OwnerID = "owner-1"

	newParentID := "parent-2"
	repo := &mockLocationRepo{locations: []domain.Location{loc, newParent}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	got, err := svc.Move(t.Context(), "owner-1", "loc-1", &newParentID)
	require.NoError(t, err)
	require.Equal(t, "loc-1", got.GetID())
	require.NotNil(t, got.ParentID)
	require.Equal(t, "parent-2", *got.ParentID)
	require.Equal(t, "parent-2", *repo.locations[0].ParentID)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestLocationServiceDeleteShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Delete(ctx, "owner-1", "loc-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceDeleteShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	repo := &mockLocationRepo{getErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceDeleteShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceDeleteShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"

	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceDeleteShouldReturnErrorWhenHasChildrenFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	repo := &mockLocationRepo{locations: []domain.Location{loc}, hasChildrenErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceDeleteShouldReturnErrConflictWhenLocationHasChildren(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	repo := &mockLocationRepo{locations: []domain.Location{loc}, hasChildrenResult: true}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestLocationServiceDeleteShouldReturnErrorWhenCountByLocationFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	locRepo := &mockLocationRepo{locations: []domain.Location{loc}}
	itemRepo := &mockItemRepo{countByLocationErr: domain.ErrIO}
	svc := NewLocationService(locRepo, itemRepo, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceDeleteShouldReturnErrConflictWhenLocationHasItemsAssigned(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	locRepo := &mockLocationRepo{locations: []domain.Location{loc}}
	itemRepo := &mockItemRepo{countByLocationResult: 1}
	svc := NewLocationService(locRepo, itemRepo, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestLocationServiceDeleteShouldReturnErrorWhenStoreDeleteFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	repo := &mockLocationRepo{locations: []domain.Location{loc}, deleteErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceDeleteShouldDeleteLocationWhenCallerIsOwnerAndNoConflicts(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.NoError(t, err)
	require.Empty(t, repo.locations)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestLocationServiceGetByIDShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.GetByID(ctx, "owner-1", "loc-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceGetByIDShouldReturnErrorWhenHasReadAccessFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"

	shareChecker := &mockShareAccessChecker{err: domain.ErrIO}
	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, shareChecker)

	_, err := svc.GetByID(t.Context(), "caller-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceGetByIDShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	repo := &mockLocationRepo{getErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.GetByID(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceGetByIDShouldReturnErrForbiddenWhenCallerHasNoShareAccess(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"

	shareChecker := &mockShareAccessChecker{hasAccess: false}
	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, shareChecker)

	_, err := svc.GetByID(t.Context(), "caller-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceGetByIDShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.GetByID(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// TestLocationServiceGetByIDShouldReturnLocationWhenShareCheckerGrantsAccess covers the
// direct-share and ancestor-share paths as a group: all resolve to HasReadAccess returning
// true. Per-path logic is tested in share_test.go.
func TestLocationServiceGetByIDShouldReturnLocationWhenShareCheckerGrantsAccess(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"
	loc.Label = "Closet"

	shareChecker := &mockShareAccessChecker{hasAccess: true}
	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, shareChecker)

	got, err := svc.GetByID(t.Context(), "caller-1", "loc-1")
	require.NoError(t, err)
	require.Equal(t, loc, got)
}

// TestLocationServiceGetByIDShouldReturnChildWhenParentLocationIsShared verifies that
// a child location is accessible when HasReadAccess grants it. Ancestor traversal is
// the ShareService's responsibility; LocationService just passes the child's ID.
func TestLocationServiceGetByIDShouldReturnChildWhenParentLocationIsShared(t *testing.T) {
	parentID := "parent-1"
	var child domain.Location
	child.ID = "child-1"
	child.OwnerID = "owner-2"
	child.ParentID = &parentID
	child.Label = "Shelf"

	shareChecker := &mockShareAccessChecker{hasAccess: true}
	repo := &mockLocationRepo{locations: []domain.Location{child}}
	svc := NewLocationService(repo, &mockItemRepo{}, shareChecker)

	got, err := svc.GetByID(t.Context(), "caller-1", "child-1")
	require.NoError(t, err)
	require.Equal(t, child, got)
}

func TestLocationServiceGetByIDShouldReturnLocationWhenCallerIsOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.Label = "Closet"

	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	got, err := svc.GetByID(t.Context(), "owner-1", "loc-1")
	require.NoError(t, err)
	require.Equal(t, loc, got)
}

// ── ListByOwner ───────────────────────────────────────────────────────────────

func TestLocationServiceListByOwnerShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListByOwner(ctx, "owner-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceListByOwnerShouldReturnErrorWhenRepoListByOwnerFails(t *testing.T) {
	repo := &mockLocationRepo{listByOwnerErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.ListByOwner(t.Context(), "owner-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceListByOwnerShouldReturnOnlyCallerLocations(t *testing.T) {
	var loc1, loc2, loc3 domain.Location
	loc1.ID = "loc-1"
	loc1.OwnerID = "owner-1"
	loc2.ID = "loc-2"
	loc2.OwnerID = "owner-2"
	loc3.ID = "loc-3"
	loc3.OwnerID = "owner-1"

	repo := &mockLocationRepo{locations: []domain.Location{loc1, loc2, loc3}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	got, err := svc.ListByOwner(t.Context(), "owner-1")
	require.NoError(t, err)
	require.ElementsMatch(t, []domain.Location{loc1, loc3}, got)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestLocationServiceUpdateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Update(ctx, "owner-1", "loc-1", "New Label")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceUpdateShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	repo := &mockLocationRepo{getErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Update(t.Context(), "owner-1", "loc-1", "New Label")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceUpdateShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Update(t.Context(), "owner-1", "loc-1", "New Label")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceUpdateShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"

	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Update(t.Context(), "owner-1", "loc-1", "New Label")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceUpdateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	repo := &mockLocationRepo{locations: []domain.Location{loc}, saveErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Update(t.Context(), "owner-1", "loc-1", "New Label")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceUpdateShouldReturnUpdatedLocationWhenCallerIsOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.Label = "Old Label"

	repo := &mockLocationRepo{locations: []domain.Location{loc}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	got, err := svc.Update(t.Context(), "owner-1", "loc-1", "New Label")
	require.NoError(t, err)
	require.Equal(t, "loc-1", got.GetID())
	require.Equal(t, "owner-1", got.OwnerID)
	require.Equal(t, "New Label", got.Label)
	require.Equal(t, "New Label", repo.locations[0].Label)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestLocationServiceCreateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Create(ctx, "owner-1", "Closet", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceCreateShouldReturnErrorWhenParentStoreGetFails(t *testing.T) {
	parentID := "parent-1"
	repo := &mockLocationRepo{getErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceCreateShouldReturnErrNotFoundWhenParentDoesNotExist(t *testing.T) {
	parentID := "parent-1"
	svc := NewLocationService(&mockLocationRepo{}, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceCreateShouldReturnErrForbiddenWhenParentBelongsToDifferentCaller(t *testing.T) {
	var parent domain.Location
	parent.ID = "parent-1"
	parent.OwnerID = "owner-2"

	parentID := "parent-1"
	repo := &mockLocationRepo{locations: []domain.Location{parent}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceCreateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	repo := &mockLocationRepo{saveErr: domain.ErrIO}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	_, err := svc.Create(t.Context(), "owner-1", "Closet", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceCreateShouldCreateRootLocationWhenParentIDIsNil(t *testing.T) {
	repo := &mockLocationRepo{}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	loc, err := svc.Create(t.Context(), "owner-1", "Closet", nil)
	require.NoError(t, err)
	require.NotEmpty(t, loc.GetID())
	require.Equal(t, "owner-1", loc.OwnerID)
	require.Equal(t, "Closet", loc.Label)
	require.Nil(t, loc.ParentID)
	require.False(t, loc.CreatedAt.IsZero())
	require.Len(t, repo.locations, 1)
}

func TestLocationServiceCreateShouldCreateChildLocationWhenParentExists(t *testing.T) {
	var parent domain.Location
	parent.ID = "parent-1"
	parent.OwnerID = "owner-1"
	parent.Label = "Closet"

	parentID := "parent-1"
	repo := &mockLocationRepo{locations: []domain.Location{parent}}
	svc := NewLocationService(repo, &mockItemRepo{}, &mockShareAccessChecker{})

	loc, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.NoError(t, err)
	require.NotEmpty(t, loc.GetID())
	require.Equal(t, "owner-1", loc.OwnerID)
	require.Equal(t, "Shelf", loc.Label)
	require.NotNil(t, loc.ParentID)
	require.Equal(t, "parent-1", *loc.ParentID)
	require.False(t, loc.CreatedAt.IsZero())
	require.Len(t, repo.locations, 2)
}

// ── Full cycle ────────────────────────────────────────────────────────────────

func TestLocationServiceShouldSucceedWhenRunningFullCreateUpdateListMoveDeleteCycle(t *testing.T) {
	locRepo := &mockLocationRepo{}
	svc := NewLocationService(locRepo, &mockItemRepo{}, &mockShareAccessChecker{})
	ctx := t.Context()

	// Create root location.
	root, err := svc.Create(ctx, "owner-1", "Wardrobe", nil)
	require.NoError(t, err)
	require.NotEmpty(t, root.GetID())

	// Create child location under root.
	rootID := root.GetID()
	child, err := svc.Create(ctx, "owner-1", "Shelf", &rootID)
	require.NoError(t, err)
	require.NotEmpty(t, child.GetID())
	require.Equal(t, rootID, *child.ParentID)

	// Update child label.
	updated, err := svc.Update(ctx, "owner-1", child.GetID(), "Top Shelf")
	require.NoError(t, err)
	require.Equal(t, "Top Shelf", updated.Label)

	// List — both locations belong to owner-1.
	locs, err := svc.ListByOwner(ctx, "owner-1")
	require.NoError(t, err)
	require.Len(t, locs, 2)

	// Move child to root (nil parent).
	moved, err := svc.Move(ctx, "owner-1", child.GetID(), nil)
	require.NoError(t, err)
	require.Nil(t, moved.ParentID)

	// Delete child first (now at root, no children, no items).
	err = svc.Delete(ctx, "owner-1", child.GetID())
	require.NoError(t, err)

	// Delete root.
	err = svc.Delete(ctx, "owner-1", root.GetID())
	require.NoError(t, err)

	// Store must be empty.
	require.Empty(t, locRepo.locations)
}
