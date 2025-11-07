package postgres

import (
	"context"
	"testing"

	pgxmock "github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"chatterstack/internal/domain/models"
)

func TestRoomRepository_Create(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewRoomRepository(poolWrapper)

	mockPool.ExpectBegin()
	mockPool.ExpectExec("INSERT INTO rooms").
		WithArgs(pgxmock.AnyArg(), "dev-room", true, "creator-1", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mockPool.ExpectExec("INSERT INTO room_members").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "creator-1", models.RoomRoleAdmin).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mockPool.ExpectCommit()

	room := &models.Room{
		Name:      "dev-room",
		IsGroup:   true,
		CreatedBy: "creator-1",
	}
	members := []models.RoomMember{
		{UserID: "creator-1", Role: models.RoomRoleAdmin},
	}

	require.NoError(t, repo.Create(ctx, room, members))
	require.NotEmpty(t, room.ID)
	require.False(t, room.CreatedAt.IsZero())

	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestRoomRepository_AddMember(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewRoomRepository(poolWrapper)

	mockPool.ExpectExec("INSERT INTO room_members").
		WithArgs(pgxmock.AnyArg(), "room-1", "user-2", models.RoomRoleMember).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	require.NoError(t, repo.AddMember(ctx, models.RoomMember{RoomID: "room-1", UserID: "user-2", Role: models.RoomRoleMember}))
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestRoomRepository_RemoveMember(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewRoomRepository(poolWrapper)

	mockPool.ExpectExec("DELETE FROM room_members").
		WithArgs("room-1", "user-2").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	require.NoError(t, repo.RemoveMember(ctx, "room-1", "user-2"))
	require.NoError(t, mockPool.ExpectationsWereMet())
}

func TestRoomRepository_ListMembers(t *testing.T) {
	ctx := context.Background()
	poolWrapper, mockPool := newPgxMockPool(t)
	repo := NewRoomRepository(poolWrapper)

	rows := pgxmock.NewRows([]string{"id", "room_id", "user_id", "role"}).
		AddRow("m1", "room-1", "user-1", models.RoomRoleAdmin).
		AddRow("m2", "room-1", "user-2", models.RoomRoleMember)

	mockPool.ExpectQuery("SELECT id, room_id, user_id, role FROM room_members WHERE room_id=").
		WithArgs("room-1").
		WillReturnRows(rows)

	members, err := repo.ListMembers(ctx, "room-1")
	require.NoError(t, err)
	require.Len(t, members, 2)
	require.Equal(t, "user-1", members[0].UserID)
	require.Equal(t, models.RoomRoleMember, members[1].Role)

	require.NoError(t, mockPool.ExpectationsWereMet())
}
