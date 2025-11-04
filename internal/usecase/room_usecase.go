package usecase

import (
	"context"

	"chatterstack/internal/domain/models"
	"chatterstack/internal/domain/rooms"
)

// RoomUseCase orchestrates room-related workflows.
type RoomUseCase struct {
	RoomService rooms.Service
}

// NewRoomUseCase constructs a new RoomUseCase instance.
func NewRoomUseCase(roomService rooms.Service) *RoomUseCase {
	return &RoomUseCase{RoomService: roomService}
}

// Create delegates room creation to the domain service.
func (uc *RoomUseCase) Create(ctx context.Context, input rooms.CreateRoomInput) (*models.Room, error) {
	return uc.RoomService.Create(ctx, input)
}
