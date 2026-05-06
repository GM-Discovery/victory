# Victory Kernel Maker Field Guide

## Purpose
This guide is for a kernel maker or lower-tier coding agent that needs to change Victory without rediscovering the same runtime traps. Read it before implementing a kernel, then keep it open while testing.

Victory is a browser-based VTT and venue system. The current live table is The Cave. The Greenroom and Trailers are identity/profile spaces. The backend is Go, the database is Postgres, the frontend is static HTML/CSS/JS served by Caddy or the Go backend's routed APIs.

## Repo Map
- `/opt/victory/backend/` - Go service, HTTP APIs, WebSocket handlers, domain packages.
- `/opt/victory/backend/cmd/victory/main.go` - process entry point, bootstraps kernel surfaces, registers routes.
- `/opt/victory/backend/internal/actions/` - append-only action types, validation, authority rules.
- `/opt/victory/backend/internal/network/` - Cave WebSocket hub, presence, snapshots, live broadcasts.
- `/opt/victory/backend/internal/world/` - world and venue snapshot projection.
- `/opt/victory/backend/internal/access/` - venue visibility, roles, grants.
- `/opt/victory/backend/internal/identity/` - users, sessions, auth, invites, role helpers.
- `/opt/victory/backend/internal/profiles/` - Trailers and Greenroom profile surfaces.
- `/opt/victory/backend/internal/characters/` - character cards, drafting grants, session personas.
- `/opt/victory/database/migrations/` - SQL migrations. Add a new numbered file for schema changes.
- `/opt/victory/frontend/` - static app shell and venue pages.
- `/opt/victory/frontend/venues/the-cave/index.html` - live table UI and inline Cave client.
- `/opt/victory/frontend/venues/greenroom/index.html` - public profile and character dressing room.
- `/opt/victory/frontend/venues/trailers/index.html` - performer profile drafting and publishing.
- `/opt/victory/Construction/` - kernel specs, reports, operator notes, workflow docs.

## Domain Vocabulary
- **Location** - hosted world-space. Contains lots, venues, users, memberships, libraries.
- **Lot** - collection of venues inside a location.
- **Venue** - experience surface and rules context, such as `the-cave`, `greenroom`, `trailers`.
- **Library** - reusable element ownership. Reusable assets belong here rather than inside one venue.
- **Element** - reusable object or asset. `context_class` distinguishes props, scenery, cards, and future types.
- **Placement** - instruction that places an element into a venue/session context.
- **Session** - live instance of a venue. Actions happen here.
- **Action** - append-only session event with server-assigned ordering. Do not rewrite old actions to change history.
- **Presence** - live connection state. It is in-memory and ephemeral, not canonical history.
- **User identity** - accountable account identity. Never replace it with a character/persona.
- **Character card** - story persona owned by a user. It can be drafted, edited, and equipped.
- **Persona** - the currently equipped character for a Cave session. It sits on top of the user identity.
- **Producer/Director/Cast/Crew/Audience** - in-app roles. Producer/director are normal app authority, not shell or host authority.
- **Operator** - infrastructure authority outside ordinary app permissions.

## Current Character Kernel Baseline
Kernel 23 added character cards and Cave persona actions:
- HTTP:
  - `GET /api/character-cards/me`
  - `POST /api/character-cards`
  - `PATCH /api/character-cards/{id}`
  - `POST /api/character-card-permissions`
  - `POST /api/character-card-permissions/revoke`
- WebSocket:
  - `persona/equip`
  - `persona/unequip`
- Broadcast/update:
  - `presence/update`
- Tables:
  - `character_cards`
  - `permission_grants`
  - `current_session_personas`

Kernel 24 moves character creation and editing to The Greenroom. The Cave should only choose an existing character and put it on or take it off.

## Runtime Modes
There are two valid ways to run Victory. Do not mix them accidentally.

### Dev Mode
Use this for active kernel coding:
- Postgres runs in Docker as `victory-postgres`.
- Backend runs directly on the host with `go run`.
- Default backend port is `8081`, unless you set `PORT`.
- Host-Go must use `127.0.0.1:5432` in `DATABASE_URL`.

Start from repo root:
```bash
cd /opt/victory
docker compose up -d postgres
```

Run backend from `backend/`:
```bash
cd /opt/victory/backend
PORT=8081 DATABASE_URL='postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable' GOCACHE=/tmp/victory-gocache go run ./cmd/victory
```

If port `8081` is already occupied, either stop the old backend or use a temporary port:
```bash
PORT=18081 DATABASE_URL='postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable' GOCACHE=/tmp/victory-gocache go run ./cmd/victory
```

Health check:
```bash
curl -s http://127.0.0.1:8081/health
```

### Install Mode
Use this to validate the deployable path:
```bash
cd /opt/victory
docker compose up -d --build
```

In install mode:
- Backend runs in the `victory-backend` container.
- Postgres host inside Docker is `victory-postgres`.
- The container `DATABASE_URL` is `postgres://victory:REDACTED@victory-postgres:5432/victory?sslmode=disable`.
- Caddy proxies `/api/*` and `/ws/*` to `localhost:8081`.

## Ports And Names
- Backend HTTP/WebSocket: `8081` by default.
- Temporary alternate backend often used by agents: `18081`.
- Postgres host binding: `127.0.0.1:5432`.
- Postgres container name: `victory-postgres`.
- Backend container name: `victory-backend`.
- Caddy config: `/opt/victory/Caddyfile`.

Common checks:
```bash
docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
docker logs --tail 80 victory-backend
docker logs --tail 80 victory-postgres
curl -s http://127.0.0.1:8081/health
```

If `8081` is busy:
```bash
sudo ss -ltnp | grep ':8081'
```

Decide whether you are testing host-Go or Docker backend. Do not leave both fighting for the same port.

## Rebuild And Restart Rules
Use host-Go for fast local code changes:
1. Stop the old `go run` process.
2. Re-run `go run ./cmd/victory` from `/opt/victory/backend`.
3. Refresh the browser.

Use Docker rebuild when validating install mode or Dockerfile/dependency changes:
```bash
cd /opt/victory
docker compose up -d --build
```

If the backend behavior does not match your code, check for a stale process:
- A host `go run` may still be serving `8081`.
- The `victory-backend` container may still be serving `8081`.
- Browser cache may be holding an old static file. Hard refresh venue pages.

## Database And Migration Rules
Schema changes need two paths:
- A numbered SQL migration in `database/migrations/`.
- A runtime `EnsureKernelXX...Surface` bootstrap if the existing project pattern has one for that surface.

Apply a migration manually:
```bash
cd /opt/victory
docker exec -i victory-postgres psql -U victory -d victory < database/migrations/017_kernel24_character_sheet_links.sql
```

Inspect tables:
```bash
docker exec -it victory-postgres psql -U victory -d victory -c '\dt'
```

Rules:
- Do not destructively rewrite action history.
- Prefer additive migrations: `ADD COLUMN IF NOT EXISTS`, new tables, new indexes.
- Make migrations idempotent where practical.
- Keep bootstrap SQL compatible with already-running databases.
- Be careful with JSONB payload changes: omitted optional fields should not erase stored metadata on PATCH unless that is intentional.

## Action Stream Rules
The `actions` table is canonical session history. Treat it like an event log.

Do:
- Append new actions for new events.
- Validate authority on the server.
- Enrich actor identity server-side.
- Preserve `persona: null` fallback for old actions.
- Keep current persona as action actor metadata at action time.

Do not:
- Trust client-sent actor identity.
- Mutate old actions to fix display.
- Collapse chat, speech, reaction, and persona actions into one ambiguous lane.
- Treat presence as history.

## Frontend Rules
- The Cave is the live table. Keep it focused on session actions, presence, chat, speech, cards, and live persona use.
- The Greenroom is public profile display plus character dressing room.
- Trailers owns performer profile drafting and publishing.
- Avoid putting full editors into The Cave unless the kernel explicitly says the live table owns that workflow.
- Inline venue scripts should pass `node --check` after extraction.

Inline script check examples:
```bash
tmp=/tmp/the-cave-inline.js
sed -n '/<script>/,/<\/script>/p' frontend/venues/the-cave/index.html | sed '1d;$d' > "$tmp"
node --check "$tmp"
```

```bash
tmp=/tmp/greenroom-inline.js
sed -n '/<script>/,/<\/script>/p' frontend/venues/greenroom/index.html | sed '1d;$d' > "$tmp"
node --check "$tmp"
```

## Backend Test Commands
Run from `/opt/victory/backend`.

Full suite:
```bash
GOCACHE=/tmp/victory-gocache go test ./...
```

Focused suites:
```bash
GOCACHE=/tmp/victory-gocache go test ./internal/actions ./internal/network ./internal/world
GOCACHE=/tmp/victory-gocache go test ./internal/characters
GOCACHE=/tmp/victory-gocache go test ./internal/profiles ./internal/access
```

Use `GOCACHE=/tmp/victory-gocache` because agents often run in restricted environments where the default Go cache location is not writable.

Always finish with:
```bash
git diff --check
```

## Common Failure Modes
- **Port conflict on 8081**: Docker backend and host-Go backend are both running. Stop one or use `PORT=18081`.
- **Wrong database host**: Host-Go uses `127.0.0.1`; Docker backend uses `victory-postgres`.
- **Postgres not running**: Start `docker compose up -d postgres`.
- **Stale backend**: Code changed, but the old process is still serving. Restart the process actually bound to the port.
- **Stale static frontend**: Hard refresh the venue page. If served by Caddy, it reads `/opt/victory/frontend`.
- **Forgot `GOCACHE`**: Go tests fail because cache path is not writable. Add `GOCACHE=/tmp/victory-gocache`.
- **Migration exists but bootstrap missing**: Fresh databases work, existing databases do not, or vice versa. Add both paths when the repo pattern expects both.
- **Client claims authority**: Server must resolve current user, role, session, and persona. Client payloads are requests, not facts.
- **Presence mistaken for history**: Presence is ephemeral and in-memory. Actions are the durable record.
- **Dirty worktree collisions**: Read `git status --short` before editing. Do not revert unrelated user changes.

## Kernel Implementation Checklist
1. Read `Construction/OperatorLogs/operator-notes.md` and the newest relevant kernel docs.
2. Check `git status --short`.
3. Locate the existing package and route pattern before inventing a new one.
4. Identify database, API, WebSocket, snapshot, and frontend surfaces.
5. Add additive migrations and bootstrap SQL if schema changes.
6. Keep authority checks server-side.
7. Preserve append-only actions.
8. Update snapshots and live broadcasts together when a live state changes.
9. Keep The Cave live-session focused; move drafting/admin workflows to the right venue.
10. Run Go tests with `GOCACHE=/tmp/victory-gocache`.
11. Run `node --check` for edited inline venue scripts.
12. Run `git diff --check`.
13. Record the important result in Construction notes when the kernel changes system behavior.

## Kernel 24 Notes
For Greenroom character dressing:
- Character card payloads include `sheet_links`.
- Sheet links are JSON metadata on the character card.
- No playable sheet routes, sheet renderers, macros, dice, stats, or rules execution belong in Kernel 24.
- The Cave keeps `persona/equip`, `persona/unequip`, and `presence/update`.
- The Greenroom owns New/Edit/Save for character cards.
