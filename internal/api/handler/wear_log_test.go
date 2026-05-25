package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/outfitte/backend/internal/api/handler"
	"github.com/outfitte/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeWearLogService struct {
	logWearFn      func(ctx context.Context, callerID, itemID string, wornOn time.Time, notes *string) (domain.WearLog, error)
	listByItemFn   func(ctx context.Context, callerID, itemID string) ([]domain.WearLog, error)
	deleteWearLogFn func(ctx context.Context, callerID, logID string) error
}

func (f *fakeWearLogService) LogWear(ctx context.Context, callerID, itemID string, wornOn time.Time, notes *string) (domain.WearLog, error) {
	if f.logWearFn != nil {
		return f.logWearFn(ctx, callerID, itemID, wornOn, notes)
	}
	return domain.WearLog{}, nil
}

func (f *fakeWearLogService) ListByItem(ctx context.Context, callerID, itemID string) ([]domain.WearLog, error) {
	if f.listByItemFn != nil {
		return f.listByItemFn(ctx, callerID, itemID)
	}
	return nil, nil
}

func (f *fakeWearLogService) DeleteWearLog(ctx context.Context, callerID, logID string) error {
	if f.deleteWearLogFn != nil {
		return f.deleteWearLogFn(ctx, callerID, logID)
	}
	return nil
}

// --- helpers ---

func newWearLogHandler(svc *fakeWearLogService) *handler.WearLogHandler {
	return handler.NewWearLogHandler(svc, slog.New(slog.DiscardHandler))
}

func postWearLog(t *testing.T, h *handler.WearLogHandler, itemID, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/items/"+itemID+"/wear-logs", strings.NewReader(body))
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.LogWear(w, req)
	return w
}

func listWearLogs(t *testing.T, h *handler.WearLogHandler, itemID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/items/"+itemID+"/wear-logs", http.NoBody)
	req.SetPathValue("id", itemID)
	w := httptest.NewRecorder()
	h.ListByItem(w, req)
	return w
}

func deleteWearLog(t *testing.T, h *handler.WearLogHandler, itemID, logID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/items/"+itemID+"/wear-logs/"+logID, http.NoBody)
	req.SetPathValue("id", itemID)
	req.SetPathValue("logID", logID)
	w := httptest.NewRecorder()
	h.DeleteWearLog(w, req)
	return w
}

// ── LogWear ───────────────────────────────────────────────────────────────────

func TestLogWearShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newWearLogHandler(&fakeWearLogService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/items/item-1/wear-logs", strings.NewReader(`{"worn_on":"2026-03-01"}`))
	req.SetPathValue("id", "item-1")
	w := httptest.NewRecorder()
	h.LogWear(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLogWearShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	h := newWearLogHandler(&fakeWearLogService{})

	w := postWearLog(t, h, "item-1", "user-1", `not-json`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogWearShouldReturn400WhenDateFormatIsInvalid(t *testing.T) {
	h := newWearLogHandler(&fakeWearLogService{})

	w := postWearLog(t, h, "item-1", "user-1", `{"worn_on":"not-a-date"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogWearShouldReturn422WhenDateIsInFuture(t *testing.T) {
	svc := &fakeWearLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.WearLog, error) {
			return domain.WearLog{}, domain.ErrFutureDateNotAllowed
		},
	}
	h := newWearLogHandler(svc)

	w := postWearLog(t, h, "item-1", "user-1", `{"worn_on":"2099-12-31"}`)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestLogWearShouldReturn404WhenItemNotFound(t *testing.T) {
	svc := &fakeWearLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.WearLog, error) {
			return domain.WearLog{}, domain.ErrNotFound
		},
	}
	h := newWearLogHandler(svc)

	w := postWearLog(t, h, "item-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestLogWearShouldReturn403WhenCallerDoesNotOwnItem(t *testing.T) {
	svc := &fakeWearLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.WearLog, error) {
			return domain.WearLog{}, domain.ErrForbidden
		},
	}
	h := newWearLogHandler(svc)

	w := postWearLog(t, h, "item-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestLogWearShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeWearLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.WearLog, error) {
			return domain.WearLog{}, domain.ErrItemTransferPending
		},
	}
	h := newWearLogHandler(svc)

	w := postWearLog(t, h, "item-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
}

func TestLogWearShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeWearLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.WearLog, error) {
			return domain.WearLog{}, errors.New("unexpected")
		},
	}
	h := newWearLogHandler(svc)

	w := postWearLog(t, h, "item-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLogWearShouldReturn201WithLogWhenSuccessful(t *testing.T) {
	var wearLog domain.WearLog
	wearLog.ID = "log-1"
	wearLog.ItemID = "item-1"
	wearLog.OwnerID = "user-1"
	wearLog.WornOn = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	note := "some notes"
	wearLog.Notes = &note
	wearLog.CreatedAt = time.Now().UTC()

	svc := &fakeWearLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.WearLog, error) {
			return wearLog, nil
		},
	}
	h := newWearLogHandler(svc)

	w := postWearLog(t, h, "item-1", "user-1", `{"worn_on":"2026-03-01","notes":"some notes"}`)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Equal(t, "log-1", resp["id"])
	require.Equal(t, "item-1", resp["item_id"])
	require.Equal(t, "2026-03-01", resp["worn_on"])
	require.Equal(t, "some notes", resp["notes"])
}

// ── ListByItem ────────────────────────────────────────────────────────────────

func TestListByItemShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newWearLogHandler(&fakeWearLogService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items/item-1/wear-logs", http.NoBody)
	req.SetPathValue("id", "item-1")
	w := httptest.NewRecorder()
	h.ListByItem(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListByItemShouldReturn404WhenItemNotFound(t *testing.T) {
	svc := &fakeWearLogService{
		listByItemFn: func(_ context.Context, _, _ string) ([]domain.WearLog, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := newWearLogHandler(svc)

	w := listWearLogs(t, h, "item-1", "user-1")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListByItemShouldReturn403WhenCallerDoesNotOwnItem(t *testing.T) {
	svc := &fakeWearLogService{
		listByItemFn: func(_ context.Context, _, _ string) ([]domain.WearLog, error) {
			return nil, domain.ErrForbidden
		},
	}
	h := newWearLogHandler(svc)

	w := listWearLogs(t, h, "item-1", "user-1")

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestListByItemShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeWearLogService{
		listByItemFn: func(_ context.Context, _, _ string) ([]domain.WearLog, error) {
			return nil, errors.New("unexpected")
		},
	}
	h := newWearLogHandler(svc)

	w := listWearLogs(t, h, "item-1", "user-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListByItemShouldReturn200WithAllLogs(t *testing.T) {
	var log1, log2 domain.WearLog
	log1.ID = "log-1"
	log1.ItemID = "item-1"
	log1.OwnerID = "user-1"
	log1.WornOn = time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	log2.ID = "log-2"
	log2.ItemID = "item-1"
	log2.OwnerID = "user-1"
	log2.WornOn = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	svc := &fakeWearLogService{
		listByItemFn: func(_ context.Context, _, _ string) ([]domain.WearLog, error) {
			return []domain.WearLog{log1, log2}, nil
		},
	}
	h := newWearLogHandler(svc)

	w := listWearLogs(t, h, "item-1", "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp, 2)
	require.Equal(t, "log-1", resp[0]["id"])
	require.Equal(t, "2026-03-10", resp[0]["worn_on"])
	require.Equal(t, "log-2", resp[1]["id"])
}

// ── DeleteWearLog ─────────────────────────────────────────────────────────────

func TestDeleteWearLogShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newWearLogHandler(&fakeWearLogService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/items/item-1/wear-logs/log-1", http.NoBody)
	req.SetPathValue("id", "item-1")
	req.SetPathValue("logID", "log-1")
	w := httptest.NewRecorder()
	h.DeleteWearLog(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteWearLogShouldReturn404WhenLogNotFound(t *testing.T) {
	svc := &fakeWearLogService{
		deleteWearLogFn: func(_ context.Context, _, _ string) error {
			return domain.ErrNotFound
		},
	}
	h := newWearLogHandler(svc)

	w := deleteWearLog(t, h, "item-1", "log-1", "user-1")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteWearLogShouldReturn403WhenCallerDoesNotOwnLog(t *testing.T) {
	svc := &fakeWearLogService{
		deleteWearLogFn: func(_ context.Context, _, _ string) error {
			return domain.ErrForbidden
		},
	}
	h := newWearLogHandler(svc)

	w := deleteWearLog(t, h, "item-1", "log-1", "user-1")

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteWearLogShouldReturn409WhenItemHasPendingTransfer(t *testing.T) {
	svc := &fakeWearLogService{
		deleteWearLogFn: func(_ context.Context, _, _ string) error {
			return domain.ErrItemTransferPending
		},
	}
	h := newWearLogHandler(svc)

	w := deleteWearLog(t, h, "item-1", "log-1", "user-1")

	require.Equal(t, http.StatusConflict, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	require.Equal(t, "item has a pending transfer", body["error"])
}

func TestDeleteWearLogShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeWearLogService{
		deleteWearLogFn: func(_ context.Context, _, _ string) error {
			return errors.New("unexpected")
		},
	}
	h := newWearLogHandler(svc)

	w := deleteWearLog(t, h, "item-1", "log-1", "user-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteWearLogShouldReturn204WhenSuccessful(t *testing.T) {
	h := newWearLogHandler(&fakeWearLogService{})

	w := deleteWearLog(t, h, "item-1", "log-1", "user-1")

	require.Equal(t, http.StatusNoContent, w.Code)
}
