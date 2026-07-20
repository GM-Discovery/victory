# Kernel 72 — Single Schema Truth, Shared Stage Engine, and Session Security Hardening

**Revision:** 0.1
**Status:** PASS — deployed live 2026-07-19 (see `Construction/OperatorLogs/kernel-72-reportback.md`)
**Primary track:** O — Operational Integrity (O3 migration and data safety, O4 authority and trust boundaries)
**Complementary tracks:** V (one reusable stage engine for every future game venue), C (deploys become one-step for concierge installs)
**Depends on:** Kernel 64 (test DB isolation), Kernel 70A/71 deploy findings (prod migration drift), Kernel 51/52 runtime module split
**Expected next consumer:** Every future kernel that ships a migration or a new game venue

---

## 0. Purpose

Three structural risks, all found in the 2026-07-18 whole-repo review, all already witnessed or trivially exploitable:

1. **Two sources of schema truth.** `database/migrations/*.sql` and eight Go `Ensure*Surface` bootstraps both create schema. The Kernel 70A deploy proved the failure mode: Kernel 70's migrations were never applied to prod and nothing noticed until a live deploy broke. Six Go files still carry ~51 DDL statements that exist nowhere in `database/migrations/`.
2. **A forked stage runtime.** `frontend/venues/first-theater/` and `frontend/venues/catharsis/` each carry a ~5,000-line `runtime.js` plus 15 `runtime/*.js` modules that are copies of each other (9 byte-identical after name normalization; 23 genuinely divergent lines, almost all newer catharsis work such as Token Aura). Every stage fix must currently be made twice; each of the 3–5 planned future games would make that worse.
3. **Session-security gaps.** Production cookies are `SameSite=None` (with `COOKIE_SECURE` defaulting to true) with no CSRF tokens and no Origin check on ~173 mutating HTTP routes — cross-site request forgery is live. Auth endpoints have no rate limiting. A dev session token sits in a tracked `cookies.txt`.

## 1. Resolved operator decisions (2026-07-18)

- **Migration model: auto-apply + auto-backup.** The backend embeds all migrations, keeps a `schema_migrations` ledger, `pg_dump`s before applying anything pending, applies at startup, and refuses to boot on checksum drift. Chosen over the cheaper "verify-only gate."
- **Stage architecture: one shared engine + thin per-game layers.** First Theater is the plain template; Catharsis is the Socio game built on top; future games (3–5 planned) each get a venue that configures the shared engine rather than copying it. Where the two forks disagree, the newer (catharsis) behavior is canonical; game-only behavior (onboarding hook, new-character flow) becomes per-venue configuration.
- **`SameSite=Lax` in secure mode.** No cross-site iframe embedding exists or is planned (Discord integration is OAuth redirect + gateway bot, not an embedded Activity).
- **Process:** full kernel with reportback. **Deploy:** live when green, with prod DB backup.
- `cookies.txt` is removed and ignored but *not* scrubbed from git history (localhost dev token; the session row is revoked instead).

## 2. Scope

### Included

**Phase A — single schema truth**
- New package `backend/internal/migrate`: `//go:embed` of `database/migrations/*.sql`, `schema_migrations(filename, checksum, applied_at)` ledger, ordered apply of pending files each in its own transaction, checksum-drift refusal, `MIGRATE_ON_BOOT=false` escape hatch (verify-only, refuse to boot on pending).
- Auto-backup: when ≥1 migration is pending **and** the database is non-empty, run `pg_dump -Fc` to `$BACKUP_DIR` (default `/opt/victory/backups`) before applying; refuse to proceed if the dump fails. Fresh/empty databases skip the backup.
- Adoption path: ledger table absent + non-empty DB → back up, re-run all migrations (all are idempotent; the Kernel 71 deploy already proved a full re-run no-ops), then stamp the ledger.
- Extraction of all DDL from the six Go bootstrap files (`profiles`, `messages`, `characters`, `showings`, `assets/warehouse`, `identity/discord_gateway`) into new migrations `049`–`054`. `Ensure*Surface` functions keep only seed/import/recompute logic (which is dynamic and belongs in Go).
- `Dockerfile`: add `postgresql16-client` to the runtime stage; `docker-compose.yml`: mount `./backups`, pass `BACKUP_DIR`.
- `scripts/test/apply-test-migrations.sh` keeps working (the runner is also what the test bootstrap boots through).

**Phase B — shared stage engine**
- New `frontend/lib/stage-runtime/`: the 15 runtime modules plus the main runtime, renamed to `VictoryStage*` globals, taking the catharsis side of every real divergence (Token Aura render path, snapshot setter, chat-presentation refresh) and first-theater's cosmetically cleaner `editors.js` variant.
- The main engine exposes `window.VictoryStageRuntime.start(config)`; per-venue config carries slug, display strings, and hooks (`onHelp`, workbook button behavior/wording).
- `first-theater/` and `catharsis/` each keep only `index.html` (script tags now pointing at `/lib/stage-runtime/`), a small venue config/entry script, and catharsis's game layer (`onboarding.js`, `chapter3-quiz-data.js`) — the per-venue `runtime.js` and `runtime/` copies are deleted.
- No behavior change on first-theater except gaining Token Aura parity (a Track V1 platform feature that had only landed on the catharsis copy).

**Phase C — session security**
- `sessions.SetSessionCookie`/`ClearSessionCookie`: `SameSite=Lax` always (drop the `None`-when-secure branch).
- Per-IP token-bucket rate limiting on `/api/auth/login`, `/api/auth/signup` (and password reauth in `PATCH /api/account/email`): client IP from `X-Forwarded-For` (backend is only reachable through bread-caddy) falling back to `RemoteAddr`; typed 429 response; in-memory, single-instance by design.
- Delete `cookies.txt`, add to `.gitignore`, revoke that session token in the live DB.

### Explicitly excluded
- CSRF tokens (unnecessary once `SameSite=Lax` holds; revisit only if cross-site embedding ever becomes a requirement).
- Git-history scrub of `cookies.txt`.
- Migration *rollback/down* files (restore path is the pre-apply `pg_dump`).
- Any new stage feature; Phase B is a pure refactor plus Token Aura parity.
- Distributed/persistent rate limiting.

## 3. Acceptance criteria

**A1.** From an empty database, booting the backend applies all migrations, stamps the ledger, and every existing test passes with no manual `apply-test-migrations` DDL step diverging from it.
**A2.** Booting against a copy of a populated database with no ledger backs up first, re-runs idempotently, stamps, and boots; a second boot applies nothing.
**A3.** Tampering with an applied migration file's content causes boot refusal with a named-file checksum error.
**A4.** With `MIGRATE_ON_BOOT=false` and a pending migration, the backend refuses to boot and names the pending files.
**A5.** `grep` proves no `CREATE TABLE|CREATE INDEX|ALTER TABLE|CREATE SCHEMA` DDL remains in non-test backend Go outside `internal/migrate`.
**B1.** Both venue pages load the shared engine; `diff` proves zero per-venue copies of engine code remain; first-theater renders Token Auras.
**B2.** The catharsis onboarding help button still calls the onboarding module via its hook; first-theater has no help button.
**B3.** A two-client websocket exchange (chat/dice/token move) works in both venues via the shared engine (HTTP/WS-level proof; browser evidence still a standing gap).
**C1.** Session cookies observed on login carry `SameSite=Lax; HttpOnly` (and `Secure` in secure mode).
**C2.** Repeated failed logins from one IP hit a typed 429; a different IP is unaffected; successful auth is unaffected at normal rates.
**C3.** `cookies.txt` is untracked, ignored, and its session row is revoked.

## 4. Rollback

- Phase A: the pre-apply dump in `backups/` restores the DB; reverting the commit restores manual-apply behavior (migrations themselves are unchanged and idempotent).
- Phase B: revert restores the per-venue copies byte-for-byte.
- Phase C: revert restores prior cookie flags; rate-limit state is in-memory only.
