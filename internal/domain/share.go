package domain

import "time"

// ShareTargetType identifies the kind of entity being shared.
type ShareTargetType string

const (
	ShareTargetItem     ShareTargetType = "item"
	ShareTargetOutfit   ShareTargetType = "outfit"
	ShareTargetLocation ShareTargetType = "location"
)

// Share represents a granular sharing grant from one user to another for a specific entity.
type Share struct {
	uniqueEntity
	OwnerID     string
	RecipientID string
	TargetType  ShareTargetType
	TargetID    string
	CreatedAt   time.Time
}
