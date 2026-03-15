package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/outfitte/outfitte/internal/domain"
)

var presetCategories = []string{
	"Tops", "Bottoms", "Outerwear", "Footwear", "Accessories",
	"Underwear", "Sportswear", "Formalwear", "Bags",
}

type CategoryService struct{}

func NewCategoryService() *CategoryService {
	return &CategoryService{}
}

func (s *CategoryService) ListAll(ctx context.Context) ([]domain.Category, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	categories := make([]domain.Category, len(presetCategories))
	for i, label := range presetCategories {
		categories[i].ID = uuid.NewString()
		categories[i].Label = label
		categories[i].IsPreset = true
	}
	return categories, nil
}
