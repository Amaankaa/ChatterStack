package messages

import (
	"context"

	"chatterstack/internal/domain/models"
)

// Service defines messaging related business capabilities.
type Service interface {
	Send(ctx context.Context, payload SendMessageInput) (*models.Message, error)
	ListByRoom(ctx context.Context, roomID string, page, limit int) ([]models.Message, error)
	MarkDelivered(ctx context.Context, messageID, userID string) error
	MarkRead(ctx context.Context, messageID, userID string) error
}

// SendMessageInput captures the request payload to produce a message entry.
type SendMessageInput struct {
	RoomID      string
	SenderID    string
	Content     string
	Attachments []models.Attachment
}
