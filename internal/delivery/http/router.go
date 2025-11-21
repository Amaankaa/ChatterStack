package http

import (
	"github.com/gin-gonic/gin"

	"chatterstack/internal/usecase"
	"chatterstack/pkg/middleware"
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

	protected := api.Group("")
	protected.Use(middleware.Auth(svc.Auth))

	NewUserHandler(svc.User).RegisterRoutes(protected.Group("/users"))
	NewRoomHandler(svc.Room).RegisterRoutes(protected.Group("/rooms"))
	NewMessageHandler(svc.Message).RegisterRoutes(protected.Group("/rooms"))
}
