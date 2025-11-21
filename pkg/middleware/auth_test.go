package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubTokenValidator struct {
	userID string
	err    error
}

func (s *stubTokenValidator) ValidateAccessToken(_ context.Context, token string) (string, error) {
	if token == "valid-token" {
		return s.userID, s.err
	}
	return "", errors.New("invalid token")
}

func TestAuthMiddlewareSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	validator := &stubTokenValidator{userID: "user-123"}
	router.Use(Auth(validator))
	router.GET("/protected", func(c *gin.Context) {
		uid, _ := c.Get(UserIDContextKey)
		c.String(http.StatusOK, uid.(string))
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, "user-123", res.Body.String())
}

func TestAuthMiddlewareMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Auth(&stubTokenValidator{}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Auth(&stubTokenValidator{}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusUnauthorized, res.Code)
}
