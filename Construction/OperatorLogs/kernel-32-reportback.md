# Kernel 32 Reportback

## Status
Complete for the Discord OAuth primary-login slice.

Kernel 32 now adds Discord OAuth start/callback handling, Discord identity linking, and normal Victory session creation while keeping Victory authorization intact.

## What Was Built
- Added Discord OAuth config plumbing in `backend/cmd/victory/main.go`.
- Added routes:
  - `GET /auth/discord/start`
  - `GET /auth/discord/callback`
  - API aliases:
    - `GET /api/auth/discord/start`
    - `GET /api/auth/discord/callback`
  - provider discovery:
    - `GET /api/auth/providers`
- Added `backend/internal/identity/discord_oauth.go` for:
  - auth URL generation
  - state generation and hashing
  - token exchange
  - Discord `/users/@me` fetch
  - user linking/creation
  - Victory session creation through the existing session model
- Added `backend/internal/identity/discord_oauth_test.go` for:
  - config detection
  - safe-disabled route behavior
  - authorize URL shape
  - token/user helper behavior
  - state consumption and reuse rejection against live Postgres
- Added `backend/cmd/victory/main_test.go` for env/scoping helpers.
- Added `database/migrations/018_kernel32_discord_oauth.sql`.
- Added a Discord login button to `frontend/login/index.html`.
- Updated the canon/docs files to reflect Kernel 32 as Discord OAuth instead of template extraction.

## Evidence
- Backend suite passed:
  - `cd /opt/victory/backend && GOCACHE=/tmp/victory-gocache go test ./...`
- Discord OAuth identity integration passed against local Postgres with mocked Discord endpoints:
  - `cd /opt/victory/backend && GOCACHE=/tmp/victory-gocache go test ./internal/identity -run 'TestDiscordOAuth|TestHashOAuthTokenDeterministic' -v`
- Frontend inline login script syntax passed:
  - `node --check` on the extracted `frontend/login/index.html` script
- Diff hygiene passed:
  - `cd /opt/victory && git diff --check`
- Unconfigured-mode safety is covered by handler tests:
  - `HandleDiscordOAuthStart(nil, DiscordOAuthConfig{})` returns `503`
  - `HandleDiscordOAuthCallback(nil, DiscordOAuthConfig{}, false)` returns `503`

Live-credentials note:
- I did not complete a browser round-trip against a real Discord Developer Portal app in this environment because no live Discord client credentials were available here.
- The callback path was still proven end-to-end enough to verify state consumption and reuse rejection with mocked Discord endpoints and real Postgres.

## How To Run
1. Set or omit the Discord env vars:
   - `DISCORD_CLIENT_ID`
   - `DISCORD_CLIENT_SECRET`
   - `DISCORD_REDIRECT_URL`
   - optional `DISCORD_OAUTH_SCOPES` (defaults to `identify email`)
   - optional `DISCORD_OAUTH_ENABLED`
2. Start Victory in the usual dev mode:
   - `cd /opt/victory/backend`
   - `PORT=8081 DATABASE_URL='postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable' GOCACHE=/tmp/victory-gocache go run ./cmd/victory`
3. Visit `/login/` and choose `Log in with Discord`.
4. For local callback testing, use a redirect URL that points back to the same backend host:
   - `http://127.0.0.1:8081/auth/discord/callback`

## Operator Notes
- Discord OAuth authenticates identity only.
- Victory still owns users, sessions, memberships, roles, and venue permissions.
- Discord login does not confer producer/director authority.
- Existing local handle/password login remains available.
- Existing session cookies still use `victory_session`.
- The backend still boots safely when Discord OAuth is not configured.

## Blockers & Workarounds
- No live Discord app credentials were available for a real browser approval flow in this turn.
- The integration test uses mocked Discord HTTP responses and real Postgres to verify the callback/state logic without exposing secrets.

## Deviations From Kernel
- No Discord bot, guild linking, channel creation, voice integration, or role mapping was added.
- No OAuth tokens are stored after `/users/@me` is fetched.
- No new third-party OAuth library was added; the Go standard library handled the flow.

## Known Issues
- Live configured-browser verification still needs a real Discord app and redirect URL from the operator side.
- The new integration test self-skips if the local Postgres socket is unavailable.

## Next Recommended Step
- Wire the account display/login header to surface Discord identity more explicitly if needed, then move on to the Discord bot/guild kernel later.

## Files Changed / Created
- `backend/cmd/victory/main.go`
- `backend/cmd/victory/main_test.go`
- `backend/internal/identity/discord_oauth.go`
- `backend/internal/identity/discord_oauth_test.go`
- `database/migrations/018_kernel32_discord_oauth.sql`
- `frontend/login/index.html`
- `Construction/current-state.md`
- `Construction/roadmap.md`
- `Construction/kernel-maker-field-guide.md`
- `Construction/OperatorLogs/operator-log.md`
- `Construction/OperatorLogs/operator-notes.md`
- `Construction/OperatorLogs/Security-notes.md`
- `Construction/vendor-acknowledgements.md`
- `Construction/workflow/dev-workflow.md`
