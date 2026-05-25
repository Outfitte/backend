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

type fakeTransferService struct {
	createFn      func(ctx context.Context, callerID string, input service.CreateTransferInput) (service.TransferView, error)
	getFn         func(ctx context.Context, callerID, transferID string) (service.TransferView, error)
	listOutgoingFn func(ctx context.Context, callerID string, status *domain.TransferStatus) ([]service.TransferView, error)
	listIncomingFn func(ctx context.Context, callerID string, status *domain.TransferStatus) ([]service.TransferView, error)
	acceptFn      func(ctx context.Context, callerID, transferID string) (service.TransferView, error)
	rejectFn      func(ctx context.Context, callerID, transferID string) (service.TransferView, error)
	cancelFn      func(ctx context.Context, callerID, transferID string) (service.TransferView, error)
}

func (f *fakeTransferService) Create(ctx context.Context, callerID string, input service.CreateTransferInput) (service.TransferView, error) {
	if f.createFn != nil {
		return f.createFn(ctx, callerID, input)
	}
	return service.TransferView{}, nil
}

func (f *fakeTransferService) Get(ctx context.Context, callerID, transferID string) (service.TransferView, error) {
	if f.getFn != nil {
		return f.getFn(ctx, callerID, transferID)
	}
	return service.TransferView{}, nil
}

func (f *fakeTransferService) ListOutgoing(ctx context.Context, callerID string, status *domain.TransferStatus) ([]service.TransferView, error) {
	if f.listOutgoingFn != nil {
		return f.listOutgoingFn(ctx, callerID, status)
	}
	return nil, nil
}

func (f *fakeTransferService) ListIncoming(ctx context.Context, callerID string, status *domain.TransferStatus) ([]service.TransferView, error) {
	if f.listIncomingFn != nil {
		return f.listIncomingFn(ctx, callerID, status)
	}
	return nil, nil
}

func (f *fakeTransferService) Accept(ctx context.Context, callerID, transferID string) (service.TransferView, error) {
	if f.acceptFn != nil {
		return f.acceptFn(ctx, callerID, transferID)
	}
	return service.TransferView{}, nil
}

func (f *fakeTransferService) Reject(ctx context.Context, callerID, transferID string) (service.TransferView, error) {
	if f.rejectFn != nil {
		return f.rejectFn(ctx, callerID, transferID)
	}
	return service.TransferView{}, nil
}

func (f *fakeTransferService) Cancel(ctx context.Context, callerID, transferID string) (service.TransferView, error) {
	if f.cancelFn != nil {
		return f.cancelFn(ctx, callerID, transferID)
	}
	return service.TransferView{}, nil
}

// --- helpers ---

func newTransferHandler(svc *fakeTransferService) *handler.ItemTransferHandler {
	return handler.NewItemTransferHandler(svc, slog.New(slog.DiscardHandler))
}

func buildTransferView(t *testing.T) service.TransferView {
	t.Helper()
	var transfer domain.ItemTransfer
	transfer.ID = "transfer-1"
	transfer.Status = domain.TransferStatusPending
	transfer.TransferHistory = true
	transfer.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	var item domain.Item
	item.ID = "item-1"
	item.Name = "My Shirt"

	return service.TransferView{
		Transfer:  transfer,
		Item:      item,
		Sender:    service.UserSummary{ID: "user-1", Email: "sender@example.com"},
		Recipient: service.UserSummary{ID: "user-2", Email: "recipient@example.com"},
	}
}

func postTransfer(t *testing.T, h *handler.ItemTransferHandler, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/transfers", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, req)
	return w
}

func getTransfer(t *testing.T, h *handler.ItemTransferHandler, callerID, transferID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/transfers/"+transferID, nil)
	req.SetPathValue("id", transferID)
	w := httptest.NewRecorder()
	h.Get(w, req)
	return w
}

func listOutgoingTransfers(t *testing.T, h *handler.ItemTransferHandler, callerID string, query string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	url := "/transfers/outgoing"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	h.ListOutgoing(w, req)
	return w
}

func listIncomingTransfers(t *testing.T, h *handler.ItemTransferHandler, callerID string, query string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	url := "/transfers/incoming"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	h.ListIncoming(w, req)
	return w
}

func postTransferAction(t *testing.T, h *handler.ItemTransferHandler, action, callerID, transferID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/transfers/"+transferID+"/"+action, nil)
	req.SetPathValue("id", transferID)
	w := httptest.NewRecorder()
	switch action {
	case "accept":
		h.Accept(w, req)
	case "reject":
		h.Reject(w, req)
	case "cancel":
		h.Cancel(w, req)
	}
	return w
}

// =============================================================================
// Accept / Reject / Cancel tests
// =============================================================================

func TestTransferAcceptShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/transfers/transfer-1/accept", nil)
	req.SetPathValue("id", "transfer-1")
	w := httptest.NewRecorder()
	h.Accept(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTransferAcceptShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/transfers/transfer-1/accept", nil)
	req.SetPathValue("id", "transfer-1")
	w := httptest.NewRecorder()
	h.Accept(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferAcceptShouldReturn403WhenServiceReturnsForbiddenError(t *testing.T) {
	svc := &fakeTransferService{
		acceptFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrForbidden
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "accept", "user-2", "transfer-1")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestTransferAcceptShouldReturn404WhenServiceReturnsNotFoundError(t *testing.T) {
	svc := &fakeTransferService{
		acceptFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrNotFound
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "accept", "user-2", "transfer-ghost")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestTransferAcceptShouldReturn422WhenServiceReturnsValidationError(t *testing.T) {
	svc := &fakeTransferService{
		acceptFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrValidation
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "accept", "user-2", "transfer-1")
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestTransferAcceptShouldReturn500WhenServiceReturnsUnexpectedError(t *testing.T) {
	svc := &fakeTransferService{
		acceptFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrIO
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "accept", "user-2", "transfer-1")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferAcceptShouldReturn200WithTransferWhenSuccessful(t *testing.T) {
	view := buildTransferView(t)
	svc := &fakeTransferService{
		acceptFn: func(_ context.Context, callerID, transferID string) (service.TransferView, error) {
			require.Equal(t, "user-2", callerID)
			require.Equal(t, "transfer-1", transferID)
			return view, nil
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "accept", "user-2", "transfer-1")

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "transfer-1", body["id"])
}

func TestTransferRejectShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/transfers/transfer-1/reject", nil)
	req.SetPathValue("id", "transfer-1")
	w := httptest.NewRecorder()
	h.Reject(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTransferRejectShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/transfers/transfer-1/reject", nil)
	req.SetPathValue("id", "transfer-1")
	w := httptest.NewRecorder()
	h.Reject(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferRejectShouldReturn403WhenServiceReturnsForbiddenError(t *testing.T) {
	svc := &fakeTransferService{
		rejectFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrForbidden
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "reject", "user-2", "transfer-1")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestTransferRejectShouldReturn404WhenServiceReturnsNotFoundError(t *testing.T) {
	svc := &fakeTransferService{
		rejectFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrNotFound
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "reject", "user-2", "transfer-ghost")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestTransferRejectShouldReturn422WhenServiceReturnsValidationError(t *testing.T) {
	svc := &fakeTransferService{
		rejectFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrValidation
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "reject", "user-2", "transfer-1")
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestTransferRejectShouldReturn500WhenServiceReturnsUnexpectedError(t *testing.T) {
	svc := &fakeTransferService{
		rejectFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrIO
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "reject", "user-2", "transfer-1")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferRejectShouldReturn200WithTransferWhenSuccessful(t *testing.T) {
	view := buildTransferView(t)
	svc := &fakeTransferService{
		rejectFn: func(_ context.Context, callerID, transferID string) (service.TransferView, error) {
			require.Equal(t, "user-2", callerID)
			require.Equal(t, "transfer-1", transferID)
			return view, nil
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "reject", "user-2", "transfer-1")

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "transfer-1", body["id"])
}

func TestTransferCancelShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/transfers/transfer-1/cancel", nil)
	req.SetPathValue("id", "transfer-1")
	w := httptest.NewRecorder()
	h.Cancel(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTransferCancelShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/transfers/transfer-1/cancel", nil)
	req.SetPathValue("id", "transfer-1")
	w := httptest.NewRecorder()
	h.Cancel(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferCancelShouldReturn403WhenServiceReturnsForbiddenError(t *testing.T) {
	svc := &fakeTransferService{
		cancelFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrForbidden
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "cancel", "user-1", "transfer-1")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestTransferCancelShouldReturn404WhenServiceReturnsNotFoundError(t *testing.T) {
	svc := &fakeTransferService{
		cancelFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrNotFound
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "cancel", "user-1", "transfer-ghost")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestTransferCancelShouldReturn422WhenServiceReturnsValidationError(t *testing.T) {
	svc := &fakeTransferService{
		cancelFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrValidation
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "cancel", "user-1", "transfer-1")
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestTransferCancelShouldReturn500WhenServiceReturnsUnexpectedError(t *testing.T) {
	svc := &fakeTransferService{
		cancelFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrIO
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "cancel", "user-1", "transfer-1")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferCancelShouldReturn200WithTransferWhenSuccessful(t *testing.T) {
	view := buildTransferView(t)
	svc := &fakeTransferService{
		cancelFn: func(_ context.Context, callerID, transferID string) (service.TransferView, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "transfer-1", transferID)
			return view, nil
		},
	}
	h := newTransferHandler(svc)
	w := postTransferAction(t, h, "cancel", "user-1", "transfer-1")

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "transfer-1", body["id"])
}

// =============================================================================
// ListIncoming tests
// =============================================================================

func TestTransferListIncomingShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/transfers/incoming", nil)
	w := httptest.NewRecorder()
	h.ListIncoming(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTransferListIncomingShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/transfers/incoming", nil)
	w := httptest.NewRecorder()
	h.ListIncoming(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferListIncomingShouldReturn400WhenStatusQueryParamIsInvalid(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	w := listIncomingTransfers(t, h, "user-1", "status=bogus")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransferListIncomingShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeTransferService{
		listIncomingFn: func(_ context.Context, _ string, _ *domain.TransferStatus) ([]service.TransferView, error) {
			return nil, domain.ErrIO
		},
	}
	h := newTransferHandler(svc)
	w := listIncomingTransfers(t, h, "user-1", "")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferListIncomingShouldReturn200WithTransfersWhenSuccessful(t *testing.T) {
	view := buildTransferView(t)
	svc := &fakeTransferService{
		listIncomingFn: func(_ context.Context, callerID string, status *domain.TransferStatus) ([]service.TransferView, error) {
			require.Equal(t, "user-2", callerID)
			require.Nil(t, status)
			return []service.TransferView{view}, nil
		},
	}
	h := newTransferHandler(svc)
	w := listIncomingTransfers(t, h, "user-2", "")

	require.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body, 1)
	require.Equal(t, "transfer-1", body[0]["id"])
}

// =============================================================================
// ListOutgoing tests
// =============================================================================

func TestTransferListOutgoingShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/transfers/outgoing", nil)
	w := httptest.NewRecorder()
	h.ListOutgoing(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTransferListOutgoingShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/transfers/outgoing", nil)
	w := httptest.NewRecorder()
	h.ListOutgoing(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferListOutgoingShouldReturn400WhenStatusQueryParamIsInvalid(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	w := listOutgoingTransfers(t, h, "user-1", "status=unknown")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransferListOutgoingShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeTransferService{
		listOutgoingFn: func(_ context.Context, _ string, _ *domain.TransferStatus) ([]service.TransferView, error) {
			return nil, domain.ErrIO
		},
	}
	h := newTransferHandler(svc)
	w := listOutgoingTransfers(t, h, "user-1", "")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferListOutgoingShouldReturn200WithEmptyArrayWhenNoTransfers(t *testing.T) {
	svc := &fakeTransferService{
		listOutgoingFn: func(_ context.Context, _ string, _ *domain.TransferStatus) ([]service.TransferView, error) {
			return []service.TransferView{}, nil
		},
	}
	h := newTransferHandler(svc)
	w := listOutgoingTransfers(t, h, "user-1", "")

	require.Equal(t, http.StatusOK, w.Code)
	var body []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Empty(t, body)
}

func TestTransferListOutgoingShouldReturn200WithTransfersWhenSuccessful(t *testing.T) {
	view := buildTransferView(t)
	svc := &fakeTransferService{
		listOutgoingFn: func(_ context.Context, callerID string, status *domain.TransferStatus) ([]service.TransferView, error) {
			require.Equal(t, "user-1", callerID)
			require.Nil(t, status)
			return []service.TransferView{view}, nil
		},
	}
	h := newTransferHandler(svc)
	w := listOutgoingTransfers(t, h, "user-1", "")

	require.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Len(t, body, 1)
	require.Equal(t, "transfer-1", body[0]["id"])
}

func TestTransferListOutgoingShouldReturn200WithFilteredTransfersWhenStatusProvided(t *testing.T) {
	pending := domain.TransferStatusPending
	svc := &fakeTransferService{
		listOutgoingFn: func(_ context.Context, _ string, status *domain.TransferStatus) ([]service.TransferView, error) {
			require.NotNil(t, status)
			require.Equal(t, domain.TransferStatusPending, *status)
			return []service.TransferView{}, nil
		},
	}
	h := newTransferHandler(svc)
	_ = pending
	w := listOutgoingTransfers(t, h, "user-1", "status=pending")
	require.Equal(t, http.StatusOK, w.Code)
}

// =============================================================================
// Get tests
// =============================================================================

func TestTransferGetShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/transfers/transfer-1", nil)
	req.SetPathValue("id", "transfer-1")
	w := httptest.NewRecorder()
	h.Get(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTransferGetShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/transfers/transfer-1", nil)
	req.SetPathValue("id", "transfer-1")
	w := httptest.NewRecorder()
	h.Get(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferGetShouldReturn403WhenServiceReturnsForbiddenError(t *testing.T) {
	svc := &fakeTransferService{
		getFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrForbidden
		},
	}
	h := newTransferHandler(svc)
	w := getTransfer(t, h, "user-1", "transfer-1")
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestTransferGetShouldReturn404WhenServiceReturnsNotFoundError(t *testing.T) {
	svc := &fakeTransferService{
		getFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrNotFound
		},
	}
	h := newTransferHandler(svc)
	w := getTransfer(t, h, "user-1", "transfer-ghost")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestTransferGetShouldReturn500WhenServiceReturnsUnexpectedError(t *testing.T) {
	svc := &fakeTransferService{
		getFn: func(_ context.Context, _, _ string) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrIO
		},
	}
	h := newTransferHandler(svc)
	w := getTransfer(t, h, "user-1", "transfer-1")
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferGetShouldReturn200WithTransferWhenSuccessful(t *testing.T) {
	view := buildTransferView(t)
	svc := &fakeTransferService{
		getFn: func(_ context.Context, callerID, transferID string) (service.TransferView, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "transfer-1", transferID)
			return view, nil
		},
	}
	h := newTransferHandler(svc)
	w := getTransfer(t, h, "user-1", "transfer-1")

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "transfer-1", body["id"])
	require.Equal(t, "pending", body["status"])
}

// =============================================================================
// Create tests
// =============================================================================

func TestTransferCreateShouldReturn503WhenContextIsCancelled(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/transfers", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.Create(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTransferCreateShouldReturn500WhenCallerIDMissingFromContext(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/transfers", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.Create(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferCreateShouldReturn400WhenBodyIsInvalidJSON(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	w := postTransfer(t, h, "user-1", `not-json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransferCreateShouldReturn400WhenItemIDIsEmpty(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	w := postTransfer(t, h, "user-1", `{"item_id":"","recipient_id":"user-2"}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransferCreateShouldReturn400WhenRecipientIDIsEmpty(t *testing.T) {
	h := newTransferHandler(&fakeTransferService{})
	w := postTransfer(t, h, "user-1", `{"item_id":"item-1","recipient_id":""}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransferCreateShouldReturn422WhenServiceReturnsSelfTransferError(t *testing.T) {
	svc := &fakeTransferService{
		createFn: func(_ context.Context, _ string, _ service.CreateTransferInput) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrSelfTransfer
		},
	}
	h := newTransferHandler(svc)
	w := postTransfer(t, h, "user-1", `{"item_id":"item-1","recipient_id":"user-1"}`)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "cannot transfer to yourself", body["error"])
}

func TestTransferCreateShouldReturn409WhenServiceReturnsConflictError(t *testing.T) {
	svc := &fakeTransferService{
		createFn: func(_ context.Context, _ string, _ service.CreateTransferInput) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrConflict
		},
	}
	h := newTransferHandler(svc)
	w := postTransfer(t, h, "user-1", `{"item_id":"item-1","recipient_id":"user-2"}`)
	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
}

func TestTransferCreateShouldReturn404WhenServiceReturnsNotFoundError(t *testing.T) {
	svc := &fakeTransferService{
		createFn: func(_ context.Context, _ string, _ service.CreateTransferInput) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrNotFound
		},
	}
	h := newTransferHandler(svc)
	w := postTransfer(t, h, "user-1", `{"item_id":"item-1","recipient_id":"user-2"}`)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestTransferCreateShouldReturn403WhenServiceReturnsForbiddenError(t *testing.T) {
	svc := &fakeTransferService{
		createFn: func(_ context.Context, _ string, _ service.CreateTransferInput) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrForbidden
		},
	}
	h := newTransferHandler(svc)
	w := postTransfer(t, h, "user-1", `{"item_id":"item-1","recipient_id":"user-2"}`)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestTransferCreateShouldReturn422WhenServiceReturnsValidationError(t *testing.T) {
	svc := &fakeTransferService{
		createFn: func(_ context.Context, _ string, _ service.CreateTransferInput) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrValidation
		},
	}
	h := newTransferHandler(svc)
	w := postTransfer(t, h, "user-1", `{"item_id":"item-1","recipient_id":"user-2"}`)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestTransferCreateShouldReturn500WhenServiceReturnsUnexpectedError(t *testing.T) {
	svc := &fakeTransferService{
		createFn: func(_ context.Context, _ string, _ service.CreateTransferInput) (service.TransferView, error) {
			return service.TransferView{}, domain.ErrIO
		},
	}
	h := newTransferHandler(svc)
	w := postTransfer(t, h, "user-1", `{"item_id":"item-1","recipient_id":"user-2"}`)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTransferCreateShouldReturn201WithTransferWhenSuccessful(t *testing.T) {
	view := buildTransferView(t)
	svc := &fakeTransferService{
		createFn: func(_ context.Context, callerID string, input service.CreateTransferInput) (service.TransferView, error) {
			require.Equal(t, "user-1", callerID)
			require.Equal(t, "item-1", input.ItemID)
			require.Equal(t, "user-2", input.RecipientID)
			require.True(t, input.TransferHistory)
			return view, nil
		},
	}
	h := newTransferHandler(svc)
	w := postTransfer(t, h, "user-1", `{"item_id":"item-1","recipient_id":"user-2","transfer_history":true}`)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "transfer-1", body["id"])
	require.Equal(t, "pending", body["status"])
	require.Equal(t, true, body["transfer_history"])
	require.Equal(t, "2025-01-01T00:00:00Z", body["created_at"])
	require.Nil(t, body["decided_at"])

	item := body["item"].(map[string]any)
	require.Equal(t, "item-1", item["id"])

	sender := body["sender"].(map[string]any)
	require.Equal(t, "user-1", sender["id"])
	require.Equal(t, "sender@example.com", sender["email"])

	recipient := body["recipient"].(map[string]any)
	require.Equal(t, "user-2", recipient["id"])
	require.Equal(t, "recipient@example.com", recipient["email"])
}
