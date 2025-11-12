package auth

import (
	"context"

	"chatterstack/internal/domain/models"
)

// Service describes the business capabilities for authentication flows.
type Service interface {
	Register(ctx context.Context, input RegisterInput) (*models.User, error)
	Login(ctx context.Context, email, password string) (*TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, userID string) error
	ValidateAccessToken(ctx context.Context, accessToken string) (string, error)
}

// RegisterInput captures the minimum information required for signup.
type RegisterInput struct {
	Username string
	Email    string
	Password string
}

// TokenPair represents an access and refresh token bundle.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
