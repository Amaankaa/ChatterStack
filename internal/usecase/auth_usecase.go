package usecase

import (
	"context"

	"chatterstack/internal/domain/auth"
)

// AuthUseCase orchestrates authentication workflows.
type AuthUseCase struct {
	AuthService auth.Service
}

// NewAuthUseCase constructs a new AuthUseCase instance.
func NewAuthUseCase(authService auth.Service) *AuthUseCase {
	return &AuthUseCase{AuthService: authService}
}

// Register delegates to the domain service.
func (uc *AuthUseCase) Register(ctx context.Context, input auth.RegisterInput) error {
	_, err := uc.AuthService.Register(ctx, input)
	return err
}
