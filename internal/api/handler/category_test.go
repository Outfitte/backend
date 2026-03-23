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

func TestListCategoriesHandlerShouldReturnEmptyFieldHintsArrayWhenCategoryHasNoFieldHints(t *testing.T) {
	var cat domain.Category
	cat.ID = "cat-1"
	cat.Label = "Tops"
	cat.IsPreset = true
	// FieldHints intentionally not set — should serialize as [] not null.

	svc := &fakeCategoryService{
		listAllFn: func(_ context.Context) ([]domain.Category, error) {
			return []domain.Category{cat}, nil
		},
	}
	h := newCategoryHandler(svc)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got []struct {
		FieldHints []struct{} `json:"field_hints"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Len(t, got, 1)
	require.NotNil(t, got[0].FieldHints)
	require.Empty(t, got[0].FieldHints)
}

func TestListCategoriesHandlerShouldIncludeFieldHintsWhenCategoryHasFieldHints(t *testing.T) {
	var cat domain.Category
	cat.ID = "cat-1"
	cat.Label = "Tops"
	cat.IsPreset = true
	cat.FieldHints = []domain.FieldHint{
		{Key: "size", Label: "Size", Placeholder: "e.g. M"},
	}

	svc := &fakeCategoryService{
		listAllFn: func(_ context.Context) ([]domain.Category, error) {
			return []domain.Category{cat}, nil
		},
	}
	h := newCategoryHandler(svc)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got []struct {
		ID         string `json:"id"`
		FieldHints []struct {
			Key         string `json:"key"`
			Label       string `json:"label"`
			Placeholder string `json:"placeholder"`
		} `json:"field_hints"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Len(t, got, 1)
	require.Equal(t, "cat-1", got[0].ID)
	require.Len(t, got[0].FieldHints, 1)
	require.Equal(t, "size", got[0].FieldHints[0].Key)
	require.Equal(t, "Size", got[0].FieldHints[0].Label)
	require.Equal(t, "e.g. M", got[0].FieldHints[0].Placeholder)
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
	var got []struct {
		ID       string `json:"id"`
		Label    string `json:"label"`
		IsPreset bool   `json:"is_preset"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Len(t, got, 2)
	require.Equal(t, "cat-1", got[0].ID)
	require.Equal(t, "Tops", got[0].Label)
	require.True(t, got[0].IsPreset)
}
