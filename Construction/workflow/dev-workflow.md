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
- Cave snapshot: `/api/world/the-cave`
- Cave WebSocket: `/ws/the-cave`
- Greenroom character cards: `/api/character-cards/me`
- Trailers profile draft: `/api/profiles/me`
- Mailbox: `/api/messages`

## Database Changes
Apply migrations manually:
```bash
cd /opt/victory
docker exec -i victory-postgres psql -U victory -d victory < database/migrations/XXX.sql
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

Do not:
- assume the backend serving `8081` is the newest process
- assume a browser `404` means the route is missing without checking the live process
- confuse Showing Review with video recording goals
