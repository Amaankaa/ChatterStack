# The Journey: Shipping ChatterStack to Production

I decided to build and ship ChatterStack to become a better backend developer. I wanted a real project with real constraints, production infra, and the full set of backend responsibilities: APIs, WebSockets, persistence, caching, testing, and documentation. It took about two weeks just to deploy — and there were moments I wanted to give up — but I kept pushing because I knew the journey would teach me a ton. It did.

## What We Built
- A real-time chat backend in Go (Golang), following clean architecture principles.
- REST API for auth, users, rooms, and messages, plus a WebSocket gateway for real‑time events.
- PostgreSQL (RDS) for persistence; Redis for cache + pub/sub fan‑out.
- Docker images deployed to ECS Fargate behind an ALB, fronted by CloudFront.
- JWT-based auth with access/refresh lifecycle and secure middleware.
- Rate limiting, logging, validators, helpers, and thorough unit/integration tests.
- API documentation: OpenAPI (Swagger) + Postman collection.

Technologies and “backend fancy terms” we truly touched:
- AWS: ECS Fargate, Application Load Balancer, CloudFront, IAM, Secrets Manager, SSM Parameter Store, CloudWatch Logs
- Databases: Amazon RDS (PostgreSQL), Redis 7
- Containers: Docker + ECR, multi-stage builds
- Networking/Security: TLS/HTTPS, CORS, JWT, rate limiting
- Patterns: Clean architecture, repositories, use cases, pub/sub, WebSocket hubs
- Tooling: `go test` (unit/integration), Postman, Swagger/OpenAPI, curl, `jq`

## The Struggles (and Lessons)

### 1) Database Connectivity and SSL
Early on, auth endpoints were returning 500. CloudWatch logs revealed Postgres connection failures: SSL/auth errors. We fixed DSNs, enforced `sslmode=require`, and aligned ECS Secrets/SSM parameters with the correct RDS credentials. After that, “postgres ping succeeded” started appearing in logs.

Lesson: Be explicit about SSL and credentials. Mask DSNs in logs to avoid leaking sensitive info while still being diagnostic.

### 2) CloudFront vs Authorization Header
We verified endpoints directly via the ALB — `Authorization: Bearer <jwt>` worked perfectly. But through CloudFront, `Authorization` was silently omitted. We tried:
- Switching to managed origin request policies like “AllViewerExceptHostHeader”.
- Creating custom origin request policies to include `Authorization`.
- Using query string forwarding for things like `email`.

CloudFront stubbornly refused to forward `Authorization`. The fallback we adopted:
- Accept `X-Auth-Token` at the backend as an alternative.
- Add a custom origin request policy to forward `X-Auth-Token` (and preflight headers).

That unblocked CloudFront. Authenticated requests with `X-Auth-Token` began returning 200.

Lesson: If infra blocks `Authorization`, define and forward a custom header and support it server‑side.

### 3) IAM AccessDenied on CloudFront Functions
We explored a CloudFront Function workaround to map `Authorization` → `X-Auth-Token`. Attempts to create/list functions hit `AccessDenied`, even after broadening inline IAM policies. The denial likely came from higher‑level SCP/permission boundaries. We accepted the constraint and doubled down on `X-Auth-Token`.

Lesson: Sometimes the fastest route is to respect guardrails and design around them.

### 4) CORS Preflight Through the CDN
Frontend reported that OPTIONS preflight returned 200 but lacked the expected `Access-Control-Allow-*` headers. We:
- Updated backend CORS middleware to include `X-Auth-Token` and `Access-Control-Max-Age`.
- Disabled CloudFront caching for `/v1/*` and ensured preflight headers were forwarded.
- Removed CloudFront ResponseHeadersPolicy for `/v1/*` so origin CORS headers pass through unchanged.

After redeploy and invalidation, preflight via CloudFront started returning full headers, including `Access-Control-Allow-Headers: Content-Type, Authorization, X-Auth-Token`.

Lesson: Prefer origin-driven CORS headers and keep CDN behavior simple when possible.

### 5) WebSocket Authentication and Frontend Interception
We supported two auth paths:
- REST via CloudFront using `X-Auth-Token`.
- WebSocket using the `access_token` query string.

The frontend initially tried to intercept and inject `Authorization` headers mid-connection for WebSockets, which was unreliable given the CDN behavior. Moving the WebSocket auth to a query param was cleaner and consistent.

Lesson: For CDNs that alter headers unpredictably, query params can be a pragmatic auth transport for WSS (within your threat model).

### 6) Typing Events Missing `room_id`
The frontend flagged a UX bug: `typing_start` / `typing_stop` events sometimes arrived without `room_id`. Multi-room clients couldn’t scope indicators correctly. We updated the payload to always include `room_id` (removed `omitempty`) and redeployed. That fixed cross-room typing indicators.

Lesson: Small payload conventions make a big difference for multi-channel UX.

### 7) HTTPS and Edge Routing
We also wrestled with HTTP→HTTPS enforcement and edge routing (including Cloudflare configuration in front of public origins). Getting consistent TLS, redirects, and header behavior across CloudFront/Cloudflare/ALB took iteration. Ultimately, we ensured:
- HTTPS for REST and WSS.
- No conflicting header overrides between CDN layers.
- CORS and auth transport remained predictable end‑to‑end.

Lesson: Layered CDNs/proxies demand deliberate ownership of TLS, headers, and cache/policy precedence.

### 8) Tests, Docs, and Deploy Hygiene
During CORS edits, some older HTTP handler tests surfaced unrelated 301/307 expectations — we kept scope tight and avoided chasing non-essential changes. We added:
- An OpenAPI spec documenting CloudFront/ALB servers and dual security schemes (Bearer + `X-Auth-Token`).
- A Postman collection with variables (`base_cf`, `access_token`) and example requests.
- README examples for fetch (`X-Auth-Token`) and WebSocket (`access_token` query).

Lesson: Documentation is a product. Test only what matters for the change; avoid scope creep.

## Timeline Reality
- Two weeks focused on deployment alone.
- Multiple cycles of build → push → register task def → update ECS service → wait for steady state → validate via curl.
- Many CloudFront config changes gated by IAM; origin policies, cache policies, invalidations, and patience.
- Several instances of “I might quit” replaced by “let’s learn and iterate.”

## Final Architecture Snapshot
- Client → CloudFront → ALB → ECS Fargate → API (Gin) + WebSocket Hub
- RDS Postgres for storage (via pgx) and Redis for cache/pub/sub
- Auth: JWT (access/refresh), backend accepts `X-Auth-Token` (CloudFront); ALB supports `Authorization: Bearer`
- WebSocket: `access_token` query; rooms joined and events broadcast via Redis
- CORS: Origin-handled, forwarded through CloudFront without ResponseHeadersPolicy for `/v1/*`

## What I Learned
- Debugging infra is real engineering work: IAM, policies, managed vs custom behaviors, and knowing when to stop fighting the platform.
- Building production‑grade systems isn’t just code — it’s observability, deploy hygiene, edge behavior, and client ergonomics.
- Persistence and iteration beat frustration. I used almost every “fancy backend term” for real: AWS, ECS, load balancers, RDS/PostgreSQL, Redis pub/sub, Docker, clean architecture, unit/integration testing, API documentation, WebSockets, CORS, JWT, CI, and more.

Most importantly, I stuck with it. The result is a working, documented, and tested chat backend that I’m proud of — and a journey that made me a much stronger backend developer.
