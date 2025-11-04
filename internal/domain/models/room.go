package models

import "time"

// Room represents a conversation context (direct or group).
type Room struct {
	ID        string
	Name      string
	IsGroup   bool
	CreatedBy string
	CreatedAt time.Time
}

// RoomMember associates a user with a role inside a room.
type RoomMember struct {
	ID     string
	RoomID string
	UserID string
	Role   RoomRole
}

// RoomRole defines permissions within a room.
type RoomRole string

const (
	// RoomRoleAdmin grants management privileges to a member.
	RoomRoleAdmin RoomRole = "admin"
	// RoomRoleMember grants standard participation privileges.
	RoomRoleMember RoomRole = "member"
)
