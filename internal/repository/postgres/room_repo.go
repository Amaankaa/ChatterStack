package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"chatterstack/internal/domain/models"
)

// RoomRepository implements persistence for chat rooms.
type RoomRepository struct {
	pool PgxPool
}

// NewRoomRepository constructs a new RoomRepository.
func NewRoomRepository(pool PgxPool) *RoomRepository {
	return &RoomRepository{pool: pool}
}

// Placeholder methods until implementation is added.
func (r *RoomRepository) Create(ctx context.Context, room *models.Room, members []models.RoomMember) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer rollback(ctx, tx)
	if room.ID == "" {
		room.ID = uuid.New().String()
	}
	if room.CreatedAt.IsZero() {
		room.CreatedAt = time.Now().UTC()
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO rooms (id, name, is_group, created_by, created_at) VALUES ($1, $2, $3, $4, $5)`, room.ID, room.Name, room.IsGroup, room.CreatedBy, room.CreatedAt); err != nil {
		return err
	}

	for i := range members {
		if members[i].ID == "" {
			members[i].ID = uuid.New().String()
		}
		members[i].RoomID = room.ID
		if _, err := tx.Exec(ctx,
			`INSERT INTO room_members (id, room_id, user_id, role) VALUES ($1, $2, $3, $4)`, members[i].ID, members[i].RoomID, members[i].UserID, members[i].Role); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *RoomRepository) AddMember(ctx context.Context, member models.RoomMember) error {
	if member.ID == "" {
		member.ID = uuid.New().String()
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO room_members (id, room_id, user_id, role) VALUES ($1, $2, $3, $4) ON CONFLICT (room_id, user_id) DO UPDATE SET role = EXCLUDED.role`, member.ID, member.RoomID, member.UserID, member.Role)

	return err
}

func (r *RoomRepository) RemoveMember(ctx context.Context, roomID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM room_members WHERE room_id=$1 AND user_id=$2`, roomID, userID)
	return err
}

func (r *RoomRepository) GetByID(ctx context.Context, roomID string) (*models.Room, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, is_group, created_by, created_at FROM rooms WHERE id=$1`, roomID)

	var room models.Room
	if err := row.Scan(&room.ID, &room.Name, &room.IsGroup, &room.CreatedBy, &room.CreatedAt); err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *RoomRepository) Delete(ctx context.Context, roomID string) error {
	cmdTag, err := r.pool.Exec(ctx,
		`DELETE FROM rooms WHERE id=$1`, roomID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *RoomRepository) ListMembers(ctx context.Context, roomID string) ([]models.RoomMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, room_id, user_id, role FROM room_members WHERE room_id=$1`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.RoomMember
	for rows.Next() {
		var m models.RoomMember
		if err := rows.Scan(&m.ID, &m.RoomID, &m.UserID, &m.Role); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *RoomRepository) Search(ctx context.Context, userID, query string, limit int) ([]models.Room, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.name, r.is_group, r.created_by, r.created_at
			FROM rooms r
			JOIN room_members m ON r.id = m.room_id
			WHERE m.user_id = $1 AND ($2 = '' OR r.name ILIKE '%' || $2 || '%')
			ORDER BY r.created_at DESC
			LIMIT $3`, userID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]models.Room, 0)
	for rows.Next() {
		var room models.Room
		if err := rows.Scan(&room.ID, &room.Name, &room.IsGroup, &room.CreatedBy, &room.CreatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	return rooms, rows.Err()
}

func (r *RoomRepository) FindDirectRoom(ctx context.Context, userA, userB string) (*models.Room, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT r.id, r.name, r.is_group, r.created_by, r.created_at
		FROM rooms r
		JOIN room_members m ON r.id = m.room_id
		WHERE r.is_group = FALSE
			AND (SELECT COUNT(*) FROM room_members WHERE room_id = r.id) = 2
			AND m.user_id::text = ANY($1::text[])
			GROUP BY r.id, r.name, r.is_group, r.created_by, r.created_at
			HAVING COUNT(DISTINCT m.user_id) = 2
			LIMIT 1`, []string{userA, userB})

	var room models.Room
	if err := row.Scan(&room.ID, &room.Name, &room.IsGroup, &room.CreatedBy, &room.CreatedAt); err != nil {
		return nil, err
	}
	return &room, nil
}
