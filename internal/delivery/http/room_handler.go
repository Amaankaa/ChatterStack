package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"chatterstack/internal/usecase"
)

// RoomHandler wires room management REST endpoints.
type RoomHandler struct {
	roomUC *usecase.RoomUseCase
}

// NewRoomHandler constructs a RoomHandler.
func NewRoomHandler(roomUC *usecase.RoomUseCase) *RoomHandler {
	return &RoomHandler{roomUC: roomUC}
}

// RegisterRoutes attaches room handlers to the provided router group.
func (h *RoomHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/placeholder", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "room routes pending implementation"})
	})
}
