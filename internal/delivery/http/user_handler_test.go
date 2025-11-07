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
	"github.com/stretchr/testify/suite"

	"chatterstack/internal/domain/models"
	"chatterstack/internal/domain/users"
	"chatterstack/internal/usecase"
)

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	user, _ := args.Get(0).(*models.User)
	return user, args.Error(1)
}

func (m *mockUserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	user, _ := args.Get(0).(*models.User)
	return user, args.Error(1)
}

func (m *mockUserService) UpdateStatus(ctx context.Context, id string, status models.UserStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

type UserHandlerTestSuite struct {
	suite.Suite

	service *mockUserService
	router  *gin.Engine
}

func (s *UserHandlerTestSuite) SetupTest() {
	s.service = new(mockUserService)
	h := NewUserHandler(usecase.NewUserUseCase(s.service))

	s.router = gin.New()
	routes := s.router.Group("/users")
	h.RegisterRoutes(routes)
}

func (s *UserHandlerTestSuite) TearDownTest() {
	s.service.AssertExpectations(s.T())
}

func (s *UserHandlerTestSuite) TestGetProfileSuccess() {
	user := &models.User{ID: "user-1", Username: "alice", Email: "alice@example.com", Status: models.UserStatusOnline, CreatedAt: time.Unix(1, 0)}
	s.service.On("GetByID", mock.Anything, "user-1").Return(user, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/users/user-1", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusOK, res.Code)

	var resp userPayload
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &resp))
	s.Equal(user.ID, resp.ID)
	s.Equal(user.Email, resp.Email)
	s.Equal(user.Status, resp.Status)
}

func (s *UserHandlerTestSuite) TestGetProfileNotFound() {
	s.service.On("GetByID", mock.Anything, "user-1").Return((*models.User)(nil), users.ErrUserNotFound).Once()

	req := httptest.NewRequest(http.MethodGet, "/users/user-1", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusNotFound, res.Code)
	s.Contains(res.Body.String(), users.ErrUserNotFound.Error())
}

func (s *UserHandlerTestSuite) TestGetByEmailSuccess() {
	user := &models.User{ID: "user-1", Username: "alice", Email: "alice@example.com", Status: models.UserStatusOffline, CreatedAt: time.Unix(2, 0)}
	s.service.On("GetByEmail", mock.Anything, "alice@example.com").Return(user, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/users/?email=alice@example.com", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusOK, res.Code)

	var resp userPayload
	s.Require().NoError(json.Unmarshal(res.Body.Bytes(), &resp))
	s.Equal(user.ID, resp.ID)
}

func (s *UserHandlerTestSuite) TestGetByEmailRequiresQuery() {
	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
}

func (s *UserHandlerTestSuite) TestGetByEmailNotFound() {
	s.service.On("GetByEmail", mock.Anything, "alice@example.com").Return((*models.User)(nil), users.ErrUserNotFound).Once()

	req := httptest.NewRequest(http.MethodGet, "/users/?email=alice@example.com", nil)
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusNotFound, res.Code)
	s.Contains(res.Body.String(), users.ErrUserNotFound.Error())
}

func (s *UserHandlerTestSuite) TestUpdateStatusSuccess() {
	s.service.On("UpdateStatus", mock.Anything, "user-1", models.UserStatusOnline).Return(nil).Once()

	payload, _ := json.Marshal(map[string]any{"status": string(models.UserStatusOnline)})
	req := httptest.NewRequest(http.MethodPatch, "/users/user-1/status", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusNoContent, res.Code)
}

func (s *UserHandlerTestSuite) TestUpdateStatusInvalidJSON() {
	req := httptest.NewRequest(http.MethodPatch, "/users/user-1/status", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
}

func (s *UserHandlerTestSuite) TestUpdateStatusMissingField() {
	payload, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPatch, "/users/user-1/status", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.Body.String(), "status is required")
}

func (s *UserHandlerTestSuite) TestUpdateStatusDomainError() {
	s.service.On("UpdateStatus", mock.Anything, "user-1", models.UserStatusOffline).Return(users.ErrUserNotFound).Once()

	payload, _ := json.Marshal(map[string]any{"status": string(models.UserStatusOffline)})
	req := httptest.NewRequest(http.MethodPatch, "/users/user-1/status", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	s.router.ServeHTTP(res, req)

	s.Equal(http.StatusNotFound, res.Code)
	s.Contains(res.Body.String(), users.ErrUserNotFound.Error())
}

func TestUserHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UserHandlerTestSuite))
}
