package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeLocationService struct {
	createFn func(ctx context.Context, callerID, label string, parentID *string) (domain.Location, error)
}

func (f *fakeLocationService) Create(ctx context.Context, callerID, label string, parentID *string) (domain.Location, error) {
	if f.createFn != nil {
		return f.createFn(ctx, callerID, label, parentID)
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
