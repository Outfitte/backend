package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/service"
)

type outfitService interface {
	Create(ctx context.Context, callerID string, input service.CreateOutfitInput) (domain.Outfit, error)
	GetByID(ctx context.Context, callerID, outfitID string) (domain.Outfit, error)
	ListByOwner(ctx context.Context, callerID string) ([]domain.Outfit, error)
	ListByDateRange(ctx context.Context, callerID string, from, to time.Time) ([]domain.Outfit, error)
	Update(ctx context.Context, callerID, outfitID string, input service.UpdateOutfitInput) (domain.Outfit, error)
	Delete(ctx context.Context, callerID, outfitID string) error
	AddItem(ctx context.Context, callerID, outfitID, itemID string) error
	RemoveItem(ctx context.Context, callerID, outfitID, itemID string) error
	UploadPhoto(ctx context.Context, callerID, outfitID string, r io.Reader, filename string) error
	DeletePhoto(ctx context.Context, callerID, outfitID, mediaKey string) error
}

type outfitItemResponse struct {
	OutfitID string `json:"outfit_id"`
	ItemID   string `json:"item_id"`
	Position int    `json:"position"`
}

type outfitPhotoResponse struct {
	ID        string    `json:"id"`
	MediaKey  string    `json:"media_key"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

type outfitResponse struct {
	ID        string                `json:"id"`
	OwnerID   string                `json:"owner_id"`
	Name      *string               `json:"name"`
	Notes     *string               `json:"notes"`
	Items     []outfitItemResponse  `json:"items"`
	Photos    []outfitPhotoResponse `json:"photos"`
	CreatedAt time.Time             `json:"created_at"`
}

func toOutfitResponse(o domain.Outfit) outfitResponse {
	items := make([]outfitItemResponse, len(o.Items))
	for i, it := range o.Items {
		items[i] = outfitItemResponse{
			OutfitID: it.OutfitID,
			ItemID:   it.ItemID,
			Position: it.Position,
		}
	}
	photos := make([]outfitPhotoResponse, len(o.Photos))
	for i, p := range o.Photos {
		photos[i] = outfitPhotoResponse{
			ID:        p.ID,
			MediaKey:  p.MediaKey,
			Position:  p.Position,
			CreatedAt: p.CreatedAt,
		}
	}
	return outfitResponse{
		ID:        o.GetID(),
		OwnerID:   o.OwnerID,
		Name:      o.Name,
		Notes:     o.Notes,
		Items:     items,
		Photos:    photos,
		CreatedAt: o.CreatedAt,
	}
}

// OutfitHandler handles outfit-related HTTP endpoints.
type OutfitHandler struct {
	outfits outfitService
	log     *slog.Logger
}

// NewOutfitHandler creates an OutfitHandler with a logger pre-scoped to handler=outfit.
func NewOutfitHandler(outfits outfitService, log *slog.Logger) *OutfitHandler {
	return &OutfitHandler{outfits: outfits, log: log.With("handler", "outfit")}
}

type createOutfitRequest struct {
	Name  *string `json:"name"`
	Notes *string `json:"notes"`
}

// Create handles POST /outfits.
func (h *OutfitHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Create")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	var req createOutfitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	outfit, err := h.outfits.Create(ctx, callerID, service.CreateOutfitInput{
		Name:  req.Name,
		Notes: req.Notes,
	})
	if err != nil {
		log.ErrorContext(ctx, "create outfit failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "outfit_id", outfit.GetID())
	writeJSON(w, http.StatusCreated, toOutfitResponse(outfit))
}

// List handles GET /outfits.
// When ?from and ?to query params are present, returns outfits with at least one log in that range.
func (h *OutfitHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "List")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var outfits []domain.Outfit
	var err error

	if fromStr != "" || toStr != "" {
		if fromStr == "" || toStr == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "from and to must both be provided"})
			return
		}
		outfits, err = h.listByDateRange(ctx, w, log, callerID, fromStr, toStr)
		if err != nil {
			return // response already written by listByDateRange
		}
	} else {
		outfits, err = h.outfits.ListByOwner(ctx, callerID)
		if err != nil {
			log.ErrorContext(ctx, "list outfits failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			return
		}
	}

	responses := make([]outfitResponse, len(outfits))
	for i, o := range outfits {
		responses[i] = toOutfitResponse(o)
	}
	log.InfoContext(ctx, "succeeded", "count", len(outfits))
	writeJSON(w, http.StatusOK, responses)
}

func (h *OutfitHandler) listByDateRange(ctx context.Context, w http.ResponseWriter, log *slog.Logger, callerID, fromStr, toStr string) ([]domain.Outfit, error) {
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid from date, use YYYY-MM-DD"})
		return nil, err
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid to date, use YYYY-MM-DD"})
		return nil, err
	}
	outfits, err := h.outfits.ListByDateRange(ctx, callerID, from, to)
	if err != nil {
		if errors.Is(err, domain.ErrValidation) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "from must not be after to"})
			return nil, err
		}
		log.ErrorContext(ctx, "list outfits by date range failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return nil, err
	}
	return outfits, nil
}

// GetByID handles GET /outfits/{id}.
func (h *OutfitHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "GetByID")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	outfitID := r.PathValue("id")
	outfit, err := h.outfits.GetByID(ctx, callerID, outfitID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "get outfit failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "outfit_id", outfit.GetID())
	writeJSON(w, http.StatusOK, toOutfitResponse(outfit))
}

type updateOutfitRequest struct {
	Name  *string `json:"name"`
	Notes *string `json:"notes"`
}

// Update handles PATCH /outfits/{id}.
func (h *OutfitHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Update")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	var req updateOutfitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	outfitID := r.PathValue("id")
	outfit, err := h.outfits.Update(ctx, callerID, outfitID, service.UpdateOutfitInput{
		Name:  req.Name,
		Notes: req.Notes,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "update outfit failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "outfit_id", outfit.GetID())
	writeJSON(w, http.StatusOK, toOutfitResponse(outfit))
}

// Delete handles DELETE /outfits/{id}.
func (h *OutfitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Delete")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	outfitID := r.PathValue("id")
	if err := h.outfits.Delete(ctx, callerID, outfitID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "delete outfit failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "outfit_id", outfitID)
	w.WriteHeader(http.StatusNoContent)
}

type addItemRequest struct {
	ItemID string `json:"item_id"`
}

// AddItem handles POST /outfits/{id}/items.
func (h *OutfitHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "AddItem")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	outfitID := r.PathValue("id")
	if err := h.outfits.AddItem(ctx, callerID, outfitID, req.ItemID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "add item failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "outfit_id", outfitID, "item_id", req.ItemID)
	w.WriteHeader(http.StatusNoContent)
}

// RemoveItem handles DELETE /outfits/{id}/items/{itemID}.
func (h *OutfitHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "RemoveItem")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	outfitID := r.PathValue("id")
	itemID := r.PathValue("itemID")
	if err := h.outfits.RemoveItem(ctx, callerID, outfitID, itemID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "remove item failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "outfit_id", outfitID, "item_id", itemID)
	w.WriteHeader(http.StatusNoContent)
}

// UploadPhoto handles POST /outfits/{id}/photos.
func (h *OutfitHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "UploadPhoto")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	outfitID := r.PathValue("id")
	file, header, err := r.FormFile("photo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing or invalid photo"})
		return
	}
	defer file.Close()

	if err := h.outfits.UploadPhoto(ctx, callerID, outfitID, file, header.Filename); err != nil {
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

	log.InfoContext(ctx, "succeeded", "outfit_id", outfitID)
	w.WriteHeader(http.StatusCreated)
}

// DeletePhoto handles DELETE /outfits/{id}/photos/{key...}.
func (h *OutfitHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "DeletePhoto")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	outfitID := r.PathValue("id")
	photoKey := r.PathValue("key")
	if err := h.outfits.DeletePhoto(ctx, callerID, outfitID, photoKey); err != nil {
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

	log.InfoContext(ctx, "succeeded", "outfit_id", outfitID, "photo_key", photoKey)
	w.WriteHeader(http.StatusNoContent)
}
