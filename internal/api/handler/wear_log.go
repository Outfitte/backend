package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
)

type wearLogService interface {
	LogWear(ctx context.Context, callerID, itemID string, wornOn time.Time, notes *string) (domain.WearLog, error)
	ListByItem(ctx context.Context, callerID, itemID string) ([]domain.WearLog, error)
	DeleteWearLog(ctx context.Context, callerID, logID string) error
}

type wearLogResponse struct {
	ID        string    `json:"id"`
	ItemID    string    `json:"item_id"`
	OwnerID   string    `json:"owner_id"`
	WornOn    string    `json:"worn_on"` // calendar date only, formatted YYYY-MM-DD
	Notes     *string   `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

func toWearLogResponse(w domain.WearLog) wearLogResponse {
	return wearLogResponse{
		ID:        w.ID,
		ItemID:    w.ItemID,
		OwnerID:   w.OwnerID,
		WornOn:    w.WornOn.Format("2006-01-02"),
		Notes:     w.Notes,
		CreatedAt: w.CreatedAt,
	}
}

// WearLogHandler handles wear-log-related HTTP endpoints.
type WearLogHandler struct {
	wearLogs wearLogService
	log      *slog.Logger
}

// NewWearLogHandler creates a WearLogHandler with a logger pre-scoped to handler=wear_log.
func NewWearLogHandler(wearLogs wearLogService, log *slog.Logger) *WearLogHandler {
	return &WearLogHandler{wearLogs: wearLogs, log: log.With("handler", "wear_log")}
}

type logWearRequest struct {
	WornOn string  `json:"worn_on"`
	Notes  *string `json:"notes"`
}

// LogWear handles POST /items/{id}/wear-logs.
func (h *WearLogHandler) LogWear(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "LogWear")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	var req logWearRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	wornOn, err := time.Parse("2006-01-02", req.WornOn)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	itemID := r.PathValue("id")
	entry, err := h.wearLogs.LogWear(ctx, callerID, itemID, wornOn, req.Notes)
	if err != nil {
		if errors.Is(err, domain.ErrFutureDateNotAllowed) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "log wear failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "log_id", entry.ID)
	writeJSON(w, http.StatusCreated, toWearLogResponse(entry))
}

// ListByItem handles GET /items/{id}/wear-logs.
func (h *WearLogHandler) ListByItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "ListByItem")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	itemID := r.PathValue("id")
	logs, err := h.wearLogs.ListByItem(ctx, callerID, itemID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "list wear logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	responses := make([]wearLogResponse, len(logs))
	for i, l := range logs {
		responses[i] = toWearLogResponse(l)
	}
	log.InfoContext(ctx, "succeeded", "count", len(logs))
	writeJSON(w, http.StatusOK, responses)
}

// DeleteWearLog handles DELETE /items/{id}/wear-logs/{logID}.
func (h *WearLogHandler) DeleteWearLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "DeleteWearLog")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	logID := r.PathValue("logID")
	if err := h.wearLogs.DeleteWearLog(ctx, callerID, logID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "delete wear log failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "log_id", logID)
	w.WriteHeader(http.StatusNoContent)
}

