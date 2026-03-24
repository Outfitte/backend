package domain

import "time"

// OutfitPhoto represents a photo attached to an outfit.
type OutfitPhoto struct {
	ID        string
	MediaKey  string
	Position  int
	CreatedAt time.Time
}
