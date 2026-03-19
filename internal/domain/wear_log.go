package domain

import "time"

// WearLog records a single wear event for a wardrobe item.
// WornOn is a calendar date; store as YYYY-MM-DD string in the database.
type WearLog struct {
	uniqueEntity
	ItemID    string
	OwnerID   string
	WornOn    time.Time
	Notes     *string
	CreatedAt time.Time
}
