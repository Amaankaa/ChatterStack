package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"chatterstack/internal/domain/models"
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
	group.GET("", h.getByEmail)
	group.GET("/:userID", h.getProfile)
	group.PATCH("/:userID/status", h.updateStatus)
}

func (h *UserHandler) getProfile(c *gin.Context) {
	user, err := h.userUC.GetProfile(c.Request.Context(), c.Param("userID"))
	if err != nil {
		status, msg := mapUserError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}
	c.JSON(http.StatusOK, toUserPayload(user))
}

func (h *UserHandler) getByEmail(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		respondBadRequest(c, "email query parameter is required")
		return
	}
	user, err := h.userUC.GetByEmail(c.Request.Context(), email)
	if err != nil {
		status, msg := mapUserError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}
	c.JSON(http.StatusOK, toUserPayload(user))
}

func (h *UserHandler) updateStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON payload")
		return
	}
	if req.Status == "" {
		respondBadRequest(c, "status is required")
		return
	}

	if err := h.userUC.UpdateStatus(c.Request.Context(), c.Param("userID"), models.UserStatus(req.Status)); err != nil {
		status, msg := mapUserError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}
	c.Status(http.StatusNoContent)
}
