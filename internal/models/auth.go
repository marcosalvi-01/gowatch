package models

import "time"

type Session struct {
	UserID    int64
	ExpiresAt time.Time
}

type User struct {
	ID                    int64
	Email                 string
	Name                  string
	PasswordHash          string
	Admin                 bool
	CreatedAt             *time.Time
	PasswordResetRequired bool
}

type UserWithStats struct {
	User
	WatchedCount int64
	ListCount    int64
}
