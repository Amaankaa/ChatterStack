package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"chatterstack/internal/domain/models"
	"chatterstack/internal/domain/rooms"
	"chatterstack/internal/usecase"
	"chatterstack/pkg/middleware"
)

type mockRoomService struct {
	mock.Mock
}

func (m *mockRoomService) Create(ctx context.Context, input rooms.CreateRoomInput) (*models.Room, error) {
	args := m.Called(ctx, input)
	room, _ := args.Get(0).(*models.Room)
	return room, args.Error(1)
}

func (m *mockRoomService) AddMember(ctx context.Context, roomID, userID string, role models.RoomRole) error {
	args := m.Called(ctx, roomID, userID, role)
	return args.Error(0)
}

func (m *mockRoomService) RemoveMember(ctx context.Context, roomID, userID string) error {
	args := m.Called(ctx, roomID, userID)
	return args.Error(0)
}

func (m *mockRoomService) ListMembers(ctx context.Context, roomID string) ([]models.RoomMember, error) {
	args := m.Called(ctx, roomID)
	if res := args.Get(0); res != nil {
		return res.([]models.RoomMember), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRoomService) Search(ctx context.Context, userID, query string, limit int) ([]models.Room, error) {
	args := m.Called(ctx, userID, query, limit)
	if res := args.Get(0); res != nil {
		return res.([]models.Room), args.Error(1)
	}
	return nil, args.Error(1)
}

type RoomHandlerTestSuite struct {
	suite.Suite

	service *mockRoomService
	router  *gin.Engine
}

func (s *RoomHandlerTestSuite) SetupTest() {
	s.service = new(mockRoomService)
	h := NewRoomHandler(usecase.NewRoomUseCase(s.service))

	s.router = gin.New()
	routes := s.router.Group("/rooms")
	routes.Use(func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, "user-1")
	})
	h.RegisterRoutes(routes)
}

func (s *RoomHandlerTestSuite) TearDownTest() {
	s.service.AssertExpectations(s.T())
}

func (s *RoomHandlerTestSuite) TestSearchSuccess() {
	roomsResult := []models.Room{{ID: "room-1", Name: "Daily", CreatedBy: "user-1", CreatedAt: time.Unix(0, 0)}}
	s.service.On("Search", mock.Anything, "user-1", "daily", 5).Return(roomsResult, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/rooms/?q=daily&limit=5", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusOK, res.Code)

	var payload struct {
		Rooms []roomPayload `json:"rooms"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &payload))
	s.Len(payload.Rooms, 1)
	s.Equal("room-1", payload.Rooms[0].ID)
}

func (s *RoomHandlerTestSuite) TestSearchInvalidLimit() {
	req := httptest.NewRequest(http.MethodGet, "/rooms/?limit=abc", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
}

func (s *RoomHandlerTestSuite) TestSearchServiceError() {
	s.service.On("Search", mock.Anything, "user-1", "", 0).Return(([]models.Room)(nil), errors.New("rooms: unexpected error")).Once()

	req := httptest.NewRequest(http.MethodGet, "/rooms/", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), "rooms: unexpected error")
}

func (s *RoomHandlerTestSuite) TestCreateSuccess() {
	createdAt := time.Date(2025, 11, 8, 11, 0, 0, 0, time.UTC)
	input := rooms.CreateRoomInput{
		Name:      "General",
		IsGroup:   true,
		CreatorID: "user-1",
		Members: []rooms.RoomMemberInput{
			{UserID: "user-2", Role: models.RoomRoleMember},
		},
	}
	room := &models.Room{ID: "room-1", Name: input.Name, IsGroup: input.IsGroup, CreatedBy: input.CreatorID, CreatedAt: createdAt}

	s.service.On("Create", mock.Anything, input).Return(room, nil).Once()

	payload, _ := json.Marshal(map[string]any{
		"name":       input.Name,
		"is_group":   input.IsGroup,
		"creator_id": input.CreatorID,
		"members": []map[string]any{{
			"user_id": input.Members[0].UserID,
			"role":    string(input.Members[0].Role),
		}},
	})

	req := httptest.NewRequest(http.MethodPost, "/rooms/", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusCreated, res.Code)

	var resp roomPayload
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &resp))
	s.Equal(room.ID, resp.ID)
	s.Equal(room.Name, resp.Name)
	s.Equal(room.CreatedBy, resp.CreatedBy)
	s.Equal(room.CreatedAt, resp.CreatedAt)
}

func (s *RoomHandlerTestSuite) TestCreateDomainError() {
	matcher := mock.MatchedBy(func(input rooms.CreateRoomInput) bool {
		return input.Name == "" && input.CreatorID == "" && len(input.Members) == 0
	})
	s.service.On("Create", mock.Anything, matcher).Return((*models.Room)(nil), rooms.ErrInvalidRoomName).Once()

	req := httptest.NewRequest(http.MethodPost, "/rooms/", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), rooms.ErrInvalidRoomName.Error())
}

func (s *RoomHandlerTestSuite) TestCreateInvalidJSON() {
	req := httptest.NewRequest(http.MethodPost, "/rooms/", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
}

func (s *RoomHandlerTestSuite) TestAddMemberSuccess() {
	s.service.On("AddMember", mock.Anything, "room-1", "user-1", models.RoomRoleAdmin).Return(nil).Once()

	payload, _ := json.Marshal(map[string]any{"user_id": "user-1", "role": string(models.RoomRoleAdmin)})
	req := httptest.NewRequest(http.MethodPost, "/rooms/room-1/members", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusNoContent, res.Code)
}

func (s *RoomHandlerTestSuite) TestAddMemberDomainError() {
	s.service.On("AddMember", mock.Anything, "room-1", "", models.RoomRoleMember).Return(rooms.ErrInvalidMemberUser).Once()

	payload, _ := json.Marshal(map[string]any{"user_id": "", "role": string(models.RoomRoleMember)})
	req := httptest.NewRequest(http.MethodPost, "/rooms/room-1/members", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), rooms.ErrInvalidMemberUser.Error())
}

func (s *RoomHandlerTestSuite) TestRemoveMemberSuccess() {
	s.service.On("RemoveMember", mock.Anything, "room-1", "user-1").Return(nil).Once()

	req := httptest.NewRequest(http.MethodDelete, "/rooms/room-1/members/user-1", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusNoContent, res.Code)
}

func (s *RoomHandlerTestSuite) TestRemoveMemberDomainError() {
	s.service.On("RemoveMember", mock.Anything, "room-1", "user-1").Return(rooms.ErrInvalidMemberUser).Once()

	req := httptest.NewRequest(http.MethodDelete, "/rooms/room-1/members/user-1", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), rooms.ErrInvalidMemberUser.Error())
}

func (s *RoomHandlerTestSuite) TestListMembersSuccess() {
	members := []models.RoomMember{{ID: "m-1", RoomID: "room-1", UserID: "user-1", Role: models.RoomRoleAdmin}}
	s.service.On("ListMembers", mock.Anything, "room-1").Return(members, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/rooms/room-1/members", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusOK, res.Code)

	var resp struct {
		Members []roomMemberPayload `json:"members"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &resp))
	s.Len(resp.Members, 1)
	s.Equal(members[0].UserID, resp.Members[0].UserID)
}

func (s *RoomHandlerTestSuite) TestListMembersDomainError() {
	s.service.On("ListMembers", mock.Anything, "room-1").Return(([]models.RoomMember)(nil), rooms.ErrInvalidMemberUser).Once()

	req := httptest.NewRequest(http.MethodGet, "/rooms/room-1/members", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), rooms.ErrInvalidMemberUser.Error())
}

func TestRoomHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RoomHandlerTestSuite))
}
