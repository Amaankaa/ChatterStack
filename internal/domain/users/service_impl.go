package users

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"chatterstack/internal/domain/models"
)

var ErrUserNotFound = errors.New("users: not found")

// userRepository outlines the persistence operations needed by the domain service.
type userRepository interface {
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateStatus(ctx context.Context, id string, status models.UserStatus, lastSeen time.Time) error
}

type service struct {
	repo userRepository
}

// NewService wires the user domain service with its dependencies.
func NewService(repo userRepository) Service {
	return &service{repo: repo}
}

func (s *service) GetByID(ctx context.Context, id string) (*models.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *service) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *service) UpdateStatus(ctx context.Context, id string, status models.UserStatus) error {
	if id == "" {
		return errors.New("users: id cannot be empty")
	}
	if status == "" {
		return errors.New("users: status cannot be empty")
	}
	return s.repo.UpdateStatus(ctx, id, status, time.Now().UTC())
}
