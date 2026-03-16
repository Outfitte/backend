package handler_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/api/middleware"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeItemService struct {
	assignLocationFn func(ctx context.Context, callerID, itemID string, locationID *string) error
}

func (f *fakeItemService) AssignLocation(ctx context.Context, callerID, itemID string, locationID *string) error {
	return f.assignLocationFn(ctx, callerID, itemID, locationID)
}

// --- helpers ---

func patchItemLocation(t *testing.T, h *handler.ItemHandler, itemID, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/items/"+itemID+"/location", strings.NewReader(body))
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.AssignLocation(w, req)
	return w
}

// --- tests ---

// ── AssignLocation ────────────────────────────────────────────────────────────

func TestAssignLocationHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	svc := &fakeItemService{
		assignLocationFn: func(_ context.Context, _, _ string, _ *string) error { return nil },
	}
	h := handler.NewItemHandler(svc, slog.New(slog.DiscardHandler))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/items/item-1/location", strings.NewReader(`{}`))
	req.SetPathValue("id", "item-1")
	w := httptest.NewRecorder()
	h.AssignLocation(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAssignLocationHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	svc := &fakeItemService{
		assignLocationFn: func(_ context.Context, _, _ string, _ *string) error { return nil },
	}
	h := handler.NewItemHandler(svc, slog.New(slog.DiscardHandler))

	w := patchItemLocation(t, h, "item-1", "user-1", "not-json")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAssignLocationHandlerShouldReturn404WhenItemDoesNotExist(t *testing.T) {
	svc := &fakeItemService{
		assignLocationFn: func(_ context.Context, _, _ string, _ *string) error {
			return domain.ErrNotFound
		},
	}
	h := handler.NewItemHandler(svc, slog.New(slog.DiscardHandler))

	w := patchItemLocation(t, h, "item-1", "user-1", `{}`)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAssignLocationHandlerShouldReturn403WhenCallerIsNotItemOwner(t *testing.T) {
	svc := &fakeItemService{
		assignLocationFn: func(_ context.Context, _, _ string, _ *string) error {
			return domain.ErrForbidden
		},
	}
	h := handler.NewItemHandler(svc, slog.New(slog.DiscardHandler))

	w := patchItemLocation(t, h, "item-1", "user-2", `{}`)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestAssignLocationHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		assignLocationFn: func(_ context.Context, _, _ string, _ *string) error {
			return domain.ErrIO
		},
	}
	h := handler.NewItemHandler(svc, slog.New(slog.DiscardHandler))

	w := patchItemLocation(t, h, "item-1", "user-1", `{}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAssignLocationHandlerShouldReturn204WhenLocationAssignedSuccessfully(t *testing.T) {
	locID := "loc-1"
	var gotLocationID *string
	svc := &fakeItemService{
		assignLocationFn: func(_ context.Context, _, _ string, locationID *string) error {
			gotLocationID = locationID
			return nil
		},
	}
	h := handler.NewItemHandler(svc, slog.New(slog.DiscardHandler))

	w := patchItemLocation(t, h, "item-1", "user-1", `{"location_id":"loc-1"}`)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, gotLocationID)
	require.Equal(t, locID, *gotLocationID)
}

func TestAssignLocationHandlerShouldReturn204WhenLocationIDIsNilAndLocationCleared(t *testing.T) {
	var gotLocationID *string
	called := false
	svc := &fakeItemService{
		assignLocationFn: func(_ context.Context, _, _ string, locationID *string) error {
			called = true
			gotLocationID = locationID
			return nil
		},
	}
	h := handler.NewItemHandler(svc, slog.New(slog.DiscardHandler))

	w := patchItemLocation(t, h, "item-1", "user-1", `{"location_id":null}`)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.True(t, called)
	require.Nil(t, gotLocationID)
}
