package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"chatterstack/internal/usecase"
)

// UserHandler exposes REST endpoints for user-specific operations.
type UserHandler struct {
	userUC *usecase.UserUseCase
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(userUC *usecase.UserUseCase) *UserHandler {
	return &UserHandler{userUC: userUC}
}

// RegisterRoutes attaches user handlers to the provided router group.
func (h *UserHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/placeholder", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "user routes pending implementation"})
	})
}
