package usecase

import (
	"context"

	"chatterstack/internal/domain/models"
	"chatterstack/internal/domain/users"
)

type UserUseCase struct {
	UserService users.Service
}

func NewUserUseCase(userService users.Service) *UserUseCase {
	return &UserUseCase{UserService: userService}
}

func (uc *UserUseCase) GetProfile(ctx context.Context, id string) (*models.User, error) {
	return uc.UserService.GetByID(ctx, id)
}

func (uc *UserUseCase) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return uc.UserService.GetByEmail(ctx, email)
}

func (uc *UserUseCase) UpdateStatus(ctx context.Context, id string, status models.UserStatus) error {
	return uc.UserService.UpdateStatus(ctx, id, status)
}
