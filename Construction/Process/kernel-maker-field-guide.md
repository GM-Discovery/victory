# Victory Kernel Maker Field Guide

## Purpose
This guide is for a kernel maker or lower-tier coding agent that needs to change Victory without rediscovering the same runtime traps. Read it before implementing a kernel, then keep it open while testing.

**This file predates most of Victory's history and was written when Victory was a single-table game (The Cave, Greenroom, Trailers).** Sections below describing "current product behavior" as of an early kernel are historical snapshots, not the current API surface — Victory now spans dozens of venues (Storyboards, eWrite/Writer's Room, First Theater, Catharsis, Producer's Office, Director's Chair, Third Place, Audition Hall, and more) built across 100 kernels. **Before trusting anything below as current, check:**
- [`Construction/History/Kernel Index.md`](History/Kernel%20Index.md) — the full 1-100 kernel record, what each one actually built, and where its evidence lives.
- [`Construction/History/Kernel Lineage.md`](History/Kernel%20Lineage.md) — the narrative version, organized by architectural era.
- [`Construction/History/Superseded Doctrine Map.md`](History/Superseded%20Doctrine%20Map.md) — a "do not resurrect" list of specific architecture Victory deliberately replaced (raw Session-as-product-center, `current_session_personas` as authority, unscoped Location role resolution, and more). Read this before assuming an old comment or old spec still describes real behavior.
- [`Construction/History/Current Architecture Manifest.md`](History/Current%20Architecture%20Manifest.md) — current services, packages, and runtime modes in one place.
- [`Construction/Domains/Identity/Canonical Role and Authority Resolution.md`](Identity/Canonical%20Role%20and%20Authority%20Resolution.md) — the canonical map of which function answers which authority question (Kernel 97). **Check this before writing any new permission check** — duplicating an existing authority resolver instead of reusing it was the single biggest class of bug Kernel 97 found and fixed.
- [`Construction/Process/workflow/dev-workflow.md`](workflow/dev-workflow.md) — the actively-maintained operational reference for runtime modes, ports, testing, and frontend conventions. Where this field guide's operational instructions (Docker commands, ports, test commands) conflict with `dev-workflow.md`, **trust `dev-workflow.md`** — it is updated more frequently.

Victory is a browser-based VTT and venue system, Go backend / Postgres database / static HTML+CSS+JS frontend (plus Vue for Storyboards as of Kernel 94), served by Caddy or the Go backend's routed APIs.

Current canon references:
- [current-state.md](current-state.md)
- [roadmap.md](roadmap.md)

## Repo Map
Paths below are relative to the repository root — substitute your own clone location, do not assume `/opt/victory`. This is illustrative, not exhaustive; see `Current Architecture Manifest.md` for the fuller current package list.

- `backend/` - Go service, HTTP APIs, WebSocket handlers, domain packages.
- `backend/cmd/victory/main.go` - process entry point, bootstraps kernel surfaces, registers routes.
- `backend/cmd/victory-recover/` - operator break-glass recovery CLI (Kernel 76); `backend/cmd/victory-bootstrap/` - the original, narrower operator bootstrap CLI for safe authority grants.
- `backend/internal/actions/` - append-only action types, validation, the single WebSocket/command authorization gate (`CanAct`).
- `backend/internal/network/` - WebSocket hub, presence, snapshots, live broadcasts (now serves multiple venues, not just The Cave).
- `backend/internal/access/` - Operator identity, Location-scoped role resolution, venue access gates.
- `backend/internal/participation/` - Show Run roster resolution — canonical Cast/Player/Crew authority within a specific Show (Kernel 66, hardened Kernel 97).
- `backend/internal/showruns/` - production/management authority (`CanManageShowRun`).
- `backend/internal/identity/` - users, sessions, auth (including Discord OAuth/server-link/gateway), invites, role helpers.
- `backend/internal/profiles/`, `backend/internal/playerprofile/` - Trailers/Greenroom/Third Place profile and Face surfaces.
- `backend/internal/characters/` - character cards, drafting grants, session personas (display only — see the canonical resolver doc for why this is not an authority source).
- `backend/internal/storyboards/`, `backend/internal/ewrite/`, `backend/internal/scenes/`, `backend/internal/stageobjects/` - later, larger domain packages (Storyboards/Timeline, documents, Scene composition, canonical stage-object visibility).
- `backend/migrations/` - embedded, checksummed SQL migrations. Add a new numbered file for schema changes; see `dev-workflow.md`'s "Creating a new migration".
- `frontend/venues/` - one directory per venue; check the directory listing directly rather than assuming this guide's venue list is current.
- `Construction/` - kernel specs, reports, operator notes, workflow docs, and (as of Kernel 99) the full historical archive under `Construction/History/`.

## Domain Vocabulary
- **Location** - hosted world-space. Contains lots, venues, users, memberships, libraries.
- **Lot** - collection of venues inside a location.
- **Venue** - experience surface and rules context, such as `the-cave`, `greenroom`, `trailers`.
- **Library** - reusable element ownership. Reusable assets belong here rather than inside one venue.
- **Element** - reusable object or asset. `context_class` distinguishes props, scenery, cards, and future types.
- **Placement** - instruction that places an element into a venue/session context.
- **Session** - live connection instance. Not a durable Show owner (see below) — a Session can be linked to a Show Run, but the Show Run and its roster are what carry authority forward across reconnects.
- **Show / Showing / Show Run** - Show Run is the canonical production unit (Kernel 66+); a Showing is a scheduled/ticketed event scoped to a Show Run; a Session is a live connection, not a product entity. Do not treat Session as the canonical Show owner — that model was retired (see `Superseded Doctrine Map.md`).
- **Action** - append-only session event with server-assigned ordering. Do not rewrite old actions to change history.
- **Presence** - live connection state. It is in-memory and ephemeral, confirmed **never** used as durable authority (Kernel 97, Kernel 98).
- **User identity** - accountable account identity. Never replace it with a character/persona.
- **Character card** - story persona owned by a user. It can be drafted, edited, and equipped.
- **Persona / selected Character** - the currently equipped/selected character. This governs **display** (name/portrait shown) only. **It is not a participation-authority source** — confirmed directly by Kernel 97's audit. Real Show/Showing participation authority (Cast/Player/Crew) is resolved through the Show Run roster (`backend/internal/participation`), never through session personas.
- **Producer/Director/Cast/Crew/Audience** - in-app roles. Producer/Director are normal app management authority (`CanManageShowRun`), not shell or host authority. Crew has broad backstage *visibility* but narrower *management* authority than Producer/Director — these are deliberately different questions with deliberately different answers for Crew; see the canonical resolver doc §4-5 before assuming this is a bug.
- **Operator** - infrastructure authority outside ordinary app permissions (`access.IsOperatorUser`). An Operator's client-side role *display* must always match their real server-resolved role for the identity they're using — a Kernel 97 fix removed a bug where the UI silently overrode this.

## Historical API Snapshot (Kernel 23-33 era — NOT current)
The following described "current product behavior" at the time this guide was last substantially rewritten (around Kernel 33). It is preserved as a historical snapshot only — Victory has since added dozens of venues, the full Show/Showing/Show Run model, Storyboards, eWrite, cartography, Socio, and more. **Do not treat the list below as the current API surface.** For the real current picture, see `Construction/History/Kernel-to-Feature Map.md` and `Kernel Index.md`.

Kernel 23 added character cards and Cave persona actions. Kernel 24 moved character editing into Greenroom. Kernel 27 added the closed-showing review surface in the Director's Chair. Kernel 28 added the live Director Console for current-showing control. Kernel 32 added Discord OAuth as the primary login path. Kernel 33 added the operator bootstrap command.

Snapshot-era HTTP/WebSocket/tables (frozen at ~Kernel 33, incomplete for current Victory):
- HTTP: `GET /api/auth/providers`, `/auth/discord/start`, `/auth/discord/callback`, `/api/character-cards/me`, `POST /api/character-cards`, `PATCH /api/character-cards/{id}`, `GET /api/showings`, `/api/showings/{id}/review`, `/api/director-console/current`, `POST /api/showings/{id}/audience-view`, `/api/showings/{id}/close`, `/api/showings/start`, `/api/venues/{slug}/chat-policy`.
- WebSocket: `persona/equip`, `persona/unequip`.
- Broadcast/update: `presence/update`, `showing/update`, `venue/update`.
- Tables: `character_cards`, `auth.discord_identities`, `auth.oauth_states`, `permission_grants`, `current_session_personas` (display-only, see above).

The Greenroom owns character creation and editing. The Cave should only choose an existing character and put it on or take it off.
The Director's Chair owns closed-showing review. It should read the durable action stream, not render video playback.
The Director Console owns live current-showing control. It should be utilitarian, authority-gated, and tied to the current Show Run rather than to history.

## Runtime Modes, Ports, and Rebuild/Restart
**This guide no longer duplicates operational mechanics — `Construction/Process/workflow/dev-workflow.md` is the actively-maintained source of truth for runtime modes (dev/install/Windows-consumer), Docker/Podman commands, ports, and rebuild/restart procedure.** The compose file also moved: it now lives at `packaging/podman/compose.yml`, not a repo-root `docker-compose.yml` — dev-workflow.md has the current commands. Read it before running anything below; this guide previously listed exact `docker compose` invocations that have since gone stale twice, which is exactly the drift this update is meant to stop.

If the backend behavior does not match your code, the most common cause is still a stale process — check whether a host `go run` or a container is serving the port you're testing, and hard-refresh the browser (no frontend hot-reload exists).

## Database And Migration Rules

**As of Kernel 72, `backend/migrations/` is the single source of schema truth.**
The migration files are embedded into the backend binary (`backend/migrations/embed.go`)
and applied automatically at startup by `backend/internal/migrate`, tracked in a
`schema_migrations` ledger (filename + sha256 checksum). There is no manual
apply step on deploy anymore — see `dev-workflow.md` for the current deploy command.
Before applying anything pending to a non-empty database the
runner writes a `pg_dump` backup and refuses to
migrate if the backup fails. Set `MIGRATE_ON_BOOT=false` to make the backend
verify-only (it will refuse to boot and name the pending files).
See `Construction/History/Migration Chronology.md` for the full history of every migration mapped to its originating kernel.

Schema changes are ONE path now:
- Add a new numbered SQL migration in `backend/migrations/`. Never edit a file
  that has already shipped — the runner refuses to boot on a checksum
  mismatch; add a new file instead.
- `EnsureKernelXX...Surface` bootstraps still exist but may contain **seeds
  and data repair only, never DDL**. No Go code outside
  `backend/internal/migrate` may execute `CREATE TABLE`/`ALTER TABLE`/etc.
  (Kernel 72 acceptance criterion A5; migrations 049–054 hold the DDL that
  used to live in Go.)

Inspect tables / the ledger:
```bash
docker exec -it victory-postgres psql -U victory -d victory -c '\dt'
docker exec -it victory-postgres psql -U victory -d victory -c 'SELECT filename, applied_at FROM schema_migrations ORDER BY filename DESC LIMIT 10;'
```

Rules:
- Do not destructively rewrite action history.
- Prefer additive migrations: `ADD COLUMN IF NOT EXISTS`, new tables, new indexes.
- Every migration must be idempotent — the runner executes each file without a
  wrapping transaction (several historical files carry their own
  BEGIN/COMMIT), so idempotency is what makes a partial failure safely
  retryable on the next boot.
- Keep seed bootstraps compatible with already-running databases.
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
**See `dev-workflow.md`'s "Frontend Conventions (Vue / Pixi / Shared Shell)" for the current picture** — Vue (Storyboards, Kernel 94), Pixi (stage/canvas rendering), and the shared live-theater shell (`frontend/venues/shared/venue-shell.js`, Catharsis + First Theater) are now the three real frontend approaches in use, chosen per venue based on its needs. The rules below from the pre-Pixi-maturity era are kept only where still evergreen:
- PixiJS is a client-side renderer only. Do not let Pixi state become app truth — this remains true regardless of which venue uses it.
- Avoid putting full editors into a live-session venue unless the kernel explicitly says that venue owns the workflow.
- Inline venue scripts should pass `node --check` after extraction (see below) — most newer venues avoid large inline scripts entirely, but The Cave and a few others still have them.
- "Middle School Stage", the "hidden Stage Template venue", and "the First Theater overlay proof marker" were specific extraction experiments from Kernels 30-31 and are historical — check `Construction/History/Kernel Lineage.md` Era 2 before assuming any of them are still an active convention.

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
**See `dev-workflow.md`'s "Common Checks", "Dedicated test database (Kernel 64)", and "Domain Authority Helpers" sections for current, exact commands.** Summary of what still matters: DB-touching tests require `TEST_DATABASE_URL` (hard-fail, not skip, since Kernel 64) pointed at a dedicated `*test*`-named database, never the live one; obtain pools via `backend/internal/dbtest.OpenTestPool(t)` rather than dialing `TEST_DATABASE_URL` directly, since it now also fills in `DEFAULT_LOCATION_SLUG` if unset (Kernel 96/97); always use `GOCACHE=/tmp/victory-gocache` in restricted environments; always finish with `git diff --check`. Do not set `DATABASE_URL` for test runs.

## Common Failure Modes
- **Port conflict on 8081**: Docker backend and host-Go backend are both running. Stop one or use `PORT=18081`.
- **Wrong database host**: Host-Go uses `127.0.0.1`; Docker backend uses `victory-postgres`.
- **Postgres not running**: Start `docker compose up -d postgres`.
- **Stale backend**: Code changed, but the old process is still serving. Restart the process actually bound to the port.
- **Stale static frontend**: Hard refresh the venue page. If served by Caddy, check the Caddyfile for which `frontend/` path it actually reads — don't assume a specific absolute path.
- **Forgot `GOCACHE`**: Go tests fail because cache path is not writable. Add `GOCACHE=/tmp/victory-gocache`.
- **Migration exists but bootstrap missing**: Fresh databases work, existing databases do not, or vice versa. Add both paths when the repo pattern expects both.
- **Client claims authority**: Server must resolve current user, role, session, and persona. Client payloads are requests, not facts.
- **Presence mistaken for history**: Presence is ephemeral and in-memory. Actions are the durable record.
- **Dirty worktree collisions**: Read `git status --short` before editing. Do not revert unrelated user changes.
- **`scripts/smoke/fresh-install.sh --local` leaves an orphaned backend process / undropped throwaway DB**: fixed as of Kernel 61A — the script used to background a `go run ./cmd/victory` inside a subshell and capture `$!`, which is neither `go run`'s own PID nor the actual compiled binary's PID, so `kill "$BACKEND_PID"` in the cleanup trap often missed the real process. It now builds the binary once and runs it directly, plus a `lsof`-based port-cleanup fallback. If you ever see `ss -ltnp | grep 18081` show a stray process after a run, that's this bug regressing — check the backend-start block hasn't been changed back to a backgrounded `go run`.

## Two-Browser / Live-Update Verification Technique

Established during Kernel 61/61A for proving cross-account behavior (one user views/mutates, a second user must not be affected) and websocket live-updates. Reuse this rather than inventing a new approach:

1. **Get disposable accounts via direct fixture-row insertion — this is the canonical method as of Kernel 84, not a fallback.** Production password signup is closed (`PASSWORD_SIGNUP_ENABLED=false` since Kernel 76; confirmed closed live — `password_signup_closed`, "Victory accounts are created by signing in with Discord" — when Kernel 83 tried the old `POST /api/auth/signup` approach this section used to lead with). **Do not reopen `PASSWORD_SIGNUP_ENABLED` in production merely to make a browser proof easier** — that would weaken a deliberate security control (Kernel 76) for test convenience, which is exactly backwards. Instead, insert the rows a real signup would have created:
   - `INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text` — no password/email needed for a test-only account.
   - Generate a raw session token yourself (e.g. Node: `crypto.randomBytes(32).toString("base64url")`, matching `sessions.NewRawToken`'s shape) and insert it into `auth.sessions` with `token_hash = digest(raw_token, 'sha256')` (Postgres) or an equivalent `sha256(raw_token_bytes)` computed in your own script — this must exactly match `backend/internal/sessions.HashToken`'s algorithm (SHA-256 of the raw token *string's* UTF-8 bytes, not the 32 random bytes before base64 encoding) or the cookie won't authenticate. Set a real `expires_at` (e.g. `now() + interval '3 hours'`) and a recognizable `user_agent` (e.g. `'kernel84-proof'`) so stray rows are easy to spot and clean up later.
   - The raw (unhashed) token is what you set as the `victory_session` cookie value — via Playwright's `context.addCookies` or a raw `fetch` `Cookie` header, exactly as if it had come from a real login response.
   - This is for **disposable test accounts only**, run against whichever database the proof targets (production for a live browser proof, `victory_test` for backend dbtests) — never insert a real user's credentials or reuse a real account's session this way.
   - Cleanup: delete the `auth.sessions` row, then the `users` row (and anything the kernel under test attached — grants, owned boards, etc.), the same way disposable accounts have always been cleaned up. Verify zero residue afterward (`SELECT count(*) FROM users WHERE handle LIKE 'yourprefix%'`), not just that your own delete statements ran without erroring.
2. **Never do this for Straturli** unless the task specifically requires verifying Straturli's own account — prefer any other real or fixture account, disposable or not.
3. Drive two (or more) simultaneous browser contexts with Playwright (`/tmp/node_modules/playwright`, chromium pre-installed in this environment) — one `context`/`page` per account, cookies set via `context.addCookies`. This environment has outbound internet access and can reach the real deployed site directly (e.g. `https://victory.amurray.family`).
4. For websocket-push proofs specifically: assert the *other* tab's DOM updates without calling reload — that's the actual thing under test, not just that a second `fetch` would return the new value.
5. Clean up: delete temp `auth.sessions` rows and disposable accounts; re-check the real preservation counts (`users`, `location_memberships`, `access_grants`, `character_cards`, Straturli's UUID) haven't shifted for reasons other than your own cleanup.

## Kernel 61 / 61A Notes
For the Player Workbook / Trailer Face system:
- See `operator-notes.md`'s "Kernel 61 / 61A" section for the Account vs Player Workbook vs Trailer Face vs Character Workbook distinction — these are easy to conflate and have completely separate schemas.
- The catalogue (`backend/internal/playerprofile/catalogues/player-profile-v1.0.0.json`) is versioned data, go:embed'd — a new catalogue revision adds a new file and bumps the version, never edits v1.0.0 in place.
- `/ws/player-profile` is a separate, simpler websocket endpoint from `/ws/the-cave` / `/ws/catharsis` — it has no venue-session concept, just auth + a `watch_profile` subscribe message. Don't try to route player-profile invalidation through `ServeVenueWS`.
- `performer_profiles` (the pre-Kernel-61 table) is permanently read-only now — all 6 `/api/profiles/*` routes return `410 Gone`. Do not resurrect writes to it; extend the Player Workbook model instead.

## Kernel 62 Notes
For the private player-relationship layer (My People):
- See `operator-notes.md`'s "Kernel 62" section for the directional model, subject-invisibility rule, vocabularies, archive semantics, and shared-context limits.
- `backend/internal/playerrelationships/` deliberately mirrors `playerprofile` — if you need to change the events→facts engine in one, check whether the other has the same issue.
- **Non-owner access to any `/api/player-relationships/...` route must stay a 404, never a 403** — a 403 would confirm a private record exists. Any new subroute must go through `loadRelationshipOwned` first.
- Follow-ups must never grow reminders/notifications, and no route may let a user enumerate or count records *about* them — both are hard spec rules, not missing features.
- The Kernel 62 two-user privacy proof is scripted: `NODE_PATH=/tmp/node_modules node scripts/smoke/kernel62-browser.js` (three disposable signup accounts, 404 + DOM-scan assertions, desktop+mobile screenshots into `Construction/OperatorLogs/evidence/kernel-62/`). Extend that script for future privacy-sensitive kernels rather than hand-driving two browsers.

## Kernel 64 Notes
For DB test isolation and the live-DB safety gate:
- **`go test ./...` now requires `TEST_DATABASE_URL`** for the three DB-touching packages (`internal/identity`, `internal/network`, `internal/assets`) - see "Backend Test Commands" above. Before Kernel 64, these tests connected to a hardcoded live-looking URL directly in the test source; that hardcoding is gone.
- The safety gate is duplicated in two languages on purpose: `backend/internal/dbtest.ValidateTestDatabaseURL` (Go, enforced per test binary) and `scripts/test/require-isolated-database.sh`'s `require_isolated_database` function (shell, used by the setup/reset scripts). Keep the rules in sync if either changes - a database name must contain `test`, must not be empty/`victory`/`postgres`, must not look production-y, and must not equal `DATABASE_URL`.
- The dedicated test database (`victory_test` by convention) lives on the **same** Postgres server as the live app database - there's only one Postgres instance in this deployment. The discriminator is the database *name*, not the host. An earlier, unused version of `require-isolated-database.sh` rejected `127.0.0.1`/`victory-postgres` hosts outright, which would have made this impossible; that check is gone.
- Raw SQL migrations alone do not fully seed a fresh database - several venues (`first-theater`, `catharsis`, `middle-school-stage`, `warehouse`, `workshop`, etc.) only exist because `internal/access.EnsureKernel16VenueSurface` (and the other `Ensure*Surface` functions `cmd/victory/main.go` runs on every real boot) create them in Go, not SQL. `scripts/test/setup-test-database.sh` and `reset-test-database.sh` both build and briefly boot the real `victory` binary against the test database for exactly this reason - don't "simplify" that away or DB-touching tests that assume a real venue exists will fail against a freshly migrated database even though they pass against the long-lived live one.
- Fixed three test fixtures that only worked by accident against the live database's organic (non-migration-tracked) state: `internal/assets/warehouse_test.go`'s stats test previously passed a filesystem path where a location UUID belonged (always failed the `::uuid` cast, regardless of database); `internal/network/discord_chat_bridge_test.go` assumed a `first-theater` venue and a production already existed for `amurray-family` on whatever database it ran against - both are now created by the fixture itself (using the `the-cave` venue, which is genuinely migration-seeded, and an idempotent `ON CONFLICT DO NOTHING` production insert).
- `scripts/smoke/fresh-install.sh --local` is intentionally untouched by any of this - it manages its own fully disposable `victory_fresh_*` database per run and never reads `TEST_DATABASE_URL`. Keep it that way; don't route it through the new safety gate.

## Kernel 65 Notes
For Third Place / Headshot Commons:
- `third-place` (the venue) is seeded by a plain SQL migration (`038_kernel65_third_place.sql`), the same idempotent pattern as `trailers`/`greenroom` in `007_kernel9_profiles_greenroom_trailers.sql` - unlike `first-theater`/`catharsis`/etc. (Kernel 64 Notes above), it does **not** need a Go-side `Ensure*Surface` bootstrap step. Not every venue needs one; check whether a migration already creates what you need before assuming you have to add Go bootstrap.
- Map-tile visibility for a new authenticated-but-not-producer-only venue: don't invent a new rule. `internal/access/visibility.go`'s `ResolveVisibleVenues` already has a `performer_surface` UNION arm for `trailers`; Third Place just added its slug to the same `IN (...)` list. If a future kernel needs "same visibility as Trailers," check that function first.
- `internal/thirdplace` is the reference example for building a live-projected social feature on top of Kernel 61A/62: it imports `playerprofile.ProjectTrailerFace` and `playerrelationships.GetRelationshipBySubjectProfile` directly rather than duplicating either, and stores zero Face content of its own - the table literally has no column that could hold it. If a future kernel is tempted to cache/snapshot Face data "for performance," re-read Kernel 65's spec §3.3 first; it was explicitly rejected.
- "One active row per user, enforced by a partial unique index the upsert targets directly" (`uq_third_place_headshots_active_user`) plus the `(xmax = 0)` `RETURNING` trick to detect insert-vs-update in one round trip is a reusable pattern - reach for it before writing a read-then-write existence check anywhere else in the codebase.
- The live-update mechanism (`frontend/lib/player-profile-ws.js`'s `watchPlayerProfile`) is generic and reusable - Third Place's grid just opens one watcher per visible card. No backend change was needed to reuse it for a second UI surface beyond Trailer viewing.

## Kernel 66–70 Production And Stage Notes

- Canonical hierarchy: `Location → Production → Show Run → Show → Show Scene Placement`; Sessions are runtime windows and Showings are live/review wrappers.
- Scheduling belongs to `shows`, not the older `showings` table.
- Scene scope changed in Kernel 70: `scenes.location_id` is ownership scope and `source_production_id` is optional provenance.
- The Show owns persistent stage state. World snapshots fold Session actions plus linked Show actions while preserving identical behavior for an unlinked Session.
- Audience responses must be curated. Before adding a JSON-tagged field to a shared struct, inspect every handler that serializes it and the weakest authority gate among them.
- Cue GO requires an idempotency key. Ordered actions are fail-stop and commit independently. Implemented actions are `go_to_scene`, `emit_game_event`, and `set_show_variable`.
- Object reveal/hide/interaction Cues remain blocked on a canonical visual Scene-object mapping.
- `frontend/venues/{first-theater,catharsis}/runtime.js` is the dynamically loaded entry point for each sibling `runtime/` module tree. Check both venues and their Node tests before classifying code as unused.
- Column renames must be replay-tested against all earlier migrations; `CREATE INDEX IF NOT EXISTS` still resolves stale column references.

## Kernel Implementation Checklist
1. Read `Construction/OperatorLogs/operator-notes.md` and the newest relevant kernel docs — check `Construction/History/Kernel Index.md` for what already exists in this area before assuming it doesn't.
2. Check `git status --short`.
3. Locate the existing package and route pattern before inventing a new one; check `Construction/Domains/Identity/Canonical Role and Authority Resolution.md` before writing any new permission check.
3a. Check `Construction/History/Superseded Doctrine Map.md` before resurrecting an assumption an old comment or old spec implies — several specific ones (raw Session-as-product-center, unscoped Location role, `current_session_personas` as authority) are confirmed retired.
4. Identify database, API, WebSocket, snapshot, and frontend surfaces.
5. Add additive migrations and bootstrap SQL if schema changes.
6. Keep authority checks server-side.
7. Preserve append-only actions.
8. Update snapshots and live broadcasts together when a live state changes.
9. Keep The Cave live-session focused; move drafting/admin workflows to the right venue.
10. Run Go tests with `GOCACHE=/tmp/victory-gocache` and `TEST_DATABASE_URL` pointed at a dedicated test database for the full suite or any DB-backed package (see "Backend Test Commands" above) - never the live app database.
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
