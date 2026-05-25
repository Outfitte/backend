package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/outfitte/backend/internal/api/handler"
	"github.com/outfitte/backend/internal/api/middleware"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/service"
	"github.com/stretchr/testify/require"
)

// --- fake ---

type fakeOutfitService struct {
	createFn          func(ctx context.Context, callerID string, input service.CreateOutfitInput) (domain.Outfit, error)
	getByIDFn         func(ctx context.Context, callerID, outfitID string) (domain.Outfit, error)
	listByOwnerFn     func(ctx context.Context, callerID string) ([]domain.Outfit, error)
	listByDateRangeFn func(ctx context.Context, callerID string, from, to time.Time) ([]domain.Outfit, error)
	updateFn          func(ctx context.Context, callerID, outfitID string, input service.UpdateOutfitInput) (domain.Outfit, error)
	deleteFn          func(ctx context.Context, callerID, outfitID string) error
	addItemFn         func(ctx context.Context, callerID, outfitID, itemID string) error
	removeItemFn      func(ctx context.Context, callerID, outfitID, itemID string) error
	uploadPhotoFn     func(ctx context.Context, callerID, outfitID string, r io.Reader, filename string) (domain.OutfitPhoto, error)
	deletePhotoFn     func(ctx context.Context, callerID, outfitID, mediaKey string) error
}

func (f *fakeOutfitService) Create(ctx context.Context, callerID string, input service.CreateOutfitInput) (domain.Outfit, error) {
	if f.createFn != nil {
		return f.createFn(ctx, callerID, input)
	}
	return domain.Outfit{}, nil
}

func (f *fakeOutfitService) GetByID(ctx context.Context, callerID, outfitID string) (domain.Outfit, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, callerID, outfitID)
	}
	return domain.Outfit{}, nil
}

func (f *fakeOutfitService) ListByOwner(ctx context.Context, callerID string) ([]domain.Outfit, error) {
	if f.listByOwnerFn != nil {
		return f.listByOwnerFn(ctx, callerID)
	}
	return nil, nil
}

func (f *fakeOutfitService) ListByDateRange(ctx context.Context, callerID string, from, to time.Time) ([]domain.Outfit, error) {
	if f.listByDateRangeFn != nil {
		return f.listByDateRangeFn(ctx, callerID, from, to)
	}
	return nil, nil
}

func (f *fakeOutfitService) Update(ctx context.Context, callerID, outfitID string, input service.UpdateOutfitInput) (domain.Outfit, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, callerID, outfitID, input)
	}
	return domain.Outfit{}, nil
}

func (f *fakeOutfitService) Delete(ctx context.Context, callerID, outfitID string) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, callerID, outfitID)
	}
	return nil
}

func (f *fakeOutfitService) AddItem(ctx context.Context, callerID, outfitID, itemID string) error {
	if f.addItemFn != nil {
		return f.addItemFn(ctx, callerID, outfitID, itemID)
	}
	return nil
}

func (f *fakeOutfitService) RemoveItem(ctx context.Context, callerID, outfitID, itemID string) error {
	if f.removeItemFn != nil {
		return f.removeItemFn(ctx, callerID, outfitID, itemID)
	}
	return nil
}

func (f *fakeOutfitService) UploadPhoto(ctx context.Context, callerID, outfitID string, r io.Reader, filename string) (domain.OutfitPhoto, error) {
	if f.uploadPhotoFn != nil {
		return f.uploadPhotoFn(ctx, callerID, outfitID, r, filename)
	}
	return domain.OutfitPhoto{}, nil
}

func (f *fakeOutfitService) DeletePhoto(ctx context.Context, callerID, outfitID, mediaKey string) error {
	if f.deletePhotoFn != nil {
		return f.deletePhotoFn(ctx, callerID, outfitID, mediaKey)
	}
	return nil
}

// --- helpers ---

func newOutfitHandler(svc *fakeOutfitService) *handler.OutfitHandler {
	return handler.NewOutfitHandler(svc, slog.New(slog.DiscardHandler))
}

func outfitWithID(id, ownerID string) domain.Outfit {
	var o domain.Outfit
	o.ID = id
	o.OwnerID = ownerID
	o.CreatedAt = time.Now().UTC()
	return o
}

func postOutfit(t *testing.T, h *handler.OutfitHandler, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/outfits", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, req)
	return w
}

func listOutfits(t *testing.T, h *handler.OutfitHandler, callerID string, query string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), callerID)
	url := "/outfits"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	w := httptest.NewRecorder()
	h.List(w, req)
	return w
}

func getOutfit(t *testing.T, h *handler.OutfitHandler, outfitID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/outfits/"+outfitID, http.NoBody)
	req.SetPathValue("id", outfitID)
	w := httptest.NewRecorder()
	h.GetByID(w, req)
	return w
}

func patchOutfit(t *testing.T, h *handler.OutfitHandler, outfitID, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/outfits/"+outfitID, strings.NewReader(body))
	req.SetPathValue("id", outfitID)
	w := httptest.NewRecorder()
	h.Update(w, req)
	return w
}

func deleteOutfit(t *testing.T, h *handler.OutfitHandler, outfitID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/outfits/"+outfitID, http.NoBody)
	req.SetPathValue("id", outfitID)
	w := httptest.NewRecorder()
	h.Delete(w, req)
	return w
}

func addOutfitItem(t *testing.T, h *handler.OutfitHandler, outfitID, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/outfits/"+outfitID+"/items", strings.NewReader(body))
	req.SetPathValue("id", outfitID)
	w := httptest.NewRecorder()
	h.AddItem(w, req)
	return w
}

func removeOutfitItem(t *testing.T, h *handler.OutfitHandler, outfitID, itemID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/outfits/"+outfitID+"/items/"+itemID, http.NoBody)
	req.SetPathValue("id", outfitID)
	req.SetPathValue("itemID", itemID)
	w := httptest.NewRecorder()
	h.RemoveItem(w, req)
	return w
}

func uploadOutfitPhoto(t *testing.T, h *handler.OutfitHandler, outfitID, callerID, filename, content string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("photo", filename)
	require.NoError(t, err)
	_, err = io.WriteString(fw, content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	ctx := middleware.WithUserID(t.Context(), callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/outfits/"+outfitID+"/photos", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.SetPathValue("id", outfitID)
	w := httptest.NewRecorder()
	h.UploadPhoto(w, req)
	return w
}

func deleteOutfitPhoto(t *testing.T, h *handler.OutfitHandler, outfitID, photoKey, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := middleware.WithUserID(t.Context(), callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/outfits/"+outfitID+"/photos/"+photoKey, http.NoBody)
	req.SetPathValue("id", outfitID)
	req.SetPathValue("key", photoKey)
	w := httptest.NewRecorder()
	h.DeletePhoto(w, req)
	return w
}

// ---- Create ----

func TestOutfitCreateShouldReturn500WhenCallerIDMissing(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/outfits", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	h.Create(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitCreateShouldReturn400WhenBodyInvalid(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	ctx := middleware.WithUserID(t.Context(), "user1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/outfits", strings.NewReader("not-json"))
	w := httptest.NewRecorder()
	h.Create(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitCreateShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitService{
		createFn: func(_ context.Context, _ string, _ service.CreateOutfitInput) (domain.Outfit, error) {
			return domain.Outfit{}, errors.New("boom")
		},
	}
	w := postOutfit(t, newOutfitHandler(svc), "user1", `{}`)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitCreateShouldReturn201WhenSuccessful(t *testing.T) {
	name := "Summer Look"
	created := outfitWithID("o1", "user1")
	created.Name = &name
	svc := &fakeOutfitService{
		createFn: func(_ context.Context, callerID string, input service.CreateOutfitInput) (domain.Outfit, error) {
			require.Equal(t, "user1", callerID)
			require.Equal(t, &name, input.Name)
			return created, nil
		},
	}
	w := postOutfit(t, newOutfitHandler(svc), "user1", `{"name":"Summer Look"}`)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "o1", resp["id"])
	require.Equal(t, "Summer Look", resp["name"])
}

// ---- List ----

func TestOutfitListShouldReturn500WhenCallerIDMissing(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/outfits", http.NoBody)
	w := httptest.NewRecorder()
	h.List(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitListShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitService{
		listByOwnerFn: func(_ context.Context, _ string) ([]domain.Outfit, error) {
			return nil, errors.New("boom")
		},
	}
	w := listOutfits(t, newOutfitHandler(svc), "user1", "")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitListShouldReturn200WithOutfitsWhenSuccessful(t *testing.T) {
	outfits := []domain.Outfit{outfitWithID("o1", "user1"), outfitWithID("o2", "user1")}
	svc := &fakeOutfitService{
		listByOwnerFn: func(_ context.Context, callerID string) ([]domain.Outfit, error) {
			require.Equal(t, "user1", callerID)
			return outfits, nil
		},
	}
	w := listOutfits(t, newOutfitHandler(svc), "user1", "")
	require.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp, 2)
}

func TestOutfitListShouldReturn400WhenOnlyFromProvided(t *testing.T) {
	w := listOutfits(t, newOutfitHandler(&fakeOutfitService{}), "user1", "from=2024-06-01")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitListShouldReturn400WhenOnlyToProvided(t *testing.T) {
	w := listOutfits(t, newOutfitHandler(&fakeOutfitService{}), "user1", "to=2024-06-30")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitListShouldReturn400WhenFromDateInvalid(t *testing.T) {
	w := listOutfits(t, newOutfitHandler(&fakeOutfitService{}), "user1", "from=not-a-date&to=2024-06-30")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitListShouldReturn400WhenToDateInvalid(t *testing.T) {
	w := listOutfits(t, newOutfitHandler(&fakeOutfitService{}), "user1", "from=2024-06-01&to=not-a-date")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitListShouldReturn400WhenFromIsAfterTo(t *testing.T) {
	svc := &fakeOutfitService{
		listByDateRangeFn: func(_ context.Context, _ string, _, _ time.Time) ([]domain.Outfit, error) {
			return nil, domain.ErrValidation
		},
	}
	w := listOutfits(t, newOutfitHandler(svc), "user1", "from=2024-06-30&to=2024-06-01")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitListShouldReturn500WhenDateRangeServiceFails(t *testing.T) {
	svc := &fakeOutfitService{
		listByDateRangeFn: func(_ context.Context, _ string, _, _ time.Time) ([]domain.Outfit, error) {
			return nil, errors.New("boom")
		},
	}
	w := listOutfits(t, newOutfitHandler(svc), "user1", "from=2024-06-01&to=2024-06-30")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitListShouldReturn200WithCalendarFilterWhenDateRangeProvided(t *testing.T) {
	filtered := []domain.Outfit{outfitWithID("o1", "user1")}
	svc := &fakeOutfitService{
		listByDateRangeFn: func(_ context.Context, callerID string, from, to time.Time) ([]domain.Outfit, error) {
			require.Equal(t, "user1", callerID)
			require.Equal(t, 2024, from.Year())
			require.Equal(t, 6, int(from.Month()))
			require.Equal(t, 1, from.Day())
			return filtered, nil
		},
	}
	w := listOutfits(t, newOutfitHandler(svc), "user1", "from=2024-06-01&to=2024-06-30")
	require.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	require.Equal(t, "o1", resp[0]["id"])
}

// ---- GetByID ----

func TestOutfitGetByIDShouldReturn500WhenCallerIDMissing(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/outfits/o1", http.NoBody)
	req.SetPathValue("id", "o1")
	w := httptest.NewRecorder()
	h.GetByID(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitGetByIDShouldReturn404WhenNotFound(t *testing.T) {
	svc := &fakeOutfitService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Outfit, error) {
			return domain.Outfit{}, domain.ErrNotFound
		},
	}
	w := getOutfit(t, newOutfitHandler(svc), "o1", "user1")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestOutfitGetByIDShouldReturn403WhenForbidden(t *testing.T) {
	svc := &fakeOutfitService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Outfit, error) {
			return domain.Outfit{}, domain.ErrForbidden
		},
	}
	w := getOutfit(t, newOutfitHandler(svc), "o1", "user1")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestOutfitGetByIDShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitService{
		getByIDFn: func(_ context.Context, _, _ string) (domain.Outfit, error) {
			return domain.Outfit{}, errors.New("boom")
		},
	}
	w := getOutfit(t, newOutfitHandler(svc), "o1", "user1")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitGetByIDShouldReturn200WhenSuccessful(t *testing.T) {
	outfit := outfitWithID("o1", "user1")
	outfit.Items = []domain.OutfitItem{{OutfitID: "o1", ItemID: "i1", Position: 0}}
	outfit.Photos = []domain.OutfitPhoto{{ID: "ph1", MediaKey: "outfits/o1/p.jpg", Position: 0, CreatedAt: time.Now()}}
	svc := &fakeOutfitService{
		getByIDFn: func(_ context.Context, callerID, outfitID string) (domain.Outfit, error) {
			require.Equal(t, "user1", callerID)
			require.Equal(t, "o1", outfitID)
			return outfit, nil
		},
	}
	w := getOutfit(t, newOutfitHandler(svc), "o1", "user1")
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "o1", resp["id"])
	items := resp["items"].([]any)
	require.Len(t, items, 1)
	photos := resp["photos"].([]any)
	require.Len(t, photos, 1)
}

// ---- Update ----

func TestOutfitUpdateShouldReturn500WhenCallerIDMissing(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/outfits/o1", strings.NewReader("{}"))
	req.SetPathValue("id", "o1")
	w := httptest.NewRecorder()
	h.Update(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitUpdateShouldReturn400WhenBodyInvalid(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	ctx := middleware.WithUserID(t.Context(), "user1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/outfits/o1", strings.NewReader("bad"))
	req.SetPathValue("id", "o1")
	w := httptest.NewRecorder()
	h.Update(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitUpdateShouldReturn404WhenNotFound(t *testing.T) {
	svc := &fakeOutfitService{
		updateFn: func(_ context.Context, _, _ string, _ service.UpdateOutfitInput) (domain.Outfit, error) {
			return domain.Outfit{}, domain.ErrNotFound
		},
	}
	w := patchOutfit(t, newOutfitHandler(svc), "o1", "user1", `{}`)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestOutfitUpdateShouldReturn403WhenForbidden(t *testing.T) {
	svc := &fakeOutfitService{
		updateFn: func(_ context.Context, _, _ string, _ service.UpdateOutfitInput) (domain.Outfit, error) {
			return domain.Outfit{}, domain.ErrForbidden
		},
	}
	w := patchOutfit(t, newOutfitHandler(svc), "o1", "user1", `{}`)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestOutfitUpdateShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitService{
		updateFn: func(_ context.Context, _, _ string, _ service.UpdateOutfitInput) (domain.Outfit, error) {
			return domain.Outfit{}, errors.New("boom")
		},
	}
	w := patchOutfit(t, newOutfitHandler(svc), "o1", "user1", `{}`)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitUpdateShouldReturn200WhenSuccessful(t *testing.T) {
	name := "Winter Fit"
	updated := outfitWithID("o1", "user1")
	updated.Name = &name
	svc := &fakeOutfitService{
		updateFn: func(_ context.Context, callerID, outfitID string, input service.UpdateOutfitInput) (domain.Outfit, error) {
			require.Equal(t, "user1", callerID)
			require.Equal(t, "o1", outfitID)
			require.NotNil(t, input.Name)
			require.Equal(t, name, *input.Name)
			return updated, nil
		},
	}
	w := patchOutfit(t, newOutfitHandler(svc), "o1", "user1", `{"name":"Winter Fit"}`)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "o1", resp["id"])
	require.Equal(t, "Winter Fit", resp["name"])
}

func TestOutfitUpdateShouldPreserveNotesWhenAbsentFromBody(t *testing.T) {
	svc := &fakeOutfitService{
		updateFn: func(_ context.Context, _, _ string, input service.UpdateOutfitInput) (domain.Outfit, error) {
			require.Nil(t, input.Notes)
			return domain.Outfit{}, nil
		},
	}
	w := patchOutfit(t, newOutfitHandler(svc), "o1", "user1", `{}`)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestOutfitUpdateShouldClearNotesWhenNullInBody(t *testing.T) {
	svc := &fakeOutfitService{
		updateFn: func(_ context.Context, _, _ string, input service.UpdateOutfitInput) (domain.Outfit, error) {
			require.NotNil(t, input.Notes)
			require.Nil(t, *input.Notes)
			return domain.Outfit{}, nil
		},
	}
	w := patchOutfit(t, newOutfitHandler(svc), "o1", "user1", `{"notes":null}`)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestOutfitUpdateShouldReturn400WhenNameIsNull(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	w := patchOutfit(t, h, "o1", "user1", `{"name":null}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitUpdateShouldReturn400WhenNullableFieldHasInvalidType(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	w := patchOutfit(t, h, "o1", "user1", `{"notes":123}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitUpdateShouldReturn400WhenNameHasInvalidType(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	w := patchOutfit(t, h, "o1", "user1", `{"name":123}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// ---- Delete ----

func TestOutfitDeleteShouldReturn500WhenCallerIDMissing(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/outfits/o1", http.NoBody)
	req.SetPathValue("id", "o1")
	w := httptest.NewRecorder()
	h.Delete(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitDeleteShouldReturn404WhenNotFound(t *testing.T) {
	svc := &fakeOutfitService{
		deleteFn: func(_ context.Context, _, _ string) error { return domain.ErrNotFound },
	}
	w := deleteOutfit(t, newOutfitHandler(svc), "o1", "user1")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestOutfitDeleteShouldReturn403WhenForbidden(t *testing.T) {
	svc := &fakeOutfitService{
		deleteFn: func(_ context.Context, _, _ string) error { return domain.ErrForbidden },
	}
	w := deleteOutfit(t, newOutfitHandler(svc), "o1", "user1")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestOutfitDeleteShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitService{
		deleteFn: func(_ context.Context, _, _ string) error { return errors.New("boom") },
	}
	w := deleteOutfit(t, newOutfitHandler(svc), "o1", "user1")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitDeleteShouldReturn204WhenSuccessful(t *testing.T) {
	var called bool
	svc := &fakeOutfitService{
		deleteFn: func(_ context.Context, callerID, outfitID string) error {
			require.Equal(t, "user1", callerID)
			require.Equal(t, "o1", outfitID)
			called = true
			return nil
		},
	}
	w := deleteOutfit(t, newOutfitHandler(svc), "o1", "user1")
	require.Equal(t, http.StatusNoContent, w.Code)
	require.True(t, called)
}

// ---- AddItem ----

func TestOutfitAddItemShouldReturn500WhenCallerIDMissing(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/outfits/o1/items", strings.NewReader(`{"item_id":"i1"}`))
	req.SetPathValue("id", "o1")
	w := httptest.NewRecorder()
	h.AddItem(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitAddItemShouldReturn400WhenBodyInvalid(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	ctx := middleware.WithUserID(t.Context(), "user1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/outfits/o1/items", strings.NewReader("bad"))
	req.SetPathValue("id", "o1")
	w := httptest.NewRecorder()
	h.AddItem(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitAddItemShouldReturn404WhenNotFound(t *testing.T) {
	svc := &fakeOutfitService{
		addItemFn: func(_ context.Context, _, _, _ string) error { return domain.ErrNotFound },
	}
	w := addOutfitItem(t, newOutfitHandler(svc), "o1", "user1", `{"item_id":"i1"}`)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestOutfitAddItemShouldReturn403WhenForbidden(t *testing.T) {
	svc := &fakeOutfitService{
		addItemFn: func(_ context.Context, _, _, _ string) error { return domain.ErrForbidden },
	}
	w := addOutfitItem(t, newOutfitHandler(svc), "o1", "user1", `{"item_id":"i1"}`)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestOutfitAddItemShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeOutfitService{
		addItemFn: func(_ context.Context, _, _, _ string) error { return domain.ErrItemTransferPending },
	}
	w := addOutfitItem(t, newOutfitHandler(svc), "o1", "user1", `{"item_id":"i1"}`)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestOutfitAddItemShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitService{
		addItemFn: func(_ context.Context, _, _, _ string) error { return errors.New("boom") },
	}
	w := addOutfitItem(t, newOutfitHandler(svc), "o1", "user1", `{"item_id":"i1"}`)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitAddItemShouldReturn204WhenSuccessful(t *testing.T) {
	var called bool
	svc := &fakeOutfitService{
		addItemFn: func(_ context.Context, callerID, outfitID, itemID string) error {
			require.Equal(t, "user1", callerID)
			require.Equal(t, "o1", outfitID)
			require.Equal(t, "i1", itemID)
			called = true
			return nil
		},
	}
	w := addOutfitItem(t, newOutfitHandler(svc), "o1", "user1", `{"item_id":"i1"}`)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.True(t, called)
}

// ---- RemoveItem ----

func TestOutfitRemoveItemShouldReturn500WhenCallerIDMissing(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/outfits/o1/items/i1", http.NoBody)
	req.SetPathValue("id", "o1")
	req.SetPathValue("itemID", "i1")
	w := httptest.NewRecorder()
	h.RemoveItem(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitRemoveItemShouldReturn404WhenNotFound(t *testing.T) {
	svc := &fakeOutfitService{
		removeItemFn: func(_ context.Context, _, _, _ string) error { return domain.ErrNotFound },
	}
	w := removeOutfitItem(t, newOutfitHandler(svc), "o1", "i1", "user1")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestOutfitRemoveItemShouldReturn403WhenForbidden(t *testing.T) {
	svc := &fakeOutfitService{
		removeItemFn: func(_ context.Context, _, _, _ string) error { return domain.ErrForbidden },
	}
	w := removeOutfitItem(t, newOutfitHandler(svc), "o1", "i1", "user1")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestOutfitRemoveItemShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeOutfitService{
		removeItemFn: func(_ context.Context, _, _, _ string) error { return domain.ErrItemTransferPending },
	}
	w := removeOutfitItem(t, newOutfitHandler(svc), "o1", "i1", "user1")
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestOutfitRemoveItemShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitService{
		removeItemFn: func(_ context.Context, _, _, _ string) error { return errors.New("boom") },
	}
	w := removeOutfitItem(t, newOutfitHandler(svc), "o1", "i1", "user1")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitRemoveItemShouldReturn204WhenSuccessful(t *testing.T) {
	var called bool
	svc := &fakeOutfitService{
		removeItemFn: func(_ context.Context, callerID, outfitID, itemID string) error {
			require.Equal(t, "user1", callerID)
			require.Equal(t, "o1", outfitID)
			require.Equal(t, "i1", itemID)
			called = true
			return nil
		},
	}
	w := removeOutfitItem(t, newOutfitHandler(svc), "o1", "i1", "user1")
	require.Equal(t, http.StatusNoContent, w.Code)
	require.True(t, called)
}

// ---- UploadPhoto ----

func TestOutfitUploadPhotoShouldReturn500WhenCallerIDMissing(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/outfits/o1/photos", http.NoBody)
	req.SetPathValue("id", "o1")
	w := httptest.NewRecorder()
	h.UploadPhoto(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitUploadPhotoShouldReturn400WhenNoFile(t *testing.T) {
	ctx := middleware.WithUserID(t.Context(), "user1")
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/outfits/o1/photos", http.NoBody)
	req.SetPathValue("id", "o1")
	w := httptest.NewRecorder()
	newOutfitHandler(&fakeOutfitService{}).UploadPhoto(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOutfitUploadPhotoShouldReturn404WhenNotFound(t *testing.T) {
	svc := &fakeOutfitService{
		uploadPhotoFn: func(_ context.Context, _, _ string, _ io.Reader, _ string) (domain.OutfitPhoto, error) {
			return domain.OutfitPhoto{}, domain.ErrNotFound
		},
	}
	w := uploadOutfitPhoto(t, newOutfitHandler(svc), "o1", "user1", "photo.jpg", "data")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestOutfitUploadPhotoShouldReturn403WhenForbidden(t *testing.T) {
	svc := &fakeOutfitService{
		uploadPhotoFn: func(_ context.Context, _, _ string, _ io.Reader, _ string) (domain.OutfitPhoto, error) {
			return domain.OutfitPhoto{}, domain.ErrForbidden
		},
	}
	w := uploadOutfitPhoto(t, newOutfitHandler(svc), "o1", "user1", "photo.jpg", "data")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestOutfitUploadPhotoShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitService{
		uploadPhotoFn: func(_ context.Context, _, _ string, _ io.Reader, _ string) (domain.OutfitPhoto, error) {
			return domain.OutfitPhoto{}, errors.New("boom")
		},
	}
	w := uploadOutfitPhoto(t, newOutfitHandler(svc), "o1", "user1", "photo.jpg", "data")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitUploadPhotoShouldReturn201WithPhotoJSONWhenSuccessful(t *testing.T) {
	fixedTime := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	svc := &fakeOutfitService{
		uploadPhotoFn: func(_ context.Context, callerID, outfitID string, _ io.Reader, filename string) (domain.OutfitPhoto, error) {
			require.Equal(t, "user1", callerID)
			require.Equal(t, "o1", outfitID)
			require.Equal(t, "photo.jpg", filename)
			return domain.OutfitPhoto{
				ID:        "photo-abc",
				MediaKey:  "outfits/o1/uuid/photo.jpg",
				Position:  0,
				CreatedAt: fixedTime,
			}, nil
		},
	}
	w := uploadOutfitPhoto(t, newOutfitHandler(svc), "o1", "user1", "photo.jpg", "data")
	require.Equal(t, http.StatusCreated, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "photo-abc", got["id"])
	require.Equal(t, "outfits/o1/uuid/photo.jpg", got["media_key"])
	require.Equal(t, float64(0), got["position"])
}

// ---- DeletePhoto ----

func TestOutfitDeletePhotoShouldReturn500WhenCallerIDMissing(t *testing.T) {
	h := newOutfitHandler(&fakeOutfitService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/outfits/o1/photos/key", http.NoBody)
	req.SetPathValue("id", "o1")
	req.SetPathValue("key", "outfits/o1/uuid/img.jpg")
	w := httptest.NewRecorder()
	h.DeletePhoto(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitDeletePhotoShouldReturn404WhenNotFound(t *testing.T) {
	svc := &fakeOutfitService{
		deletePhotoFn: func(_ context.Context, _, _, _ string) error { return domain.ErrNotFound },
	}
	w := deleteOutfitPhoto(t, newOutfitHandler(svc), "o1", "outfits/o1/uuid/img.jpg", "user1")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestOutfitDeletePhotoShouldReturn403WhenForbidden(t *testing.T) {
	svc := &fakeOutfitService{
		deletePhotoFn: func(_ context.Context, _, _, _ string) error { return domain.ErrForbidden },
	}
	w := deleteOutfitPhoto(t, newOutfitHandler(svc), "o1", "outfits/o1/uuid/img.jpg", "user1")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestOutfitDeletePhotoShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitService{
		deletePhotoFn: func(_ context.Context, _, _, _ string) error { return errors.New("boom") },
	}
	w := deleteOutfitPhoto(t, newOutfitHandler(svc), "o1", "outfits/o1/uuid/img.jpg", "user1")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOutfitDeletePhotoShouldReturn204WhenSuccessful(t *testing.T) {
	var called bool
	svc := &fakeOutfitService{
		deletePhotoFn: func(_ context.Context, callerID, outfitID, mediaKey string) error {
			require.Equal(t, "user1", callerID)
			require.Equal(t, "o1", outfitID)
			require.Equal(t, "outfits/o1/uuid/img.jpg", mediaKey)
			called = true
			return nil
		},
	}
	w := deleteOutfitPhoto(t, newOutfitHandler(svc), "o1", "outfits/o1/uuid/img.jpg", "user1")
	require.Equal(t, http.StatusNoContent, w.Code)
	require.True(t, called)
}
