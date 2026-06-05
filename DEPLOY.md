# Deploying ChatterStack for ~$0–5/month

This guide brings ChatterStack back online with a public **HTTPS** URL and a
working **`wss://`** WebSocket endpoint — cheap enough to leave running as a
portfolio link.

There are two paths. **Path A (recommended)** runs the whole stack on one small
VPS with `docker compose` — it matches the code exactly (Redis over the private
network, no TLS gymnastics) and costs ~$4/mo, or **$0** on an always-free VM.

---

## Path A — Single VPS + Docker Compose (recommended)

### What you need
- A domain (or subdomain) you control — e.g. `chatterstack.example.com`.
  Cheapest option: a free subdomain, or any domain you already own.
- A small Linux VM. Any of these work:
  - **Hetzner CX22** (~€4/mo) — best value, amd64.
  - **DigitalOcean / Vultr** basic droplet (~$5–6/mo), amd64.
  - **Oracle Cloud Always Free** — genuinely $0, but ARM. See the ARM note at
    the bottom before using it.

### 1. Provision the server and install Docker
SSH into the box, then:

```bash
curl -fsSL https://get.docker.com | sh
```

Open the firewall for HTTP/HTTPS (Caddy needs 80 + 443 for Let's Encrypt):

```bash
# ufw-based distros:
sudo ufw allow 80/tcp && sudo ufw allow 443/tcp && sudo ufw allow OpenSSH && sudo ufw enable
```

### 2. Point DNS at the server
Create an **A record**: `chatterstack.example.com` → `<your server IP>`.
Wait until `dig +short chatterstack.example.com` returns that IP before the next
step (Let's Encrypt validation needs it resolving).

### 3. Get the code and configure secrets
```bash
git clone https://github.com/Amaankaa/ChatterStack.git
cd ChatterStack
cp .env.deploy.example .env
```

Edit `.env` and set real values:
- `DOMAIN` — your domain from step 2
- `ACME_EMAIL` — your email (cert notices)
- `POSTGRES_PASSWORD` — a long random string
- `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` — generate each:
  ```bash
  openssl rand -hex 32
  ```

### 4. Build and start the stack
```bash
docker compose -f docker-compose.deploy.yml up -d --build
```

This starts Postgres, Redis, the API, the WebSocket gateway, and Caddy. Caddy
automatically obtains a TLS certificate for your domain on first boot (give it
~30s; watch with `docker compose -f docker-compose.deploy.yml logs -f caddy`).

### 5. Apply the database migration
The schema lives in `db/migrations/0001_init.up.sql` (it already includes the
`messages.updated_at` column, so no follow-up migration is needed for a fresh
database):

```bash
docker compose -f docker-compose.deploy.yml exec -T postgres \
  psql -U chatterstack -d chatterstack < db/migrations/0001_init.up.sql
```

### 6. Verify it's live
```bash
# REST health (through HTTPS + Caddy):
curl -i https://chatterstack.example.com/health

# Register a user:
curl -i -X POST https://chatterstack.example.com/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","username":"demo","password":"Passw0rd!"}'
```

For the WebSocket, grab an access token from `/v1/auth/login`, then connect with
`wss://chatterstack.example.com/ws?room_id=<id>&access_token=<token>`
(use `wscat` or the Postman collection in `api/`).

Put `https://chatterstack.example.com` (REST) on your resume/README as the live
link, and record a 60–90s demo for backup in case you take it down later.

---

## Cost control

To stop paying while keeping data:
```bash
docker compose -f docker-compose.deploy.yml down        # keeps named volumes
```
To bring it back: re-run step 4. To wipe everything (including the DB volume):
```bash
docker compose -f docker-compose.deploy.yml down -v
```
A snapshot/destroy of the VM itself is the surest way to stop all charges on
paid providers — just keep the repo + `.env` values so you can recreate it.

---

## Path B — Fly.io app + managed Postgres (alternative)

More "serverless" and also ~$0 on free allowances, but two caveats:

1. **Two processes.** ChatterStack runs the API and the WebSocket gateway as
   separate modes (`-mode api` / `-mode websocket`). On Fly you'd define two
   process groups (or two apps) sharing the same image.
2. **Redis must be reachable without TLS** — `main.go` builds the Redis client
   with `redis.NewClient(&redis.Options{Addr, Password})` and does **not** set
   `TLSConfig`. Managed Redis that *requires* TLS (e.g. Upstash's default
   `rediss://` endpoint) will fail until you add a `TLSConfig` to that client.
   Use a non-TLS Redis (Fly's own Redis app on the private network, or Railway/
   Render private Redis) — or add ~3 lines of TLS config to `main.go` first.

Managed Postgres like **Neon** works out of the box: it ships `sslmode=require`
in its DSN, which `internal/config` already honors. Set `POSTGRES_DSN` to the
Neon connection string.

Given those caveats, **Path A is the lower-friction way to get the link back.**
  
---

## Note for Oracle Cloud / other ARM VMs
The `Dockerfile` builds natively for the host architecture (no hardcoded
`GOARCH`), so it works on both ARM and x86 servers with no changes — just build
on the server as in step 4.
