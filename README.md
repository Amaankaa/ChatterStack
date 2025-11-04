# ChatterStack

ChatterStack is a scalable real-time chat backend designed with a clean architecture blueprint. This repository contains the initial project scaffold, including HTTP and WebSocket delivery layers, business use cases, and repository adapters for PostgreSQL and Redis.

## Project Overview

- **Language:** Go 1.21+
- **Transport:** REST (HTTP) + WebSockets
- **Storage:** PostgreSQL
- **Cache / PubSub:** Redis
- **Auth:** JWT (access & refresh tokens)
- **Docs:** Swagger / OpenAPI
- **Containers:** Docker & Docker Compose

## Getting Started

1. **Copy `.env.example` to `.env`** and update credentials.
2. **Run dependencies** with `docker-compose up -d`.
3. **Install Go tools** using `go install ./...` (once business logic is implemented).
4. **Start the server** with `go run ./cmd/server`.

## Repository Layout

Refer to the in-repo comments and package docs for details on the clean architecture layers and responsibilities. Each directory contains README stubs that describe its purpose.

## Next Steps

- Implement configuration loading in `internal/config`.
- Flesh out domain models and validation logic in `internal/domain`.
- Implement use case orchestrations in `internal/usecase`.
- Wire up handlers, middleware, and routing in `internal/delivery` and `pkg/middleware`.
- Configure database migrations and repository adapters.
- Complete Swagger definitions in `api/swagger.yaml` and sync the Postman collection.
- Add CI workflows for linting, tests, and build pipelines.

## License

TBD
