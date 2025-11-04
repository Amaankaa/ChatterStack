package config

import (
	"fmt"
	"os"
	"time"
)

// Config aggregates all runtime configuration sections.
type Config struct {
	AppEnv    string
	HTTP      HTTPConfig
	Websocket WebsocketConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
	JWT       JWTConfig
}

// HTTPConfig holds REST server options.
type HTTPConfig struct {
	Host string
	Port string
}

// Address builds the host:port pair used when starting the HTTP server.
func (c HTTPConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// WebsocketConfig holds WebSocket server options.
type WebsocketConfig struct {
	Host string
	Port string
}

// Address builds the host:port pair used when starting the WebSocket server.
func (c WebsocketConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// PostgresConfig stores PostgreSQL connectivity settings.
type PostgresConfig struct {
	DSN string
}

// RedisConfig stores Redis connectivity settings.
type RedisConfig struct {
	Addr     string
	Password string
}

// JWTConfig stores secrets and TTLs for access and refresh tokens.
type JWTConfig struct {
	AccessSecret  string
	AccessTTL     time.Duration
	RefreshSecret string
	RefreshTTL    time.Duration
}

// Load reads configuration from environment variables, applying sensible defaults.
func Load() (Config, error) {
	cfg := Config{
		AppEnv: getEnv("APP_ENV", "development"),
		HTTP: HTTPConfig{
			Host: getEnv("HTTP_HOST", "0.0.0.0"),
			Port: getEnv("HTTP_PORT", "8080"),
		},
		Websocket: WebsocketConfig{
			Host: getEnv("WEBSOCKET_HOST", "0.0.0.0"),
			Port: getEnv("WEBSOCKET_PORT", "8081"),
		},
		Postgres: PostgresConfig{
			DSN: getEnv("POSTGRES_DSN", "postgres://chatterstack:chatterstack@localhost:5432/chatterstack?sslmode=disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "change-me"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "change-me-too"),
		},
	}

	accessTTL, err := parseDurationEnv("JWT_ACCESS_TTL", "15m")
	if err != nil {
		return Config{}, err
	}
	cfg.JWT.AccessTTL = accessTTL

	refreshTTL, err := parseDurationEnv("JWT_REFRESH_TTL", "168h")
	if err != nil {
		return Config{}, err
	}
	cfg.JWT.RefreshTTL = refreshTTL

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func parseDurationEnv(key, fallback string) (time.Duration, error) {
	value := getEnv(key, fallback)
	dur, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return dur, nil
}
