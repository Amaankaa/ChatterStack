package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chatterstack/internal/domain/messages"
	"chatterstack/internal/domain/models"
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
	group.GET("/:roomID/messages", h.listByRoom)
	group.POST("/:roomID/messages", h.send)
	group.POST("/:roomID/messages/:messageID/deliver", h.markDelivered)
	group.POST("/:roomID/messages/:messageID/read", h.markRead)
}

func (h *MessageHandler) send(c *gin.Context) {
	roomID := c.Param("roomID")
	var req struct {
		SenderID    string `json:"sender_id"`
		Content     string `json:"content"`
		Attachments []struct {
			URL       string `json:"url"`
			MimeType  string `json:"mime_type"`
			SizeBytes int64  `json:"size_bytes"`
		} `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON payload")
		return
	}

	attachments := make([]models.Attachment, 0, len(req.Attachments))
	for _, att := range req.Attachments {
		attachments = append(attachments, models.Attachment{
			URL:       att.URL,
			MimeType:  att.MimeType,
			SizeBytes: att.SizeBytes,
		})
	}

	msg, err := h.messageUC.Send(c.Request.Context(), messages.SendMessageInput{
		RoomID:      roomID,
		SenderID:    req.SenderID,
		Content:     req.Content,
		Attachments: attachments,
	})
	if err != nil {
		status, msgErr := mapMessageError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msgErr)
		return
	}

	c.JSON(http.StatusCreated, toMessagePayload(*msg))
}

func (h *MessageHandler) listByRoom(c *gin.Context) {
	roomID := c.Param("roomID")
	page, limit, ok := parsePagination(c)
	if !ok {
		respondBadRequest(c, "invalid pagination parameters")
		return
	}

	msgs, err := h.messageUC.ListByRoom(c.Request.Context(), roomID, page, limit)
	if err != nil {
		status, msgErr := mapMessageError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msgErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": toMessagePayloads(msgs)})
}

func (h *MessageHandler) markDelivered(c *gin.Context) {
	messageID := c.Param("messageID")
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON payload")
		return
	}
	if req.UserID == "" {
		respondBadRequest(c, "user_id is required")
		return
	}

	if err := h.messageUC.MarkDelivered(c.Request.Context(), messageID, req.UserID); err != nil {
		status, msgErr := mapMessageError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msgErr)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *MessageHandler) markRead(c *gin.Context) {
	messageID := c.Param("messageID")
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON payload")
		return
	}
	if req.UserID == "" {
		respondBadRequest(c, "user_id is required")
		return
	}

	if err := h.messageUC.MarkRead(c.Request.Context(), messageID, req.UserID); err != nil {
		status, msgErr := mapMessageError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msgErr)
		return
	}

	c.Status(http.StatusNoContent)
}

func parsePagination(c *gin.Context) (int, int, bool) {
	page := 1
	limit := 0
	if v := c.Query("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p <= 0 {
			return 0, 0, false
		}
		page = p
	}
	if v := c.Query("limit"); v != "" {
		l, err := strconv.Atoi(v)
		if err != nil || l <= 0 {
			return 0, 0, false
		}
		limit = l
	}
	return page, limit, true
}
