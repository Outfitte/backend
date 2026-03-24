package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/outfitte/outfitte/internal/api/middleware"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/outfitte/outfitte/internal/service"
)

type photoResponse struct {
	ID        string    `json:"id"`
	MediaKey  string    `json:"media_key"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

type itemResponse struct {
	ID            string            `json:"id"`
	OwnerID       string            `json:"owner_id"`
	Name          string            `json:"name"`
	Brand         *string           `json:"brand"`
	CategoryID    *string           `json:"category_id"`
	Color         *string           `json:"color"`
	Metadata      map[string]string `json:"metadata"`
	Photos        []photoResponse   `json:"photos"`
	LocationID    *string           `json:"location_id"`
	PurchasePrice *string           `json:"purchase_price"`
	// PurchaseDate is omitted intentionally — it is deferred to M4+.
	CreatedAt  time.Time `json:"created_at"`
	WearCount  int       `json:"wear_count"`
	LastWornAt *string   `json:"last_worn_at"`
}

func toItemResponse(item domain.Item) itemResponse {
	photos := make([]photoResponse, len(item.Photos))
	for i, p := range item.Photos {
		photos[i] = photoResponse{
			ID:        p.ID,
			MediaKey:  p.MediaKey,
			Position:  p.Position,
			CreatedAt: p.CreatedAt,
		}
	}
	meta := item.Metadata.Fields
	if meta == nil {
		meta = map[string]string{}
	}
	return itemResponse{
		ID:            item.ID,
		OwnerID:       item.OwnerID,
		Name:          item.Name,
		Brand:         item.Brand,
		CategoryID:    item.CategoryID,
		Color:         item.Color,
		Metadata:      meta,
		Photos:        photos,
		LocationID:    item.LocationID,
		PurchasePrice: item.PurchasePrice,
		CreatedAt:     item.CreatedAt,
	}
}

func toRichItemResponse(item service.RichItem) itemResponse {
	r := toItemResponse(item.Item)
	r.WearCount = item.WearCount
	if item.LastWornAt != nil {
		s := item.LastWornAt.Format("2006-01-02")
		r.LastWornAt = &s
	}
	return r
}

type itemService interface {
	Create(ctx context.Context, callerID string, input service.CreateItemInput) (domain.Item, error)
	ListByOwner(ctx context.Context, callerID string, filter ports.ItemListFilter) ([]service.RichItem, error)
	GetByID(ctx context.Context, callerID, itemID string) (service.RichItem, error)
	Update(ctx context.Context, callerID, itemID string, input service.UpdateItemInput) (domain.Item, error)
	Delete(ctx context.Context, callerID, itemID string) error
	UploadPhoto(ctx context.Context, callerID, itemID string, r io.Reader, filename string) error
	DeletePhoto(ctx context.Context, callerID, itemID, photoKey string) error
	AssignLocation(ctx context.Context, callerID, itemID string, locationID *string) error
	Archive(ctx context.Context, callerID, itemID string) error
	Unarchive(ctx context.Context, callerID, itemID string) error
	Dispose(ctx context.Context, callerID, itemID string, reason domain.DisposalReason) error
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
	Name       string            `json:"name"`
	Brand      *string           `json:"brand"`
	CategoryID *string           `json:"category_id"`
	Color      *string           `json:"color"`
	Metadata   map[string]string `json:"metadata"`
	PhotoKeys  []string          `json:"photo_keys"`
	LocationID *string           `json:"location_id"`
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
		Metadata:   domain.ItemMetadata{Fields: req.Metadata},
		PhotoKeys:  req.PhotoKeys,
		LocationID: req.LocationID,
	}

	item, err := h.items.Create(ctx, callerID, input)
	if err != nil {
		if errors.Is(err, domain.ErrValidation) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "validation error"})
			return
		}
		log.ErrorContext(ctx, "create item failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "item_id", item.ID)
	writeJSON(w, http.StatusCreated, toItemResponse(item))
}

type updateItemRequest struct {
	Name          string            `json:"name"`
	Brand         *string           `json:"brand"`
	CategoryID    *string           `json:"category_id"`
	Color         *string           `json:"color"`
	Metadata      map[string]string `json:"metadata"`
	PhotoKeys     []string          `json:"photo_keys"`
	LocationID    *string           `json:"location_id"`
	PurchasePrice *string           `json:"purchase_price"`
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
	ctx := r.Context()
	log := h.log.With("call", "List")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	status := r.URL.Query().Get("status")
	if status == "" {
		status = ports.ItemStatusActive
	}
	switch status {
	case ports.ItemStatusActive, ports.ItemStatusArchived, ports.ItemStatusAll:
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status filter"})
		return
	}
	items, err := h.items.ListByOwner(ctx, callerID, ports.ItemListFilter{Status: status})
	if err != nil {
		log.ErrorContext(ctx, "list items failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	responses := make([]itemResponse, len(items))
	for i, item := range items {
		responses[i] = toRichItemResponse(item)
	}
	log.InfoContext(ctx, "succeeded", "count", len(items))
	writeJSON(w, http.StatusOK, responses)
}

// GetByID handles GET /items/{id}.
func (h *ItemHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "GetByID")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	itemID := r.PathValue("id")
	item, err := h.items.GetByID(ctx, callerID, itemID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "get item failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "item_id", item.ID)
	writeJSON(w, http.StatusOK, toRichItemResponse(item))
}

// Update handles PATCH /items/{id}.
func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Update")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	var req updateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	itemID := r.PathValue("id")
	input := service.UpdateItemInput{
		Name:          req.Name,
		Brand:         req.Brand,
		CategoryID:    req.CategoryID,
		Color:         req.Color,
		Metadata:      domain.ItemMetadata{Fields: req.Metadata},
		PhotoKeys:     req.PhotoKeys,
		LocationID:    req.LocationID,
		PurchasePrice: req.PurchasePrice,
	}
	item, err := h.items.Update(ctx, callerID, itemID, input)
	if err != nil {
		if errors.Is(err, domain.ErrValidation) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "validation error"})
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
		log.ErrorContext(ctx, "update item failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	log.InfoContext(ctx, "succeeded", "item_id", item.ID)
	writeJSON(w, http.StatusOK, toItemResponse(item))
}

// Delete handles DELETE /items/{id}.
func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Delete")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	itemID := r.PathValue("id")
	if err := h.items.Delete(ctx, callerID, itemID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "delete item failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "item_id", itemID)
	w.WriteHeader(http.StatusNoContent)
}

// UploadPhoto handles POST /items/{id}/photos.
func (h *ItemHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "UploadPhoto")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	itemID := r.PathValue("id")
	file, header, err := r.FormFile("photo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing or invalid photo"})
		return
	}
	defer file.Close()

	if err := h.items.UploadPhoto(ctx, callerID, itemID, file, header.Filename); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "upload photo failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "item_id", itemID)
	w.WriteHeader(http.StatusCreated)
}

// DeletePhoto handles DELETE /items/{id}/photos/{key}.
func (h *ItemHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "DeletePhoto")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	itemID := r.PathValue("id")
	photoKey := r.PathValue("key")
	if err := h.items.DeletePhoto(ctx, callerID, itemID, photoKey); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "delete photo failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "item_id", itemID, "photo_key", photoKey)
	w.WriteHeader(http.StatusNoContent)
}

// Archive handles POST /items/{id}/archive.
func (h *ItemHandler) Archive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Archive")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	itemID := r.PathValue("id")
	if err := h.items.Archive(ctx, callerID, itemID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		if errors.Is(err, domain.ErrAlreadyArchived) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "already archived"})
			return
		}
		log.ErrorContext(ctx, "archive item failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "item_id", itemID)
	w.WriteHeader(http.StatusNoContent)
}

// Unarchive handles POST /items/{id}/unarchive.
func (h *ItemHandler) Unarchive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Unarchive")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	itemID := r.PathValue("id")
	if err := h.items.Unarchive(ctx, callerID, itemID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		if errors.Is(err, domain.ErrNotArchived) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "not archived"})
			return
		}
		log.ErrorContext(ctx, "unarchive item failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "item_id", itemID)
	w.WriteHeader(http.StatusNoContent)
}

type disposeItemRequest struct {
	Reason string `json:"reason"`
}

// Dispose handles POST /items/{id}/dispose.
func (h *ItemHandler) Dispose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Dispose")
	log.InfoContext(ctx, "started")

	callerID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		log.ErrorContext(ctx, "missing caller ID in context")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	var req disposeItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	reason := domain.DisposalReason(req.Reason)
	switch reason {
	case domain.DisposalDonated, domain.DisposalSold, domain.DisposalDiscarded, domain.DisposalLost, domain.DisposalOther:
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid disposal reason"})
		return
	}

	itemID := r.PathValue("id")
	if err := h.items.Dispose(ctx, callerID, itemID, reason); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "dispose item failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "item_id", itemID, "reason", reason)
	w.WriteHeader(http.StatusNoContent)
}
