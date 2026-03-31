package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/outfitte/backend/internal/api/handler"
	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeLocationService struct {
	createFn      func(ctx context.Context, callerID, label string, parentID *string) (domain.Location, error)
	listByOwnerFn func(ctx context.Context, callerID string) ([]domain.Location, error)
	getByIDFn     func(ctx context.Context, callerID, locationID string) (domain.Location, error)
	updateFn      func(ctx context.Context, callerID, locationID, label string) (domain.Location, error)
	deleteFn      func(ctx context.Context, callerID, locationID string) error
	moveFn        func(ctx context.Context, callerID, locationID string, newParentID *string) (domain.Location, error)
}

func (f *fakeLocationService) Create(ctx context.Context, callerID, label string, parentID *string) (domain.Location, error) {
	if f.createFn != nil {
		return f.createFn(ctx, callerID, label, parentID)
	}
	return domain.Location{}, nil
}

func (f *fakeLocationService) ListByOwner(ctx context.Context, callerID string) ([]domain.Location, error) {
	if f.listByOwnerFn != nil {
		return f.listByOwnerFn(ctx, callerID)
	}
	return nil, nil
}

func (f *fakeLocationService) GetByID(ctx context.Context, callerID, locationID string) (domain.Location, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, callerID, locationID)
	}
	return domain.Location{}, nil
}

func (f *fakeLocationService) Update(ctx context.Context, callerID, locationID, label string) (domain.Location, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, callerID, locationID, label)
	}
	return domain.Location{}, nil
}

func (f *fakeLocationService) Delete(ctx context.Context, callerID, locationID string) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, callerID, locationID)
	}
	return nil
}

func (f *fakeLocationService) Move(ctx context.Context, callerID, locationID string, newParentID *string) (domain.Location, error) {
	if f.moveFn != nil {
		return f.moveFn(ctx, callerID, locationID, newParentID)
	}
	return domain.Location{}, nil
}

// --- helpers ---

func newLocationHandler(svc *fakeLocationService) *handler.LocationHandler {
	return handler.NewLocationHandler(svc, slog.New(slog.DiscardHandler))
}

func postLocation(t *testing.T, h *handler.LocationHandler, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/locations", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, req)
	return w
}

// --- tests ---

// ── Create ────────────────────────────────────────────────────────────────────

func TestCreateLocationHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/locations", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.Create(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateLocationHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	w := postLocation(t, h, "user-1", "not-json")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateLocationHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeLocationService{
		createFn: func(_ context.Context, _, _ string, _ *string) (domain.Location, error) {
			return domain.Location{}, domain.ErrIO
		},
	}
	h := newLocationHandler(svc)

	w := postLocation(t, h, "user-1", `{"label":"Wardrobe"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateLocationHandlerShouldReturn201WithLocationWhenCreatedSuccessfully(t *testing.T) {
	parentID := "loc-parent"
	var created domain.Location
	created.ID = "loc-42"
	created.OwnerID = "user-1"
	created.Label = "Wardrobe"

	svc := &fakeLocationService{
		createFn: func(_ context.Context, callerID, label string, pID *string) (domain.Location, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "Wardrobe", label)
			require.NotNil(t, pID)
			require.Equal(t, parentID, *pID)
			return created, nil
		},
	}
	h := newLocationHandler(svc)

	w := postLocation(t, h, "user-1", `{"label":"Wardrobe","parent_id":"loc-parent"}`)

	require.Equal(t, http.StatusCreated, w.Code)
	var got domain.Location
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "loc-42", got.ID)
	require.Equal(t, "Wardrobe", got.Label)
}

func TestCreateLocationHandlerShouldReturn404WhenParentLocationNotFound(t *testing.T) {
	svc := &fakeLocationService{
		createFn: func(_ context.Context, _, _ string, _ *string) (domain.Location, error) {
			return domain.Location{}, domain.ErrNotFound
		},
	}
	h := newLocationHandler(svc)

	w := postLocation(t, h, "user-1", `{"label":"Shelf","parent_id":"ghost-id"}`)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateLocationHandlerShouldReturn403WhenParentLocationForbidden(t *testing.T) {
	svc := &fakeLocationService{
		createFn: func(_ context.Context, _, _ string, _ *string) (domain.Location, error) {
			return domain.Location{}, domain.ErrForbidden
		},
	}
	h := newLocationHandler(svc)

	w := postLocation(t, h, "user-1", `{"label":"Shelf","parent_id":"other-user-loc"}`)

	require.Equal(t, http.StatusForbidden, w.Code)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestGetLocationByIDHandlerShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/locations/loc-1", nil)
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetLocationByIDHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/locations/loc-1", nil)
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetLocationByIDHandlerShouldReturn404WhenLocationNotFound(t *testing.T) {
	svc := &fakeLocationService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Location, error) {
			return domain.Location{}, domain.ErrNotFound
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/locations/loc-ghost", nil)
	req.SetPathValue("id", "loc-ghost")
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetLocationByIDHandlerShouldReturn403WhenLocationForbidden(t *testing.T) {
	svc := &fakeLocationService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Location, error) {
			return domain.Location{}, domain.ErrForbidden
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/locations/loc-other", nil)
	req.SetPathValue("id", "loc-other")
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetLocationByIDHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeLocationService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Location, error) {
			return domain.Location{}, domain.ErrIO
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/locations/loc-1", nil)
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetLocationByIDHandlerShouldReturn200WithLocationWhenSuccessful(t *testing.T) {
	var loc domain.Location
	loc.ID = "loc-42"
	loc.OwnerID = "user-1"
	loc.Label = "Wardrobe"

	svc := &fakeLocationService{
		getByIDFn: func(_ context.Context, callerID, locationID string) (domain.Location, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "loc-42", locationID)
			return loc, nil
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/locations/loc-42", nil)
	req.SetPathValue("id", "loc-42")
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got domain.Location
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "loc-42", got.ID)
	require.Equal(t, "Wardrobe", got.Label)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestUpdateLocationHandlerShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-1", strings.NewReader(`{"label":"New"}`))
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Update(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateLocationHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/locations/loc-1", strings.NewReader(`{"label":"New"}`))
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Update(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateLocationHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-1", strings.NewReader("not-json"))
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Update(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateLocationHandlerShouldReturn404WhenLocationNotFound(t *testing.T) {
	svc := &fakeLocationService{
		updateFn: func(_ context.Context, _, _, _ string) (domain.Location, error) {
			return domain.Location{}, domain.ErrNotFound
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-ghost", strings.NewReader(`{"label":"New"}`))
	req.SetPathValue("id", "loc-ghost")
	w := httptest.NewRecorder()
	h.Update(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateLocationHandlerShouldReturn403WhenLocationForbidden(t *testing.T) {
	svc := &fakeLocationService{
		updateFn: func(_ context.Context, _, _, _ string) (domain.Location, error) {
			return domain.Location{}, domain.ErrForbidden
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-other", strings.NewReader(`{"label":"New"}`))
	req.SetPathValue("id", "loc-other")
	w := httptest.NewRecorder()
	h.Update(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateLocationHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeLocationService{
		updateFn: func(_ context.Context, _, _, _ string) (domain.Location, error) {
			return domain.Location{}, domain.ErrIO
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-1", strings.NewReader(`{"label":"New"}`))
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Update(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateLocationHandlerShouldReturn200WithUpdatedLocationWhenSuccessful(t *testing.T) {
	var updated domain.Location
	updated.ID = "loc-42"
	updated.OwnerID = "user-1"
	updated.Label = "New Label"

	svc := &fakeLocationService{
		updateFn: func(_ context.Context, callerID, locationID, label string) (domain.Location, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "loc-42", locationID)
			require.Equal(t, "New Label", label)
			return updated, nil
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-42", strings.NewReader(`{"label":"New Label"}`))
	req.SetPathValue("id", "loc-42")
	w := httptest.NewRecorder()
	h.Update(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got domain.Location
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "loc-42", got.ID)
	require.Equal(t, "New Label", got.Label)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestListLocationsHandlerShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/locations", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListLocationsHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/locations", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListLocationsHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeLocationService{
		listByOwnerFn: func(_ context.Context, _ string) ([]domain.Location, error) {
			return nil, domain.ErrIO
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/locations", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestDeleteLocationHandlerShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/locations/loc-1", nil)
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteLocationHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/locations/loc-1", nil)
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteLocationHandlerShouldReturn404WhenLocationNotFound(t *testing.T) {
	svc := &fakeLocationService{
		deleteFn: func(_ context.Context, _, _ string) error {
			return domain.ErrNotFound
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/locations/loc-ghost", nil)
	req.SetPathValue("id", "loc-ghost")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteLocationHandlerShouldReturn403WhenCallerDoesNotOwnLocation(t *testing.T) {
	svc := &fakeLocationService{
		deleteFn: func(_ context.Context, _, _ string) error {
			return domain.ErrForbidden
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/locations/loc-other", nil)
	req.SetPathValue("id", "loc-other")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteLocationHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeLocationService{
		deleteFn: func(_ context.Context, _, _ string) error {
			return domain.ErrIO
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/locations/loc-1", nil)
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteLocationHandlerShouldReturn204WhenDeletedSuccessfully(t *testing.T) {
	svc := &fakeLocationService{
		deleteFn: func(_ context.Context, callerID, locationID string) error {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "loc-42", locationID)
			return nil
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/locations/loc-42", nil)
	req.SetPathValue("id", "loc-42")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Empty(t, w.Body.String())
}

func TestListLocationsHandlerShouldReturn200WithLocationsWhenSuccessful(t *testing.T) {
	parentID := "loc-root"
	var loc1, loc2 domain.Location
	loc1.ID = "loc-1"
	loc1.OwnerID = "user-1"
	loc1.Label = "Wardrobe"
	loc2.ID = "loc-2"
	loc2.OwnerID = "user-1"
	loc2.Label = "Shelf"
	loc2.ParentID = &parentID

	svc := &fakeLocationService{
		listByOwnerFn: func(_ context.Context, callerID string) ([]domain.Location, error) {
			require.Equal(t, "user-1", callerID)
			return []domain.Location{loc1, loc2}, nil
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/locations", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got []domain.Location
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Len(t, got, 2)
	require.Equal(t, "loc-1", got[0].ID)
	require.Equal(t, "loc-2", got[1].ID)
	require.Equal(t, "Shelf", got[1].Label)
	require.NotNil(t, got[1].ParentID)
	require.Equal(t, parentID, *got[1].ParentID)
}

// ── Move ──────────────────────────────────────────────────────────────────────

func TestMoveLocationHandlerShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-1/move", strings.NewReader(`{}`))
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Move(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestMoveLocationHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/locations/loc-1/move", strings.NewReader(`{}`))
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Move(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMoveLocationHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	h := newLocationHandler(&fakeLocationService{})

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-1/move", strings.NewReader("not-json"))
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Move(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMoveLocationHandlerShouldReturn404WhenLocationNotFound(t *testing.T) {
	svc := &fakeLocationService{
		moveFn: func(_ context.Context, _, _ string, _ *string) (domain.Location, error) {
			return domain.Location{}, domain.ErrNotFound
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-ghost/move", strings.NewReader(`{}`))
	req.SetPathValue("id", "loc-ghost")
	w := httptest.NewRecorder()
	h.Move(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMoveLocationHandlerShouldReturn403WhenLocationForbidden(t *testing.T) {
	svc := &fakeLocationService{
		moveFn: func(_ context.Context, _, _ string, _ *string) (domain.Location, error) {
			return domain.Location{}, domain.ErrForbidden
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-other/move", strings.NewReader(`{}`))
	req.SetPathValue("id", "loc-other")
	w := httptest.NewRecorder()
	h.Move(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestMoveLocationHandlerShouldReturn409WhenConflict(t *testing.T) {
	svc := &fakeLocationService{
		moveFn: func(_ context.Context, _, _ string, _ *string) (domain.Location, error) {
			return domain.Location{}, domain.ErrConflict
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-1/move", strings.NewReader(`{"parent_id":"loc-1"}`))
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Move(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestMoveLocationHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeLocationService{
		moveFn: func(_ context.Context, _, _ string, _ *string) (domain.Location, error) {
			return domain.Location{}, domain.ErrIO
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-1/move", strings.NewReader(`{}`))
	req.SetPathValue("id", "loc-1")
	w := httptest.NewRecorder()
	h.Move(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMoveLocationHandlerShouldReturn200WithLocationWhenMovedToRoot(t *testing.T) {
	var moved domain.Location
	moved.ID = "loc-42"
	moved.OwnerID = "user-1"
	moved.Label = "Wardrobe"

	svc := &fakeLocationService{
		moveFn: func(_ context.Context, callerID, locationID string, newParentID *string) (domain.Location, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "loc-42", locationID)
			require.Nil(t, newParentID)
			return moved, nil
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-42/move", strings.NewReader(`{"parent_id":null}`))
	req.SetPathValue("id", "loc-42")
	w := httptest.NewRecorder()
	h.Move(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got domain.Location
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "loc-42", got.ID)
	require.Nil(t, got.ParentID)
}

func TestMoveLocationHandlerShouldReturn200WithLocationWhenReparented(t *testing.T) {
	newParent := "loc-parent"
	var moved domain.Location
	moved.ID = "loc-42"
	moved.OwnerID = "user-1"
	moved.Label = "Wardrobe"
	moved.ParentID = &newParent

	svc := &fakeLocationService{
		moveFn: func(_ context.Context, callerID, locationID string, newParentID *string) (domain.Location, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "loc-42", locationID)
			require.NotNil(t, newParentID)
			require.Equal(t, newParent, *newParentID)
			return moved, nil
		},
	}
	h := newLocationHandler(svc)

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/locations/loc-42/move", strings.NewReader(`{"parent_id":"loc-parent"}`))
	req.SetPathValue("id", "loc-42")
	w := httptest.NewRecorder()
	h.Move(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got domain.Location
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "loc-42", got.ID)
	require.NotNil(t, got.ParentID)
	require.Equal(t, newParent, *got.ParentID)
}
