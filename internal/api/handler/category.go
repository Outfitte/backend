package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/outfitte/outfitte/internal/domain"
)

type categoryService interface {
	ListAll(ctx context.Context) ([]domain.Category, error)
}

// CategoryHandler handles category-related HTTP endpoints.
type CategoryHandler struct {
	categories categoryService
	log        *slog.Logger
}

// NewCategoryHandler creates a CategoryHandler with a logger pre-scoped to handler=category.
func NewCategoryHandler(categories categoryService, log *slog.Logger) *CategoryHandler {
	return &CategoryHandler{categories: categories, log: log.With("handler", "category")}
}

type fieldHintResponse struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
}

type categoryResponse struct {
	ID         string              `json:"id"`
	Label      string              `json:"label"`
	IsPreset   bool                `json:"is_preset"`
	FieldHints []fieldHintResponse `json:"field_hints"`
}

func toCategoryResponse(cat domain.Category) categoryResponse {
	hints := make([]fieldHintResponse, len(cat.FieldHints))
	for i, h := range cat.FieldHints {
		hints[i] = fieldHintResponse{Key: h.Key, Label: h.Label, Placeholder: h.Placeholder}
	}
	return categoryResponse{
		ID:         cat.ID,
		Label:      cat.Label,
		IsPreset:   cat.IsPreset,
		FieldHints: hints,
	}
}

// List handles GET /categories.
func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "List")
	log.InfoContext(ctx, "started")

	cats, err := h.categories.ListAll(ctx)
	if err != nil {
		log.ErrorContext(ctx, "list categories failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	responses := make([]categoryResponse, len(cats))
	for i, cat := range cats {
		responses[i] = toCategoryResponse(cat)
	}
	log.InfoContext(ctx, "succeeded", "count", len(cats))
	writeJSON(w, http.StatusOK, responses)
}
