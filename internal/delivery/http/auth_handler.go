package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"chatterstack/internal/usecase"
)

// AuthHandler wires authentication HTTP endpoints.
type AuthHandler struct {
	authUC *usecase.AuthUseCase
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(authUC *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUC: authUC}
}

// RegisterRoutes attaches authentication routes to the provided router group.
func (h *AuthHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	// TODO: replace placeholder with real auth endpoints.
}
