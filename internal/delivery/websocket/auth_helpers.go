package websocket

import (
	"context"
	"log"
	"strings"

	"chatterstack/internal/domain/models"
)

// RoomMemberLister describes the subset of room use case functionality needed for authorization.
type RoomMemberLister interface {
	ListMembers(ctx context.Context, roomID string) ([]models.RoomMember, error)
}

// ExtractBearerToken pulls the JWT token component from an Authorization header.
func ExtractBearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// FilterAuthorizedRooms retains only rooms the user belongs to, logging lookups that fail.
func FilterAuthorizedRooms(ctx context.Context, lister RoomMemberLister, userID string, roomIDs []string) []string {
	authorized := make([]string, 0, len(roomIDs))
	for _, roomID := range roomIDs {
		members, err := lister.ListMembers(ctx, roomID)
		if err != nil {
			log.Printf("websocket: list members for %s failed: %v", roomID, err)
			continue
		}
		for _, member := range members {
			if member.UserID == userID {
				authorized = append(authorized, roomID)
				break
			}
		}
	}
	return authorized
}
