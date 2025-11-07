package http

import (
	"github.com/gin-gonic/gin"

	"chatterstack/internal/usecase"
)

type Services struct {
	Auth    *usecase.AuthUseCase
	User    *usecase.UserUseCase
	Room    *usecase.RoomUseCase
	Message *usecase.MessageUseCase
}

func RegisterRoutes(router *gin.Engine, svc Services) {
	api := router.Group("/v1")

	NewAuthHandler(svc.Auth).RegisterRoutes(api.Group("/auth"))
	NewUserHandler(svc.User).RegisterRoutes(api.Group("/users"))
	NewRoomHandler(svc.Room).RegisterRoutes(api.Group("/rooms"))
	NewMessageHandler(svc.Message).RegisterRoutes(api.Group("/rooms"))
}
