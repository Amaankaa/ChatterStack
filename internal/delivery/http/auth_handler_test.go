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

	"chatterstack/internal/domain/auth"
	"chatterstack/internal/domain/models"
	"chatterstack/internal/usecase"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type stubAuthService struct {
	registerFn func(context.Context, auth.RegisterInput) (*models.User, error)
	loginFn    func(context.Context, string, string) (*auth.TokenPair, error)
	refreshFn  func(context.Context, string) (*auth.TokenPair, error)
	logoutFn   func(context.Context, string) error
	validateFn func(context.Context, string) (string, error)
}

func (s *stubAuthService) Register(ctx context.Context, input auth.RegisterInput) (*models.User, error) {
	if s.registerFn != nil {
		return s.registerFn(ctx, input)
	}
	return nil, nil
}

func (s *stubAuthService) Login(ctx context.Context, email, password string) (*auth.TokenPair, error) {
	if s.loginFn != nil {
		return s.loginFn(ctx, email, password)
	}
	return nil, nil
}

func (s *stubAuthService) Refresh(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	if s.refreshFn != nil {
		return s.refreshFn(ctx, refreshToken)
	}
	return nil, nil
}

func (s *stubAuthService) Logout(ctx context.Context, userID string) error {
	if s.logoutFn != nil {
		return s.logoutFn(ctx, userID)
	}
	return nil
}

func (s *stubAuthService) ValidateAccessToken(ctx context.Context, token string) (string, error) {
	if s.validateFn != nil {
		return s.validateFn(ctx, token)
	}
	return "", nil
}

func newAuthHandlerWithStub(stub *stubAuthService) *AuthHandler {
	uc := usecase.NewAuthUseCase(stub)
	return NewAuthHandler(uc)
}

func TestAuthHandler_RegisterSuccess(t *testing.T) {
	createdAt := time.Date(2024, 10, 5, 12, 0, 0, 0, time.UTC)
	stub := &stubAuthService{
		registerFn: func(ctx context.Context, input auth.RegisterInput) (*models.User, error) {
			if input.Username != "alice" || input.Email != "alice@example.com" || input.Password != "s3cret" {
				t.Fatalf("unexpected register input: %#v", input)
			}
			return &models.User{
				ID:        "user-123",
				Username:  "alice",
				Email:     "alice@example.com",
				Status:    models.UserStatusOffline,
				CreatedAt: createdAt,
			}, nil
		},
	}

	handler := newAuthHandlerWithStub(stub)
	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router.Group("/auth"))

	payload := []byte(`{"username":"alice","email":"alice@example.com","password":"s3cret"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp userPayload
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID != "user-123" {
		t.Errorf("unexpected user id: %s", resp.ID)
	}
	if resp.Username != "alice" {
		t.Errorf("unexpected username: %s", resp.Username)
	}
	if resp.Email != "alice@example.com" {
		t.Errorf("unexpected email: %s", resp.Email)
	}
	if resp.Status != models.UserStatusOffline {
		t.Errorf("unexpected status: %s", resp.Status)
	}
	if resp.LastSeenAt != nil {
		t.Errorf("expected nil last seen, got %v", resp.LastSeenAt)
	}
	if !resp.CreatedAt.Equal(createdAt) {
		t.Errorf("unexpected created at: %v", resp.CreatedAt)
	}
}

func TestAuthHandler_RegisterValidationErrors(t *testing.T) {
	handler := newAuthHandlerWithStub(&stubAuthService{})
	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router.Group("/auth"))

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(`{"username":}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] != "invalid JSON payload" {
		t.Fatalf("unexpected error message: %s", resp["error"])
	}
}

func TestAuthHandler_RegisterDomainConflict(t *testing.T) {
	stub := &stubAuthService{
		registerFn: func(ctx context.Context, input auth.RegisterInput) (*models.User, error) {
			return nil, auth.ErrEmailAlreadyUsed
		},
	}

	handler := newAuthHandlerWithStub(stub)
	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router.Group("/auth"))

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(`{"username":"alice","email":"alice@example.com","password":"s3cret"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] != auth.ErrEmailAlreadyUsed.Error() {
		t.Fatalf("unexpected error message: %s", resp["error"])
	}
}

func TestAuthHandler_LoginSuccess(t *testing.T) {
	stub := &stubAuthService{
		loginFn: func(ctx context.Context, email, password string) (*auth.TokenPair, error) {
			if email != "alice@example.com" || password != "s3cret" {
				t.Fatalf("unexpected login input: %s %s", email, password)
			}
			return &auth.TokenPair{AccessToken: "access", RefreshToken: "refresh"}, nil
		},
	}
	handler := newAuthHandlerWithStub(stub)
	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router.Group("/auth"))

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{"email":"alice@example.com","password":"s3cret"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp tokenPayload
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.AccessToken != "access" || resp.RefreshToken != "refresh" {
		t.Fatalf("unexpected tokens in response: %+v", resp)
	}
}

func TestAuthHandler_LoginUnauthorized(t *testing.T) {
	stub := &stubAuthService{
		loginFn: func(ctx context.Context, email, password string) (*auth.TokenPair, error) {
			return nil, auth.ErrInvalidCredentials
		},
	}
	handler := newAuthHandlerWithStub(stub)
	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router.Group("/auth"))

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{"email":"alice@example.com","password":"bad"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] != auth.ErrInvalidCredentials.Error() {
		t.Fatalf("unexpected error message: %s", resp["error"])
	}
}

func TestAuthHandler_LoginInternalError(t *testing.T) {
	stub := &stubAuthService{
		loginFn: func(ctx context.Context, email, password string) (*auth.TokenPair, error) {
			return nil, errors.New("database offline")
		},
	}
	handler := newAuthHandlerWithStub(stub)
	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router.Group("/auth"))

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{"email":"alice@example.com","password":"secret"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] != "internal server error" {
		t.Fatalf("unexpected error message: %s", resp["error"])
	}
}

func TestAuthHandler_RefreshUnauthorized(t *testing.T) {
	stub := &stubAuthService{
		refreshFn: func(ctx context.Context, token string) (*auth.TokenPair, error) {
			return nil, auth.ErrInvalidRefreshToken
		},
	}
	handler := newAuthHandlerWithStub(stub)
	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router.Group("/auth"))

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader([]byte(`{"refresh_token":"invalid"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] != auth.ErrInvalidRefreshToken.Error() {
		t.Fatalf("unexpected error message: %s", resp["error"])
	}
}

func TestAuthHandler_RefreshSuccess(t *testing.T) {
	stub := &stubAuthService{
		refreshFn: func(ctx context.Context, token string) (*auth.TokenPair, error) {
			if token != "refresh-token" {
				t.Fatalf("unexpected refresh token: %s", token)
			}
			return &auth.TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh"}, nil
		},
	}
	handler := newAuthHandlerWithStub(stub)
	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router.Group("/auth"))

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader([]byte(`{"refresh_token":"refresh-token"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp tokenPayload
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.AccessToken != "new-access" || resp.RefreshToken != "new-refresh" {
		t.Fatalf("unexpected tokens in response: %+v", resp)
	}
}

func TestAuthHandler_LogoutSuccess(t *testing.T) {
	var called bool
	stub := &stubAuthService{
		logoutFn: func(ctx context.Context, userID string) error {
			called = true
			if userID != "user-123" {
				t.Fatalf("unexpected user id: %s", userID)
			}
			return nil
		},
	}
	handler := newAuthHandlerWithStub(stub)
	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router.Group("/auth"))

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader([]byte(`{"user_id":"user-123"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
	if !called {
		t.Fatal("expected logout to be invoked")
	}
}

func TestAuthHandler_LogoutRequiresUserID(t *testing.T) {
	handler := newAuthHandlerWithStub(&stubAuthService{})
	router := gin.New()
	router.Use(gin.Recovery())
	handler.RegisterRoutes(router.Group("/auth"))

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader([]byte(`{"user_id":""}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] != "user_id is required" {
		t.Fatalf("unexpected error message: %s", resp["error"])
	}
}
