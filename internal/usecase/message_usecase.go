package usecase

import (
	"context"

	"chatterstack/internal/domain/messages"
	"chatterstack/internal/domain/models"
)

type MessageUseCase struct {
	MessageService messages.Service
}

func NewMessageUseCase(messageService messages.Service) *MessageUseCase {
	return &MessageUseCase{MessageService: messageService}
}

func (uc *MessageUseCase) Send(ctx context.Context, input messages.SendMessageInput) (*models.Message, error) {
	return uc.MessageService.Send(ctx, input)
}

func (uc *MessageUseCase) ListByRoom(ctx context.Context, roomID string, page, limit int) ([]models.Message, error) {
	return uc.MessageService.ListByRoom(ctx, roomID, page, limit)
}

func (uc *MessageUseCase) MarkDelivered(ctx context.Context, messageID, userID string) error {
	return uc.MessageService.MarkDelivered(ctx, messageID, userID)
}

func (uc *MessageUseCase) MarkRead(ctx context.Context, messageID, userID string) error {
	return uc.MessageService.MarkRead(ctx, messageID, userID)
}

func (uc *MessageUseCase) Search(ctx context.Context, userID, roomID, query string, limit int) ([]models.Message, error) {
	return uc.MessageService.Search(ctx, userID, roomID, query, limit)
}
