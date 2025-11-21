# ChatterStack

ChatterStack is a real-time chat backend written in Go. It exposes a REST API for auth, user, room, and message management, and a WebSocket gateway for low-latency messaging. The project embraces a clean architecture layout, separating delivery layers, use cases, and domain services from infrastructure concerns such as Postgres persistence and Redis caching/pub-sub.

## Features

- JWT-based authentication with access/refresh token lifecycle.
- Modular HTTP delivery using Gin and a WebSocket hub for real-time fan-out.
- PostgreSQL repositories for users, rooms, and messages (with migrations scaffolded under `db/migrations`).
- Redis cache for session tracking and Redis pub/sub for cross-node WebSocket broadcasts.
- Configurable request rate limiting and auth middleware.
- Comprehensive unit tests across delivery, domain, repository, and middleware layers.
- GitHub Actions CI pipeline enforcing `gofmt` and `go test ./...`.
- Generated `API_documentation.md` for all endpoints.

## Project Structure

```
cmd/server/main.go        # Application entrypoint supporting API or WebSocket mode
internal/config           # Environment-driven configuration loading
internal/delivery         # HTTP & WebSocket handlers
internal/domain           # Domain entities and services (auth, rooms, messages, users)
internal/repository       # Postgres & Redis adapters
internal/usecase          # Orchestrates domain services for delivery layers
pkg/middleware            # HTTP middleware (auth, logging, rate limiting)
api/                      # Swagger + Postman artifacts
db/migrations             # SQL migrations scaffolding
```

## Prerequisites

- Go >= 1.21
- Docker & Docker Compose (optional but recommended)
- PostgreSQL 14+
- Redis 7+

## Getting Started

### 1. Configure Environment

```
cp .env.example .env
```

Update values such as `POSTGRES_DSN`, `REDIS_ADDR`, and JWT secrets. Rate limiting can be tuned via:

- `RATE_LIMIT_REQUESTS` (default: 100 requests)
- `RATE_LIMIT_BURST` (default: same as requests)
- `RATE_LIMIT_WINDOW` (default: 1m)

### 2. Launch Dependencies

```
docker-compose up -d
```

This brings up Postgres and Redis using the credentials matched in `.env`.

### 3. Apply Migrations (Optional)

Use your preferred migration tool (e.g., `migrate` or `golang-migrate`) with the SQL files in `db/migrations`.

### 4. Run the API Server

```
go run ./cmd/server -mode api
```

The HTTP service listens on `HTTP_HOST`:`HTTP_PORT` (default `0.0.0.0:8080`). REST endpoints are namespaced under `/v1`.

### 5. Run the WebSocket Gateway

```
go run ./cmd/server -mode websocket
```

The WebSocket service listens on `WEBSOCKET_HOST`:`WEBSOCKET_PORT` (default `0.0.0.0:8081`). Clients connect to `/ws` with `room_id` query parameters.

## API Documentation

See [`API_documentation.md`](API_documentation.md) for comprehensive request/response payloads for all REST endpoints and WebSocket events.

## Testing

- Unit tests:

	```bash
	go test ./...
	```

- Continuous Integration: GitHub Actions run `gofmt` and `go test` on pushes and pull requests (defined in `.github/workflows/ci.yml`).

## Configuration Summary

| Variable | Description | Default |
| --- | --- | --- |
| `APP_ENV` | Environment name (development, production, etc.) | `development` |
| `HTTP_HOST`, `HTTP_PORT` | REST server host/port | `0.0.0.0`, `8080` |
| `WEBSOCKET_HOST`, `WEBSOCKET_PORT` | WebSocket server host/port | `0.0.0.0`, `8081` |
| `POSTGRES_DSN` | Postgres connection string | `postgres://chatterstack:chatterstack@localhost:5432/chatterstack?sslmode=disable` |
| `REDIS_ADDR`, `REDIS_PASSWORD` | Redis connection info | `localhost:6379`, empty |
| `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET` | Secrets for signing tokens | `change-me`, `change-me-too` |
| `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL` | Token lifetime | `15m`, `168h` |
| `RATE_LIMIT_REQUESTS`, `RATE_LIMIT_BURST`, `RATE_LIMIT_WINDOW` | Rate limiting parameters | `100`, same as requests, `1m` |

## Development Tips

- `make` targets can be added or customized to orchestrate linting, tests, or builds.
- Use `air` (configured via `air.toml`) for live reload during development if desired.
- The WebSocket hub already fans out messages received via HTTP—clients receive updates regardless of which interface originated the message.

## Roadmap Ideas

- Expand CI pipeline with static analysis (e.g., `golangci-lint`) and container builds.
- Harden the auth middleware by rotating secrets and supporting multiple issuers/audiences.
- Add metrics/observability (Prometheus, OpenTelemetry).
- Flesh out `api/swagger.yaml` and share a Postman collection for consumer teams.

## License

TBD
