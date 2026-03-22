package domain

import "time"

type Item struct {
	uniqueEntity
	OwnerID       string
	Name          string
	Brand         *string    // optional
	CategoryID    *string    // optional; nil = uncategorised
	Color         *string    // optional
	Metadata      ItemMetadata
	Photos        []ItemPhoto
	LocationID    *string    // optional
	PurchasePrice *string    // optional, deferred to M4+; string to avoid decimal dep
	PurchaseDate  *time.Time // optional, deferred to M4+
	CreatedAt     time.Time
	// WearCount and LastWornAt are maintained by WearLogService — not set directly by the user.
	WearCount  int
	LastWornAt *time.Time
}
