# Fresh Install / Deployment Proof

## 1. Prerequisites
- Docker and Docker Compose
- Go toolchain for `go run`
- PostgreSQL container available locally for the install proof
- Repo checked out on the target machine

## 2. Clone Repo
- `git clone <repo-url>`
- `cd victory`

## 3. Copy `.env.example` to `.env`
- `cp .env.example .env`
- Fill in only the values you actually have.

## 4. Set Database Config
- `DATABASE_URL` must point at the local install database.
- For the clean-install smoke proof, a temporary database is created inside the local `victory-postgres` container.
- Do not point the smoke proof at a live production database.

## 5. Set Discord OAuth Config
- `DISCORD_CLIENT_ID`
- `DISCORD_CLIENT_SECRET`
- `DISCORD_REDIRECT_URL`
- `DISCORD_OAUTH_SCOPES`
- `DISCORD_OAUTH_ENABLED`
- If these are blank, the backend should still boot.

## 6. Set Discord Bot / Gateway Config
- `DISCORD_APPLICATION_ID`
- `DISCORD_BOT_TOKEN`
- `DISCORD_BOT_PERMISSIONS`
- `DISCORD_BOT_REDIRECT_URL`
- `DISCORD_PUBLIC_KEY`
- `DISCORD_SERVER_LINK_ENABLED`
- `DISCORD_GATEWAY_ENABLED`
- `DISCORD_GATEWAY_INTENTS`
- `DISCORD_GATEWAY_URL`
- If these are blank, the backend should still boot and the Discord surfaces should report unavailable or disabled.

## 7. Run Migrations / Start Stack
- The backend boot path applies the kernel schema surfaces during startup.
- For a clean install proof, use the smoke script:
  - `./scripts/smoke/fresh-install.sh --local`
- That script creates a temporary database, applies the SQL migrations from `database/migrations/`, starts a throwaway backend on port `18081`, and runs the smoke checks.

## 8. Create First Login Through Discord or Local Auth
- For the proof path, a temporary local user is seeded directly into the temporary database.
- In a real deployment, Discord login or local auth can create the first user.

## 9. Run Bootstrap Producer Command
- Command:
  - `go run ./cmd/victory-bootstrap producer --handle <victory_handle>`
  - or `--discord-user-id <discord_user_id>`
  - or `--user-id <victory_user_id>`
- The command is idempotent.
- Fresh installs now default to the neutral `victory-theater` location unless `DEFAULT_LOCATION_SLUG` is set.
- The legacy `amurray-family` location remains available for compatibility when it exists.
- Discord login alone does not grant producer authority.

## 10. Verify Account Authority
- Call `GET /api/account/me` with the user session cookie.
- Confirm the response shows `authority.is_producer=true` and the expected producer locations.

## 11. Link Discord Server
- Open Producer’s Office.
- Use the Discord Server link/install controls.
- If Discord config is missing, the UI should show unavailable or not configured rather than crashing.

## 12. Repair Discord Channels
- Use the Discord channel mapping repair controls in Producer’s Office after linking the server.
- This is safe to skip when Discord config is absent.

## 13. Test `/mic`
- Open First Theater.
- Use `/mic status`, `/mic hot`, and `/mic off` in the house-mic chat flow.
- If Discord config is absent, the app should still boot and the mic control should report unavailable rather than crashing.

## 14. Test House-Mic Bridge
- With real Discord config, verify Victory chat mirrors into Discord and Discord `/mic` thread messages import back into Victory.
- With no Discord config, confirm the Discord surfaces fail safely and the backend stays up.

## 15. Common Failures
- Missing `DATABASE_URL` or an unreachable Postgres container
- Forgetting to run the fresh-install proof against the temporary database
- Expecting Discord features to work when the Discord env vars are blank
- Confusing Discord login with producer authority
- Using the live database for smoke proof work
- Assuming the bootstrap default still points at `amurray-family`

## Smoke Proof Checklist
- `GET /health`
- `GET /api/auth/providers`
- `GET /api/account/me` anonymous returns `401`
- `GET /api/discord/server-link/status` anonymous returns `401`
- `GET /api/discord/channel-mapping/status` anonymous returns `401`
- `GET /api/discord/gateway/status` should return safely even when Discord config is absent

## Live-Stack Safety
- The clean-install proof does not wipe or alter the live production database.
- It uses a temporary database name inside the local Postgres container and removes it afterward.
