package domain

import "time"

type User struct {
	uniqueEntity
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}
