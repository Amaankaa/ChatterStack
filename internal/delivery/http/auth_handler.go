package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"chatterstack/internal/domain/auth"
	"chatterstack/internal/domain/models"
	"chatterstack/internal/usecase"
)

type AuthUseCase interface {
	Register(ctx context.Context, input auth.RegisterInput) (*models.User, error)
	Login(ctx context.Context, email, password string) (*auth.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*auth.TokenPair, error)
	Logout(ctx context.Context, userID string) error
}

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
	group.POST("/register", h.register)
	group.POST("/login", h.login)
	group.POST("/refresh", h.refresh)
	group.POST("/logout", h.logout)
}

func (h *AuthHandler) register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON payload")
		return
	}

	user, err := h.authUC.Register(c.Request.Context(), auth.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		status, msg := mapAuthError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}

	c.JSON(http.StatusCreated, toUserPayload(user))
}

func (h *AuthHandler) login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON payload")
		return
	}

	tokens, err := h.authUC.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		status, msg := mapAuthError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}

	c.JSON(http.StatusOK, tokenPayload{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *AuthHandler) refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid JSON payload")
		return
	}

	tokens, err := h.authUC.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		status, msg := mapAuthError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}

		respondJSONError(c, status, msg)
		return
	}

	c.JSON(http.StatusOK, tokenPayload{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *AuthHandler) logout(c *gin.Context) {
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

	if err := h.authUC.Logout(c.Request.Context(), req.UserID); err != nil {
		status, msg := mapAuthError(err)
		if status == http.StatusInternalServerError {
			respondInternalServerError(c, err)
			return
		}
		respondJSONError(c, status, msg)
		return
	}

	c.Status(http.StatusNoContent)
}
