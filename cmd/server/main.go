package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sync/atomic"

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
	"chatterstack/pkg/middleware"

	wsdelivery "chatterstack/internal/delivery/websocket"

	ws "github.com/gorilla/websocket"
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
		if err := startWebsocketServer(ctx, cfg); err != nil {
			log.Fatalf("websocket server error: %v", err)
		}
	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}

func startAPIServer(ctx context.Context, cfg config.Config) error {
	// Log DSN with password masked and sslmode for diagnostics
	masked, ssl := maskDSN(cfg.Postgres.DSN)
	log.Printf("postgres dsn: %s (sslmode=%s)", masked, ssl)
	pgxCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		return fmt.Errorf("parse postgres dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	// Start background ping for Postgres so the container doesn't exit on
	// transient startup failures. We log attempts and continue running the
	// HTTP server so we can inspect logs and let the orchestrator manage
	// restarts if necessary.
	var depsReady int32
	go func() {
		backoff := time.Second
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := pool.Ping(ctx); err != nil {
				log.Printf("postgres ping failed: %v; retrying in %s", err, backoff)
			} else {
				log.Printf("postgres ping succeeded")
				break
			}
			time.Sleep(backoff)
			if backoff < 10*time.Second {
				backoff *= 2
			}
		}
		atomic.StoreInt32(&depsReady, 1)
	}()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
	defer redisClient.Close()

	// Ping Redis in background as well to avoid exiting the process on
	// transient connectivity/auth failures during startup.
	go func() {
		backoff := time.Second
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := redisClient.Ping(ctx).Err(); err != nil {
				log.Printf("redis ping failed: %v; retrying in %s", err, backoff)
			} else {
				log.Printf("redis ping succeeded")
				return
			}
			time.Sleep(backoff)
			if backoff < 10*time.Second {
				backoff *= 2
			}
		}
	}()

	userRepo := postgresrepo.NewUserRepository(pool)
	roomRepo := postgresrepo.NewRoomRepository(pool)
	messageRepo := postgresrepo.NewMessageRepository(pool)

	cache := redisrepo.NewCache(redisClient)
	pubsub := redisrepo.NewPubSub(redisClient)

	authService := auth.NewService(userRepo, cache, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	userService := users.NewService(userRepo)
	roomService := rooms.NewService(roomRepo, pubsub)
	messageService := messages.NewService(messageRepo, pubsub)

	authUC := usecase.NewAuthUseCase(authService)
	userUC := usecase.NewUserUseCase(userService)
	roomUC := usecase.NewRoomUseCase(roomService)
	messageUC := usecase.NewMessageUseCase(messageService)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORS())
	router.Use(middleware.RateLimit(middleware.RateLimiterConfig{
		Requests: cfg.RateLimit.Requests,
		Burst:    cfg.RateLimit.Burst,
		Window:   cfg.RateLimit.Window,
	}))

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

func startWebsocketServer(ctx context.Context, cfg config.Config) error {
	// Log DSN with password masked and sslmode for diagnostics
	masked, ssl := maskDSN(cfg.Postgres.DSN)
	log.Printf("postgres dsn: %s (sslmode=%s)", masked, ssl)
	pgxCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		return fmt.Errorf("parse postgres dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	// Background ping for Postgres (non-fatal) so the websocket process
	// doesn't exit immediately on transient failures.
	go func() {
		backoff := time.Second
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := pool.Ping(ctx); err != nil {
				log.Printf("postgres ping failed: %v; retrying in %s", err, backoff)
			} else {
				log.Printf("postgres ping succeeded")
				return
			}
			time.Sleep(backoff)
			if backoff < 10*time.Second {
				backoff *= 2
			}
		}
	}()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
	defer redisClient.Close()
	go func() {
		backoff := time.Second
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := redisClient.Ping(ctx).Err(); err != nil {
				log.Printf("redis ping failed: %v; retrying in %s", err, backoff)
			} else {
				log.Printf("redis ping succeeded")
				return
			}
			time.Sleep(backoff)
			if backoff < 10*time.Second {
				backoff *= 2
			}
		}
	}()

	userRepo := postgresrepo.NewUserRepository(pool)
	roomRepo := postgresrepo.NewRoomRepository(pool)
	messageRepo := postgresrepo.NewMessageRepository(pool)

	cache := redisrepo.NewCache(redisClient)
	pubsub := redisrepo.NewPubSub(redisClient)

	authService := auth.NewService(userRepo, cache, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	userService := users.NewService(userRepo)
	roomService := rooms.NewService(roomRepo, pubsub)
	messageService := messages.NewService(messageRepo, pubsub)

	authUC := usecase.NewAuthUseCase(authService)
	userUC := usecase.NewUserUseCase(userService)
	roomUC := usecase.NewRoomUseCase(roomService)
	messageUC := usecase.NewMessageUseCase(messageService)

	hub := wsdelivery.NewHub()
	hubCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go hub.Run(hubCtx)

	wsdelivery.StartMessageRelay(hubCtx, hub, pubsub)

	upgrader := ws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(*http.Request) bool { return true },
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := wsdelivery.ExtractBearerToken(r.Header.Get("Authorization"))
		if token == "" {
			// Allow browser clients to pass JWT via query param when headers aren't available
			token = strings.TrimSpace(r.URL.Query().Get("access_token"))
		}
		if token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		userID, err := authUC.ValidateAccessToken(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid access token", http.StatusUnauthorized)
			return
		}

		rawRooms := r.URL.Query()["room_id"]
		if len(rawRooms) == 0 {
			http.Error(w, "room_id is required", http.StatusBadRequest)
			return
		}

		allowed := wsdelivery.FilterAuthorizedRooms(r.Context(), roomUC, userID, rawRooms)
		if len(allowed) == 0 {
			http.Error(w, "no authorized rooms", http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade failed: %v", err)
			return
		}

		profile, err := userUC.GetProfile(r.Context(), userID)
		if err != nil {
			log.Printf("websocket: fetch user profile failed: %v", err)
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		client := wsdelivery.NewClient(conn, hub, userID, allowed, messageUC, profile.Username)
		hub.Register(client)

		go client.WritePump()
		client.ReadPump(hubCtx)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:         cfg.Websocket.Address(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errChan := make(chan error, 1)

	go func() {
		log.Printf("Websocket server listening on %s", cfg.Websocket.Address())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("websocket server shutdown: %w", err)
		}
		return nil
	case err := <-errChan:
		return err
	}
}

// maskDSN masks the password in a postgres DSN and returns sslmode.
func maskDSN(dsn string) (string, string) {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn, ""
	}
	// Extract sslmode from query
	q := u.Query()
	ssl := q.Get("sslmode")
	// Mask password
	if u.User != nil {
		username := u.User.Username()
		if _, has := u.User.Password(); has {
			u.User = url.UserPassword(username, "****")
		} else {
			u.User = url.User(username)
		}
	}
	return u.String(), ssl
}
