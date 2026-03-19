package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/service"
)

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestCategoryGetByIDShouldReturnContextErrorWhenContextIsCancelled(t *testing.T) {
	svc := service.NewCategoryService()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.GetByID(ctx, "any-id")
	require.ErrorIs(t, err, context.Canceled)
}

func TestCategoryGetByIDShouldReturnErrNotFoundWhenIDIsUnknown(t *testing.T) {
	svc := service.NewCategoryService()

	_, err := svc.GetByID(t.Context(), "unknown-id")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCategoryGetByIDShouldReturnCategoryWhenIDIsKnown(t *testing.T) {
	svc := service.NewCategoryService()

	categories, err := svc.ListAll(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, categories)

	cat, err := svc.GetByID(t.Context(), categories[0].ID)
	require.NoError(t, err)
	require.Equal(t, categories[0], cat)
}

// ── ListAll ───────────────────────────────────────────────────────────────────

func TestCategoryListAllShouldFailWhenContextIsCancelled(t *testing.T) {
	svc := service.NewCategoryService()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := svc.ListAll(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestCategoryListAllShouldReturnAllPresetsWhenContextIsActive(t *testing.T) {
	svc := service.NewCategoryService()

	categories, err := svc.ListAll(t.Context())
	require.NoError(t, err)

	labels := make([]string, len(categories))
	for i, c := range categories {
		labels[i] = c.Label
	}

	require.ElementsMatch(t, []string{
		"Tops", "Bottoms", "Outerwear", "Footwear", "Accessories",
		"Underwear", "Sportswear", "Formalwear", "Bags",
		"Jewellery", "Watches",
	}, labels)
	for _, c := range categories {
		require.True(t, c.IsPreset)
		require.NotEmpty(t, c.ID)
	}
}

func TestCategoryListAllShouldReturnFieldHintsWhenContextIsActive(t *testing.T) {
	svc := service.NewCategoryService()

	categories, err := svc.ListAll(t.Context())
	require.NoError(t, err)

	for _, c := range categories {
		require.NotEmpty(t, c.FieldHints, "category %q should have field hints", c.Label)
	}
}

func TestCategoryListAllShouldReturnStableIDsWhenCalledMultipleTimes(t *testing.T) {
	svc := service.NewCategoryService()

	first, err := svc.ListAll(t.Context())
	require.NoError(t, err)

	second, err := svc.ListAll(t.Context())
	require.NoError(t, err)

	require.Equal(t, len(first), len(second))
	for i := range first {
		require.Equal(t, first[i].ID, second[i].ID)
	}
}
