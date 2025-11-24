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
		WithArgs(pgxmock.AnyArg(), "room-1", "user-1", "hello", models.MessageStatusSent, pgxmock.AnyArg(), pgxmock.AnyArg()).
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
	rows := pgxmock.NewRows([]string{"id", "room_id", "sender_id", "content", "status", "created_at", "updated_at"}).
		AddRow("m1", "room-1", "user-1", "hello", models.MessageStatusSent, created, created).
		AddRow("m2", "room-1", "user-2", "hi", models.MessageStatusDelivered, created.Add(time.Minute), created.Add(time.Minute))

	mockPool.ExpectQuery(`SELECT id, room_id, sender_id, content, status, created_at, updated_at\s+FROM messages`).
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
	rows := pgxmock.NewRows([]string{"id", "room_id", "sender_id", "content", "status", "created_at", "updated_at"}).
		AddRow("m1", "room-1", "user-2", "daily sync", models.MessageStatusSent, created, created)

	mockPool.ExpectQuery(`SELECT m.id, m.room_id, m.sender_id, m.content, m.status, m.created_at, m.updated_at`).
		WithArgs("user-1", "room-1", "daily", 10).
		WillReturnRows(rows)

	msgs, err := repo.Search(ctx, "user-1", "room-1", "daily", 10)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, "m1", msgs[0].ID)
	require.Equal(t, created, msgs[0].CreatedAt)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageRepository_ListAround(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewMessageRepository(poolWrapper)

	targetCreated := time.Date(2025, 11, 22, 12, 0, 0, 0, time.UTC)
	targetRow := pgxmock.NewRows([]string{"id", "room_id", "sender_id", "content", "status", "created_at", "updated_at"}).
		AddRow("msg-target", "room-1", "user-1", "target", models.MessageStatusSent, targetCreated, targetCreated)

	mockPool.ExpectQuery("SELECT id, room_id, sender_id, content, status, created_at, updated_at\\s+FROM messages\\s+WHERE id =").
		WithArgs("msg-target").
		WillReturnRows(targetRow)

	beforeCreated := targetCreated.Add(-time.Minute)
	beforeRows := pgxmock.NewRows([]string{"id", "room_id", "sender_id", "content", "status", "created_at", "updated_at"}).
		AddRow("msg-before", "room-1", "user-2", "before", models.MessageStatusDelivered, beforeCreated, beforeCreated)

	mockPool.ExpectQuery("SELECT id, room_id, sender_id, content, status, created_at, updated_at\\s+FROM messages\\s+WHERE room_id =").
		WithArgs("room-1", targetCreated, 1).
		WillReturnRows(beforeRows)

	afterCreated := targetCreated.Add(time.Minute)
	afterRows := pgxmock.NewRows([]string{"id", "room_id", "sender_id", "content", "status", "created_at", "updated_at"}).
		AddRow("msg-after", "room-1", "user-3", "after", models.MessageStatusRead, afterCreated, afterCreated)

	mockPool.ExpectQuery("SELECT id, room_id, sender_id, content, status, created_at, updated_at\\s+FROM messages\\s+WHERE room_id =").
		WithArgs("room-1", targetCreated, 1).
		WillReturnRows(afterRows)

	msgs, err := repo.ListAround(ctx, "room-1", "msg-target", 1, 1)
	require.NoError(t, err)
	require.Len(t, msgs, 3)
	require.Equal(t, "msg-before", msgs[0].ID)
	require.Equal(t, "msg-target", msgs[1].ID)
	require.Equal(t, "msg-after", msgs[2].ID)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewMessageRepository(poolWrapper)

	created := time.Now().UTC()
	row := pgxmock.NewRows([]string{"id", "room_id", "sender_id", "content", "status", "created_at", "updated_at"}).
		AddRow("msg-1", "room-1", "user-1", "hello", models.MessageStatusSent, created, created.Add(time.Second))

	mockPool.ExpectQuery("SELECT id, room_id, sender_id, content, status, created_at, updated_at\\s+FROM messages\\s+WHERE id =").
		WithArgs("msg-1").
		WillReturnRows(row)

	msg, err := repo.GetByID(ctx, "msg-1")
	require.NoError(t, err)
	require.Equal(t, "msg-1", msg.ID)
	require.Equal(t, created.Add(time.Second), msg.UpdatedAt)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageRepository_UpdateContent(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewMessageRepository(poolWrapper)

	created := time.Now().UTC()
	row := pgxmock.NewRows([]string{"id", "room_id", "sender_id", "content", "status", "created_at", "updated_at"}).
		AddRow("msg-1", "room-1", "user-1", "updated", models.MessageStatusSent, created, created.Add(time.Minute))

	mockPool.ExpectQuery(`UPDATE messages\s+SET content = \$2, updated_at = NOW\(\)\s+WHERE id = \$1\s+RETURNING`).
		WithArgs("msg-1", "updated").
		WillReturnRows(row)

	msg, err := repo.UpdateContent(ctx, "msg-1", "updated")
	require.NoError(t, err)
	require.Equal(t, "updated", msg.Content)

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestMessageRepository_Delete(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewMessageRepository(poolWrapper)

	mockPool.ExpectExec("DELETE FROM messages WHERE id =").
		WithArgs("msg-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	require.NoError(t, repo.Delete(ctx, "msg-1"))
	require.NoError(t, mockPool.ExpectationsWereMet())
}
