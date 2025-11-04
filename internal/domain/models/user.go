package models

import "time"

// User represents an authenticated account within the system.
type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	Status       UserStatus
	LastSeenAt   time.Time
	CreatedAt    time.Time
}

// UserStatus indicates the online presence state tracked by the platform.
type UserStatus string

const (
	// UserStatusOnline marks an actively connected user.
	UserStatusOnline UserStatus = "online"
	// UserStatusOffline marks a user with no live connection.
	UserStatusOffline UserStatus = "offline"
)
