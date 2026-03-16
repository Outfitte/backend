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

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestLocationServiceGetByIDShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.GetByID(ctx, "owner-1", "loc-1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceGetByIDShouldReturnErrorWhenStoreGetFails(t *testing.T) {
	store := &mockLocationStore{getErr: domain.ErrIO}
	svc := NewLocationService(store)

	_, err := svc.GetByID(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceGetByIDShouldReturnErrForbiddenWhenCallerIsNotOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"

	store := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(store)

	_, err := svc.GetByID(t.Context(), "owner-2", "loc-1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceGetByIDShouldReturnErrNotFoundWhenLocationDoesNotExist(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{})

	_, err := svc.GetByID(t.Context(), "owner-1", "loc-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceGetByIDShouldReturnLocationWhenCallerIsOwner(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-1"
	loc.OwnerID = "owner-1"
	loc.Label = "Closet"

	store := &mockLocationStore{locations: []domain.Location{loc}}
	svc := NewLocationService(store)

	got, err := svc.GetByID(t.Context(), "owner-1", "loc-1")
	require.NoError(t, err)
	require.Equal(t, loc, got)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestLocationServiceCreateShouldReturnErrorWhenContextIsCancelled(t *testing.T) {
	svc := NewLocationService(&mockLocationStore{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.Create(ctx, "owner-1", "Closet", nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestLocationServiceCreateShouldReturnErrorWhenParentStoreGetFails(t *testing.T) {
	parentID := "parent-1"
	store := &mockLocationStore{getErr: domain.ErrIO}
	svc := NewLocationService(store)

	_, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceCreateShouldReturnErrNotFoundWhenParentDoesNotExist(t *testing.T) {
	parentID := "parent-1"
	svc := NewLocationService(&mockLocationStore{})

	_, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLocationServiceCreateShouldReturnErrForbiddenWhenParentBelongsToDifferentCaller(t *testing.T) {
	var parent domain.Location
	parent.ID = "parent-1"
	parent.OwnerID = "owner-2"

	parentID := "parent-1"
	store := &mockLocationStore{locations: []domain.Location{parent}}
	svc := NewLocationService(store)

	_, err := svc.Create(t.Context(), "owner-1", "Shelf", &parentID)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestLocationServiceCreateShouldReturnErrorWhenStoreSaveFails(t *testing.T) {
	store := &mockLocationStore{saveErr: domain.ErrIO}
	svc := NewLocationService(store)

	_, err := svc.Create(t.Context(), "owner-1", "Closet", nil)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestLocationServiceCreateShouldCreateRootLocationWhenParentIDIsNil(t *testing.T) {
	store := &mockLocationStore{}
	svc := NewLocationService(store)

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
	svc := NewLocationService(store)

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
