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

	log.InfoContext(ctx, "succeeded", "count", len(cats))
	writeJSON(w, http.StatusOK, cats)
}
