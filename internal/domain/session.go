package domain

import "time"

type Session struct {
	uniqueEntity
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}
