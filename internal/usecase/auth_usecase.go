package usecase

import (
	"context"

	"chatterstack/internal/domain/auth"
	"chatterstack/internal/domain/models"
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
func (uc *AuthUseCase) Register(ctx context.Context, input auth.RegisterInput) (*models.User, error) {
	return uc.AuthService.Register(ctx, input)
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (*auth.TokenPair, error) {
	return uc.AuthService.Login(ctx, email, password)
}

func (uc *AuthUseCase) Refresh(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	return uc.AuthService.Refresh(ctx, refreshToken)
}

func (uc *AuthUseCase) Logout(ctx context.Context, userID string) error {
	return uc.AuthService.Logout(ctx, userID)
}

func (uc *AuthUseCase) ValidateAccessToken(ctx context.Context, accessToken string) (string, error) {
	return uc.AuthService.ValidateAccessToken(ctx, accessToken)
}
