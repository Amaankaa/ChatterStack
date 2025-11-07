package http

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"chatterstack/internal/domain/auth"
	"chatterstack/internal/domain/models"
)

type userPayload struct {
	ID         string            `json:"id"`
	Username   string            `json:"username"`
	Email      string            `json:"email"`
	Status     models.UserStatus `json:"status"`
	LastSeenAt *time.Time        `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

type tokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func toUserPayload(u *models.User) userPayload {
	var lastSeen *time.Time
	if !u.LastSeenAt.IsZero() {
		ts := u.LastSeenAt
		lastSeen = &ts
	}

	return userPayload{
		ID:         u.ID,
		Username:   u.Username,
		Email:      u.Email,
		Status:     u.Status,
		LastSeenAt: lastSeen,
		CreatedAt:  u.CreatedAt,
	}
}

func respondBadRequest(c *gin.Context, message string) {
	respondJSONError(c, http.StatusBadRequest, message)
}

func respondJSONError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func respondInternalServerError(c *gin.Context, err error) {
	log.Printf("http: internal error: %v", err)
	respondJSONError(c, http.StatusInternalServerError, "internal server error")
}

func mapAuthError(err error) (int, string) {
	switch {
	case errors.Is(err, auth.ErrMissingUsername),
		errors.Is(err, auth.ErrMissingEmail),
		errors.Is(err, auth.ErrMissingPassword):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, auth.ErrEmailAlreadyUsed):
		return http.StatusConflict, err.Error()
	case errors.Is(err, auth.ErrInvalidCredentials),
		errors.Is(err, auth.ErrInvalidRefreshToken):
		return http.StatusUnauthorized, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
