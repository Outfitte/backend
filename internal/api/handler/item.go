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

type itemService interface {
	AssignLocation(ctx context.Context, callerID, itemID string, locationID *string) error
}

// ItemHandler handles item-related HTTP endpoints.
type ItemHandler struct {
	items itemService
	log   *slog.Logger
}

// NewItemHandler creates an ItemHandler with a logger pre-scoped to handler=item.
func NewItemHandler(items itemService, log *slog.Logger) *ItemHandler {
	return &ItemHandler{items: items, log: log.With("handler", "item")}
}

type assignLocationRequest struct {
	LocationID *string `json:"location_id"`
}

// AssignLocation handles PATCH /items/{id}/location.
func (h *ItemHandler) AssignLocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "AssignLocation")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	var req assignLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	itemID := r.PathValue("id")
	if err := h.items.AssignLocation(ctx, callerID, itemID, req.LocationID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "assign location failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "item_id", itemID)
	w.WriteHeader(http.StatusNoContent)
}
