# Kernel 72 Reportback — Single Schema Truth, Shared Stage Engine, Session Security

## Status

**PASS. Deployed live 2026-07-19.** All three structural risks from the 2026-07-18 whole-repo review are closed: (A) migrations are now embedded in the backend binary and auto-applied at boot against a checksummed `schema_migrations` ledger with a pre-apply `pg_dump`; (B) the ~5,000-line First Theater / Catharsis runtime fork is collapsed into one shared stage engine at `frontend/lib/stage-runtime/` with thin per-venue configs; (C) session cookies are `SameSite=Lax` in every mode, credential endpoints are per-IP rate limited, and the tracked `cookies.txt` (which held a still-valid session token) is removed, ignored, and its session revoked in the live DB.

Everything is uncommitted for operator review, per project practice (Kernels 62, 65–71).

## Phase A — single schema truth

- `database/migrations/` moved (git mv) to **`backend/migrations/`** so `go:embed` can reach it inside the backend module/build context. `backend/migrations/embed.go` embeds every `*.sql`; `backend/internal/migrate` applies pending files in filename order at startup, recording filename + sha256 in `schema_migrations`.
- Refusal behaviors: checksum drift on an applied file names the file and refuses boot; a recorded-but-missing file refuses boot; `MIGRATE_ON_BOOT=false` verifies only and refuses boot naming pending files.
- Auto-backup: ≥1 pending + non-empty DB → `pg_dump --format=custom` to `BACKUP_DIR` (default `/opt/victory/backups`, now volume-mounted in compose) **before** applying; a failed dump aborts the boot. Fresh/empty DBs skip it. `MIGRATE_DANGEROUSLY_SKIP_BACKUP=1` exists for throwaway databases only and is hard-guarded: it refuses any database whose name lacks `test`/`fresh` (`requireDisposableDatabase`, mirroring the Kernel 64 dbtest gate).
- Files execute without a wrapping transaction (several historical files carry their own BEGIN/COMMIT); repo-wide idempotency is what makes partial failure retryable. The Kernel 71 deploy had already proven a full idempotent re-run.
- **All DDL extracted out of Go.** Migrations `049`–`054` hold, verbatim, the DDL that lived in `profiles`, `messages`, `showings`, `characters`, `assets/warehouse`, and `identity/discord_gateway` bootstraps. `EnsureKernel11MessagesSurface`/`22Showing`/`23Character` were pure DDL and are deleted outright (with their `main.go` calls); the other three keep only seeds (venue rows, warehouse settings, `discord_bridge` user — the seed the 14 discord test fixtures call). Acceptance grep: zero `CREATE TABLE|ALTER TABLE|CREATE INDEX|CREATE SCHEMA` in non-test Go outside `internal/migrate`.
- `backend/Dockerfile` runtime stage adds `postgresql16-client` (matches the postgres:16 server). Compose passes `MIGRATE_ON_BOOT`/`BACKUP_DIR` and mounts `./backups`.
- Scripts/docs updated to the new path: `lib-migrate-and-bootstrap.sh`, `setup-test-database.sh`, `fresh-install.sh`, `filepaths.md`, field guide (whose "Database And Migration Rules" section is rewritten around the runner — deploys are now just `docker compose up -d --build backend`).

## Phase B — shared stage engine

- Pre-work diff proved the fork was 9/15 modules byte-identical (after name normalization) with only **23 genuinely divergent runtime.js lines** — almost all newer Catharsis work. New `frontend/lib/stage-runtime/` (16 files, `VictoryStage*` globals) takes the Catharsis side everywhere it mattered (Token Aura rendering, snapshot setter, chat-presentation refresh) and First Theater's cosmetically cleaner `editors.js`.
- Venue config: each page defines `window.VictoryStageVenue` (`slug`, `name`, optional `workbookOpenLabel`, `onCardEditor`, `onHelp`) in a small `venue.js` **before** the engine loads. Modules read the slug at call time via `globalThis` (Node-test friendly). API URLs, WS path, payload `venue_slug`, tray element ids (`<slug>-dice-tray`/`<slug>-audio-tray`), and pixi-loader dedup markers are all config-driven; module-level status strings went venue-neutral ("the stage") since modules have no display-name config.
- First Theater = template (defaults: Greenroom card-editor navigation, "Open the workbook"); Catharsis = game layer (`new_character=1` flow, onboarding help hook, `onboarding.js` + `chapter3-quiz-data.js` untouched). **A future game venue is now: one `venue.js` + its game scripts.** First Theater gains Token Aura parity as a side effect (it's a Track V1 platform feature that had only landed on the Catharsis copy).
- Both `index.html` pages rewired to `/lib/stage-runtime/*?v=kernel72-1`; the old `runtime.js` + `runtime/` trees (both venues) are deleted.
- Tests: the mirrored `tests/first-theater/` + `tests/catharsis/` suites merged into **`tests/stage-runtime/`** (paths, `VictoryStage` globals, venue-config pin, neutral-wording expectations). `tests/contract/scene-nodes.contract.test.js` — which existed to prove the two forks stayed aligned — now asserts the same contract once against the real shared module. `alpha-gate.sh` updated; the tracked non-blocking dice.test.js exception halves from 18 titles to 9 (same nine tests, no longer mirrored).

## Phase C — session security

- `sessions.SetSessionCookie`/`ClearSessionCookie`: **`SameSite=Lax` unconditionally** (was `None` whenever `secure`, i.e. in production). With no CSRF tokens anywhere, `None` left every mutating route forgeable cross-site. Nothing embeds Victory in a cross-site iframe (Discord is OAuth redirect + gateway bot), so Lax closes CSRF outright; the code comment says what to do if embedding ever becomes real.
- New `backend/internal/ratelimit`: in-memory per-client-IP token bucket (burst 10, refill 10/min), client IP from first `X-Forwarded-For` hop (backend is only reachable through bread-caddy) with `RemoteAddr` fallback, typed `429 rate_limited` JSON + `Retry-After`. Applied to exactly the credential-guess endpoints: `/api/auth/signup`, `/api/auth/login`, `/api/auth/password-reset/request`, `/api/auth/password-reset/confirm`, `/api/account/email` (password reauth). Ordinary API routes deliberately unthrottled.
- `cookies.txt` git-rm'd and gitignored; its session row (sha256 of the leaked raw token) revoked in the live DB (`UPDATE 1` — it was still unrevoked). Git history not scrubbed (operator decision; localhost dev token, now dead).

## Verification evidence

- **Full Go suite green** (`go test ./...` with `TEST_DATABASE_URL` → victory_test), including new `internal/migrate` (embed/sort/checksum, drift refusal, missing-file refusal, pending-order, full adoption→no-op→tamper→repair lifecycle against the real test DB, disposable-name guard) and `internal/ratelimit` (burst/refusal/refill, per-client isolation, XFF resolution, typed 429).
- **Fresh-DB boot (A1)**: empty `victory_fresh_test` → binary boot applied all 55, `/health` OK; second boot "schema current", applies nothing.
- **Backup refusal**: host boot with 1 pending and no `pg_dump` on PATH → "pre-apply backup failed, refusing to migrate" and no schema change — exactly the fail-closed design.
- **Node suites**: 60 tests, 51 pass, only the 9 tracked `dice tray` titles fail (pre-existing at HEAD — re-proven by running HEAD's own test+module pair from git), zero unexpected.
- **Full `alpha-gate.sh`: all automated steps PASS** (with the tracked dice exception), including the fresh-install smoke end-to-end.
- **C1/C2 live-behavior proof** (scratch boot, `COOKIE_SECURE=true`): signup `Set-Cookie: … HttpOnly; Secure; SameSite=Lax`; 10 bad logins from one forwarded IP then `429` on the 11th while a different IP logs in `200` with the right password.
- **Engine syntax**: all 16 engine files pass `node --check`; a fail-loud transform guaranteed no venue-literal remnants.

## Live deployment (2026-07-19)

Belt-and-suspenders manual dump `backups/victory_pre_kernel72_manual_20260719_163330.dump`, then `docker compose up -d --build backend`. First boot took the adoption path exactly as designed: ledger absent → 55 pending named in the log → **the runner's own in-container backup** `victory_pre_migrate_20260719_163504_55pending.dump` (3.1 MB, size-consistent with the manual dump) → idempotent re-run → 55 ledger rows confirmed in prod. Post-deploy live checks: `/api/auth/providers` 200; login 401 on bad creds then **429 on the 11th attempt** (limiter live); catharsis page and `/lib/stage-runtime/runtime.js` both 200 via `victory.amurray.family`; Discord gateway reconnected clean. Note for future deploy checks: `/health` is not proxied by Caddy (only `/api/*`, `/ws/*`, `/auth/*`) — check it from inside the Docker network or use an `/api/*` route.

## Known limitations

- No browser/screenshot evidence (standing gap since Kernel 65 — no browser automation in this environment). The engine de-fork is exactly the kind of change Playwright smoke would de-risk further; strongly recommended as its own kernel.
- A successful live login's `SameSite=Lax` header wasn't observed on prod (no credentials available to this session); the identical code path was proven on the scratch boot in secure mode.
- The 9 dice.test.js failures remain a tracked, non-blocking exception (now listed once, not twice).
- Rate-limit state is in-memory and single-instance by design; restarting the backend resets buckets.
- Module-level stage status strings are now venue-neutral ("the stage" instead of "the Catharsis stage") — a deliberate, minor user-visible wording change.
- The old `victory-bootstrap` cmd and `middle-school-stage` venue page were not touched (out of scope; middle-school-stage has no runtime fork).

## Post-deploy operator report triage (2026-07-19, deployed same day)

> **Superseded/consolidated:** the fixes below, plus the venue-capability-flag refactor that replaced the interim hardcoded-allowlist hotfix, are written up as **Kernel 72A** — see `kernel-72A-reportback.md`. The sections below are kept as the original diagnosis record.

The operator reported four live issues on the First Theater stage plus a warehouse upload failure. Diagnosis from the backend request log + live DB:

1. **Root cause of the FT stage failures** (renderer fallback, "Snapshot refresh failed: no rows in result set", "socket unavailable"): `first-theater` has had **no rehearsal/live Session since Kernel 70A closed the legacy live stages** (catharsis and the-cave both have one — verified in the live DB). The engine's `startStageRuntime` treated a failed venue join as a fatal boot error, so the snapshot, websocket, Pixi init, and map-editor wiring all died together. **Not a Kernel 72 regression** — the pre-fork First Theater runtime had the identical join-first bootstrap, and the catharsis page (same shared engine) worked throughout. Fixes:
   - `identity.JoinVenue` now returns typed `no_active_session` on the empty-session lookup instead of leaking the raw pgx error.
   - The engine tolerates exactly that typed refusal: it skips the socket, keeps the stage read-only ("No Show is on stage here yet…"), still fetches the world snapshot (whose Kernel 70A `theater_context` explains the idle state), and still initializes Pixi and the editors. Any other join failure still aborts loudly.
   - Scratch-boot proof (fresh DB, operator account): FT join → `{"error":"no_active_session"}`; FT world snapshot → `ok:true` with `theater_context.kind: backstage`; the stage becomes fully live again the moment a Show Session is started from Stage Management.
2. **"Rehearsal available — open Stage Management" visibility**: was gated only on the venue-config flag, so every viewer saw it. Now additionally gated on `theater_context.kind === "backstage"` (the server-computed Director/Producer/Operator/Crew classification — no client-side role re-derivation). Known nuance: a Director who is *also* a registered player classifies as `participant` and won't see the banner; they still have Stage Management in nav.
3. **Map upload "saved but didn't mount" + token `warehouse_physical_reserve_exceeded`**: one root cause — the asset upload itself was being refused (five ~3 s `POST /api/workshop/assets` attempts in the log), so there was never an asset to mount; the editor preview is a local-file preview. The refusal was correct behavior: the 8 GiB `WarehousePhysicalReserveBytes` exceeded the host's 6.6 GiB free disk. Freed ~4.3 GB of Docker build cache + dangling images (`docker builder prune -af`, `docker image prune -f`); ~10 GB free post-deploy → ~2 GB upload headroom. **Operator decisions available for more headroom**: unused `microscope-app` (1.63 GB) and `mysql:8.0.27` (684 MB) images belong to other projects and were left alone; alternatively the 8 GiB reserve constant is very conservative for a 38 GB disk.
4. **"Ping → socket unavailable"**: nothing was removed — the socket never opened because the bootstrap aborted (issue 1). Post-fix, the socket is deliberately not opened for a session-less venue and connects normally once a Session exists.

Verification: full Go suite green post-fix; Node suites unchanged (51/60, the 9 tracked dice titles); `node --check` clean on both touched engine files; backend rebuilt and redeployed — second-deploy boot logged `migrate: schema current (55 migrations recorded)`, the runner's designed no-op path.

### Second operator report (same day): map-save delay + "cannot add tokens"

- **Map save**: worked; the perceived delay is the `POST /api/workshop/assets` image-processing step (~3–4 s per upload in the log — resize/thumbnail generation) plus Pixi fetching/decoding the full map texture after the websocket `venue_map_updated` push. The map save itself returns in ~15 ms. No defect found; if the delay matters, the fix is async thumbnailing/progress UI, not a bug fix.
- **Token creation on catharsis: real pre-existing authority gap.** Log showed `create token denied … reason=unknown_target`. `actions.isSingleVenueLegacySlug` — the gate for the whole element-action surface (tokens, reveal, remove, duplicate, index cards) — only allowed `the-cave` and `first-theater`. Kernel 70A had extended it additively for First Theater; **catharsis shipped the full stage toolset but was never added**, so every element action there was denied. Fixed additively in four places that must move together: `isSingleVenueLegacySlug` (+`catharsis`), the venue-access SQL in `authority.go`, and the two `v.slug = 'the-cave'` filters in `indexcard.go` (placement-row insert and scope resolution — widening only the authority check would have created index cards with no tray placement row, i.e. invisible cards). Full Go suite green; backend redeployed (boot: `schema current`, no-op).
- Follow-up worth its own kernel: replace the hardcoded slug allowlists (`isSingleVenueLegacySlug`, `session_control.go`'s list) with a venue-config capability flag, per `rehearsal_capability.go`'s own stated precedent — required anyway before the planned 3–5 new game venues.

## Files changed (summary)

- New: `backend/migrations/embed.go`, `backend/internal/migrate/` (+tests), `backend/internal/ratelimit/` (+tests), `frontend/lib/stage-runtime/` (16 files), `frontend/venues/{catharsis,first-theater}/venue.js`, migrations `049`–`054`, this reportback, `Construction/Kernels/kernel-72-schema-truth-stage-engine-security-v0.1.md`.
- Moved: `database/migrations/` → `backend/migrations/`; `tests/catharsis/` → `tests/stage-runtime/`.
- Deleted: both venue `runtime.js` + `runtime/` trees, `tests/first-theater/`, `cookies.txt`, three pure-DDL `Ensure*` functions.
- Modified: `main.go` (migrate wiring, limiter, dead bootstrap calls removed), `sessions/session.go`, the three seed-only bootstraps, both venue `index.html`, `scene-nodes.contract.test.js`, `alpha-gate.sh`, `lib-migrate-and-bootstrap.sh`, `setup-test-database.sh`, `fresh-install.sh`, `Dockerfile`, `docker-compose.yml`, `.gitignore`, field guide, `filepaths.md`.
