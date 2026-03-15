package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/outfitte/outfitte/internal/domain"
)

// categoryNS is the UUID v5 namespace used to derive stable IDs for preset categories.
var categoryNS = uuid.MustParse("c3d4e5f6-a7b8-4c9d-8e0f-1a2b3c4d5e6f")

// presetCategories is computed once at startup; IDs are stable across calls.
var presetCategories = buildPresets([]string{
	"Tops", "Bottoms", "Outerwear", "Footwear", "Accessories",
	"Underwear", "Sportswear", "Formalwear", "Bags",
})

func buildPresets(labels []string) []domain.Category {
	cats := make([]domain.Category, len(labels))
	for i, label := range labels {
		cats[i].ID = uuid.NewSHA1(categoryNS, []byte(label)).String()
		cats[i].Label = label
		cats[i].IsPreset = true
	}
	return cats
}

type CategoryService struct{}

func NewCategoryService() *CategoryService {
	return &CategoryService{}
}

func (s *CategoryService) ListAll(ctx context.Context) ([]domain.Category, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return presetCategories, nil
}
