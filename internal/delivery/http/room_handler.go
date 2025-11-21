package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"chatterstack/internal/domain/models"
	"chatterstack/internal/domain/rooms"
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
	group.GET("/", h.search)
	group.POST("/", h.create)
	group.POST("/:roomID/members", h.addMember)
	group.DELETE("/:roomID/members/:userID", h.removeMember)
	group.GET("/:roomID/members", h.listMembers)
}

func (h *RoomHandler) search(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		respondJSONError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 0
	if v := c.Query("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 {
			respondBadRequest(c, "invalid limit parameter")
			return
		}
		limit = parsed
	}

	rooms, err := h.roomUC.Search(c.Request.Context(), userID, c.Query("q"), limit)
	if err != nil {
		status, msg := mapRoomError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}

	c.JSON(http.StatusOK, gin.H{"rooms": toRoomPayloads(rooms)})
}

func (h *RoomHandler) create(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		IsGroup   bool   `json:"is_group"`
		CreatorID string `json:"creator_id"`
		Members   []struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		} `json:"members"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON payload")
		return
	}

	inputs := make([]rooms.RoomMemberInput, 0, len(req.Members))
	for _, m := range req.Members {
		inputs = append(inputs, rooms.RoomMemberInput{
			UserID: m.UserID,
			Role:   models.RoomRole(m.Role),
		})
	}

	room, err := h.roomUC.Create(c.Request.Context(), rooms.CreateRoomInput{
		Name:      req.Name,
		IsGroup:   req.IsGroup,
		CreatorID: req.CreatorID,
		Members:   inputs,
	})
	if err != nil {
		status, msg := mapRoomError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}

	c.JSON(http.StatusCreated, toRoomPayload(room))
}

func (h *RoomHandler) addMember(c *gin.Context) {
	roomID := c.Param("roomID")
	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON payload")
		return
	}
	role := models.RoomRole(req.Role)
	if err := h.roomUC.AddMember(c.Request.Context(), roomID, req.UserID, role); err != nil {
		status, msg := mapRoomError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *RoomHandler) removeMember(c *gin.Context) {
	roomID := c.Param("roomID")
	userID := c.Param("userID")
	if err := h.roomUC.RemoveMember(c.Request.Context(), roomID, userID); err != nil {
		status, msg := mapRoomError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *RoomHandler) listMembers(c *gin.Context) {
	roomID := c.Param("roomID")
	members, err := h.roomUC.ListMembers(c.Request.Context(), roomID)
	if err != nil {
		status, msg := mapRoomError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": toRoomMemberPayloads(members)})
}
