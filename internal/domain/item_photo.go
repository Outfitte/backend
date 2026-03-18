package domain

import "time"

// ItemPhoto represents a photo attached to a wardrobe item.
type ItemPhoto struct {
	ID        string
	MediaKey  string
	Position  int
	CreatedAt time.Time
}
