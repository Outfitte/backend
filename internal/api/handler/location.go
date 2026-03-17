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
	GetByID(ctx context.Context, callerID, locationID string) (domain.Location, error)
	Update(ctx context.Context, callerID, locationID, label string) (domain.Location, error)
	Delete(ctx context.Context, callerID, locationID string) error
	Move(ctx context.Context, callerID, locationID string, newParentID *string) (domain.Location, error)
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

// GetByID handles GET /locations/{id}.
func (h *LocationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "GetByID")
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

	locationID := r.PathValue("id")
	loc, err := h.locations.GetByID(ctx, callerID, locationID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "get location failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "location_id", loc.ID)
	writeJSON(w, http.StatusOK, loc)
}

type updateLocationRequest struct {
	Label string `json:"label"`
}

// Update handles PATCH /locations/{id}.
func (h *LocationHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Update")
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

	locationID := r.PathValue("id")

	var req updateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	loc, err := h.locations.Update(ctx, callerID, locationID, req.Label)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "update location failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "location_id", loc.ID)
	writeJSON(w, http.StatusOK, loc)
}

// Delete handles DELETE /locations/{id}.
func (h *LocationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Delete")
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

	locationID := r.PathValue("id")

	if err := h.locations.Delete(ctx, callerID, locationID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "delete location failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "location_id", locationID)
	w.WriteHeader(http.StatusNoContent)
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

type moveLocationRequest struct {
	ParentID *string `json:"parent_id"`
}

// Move handles PATCH /locations/{id}/move.
func (h *LocationHandler) Move(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Move")
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

	locationID := r.PathValue("id")

	var req moveLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	loc, err := h.locations.Move(ctx, callerID, locationID, req.ParentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "conflict"})
			return
		}
		log.ErrorContext(ctx, "move location failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "location_id", loc.ID)
	writeJSON(w, http.StatusOK, loc)
}
