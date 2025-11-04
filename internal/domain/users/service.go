package users

import (
	"context"

	"chatterstack/internal/domain/models"
)

// Service encapsulates domain operations for user management.
type Service interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateStatus(ctx context.Context, id string, status models.UserStatus) error
}
