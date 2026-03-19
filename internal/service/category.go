package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/outfitte/outfitte/internal/domain"
)

// categoryNS is the UUID v5 namespace used to derive stable IDs for preset categories.
var categoryNS = uuid.MustParse("c3d4e5f6-a7b8-4c9d-8e0f-1a2b3c4d5e6f")

type presetDef struct {
	label string
	hints []domain.FieldHint
}

func hint(key, label, placeholder string) domain.FieldHint {
	return domain.FieldHint{Key: key, Label: label, Placeholder: placeholder}
}

// presetDefs lists all preset categories with their field hints.
var presetDefs = []presetDef{
	{
		label: "Tops",
		hints: []domain.FieldHint{
			hint("size", "Size", "e.g. M"),
			hint("fabric", "Fabric", "e.g. Cotton"),
			hint("fit", "Fit", "e.g. Regular"),
		},
	},
	{
		label: "Bottoms",
		hints: []domain.FieldHint{
			hint("size", "Size", "e.g. 32W 30L"),
			hint("fabric", "Fabric", "e.g. Denim"),
			hint("fit", "Fit", "e.g. Slim"),
			hint("length", "Length", "e.g. Full"),
		},
	},
	{
		label: "Outerwear",
		hints: []domain.FieldHint{
			hint("size", "Size", "e.g. L"),
			hint("fabric", "Fabric", "e.g. Wool"),
			hint("shell_material", "Shell Material", "e.g. Gore-Tex"),
		},
	},
	{
		label: "Footwear",
		hints: []domain.FieldHint{
			hint("size", "Size", "e.g. UK 9"),
			hint("material", "Material", "e.g. Leather"),
			hint("sole_type", "Sole Type", "e.g. Rubber"),
		},
	},
	{
		label: "Accessories",
		hints: []domain.FieldHint{
			hint("material", "Material", "e.g. Canvas"),
			hint("dimensions", "Dimensions", "e.g. 30×20cm"),
		},
	},
	{
		label: "Underwear",
		hints: []domain.FieldHint{
			hint("size", "Size", "e.g. M"),
			hint("fabric", "Fabric", "e.g. Modal"),
		},
	},
	{
		label: "Sportswear",
		hints: []domain.FieldHint{
			hint("size", "Size", "e.g. S"),
			hint("fabric", "Fabric", "e.g. Polyester"),
			hint("sport", "Sport", "e.g. Running"),
		},
	},
	{
		label: "Formalwear",
		hints: []domain.FieldHint{
			hint("size", "Size", "e.g. 40R"),
			hint("fabric", "Fabric", "e.g. Wool"),
			hint("lining", "Lining", "e.g. Silk"),
		},
	},
	{
		label: "Bags",
		hints: []domain.FieldHint{
			hint("dimensions", "Dimensions", "e.g. 40×30×15cm"),
			hint("material", "Material", "e.g. Leather"),
			hint("strap_type", "Strap Type", "e.g. Crossbody"),
		},
	},
	{
		label: "Jewellery",
		hints: []domain.FieldHint{
			hint("material", "Material", "e.g. Gold"),
			hint("stone", "Stone", "e.g. Diamond"),
			hint("ring_size", "Ring Size", "e.g. N"),
		},
	},
	{
		label: "Watches",
		hints: []domain.FieldHint{
			hint("case_diameter", "Case Diameter", "e.g. 40mm"),
			hint("movement_type", "Movement Type", "e.g. Automatic"),
			hint("strap_material", "Strap Material", "e.g. Leather"),
			hint("water_resistance", "Water Resistance", "e.g. 100m"),
		},
	},
}

// presetCategories is computed once at startup; IDs are stable across calls.
var presetCategories = buildPresets(presetDefs)

func buildPresets(defs []presetDef) []domain.Category {
	cats := make([]domain.Category, len(defs))
	for i, def := range defs {
		cats[i].ID = uuid.NewSHA1(categoryNS, []byte(def.label)).String()
		cats[i].Label = def.label
		cats[i].IsPreset = true
		cats[i].FieldHints = def.hints
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
