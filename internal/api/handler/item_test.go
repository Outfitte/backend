package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/api/middleware"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/service"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeItemService struct {
	assignLocationFn func(ctx context.Context, callerID, itemID string, locationID *string) error
	createFn         func(ctx context.Context, callerID string, input service.CreateItemInput) (domain.Item, error)
	listByOwnerFn    func(ctx context.Context, callerID string) ([]domain.Item, error)
	getByIDFn        func(ctx context.Context, callerID, itemID string) (domain.Item, error)
	updateFn         func(ctx context.Context, callerID, itemID string, input service.UpdateItemInput) (domain.Item, error)
	deleteFn         func(ctx context.Context, callerID, itemID string) error
	uploadPhotoFn    func(ctx context.Context, callerID, itemID string, r io.Reader, filename string) error
	deletePhotoFn    func(ctx context.Context, callerID, itemID, photoKey string) error
}

func (f *fakeItemService) AssignLocation(ctx context.Context, callerID, itemID string, locationID *string) error {
	return f.assignLocationFn(ctx, callerID, itemID, locationID)
}

func (f *fakeItemService) Create(ctx context.Context, callerID string, input service.CreateItemInput) (domain.Item, error) {
	if f.createFn != nil {
		return f.createFn(ctx, callerID, input)
	}
	return domain.Item{}, nil
}

func (f *fakeItemService) ListByOwner(ctx context.Context, callerID string) ([]domain.Item, error) {
	if f.listByOwnerFn != nil {
		return f.listByOwnerFn(ctx, callerID)
	}
	return nil, nil
}

func (f *fakeItemService) GetByID(ctx context.Context, callerID, itemID string) (domain.Item, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, callerID, itemID)
	}
	return domain.Item{}, nil
}

func (f *fakeItemService) Update(ctx context.Context, callerID, itemID string, input service.UpdateItemInput) (domain.Item, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, callerID, itemID, input)
	}
	return domain.Item{}, nil
}

func (f *fakeItemService) Delete(ctx context.Context, callerID, itemID string) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, callerID, itemID)
	}
	return nil
}

func (f *fakeItemService) UploadPhoto(ctx context.Context, callerID, itemID string, r io.Reader, filename string) error {
	if f.uploadPhotoFn != nil {
		return f.uploadPhotoFn(ctx, callerID, itemID, r, filename)
	}
	return nil
}

func (f *fakeItemService) DeletePhoto(ctx context.Context, callerID, itemID, photoKey string) error {
	if f.deletePhotoFn != nil {
		return f.deletePhotoFn(ctx, callerID, itemID, photoKey)
	}
	return nil
}

// --- helpers ---

func newItemHandler(svc *fakeItemService) *handler.ItemHandler {
	return handler.NewItemHandler(svc, slog.New(slog.DiscardHandler))
}

func ctxWithUser(t *testing.T, callerID string) context.Context {
	t.Helper()
	return middleware.WithUserID(t.Context(), callerID)
}

func patchItemLocation(t *testing.T, h *handler.ItemHandler, itemID, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/items/"+itemID+"/location", strings.NewReader(body))
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.AssignLocation(w, req)
	return w
}

func postItem(t *testing.T, h *handler.ItemHandler, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/items", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, req)
	return w
}

func listItems(t *testing.T, h *handler.ItemHandler, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/items", http.NoBody)
	w := httptest.NewRecorder()
	h.List(w, req)
	return w
}

func getItem(t *testing.T, h *handler.ItemHandler, itemID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/items/"+itemID, http.NoBody)
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.GetByID(w, req)
	return w
}

func patchItem(t *testing.T, h *handler.ItemHandler, itemID, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/items/"+itemID, strings.NewReader(body))
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.Update(w, req)
	return w
}

func deleteItem(t *testing.T, h *handler.ItemHandler, itemID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/items/"+itemID, http.NoBody)
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.Delete(w, req)
	return w
}

func uploadPhoto(t *testing.T, h *handler.ItemHandler, itemID, callerID, filename, content string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("photo", filename)
	require.NoError(t, err)
	_, err = io.WriteString(fw, content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/items/"+itemID+"/photos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.UploadPhoto(w, req)
	return w
}

func deletePhoto(t *testing.T, h *handler.ItemHandler, itemID, callerID, photoKey string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/items/"+itemID+"/photos/"+photoKey, http.NoBody)
	req.SetPathValue("id", itemID)
	req.SetPathValue("key", photoKey)
	w := httptest.NewRecorder()
	h.DeletePhoto(w, req)
	return w
}

// --- tests ---

// ── Create ────────────────────────────────────────────────────────────────────

func TestCreateHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.Create(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	w := postItem(t, h, "user-1", "not-json")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		createFn: func(_ context.Context, _ string, _ service.CreateItemInput) (domain.Item, error) {
			return domain.Item{}, domain.ErrIO
		},
	}
	h := newItemHandler(svc)

	w := postItem(t, h, "user-1", `{"name":"shirt"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateHandlerShouldReturn201WithItemWhenCreatedSuccessfully(t *testing.T) {
	var created domain.Item
	created.ID = "item-42"
	created.OwnerID = "user-1"
	created.Name = "Blue Shirt"

	svc := &fakeItemService{
		createFn: func(_ context.Context, callerID string, input service.CreateItemInput) (domain.Item, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "Blue Shirt", input.Name)
			return created, nil
		},
	}
	h := newItemHandler(svc)

	w := postItem(t, h, "user-1", `{"name":"Blue Shirt"}`)

	require.Equal(t, http.StatusCreated, w.Code)
	var got domain.Item
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "item-42", got.ID)
	require.Equal(t, "Blue Shirt", got.Name)
}

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
