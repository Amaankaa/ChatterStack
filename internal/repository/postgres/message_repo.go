package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"chatterstack/internal/domain/models"
)

// MessageRepository implements persistence for chat messages.
type MessageRepository struct {
	pool PgxPool
}

// NewMessageRepository constructs a new MessageRepository.
func NewMessageRepository(pool PgxPool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

func (r *MessageRepository) Create(ctx context.Context, msg *models.Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	if msg.Status == "" {
		msg.Status = models.MessageStatusSent
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO messages (id, room_id, sender_id, content, status, created_at) VALUES ($1, $2, $3, $4, $5, $6)`, msg.ID, msg.RoomID, msg.SenderID, msg.Content, msg.Status, msg.CreatedAt)
	return err
}

func (r *MessageRepository) ListByRoom(ctx context.Context, roomID string, limit, offset int) ([]models.Message, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, room_id, sender_id, content, status, created_at
		FROM messages
		WHERE room_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, roomID, limit, offset)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.RoomID, &m.SenderID, &m.Content, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *MessageRepository) ListAround(ctx context.Context, roomID, messageID string, before, after int) ([]models.Message, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, room_id, sender_id, content, status, created_at
		FROM messages
		WHERE id = $1`, messageID)

	var target models.Message
	if err := row.Scan(&target.ID, &target.RoomID, &target.SenderID, &target.Content, &target.Status, &target.CreatedAt); err != nil {
		return nil, err
	}

	if target.RoomID != roomID {
		return nil, pgx.ErrNoRows
	}

	beforeMsgs := make([]models.Message, 0, before)
	if before > 0 {
		rows, err := r.pool.Query(ctx,
			`SELECT id, room_id, sender_id, content, status, created_at
			FROM messages
			WHERE room_id = $1 AND created_at < $2
			ORDER BY created_at DESC
			LIMIT $3`, roomID, target.CreatedAt, before)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var m models.Message
			if err := rows.Scan(&m.ID, &m.RoomID, &m.SenderID, &m.Content, &m.Status, &m.CreatedAt); err != nil {
				rows.Close()
				return nil, err
			}
			beforeMsgs = append(beforeMsgs, m)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
		// reverse to chronological order
		for i, j := 0, len(beforeMsgs)-1; i < j; i, j = i+1, j-1 {
			beforeMsgs[i], beforeMsgs[j] = beforeMsgs[j], beforeMsgs[i]
		}
	}

	afterMsgs := make([]models.Message, 0, after)
	if after > 0 {
		rows, err := r.pool.Query(ctx,
			`SELECT id, room_id, sender_id, content, status, created_at
			FROM messages
			WHERE room_id = $1 AND created_at > $2
			ORDER BY created_at ASC
			LIMIT $3`, roomID, target.CreatedAt, after)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var m models.Message
			if err := rows.Scan(&m.ID, &m.RoomID, &m.SenderID, &m.Content, &m.Status, &m.CreatedAt); err != nil {
				rows.Close()
				return nil, err
			}
			afterMsgs = append(afterMsgs, m)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	msgs := make([]models.Message, 0, len(beforeMsgs)+1+len(afterMsgs))
	msgs = append(msgs, beforeMsgs...)
	msgs = append(msgs, target)
	msgs = append(msgs, afterMsgs...)

	return msgs, nil
}

func (r *MessageRepository) UpdateStatus(ctx context.Context, id string, status models.MessageStatus) error {
	_, err := r.pool.Exec(ctx, `UPDATE messages SET status=$1 WHERE id=$2`, status, id)
	return err
}

// UpsertReceipt records or updates a message receipt for a user.
func (r *MessageRepository) UpsertReceipt(ctx context.Context, receipt *models.MessageReceipt) error {
	if receipt.ID == "" {
		receipt.ID = uuid.New().String()
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO message_receipts (id, message_id, user_id, status, seen_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (message_id, user_id) DO UPDATE SET status = EXCLUDED.status, seen_at = EXCLUDED.seen_at`,
		receipt.ID, receipt.MessageID, receipt.UserID, receipt.Status, receipt.SeenAt,
	)
	return err
}

func (r *MessageRepository) Search(ctx context.Context, userID, roomID, query string, limit int) ([]models.Message, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT m.id, m.room_id, m.sender_id, m.content, m.status, m.created_at
		FROM messages m
		JOIN room_members rm ON m.room_id = rm.room_id
		WHERE rm.user_id = $1 AND ($2 = '' OR m.room_id::text = $2)
		AND m.content ILIKE '%' || $3 || '%'
		ORDER BY m.created_at DESC
		LIMIT $4`, userID, roomID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.RoomID, &m.SenderID, &m.Content, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}

	return msgs, rows.Err()
}
