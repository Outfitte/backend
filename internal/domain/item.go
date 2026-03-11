package domain

import "time"

type Item struct {
	uniqueEntity
	OwnerID       string
	Name          string
	Brand         string
	CategoryID    string
	Color         string
	Size          string
	PhotoKeys     []string
	LocationID    *string    // optional
	PurchasePrice *string    // optional, deferred to M4+; string to avoid decimal dep
	PurchaseDate  *time.Time // optional, deferred to M4+
	CreatedAt     time.Time
}
