package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRateLimitAllowsWithinQuota(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimit(RateLimiterConfig{Requests: 10, Window: time.Second, Burst: 10, KeyFunc: func(*gin.Context) string { return "fixed" }}))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for i := 0; i < 5; i++ {
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, http.StatusOK, res.Code)
	}
}

func TestRateLimitRejectsWhenQuotaExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimit(RateLimiterConfig{Requests: 1, Window: time.Minute, Burst: 1, KeyFunc: func(*gin.Context) string { return "fixed" }}))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	first := httptest.NewRecorder()
	router.ServeHTTP(first, req)
	require.Equal(t, http.StatusOK, first.Code)

	second := httptest.NewRecorder()
	router.ServeHTTP(second, req)
	require.Equal(t, http.StatusTooManyRequests, second.Code)
}
