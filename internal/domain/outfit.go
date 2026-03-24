package domain

import "time"

// Outfit represents a saved combination of wardrobe items.
type Outfit struct {
	uniqueEntity
	OwnerID   string
	Name      *string // optional
	Notes     *string // optional
	Items     []OutfitItem
	Photos    []OutfitPhoto
	CreatedAt time.Time
}
