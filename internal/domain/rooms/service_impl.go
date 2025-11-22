package rooms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"chatterstack/internal/domain/models"
)

var (
	ErrInvalidRoomName   = errors.New("rooms: name is required")
	ErrInvalidCreatorID  = errors.New("rooms: creator id is required")
	ErrInvalidMemberUser = errors.New("rooms: member user id is required")
	ErrInvalidUserID     = errors.New("rooms: user id is required")
	ErrDirectMessageSelf = errors.New("rooms: direct message requires two distinct users")
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 100
)

// roomRepository outlines the persistence operations needed by the domain service.
type roomRepository interface {
	Create(ctx context.Context, room *models.Room, members []models.RoomMember) error
	AddMember(ctx context.Context, member models.RoomMember) error
	RemoveMember(ctx context.Context, roomID, userID string) error
	ListMembers(ctx context.Context, roomID string) ([]models.RoomMember, error)
	Search(ctx context.Context, userID, query string, limit int) ([]models.Room, error)
	FindDirectRoom(ctx context.Context, userA, userB string) (*models.Room, error)
}

type service struct {
	repo roomRepository
}

// NewService wires the room domain service with its dependencies.
func NewService(repo roomRepository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, input CreateRoomInput) (*models.Room, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidRoomName
	}
	if strings.TrimSpace(input.CreatorID) == "" {
		return nil, ErrInvalidCreatorID
	}

	room := &models.Room{
		Name:      name,
		IsGroup:   input.IsGroup,
		CreatedBy: input.CreatorID,
	}

	members := buildMembers(input)
	if len(members) == 0 {
		return nil, fmt.Errorf("rooms: at least creator must be a member")
	}

	if err := s.repo.Create(ctx, room, members); err != nil {
		return nil, err
	}
	return room, nil
}

func (s *service) AddMember(ctx context.Context, roomID, userID string, role models.RoomRole) error {
	if strings.TrimSpace(roomID) == "" {
		return errors.New("rooms: room id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidMemberUser
	}
	if role == "" {
		role = models.RoomRoleMember
	}
	member := models.RoomMember{RoomID: roomID, UserID: userID, Role: role}
	return s.repo.AddMember(ctx, member)
}

func (s *service) RemoveMember(ctx context.Context, roomID, userID string) error {
	if strings.TrimSpace(roomID) == "" {
		return errors.New("rooms: room id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidMemberUser
	}
	return s.repo.RemoveMember(ctx, roomID, userID)
}

func (s *service) ListMembers(ctx context.Context, roomID string) ([]models.RoomMember, error) {
	if strings.TrimSpace(roomID) == "" {
		return nil, errors.New("rooms: room id is required")
	}
	return s.repo.ListMembers(ctx, roomID)
}

func (s *service) Search(ctx context.Context, userID, query string, limit int) ([]models.Room, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	query = strings.TrimSpace(query)

	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	return s.repo.Search(ctx, userID, query, limit)
}

func (s *service) EnsureDirectRoom(ctx context.Context, userA, userB string) (*models.Room, bool, error) {
	userA = strings.TrimSpace(userA)
	userB = strings.TrimSpace(userB)
	if userA == "" || userB == "" {
		return nil, false, ErrInvalidUserID
	}
	if userA == userB {
		return nil, false, ErrDirectMessageSelf
	}

	primary, secondary := normalizePair(userA, userB)
	if room, err := s.repo.FindDirectRoom(ctx, primary, secondary); err == nil {
		return room, false, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	generatedName := fmt.Sprintf("dm:%s:%s", primary, secondary)

	room, err := s.Create(ctx, CreateRoomInput{
		Name:      generatedName,
		IsGroup:   false,
		CreatorID: userA,
		Members: []RoomMemberInput{
			{UserID: userB, Role: models.RoomRoleMember},
		},
	})
	if err != nil {
		return nil, false, err
	}
	return room, true, nil
}

func normalizePair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

func buildMembers(input CreateRoomInput) []models.RoomMember {
	seen := map[string]models.RoomMember{}

	creator := models.RoomMember{UserID: input.CreatorID, Role: models.RoomRoleAdmin}
	seen[creator.UserID] = creator

	for _, m := range input.Members {
		userID := strings.TrimSpace(m.UserID)
		if userID == "" {
			continue
		}
		role := m.Role
		if role == "" {
			role = models.RoomRoleMember
		}
		if _, exists := seen[userID]; !exists {
			seen[userID] = models.RoomMember{UserID: userID, Role: role}
		}
	}

	members := make([]models.RoomMember, 0, len(seen))
	for _, m := range seen {
		members = append(members, m)
	}
	return members
}
