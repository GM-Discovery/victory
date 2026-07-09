# Victory Kernel Maker Field Guide

## Purpose
This guide is for a kernel maker or lower-tier coding agent that needs to change Victory without rediscovering the same runtime traps. Read it before implementing a kernel, then keep it open while testing.

Victory is a browser-based VTT and venue system. The current live table is The Cave. The Greenroom and Trailers are identity/profile spaces. The backend is Go, the database is Postgres, the frontend is static HTML/CSS/JS served by Caddy or the Go backend's routed APIs.

Current canon references:
- [current-state.md](/opt/victory/Construction/current-state.md)
- [roadmap.md](/opt/victory/Construction/roadmap.md)
- PixiJS is currently experimental and lives in First Theater, not The Cave; The Cave remains the proving ground for tool clutter and server-authoritative staging

## Repo Map
- `/opt/victory/backend/` - Go service, HTTP APIs, WebSocket handlers, domain packages.
- `/opt/victory/backend/cmd/victory/main.go` - process entry point, bootstraps kernel surfaces, registers routes.
- `/opt/victory/backend/cmd/victory-bootstrap/main.go` - operator bootstrap CLI for safe authority grants.
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
Kernel 23 added character cards and Cave persona actions. Kernel 24 moved character editing into Greenroom. Kernel 27 added the closed-showing review surface in the Director's Chair. Kernel 28 added the live Director Console for current-showing control.
Kernel 32 adds Discord OAuth as the primary login path while keeping Victory sessions, users, and role authority authoritative.
Kernel 33 adds the operator bootstrap command and canon capture so producer authority still comes from Victory, not from Discord login.

Current product behavior:
- HTTP:
  - `GET /api/auth/providers`
  - `GET /auth/discord/start`
  - `GET /auth/discord/callback`
  - `GET /api/character-cards/me`
  - `POST /api/character-cards`
  - `PATCH /api/character-cards/{id}`
  - `GET /api/showings`
  - `GET /api/showings/{id}/review`
  - `GET /api/director-console/current`
  - `POST /api/showings/{id}/audience-view`
  - `POST /api/showings/{id}/close`
  - `POST /api/showings/start`
  - `POST /api/venues/{slug}/chat-policy`
- WebSocket:
  - `persona/equip`
  - `persona/unequip`
- Broadcast/update:
  - `presence/update`
  - `showing/update`
  - `venue/update`
- Tables:
  - `character_cards`
  - `auth.discord_identities`
  - `auth.oauth_states`
  - `permission_grants`
  - `current_session_personas`

Current authority rule:
- any active performer role (`producer`, `director`, `cast`, `crew`) can draft character cards in Greenroom
- legacy character permission routes still exist in code, but current Greenroom behavior does not depend on them

The Greenroom owns character creation and editing. The Cave should only choose an existing character and put it on or take it off.
The Director's Chair owns closed-showing review. It should read the durable action stream, not render video playback.
The Director Console owns live current-showing control. It should be utilitarian, authority-gated, and tied to the current Cave session rather than to history.

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
- The Cave is the live table and the full-feature proving-ground venue.
- It is acceptable to build runtime tools visibly in The Cave first.
- Once stable, move them into cleaner overlays, drawers, context menus, or secondary surfaces.
- A clean template venue should later be extracted from the organized stage-shell surface.
- Future venues should descend from that cleaned template.
- PixiJS, when present, is a client-side renderer only. Do not let Pixi state become app truth.
- Middle School Stage is the first clean stage-shell venue. Keep it simple, producer-only, and drawer-first.
- The hidden Stage Template venue is the reusable shell descendant and the canonical extraction target.
- First Theater should use the portable overlay above Pixi so renderer and overlay can be compared side by side.
- The First Theater overlay proof marker is smoke-only. Treat it as a test affordance for verifying DOM chrome over Pixi, not as normal venue UI.
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
- **`scripts/smoke/fresh-install.sh --local` leaves an orphaned backend process / undropped throwaway DB**: fixed as of Kernel 61A — the script used to background a `go run ./cmd/victory` inside a subshell and capture `$!`, which is neither `go run`'s own PID nor the actual compiled binary's PID, so `kill "$BACKEND_PID"` in the cleanup trap often missed the real process. It now builds the binary once and runs it directly, plus a `lsof`-based port-cleanup fallback. If you ever see `ss -ltnp | grep 18081` show a stray process after a run, that's this bug regressing — check the backend-start block hasn't been changed back to a backgrounded `go run`.

## Two-Browser / Live-Update Verification Technique

Established during Kernel 61/61A for proving cross-account behavior (one user views/mutates, a second user must not be affected) and websocket live-updates. Reuse this rather than inventing a new approach:

1. Get two real accounts. Prefer creating disposable ones via the real `POST /api/auth/signup` endpoint (captures a genuine session cookie from `Set-Cookie` in the response headers, and gives you a *known* password so you can also test reauth flows) over reusing long-lived fixture accounts (`testflow1`, `tester1`, etc. — fine for read/UI checks, but their passwords aren't known to you). Delete disposable accounts afterward (`DELETE FROM users WHERE handle = '...'` — cascades through most owned data, but tables without `ON DELETE CASCADE` like `showings.created_by` can block it; don't fight that, it's not your problem to fix mid-kernel).
2. For a fixture account where you don't know the password, insert a temporary row directly into `auth.sessions` (raw random token + `digest(raw_token,'sha256')` as `token_hash`, matching `backend/internal/sessions.HashToken`) and delete it when done. **Never do this for Straturli** unless the task specifically requires verifying Straturli's own account — prefer any other real or fixture account.
3. Drive two (or more) simultaneous browser contexts with Playwright (`/tmp/node_modules/playwright`, chromium pre-installed in this environment) — one `context`/`page` per account, cookies set via `context.addCookies`. This environment has outbound internet access and can reach the real deployed site directly (e.g. `https://victory.amurray.family`).
4. For websocket-push proofs specifically: assert the *other* tab's DOM updates without calling reload — that's the actual thing under test, not just that a second `fetch` would return the new value.
5. Clean up: delete temp `auth.sessions` rows and disposable accounts; re-check the real preservation counts (`users`, `location_memberships`, `access_grants`, `character_cards`, Straturli's UUID) haven't shifted for reasons other than your own cleanup.

## Kernel 61 / 61A Notes
For the Player Workbook / Trailer Face system:
- See `operator-notes.md`'s "Kernel 61 / 61A" section for the Account vs Player Workbook vs Trailer Face vs Character Workbook distinction — these are easy to conflate and have completely separate schemas.
- The catalogue (`backend/internal/playerprofile/catalogues/player-profile-v1.0.0.json`) is versioned data, go:embed'd — a new catalogue revision adds a new file and bumps the version, never edits v1.0.0 in place.
- `/ws/player-profile` is a separate, simpler websocket endpoint from `/ws/the-cave` / `/ws/catharsis` — it has no venue-session concept, just auth + a `watch_profile` subscribe message. Don't try to route player-profile invalidation through `ServeVenueWS`.
- `performer_profiles` (the pre-Kernel-61 table) is permanently read-only now — all 6 `/api/profiles/*` routes return `410 Gone`. Do not resurrect writes to it; extend the Player Workbook model instead.

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

## Recording Language
- Showing Review means review of chat, stage speech, reactions, reveals/hides, overlays, index cards, persona changes, and showing/session history.
- Video Recording is future work and not a near-term planning target.
