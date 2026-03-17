package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/outfitte/outfitte/internal/api/middleware"
	"github.com/outfitte/outfitte/internal/domain"
)


type locationService interface {
	Create(ctx context.Context, callerID, label string, parentID *string) (domain.Location, error)
	ListByOwner(ctx context.Context, callerID string) ([]domain.Location, error)
}

// LocationHandler handles location-related HTTP endpoints.
type LocationHandler struct {
	locations locationService
	log       *slog.Logger
}

// NewLocationHandler creates a LocationHandler with a logger pre-scoped to handler=location.
func NewLocationHandler(locations locationService, log *slog.Logger) *LocationHandler {
	return &LocationHandler{locations: locations, log: log.With("handler", "location")}
}

type createLocationRequest struct {
	Label    string  `json:"label"`
	ParentID *string `json:"parent_id"`
}

// Create handles POST /locations.
func (h *LocationHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Create")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	var req createLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	loc, err := h.locations.Create(ctx, callerID, req.Label, req.ParentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "create location failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "location_id", loc.ID)
	writeJSON(w, http.StatusCreated, loc)
}

// List handles GET /locations.
func (h *LocationHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "List")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	locs, err := h.locations.ListByOwner(ctx, callerID)
	if err != nil {
		log.ErrorContext(ctx, "list locations failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "count", len(locs))
	writeJSON(w, http.StatusOK, locs)
}
