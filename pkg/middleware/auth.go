package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenValidator describes the dependency needed to verify access tokens.
type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, token string) (string, error)
}

// UserIDContextKey is the key used to store the authenticated user ID inside gin.Context.
const UserIDContextKey = "userID"

// Auth returns a Gin middleware that validates bearer tokens and stores the user id in the context.
func Auth(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if validator == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth middleware not configured"})
			return
		}

		header := c.GetHeader("Authorization")
		token := extractBearerToken(header)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		userID, err := validator.ValidateAccessToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
			return
		}

		c.Set(UserIDContextKey, userID)
		c.Next()
	}
}

func extractBearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
