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
# ChatterStack

ChatterStack is a real-time chat backend written in Go that exposes a REST API and a WebSocket gateway for low-latency messaging. It follows clean architecture principles so delivery layers (HTTP, WebSocket) stay isolated from domain logic and infrastructure adapters (PostgreSQL, Redis). Use it as a reference architecture or as a starting point for production chat workloads.

---

## Table of Contents
- [Highlights](#highlights)
- [System Architecture](#system-architecture)
- [Tech Stack](#tech-stack)
- [Project Layout](#project-layout)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Database & Migrations](#database--migrations)
- [Running the Services](#running-the-services)
- [API Overview](#api-overview)
- [WebSocket Events](#websocket-events)
- [Testing](#testing)
- [Deployment Notes](#deployment-notes)
- [Troubleshooting](#troubleshooting)
- [Development Workflow](#development-workflow)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

## Highlights
- JWT-based auth with refresh token rotation and logout support.
- REST endpoints for users, rooms, and messages plus real-time WebSocket fan-out.
- Message edit/delete lifecycle with Redis-backed broadcast of `message.deleted` events.
- Typing indicator support (`typing_start` / `typing_stop`) with username broadcast and rate limiting.
- Postgres repositories with SQL migrations and Redis caching/pub-sub.
- Structured logging, request rate limiting, and consistent error handling helpers.
- Unit tests at delivery, use case, and repository layers.
- API reference maintained in [`API_documentation.md`](API_documentation.md) and Postman assets under `api/`.

## System Architecture

ChatterStack ships as two cooperating processes:

1. **HTTP API server** (REST) managing authentication, room/member CRUD, and message history.
2. **WebSocket hub** providing bi-directional messaging, typing indicators, and deletion broadcasts.

Both services share PostgreSQL for persistence and Redis for caching and cross-node message propagation. Redis pub/sub ensures every WebSocket node hears about messages created through the HTTP API or another WebSocket client.

## Tech Stack
- Go 1.21+
- Gin (HTTP routing)
- Gorilla WebSocket
- PostgreSQL + pgx
- Redis 7 (cache/pub-sub)
- Docker & Docker Compose (optional for local development)
- Make, Air, and GitHub Actions integrations available out of the box

## Project Layout
```
cmd/server/main.go        # Application entrypoint (launches REST or WS mode)
internal/config           # Environment + config loader
internal/delivery/http    # Gin handlers, middleware wiring, payload helpers
internal/delivery/websocket # Hub, client, relay, and WS payloads
internal/domain           # Core business logic interfaces and errors
internal/repository       # PostgreSQL and Redis adapters + mocks
internal/usecase          # Application services orchestrating domain + repositories
pkg/middleware            # Shared HTTP middleware (auth, logger, rate limiter)
api/                      # Swagger spec + Postman collection
db/migrations             # SQL migrations (apply in order)
```

## Prerequisites
- Go 1.21 or later
- PostgreSQL 14+
- Redis 7+
- Docker & Docker Compose (optional but recommended for local onboarding)
- `make`, `curl`, and `jq` for convenience scripts/tests

## Quick Start
1. **Clone the repo**
	```bash
	git clone https://github.com/Amaankaa/ChatterStack.git
	cd ChatterStack
	```
2. **Copy environment template**
	```bash
	cp .env.example .env
	```
3. **Start Postgres + Redis** (optional but easiest for local dev)
	```bash
	docker-compose up -d
	```
4. **Apply migrations** (see [Database & Migrations](#database--migrations))
5. **Launch both services in separate terminals**
	```bash
	go run ./cmd/server -mode api        # REST API on :8080 by default
	go run ./cmd/server -mode websocket  # WebSocket gateway on :8081
	```
6. **Hit the health endpoints**
	```bash
	curl -i http://localhost:8080/v1/health
	```
7. **Open the WebSocket** using your client of choice (e.g., wscat or the supplied Postman collection).

## Configuration

Environment variables are parsed by `internal/config`. Defaults support local development; override in production. Key values:

| Variable | Description | Default |
| --- | --- | --- |
| `APP_ENV` | Environment label: `development`, `staging`, `production`, etc. | `development` |
| `HTTP_HOST`, `HTTP_PORT` | REST bind host/port | `0.0.0.0`, `8080` |
| `WEBSOCKET_HOST`, `WEBSOCKET_PORT` | WebSocket bind host/port | `0.0.0.0`, `8081` |
| `POSTGRES_DSN` | PostgreSQL DSN including credentials and database | `postgres://chatterstack:chatterstack@localhost:5432/chatterstack?sslmode=disable` |
| `REDIS_ADDR`, `REDIS_PASSWORD` | Redis connection settings | `localhost:6379`, empty |
| `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET` | Signing keys for tokens | `change-me`, `change-me-too` |
| `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL` | Token expiration windows | `15m`, `168h` |
| `RATE_LIMIT_REQUESTS`, `RATE_LIMIT_BURST`, `RATE_LIMIT_WINDOW` | Sliding window rate limiter | `100`, `100`, `1m` |
| `LOG_LEVEL` | Minimum log level (`debug`, `info`, etc.) | `info` |

The server surfaces config errors early so missing secrets or malformed DSNs fail fast.

## Database & Migrations

All schema definitions live under `db/migrations`. The baseline migration (`0001_init.up.sql`) creates user, room, membership, and message tables. When deploying the message edit/delete feature, run the follow-up migration to ensure the `messages.updated_at` column exists:

```sql
ALTER TABLE messages
	 ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
```

We recommend `golang-migrate` for applying migrations:

```bash
migrate -path db/migrations -database "$POSTGRES_DSN" up
```

## Running the Services

```bash
go run ./cmd/server -mode api        # Starts REST API
go run ./cmd/server -mode websocket  # Starts WebSocket hub
```

Supply `-config /path/to/config.yaml` if you prefer static config files. Both modes honour environment variables.

- **REST base URL:** `http://{HTTP_HOST}:{HTTP_PORT}/v1`
- **WebSocket URL:** `ws://{WEBSOCKET_HOST}:{WEBSOCKET_PORT}/ws?room_id=<roomID>&room_id=<roomID>`

The WebSocket hub validates room membership during the upgrade handshake and rejects unauthorized joins.

## API Overview

The REST API covers:

- **Auth:** register, login, refresh, logout.
- **Users:** fetch by ID/email, update presence.
- **Rooms:** create, manage membership, search, ensure direct rooms, delete.
- **Messages:** send, list (paginated or around a specific message), search, mark delivered/read, edit, delete.

Message resources now include both `created_at` and `updated_at` timestamps. The full reference with example payloads lives in [`API_documentation.md`](API_documentation.md).

## WebSocket Events

Clients send JSON envelopes shaped as `{ "event": string, "data": object }`. Supported events include:

### Client → Server
- `send_message`: broadcast chat content.
- `typing_start` / `typing_stop`: notify room peers; optional `room_id` when connected to a single room.

### Server → Client
- `receive_message`: emitted for every new or edited message.
- `message.deleted`: indicates a message was removed.
- `typing_start` / `typing_stop`: mirrors typing activity to other participants; payload includes `username` and `room_id`.

Events route through Redis so WebSocket nodes stay in sync with HTTP-originated actions.

## Testing

Run the full test suite:

```bash
go test ./...
```

Key packages with dedicated tests:

- `internal/delivery/websocket`: hub/client behaviour, typing indicators, relay integration.
- `internal/repository/postgres`: repository queries with pgx mocks.
- `pkg/middleware`: auth and rate-limiting helpers.

CI (see `.github/workflows/ci.yml`) enforces formatting via `gofmt` and executes `go test ./...` on every push/PR.

## Deployment Notes
- Run both API and WebSocket services; scale horizontally behind a load balancer if needed.
- Ensure Postgres and Redis are reachable with production credentials.
- Seed initial migrations before first boot. Monitor schema drift after feature additions.
- Set strong, rotated JWT secrets and adjust token TTLs for your security posture.
- Configure rate limiting per your expected traffic profile.
- Use HTTPS and secure WebSocket (`wss://`) behind a reverse proxy (Nginx, Traefik, AWS ALB, etc.).

## Client Integration Examples

Use `X-Auth-Token` for REST via CloudFront and `access_token` query for WebSocket.

### REST (CloudFront)

```ts
// Example using fetch from a browser app
const CF_BASE = 'https://d1176qoi9kdya5.cloudfront.net/v1';

async function getUserByEmail(email: string, accessToken: string) {
	const res = await fetch(`${CF_BASE}/users?email=${encodeURIComponent(email)}` , {
		method: 'GET',
		headers: {
			'Content-Type': 'application/json',
			'X-Auth-Token': accessToken,
		},
		credentials: 'include', // optional if you later use cookies
	});
	if (!res.ok) throw new Error(`HTTP ${res.status}`);
	return res.json();
}
```

### WebSocket (CloudFront)

```ts
const CF_WS = 'wss://d1176qoi9kdya5.cloudfront.net/ws';

function connectWS(accessToken: string, roomIds: string[]) {
	const params = new URLSearchParams();
	roomIds.forEach((id) => params.append('room_id', id));
	params.set('access_token', accessToken);

	const ws = new WebSocket(`${CF_WS}?${params.toString()}`);

	ws.onopen = () => {
		console.log('WS connected');
	};
	ws.onmessage = (ev) => {
		const msg = JSON.parse(ev.data);
		console.log('WS event', msg);
	};
	ws.onclose = () => {
		console.log('WS closed');
	};
	ws.onerror = (err) => {
		console.error('WS error', err);
	};

	return ws;
}
```

### CORS

- Default allowed origin: `https://chatterstack.vercel.app`.
- Preflight includes `Access-Control-Allow-Headers: Content-Type, Authorization, X-Auth-Token` and `Access-Control-Max-Age: 600`.
- For local dev, set env `CORS_ALLOWED_ORIGINS=https://localhost:3000` (and any others) before starting the server.

## Troubleshooting
- **`column messages.updated_at does not exist`**: run the latest migration (see [Database & Migrations](#database--migrations)).
- **WebSocket typing indicators missing**: confirm the WebSocket process was restarted after deploying the typing feature and that clients include `room_id` when joined to multiple rooms.
- **Redis pub/sub not relaying messages**: check Redis credentials, ensure pattern subscriptions permit `rooms:*:messages`, and verify firewall rules between processes.
- **Unauthorized errors on REST calls**: confirm `Authorization: Bearer <token>` header is present and refresh tokens are rotated after expiry.

## Development Workflow
- Use `air` (configured via `air.toml`) for hot reloading during feature work.
- Apply `gofmt` before committing; CI will enforce it.
- Add targeted unit tests alongside features—especially for WebSocket events to avoid regressions.
- When extending the API, update [`API_documentation.md`](API_documentation.md) and, if relevant, the Postman collection in `api/postman.json`.

## Roadmap
- Harden CI with static analysis (`golangci-lint`) and vulnerability scans.
- Add observability hooks (Prometheus metrics, OpenTelemetry traces).
- Publish Docker images with multi-stage builds.
- Expand WebSocket features (presence updates, read receipts fan-out).
- Complete the Swagger spec under `api/swagger.yaml`.

## Contributing
Issues and pull requests are welcome. Please include tests for new behaviour and keep the documentation in sync. Run `gofmt` and `go test ./...` before submitting.

## License

TBD
