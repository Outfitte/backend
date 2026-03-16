package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/outfitte/outfitte/internal/api/middleware"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/service"
)

type itemService interface {
	Create(ctx context.Context, callerID string, input service.CreateItemInput) (domain.Item, error)
	ListByOwner(ctx context.Context, callerID string) ([]domain.Item, error)
	GetByID(ctx context.Context, callerID, itemID string) (domain.Item, error)
	Update(ctx context.Context, callerID, itemID string, input service.UpdateItemInput) (domain.Item, error)
	Delete(ctx context.Context, callerID, itemID string) error
	UploadPhoto(ctx context.Context, callerID, itemID string, r io.Reader, filename string) error
	DeletePhoto(ctx context.Context, callerID, itemID, photoKey string) error
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

type createItemRequest struct {
	Name       string   `json:"name"`
	Brand      string   `json:"brand"`
	CategoryID string   `json:"category_id"`
	Color      string   `json:"color"`
	Size       string   `json:"size"`
	PhotoKeys  []string `json:"photo_keys"`
	LocationID *string  `json:"location_id"`
}

// Create handles POST /items.
func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Create")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	input := service.CreateItemInput{
		Name:       req.Name,
		Brand:      req.Brand,
		CategoryID: req.CategoryID,
		Color:      req.Color,
		Size:       req.Size,
		PhotoKeys:  req.PhotoKeys,
		LocationID: req.LocationID,
	}

	item, err := h.items.Create(ctx, callerID, input)
	if err != nil {
		log.ErrorContext(ctx, "create item failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "item_id", item.ID)
	writeJSON(w, http.StatusCreated, item)
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

// List handles GET /items.
func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
}

// GetByID handles GET /items/{id}.
func (h *ItemHandler) GetByID(w http.ResponseWriter, r *http.Request) {
}

// Update handles PATCH /items/{id}.
func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
}

// Delete handles DELETE /items/{id}.
func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
}

// UploadPhoto handles POST /items/{id}/photos.
func (h *ItemHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
}

// DeletePhoto handles DELETE /items/{id}/photos/{key}.
func (h *ItemHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
}
