# Dev Workflow — VICTORY

## Current Note
This file is current workflow guidance. Older kernel-specific workflow assumptions are historical.

For overall current truth, also read:
- [current-state.md](../current-state.md)
- [kernel-maker-field-guide.md](../kernel-maker-field-guide.md)

Paths below are given relative to the repository root (wherever you cloned Victory) — they are examples, not law. Substitute your own clone location.

## Core Principle
Three valid runtime modes exist:
- **Dev mode**: host-Go backend + Docker Postgres (day-to-day kernel work)
- **Install mode**: Docker backend + Docker Postgres (validating the deployable server/Linux path)
- **Windows consumer mode**: the packaged installer built by Kernel 100 (Velopack-based, bundled Postgres/Caddy/cloudflared) — see `Docs/Operator/Windows Install Guide.md`; not a mode you run from this source tree directly

Do not confuse them while testing.

## Dev Mode
Use this for active kernel work.

Start Postgres:
```bash
cd packaging/podman
docker compose up -d postgres
```

Run backend on host (from the repo root):
```bash
cd backend
PORT=8081 DATABASE_URL='postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory?sslmode=disable' GOCACHE=/tmp/victory-gocache go run ./cmd/victory
```

Optional Discord OAuth environment for Kernel 32:
```bash
DISCORD_CLIENT_ID='<client id>'
DISCORD_CLIENT_SECRET='<client secret>'
DISCORD_REDIRECT_URL='http://127.0.0.1:8081/auth/discord/callback'
DISCORD_OAUTH_SCOPES='identify email'
```

Operator bootstrap command for Kernel 33:
```bash
cd backend
DATABASE_URL='postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory?sslmode=disable' GOCACHE=/tmp/victory-gocache go run ./cmd/victory-bootstrap producer --discord-user-id <discord_user_id>
```

If `8081` is occupied:
```bash
PORT=18081 DATABASE_URL='postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory?sslmode=disable' GOCACHE=/tmp/victory-gocache go run ./cmd/victory
```

## Install Mode
Use this when validating the deployable path:
```bash
cd packaging/podman
docker compose up -d --build
```
(Requires a `.env` here first -- run `./generate-env.sh` once if one doesn't exist yet. Kernel 96: the repo root's own docker-compose.yml, an old bespoke production file, is retired -- this is the one deployment path now, for production and local dev alike. Production additionally layers `compose.production.yml` for its shared-Caddy `edge_net` wiring: `docker compose -f compose.yml -f compose.production.yml up -d --build`.)

## Common Checks
```bash
curl -s http://127.0.0.1:8081/health
curl -s http://127.0.0.1:8081/api/world/the-cave
wscat -c ws://127.0.0.1:8081/ws/the-cave
GOCACHE=/tmp/victory-gocache TEST_DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory_test?sslmode=disable" go test ./...
git diff --check
```

As of Kernel 64, `TEST_DATABASE_URL` is required for `go test ./...` to fully pass. Later access, social, Show Run, Show, Scene, Cue, Third Place, and world tests joined the original identity/network/assets DB-backed suites and hard-fail without it (by design, not a bug). See "Database Changes" below for one-time setup. Never point `TEST_DATABASE_URL` at the live `victory` database.

## Current Route Checks Worth Knowing
- Discord OAuth start: `/auth/discord/start`
- Discord OAuth callback: `/auth/discord/callback`
- Auth providers: `/api/auth/providers`
- Operator producer bootstrap: `go run ./cmd/victory-bootstrap producer --discord-user-id <discord_user_id>`
- Cave snapshot: `/api/world/the-cave`
- Cave WebSocket: `/ws/the-cave`
- First Theater Pixi spike: `/venues/first-theater/`
- The Cave proving ground: `/venues/the-cave/`
- Greenroom character cards: `/api/character-cards/me`
- Trailers profile draft: `/api/profiles/me`
- Mailbox: `/api/messages`
- Stage Management: `/venues/show-runs/`
- Show Runs: `/api/show-runs`
- Shows: `/api/shows/{show_id}`
- Scene Library: `/api/scenes`
- Show Scene placements: `/api/shows/{show_id}/scenes`
- Placement Cues: `/api/shows/{show_id}/scenes/{placement_id}/cues`
- Player Cue buttons: `/api/shows/{show_id}/scenes/{placement_id}/player-cues`
- Cue GO: `/api/cues/{cue_id}/go`

## Database Changes
Apply migrations manually:
```bash
docker exec -i victory-postgres psql -U victory -d victory < backend/migrations/XXX.sql
```

Inspect tables:
```bash
docker exec -it victory-postgres psql -U victory -d victory -c '\dt'
```

### Creating a new migration
Migrations live in `backend/migrations/`, numbered sequentially and named `NNN_kernelXX_description.sql` (see the directory for current conventions and the highest existing number). The runner (`backend/internal/migrate/migrate.go`) sorts by filename and tracks an applied-checksum ledger — it does **not** require contiguous numbering (a gap in the sequence is not itself a bug), but every migration must be safe to replay against an already-migrated database, not just an empty one (see "Migration replay matters" below). Some venues/surfaces are established by Go-side `Ensure*Surface` bootstrap code at backend startup rather than by a migration at all — check `backend/internal/*/seed*.go`-style files before assuming a missing row means a missing migration.

### Dedicated test database (Kernel 64)
`go test ./...` no longer touches the live `victory` database. Set up the dedicated `victory_test` database once (safe to re-run - non-destructive):
```bash
TEST_DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory_test?sslmode=disable" \
  scripts/test/setup-test-database.sh
```
This creates `victory_test` on the same Postgres container if it doesn't exist, applies every migration, and boots the real backend once against it (so Go-side `Ensure*Surface` bootstrap runs too - some venues, like `first-theater`, only exist because of that, not because of any SQL migration). For a full wipe-and-rebuild from empty:
```bash
CONFIRM_TEST_DB_RESET=1 \
  TEST_DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory_test?sslmode=disable" \
  scripts/test/reset-test-database.sh
```
All DB-touching Go tests should obtain their pool via `backend/internal/dbtest.OpenTestPool(t)` rather than dialing `TEST_DATABASE_URL` directly — it centralizes the safety checks above and (as of Kernel 96/97 work) also fills in `DEFAULT_LOCATION_SLUG` if unset, so tests get a consistent default Location without each one hardcoding it.
Both scripts refuse to run against anything that isn't clearly a dedicated test database (see `scripts/test/require-isolated-database.sh`). `scripts/smoke/fresh-install.sh --local` is unrelated to this - it manages its own fully disposable database per run.

Migration replay matters: these scripts reapply all migration files and do not use a migrations ledger. A column rename/drop must be safe when the entire sequence is replayed over an already-migrated database, not only on an empty database. See the Kernel 70 operator notes.

## Browser Proof Scripts (Playwright)
Playwright is not vendored in the repo. Install it into a scratch directory (e.g. `/tmp/node_modules`, with browsers cached under `~/.cache/ms-playwright`) and run proof scripts with `NODE_PATH` pointed at it:
```bash
NODE_PATH=/tmp/node_modules node scripts/smoke/kernel62-browser.js
```
If the scratch install has been cleared, reinstall with `npm i playwright` in that scratch dir and `npx playwright install chromium`. Scripts target the deployed site by default (`--host-resolver-rules` maps the domain to 127.0.0.1) and create clearly-named disposable accounts. **As of Kernel 84: production password signup is closed** (`PASSWORD_SIGNUP_ENABLED=false`, confirmed live in Kernel 83), so `/api/auth/signup` is no longer a working way to create a disposable account — older scripts like `kernel62-browser.js` used it when it still worked and are left as historical reference, but new scripts must create disposable accounts via direct fixture-row insertion instead. See `kernel-maker-field-guide.md`'s "Two-Browser / Live-Update Verification Technique" for the current canonical method (raw `users` + `auth.sessions` row insertion, token hashed to match `sessions.HashToken`).

## Frontend Conventions (Vue / Pixi / Shared Shell)
There is no frontend build step — venues are served directly as static HTML/JS/CSS (`frontend/venues/<venue>/`), no `package.json`/bundler in the loop. Hard-refresh after any frontend edit; there's no hot-reload.

- **Vue** is used for newer, more interaction-heavy surfaces (Storyboards was rebuilt in Vue at Kernel 94 — read `frontend/venues/storyboards/` as the current reference implementation before adding Vue elsewhere).
- **PixiJS** is the stage/canvas renderer for live composition (tokens, drawing, maps). Treat it as a renderer-only layer — it does not own domain state — unless a specific kernel has explicitly proven otherwise for that surface.
- **Shared shell** (`frontend/venues/shared/venue-shell.js`, `frontend/lib/stage-runtime/`) is the common live-theater tray/runtime primitive Catharsis and First Theater both build on. Kernel 95 unifies this further — check its status in `Construction/History/Kernel Index.md` before assuming the shell has already been generalized beyond those two venues.

## Testing WebSockets
```bash
wscat -c ws://127.0.0.1:8081/ws/the-cave
```
Test both the HTTP snapshot endpoint (`/api/world/<venue>`) and the WebSocket endpoint together for any change touching live/stage behavior — they can drift independently. See `Construction/Domains/Identity/Canonical Role and Authority Resolution.md` §9 for why WebSocket/command authority should never diverge from HTTP authority for the same action.

## Domain Authority Helpers
Before writing a new permission check, check whether one of these already answers your question (duplicating one of these was the single biggest class of bug Kernel 97 found and fixed):
- `backend/internal/access` — Operator identity, Location-scoped role resolution, venue access gates.
- `backend/internal/participation` — Show Run roster resolution (Cast/Player/Crew authority within a specific Show).
- `backend/internal/showruns` — production/management authority (`CanManageShowRun`), backstage visibility.
- `backend/internal/actions` — the single authorization gate for WebSocket/command stage actions (`CanAct`).

`Construction/Domains/Identity/Canonical Role and Authority Resolution.md` is the canonical map of which function answers which authority question — read it first.

## Working Rules
Do:
- keep Postgres in Docker
- use `GOCACHE=/tmp/victory-gocache`
- verify whether `8081` is host-Go or Docker before debugging routes
- hard refresh venue pages after frontend changes
- test both HTTP and WebSocket surfaces for Cave-facing kernels
- treat PixiJS as a renderer-only experiment unless a kernel explicitly proves otherwise
- test both First Theater and Catharsis when changing their parallel stage runtime trees
- keep Show-owned durable state separate from Session-local runtime state

Do not:
- assume the backend serving `8081` is the newest process
- assume a browser `404` means the route is missing without checking the live process
- confuse Showing Review with video recording goals
