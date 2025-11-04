package usecase

import (
	"context"

	"chatterstack/internal/domain/messages"
	"chatterstack/internal/domain/models"
)

// MessageUseCase orchestrates message-related workflows.
type MessageUseCase struct {
	MessageService messages.Service
}

// NewMessageUseCase constructs a new MessageUseCase instance.
func NewMessageUseCase(messageService messages.Service) *MessageUseCase {
	return &MessageUseCase{MessageService: messageService}
}

// Send delegates message creation to the domain service.
func (uc *MessageUseCase) Send(ctx context.Context, input messages.SendMessageInput) (*models.Message, error) {
	return uc.MessageService.Send(ctx, input)
}
