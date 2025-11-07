package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"chatterstack/internal/config"
	httpdelivery "chatterstack/internal/delivery/http"
	"chatterstack/internal/domain/auth"
	"chatterstack/internal/domain/messages"
	"chatterstack/internal/domain/rooms"
	"chatterstack/internal/domain/users"
	postgresrepo "chatterstack/internal/repository/postgres"
	redisrepo "chatterstack/internal/repository/redis"
	"chatterstack/internal/usecase"
)

func main() {
	mode := flag.String("mode", "api", "service mode: api or websocket")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch *mode {
	case "api":
		if err := startAPIServer(ctx, cfg); err != nil {
			log.Fatalf("api server error: %v", err)
		}
	case "websocket":
		log.Println("WebSocket server bootstrap incomplete; implement hub startup in internal/delivery/websocket")
	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}

func startAPIServer(ctx context.Context, cfg config.Config) error {
	pgxCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		return fmt.Errorf("parse postgres dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}

	userRepo := postgresrepo.NewUserRepository(pool)
	roomRepo := postgresrepo.NewRoomRepository(pool)
	messageRepo := postgresrepo.NewMessageRepository(pool)

	cache := redisrepo.NewCache(redisClient)
	pubsub := redisrepo.NewPubSub(redisClient)

	authService := auth.NewService(userRepo, cache, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	userService := users.NewService(userRepo)
	roomService := rooms.NewService(roomRepo)
	messageService := messages.NewService(messageRepo, pubsub)

	authUC := usecase.NewAuthUseCase(authService)
	userUC := usecase.NewUserUseCase(userService)
	roomUC := usecase.NewRoomUseCase(roomService)
	messageUC := usecase.NewMessageUseCase(messageService)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	httpdelivery.RegisterRoutes(router, httpdelivery.Services{
		Auth:    authUC,
		User:    userUC,
		Room:    roomUC,
		Message: messageUC,
	})

	server := &http.Server{
		Addr:         cfg.HTTP.Address(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errChan := make(chan error, 1)

	go func() {
		log.Printf("HTTP server listening on %s", cfg.HTTP.Address())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}
		return nil
	case err := <-errChan:
		return err
	}
}
