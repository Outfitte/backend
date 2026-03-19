package domain_test

import (
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/assert"
)

func TestItemShouldHaveCategoryIDAsOptionalPointerWhenUncategorised(t *testing.T) {
	var item domain.Item
	item.ID = "42"
	item.Name = "Jacket"
	// CategoryID nil means uncategorised
	assert.Nil(t, item.CategoryID)
	catID := "cat-1"
	item.CategoryID = &catID
	assert.Equal(t, "cat-1", *item.CategoryID)
}

func TestItemShouldHaveBrandAndColorAsOptionalPointersWhenAbsent(t *testing.T) {
	var item domain.Item
	item.ID = "42"
	item.Name = "Jacket"
	assert.Nil(t, item.Brand)
	assert.Nil(t, item.Color)
	brand := "Patagonia"
	color := "Blue"
	item.Brand = &brand
	item.Color = &color
	assert.Equal(t, "Patagonia", *item.Brand)
	assert.Equal(t, "Blue", *item.Color)
}

func TestItemShouldCarryMetadataFieldsWhenSet(t *testing.T) {
	var item domain.Item
	item.ID = "42"
	item.Name = "Jacket"
	item.Metadata = domain.ItemMetadata{Fields: map[string]string{"size": "M"}}
	assert.Equal(t, "M", item.Metadata.Fields["size"])
}

func TestItemMetadataShouldHoldStringFieldsWhenConstructed(t *testing.T) {
	meta := domain.ItemMetadata{
		Fields: map[string]string{"size": "M", "fit": "slim"},
	}
	assert.Equal(t, "M", meta.Fields["size"])
	assert.Equal(t, "slim", meta.Fields["fit"])
}

func TestCategoryUncategorisedShouldBeEmptyStringWhenUsedAsSentinel(t *testing.T) {
	assert.Equal(t, "", domain.CategoryUncategorised)
}

func TestEntitiesImplementPortsEntity(t *testing.T) {
	iface := (*ports.Entity)(nil)
	assert.Implements(t, iface, domain.User{})
	assert.Implements(t, iface, domain.Item{})
	assert.Implements(t, iface, domain.Location{})
	assert.Implements(t, iface, domain.Category{})
	assert.Implements(t, iface, domain.Session{})
}
