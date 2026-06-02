package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/backend/internal/api/handler"
	"github.com/outfitte/backend/internal/api/middleware"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
	"github.com/outfitte/backend/internal/service"
)

// --- fakes ---

type fakeItemService struct {
	assignLocationFn func(ctx context.Context, callerID, itemID string, locationID *string) error
	createFn         func(ctx context.Context, callerID string, input service.CreateItemInput) (domain.Item, error)
	listByOwnerFn    func(ctx context.Context, callerID string, filter ports.ItemListFilter) ([]domain.Item, error)
	getByIDFn        func(ctx context.Context, callerID, itemID string) (domain.Item, error)
	updateFn         func(ctx context.Context, callerID, itemID string, input service.UpdateItemInput) (domain.Item, error)
	deleteFn         func(ctx context.Context, callerID, itemID string) error
	uploadPhotoFn    func(ctx context.Context, callerID, itemID string, r io.Reader, filename string) error
	deletePhotoFn    func(ctx context.Context, callerID, itemID, photoKey string) error
	archiveFn        func(ctx context.Context, callerID, itemID string) error
	unarchiveFn      func(ctx context.Context, callerID, itemID string) error
	disposeFn        func(ctx context.Context, callerID, itemID string, reason domain.DisposalReason) error
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

func (f *fakeItemService) ListByOwner(ctx context.Context, callerID string, filter ports.ItemListFilter) ([]domain.Item, error) {
	if f.listByOwnerFn != nil {
		return f.listByOwnerFn(ctx, callerID, filter)
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

func (f *fakeItemService) Archive(ctx context.Context, callerID, itemID string) error {
	if f.archiveFn != nil {
		return f.archiveFn(ctx, callerID, itemID)
	}
	return nil
}

func (f *fakeItemService) Unarchive(ctx context.Context, callerID, itemID string) error {
	if f.unarchiveFn != nil {
		return f.unarchiveFn(ctx, callerID, itemID)
	}
	return nil
}

func (f *fakeItemService) Dispose(ctx context.Context, callerID, itemID string, reason domain.DisposalReason) error {
	if f.disposeFn != nil {
		return f.disposeFn(ctx, callerID, itemID, reason)
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

func TestCreateHandlerShouldReturn422WhenServiceReturnsValidationError(t *testing.T) {
	svc := &fakeItemService{
		createFn: func(_ context.Context, _ string, _ service.CreateItemInput) (domain.Item, error) {
			return domain.Item{}, domain.ErrValidation
		},
	}
	h := newItemHandler(svc)

	w := postItem(t, h, "user-1", `{"name":"shirt","metadata":{"bad key":"v"}}`)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
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
	var got testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "item-42", got.ID)
	require.Equal(t, "Blue Shirt", got.Name)
}

func TestCreateHandlerShouldReturn400WhenPurchaseDateIsInvalid(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	w := postItem(t, h, "user-1", `{"name":"shirt","purchase_date":"not-a-date"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateHandlerShouldReturn422WhenServiceReturnsErrFutureDateNotAllowed(t *testing.T) {
	svc := &fakeItemService{
		createFn: func(_ context.Context, _ string, _ service.CreateItemInput) (domain.Item, error) {
			return domain.Item{}, domain.ErrFutureDateNotAllowed
		},
	}
	h := newItemHandler(svc)

	w := postItem(t, h, "user-1", `{"name":"shirt","purchase_date":"2020-01-01"}`)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateHandlerShouldReturn201WithPurchaseFieldsInResponse(t *testing.T) {
	price := "49.99"
	currency := "USD"
	seller := "https://example.com/shirt"
	purchaseDate, _ := time.Parse("2006-01-02", "2020-06-15")
	var item domain.Item
	item.ID = "item-1"
	item.OwnerID = "user-1"
	item.Name = "shirt"
	item.PurchasePrice = &price
	item.PurchaseCurrency = &currency
	item.PurchaseDate = &purchaseDate
	item.SellerURL = &seller

	svc := &fakeItemService{
		createFn: func(_ context.Context, _ string, input service.CreateItemInput) (domain.Item, error) {
			require.Equal(t, &price, input.PurchasePrice)
			require.Equal(t, &currency, input.PurchaseCurrency)
			require.Equal(t, &seller, input.SellerURL)
			require.NotNil(t, input.PurchaseDate)
			require.Equal(t, "2020-06-15", input.PurchaseDate.Format("2006-01-02"))
			return item, nil
		},
	}
	h := newItemHandler(svc)

	w := postItem(t, h, "user-1", `{"name":"shirt","purchase_price":"49.99","purchase_currency":"USD","purchase_date":"2020-06-15","seller_url":"https://example.com/shirt"}`)

	require.Equal(t, http.StatusCreated, w.Code)
	var got testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, &currency, got.PurchaseCurrency)
	require.Equal(t, ptr("2020-06-15"), got.PurchaseDate)
	require.Equal(t, &seller, got.SellerURL)
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

func TestAssignLocationHandlerShouldReturn409WhenServiceReturnsItemTransferPendingError(t *testing.T) {
	svc := &fakeItemService{
		assignLocationFn: func(_ context.Context, _, _ string, _ *string) error {
			return domain.ErrItemTransferPending
		},
	}
	h := handler.NewItemHandler(svc, slog.New(slog.DiscardHandler))

	w := patchItemLocation(t, h, "item-1", "user-1", `{}`)

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
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
		listByOwnerFn: func(_ context.Context, _ string, _ ports.ItemListFilter) ([]domain.Item, error) {
			return nil, domain.ErrIO
		},
	}
	h := newItemHandler(svc)

	w := listItems(t, h, "user-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListHandlerShouldReturn200WithItemsWhenListedSuccessfully(t *testing.T) {
	now := time.Now()
	var item1, item2 domain.Item
	item1.ID = "item-1"
	item1.OwnerID = "user-1"
	item1.Name = "Blue Shirt"
	item2.ID = "item-2"
	item2.OwnerID = "user-1"
	item2.Name = "Black Jeans"
	item2.ArchivedAt = &now

	svc := &fakeItemService{
		listByOwnerFn: func(_ context.Context, callerID string, _ ports.ItemListFilter) ([]domain.Item, error) {
			require.Equal(t, "user-1", callerID)
			return []domain.Item{item1, item2}, nil
		},
	}
	h := newItemHandler(svc)

	w := listItems(t, h, "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var got []testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Len(t, got, 2)
	require.Equal(t, "item-1", got[0].ID)
	require.Equal(t, "active", got[0].Status)
	require.Equal(t, "item-2", got[1].ID)
	require.Equal(t, "archived", got[1].Status)
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
	var got testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "item-42", got.ID)
	require.Equal(t, "Blue Shirt", got.Name)
}

func TestGetByIDHandlerShouldReturnArchivedStatusWhenItemIsArchived(t *testing.T) {
	now := time.Now()
	var item domain.Item
	item.ID = "item-42"
	item.OwnerID = "user-1"
	item.Name = "Blue Shirt"
	item.ArchivedAt = &now

	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Item, error) {
			return item, nil
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-42", "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var got testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "archived", got.Status)
}

func TestGetByIDHandlerShouldReturnDisposedStatusWhenItemIsDisposed(t *testing.T) {
	now := time.Now()
	reason := domain.DisposalDonated
	var item domain.Item
	item.ID = "item-42"
	item.OwnerID = "user-1"
	item.Name = "Blue Shirt"
	item.ArchivedAt = &now
	item.DisposalReason = &reason

	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Item, error) {
			return item, nil
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-42", "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var got testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "disposed", got.Status)
}

func TestGetByIDHandlerShouldReturnDisposeReasonWhenItemIsDisposed(t *testing.T) {
	now := time.Now()
	reason := domain.DisposalDonated
	var item domain.Item
	item.ID = "item-42"
	item.OwnerID = "user-1"
	item.Name = "Blue Shirt"
	item.ArchivedAt = &now
	item.DisposalReason = &reason

	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Item, error) {
			return item, nil
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-42", "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var got testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.NotNil(t, got.DisposeReason)
	require.Equal(t, "donated", *got.DisposeReason)
}

func TestGetByIDHandlerShouldReturnNullDisposeReasonWhenItemIsNotDisposed(t *testing.T) {
	var item domain.Item
	item.ID = "item-42"
	item.OwnerID = "user-1"
	item.Name = "Blue Shirt"

	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Item, error) {
			return item, nil
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-42", "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var got testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Nil(t, got.DisposeReason)
}

func TestGetByIDHandlerShouldReturnActiveStatusWhenItemIsNotArchivedOrDisposed(t *testing.T) {
	var item domain.Item
	item.ID = "item-42"
	item.OwnerID = "user-1"
	item.Name = "Blue Shirt"

	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Item, error) {
			return item, nil
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-42", "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var got testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "active", got.Status)
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

func TestUpdateHandlerShouldReturn422WhenServiceReturnsValidationError(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, _ service.UpdateItemInput) (domain.Item, error) {
			return domain.Item{}, domain.ErrValidation
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{"name":"shirt","metadata":{"bad key":"v"}}`)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
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

func TestUpdateHandlerShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, _ service.UpdateItemInput) (domain.Item, error) {
			return domain.Item{}, domain.ErrItemTransferPending
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{"name":"shirt"}`)

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
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
			require.NotNil(t, input.Name)
			require.Equal(t, "Red Jacket", *input.Name)
			require.NotNil(t, input.PurchasePrice)
			require.NotNil(t, *input.PurchasePrice)
			require.Equal(t, price, **input.PurchasePrice)
			return updated, nil
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-42", "user-1", `{"name":"Red Jacket","purchase_price":"49.99"}`)

	require.Equal(t, http.StatusOK, w.Code)
	var got testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, "item-42", got.ID)
	require.Equal(t, "Red Jacket", got.Name)
}

func TestUpdateHandlerShouldReturn400WhenNameIsNull(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	w := patchItem(t, h, "item-1", "user-1", `{"name":null}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateHandlerShouldPreserveNameWhenAbsentFromBody(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, input service.UpdateItemInput) (domain.Item, error) {
			require.Nil(t, input.Name)
			return domain.Item{}, nil
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{}`)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateHandlerShouldPreserveBrandWhenAbsentFromBody(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, input service.UpdateItemInput) (domain.Item, error) {
			require.Nil(t, input.Brand)
			return domain.Item{}, nil
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{}`)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateHandlerShouldClearBrandWhenNullInBody(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, input service.UpdateItemInput) (domain.Item, error) {
			require.NotNil(t, input.Brand)
			require.Nil(t, *input.Brand)
			return domain.Item{}, nil
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{"brand":null}`)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateHandlerShouldReturn400WhenNullableFieldHasInvalidType(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	// Sending a number for brand (expected string or null) triggers decodePatchNullable error.
	w := patchItem(t, h, "item-1", "user-1", `{"brand":123}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateHandlerShouldReturn400WhenPurchaseDateIsInvalid(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	w := patchItem(t, h, "item-1", "user-1", `{"name":"shirt","purchase_date":"not-a-date"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateHandlerShouldReturn400WhenPurchaseDateHasInvalidJSONType(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	// Sending a number for purchase_date (expected string or null) triggers json.Unmarshal error.
	w := patchItem(t, h, "item-1", "user-1", `{"purchase_date":123}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateHandlerShouldClearPurchaseDateWhenNullInBody(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, input service.UpdateItemInput) (domain.Item, error) {
			require.NotNil(t, input.PurchaseDate)
			require.Nil(t, *input.PurchaseDate)
			return domain.Item{}, nil
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{"purchase_date":null}`)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateHandlerShouldReturn422WhenServiceReturnsErrFutureDateNotAllowed(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, _ service.UpdateItemInput) (domain.Item, error) {
			return domain.Item{}, domain.ErrFutureDateNotAllowed
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{"name":"shirt","purchase_date":"2020-01-01"}`)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateHandlerShouldReturn200WithPurchaseFieldsInResponse(t *testing.T) {
	currency := "EUR"
	seller := "https://example.com/jacket"
	purchaseDate, _ := time.Parse("2006-01-02", "2021-03-10")
	var updated domain.Item
	updated.ID = "item-42"
	updated.OwnerID = "user-1"
	updated.Name = "Red Jacket"
	updated.PurchaseCurrency = &currency
	updated.PurchaseDate = &purchaseDate
	updated.SellerURL = &seller

	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, input service.UpdateItemInput) (domain.Item, error) {
			require.NotNil(t, input.PurchaseCurrency)
			require.NotNil(t, *input.PurchaseCurrency)
			require.Equal(t, currency, **input.PurchaseCurrency)
			require.NotNil(t, input.SellerURL)
			require.NotNil(t, *input.SellerURL)
			require.Equal(t, seller, **input.SellerURL)
			require.NotNil(t, input.PurchaseDate)
			require.NotNil(t, *input.PurchaseDate)
			require.Equal(t, "2021-03-10", (*input.PurchaseDate).Format("2006-01-02"))
			return updated, nil
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-42", "user-1", `{"name":"Red Jacket","purchase_currency":"EUR","purchase_date":"2021-03-10","seller_url":"https://example.com/jacket"}`)

	require.Equal(t, http.StatusOK, w.Code)
	var got testItemResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, &currency, got.PurchaseCurrency)
	require.Equal(t, ptr("2021-03-10"), got.PurchaseDate)
	require.Equal(t, &seller, got.SellerURL)
}

func TestUpdateHandlerShouldReturn400WhenMetadataIsInvalid(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	w := patchItem(t, h, "item-1", "user-1", `{"metadata":123}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateHandlerShouldPassMetadataToServiceWhenPresentInBody(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, input service.UpdateItemInput) (domain.Item, error) {
			require.NotNil(t, input.Metadata)
			require.Equal(t, map[string]string{"size": "L"}, input.Metadata.Fields)
			return domain.Item{}, nil
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{"metadata":{"size":"L"}}`)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateHandlerShouldPreserveMetadataWhenAbsentFromBody(t *testing.T) {
	svc := &fakeItemService{
		updateFn: func(_ context.Context, _, _ string, input service.UpdateItemInput) (domain.Item, error) {
			require.Nil(t, input.Metadata)
			return domain.Item{}, nil
		},
	}
	h := newItemHandler(svc)

	w := patchItem(t, h, "item-1", "user-1", `{"name":"shirt"}`)

	require.Equal(t, http.StatusOK, w.Code)
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

func TestDeleteHandlerShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeItemService{
		deleteFn: func(_ context.Context, _, _ string) error {
			return domain.ErrItemTransferPending
		},
	}
	h := newItemHandler(svc)

	w := deleteItem(t, h, "item-1", "user-1")

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
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

func TestUploadPhotoHandlerShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeItemService{
		uploadPhotoFn: func(_ context.Context, _, _ string, _ io.Reader, _ string) error {
			return domain.ErrItemTransferPending
		},
	}
	h := newItemHandler(svc)

	w := uploadPhoto(t, h, "item-1", "user-1", "photo.jpg", "data")

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
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

func TestDeletePhotoHandlerShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeItemService{
		deletePhotoFn: func(_ context.Context, _, _, _ string) error {
			return domain.ErrItemTransferPending
		},
	}
	h := newItemHandler(svc)

	w := deletePhoto(t, h, "item-1", "user-1", "key-1")

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
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

// testItemResp mirrors the snake_case JSON shape returned by the item handlers.
type testItemResp struct {
	ID               string            `json:"id"`
	OwnerID          string            `json:"owner_id"`
	Name             string            `json:"name"`
	Brand            *string           `json:"brand"`
	Color            *string           `json:"color"`
	Meta             map[string]string `json:"metadata"`
	PurchaseCurrency *string           `json:"purchase_currency"`
	PurchaseDate     *string           `json:"purchase_date"`
	SellerURL        *string           `json:"seller_url"`
	Status           string            `json:"status"`
	DisposeReason    *string           `json:"dispose_reason"`
}

func ptr(s string) *string { return &s }

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

func (s *statefulFakeItemService) ListByOwner(ctx context.Context, callerID string, _ ports.ItemListFilter) ([]domain.Item, error) {
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
	if input.Name != nil {
		item.Name = *input.Name
	}
	if input.Brand != nil {
		item.Brand = *input.Brand
	}
	if input.CategoryID != nil {
		item.CategoryID = *input.CategoryID
	}
	if input.Color != nil {
		item.Color = *input.Color
	}
	if input.Metadata != nil {
		item.Metadata = *input.Metadata
	}
	if input.LocationID != nil {
		item.LocationID = *input.LocationID
	}
	if input.PurchasePrice != nil {
		item.PurchasePrice = *input.PurchasePrice
	}
	if input.PurchaseDate != nil {
		item.PurchaseDate = *input.PurchaseDate
	}
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

func (s *statefulFakeItemService) Archive(ctx context.Context, _, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *statefulFakeItemService) Unarchive(ctx context.Context, _, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *statefulFakeItemService) Dispose(ctx context.Context, _, _ string, _ domain.DisposalReason) error {
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
	var created testItemResp
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
	var listed []testItemResp
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
	var fetched testItemResp
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
	var updated testItemResp
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

func TestGetByIDHandlerShouldIncludePhotosInResponseWhenItemHasPhotos(t *testing.T) {
	var item domain.Item
	item.ID = "item-42"
	item.OwnerID = "user-1"
	item.Name = "Blue Shirt"
	item.Photos = []domain.ItemPhoto{
		{ID: "photo-1", MediaKey: "media/photo.jpg", Position: 0},
	}

	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Item, error) {
			return item, nil
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-42", "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var got struct {
		Photos []struct {
			ID       string `json:"id"`
			MediaKey string `json:"media_key"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Len(t, got.Photos, 1)
	require.Equal(t, "photo-1", got.Photos[0].ID)
	require.Equal(t, "media/photo.jpg", got.Photos[0].MediaKey)
	require.Equal(t, 0, got.Photos[0].Position)
}

func TestGetByIDHandlerShouldIncludeMetadataAsFlatMapWhenItemHasMetadataFields(t *testing.T) {
	var item domain.Item
	item.ID = "item-42"
	item.OwnerID = "user-1"
	item.Name = "Blue Shirt"
	item.Metadata = domain.ItemMetadata{Fields: map[string]string{"size": "M", "material": "cotton"}}

	svc := &fakeItemService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Item, error) {
			return item, nil
		},
	}
	h := newItemHandler(svc)

	w := getItem(t, h, "item-42", "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var got struct {
		Metadata map[string]string `json:"metadata"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, map[string]string{"size": "M", "material": "cotton"}, got.Metadata)
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

// ── Archive ───────────────────────────────────────────────────────────────────

func postArchive(t *testing.T, h *handler.ItemHandler, itemID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/items/"+itemID+"/archive", http.NoBody)
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.Archive(w, req)
	return w
}

func TestArchiveHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items/42/archive", http.NoBody)
	req.SetPathValue("id", "42")
	w := httptest.NewRecorder()
	h.Archive(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestArchiveHandlerShouldReturn404WhenItemNotFound(t *testing.T) {
	svc := &fakeItemService{
		archiveFn: func(_ context.Context, _, _ string) error { return domain.ErrNotFound },
	}
	w := postArchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestArchiveHandlerShouldReturn403WhenCallerDoesNotOwnItem(t *testing.T) {
	svc := &fakeItemService{
		archiveFn: func(_ context.Context, _, _ string) error { return domain.ErrForbidden },
	}
	w := postArchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestArchiveHandlerShouldReturn409WhenItemAlreadyArchived(t *testing.T) {
	svc := &fakeItemService{
		archiveFn: func(_ context.Context, _, _ string) error { return domain.ErrAlreadyArchived },
	}
	w := postArchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestArchiveHandlerShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeItemService{
		archiveFn: func(_ context.Context, _, _ string) error { return domain.ErrItemTransferPending },
	}
	w := postArchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
}

func TestArchiveHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		archiveFn: func(_ context.Context, _, _ string) error { return errors.New("unexpected") },
	}
	w := postArchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestArchiveHandlerShouldReturn204WhenSuccessful(t *testing.T) {
	var capturedItemID string
	svc := &fakeItemService{
		archiveFn: func(_ context.Context, _, itemID string) error {
			capturedItemID = itemID
			return nil
		},
	}
	w := postArchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "42", capturedItemID)
}

// ── Unarchive ─────────────────────────────────────────────────────────────────

func postUnarchive(t *testing.T, h *handler.ItemHandler, itemID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/items/"+itemID+"/unarchive", http.NoBody)
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.Unarchive(w, req)
	return w
}

func TestUnarchiveHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items/42/unarchive", http.NoBody)
	req.SetPathValue("id", "42")
	w := httptest.NewRecorder()
	h.Unarchive(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUnarchiveHandlerShouldReturn404WhenItemNotFound(t *testing.T) {
	svc := &fakeItemService{
		unarchiveFn: func(_ context.Context, _, _ string) error { return domain.ErrNotFound },
	}
	w := postUnarchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUnarchiveHandlerShouldReturn403WhenCallerDoesNotOwnItem(t *testing.T) {
	svc := &fakeItemService{
		unarchiveFn: func(_ context.Context, _, _ string) error { return domain.ErrForbidden },
	}
	w := postUnarchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUnarchiveHandlerShouldReturn409WhenItemNotArchived(t *testing.T) {
	svc := &fakeItemService{
		unarchiveFn: func(_ context.Context, _, _ string) error { return domain.ErrNotArchived },
	}
	w := postUnarchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestUnarchiveHandlerShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeItemService{
		unarchiveFn: func(_ context.Context, _, _ string) error { return domain.ErrItemTransferPending },
	}
	w := postUnarchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
}

func TestUnarchiveHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		unarchiveFn: func(_ context.Context, _, _ string) error { return errors.New("unexpected") },
	}
	w := postUnarchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUnarchiveHandlerShouldReturn204WhenSuccessful(t *testing.T) {
	var capturedItemID string
	svc := &fakeItemService{
		unarchiveFn: func(_ context.Context, _, itemID string) error {
			capturedItemID = itemID
			return nil
		},
	}
	w := postUnarchive(t, newItemHandler(svc), "42", "user-1")
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "42", capturedItemID)
}

// ── Dispose ───────────────────────────────────────────────────────────────────

func postDispose(t *testing.T, h *handler.ItemHandler, itemID, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/items/"+itemID+"/dispose", strings.NewReader(body))
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.Dispose(w, req)
	return w
}

func TestDisposeHandlerShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newItemHandler(&fakeItemService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items/42/dispose", strings.NewReader(`{"reason":"donated"}`))
	req.SetPathValue("id", "42")
	w := httptest.NewRecorder()
	h.Dispose(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDisposeHandlerShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	w := postDispose(t, newItemHandler(&fakeItemService{}), "42", "user-1", `not-json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDisposeHandlerShouldReturn400WhenReasonIsEmpty(t *testing.T) {
	w := postDispose(t, newItemHandler(&fakeItemService{}), "42", "user-1", `{"reason":""}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDisposeHandlerShouldReturn400WhenReasonIsUnknown(t *testing.T) {
	w := postDispose(t, newItemHandler(&fakeItemService{}), "42", "user-1", `{"reason":"unknown"}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDisposeHandlerShouldReturn404WhenItemNotFound(t *testing.T) {
	svc := &fakeItemService{
		disposeFn: func(_ context.Context, _, _ string, _ domain.DisposalReason) error { return domain.ErrNotFound },
	}
	w := postDispose(t, newItemHandler(svc), "42", "user-1", `{"reason":"donated"}`)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDisposeHandlerShouldReturn403WhenCallerDoesNotOwnItem(t *testing.T) {
	svc := &fakeItemService{
		disposeFn: func(_ context.Context, _, _ string, _ domain.DisposalReason) error { return domain.ErrForbidden },
	}
	w := postDispose(t, newItemHandler(svc), "42", "user-1", `{"reason":"donated"}`)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestDisposeHandlerShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeItemService{
		disposeFn: func(_ context.Context, _, _ string, _ domain.DisposalReason) error {
			return domain.ErrItemTransferPending
		},
	}
	w := postDispose(t, newItemHandler(svc), "42", "user-1", `{"reason":"donated"}`)
	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
}

func TestDisposeHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeItemService{
		disposeFn: func(_ context.Context, _, _ string, _ domain.DisposalReason) error { return errors.New("unexpected") },
	}
	w := postDispose(t, newItemHandler(svc), "42", "user-1", `{"reason":"donated"}`)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDisposeHandlerShouldReturn204WhenSuccessful(t *testing.T) {
	var capturedItemID string
	var capturedReason domain.DisposalReason
	svc := &fakeItemService{
		disposeFn: func(_ context.Context, _, itemID string, reason domain.DisposalReason) error {
			capturedItemID = itemID
			capturedReason = reason
			return nil
		},
	}
	w := postDispose(t, newItemHandler(svc), "42", "user-1", `{"reason":"donated"}`)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "42", capturedItemID)
	require.Equal(t, domain.DisposalDonated, capturedReason)
}

// ── List with status filter ───────────────────────────────────────────────────

func listItemsWithStatus(t *testing.T, h *handler.ItemHandler, callerID, status string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	url := "/items"
	if status != "" {
		url += "?status=" + status
	}
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	w := httptest.NewRecorder()
	h.List(w, req)
	return w
}

func TestListHandlerShouldReturn400WhenStatusQueryParamIsUnknown(t *testing.T) {
	w := listItemsWithStatus(t, newItemHandler(&fakeItemService{}), "user-1", "invalid")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListHandlerShouldPassActiveStatusFilterByDefaultWhenNoQueryParam(t *testing.T) {
	var gotFilter ports.ItemListFilter
	svc := &fakeItemService{
		listByOwnerFn: func(_ context.Context, _ string, filter ports.ItemListFilter) ([]domain.Item, error) {
			gotFilter = filter
			return nil, nil
		},
	}
	listItemsWithStatus(t, newItemHandler(svc), "user-1", "")
	require.Equal(t, ports.ItemStatusActive, gotFilter.Status)
}

func TestListHandlerShouldPassArchivedStatusFilterWhenQueryParamIsArchived(t *testing.T) {
	var gotFilter ports.ItemListFilter
	svc := &fakeItemService{
		listByOwnerFn: func(_ context.Context, _ string, filter ports.ItemListFilter) ([]domain.Item, error) {
			gotFilter = filter
			return nil, nil
		},
	}
	listItemsWithStatus(t, newItemHandler(svc), "user-1", "archived")
	require.Equal(t, ports.ItemStatusArchived, gotFilter.Status)
}

func TestListHandlerShouldPassAllStatusFilterWhenQueryParamIsAll(t *testing.T) {
	var gotFilter ports.ItemListFilter
	svc := &fakeItemService{
		listByOwnerFn: func(_ context.Context, _ string, filter ports.ItemListFilter) ([]domain.Item, error) {
			gotFilter = filter
			return nil, nil
		},
	}
	listItemsWithStatus(t, newItemHandler(svc), "user-1", "all")
	require.Equal(t, ports.ItemStatusAll, gotFilter.Status)
}
