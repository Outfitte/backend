package domain

import "time"

// OutfitLog records a single outfit wear event on a calendar date.
// WearLogIDs is populated at read time from the outfit_log_wear_logs join
// table — it is not persisted on the struct itself.
type OutfitLog struct {
	uniqueEntity
	OutfitID   string
	OwnerID    string
	WornOn     time.Time // calendar date, stored as YYYY-MM-DD
	Notes      *string   // optional
	WearLogIDs []string  // IDs of auto-generated wear logs linked via join table
	CreatedAt  time.Time
}
