package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

// ── List ──────────────────────────────────────────────────────────────────────

func TestListHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items", http.NoBody)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		listByOwnerFn: func(_ context.Context, _ string) ([]domain.Item, error) {
			return nil, domain.ErrIO
		},
	}
	h := newItemHandler(svc)

	w := listItems(t, h, "user-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListHandlerShouldReturn200WithItemsWhenListedSuccessfully(t *testing.T) {
	var item1, item2 domain.Item
	item1.ID = "item-1"
	item1.OwnerID = "user-1"
	item1.Name = "Blue Shirt"
	item2.ID = "item-2"
	item2.OwnerID = "user-1"
	item2.Name = "Black Jeans"

	svc := &fakeItemService{
		listByOwnerFn: func(_ context.Context, callerID string) ([]domain.Item, error) {
			require.Equal(t, "user-1", callerID)
			return []domain.Item{item1, item2}, nil
		},
	}
	h := newItemHandler(svc)

	w := listItems(t, h, "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var got []domain.Item
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Len(t, got, 2)
	require.Equal(t, "item-1", got[0].ID)
	require.Equal(t, "item-2", got[1].ID)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestGetByIDHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items/item-1", http.NoBody)
	req.SetPathValue("id", "item-1")
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetByIDHandlerShouldReturn404WhenItemDoesNotExist(t *testing.T) {
	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Item, error) {
			return domain.Item{}, domain.ErrNotFound
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-1", "user-1")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetByIDHandlerShouldReturn403WhenCallerIsNotItemOwner(t *testing.T) {
	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Item, error) {
			return domain.Item{}, domain.ErrForbidden
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-1", "user-2")

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetByIDHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Item, error) {
			return domain.Item{}, domain.ErrIO
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-1", "user-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetByIDHandlerShouldReturn200WithItemWhenFoundSuccessfully(t *testing.T) {
	var item domain.Item
	item.ID = "item-42"
	item.OwnerID = "user-1"
	item.Name = "Blue Shirt"

	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, callerID, itemID string) (domain.Item, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "item-42", itemID)
			return item, nil
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-42", "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var got domain.Item
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "item-42", got.ID)
	require.Equal(t, "Blue Shirt", got.Name)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestUpdateHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/items/item-1", strings.NewReader(`{}`))
	req.SetPathValue("id", "item-1")
	w := httptest.NewRecorder()
	h.Update(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	w := patchItem(t, h, "item-1", "user-1", "not-json")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateHandlerShouldReturn404WhenItemDoesNotExist(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, _ service.UpdateItemInput) (domain.Item, error) {
			return domain.Item{}, domain.ErrNotFound
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{"name":"shirt"}`)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateHandlerShouldReturn403WhenCallerIsNotItemOwner(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, _ service.UpdateItemInput) (domain.Item, error) {
			return domain.Item{}, domain.ErrForbidden
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-2", `{"name":"shirt"}`)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, _ service.UpdateItemInput) (domain.Item, error) {
			return domain.Item{}, domain.ErrIO
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{"name":"shirt"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateHandlerShouldReturn200WithUpdatedItemWhenSuccessful(t *testing.T) {
	price := "49.99"
	var updated domain.Item
	updated.ID = "item-42"
	updated.OwnerID = "user-1"
	updated.Name = "Red Jacket"

	svc := &fakeItemService{
		updateFn: func(_ context.Context, callerID, itemID string, input service.UpdateItemInput) (domain.Item, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "item-42", itemID)
			require.Equal(t, "Red Jacket", input.Name)
			require.Equal(t, &price, input.PurchasePrice)
			return updated, nil
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-42", "user-1", `{"name":"Red Jacket","purchase_price":"49.99"}`)

	require.Equal(t, http.StatusOK, w.Code)
	var got domain.Item
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "item-42", got.ID)
	require.Equal(t, "Red Jacket", got.Name)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestDeleteHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/items/item-1", http.NoBody)
	req.SetPathValue("id", "item-1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteHandlerShouldReturn404WhenItemDoesNotExist(t *testing.T) {
	svc := &fakeItemService{
		deleteFn: func(_ context.Context, _, _ string) error {
			return domain.ErrNotFound
		},
	}
	h := newItemHandler(svc)

	w := deleteItem(t, h, "item-1", "user-1")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteHandlerShouldReturn403WhenCallerIsNotItemOwner(t *testing.T) {
	svc := &fakeItemService{
		deleteFn: func(_ context.Context, _, _ string) error {
			return domain.ErrForbidden
		},
	}
	h := newItemHandler(svc)

	w := deleteItem(t, h, "item-1", "user-2")

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		deleteFn: func(_ context.Context, _, _ string) error {
			return domain.ErrIO
		},
	}
	h := newItemHandler(svc)

	w := deleteItem(t, h, "item-1", "user-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteHandlerShouldReturn204WhenItemDeletedSuccessfully(t *testing.T) {
	var gotCallerID, gotItemID string
	svc := &fakeItemService{
		deleteFn: func(_ context.Context, callerID, itemID string) error {
			gotCallerID = callerID
			gotItemID = itemID
			return nil
		},
	}
	h := newItemHandler(svc)

	w := deleteItem(t, h, "item-42", "user-1")

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "user-1", gotCallerID)
	require.Equal(t, "item-42", gotItemID)
}

// ── UploadPhoto ───────────────────────────────────────────────────────────────

func TestUploadPhotoHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.Close())
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items/item-1/photos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.SetPathValue("id", "item-1")
	w := httptest.NewRecorder()
	h.UploadPhoto(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUploadPhotoHandlerShouldReturn400WhenPhotoFieldIsMissing(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.Close())
	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/items/item-1/photos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.SetPathValue("id", "item-1")
	w := httptest.NewRecorder()
	h.UploadPhoto(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUploadPhotoHandlerShouldReturn404WhenItemDoesNotExist(t *testing.T) {
	svc := &fakeItemService{
		uploadPhotoFn: func(_ context.Context, _, _ string, _ io.Reader, _ string) error {
			return domain.ErrNotFound
		},
	}
	h := newItemHandler(svc)

	w := uploadPhoto(t, h, "item-1", "user-1", "photo.jpg", "data")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUploadPhotoHandlerShouldReturn403WhenCallerIsNotItemOwner(t *testing.T) {
	svc := &fakeItemService{
		uploadPhotoFn: func(_ context.Context, _, _ string, _ io.Reader, _ string) error {
			return domain.ErrForbidden
		},
	}
	h := newItemHandler(svc)

	w := uploadPhoto(t, h, "item-1", "user-2", "photo.jpg", "data")

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUploadPhotoHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		uploadPhotoFn: func(_ context.Context, _, _ string, _ io.Reader, _ string) error {
			return domain.ErrIO
		},
	}
	h := newItemHandler(svc)

	w := uploadPhoto(t, h, "item-1", "user-1", "photo.jpg", "data")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUploadPhotoHandlerShouldReturn201WhenPhotoUploadedSuccessfully(t *testing.T) {
	var gotCallerID, gotItemID, gotFilename string
	svc := &fakeItemService{
		uploadPhotoFn: func(_ context.Context, callerID, itemID string, _ io.Reader, filename string) error {
			gotCallerID = callerID
			gotItemID = itemID
			gotFilename = filename
			return nil
		},
	}
	h := newItemHandler(svc)

	w := uploadPhoto(t, h, "item-42", "user-1", "photo.jpg", "image data")

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "user-1", gotCallerID)
	require.Equal(t, "item-42", gotItemID)
	require.Equal(t, "photo.jpg", gotFilename)
}

// ── DeletePhoto ───────────────────────────────────────────────────────────────

func TestDeletePhotoHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/items/item-1/photos/key-1", http.NoBody)
	req.SetPathValue("id", "item-1")
	req.SetPathValue("key", "key-1")
	w := httptest.NewRecorder()
	h.DeletePhoto(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeletePhotoHandlerShouldReturn404WhenPhotoDoesNotExist(t *testing.T) {
	svc := &fakeItemService{
		deletePhotoFn: func(_ context.Context, _, _, _ string) error {
			return domain.ErrNotFound
		},
	}
	h := newItemHandler(svc)

	w := deletePhoto(t, h, "item-1", "user-1", "key-1")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeletePhotoHandlerShouldReturn403WhenCallerIsNotItemOwner(t *testing.T) {
	svc := &fakeItemService{
		deletePhotoFn: func(_ context.Context, _, _, _ string) error {
			return domain.ErrForbidden
		},
	}
	h := newItemHandler(svc)

	w := deletePhoto(t, h, "item-1", "user-2", "key-1")

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeletePhotoHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		deletePhotoFn: func(_ context.Context, _, _, _ string) error {
			return domain.ErrIO
		},
	}
	h := newItemHandler(svc)

	w := deletePhoto(t, h, "item-1", "user-1", "key-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeletePhotoHandlerShouldReturn204WhenPhotoDeletedSuccessfully(t *testing.T) {
	var gotCallerID, gotItemID, gotPhotoKey string
	svc := &fakeItemService{
		deletePhotoFn: func(_ context.Context, callerID, itemID, photoKey string) error {
			gotCallerID = callerID
			gotItemID = itemID
			gotPhotoKey = photoKey
			return nil
		},
	}
	h := newItemHandler(svc)

	w := deletePhoto(t, h, "item-42", "user-1", "key-99")

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "user-1", gotCallerID)
	require.Equal(t, "item-42", gotItemID)
	require.Equal(t, "key-99", gotPhotoKey)
}

// ── Full lifecycle integration ────────────────────────────────────────────────

// statefulFakeItemService is an in-memory item store used in the lifecycle integration test.
type statefulFakeItemService struct {
	mu     sync.Mutex
	items  map[string]domain.Item
	nextID int
}

func newStatefulFakeItemService() *statefulFakeItemService {
	return &statefulFakeItemService{items: make(map[string]domain.Item)}
}

func (s *statefulFakeItemService) Create(ctx context.Context, callerID string, input service.CreateItemInput) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	var item domain.Item
	item.ID = fmt.Sprintf("item-%d", s.nextID)
	item.OwnerID = callerID
	item.Name = input.Name
	item.Brand = input.Brand
	item.Color = input.Color
	item.Metadata = input.Metadata
	s.items[item.ID] = item
	return item, nil
}

func (s *statefulFakeItemService) ListByOwner(ctx context.Context, callerID string) ([]domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []domain.Item
	for _, item := range s.items {
		if item.OwnerID == callerID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *statefulFakeItemService) GetByID(ctx context.Context, callerID, itemID string) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[itemID]
	if !ok {
		return domain.Item{}, domain.ErrNotFound
	}
	if item.OwnerID != callerID {
		return domain.Item{}, domain.ErrForbidden
	}
	return item, nil
}

func (s *statefulFakeItemService) Update(ctx context.Context, callerID, itemID string, input service.UpdateItemInput) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[itemID]
	if !ok {
		return domain.Item{}, domain.ErrNotFound
	}
	if item.OwnerID != callerID {
		return domain.Item{}, domain.ErrForbidden
	}
	item.Name = input.Name
	item.Brand = input.Brand
	item.CategoryID = input.CategoryID
	item.Color = input.Color
	item.Metadata = input.Metadata
	item.PhotoKeys = input.PhotoKeys
	item.LocationID = input.LocationID
	item.PurchasePrice = input.PurchasePrice
	item.PurchaseDate = input.PurchaseDate
	s.items[itemID] = item
	return item, nil
}

func (s *statefulFakeItemService) Delete(ctx context.Context, callerID, itemID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[itemID]
	if !ok {
		return domain.ErrNotFound
	}
	if item.OwnerID != callerID {
		return domain.ErrForbidden
	}
	delete(s.items, itemID)
	return nil
}

func (s *statefulFakeItemService) UploadPhoto(ctx context.Context, _, _ string, _ io.Reader, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *statefulFakeItemService) DeletePhoto(ctx context.Context, _, _, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *statefulFakeItemService) AssignLocation(ctx context.Context, _, _ string, _ *string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func TestItemHandlerShouldHandleFullItemLifecycle(t *testing.T) {
	const callerID = "user-1"
	svc := newStatefulFakeItemService()
	h := handler.NewItemHandler(svc, slog.New(slog.DiscardHandler))

	withUser := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.WithUserID(r.Context(), callerID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	mux := http.NewServeMux()
	mux.Handle("POST /items", withUser(http.HandlerFunc(h.Create)))
	mux.Handle("GET /items", withUser(http.HandlerFunc(h.List)))
	mux.Handle("GET /items/{id}", withUser(http.HandlerFunc(h.GetByID)))
	mux.Handle("PATCH /items/{id}", withUser(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /items/{id}", withUser(http.HandlerFunc(h.Delete)))

	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := ts.Client()

	// 1. POST /items — create an item, assert 201 and returned item fields.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/items", strings.NewReader(`{"name":"Blue Shirt","brand":"Acme","color":"Blue"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created domain.Item
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	require.NotEmpty(t, created.ID)
	require.Equal(t, callerID, created.OwnerID)
	require.Equal(t, "Blue Shirt", created.Name)
	require.NotNil(t, created.Brand)
	require.Equal(t, "Acme", *created.Brand)
	require.NotNil(t, created.Color)
	require.Equal(t, "Blue", *created.Color)

	itemID := created.ID

	// 2. GET /items — list all items, assert the created item is present.
	req, err = http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/items", http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var listed []domain.Item
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&listed))
	require.Len(t, listed, 1)
	require.Equal(t, itemID, listed[0].ID)
	require.Equal(t, callerID, listed[0].OwnerID)

	// 3. GET /items/{id} — fetch the item by ID, assert fields match.
	req, err = http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/items/"+itemID, http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var fetched domain.Item
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&fetched))
	require.Equal(t, itemID, fetched.ID)
	require.Equal(t, callerID, fetched.OwnerID)
	require.Equal(t, "Blue Shirt", fetched.Name)
	require.NotNil(t, fetched.Color)
	require.Equal(t, "Blue", *fetched.Color)

	// 4. PATCH /items/{id} — update the item, assert 200 and updated fields.
	req, err = http.NewRequestWithContext(t.Context(), http.MethodPatch, ts.URL+"/items/"+itemID, strings.NewReader(`{"name":"Red Jacket","brand":"Acme","color":"Red"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var updated domain.Item
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&updated))
	require.Equal(t, itemID, updated.ID)
	require.Equal(t, "Red Jacket", updated.Name)
	require.NotNil(t, updated.Color)
	require.Equal(t, "Red", *updated.Color)

	// 5. DELETE /items/{id} — delete the item, assert 204.
	req, err = http.NewRequestWithContext(t.Context(), http.MethodDelete, ts.URL+"/items/"+itemID, http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// 6. GET /items/{id} — confirm item is gone, assert 404.
	req, err = http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/items/"+itemID, http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
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
