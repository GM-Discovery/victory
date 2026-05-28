# Kernel 33 Reportback

## Status
Complete for the operator bootstrap + canon capture slice.

Kernel 33 adds a safe operator CLI for producer grants and folds Kernel 32 plus the live-site follow-on work into the project canon.

## What Was Built
- Added a new operator bootstrap CLI at `backend/cmd/victory-bootstrap/main.go`.
- Added shared bootstrap helpers in `backend/internal/identity/bootstrap.go`.
- Added tests in `backend/internal/identity/bootstrap_test.go`.
- The bootstrap command supports:
  - `go run ./cmd/victory-bootstrap producer --discord-user-id <discord_user_id>`
  - `go run ./cmd/victory-bootstrap producer --user-id <victory_user_id>`
  - `go run ./cmd/victory-bootstrap producer --handle <victory_handle>`
- The command grants only `producer` at the default location scope (`amurray-family` unless `--location` is supplied).
- The command is idempotent and reports whether the membership already existed.
- Unknown Discord user IDs fail clearly instead of creating a grant.

Canon / docs updates:
- Updated `Construction/current-state.md`
- Updated `Construction/roadmap.md`
- Updated `Construction/kernel-maker-field-guide.md`
- Updated `Construction/OperatorLogs/operator-log.md`
- Updated `Construction/OperatorLogs/operator-notes.md`
- Updated `Construction/OperatorLogs/Security-notes.md`
- Updated `Construction/vendor-acknowledgements.md`
- Updated `Construction/workflow/dev-workflow.md`

Kernel 32 and follow-on live-site work captured in canon:
- Discord OAuth routes still exist:
  - `GET /auth/discord/start`
  - `GET /auth/discord/callback`
  - `GET /api/auth/discord/start`
  - `GET /api/auth/discord/callback`
  - `GET /api/auth/providers`
- Legal pages still exist:
  - `/legal/terms/`
  - `/legal/privacy/`
- Favicon asset still exists at `frontend/assets/favicon.png`
- Favicon wiring remains on the main site and venue shells
- Victory Theater map icon still uses the favicon asset
- `/auth/*` live proxy routing is still fixed
- Docker/env wiring for Discord still exists
- `.env` remains local and gitignored

## Evidence
Backend tests passed:
- `cd /opt/victory/backend && GOCACHE=/tmp/victory-gocache go test ./...`

Safe failure proof:
- `go run ./cmd/victory-bootstrap producer --discord-user-id definitely-not-real`
- Result: `victory-bootstrap: no Victory user linked to Discord user ID "definitely-not-real"`

Real grant proof:
- A Discord-linked Victory user exists with Discord user ID `694402257331945562`
- First grant run:
  - `go run ./cmd/victory-bootstrap producer --discord-user-id 694402257331945562`
  - Result: `Already existed: false`
- Repeat run:
  - `go run ./cmd/victory-bootstrap producer --discord-user-id 694402257331945562`
  - Result: `Already existed: true`
- Database verification showed exactly one active producer membership row for that user.

Surface verification:
- `GET /api/auth/providers` returns `{"ok":true,"data":{"discord_enabled":true,"discord_login_url":"/auth/discord/start"}}` through the live proxy
- `GET /auth/discord/start` still redirects through the live proxy to Discord OAuth
- Live legal pages returned `200 OK` through the proxy
- `frontend/assets/favicon.png` exists

Diff hygiene:
- `cd /opt/victory && git diff --check`

No new dependency was added.

## How To Run
Grant producer from a Discord-linked Victory user:

```bash
cd /opt/victory/backend
DATABASE_URL='postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable' \
GOCACHE=/tmp/victory-gocache \
go run ./cmd/victory-bootstrap producer --discord-user-id <discord_user_id>
```

Other supported forms:

```bash
go run ./cmd/victory-bootstrap producer --user-id <victory_user_id>
go run ./cmd/victory-bootstrap producer --handle <victory_handle>
```

## Operator Notes
- Discord OAuth still authenticates identity only.
- Victory still authorizes users through its own sessions and memberships.
- Producer authority still comes from Victory, not from Discord login.
- The bootstrap command is operator/server context only; it is not a public HTTP endpoint.
- The bootstrap command uses the same `location_memberships` authority path that the app already trusts for producer access.

## Blockers & Workarounds
- Running the bootstrap command inside the sandbox hit the local TCP fence on the first attempt, so I reran it with escalated approval to capture the real operator-facing failure and success messages.
- Live Discord OAuth still depends on the correct redirect URI being registered in the Discord Developer Portal.

## Deviations From Kernel
- None beyond the allowed CLI shape choice. I used a dedicated `victory-bootstrap` command because that was the smallest clean operator surface in this repo.

## Known Issues
- Browsers cache favicons aggressively, so a hard refresh may still be needed before the tab icon updates everywhere.
- The live Discord OAuth path still depends on the Discord app being configured with the right redirect URI.

## Next Recommended Step
- Use the bootstrap CLI to grant producer to the Discord-linked account you actually want to operate with, then continue to the next identity/authority kernel.

## Files Changed / Created
- `backend/cmd/victory-bootstrap/main.go`
- `backend/internal/identity/bootstrap.go`
- `backend/internal/identity/bootstrap_test.go`
- `Construction/current-state.md`
- `Construction/roadmap.md`
- `Construction/kernel-maker-field-guide.md`
- `Construction/OperatorLogs/operator-log.md`
- `Construction/OperatorLogs/operator-notes.md`
- `Construction/OperatorLogs/Security-notes.md`
- `Construction/vendor-acknowledgements.md`
- `Construction/workflow/dev-workflow.md`
- `Construction/OperatorLogs/kernel-33-reportback.md`
