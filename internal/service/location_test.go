package service

import (
	"context"
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// mockLocationStore is an in-memory StorageProvider[domain.Location] for tests.
type mockLocationStore struct {
	locations []domain.Location
	getErr    error
	listErr   error
	saveErr   error
	deleteErr error
}

func (m *mockLocationStore) Get(_ context.Context, id string) (domain.Location, error) {
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

func (m *mockLocationStore) List(_ context.Context) ([]domain.Location, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.locations, nil
}

func (m *mockLocationStore) Save(_ context.Context, loc domain.Location) error {
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

func (m *mockLocationStore) Delete(_ context.Context, id string) error {
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

// ── Move ──────────────────────────────────────────────────────────────────────

func TestLocationServiceMoveShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Move(ctx, "owner-1", "loc-1", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceMoveShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	store := &mockLocationStore{locations: []domain.Location{loc}, saveErr: domain.ErrIO}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceMoveShouldReturnErrorWhenStoreListFailsDuringCircularCheck(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	var parent domain.Location
	parent.ID = "parent-1"
	parent.OwnerID = "owner-1"

	newParentID := "parent-1"
	store := &mockLocationStore{locations: []domain.Location{loc, parent}, listErr: domain.ErrIO}
	// getErr must be nil so Get works (needed for getOwnedLocation and validateParent)
	// but listErr causes list to fail during circular check
	svc := NewLocationService(store, &mockItemStore{})

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
	store := &mockLocationStore{locations: []domain.Location{loc, child}}
	svc := NewLocationService(store, &mockItemStore{})

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
	store := &mockLocationStore{locations: []domain.Location{loc, other}}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", &newParentID)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceMoveShouldReturnErrNotFoundWhenNewParentDoesNotExist(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	newParentID := "nonexistent-parent"
	store := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", &newParentID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceMoveShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"

	store := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", nil)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceMoveShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", nil)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceMoveShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockLocationStore{getErr: domain.ErrIO}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Move(t.Context(), "owner-1", "loc-1", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceMoveShouldMakeLocationRootWhenNewParentIDIsNil(t *testing.T) {
	parentID := "parent-1"
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.ParentID = &parentID

	store := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(store, &mockItemStore{})

	got, err := svc.Move(t.Context(), "owner-1", "loc-1", nil)
	require.NoError(t, err)
	require.Equal(t, "loc-1", got.GetID())
	require.Nil(t, got.ParentID)
	require.Nil(t, store.locations[0].ParentID)
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
	store := &mockLocationStore{locations: []domain.Location{loc, newParent}}
	svc := NewLocationService(store, &mockItemStore{})

	got, err := svc.Move(t.Context(), "owner-1", "loc-1", &newParentID)
	require.NoError(t, err)
	require.Equal(t, "loc-1", got.GetID())
	require.NotNil(t, got.ParentID)
	require.Equal(t, "parent-2", *got.ParentID)
	require.Equal(t, "parent-2", *store.locations[0].ParentID)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestLocationServiceDeleteShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := svc.Delete(ctx, "owner-1", "loc-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceDeleteShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockLocationStore{getErr: domain.ErrIO}
	svc := NewLocationService(store, &mockItemStore{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceDeleteShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceDeleteShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"

	store := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(store, &mockItemStore{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceDeleteShouldReturnErrorWhenStoreListFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	store := &mockLocationStore{locations: []domain.Location{loc}, listErr: domain.ErrIO}
	svc := NewLocationService(store, &mockItemStore{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceDeleteShouldReturnErrConflictWhenLocationHasChildren(t *testing.T) {
	parentID := "loc-1"
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	var child domain.Location
	child.ID = "loc-2"
	child.OwnerID = "owner-1"
	child.ParentID = &parentID

	store := &mockLocationStore{locations: []domain.Location{loc, child}}
	svc := NewLocationService(store, &mockItemStore{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestLocationServiceDeleteShouldReturnErrorWhenItemStoreListFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	locStore := &mockLocationStore{locations: []domain.Location{loc}}
	itemStore := &mockItemStore{listErr: domain.ErrIO}
	svc := NewLocationService(locStore, itemStore)

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceDeleteShouldReturnErrConflictWhenLocationHasItemsAssigned(t *testing.T) {
	locID := "loc-1"
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "owner-1"
	item.LocationID = &locID

	locStore := &mockLocationStore{locations: []domain.Location{loc}}
	itemStore := &mockItemStore{items: []domain.Item{item}}
	svc := NewLocationService(locStore, itemStore)

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestLocationServiceDeleteShouldReturnErrorWhenStoreDeleteFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	locStore := &mockLocationStore{locations: []domain.Location{loc}, deleteErr: domain.ErrIO}
	svc := NewLocationService(locStore, &mockItemStore{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceDeleteShouldDeleteLocationWhenCallerIsOwnerAndNoConflicts(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	locStore := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(locStore, &mockItemStore{})

	err := svc.Delete(t.Context(), "owner-1", "loc-1")
	require.NoError(t, err)
	require.Empty(t, locStore.locations)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestLocationServiceGetByIDShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.GetByID(ctx, "owner-1", "loc-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceGetByIDShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockLocationStore{getErr: domain.ErrIO}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.GetByID(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceGetByIDShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	store := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.GetByID(t.Context(), "owner-2", "loc-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceGetByIDShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})

	_, err := svc.GetByID(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceGetByIDShouldReturnLocationWhenCallerIsOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.Label = "Closet"

	store := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(store, &mockItemStore{})

	got, err := svc.GetByID(t.Context(), "owner-1", "loc-1")
	require.NoError(t, err)
	require.Equal(t, loc, got)
}

// ── ListByOwner ───────────────────────────────────────────────────────────────

func TestLocationServiceListByOwnerShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListByOwner(ctx, "owner-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceListByOwnerShouldReturnErrorWhenStoreListFails(t *testing.T) {
	store := &mockLocationStore{listErr: domain.ErrIO}
	svc := NewLocationService(store, &mockItemStore{})

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

	store := &mockLocationStore{locations: []domain.Location{loc1, loc2, loc3}}
	svc := NewLocationService(store, &mockItemStore{})

	got, err := svc.ListByOwner(t.Context(), "owner-1")
	require.NoError(t, err)
	require.ElementsMatch(t, []domain.Location{loc1, loc3}, got)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestLocationServiceUpdateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Update(ctx, "owner-1", "loc-1", "New Label")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceUpdateShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockLocationStore{getErr: domain.ErrIO}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Update(t.Context(), "owner-1", "loc-1", "New Label")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceUpdateShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})

	_, err := svc.Update(t.Context(), "owner-1", "loc-1", "New Label")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceUpdateShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-2"

	store := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Update(t.Context(), "owner-1", "loc-1", "New Label")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceUpdateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	store := &mockLocationStore{locations: []domain.Location{loc}, saveErr: domain.ErrIO}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Update(t.Context(), "owner-1", "loc-1", "New Label")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceUpdateShouldReturnUpdatedLocationWhenCallerIsOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.Label = "Old Label"

	store := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(store, &mockItemStore{})

	got, err := svc.Update(t.Context(), "owner-1", "loc-1", "New Label")
	require.NoError(t, err)
	require.Equal(t, "loc-1", got.GetID())
	require.Equal(t, "owner-1", got.OwnerID)
	require.Equal(t, "New Label", got.Label)
	require.Equal(t, "New Label", store.locations[0].Label)
}

// ── Full cycle ────────────────────────────────────────────────────────────────

func TestLocationServiceShouldSucceedWhenRunningFullCreateUpdateListMoveDeleteCycle(t *testing.T) {
	locStore := &mockLocationStore{}
	svc := NewLocationService(locStore, &mockItemStore{})
	ctx := t.Context()

	// Create root location.
	root, err := svc.Create(ctx, "owner-1", "Wardrobe", nil)
	require.NoError(t, err)
	require.NotEmpty(t, root.GetID())

	// Create child location under root.
	child, err := svc.Create(ctx, "owner-1", "Shelf", &[]string{root.GetID()}[0])
	require.NoError(t, err)
	require.NotEmpty(t, child.GetID())
	require.Equal(t, root.GetID(), *child.ParentID)

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
	require.Empty(t, locStore.locations)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestLocationServiceCreateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Create(ctx, "owner-1", "Closet", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceCreateShouldReturnErrorWhenParentStoreGetFails(t *testing.T) {
	parentID := "parent-1"
	store := &mockLocationStore{getErr: domain.ErrIO}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceCreateShouldReturnErrNotFoundWhenParentDoesNotExist(t *testing.T) {
	parentID := "parent-1"
	svc := NewLocationService(&mockLocationStore{}, &mockItemStore{})

	_, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceCreateShouldReturnErrForbiddenWhenParentBelongsToDifferentCaller(t *testing.T) {
	var parent domain.Location
	parent.ID = "parent-1"
	parent.OwnerID = "owner-2"

	parentID := "parent-1"
	store := &mockLocationStore{locations: []domain.Location{parent}}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceCreateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	store := &mockLocationStore{saveErr: domain.ErrIO}
	svc := NewLocationService(store, &mockItemStore{})

	_, err := svc.Create(t.Context(), "owner-1", "Closet", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceCreateShouldCreateRootLocationWhenParentIDIsNil(t *testing.T) {
	store := &mockLocationStore{}
	svc := NewLocationService(store, &mockItemStore{})

	loc, err := svc.Create(t.Context(), "owner-1", "Closet", nil)
	require.NoError(t, err)
	require.NotEmpty(t, loc.GetID())
	require.Equal(t, "owner-1", loc.OwnerID)
	require.Equal(t, "Closet", loc.Label)
	require.Nil(t, loc.ParentID)
	require.False(t, loc.CreatedAt.IsZero())
	require.Len(t, store.locations, 1)
}

func TestLocationServiceCreateShouldCreateChildLocationWhenParentExists(t *testing.T) {
	var parent domain.Location
	parent.ID = "parent-1"
	parent.OwnerID = "owner-1"
	parent.Label = "Closet"

	parentID := "parent-1"
	store := &mockLocationStore{locations: []domain.Location{parent}}
	svc := NewLocationService(store, &mockItemStore{})

	loc, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.NoError(t, err)
	require.NotEmpty(t, loc.GetID())
	require.Equal(t, "owner-1", loc.OwnerID)
	require.Equal(t, "Shelf", loc.Label)
	require.NotNil(t, loc.ParentID)
	require.Equal(t, "parent-1", *loc.ParentID)
	require.False(t, loc.CreatedAt.IsZero())
	require.Len(t, store.locations, 2)
}
