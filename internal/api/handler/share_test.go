package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/outfitte/backend/internal/api/handler"
	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/service"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeShareService struct {
	createFn          func(ctx context.Context, callerID string, input service.CreateShareInput) (domain.Share, error)
	listOutgoingFn    func(ctx context.Context, callerID string) ([]service.ShareView, error)
	listSharedWithMeFn func(ctx context.Context, callerID string) (service.SharedWithMeResult, error)
	revokeFn          func(ctx context.Context, callerID, shareID string) error
}

func (f *fakeShareService) Create(ctx context.Context, callerID string, input service.CreateShareInput) (domain.Share, error) {
	if f.createFn != nil {
		return f.createFn(ctx, callerID, input)
	}
	return domain.Share{}, nil
}

func (f *fakeShareService) ListOutgoing(ctx context.Context, callerID string) ([]service.ShareView, error) {
	if f.listOutgoingFn != nil {
		return f.listOutgoingFn(ctx, callerID)
	}
	return nil, nil
}

func (f *fakeShareService) ListSharedWithMe(ctx context.Context, callerID string) (service.SharedWithMeResult, error) {
	if f.listSharedWithMeFn != nil {
		return f.listSharedWithMeFn(ctx, callerID)
	}
	return service.SharedWithMeResult{}, nil
}

func (f *fakeShareService) Revoke(ctx context.Context, callerID, shareID string) error {
	if f.revokeFn != nil {
		return f.revokeFn(ctx, callerID, shareID)
	}
	return nil
}

// --- helpers ---

func newShareHandler(svc *fakeShareService) *handler.ShareHandler {
	return handler.NewShareHandler(svc, slog.New(slog.DiscardHandler))
}

func postShare(t *testing.T, h *handler.ShareHandler, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/shares", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, req)
	return w
}

func getShares(t *testing.T, h *handler.ShareHandler, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/shares", nil)
	w := httptest.NewRecorder()
	h.ListOutgoing(w, req)
	return w
}

func getSharesWithMe(t *testing.T, h *handler.ShareHandler, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/shares/with-me", nil)
	w := httptest.NewRecorder()
	h.ListSharedWithMe(w, req)
	return w
}

func deleteShare(t *testing.T, h *handler.ShareHandler, callerID, shareID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/shares/"+shareID, nil)
	req.SetPathValue("id", shareID)
	w := httptest.NewRecorder()
	h.Revoke(w, req)
	return w
}

func noCallerCtx(t *testing.T) context.Context {
	t.Helper()
	return t.Context()
}

// --- Create tests ---

func TestShareCreateShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newShareHandler(&fakeShareService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/shares", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.Create(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestShareCreateShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newShareHandler(&fakeShareService{})
	req := httptest.NewRequestWithContext(noCallerCtx(t), http.MethodPost, "/shares", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.Create(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShareCreateShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	h := newShareHandler(&fakeShareService{})

	w := postShare(t, h, "user-1", `not-json`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShareCreateShouldReturn400WhenRecipientIDIsEmpty(t *testing.T) {
	h := newShareHandler(&fakeShareService{})

	w := postShare(t, h, "user-1", `{"recipient_id":"","target_type":"item","target_id":"item-1"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShareCreateShouldReturn400WhenTargetIDIsEmpty(t *testing.T) {
	h := newShareHandler(&fakeShareService{})

	w := postShare(t, h, "user-1", `{"recipient_id":"user-2","target_type":"item","target_id":""}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShareCreateShouldReturn400WhenTargetTypeIsInvalid(t *testing.T) {
	h := newShareHandler(&fakeShareService{})

	w := postShare(t, h, "user-1", `{"recipient_id":"user-2","target_type":"unknown","target_id":"item-1"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShareCreateShouldReturn422WhenServiceReturnsSelfShareError(t *testing.T) {
	svc := &fakeShareService{
		createFn: func(_ context.Context, _ string, _ service.CreateShareInput) (domain.Share, error) {
			return domain.Share{}, domain.ErrSelfShare
		},
	}
	h := newShareHandler(svc)

	w := postShare(t, h, "user-1", `{"recipient_id":"user-1","target_type":"item","target_id":"item-1"}`)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "cannot share with yourself", body["error"])
}

func TestShareCreateShouldReturn409WhenServiceReturnsDuplicateShareError(t *testing.T) {
	svc := &fakeShareService{
		createFn: func(_ context.Context, _ string, _ service.CreateShareInput) (domain.Share, error) {
			return domain.Share{}, domain.ErrDuplicateShare
		},
	}
	h := newShareHandler(svc)

	w := postShare(t, h, "user-1", `{"recipient_id":"user-2","target_type":"item","target_id":"item-1"}`)

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestShareCreateShouldReturn404WhenServiceReturnsNotFoundError(t *testing.T) {
	svc := &fakeShareService{
		createFn: func(_ context.Context, _ string, _ service.CreateShareInput) (domain.Share, error) {
			return domain.Share{}, domain.ErrNotFound
		},
	}
	h := newShareHandler(svc)

	w := postShare(t, h, "user-1", `{"recipient_id":"user-2","target_type":"item","target_id":"item-1"}`)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestShareCreateShouldReturn403WhenServiceReturnsForbiddenError(t *testing.T) {
	svc := &fakeShareService{
		createFn: func(_ context.Context, _ string, _ service.CreateShareInput) (domain.Share, error) {
			return domain.Share{}, domain.ErrForbidden
		},
	}
	h := newShareHandler(svc)

	w := postShare(t, h, "user-1", `{"recipient_id":"user-2","target_type":"item","target_id":"item-1"}`)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestShareCreateShouldReturn500WhenServiceReturnsUnexpectedError(t *testing.T) {
	svc := &fakeShareService{
		createFn: func(_ context.Context, _ string, _ service.CreateShareInput) (domain.Share, error) {
			return domain.Share{}, domain.ErrIO
		},
	}
	h := newShareHandler(svc)

	w := postShare(t, h, "user-1", `{"recipient_id":"user-2","target_type":"item","target_id":"item-1"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShareCreateShouldReturn409WhenServiceReturnsItemTransferPendingError(t *testing.T) {
	svc := &fakeShareService{
		createFn: func(_ context.Context, _ string, _ service.CreateShareInput) (domain.Share, error) {
			return domain.Share{}, domain.ErrItemTransferPending
		},
	}
	h := newShareHandler(svc)

	w := postShare(t, h, "user-1", `{"recipient_id":"user-2","target_type":"item","target_id":"item-1"}`)

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
}

func TestShareCreateShouldReturn201WithShareWhenSuccessful(t *testing.T) {
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	var share domain.Share
	share.ID = "share-1"
	share.RecipientID = "user-2"
	share.TargetType = domain.ShareTargetItem
	share.TargetID = "item-1"
	share.CreatedAt = createdAt

	svc := &fakeShareService{
		createFn: func(_ context.Context, callerID string, input service.CreateShareInput) (domain.Share, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "user-2", input.RecipientID)
			require.Equal(t, domain.ShareTargetItem, input.TargetType)
			require.Equal(t, "item-1", input.TargetID)
			return share, nil
		},
	}
	h := newShareHandler(svc)

	w := postShare(t, h, "user-1", `{"recipient_id":"user-2","target_type":"item","target_id":"item-1"}`)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "share-1", body["id"])
	require.Equal(t, "user-2", body["recipient_id"])
	require.Equal(t, "item", body["target_type"])
	require.Equal(t, "item-1", body["target_id"])
	require.Equal(t, "2025-01-01T00:00:00Z", body["created_at"])
}

// --- ListOutgoing tests ---

func TestShareListOutgoingShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newShareHandler(&fakeShareService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/shares", nil)
	w := httptest.NewRecorder()
	h.ListOutgoing(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestShareListOutgoingShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newShareHandler(&fakeShareService{})
	req := httptest.NewRequestWithContext(noCallerCtx(t), http.MethodGet, "/shares", nil)
	w := httptest.NewRecorder()
	h.ListOutgoing(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShareListOutgoingShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeShareService{
		listOutgoingFn: func(_ context.Context, _ string) ([]service.ShareView, error) {
			return nil, domain.ErrIO
		},
	}
	h := newShareHandler(svc)

	w := getShares(t, h, "user-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShareListOutgoingShouldReturn200WithEmptyArrayWhenNoShares(t *testing.T) {
	svc := &fakeShareService{
		listOutgoingFn: func(_ context.Context, _ string) ([]service.ShareView, error) {
			return []service.ShareView{}, nil
		},
	}
	h := newShareHandler(svc)

	w := getShares(t, h, "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Empty(t, body)
}

func TestShareListOutgoingShouldReturn200WithShareViewsWhenSharesExist(t *testing.T) {
	createdAt := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	var share domain.Share
	share.ID = "share-1"
	share.RecipientID = "user-2"
	share.TargetType = domain.ShareTargetItem
	share.TargetID = "item-1"
	share.CreatedAt = createdAt

	svc := &fakeShareService{
		listOutgoingFn: func(_ context.Context, callerID string) ([]service.ShareView, error) {
			require.Equal(t, "user-1", callerID)
			return []service.ShareView{
				{
					Share:     share,
					Recipient: service.UserSummary{ID: "user-2", Email: "bob@example.com"},
				},
			}, nil
		},
	}
	h := newShareHandler(svc)

	w := getShares(t, h, "user-1")

	require.Equal(t, http.StatusOK, w.Code)

	var body []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body, 1)

	require.Equal(t, "share-1", body[0]["id"])
	recipient := body[0]["recipient"].(map[string]any)
	require.Equal(t, "user-2", recipient["id"])
	require.Equal(t, "bob@example.com", recipient["email"])
	require.Equal(t, "item", body[0]["target_type"])
	require.Equal(t, "item-1", body[0]["target_id"])
	require.Equal(t, "2025-03-01T00:00:00Z", body[0]["created_at"])
}

// --- ListSharedWithMe tests ---

func TestShareListSharedWithMeShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newShareHandler(&fakeShareService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/shares/with-me", nil)
	w := httptest.NewRecorder()
	h.ListSharedWithMe(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestShareListSharedWithMeShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newShareHandler(&fakeShareService{})
	req := httptest.NewRequestWithContext(noCallerCtx(t), http.MethodGet, "/shares/with-me", nil)
	w := httptest.NewRecorder()
	h.ListSharedWithMe(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShareListSharedWithMeShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeShareService{
		listSharedWithMeFn: func(_ context.Context, _ string) (service.SharedWithMeResult, error) {
			return service.SharedWithMeResult{}, domain.ErrIO
		},
	}
	h := newShareHandler(svc)

	w := getSharesWithMe(t, h, "user-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShareListSharedWithMeShouldReturn200WithEmptyResultWhenNothingShared(t *testing.T) {
	svc := &fakeShareService{
		listSharedWithMeFn: func(_ context.Context, _ string) (service.SharedWithMeResult, error) {
			return service.SharedWithMeResult{
				Items:     []service.SharedEntity[domain.Item]{},
				Outfits:   []service.SharedEntity[domain.Outfit]{},
				Locations: []service.SharedLocation{},
			}, nil
		},
	}
	h := newShareHandler(svc)

	w := getSharesWithMe(t, h, "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Empty(t, body["items"])
	require.Empty(t, body["outfits"])
	require.Empty(t, body["locations"])
}

func TestShareListSharedWithMeShouldReturn200WithHydratedDataWhenSharesExist(t *testing.T) {
	var item domain.Item
	item.ID = "item-1"
	item.Name = "My Shirt"

	var outfit domain.Outfit
	outfit.ID = "outfit-1"

	var loc domain.Location
	loc.ID = "loc-1"
	loc.Label = "Wardrobe"

	sharedBy := service.UserSummary{ID: "user-2", Email: "alice@example.com"}

	svc := &fakeShareService{
		listSharedWithMeFn: func(_ context.Context, callerID string) (service.SharedWithMeResult, error) {
			require.Equal(t, "user-1", callerID)
			return service.SharedWithMeResult{
				Items:   []service.SharedEntity[domain.Item]{{Entity: item, SharedBy: sharedBy}},
				Outfits: []service.SharedEntity[domain.Outfit]{{Entity: outfit, SharedBy: sharedBy}},
				Locations: []service.SharedLocation{{
					Location: loc,
					Items:    []domain.Item{item},
					SharedBy: sharedBy,
				}},
			}, nil
		},
	}
	h := newShareHandler(svc)

	w := getSharesWithMe(t, h, "user-1")

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

	items := body["items"].([]any)
	require.Len(t, items, 1)
	itemEntry := items[0].(map[string]any)
	require.Equal(t, "item-1", itemEntry["id"])
	sharedByField := itemEntry["shared_by"].(map[string]any)
	require.Equal(t, "user-2", sharedByField["id"])
	require.Equal(t, "alice@example.com", sharedByField["email"])

	outfits := body["outfits"].([]any)
	require.Len(t, outfits, 1)
	outfitEntry := outfits[0].(map[string]any)
	require.Equal(t, "outfit-1", outfitEntry["id"])

	locations := body["locations"].([]any)
	require.Len(t, locations, 1)
	locEntry := locations[0].(map[string]any)
	location := locEntry["location"].(map[string]any)
	require.Equal(t, "loc-1", location["id"])
	locItems := locEntry["items"].([]any)
	require.Len(t, locItems, 1)
	locSharedBy := locEntry["shared_by"].(map[string]any)
	require.Equal(t, "user-2", locSharedBy["id"])
}

// --- Revoke tests ---

func TestShareRevokeShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newShareHandler(&fakeShareService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/shares/share-1", nil)
	req.SetPathValue("id", "share-1")
	w := httptest.NewRecorder()
	h.Revoke(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestShareRevokeShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newShareHandler(&fakeShareService{})
	req := httptest.NewRequestWithContext(noCallerCtx(t), http.MethodDelete, "/shares/share-1", nil)
	req.SetPathValue("id", "share-1")
	w := httptest.NewRecorder()
	h.Revoke(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShareRevokeShouldReturn403WhenServiceReturnsForbiddenError(t *testing.T) {
	svc := &fakeShareService{
		revokeFn: func(_ context.Context, _, _ string) error {
			return domain.ErrForbidden
		},
	}
	h := newShareHandler(svc)

	w := deleteShare(t, h, "user-1", "share-1")

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestShareRevokeShouldReturn404WhenServiceReturnsNotFoundError(t *testing.T) {
	svc := &fakeShareService{
		revokeFn: func(_ context.Context, _, _ string) error {
			return domain.ErrNotFound
		},
	}
	h := newShareHandler(svc)

	w := deleteShare(t, h, "user-1", "share-ghost")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestShareRevokeShouldReturn500WhenServiceReturnsUnexpectedError(t *testing.T) {
	svc := &fakeShareService{
		revokeFn: func(_ context.Context, _, _ string) error {
			return domain.ErrIO
		},
	}
	h := newShareHandler(svc)

	w := deleteShare(t, h, "user-1", "share-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestShareRevokeShouldReturn409WhenServiceReturnsItemTransferPendingError(t *testing.T) {
	svc := &fakeShareService{
		revokeFn: func(_ context.Context, _, _ string) error {
			return domain.ErrItemTransferPending
		},
	}
	h := newShareHandler(svc)

	w := deleteShare(t, h, "user-1", "share-1")

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
}

func TestShareRevokeShouldReturn204WhenSuccessful(t *testing.T) {
	svc := &fakeShareService{
		revokeFn: func(_ context.Context, callerID, shareID string) error {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "share-1", shareID)
			return nil
		},
	}
	h := newShareHandler(svc)

	w := deleteShare(t, h, "user-1", "share-1")

	require.Equal(t, http.StatusNoContent, w.Code)
}
