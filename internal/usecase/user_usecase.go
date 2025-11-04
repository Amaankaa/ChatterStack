package usecase

import (
	"context"

	"chatterstack/internal/domain/models"
	"chatterstack/internal/domain/users"
)

// UserUseCase orchestrates user-related workflows.
type UserUseCase struct {
	UserService users.Service
}

// NewUserUseCase constructs a new UserUseCase instance.
func NewUserUseCase(userService users.Service) *UserUseCase {
	return &UserUseCase{UserService: userService}
}

// GetProfile returns the user profile for a given ID.
func (uc *UserUseCase) GetProfile(ctx context.Context, id string) (*models.User, error) {
	return uc.UserService.GetByID(ctx, id)
}
