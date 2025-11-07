package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"chatterstack/internal/domain/models"
)

// UserRepository implements persistence for users via PostgreSQL.
type UserRepository struct {
	pool PgxPool
}

// NewUserRepository constructs a new UserRepository.
func NewUserRepository(pool PgxPool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, username, email, password_hash, status, last_seen_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Status, user.LastSeenAt, user.CreatedAt,
	)

	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, username, email, password_hash, status, last_seen_at, created_at FROM users WHERE id = $1`, id)
	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Status, &u.LastSeenAt, &u.CreatedAt); err != nil {
		if err == ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, username, email, password_hash, status, last_seen_at, created_at FROM users WHERE email = $1`, email)
	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Status, &u.LastSeenAt, &u.CreatedAt); err != nil {
		if err == ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) UpdateStatus(ctx context.Context, id string, status models.UserStatus, lastSeen time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET status=$1, last_seen_at=$2 WHERE id = $3`, status, lastSeen, id)
	return err
}
