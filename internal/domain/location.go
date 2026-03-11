package domain

import "time"

type Location struct {
	uniqueEntity
	OwnerID   string
	ParentID  *string // optional (nil = root location)
	Label     string
	CreatedAt time.Time
}
