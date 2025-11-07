package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"chatterstack/internal/usecase"
)

// MessageHandler wires REST endpoints for message retrieval and creation.
type MessageHandler struct {
	messageUC *usecase.MessageUseCase
}

// NewMessageHandler constructs a MessageHandler.
func NewMessageHandler(messageUC *usecase.MessageUseCase) *MessageHandler {
	return &MessageHandler{messageUC: messageUC}
}

// RegisterRoutes attaches message handlers to the provided router group.
func (h *MessageHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/:roomID/messages/placeholder", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "message routes pending implementation"})
	})
}
