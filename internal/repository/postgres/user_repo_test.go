package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pgxmock "github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"chatterstack/internal/domain/models"
)

func TestUserRepository_Create(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewUserRepository(poolWrapper)

	mockPool.ExpectExec("INSERT INTO users").
		WithArgs(pgxmock.AnyArg(), "john", "john@example.com", "hash", models.UserStatusOnline, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	user := &models.User{
		Username:     "john",
		Email:        "john@example.com",
		PasswordHash: "hash",
		Status:       models.UserStatusOnline,
	}

	require.NoError(t, repo.Create(ctx, user))
	require.NotEmpty(t, user.ID)
	require.False(t, user.CreatedAt.IsZero())

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestUserRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewUserRepository(poolWrapper)

	createdAt := time.Now().UTC()
	lastSeen := createdAt.Add(-time.Minute)
	row := pgxmock.NewRows([]string{"id", "username", "email", "password_hash", "status", "last_seen_at", "created_at"}).
		AddRow("user-1", "john", "john@example.com", "hash", models.UserStatusOnline, lastSeen, createdAt)

	mockPool.ExpectQuery(`SELECT id, username, email, password_hash, status, last_seen_at, created_at FROM users WHERE id = \$1`).
		WithArgs("user-1").
		WillReturnRows(row)

	res, err := repo.GetByID(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, "user-1", res.ID)
	require.Equal(t, "john", res.Username)
	require.Equal(t, models.UserStatusOnline, res.Status)
	// timestamp comparisons should match expected values
	require.True(t, res.LastSeenAt.Equal(lastSeen))
	require.True(t, res.CreatedAt.Equal(createdAt))

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewUserRepository(poolWrapper)

	mockPool.ExpectQuery(`SELECT id, username, email, password_hash, status, last_seen_at, created_at FROM users WHERE id = \$1`).
		WithArgs("missing").
		WillReturnError(ErrNotFound)

	user, err := repo.GetByID(ctx, "missing")
	require.ErrorIs(t, err, ErrNotFound)
	require.Nil(t, user)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestUserRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewUserRepository(poolWrapper)

	lastSeen := time.Now().UTC()

	mockPool.ExpectExec("UPDATE users SET status=").
		WithArgs(models.UserStatusOffline, lastSeen, "user-1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	require.NoError(t, repo.UpdateStatus(ctx, "user-1", models.UserStatusOffline, lastSeen))
	require.NoError(t, mockPool.ExpectationsWereMet())
}

// Silence unused import warning if pgxpool isn't referenced elsewhere in tests.
var _ *pgxpool.Pool
