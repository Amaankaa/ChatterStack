package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"chatterstack/internal/domain/messages"
	"chatterstack/internal/domain/models"
	"chatterstack/internal/usecase"
	"chatterstack/pkg/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockMessageService struct {
	mock.Mock
}

func (m *mockMessageService) Send(ctx context.Context, input messages.SendMessageInput) (*models.Message, error) {
	args := m.Called(ctx, input)
	msg, _ := args.Get(0).(*models.Message)
	return msg, args.Error(1)
}

func (m *mockMessageService) ListByRoom(ctx context.Context, roomID string, page, limit int) ([]models.Message, error) {
	args := m.Called(ctx, roomID, page, limit)
	if res := args.Get(0); res != nil {
		return res.([]models.Message), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockMessageService) MarkDelivered(ctx context.Context, messageID, userID string) error {
	args := m.Called(ctx, messageID, userID)
	return args.Error(0)
}

func (m *mockMessageService) MarkRead(ctx context.Context, messageID, userID string) error {
	args := m.Called(ctx, messageID, userID)
	return args.Error(0)
}

type MessageHandlerTestSuite struct {
	suite.Suite

	service *mockMessageService
	handler *MessageHandler
	router  *gin.Engine
}

func (s *MessageHandlerTestSuite) SetupTest() {
	s.service = new(mockMessageService)
	uc := usecase.NewMessageUseCase(s.service)
	s.handler = NewMessageHandler(uc)

	s.router = gin.New()
	s.router.Use(func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, "user-1")
		c.Next()
	})
	routes := s.router.Group("/rooms")
	s.handler.RegisterRoutes(routes)
}

func (s *MessageHandlerTestSuite) TearDownTest() {
	s.service.AssertExpectations(s.T())
}

func (s *MessageHandlerTestSuite) TestSendSuccess() {
	createdAt := time.Date(2025, 11, 8, 10, 0, 0, 0, time.UTC)
	expected := messages.SendMessageInput{
		RoomID:   "room-1",
		SenderID: "user-1",
		Content:  "hello",
		Attachments: []models.Attachment{
			{URL: "https://example.com/img.png", MimeType: "image/png", SizeBytes: 42},
		},
	}
	result := &models.Message{
		ID:          "msg-1",
		RoomID:      expected.RoomID,
		SenderID:    expected.SenderID,
		Content:     expected.Content,
		Attachments: expected.Attachments,
		Status:      models.MessageStatusSent,
		CreatedAt:   createdAt,
	}

	s.service.On("Send", mock.Anything, expected).Return(result, nil).Once()

	body := map[string]any{
		"content": expected.Content,
		"attachments": []map[string]any{
			{
				"url":        expected.Attachments[0].URL,
				"mime_type":  expected.Attachments[0].MimeType,
				"size_bytes": expected.Attachments[0].SizeBytes,
			},
		},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/rooms/room-1/messages", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Require().Equal(http.StatusCreated, res.Code)

	var resp messagePayload
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &resp))
	s.Equal(result.ID, resp.ID)
	s.Equal(result.RoomID, resp.RoomID)
	s.Equal(result.SenderID, resp.SenderID)
	s.Equal(result.Content, resp.Content)
	s.Equal(result.Status, resp.Status)
	s.Equal(result.CreatedAt, resp.CreatedAt)
	s.Len(resp.Attachments, 1)
	s.Equal(result.Attachments[0].URL, resp.Attachments[0].URL)
}

func (s *MessageHandlerTestSuite) TestSendDomainError() {
	matcher := mock.MatchedBy(func(input messages.SendMessageInput) bool {
		return input.RoomID == "room-1" && input.SenderID == "user-1" && input.Content == "" && len(input.Attachments) == 0
	})
	s.service.On("Send", mock.Anything, matcher).Return((*models.Message)(nil), messages.ErrInvalidContent).Once()

	payload, _ := json.Marshal(map[string]any{"content": ""})

	req := httptest.NewRequest(http.MethodPost, "/rooms/room-1/messages", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), messages.ErrInvalidContent.Error())
}

func (s *MessageHandlerTestSuite) TestSendInvalidJSON() {
	req := httptest.NewRequest(http.MethodPost, "/rooms/room-1/messages", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), "invalid JSON payload")
}

func (s *MessageHandlerTestSuite) TestListByRoomSuccess() {
	s.service.On("ListByRoom", mock.Anything, "room-1", 2, 10).Return([]models.Message{
		{ID: "msg-1", RoomID: "room-1", SenderID: "user-1", Content: "hello", Status: models.MessageStatusSent, CreatedAt: time.Unix(1, 0)},
	}, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/rooms/room-1/messages?page=2&limit=10", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusOK, res.Code)

	var resp struct {
		Messages []messagePayload `json:"messages"`
	}
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &resp))
	s.Len(resp.Messages, 1)
	s.Equal("msg-1", resp.Messages[0].ID)
}

func (s *MessageHandlerTestSuite) TestListByRoomInvalidPagination() {
	req := httptest.NewRequest(http.MethodGet, "/rooms/room-1/messages?limit=abc", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
}

func (s *MessageHandlerTestSuite) TestListByRoomDomainError() {
	s.service.On("ListByRoom", mock.Anything, "room-1", 1, 0).Return(nil, messages.ErrInvalidRoomID).Once()

	req := httptest.NewRequest(http.MethodGet, "/rooms/room-1/messages", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), messages.ErrInvalidRoomID.Error())
}

func (s *MessageHandlerTestSuite) TestMarkDeliveredSuccess() {
	s.service.On("MarkDelivered", mock.Anything, "msg-1", "user-1").Return(nil).Once()

	req := httptest.NewRequest(http.MethodPost, "/rooms/room-1/messages/msg-1/deliver", nil)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusNoContent, res.Code)
}

func (s *MessageHandlerTestSuite) TestMarkDeliveredDomainError() {
	s.service.On("MarkDelivered", mock.Anything, "msg-1", "user-1").Return(messages.ErrInvalidMessageID).Once()

	req := httptest.NewRequest(http.MethodPost, "/rooms/room-1/messages/msg-1/deliver", nil)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), messages.ErrInvalidMessageID.Error())
}

func (s *MessageHandlerTestSuite) TestMarkReadSuccess() {
	s.service.On("MarkRead", mock.Anything, "msg-1", "user-1").Return(nil).Once()

	req := httptest.NewRequest(http.MethodPost, "/rooms/room-1/messages/msg-1/read", nil)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusNoContent, res.Code)
}

func (s *MessageHandlerTestSuite) TestMarkReadDomainError() {
	s.service.On("MarkRead", mock.Anything, "msg-1", "user-1").Return(messages.ErrInvalidUserID).Once()

	req := httptest.NewRequest(http.MethodPost, "/rooms/room-1/messages/msg-1/read", nil)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), messages.ErrInvalidUserID.Error())
}

func TestMessageHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(MessageHandlerTestSuite))
}

func TestMessageHandlerRequiresAuthentication(t *testing.T) {
	service := new(mockMessageService)
	uc := usecase.NewMessageUseCase(service)
	h := NewMessageHandler(uc)

	router := gin.New()
	routes := router.Group("/rooms")
	h.RegisterRoutes(routes)

	req := httptest.NewRequest(http.MethodPost, "/rooms/room-1/messages", bytes.NewBufferString(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusUnauthorized, res.Code)
	service.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
}
