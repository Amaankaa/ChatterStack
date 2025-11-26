package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
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
	RateLimit RateLimitConfig
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

// RateLimitConfig stores API rate limiting parameters.
type RateLimitConfig struct {
	Requests int
	Burst    int
	Window   time.Duration
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
		Postgres: PostgresConfig{},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "change-me"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "change-me-too"),
		},
	}

	accessTTL, err := parseDurationEnv("JWT_ACCESS_TTL", "1h")
	if err != nil {
		return Config{}, err
	}
	cfg.JWT.AccessTTL = accessTTL

	refreshTTL, err := parseDurationEnv("JWT_REFRESH_TTL", "168h")
	if err != nil {
		return Config{}, err
	}
	cfg.JWT.RefreshTTL = refreshTTL

	requests, err := parseIntEnv("RATE_LIMIT_REQUESTS", 100)
	if err != nil {
		return Config{}, err
	}
	burst, err := parseIntEnv("RATE_LIMIT_BURST", requests)
	if err != nil {
		return Config{}, err
	}
	window, err := parseDurationEnv("RATE_LIMIT_WINDOW", "1m")
	if err != nil {
		return Config{}, err
	}
	cfg.RateLimit = RateLimitConfig{Requests: requests, Burst: burst, Window: window}

	// Prefer an explicit POSTGRES_DSN. If not provided, compose it from parts.
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		cfg.Postgres.DSN = dsn
	} else {
		user := getEnv("POSTGRES_USER", "chatterstack")
		pass := os.Getenv("POSTGRES_PASSWORD")
		if pass == "" {
			pass = getEnv("POSTGRES_PASSWORD", "chatterstack")
		}
		host := getEnv("POSTGRES_HOST", "localhost")
		db := getEnv("POSTGRES_DB", "chatterstack")
		eu := url.QueryEscape(user)
		ep := url.QueryEscape(pass)
		cfg.Postgres.DSN = fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", eu, ep, host, db)
	}

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

func parseIntEnv(key string, fallback int) (int, error) {
	value := getEnv(key, fmt.Sprintf("%d", fallback))
	i, err := strconv.Atoi(value)
	if err != nil || i < 0 {
		return 0, fmt.Errorf("invalid integer for %s: %v", key, err)
	}
	return i, nil
}
