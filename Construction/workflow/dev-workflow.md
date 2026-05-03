# Dev Workflow — VICTORY (Active Build Mode)

## Purpose
Fast iteration without breaking the install model.

This workflow is for active development on a live server (e.g. `murray-vserver`).

---

## Core Principle

Two modes exist:

### Dev Mode (current use)
- Go backend runs on host
- Postgres runs in Docker
- Fast iteration

### Install Mode (future / validation)
- Backend runs in Docker
- Postgres runs in Docker
- Fully reproducible

Do not confuse them.

---

## Start Dev Mode

### 1. Ensure Postgres is running
```bash
cd /opt/victory
docker compose up -d postgres

Verify:

docker ps
2. Stop Docker backend (if running)
docker stop victory-backend

Reason:

host Go backend uses port 8081
cannot run both at same time
3. Run backend on host
cd /opt/victory/backend
DATABASE_URL='postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable' go run ./cmd/victory

Expected:

victory backend listening on :8081
Test Loop
HTTP
curl -s http://127.0.0.1:8081/health
curl -s http://127.0.0.1:8081/api/world/the-cave
WebSocket
wscat -c ws://127.0.0.1:8081/ws/the-cave

For multi-user presence and attribution checks, open a second `wscat` session or a second browser tab and verify `presence/join` / `presence/leave` plus action attribution.

Kernel 8 identity check:

GOCACHE=/tmp/victory-gocache go test ./internal/network -run TestKernel8IdentitySurfaceAndPersonaNull -v

Kernel 9 profile check:

GOCACHE=/tmp/victory-gocache go test ./internal/profiles ./internal/access ./internal/actions ./internal/network ./internal/world

Kernel 10 mailbox check:

GOCACHE=/tmp/victory-gocache go test ./internal/messages ./internal/profiles ./internal/access ./internal/actions ./internal/network ./internal/world

Kernel 11 note-card check:

GOCACHE=/tmp/victory-gocache go test ./internal/messages ./internal/profiles ./internal/access ./internal/actions ./internal/network ./internal/world

New venue routes:

/venues/greenroom/

/venues/trailers/

New message route:

/mailbox/

New note-card route:

/api/note-cards
Making Code Changes

Typical loop:

edit Go file
stop running process (Ctrl+C)
re-run:
go run ./cmd/victory

Expected restart time: ~1–2 seconds

Optional Upgrade (Auto-Restart)

Install:

go install github.com/cosmtrek/air@latest

Run:

air

Behavior:

watches files
auto rebuilds + restarts backend
Database Changes
Apply migrations manually
docker exec -i victory-postgres psql -U victory -d victory < database/migrations/XXX.sql
Verify
docker exec -it victory-postgres psql -U victory -d victory -c "\dt"
Switching Back to Install Mode

Before pushing or testing install:

cd /opt/victory
docker compose up -d --build

Then test:

curl -s http://127.0.0.1:8081/health
Rules
Do
use host-Go for speed
keep Postgres in Docker
test WebSocket behavior frequently
verify DB state after actions
Do Not
remove Docker deployment path
hardcode host-only assumptions into backend
expose Postgres publicly
rely on dev-only hacks for production behavior
Known Friction
Docker rebuild ~50s (expected)
host-Go requires correct DATABASE_URL
port conflicts if backend container is still running
Current Priority Flow
observe
react (done)
speak (next)
multi-user proof
greenroom / trailers profile surface
info booth / mailbox foundation
note card delivery system
index card element v1
