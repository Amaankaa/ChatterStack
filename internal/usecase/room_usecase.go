package usecase

import (
	"context"

	"chatterstack/internal/domain/models"
	"chatterstack/internal/domain/rooms"
)

type RoomUseCase struct {
	RoomService rooms.Service
}

func NewRoomUseCase(roomService rooms.Service) *RoomUseCase {
	return &RoomUseCase{RoomService: roomService}
}

func (uc *RoomUseCase) Create(ctx context.Context, input rooms.CreateRoomInput) (*models.Room, error) {
	return uc.RoomService.Create(ctx, input)
}

func (uc *RoomUseCase) AddMember(ctx context.Context, roomID, userID string, role models.RoomRole) error {
	return uc.RoomService.AddMember(ctx, roomID, userID, role)
}

func (uc *RoomUseCase) RemoveMember(ctx context.Context, roomID, userID string) error {
	return uc.RoomService.RemoveMember(ctx, roomID, userID)
}

func (uc *RoomUseCase) ListMembers(ctx context.Context, roomID string) ([]models.RoomMember, error) {
	return uc.RoomService.ListMembers(ctx, roomID)
}

func (uc *RoomUseCase) Search(ctx context.Context, userID, query string, limit int) ([]models.Room, error) {
	return uc.RoomService.Search(ctx, userID, query, limit)
}

func (uc *RoomUseCase) EnsureDirectRoom(ctx context.Context, userA, userB string) (*models.Room, bool, error) {
	return uc.RoomService.EnsureDirectRoom(ctx, userA, userB)
}
