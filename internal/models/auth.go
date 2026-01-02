package models

import "time"

type Session struct {
	UserID    int64
	ExpiresAt time.Time
}

type User struct {
	Email        string
	Name         string
	PasswordHash string
	ID           int64
}
