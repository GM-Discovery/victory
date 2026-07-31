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
cd /opt/victory/backend
DATABASE_URL='postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory?sslmode=disable' GOCACHE=/tmp/victory-gocache go run ./cmd/victory-bootstrap producer --discord-user-id <discord_user_id>
```

If `8081` is occupied:
```bash
PORT=18081 DATABASE_URL='postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory?sslmode=disable' GOCACHE=/tmp/victory-gocache go run ./cmd/victory
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

### Dedicated test database (Kernel 64)
`go test ./...` no longer touches the live `victory` database. Set up the dedicated `victory_test` database once (safe to re-run - non-destructive):
```bash
cd /opt/victory
TEST_DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory_test?sslmode=disable" \
  scripts/test/setup-test-database.sh
```
This creates `victory_test` on the same Postgres container if it doesn't exist, applies every migration, and boots the real backend once against it (so Go-side `Ensure*Surface` bootstrap runs too - some venues, like `first-theater`, only exist because of that, not because of any SQL migration). For a full wipe-and-rebuild from empty:
```bash
CONFIRM_TEST_DB_RESET=1 \
  TEST_DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory_test?sslmode=disable" \
  scripts/test/reset-test-database.sh
```
Both scripts refuse to run against anything that isn't clearly a dedicated test database (see `scripts/test/require-isolated-database.sh`). `scripts/smoke/fresh-install.sh --local` is unrelated to this - it manages its own fully disposable database per run.

Migration replay matters: these scripts reapply all migration files and do not use a migrations ledger. A column rename/drop must be safe when the entire sequence is replayed over an already-migrated database, not only on an empty database. See the Kernel 70 operator notes.

## Browser Proof Scripts (Playwright)
Playwright is not vendored in the repo. The working install lives at `/tmp/node_modules` (browsers in `/root/.cache/ms-playwright`), so run proof scripts as:
```bash
cd /opt/victory
NODE_PATH=/tmp/node_modules node scripts/smoke/kernel62-browser.js
```
If `/tmp` has been cleared, reinstall with `npm i playwright` in a scratch dir and `npx playwright install chromium`. Scripts target the deployed site by default (`--host-resolver-rules` maps the domain to 127.0.0.1) and create clearly-named disposable accounts via real `/api/auth/signup`.

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
