package rooms

import (
	"context"

	"chatterstack/internal/domain/models"
)

// Service defines group and direct room management operations.
type Service interface {
	Create(ctx context.Context, input CreateRoomInput) (*models.Room, error)
	AddMember(ctx context.Context, roomID, userID string, role models.RoomRole) error
	RemoveMember(ctx context.Context, roomID, userID string) error
	ListMembers(ctx context.Context, roomID string) ([]models.RoomMember, error)
}

// CreateRoomInput encapsulates attributes required to create a room.
type CreateRoomInput struct {
	Name      string
	IsGroup   bool
	CreatorID string
	Members   []RoomMemberInput
}

// RoomMemberInput expresses a membership relation request.
type RoomMemberInput struct {
	UserID string
	Role   models.RoomRole
}
