package postgres

import (
	"context"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"chatterstack/internal/domain/models"
)

func TestMessageRepository_Create(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewMessageRepository(poolWrapper)

	mockPool.ExpectExec("INSERT INTO messages").
		WithArgs(pgxmock.AnyArg(), "room-1", "user-1", "hello", models.MessageStatusSent, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	msg := &models.Message{
		RoomID:   "room-1",
		SenderID: "user-1",
		Content:  "hello",
	}

	require.NoError(t, repo.Create(ctx, msg))
	require.NotEmpty(t, msg.ID)
	require.False(t, msg.CreatedAt.IsZero())
	require.Equal(t, models.MessageStatusSent, msg.Status)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageRepository_ListByRoom(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewMessageRepository(poolWrapper)

	created := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"id", "room_id", "sender_id", "content", "status", "created_at"}).
		AddRow("m1", "room-1", "user-1", "hello", models.MessageStatusSent, created).
		AddRow("m2", "room-1", "user-2", "hi", models.MessageStatusDelivered, created.Add(time.Minute))

	mockPool.ExpectQuery(`SELECT id, room_id, sender_id, content, status, created_at\s+FROM messages`).
		WithArgs("room-1", 2, 0).
		WillReturnRows(rows)

	msgs, err := repo.ListByRoom(ctx, "room-1", 2, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.Equal(t, "m1", msgs[0].ID)
	require.Equal(t, models.MessageStatusDelivered, msgs[1].Status)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewMessageRepository(poolWrapper)

	mockPool.ExpectExec("UPDATE messages SET status=").
		WithArgs(models.MessageStatusRead, "msg-1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	require.NoError(t, repo.UpdateStatus(ctx, "msg-1", models.MessageStatusRead))
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageRepository_UpsertReceipt(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewMessageRepository(poolWrapper)

	mockPool.ExpectExec("INSERT INTO message_receipts").
		WithArgs(pgxmock.AnyArg(), "msg-1", "user-1", models.MessageStatusDelivered, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	receipt := &models.MessageReceipt{MessageID: "msg-1", UserID: "user-1", Status: models.MessageStatusDelivered}
	require.NoError(t, repo.UpsertReceipt(ctx, receipt))
	require.NotEmpty(t, receipt.ID)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageRepository_Search(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewMessageRepository(poolWrapper)

	created := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"id", "room_id", "sender_id", "content", "status", "created_at"}).
		AddRow("m1", "room-1", "user-2", "daily sync", models.MessageStatusSent, created)

	mockPool.ExpectQuery(`SELECT m.id, m.room_id, m.sender_id, m.content, m.status, m.created_at`).
		WithArgs("user-1", "room-1", "daily", 10).
		WillReturnRows(rows)

	msgs, err := repo.Search(ctx, "user-1", "room-1", "daily", 10)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, "m1", msgs[0].ID)
	require.Equal(t, created, msgs[0].CreatedAt)

	require.NoError(t, mockPool.ExpectationsWereMet())
}
