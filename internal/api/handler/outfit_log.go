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

type outfitLogService interface {
	LogWear(ctx context.Context, callerID, outfitID string, wornOn time.Time, notes *string) (domain.OutfitLog, error)
	ListByOutfit(ctx context.Context, callerID, outfitID string) ([]domain.OutfitLog, error)
	ListByDateRange(ctx context.Context, callerID string, from, to time.Time) ([]domain.OutfitLog, error)
	UpdateDate(ctx context.Context, callerID, outfitLogID string, newDate time.Time) (domain.OutfitLog, error)
	Delete(ctx context.Context, callerID, outfitLogID string) error
}

type outfitLogResponse struct {
	ID         string    `json:"id"`
	OutfitID   string    `json:"outfit_id"`
	OwnerID    string    `json:"owner_id"`
	WornOn     string    `json:"worn_on"` // calendar date only, formatted YYYY-MM-DD
	Notes      *string   `json:"notes"`
	WearLogIDs []string  `json:"wear_log_ids"`
	CreatedAt  time.Time `json:"created_at"`
}

func toOutfitLogResponse(ol domain.OutfitLog) outfitLogResponse {
	wearLogIDs := ol.WearLogIDs
	if wearLogIDs == nil {
		wearLogIDs = []string{}
	}
	return outfitLogResponse{
		ID:         ol.GetID(),
		OutfitID:   ol.OutfitID,
		OwnerID:    ol.OwnerID,
		WornOn:     ol.WornOn.Format("2006-01-02"),
		Notes:      ol.Notes,
		WearLogIDs: wearLogIDs,
		CreatedAt:  ol.CreatedAt,
	}
}

// OutfitLogHandler handles outfit-log-related HTTP endpoints.
type OutfitLogHandler struct {
	outfitLogs outfitLogService
	log        *slog.Logger
}

// NewOutfitLogHandler creates an OutfitLogHandler with a logger pre-scoped to handler=outfit_log.
func NewOutfitLogHandler(outfitLogs outfitLogService, log *slog.Logger) *OutfitLogHandler {
	return &OutfitLogHandler{outfitLogs: outfitLogs, log: log.With("handler", "outfit_log")}
}

type logOutfitWearRequest struct {
	WornOn string  `json:"worn_on"`
	Notes  *string `json:"notes"`
}

// LogWear handles POST /outfits/{id}/logs.
func (h *OutfitLogHandler) LogWear(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "LogWear")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	var req logOutfitWearRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	wornOn, err := time.Parse("2006-01-02", req.WornOn)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	outfitID := r.PathValue("id")
	entry, err := h.outfitLogs.LogWear(ctx, callerID, outfitID, wornOn, req.Notes)
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
		log.ErrorContext(ctx, "log outfit wear failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "log_id", entry.GetID())
	writeJSON(w, http.StatusCreated, toOutfitLogResponse(entry))
}

// ListByOutfit handles GET /outfits/{id}/logs.
func (h *OutfitLogHandler) ListByOutfit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "ListByOutfit")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	outfitID := r.PathValue("id")
	logs, err := h.outfitLogs.ListByOutfit(ctx, callerID, outfitID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "list outfit logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	responses := make([]outfitLogResponse, len(logs))
	for i, l := range logs {
		responses[i] = toOutfitLogResponse(l)
	}
	log.InfoContext(ctx, "succeeded", "count", len(logs))
	writeJSON(w, http.StatusOK, responses)
}

type updateOutfitLogDateRequest struct {
	WornOn string `json:"worn_on"`
}

// UpdateDate handles PATCH /outfits/{id}/logs/{logID}.
func (h *OutfitLogHandler) UpdateDate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "UpdateDate")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	var req updateOutfitLogDateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	newDate, err := time.Parse("2006-01-02", req.WornOn)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	logID := r.PathValue("logID")
	updated, err := h.outfitLogs.UpdateDate(ctx, callerID, logID, newDate)
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
		log.ErrorContext(ctx, "update outfit log date failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "log_id", logID)
	writeJSON(w, http.StatusOK, toOutfitLogResponse(updated))
}

// Delete handles DELETE /outfits/{id}/logs/{logID}.
func (h *OutfitLogHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Delete")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	logID := r.PathValue("logID")
	if err := h.outfitLogs.Delete(ctx, callerID, logID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "delete outfit log failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "log_id", logID)
	w.WriteHeader(http.StatusNoContent)
}

// ListByDateRange handles GET /outfit-logs?from=YYYY-MM-DD&to=YYYY-MM-DD.
func (h *OutfitLogHandler) ListByDateRange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "ListByDateRange")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" || toStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "from and to are required"})
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid from date, use YYYY-MM-DD"})
		return
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid to date, use YYYY-MM-DD"})
		return
	}

	if from.After(to) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "from must not be after to"})
		return
	}

	logs, err := h.outfitLogs.ListByDateRange(ctx, callerID, from, to)
	if err != nil {
		log.ErrorContext(ctx, "list outfit logs by date range failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	responses := make([]outfitLogResponse, len(logs))
	for i, l := range logs {
		responses[i] = toOutfitLogResponse(l)
	}
	log.InfoContext(ctx, "succeeded", "count", len(logs))
	writeJSON(w, http.StatusOK, responses)
}
