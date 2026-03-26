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

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fake ---

type fakeOutfitLogService struct {
	logWearFn         func(ctx context.Context, callerID, outfitID string, wornOn time.Time, notes *string) (domain.OutfitLog, error)
	listByOutfitFn    func(ctx context.Context, callerID, outfitID string) ([]domain.OutfitLog, error)
	listByDateRangeFn func(ctx context.Context, callerID string, from, to time.Time) ([]domain.OutfitLog, error)
	updateDateFn      func(ctx context.Context, callerID, outfitLogID string, newDate time.Time) (domain.OutfitLog, error)
	deleteFn          func(ctx context.Context, callerID, outfitLogID string) error
}

func (f *fakeOutfitLogService) LogWear(ctx context.Context, callerID, outfitID string, wornOn time.Time, notes *string) (domain.OutfitLog, error) {
	if f.logWearFn != nil {
		return f.logWearFn(ctx, callerID, outfitID, wornOn, notes)
	}
	return domain.OutfitLog{}, nil
}

func (f *fakeOutfitLogService) ListByOutfit(ctx context.Context, callerID, outfitID string) ([]domain.OutfitLog, error) {
	if f.listByOutfitFn != nil {
		return f.listByOutfitFn(ctx, callerID, outfitID)
	}
	return nil, nil
}

func (f *fakeOutfitLogService) ListByDateRange(ctx context.Context, callerID string, from, to time.Time) ([]domain.OutfitLog, error) {
	if f.listByDateRangeFn != nil {
		return f.listByDateRangeFn(ctx, callerID, from, to)
	}
	return nil, nil
}

func (f *fakeOutfitLogService) UpdateDate(ctx context.Context, callerID, outfitLogID string, newDate time.Time) (domain.OutfitLog, error) {
	if f.updateDateFn != nil {
		return f.updateDateFn(ctx, callerID, outfitLogID, newDate)
	}
	return domain.OutfitLog{}, nil
}

func (f *fakeOutfitLogService) Delete(ctx context.Context, callerID, outfitLogID string) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, callerID, outfitLogID)
	}
	return nil
}

// --- helpers ---

func newOutfitLogHandler(svc *fakeOutfitLogService) *handler.OutfitLogHandler {
	return handler.NewOutfitLogHandler(svc, slog.New(slog.DiscardHandler))
}

// --- request helpers ---

func postOutfitLog(t *testing.T, h *handler.OutfitLogHandler, outfitID, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/outfits/"+outfitID+"/logs", strings.NewReader(body))
	req.SetPathValue("id", outfitID)
	w := httptest.NewRecorder()
	h.LogWear(w, req)
	return w
}

func listOutfitLogs(t *testing.T, h *handler.OutfitLogHandler, outfitID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/outfits/"+outfitID+"/logs", http.NoBody)
	req.SetPathValue("id", outfitID)
	w := httptest.NewRecorder()
	h.ListByOutfit(w, req)
	return w
}

func patchOutfitLogDate(t *testing.T, h *handler.OutfitLogHandler, outfitID, logID, callerID, body string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/outfits/"+outfitID+"/logs/"+logID, strings.NewReader(body))
	req.SetPathValue("id", outfitID)
	req.SetPathValue("logID", logID)
	w := httptest.NewRecorder()
	h.UpdateDate(w, req)
	return w
}

func deleteOutfitLog(t *testing.T, h *handler.OutfitLogHandler, outfitID, logID, callerID string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/outfits/"+outfitID+"/logs/"+logID, http.NoBody)
	req.SetPathValue("id", outfitID)
	req.SetPathValue("logID", logID)
	w := httptest.NewRecorder()
	h.Delete(w, req)
	return w
}

func listOutfitLogsByDateRange(t *testing.T, h *handler.OutfitLogHandler, callerID, from, to string) *httptest.ResponseRecorder {
	t.Helper()
	ctx := ctxWithUser(t, callerID)
	url := "/outfit-logs"
	if from != "" || to != "" {
		url += "?from=" + from + "&to=" + to
	}
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	w := httptest.NewRecorder()
	h.ListByDateRange(w, req)
	return w
}

// ── LogWear ───────────────────────────────────────────────────────────────────

func TestLogOutfitWearShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/outfits/outfit-1/logs", http.NoBody)
	req.SetPathValue("id", "outfit-1")
	w := httptest.NewRecorder()
	h.LogWear(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLogOutfitWearShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	w := postOutfitLog(t, h, "outfit-1", "user-1", `not-json`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogOutfitWearShouldReturn400WhenDateFormatIsInvalid(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	w := postOutfitLog(t, h, "outfit-1", "user-1", `{"worn_on":"not-a-date"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogOutfitWearShouldReturn422WhenDateIsInFuture(t *testing.T) {
	svc := &fakeOutfitLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.OutfitLog, error) {
			return domain.OutfitLog{}, domain.ErrFutureDateNotAllowed
		},
	}
	h := newOutfitLogHandler(svc)

	w := postOutfitLog(t, h, "outfit-1", "user-1", `{"worn_on":"2099-12-31"}`)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestLogOutfitWearShouldReturn404WhenOutfitNotFound(t *testing.T) {
	svc := &fakeOutfitLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.OutfitLog, error) {
			return domain.OutfitLog{}, domain.ErrNotFound
		},
	}
	h := newOutfitLogHandler(svc)

	w := postOutfitLog(t, h, "outfit-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestLogOutfitWearShouldReturn403WhenCallerDoesNotOwnOutfit(t *testing.T) {
	svc := &fakeOutfitLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.OutfitLog, error) {
			return domain.OutfitLog{}, domain.ErrForbidden
		},
	}
	h := newOutfitLogHandler(svc)

	w := postOutfitLog(t, h, "outfit-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestLogOutfitWearShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.OutfitLog, error) {
			return domain.OutfitLog{}, errors.New("unexpected")
		},
	}
	h := newOutfitLogHandler(svc)

	w := postOutfitLog(t, h, "outfit-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLogOutfitWearShouldReturn201WithEmptyWearLogIDsWhenLogHasNone(t *testing.T) {
	var outfitLog domain.OutfitLog
	outfitLog.ID = "log-1"
	outfitLog.OutfitID = "outfit-1"
	outfitLog.OwnerID = "user-1"
	outfitLog.WornOn = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	// WearLogIDs intentionally left nil (zero value)

	svc := &fakeOutfitLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.OutfitLog, error) {
			return outfitLog, nil
		},
	}
	h := newOutfitLogHandler(svc)

	w := postOutfitLog(t, h, "outfit-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	wearLogIDs := resp["wear_log_ids"].([]any)
	require.Empty(t, wearLogIDs)
}

func TestLogOutfitWearShouldReturn201WithLogWhenSuccessful(t *testing.T) {
	var outfitLog domain.OutfitLog
	outfitLog.ID = "log-1"
	outfitLog.OutfitID = "outfit-1"
	outfitLog.OwnerID = "user-1"
	outfitLog.WornOn = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	note := "some notes"
	outfitLog.Notes = &note
	outfitLog.WearLogIDs = []string{"wl-1", "wl-2"}
	outfitLog.CreatedAt = time.Now().UTC()

	svc := &fakeOutfitLogService{
		logWearFn: func(_ context.Context, _, _ string, _ time.Time, _ *string) (domain.OutfitLog, error) {
			return outfitLog, nil
		},
	}
	h := newOutfitLogHandler(svc)

	w := postOutfitLog(t, h, "outfit-1", "user-1", `{"worn_on":"2026-03-01","notes":"some notes"}`)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Equal(t, "log-1", resp["id"])
	require.Equal(t, "outfit-1", resp["outfit_id"])
	require.Equal(t, "2026-03-01", resp["worn_on"])
	require.Equal(t, "some notes", resp["notes"])
	wearLogIDs := resp["wear_log_ids"].([]any)
	require.Len(t, wearLogIDs, 2)
}

// ── ListByOutfit ──────────────────────────────────────────────────────────────

func TestListOutfitLogsShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/outfits/outfit-1/logs", http.NoBody)
	req.SetPathValue("id", "outfit-1")
	w := httptest.NewRecorder()
	h.ListByOutfit(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListOutfitLogsShouldReturn404WhenOutfitNotFound(t *testing.T) {
	svc := &fakeOutfitLogService{
		listByOutfitFn: func(_ context.Context, _, _ string) ([]domain.OutfitLog, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := newOutfitLogHandler(svc)

	w := listOutfitLogs(t, h, "outfit-1", "user-1")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListOutfitLogsShouldReturn403WhenCallerDoesNotOwnOutfit(t *testing.T) {
	svc := &fakeOutfitLogService{
		listByOutfitFn: func(_ context.Context, _, _ string) ([]domain.OutfitLog, error) {
			return nil, domain.ErrForbidden
		},
	}
	h := newOutfitLogHandler(svc)

	w := listOutfitLogs(t, h, "outfit-1", "user-1")

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestListOutfitLogsShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitLogService{
		listByOutfitFn: func(_ context.Context, _, _ string) ([]domain.OutfitLog, error) {
			return nil, errors.New("unexpected")
		},
	}
	h := newOutfitLogHandler(svc)

	w := listOutfitLogs(t, h, "outfit-1", "user-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListOutfitLogsShouldReturn200WithAllLogsWhenSuccessful(t *testing.T) {
	var log1, log2 domain.OutfitLog
	log1.ID = "log-1"
	log1.OutfitID = "outfit-1"
	log1.OwnerID = "user-1"
	log1.WornOn = time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	log1.WearLogIDs = []string{"wl-1"}
	log2.ID = "log-2"
	log2.OutfitID = "outfit-1"
	log2.OwnerID = "user-1"
	log2.WornOn = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	log2.WearLogIDs = []string{}

	svc := &fakeOutfitLogService{
		listByOutfitFn: func(_ context.Context, _, _ string) ([]domain.OutfitLog, error) {
			return []domain.OutfitLog{log1, log2}, nil
		},
	}
	h := newOutfitLogHandler(svc)

	w := listOutfitLogs(t, h, "outfit-1", "user-1")

	require.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp, 2)
	require.Equal(t, "log-1", resp[0]["id"])
	require.Equal(t, "2026-03-10", resp[0]["worn_on"])
	require.Equal(t, "log-2", resp[1]["id"])
}

// ── UpdateDate ────────────────────────────────────────────────────────────────

func TestUpdateOutfitLogDateShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/outfits/outfit-1/logs/log-1", http.NoBody)
	req.SetPathValue("id", "outfit-1")
	req.SetPathValue("logID", "log-1")
	w := httptest.NewRecorder()
	h.UpdateDate(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateOutfitLogDateShouldReturn400WhenBodyIsInvalid(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	w := patchOutfitLogDate(t, h, "outfit-1", "log-1", "user-1", `not-json`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOutfitLogDateShouldReturn400WhenDateFormatIsInvalid(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	w := patchOutfitLogDate(t, h, "outfit-1", "log-1", "user-1", `{"worn_on":"bad-date"}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOutfitLogDateShouldReturn422WhenDateIsInFuture(t *testing.T) {
	svc := &fakeOutfitLogService{
		updateDateFn: func(_ context.Context, _, _ string, _ time.Time) (domain.OutfitLog, error) {
			return domain.OutfitLog{}, domain.ErrFutureDateNotAllowed
		},
	}
	h := newOutfitLogHandler(svc)

	w := patchOutfitLogDate(t, h, "outfit-1", "log-1", "user-1", `{"worn_on":"2099-12-31"}`)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateOutfitLogDateShouldReturn404WhenLogNotFound(t *testing.T) {
	svc := &fakeOutfitLogService{
		updateDateFn: func(_ context.Context, _, _ string, _ time.Time) (domain.OutfitLog, error) {
			return domain.OutfitLog{}, domain.ErrNotFound
		},
	}
	h := newOutfitLogHandler(svc)

	w := patchOutfitLogDate(t, h, "outfit-1", "log-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateOutfitLogDateShouldReturn403WhenCallerDoesNotOwnLog(t *testing.T) {
	svc := &fakeOutfitLogService{
		updateDateFn: func(_ context.Context, _, _ string, _ time.Time) (domain.OutfitLog, error) {
			return domain.OutfitLog{}, domain.ErrForbidden
		},
	}
	h := newOutfitLogHandler(svc)

	w := patchOutfitLogDate(t, h, "outfit-1", "log-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateOutfitLogDateShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitLogService{
		updateDateFn: func(_ context.Context, _, _ string, _ time.Time) (domain.OutfitLog, error) {
			return domain.OutfitLog{}, errors.New("unexpected")
		},
	}
	h := newOutfitLogHandler(svc)

	w := patchOutfitLogDate(t, h, "outfit-1", "log-1", "user-1", `{"worn_on":"2026-03-01"}`)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateOutfitLogDateShouldReturn200WithUpdatedLogWhenSuccessful(t *testing.T) {
	var outfitLog domain.OutfitLog
	outfitLog.ID = "log-1"
	outfitLog.OutfitID = "outfit-1"
	outfitLog.OwnerID = "user-1"
	outfitLog.WornOn = time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	outfitLog.WearLogIDs = []string{}
	outfitLog.CreatedAt = time.Now().UTC()

	svc := &fakeOutfitLogService{
		updateDateFn: func(_ context.Context, _, _ string, _ time.Time) (domain.OutfitLog, error) {
			return outfitLog, nil
		},
	}
	h := newOutfitLogHandler(svc)

	w := patchOutfitLogDate(t, h, "outfit-1", "log-1", "user-1", `{"worn_on":"2026-03-15"}`)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Equal(t, "log-1", resp["id"])
	require.Equal(t, "2026-03-15", resp["worn_on"])
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestDeleteOutfitLogShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/outfits/outfit-1/logs/log-1", http.NoBody)
	req.SetPathValue("id", "outfit-1")
	req.SetPathValue("logID", "log-1")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteOutfitLogShouldReturn404WhenLogNotFound(t *testing.T) {
	svc := &fakeOutfitLogService{
		deleteFn: func(_ context.Context, _, _ string) error {
			return domain.ErrNotFound
		},
	}
	h := newOutfitLogHandler(svc)

	w := deleteOutfitLog(t, h, "outfit-1", "log-1", "user-1")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteOutfitLogShouldReturn403WhenCallerDoesNotOwnLog(t *testing.T) {
	svc := &fakeOutfitLogService{
		deleteFn: func(_ context.Context, _, _ string) error {
			return domain.ErrForbidden
		},
	}
	h := newOutfitLogHandler(svc)

	w := deleteOutfitLog(t, h, "outfit-1", "log-1", "user-1")

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestDeleteOutfitLogShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitLogService{
		deleteFn: func(_ context.Context, _, _ string) error {
			return errors.New("unexpected")
		},
	}
	h := newOutfitLogHandler(svc)

	w := deleteOutfitLog(t, h, "outfit-1", "log-1", "user-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteOutfitLogShouldReturn204WhenSuccessful(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	w := deleteOutfitLog(t, h, "outfit-1", "log-1", "user-1")

	require.Equal(t, http.StatusNoContent, w.Code)
}

// ── ListByDateRange ───────────────────────────────────────────────────────────

func TestListOutfitLogsByDateRangeShouldReturn500WhenCallerIDIsMissingFromContext(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/outfit-logs?from=2026-03-01&to=2026-03-31", http.NoBody)
	w := httptest.NewRecorder()
	h.ListByDateRange(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListOutfitLogsByDateRangeShouldReturn400WhenFromIsMissing(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/outfit-logs?to=2026-03-31", http.NoBody)
	w := httptest.NewRecorder()
	h.ListByDateRange(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListOutfitLogsByDateRangeShouldReturn400WhenToIsMissing(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	ctx := ctxWithUser(t, "user-1")
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/outfit-logs?from=2026-03-01", http.NoBody)
	w := httptest.NewRecorder()
	h.ListByDateRange(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListOutfitLogsByDateRangeShouldReturn400WhenFromDateFormatIsInvalid(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	w := listOutfitLogsByDateRange(t, h, "user-1", "bad-date", "2026-03-31")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListOutfitLogsByDateRangeShouldReturn400WhenToDateFormatIsInvalid(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	w := listOutfitLogsByDateRange(t, h, "user-1", "2026-03-01", "bad-date")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListOutfitLogsByDateRangeShouldReturn400WhenFromIsAfterTo(t *testing.T) {
	h := newOutfitLogHandler(&fakeOutfitLogService{})

	w := listOutfitLogsByDateRange(t, h, "user-1", "2026-03-31", "2026-03-01")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListOutfitLogsByDateRangeShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeOutfitLogService{
		listByDateRangeFn: func(_ context.Context, _ string, _, _ time.Time) ([]domain.OutfitLog, error) {
			return nil, errors.New("unexpected")
		},
	}
	h := newOutfitLogHandler(svc)

	w := listOutfitLogsByDateRange(t, h, "user-1", "2026-03-01", "2026-03-31")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListOutfitLogsByDateRangeShouldReturn200WithLogsWhenSuccessful(t *testing.T) {
	var log1 domain.OutfitLog
	log1.ID = "log-1"
	log1.OutfitID = "outfit-1"
	log1.OwnerID = "user-1"
	log1.WornOn = time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	log1.WearLogIDs = []string{"wl-1", "wl-2"}

	svc := &fakeOutfitLogService{
		listByDateRangeFn: func(_ context.Context, _ string, _, _ time.Time) ([]domain.OutfitLog, error) {
			return []domain.OutfitLog{log1}, nil
		},
	}
	h := newOutfitLogHandler(svc)

	w := listOutfitLogsByDateRange(t, h, "user-1", "2026-03-01", "2026-03-31")

	require.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp, 1)
	require.Equal(t, "log-1", resp[0]["id"])
	require.Equal(t, "outfit-1", resp[0]["outfit_id"])
	require.Equal(t, "2026-03-10", resp[0]["worn_on"])
	wearLogIDs := resp[0]["wear_log_ids"].([]any)
	require.Len(t, wearLogIDs, 2)
}
