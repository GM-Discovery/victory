# Dev Workflow — VICTORY

## Current Note
This file is current workflow guidance. Older kernel-specific workflow assumptions are historical.

For overall current truth, also read:
- [current-state.md](/opt/victory/Construction/current-state.md)
- [kernel-maker-field-guide.md](/opt/victory/Construction/kernel-maker-field-guide.md)

## Core Principle
Two valid runtime modes exist:
- Dev mode: host-Go backend + Docker Postgres
- Install mode: Docker backend + Docker Postgres

Do not confuse them while testing.

## Dev Mode
Use this for active kernel work.

Start Postgres:
```bash
cd /opt/victory
docker compose up -d postgres
```

Run backend on host:
```bash
cd /opt/victory/backend
PORT=8081 DATABASE_URL='postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable' GOCACHE=/tmp/victory-gocache go run ./cmd/victory
```

Optional Discord OAuth environment for Kernel 32:
```bash
DISCORD_CLIENT_ID='<client id>'
DISCORD_CLIENT_SECRET='<client secret>'
DISCORD_REDIRECT_URL='http://127.0.0.1:8081/auth/discord/callback'
DISCORD_OAUTH_SCOPES='identify email'
```

If `8081` is occupied:
```bash
PORT=18081 DATABASE_URL='postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable' GOCACHE=/tmp/victory-gocache go run ./cmd/victory
```

## Install Mode
Use this when validating the deployable path:
```bash
cd /opt/victory
docker compose up -d --build
```

## Common Checks
```bash
curl -s http://127.0.0.1:8081/health
curl -s http://127.0.0.1:8081/api/world/the-cave
wscat -c ws://127.0.0.1:8081/ws/the-cave
GOCACHE=/tmp/victory-gocache go test ./...
git diff --check
```

## Current Route Checks Worth Knowing
- Discord OAuth start: `/auth/discord/start`
- Discord OAuth callback: `/auth/discord/callback`
- Auth providers: `/api/auth/providers`
- Cave snapshot: `/api/world/the-cave`
- Cave WebSocket: `/ws/the-cave`
- First Theater Pixi spike: `/venues/first-theater/`
- The Cave proving ground: `/venues/the-cave/`
- Greenroom character cards: `/api/character-cards/me`
- Trailers profile draft: `/api/profiles/me`
- Mailbox: `/api/messages`

## Database Changes
Apply migrations manually:
```bash
cd /opt/victory
docker exec -i victory-postgres psql -U victory -d victory < database/migrations/XXX.sql
```

Kernel 32 migration:
```bash
docker exec -i victory-postgres psql -U victory -d victory < database/migrations/018_kernel32_discord_oauth.sql
```

Inspect tables:
```bash
docker exec -it victory-postgres psql -U victory -d victory -c '\dt'
```

## Working Rules
Do:
- keep Postgres in Docker
- use `GOCACHE=/tmp/victory-gocache`
- verify whether `8081` is host-Go or Docker before debugging routes
- hard refresh venue pages after frontend changes
- test both HTTP and WebSocket surfaces for Cave-facing kernels
- treat PixiJS as a renderer-only experiment unless a kernel explicitly proves otherwise

Do not:
- assume the backend serving `8081` is the newest process
- assume a browser `404` means the route is missing without checking the live process
- confuse Showing Review with video recording goals
