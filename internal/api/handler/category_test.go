package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeCategoryService struct {
	listAllFn func(ctx context.Context) ([]domain.Category, error)
}

func (f *fakeCategoryService) ListAll(ctx context.Context) ([]domain.Category, error) {
	if f.listAllFn != nil {
		return f.listAllFn(ctx)
	}
	return nil, nil
}

// --- helpers ---

func newCategoryHandler(svc *fakeCategoryService) *handler.CategoryHandler {
	return handler.NewCategoryHandler(svc, slog.New(slog.DiscardHandler))
}

// --- tests ---

// ── List ──────────────────────────────────────────────────────────────────────

func TestListCategoriesHandlerShouldReturn500WhenServiceFails(t *testing.T) {
	svc := &fakeCategoryService{
		listAllFn: func(_ context.Context) ([]domain.Category, error) {
			return nil, domain.ErrIO
		},
	}
	h := newCategoryHandler(svc)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListCategoriesHandlerShouldReturn200WithCategoriesWhenSuccessful(t *testing.T) {
	var cat1, cat2 domain.Category
	cat1.ID = "cat-1"
	cat1.Label = "Tops"
	cat1.IsPreset = true
	cat2.ID = "cat-2"
	cat2.Label = "Bottoms"
	cat2.IsPreset = true

	svc := &fakeCategoryService{
		listAllFn: func(_ context.Context) ([]domain.Category, error) {
			return []domain.Category{cat1, cat2}, nil
		},
	}
	h := newCategoryHandler(svc)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got []domain.Category
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Len(t, got, 2)
	require.Equal(t, "cat-1", got[0].ID)
	require.Equal(t, "Tops", got[0].Label)
	require.True(t, got[0].IsPreset)
}
