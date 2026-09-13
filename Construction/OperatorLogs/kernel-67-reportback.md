# Kernel Report Back — Kernel 67: Show Instance Model and Show Run Bridge

## 1. Status

**PASS.** Deployed live: migration 040 applied to the live `victory` database, backend
rebuilt and restarted, `/health` OK. Uncommitted, per the same operator convention used
for kernels 62–66 (the operator reviews before committing).

## 2. Existing `showings`/`sessions` audit summary

Before writing any code, the live repository was audited against the draft spec's own
§2 checklist:

- `sessions` (`database/migrations/001_init.sql:146-152`): `id, venue_id, status
  session_status, started_at, ended_at`. No FK to any higher-level container — only
  `venue_id`.
- `showings` (`database/migrations/015_kernel22_showings.sql`,
  `backend/internal/showings/showings.go`): `session_id UUID NOT NULL UNIQUE REFERENCES
  sessions(id) ON DELETE CASCADE`, lazily created on first action-write via
  `EnsureForSession` (not a DB trigger — a session can exist with zero showings until
  something writes to it). `run_id` references `sessions`, not any run concept, and is
  confirmed unused/always NULL in all existing code.
- **Blast radius**: `EnsureForSession` is called from 20+ sites — every write in
  `backend/internal/actions/*.go` (chat, token, reveal, stage_actions, discord_chat,
  indexcard, duplicate, place, overlay, remove, game_event, react, persona, ooc, dice),
  `network/session_control.go` (session start/reattach/close), `identity/discord_mic.go`
  (mic threads carry both `SessionID` and `ShowingID`), and `identity/join.go`.
- **Conclusion**: `showings.session_id`'s `NOT NULL UNIQUE` constraint cannot be safely
  loosened to support a pre-session, multi-session Show without touching all of the
  above. This directly contradicts the original draft spec's default instruction to
  "prefer extending existing showings" — flagged to the operator before any code was
  written, who confirmed building a separate `shows` table instead.

## 3. Data model actually implemented

New migration `database/migrations/040_kernel67_shows.sql`:
- **`shows`**: `id, show_run_id (FK show_runs CASCADE NOT NULL), slug, title,
  description, audience_title, audience_program_blurb, status (CHECK: 7 values —
  draft/scheduled/live/paused/completed/cancelled/archived, deliberately different from
  `show_runs.status`'s 5), scheduled_start_at, scheduled_end_at, actual_start_at,
  actual_end_at, created_by_user_id, created_at, updated_at, archived_at`.
  `UNIQUE(show_run_id, slug)`. Three CHECK constraints: status enum, archived/archived_at
  consistency (mirrors Kernel 66's pattern), and a new
  `scheduled_end_at >= scheduled_start_at` (both-set case) — proven live to reject an
  inverted range and accept a valid one.
- **`sessions.show_id`**: nullable `ADD COLUMN IF NOT EXISTS ... REFERENCES shows(id) ON
  DELETE SET NULL`. `showings` itself: zero changes.

## 4. How user-facing "Show" maps to internal code

"Show" is a brand-new, separate concept — it does **not** map onto `showings` at all.
`showings` remains exactly what it was before this kernel: the live, 1:1 runtime wrapper
Kernel 22 built, unrenamed and unmodified. The user-facing Show is implemented entirely
by the new `shows` table and `backend/internal/shows` package. `Construction/
Dictionary.txt`'s "Showing vs. Show" note was rewritten to state this explicitly, since
Kernel 66's version of that note predicted the opposite (that scheduling would extend
`showings`) and would otherwise mislead a future kernel.

## 5. Proof that `/session` and `/mic` behavior was not broken

The new `sessions.show_id` column is additive and nullable — no existing column,
constraint, or query was touched. Verified, not assumed: the full existing
`internal/network` and `internal/actions` test suites (the packages that own
`/session`/`/mic`-adjacent behavior) were re-run unchanged after the migration landed,
alongside the rest of the suite:
```
$ TEST_DATABASE_URL=... go test -count=1 ./...
ok  victory/backend/internal/network   2.051s
ok  victory/backend/internal/actions   0.015s
... (all 17 packages, zero failures)
```

## 6. Authority proof, including wrong-location negative case

`backend/internal/showruns/authority.go`'s `canManageShowRun`/`canViewShowRun` were
exported to `CanManageShowRun`/`CanViewShowRun` (pure rename, ~16 in-package call sites
updated via scoped find/replace, confirmed via `go build`) so `backend/internal/shows`
reuses the exact same location-scoped authority logic rather than a second copy.
`TestCreateShowRequiresManageAuthorityAtRunLocation` proves: a Producer at the Show
Run's own location can create a Show; a Producer at a **different** location is
rejected (`not_authorized`) even though they hold a real Producer role somewhere; a user
with no location membership at all is rejected. Live-verified: `GET /api/shows/{id}`
anonymously → 401.

## 7. Audience Program privacy proof

`TestAudienceProgramIsShowSpecificAndExcludesBackstageFields` proves two Shows under the
same Show Run render distinct `audience_title`/`audience_program_blurb` values (not the
parent Show Run's title/description). Live-verified: `GET /api/shows/{id}/program` as
the disposable Audience account returned this Show's own `"audience_title":"Opening
Night"` and contained no `added_by_user_id` or other internal-only roster field.

## 8. Dedicated `TEST_DATABASE_URL` proof

```
$ TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
    scripts/test/setup-test-database.sh
... PASS: victory_test is set up, migrated, and Go-side bootstrapped.

$ TEST_DATABASE_URL=... go test -count=1 -v ./internal/shows/...
--- PASS: TestCreateShowRequiresManageAuthorityAtRunLocation
--- PASS: TestShowBelongsToParentShowRunAndListReturnsMultiple
--- PASS: TestShowStatusCheckRejectsInvalidValue
--- PASS: TestShowArchivedAtMatchesStatusConstraint
--- PASS: TestShowScheduledFieldsAreOptionalAndOrderIsEnforced
--- PASS: TestAudienceProgramIsShowSpecificAndExcludesBackstageFields
--- PASS: TestShowRosterInheritsShowRunVisibilityWithNoOwnRosterTable
--- PASS: TestSessionLinkAndUnlinkSetsAndClearsShowID
PASS

$ unset TEST_DATABASE_URL; go test ./internal/shows/...
--- FAIL (all 8): dbtest: TEST_DATABASE_URL is required for database-touching tests
```

## 9. Fresh-install proof

```
$ scripts/smoke/fresh-install.sh --local
... (all pre-existing Kernel 61A/62/63/65/66 assertions still PASS) ...
PASS Shows list rejects anonymous requests
PASS Producer A created a Show (...)
PASS Show detail fetch confirms manage authority
PASS Show PATCH round-trips audience_title/blurb/status (snake_case JSON confirmed)
PASS Show Program is curated and Show-specific
PASS Show archive sets status and archived_at
PASS Show page files exist
PASS clean-install smoke complete
```
Migration `040` was added to `scripts/smoke/fresh-install.sh`'s hardcoded migration
array (still a plain bash array, not a glob — the exact gap Kernel 66 hit on its first
run; this kernel added it correctly the first time because of that documented lesson).

## 10. Live deployment proof

```
$ docker exec -i victory-postgres psql -U victory -d victory < database/migrations/040_kernel67_shows.sql
BEGIN ... COMMIT   # additive-only
$ docker compose up -d --build backend
 Container victory-backend Started
$ docker exec victory-backend wget -qO- http://127.0.0.1:8081/health
{"ok":true,"service":"victory-backend", ...}
$ curl -s https://victory.amurray.family/venues/show-runs/show.html   → 200
$ curl -s https://victory.amurray.family/api/shows/nonexistent        → {"ok":false,"data":{"error":"not_authenticated"}}
```

Live authenticated proof (disposable `k67live_producer_*` bootstrapped as Producer at
`amurray-family`, `k67live_audience_*` plain signup; real existing "Main Production"):
create Show Run → create Show (`status:"draft"` by default, confirmed) → PATCH with
`audience_title`/`audience_program_blurb`/`status:"scheduled"`/scheduled dates (full
snake_case round-trip confirmed) → add Producer to the Show Run roster → `GET
/api/shows/{id}` confirms the roster is inherited with zero Show-scoped roster rows →
enable self-join → Audience account's `GET /api/shows/{id}/program` shows this Show's
own `audience_title`, no internal fields → inserted a real `sessions` row directly,
linked it to the Show (`sessions.show_id` set, confirmed via direct query), unlinked it
(cleared back to NULL, confirmed) → anonymous request to `GET /api/shows/{id}` → 401 →
archived the Show (`status:"archived"`, `archived_at` set).

No screenshot-based browser proof — `chromium-cli` (and no other headless-browser tool)
was still not available in this environment, same gap as Kernel 66. Substituted the same
real authenticated-HTTP-session proof (real signup cookies, real `victory_session`
cookie on every request) as before.

### Cleanup
All disposable live-proof data removed and verified zero-residue:
```sql
SELECT 'users', count(*) FROM users WHERE handle LIKE 'k67live%';                    -- 0
SELECT 'shows', count(*) FROM shows WHERE slug = 'kernel67-live-proof-show';         -- 0
SELECT 'show_runs', count(*) FROM show_runs WHERE slug = 'kernel67-live-proof-run';  -- 0
```
The disposable test `sessions` row inserted for the link/unlink proof was deleted by its
exact ID and confirmed gone. The real "Main Production" and `amurray-family` location
used as fixtures were read-only throughout.

## 11. Dictionary update summary

`Construction/Canon/Dictionary.txt`: added a `Show (Show Instance)` entry (hierarchy,
distinction from Session and Showing, notes `sessions.show_id` is a manual, non-wired
link). **Corrected** the existing `Showing (live/runtime) vs. future scheduled
occurrence` note from Kernel 66 — it previously predicted future scheduling would
extend `showings`; the live audit in §2 above found that unsafe, so the note now states
`shows` (not `showings`) is the scheduling primitive going forward, and explains why the
earlier prediction changed.

## 12. Known issues and next recommended step

**Known issues:**
- One incidental, unrelated flake observed during the final full-suite validation run:
  `TestDiscordGatewayDebugToggleEndpoint` (`internal/identity`) failed once with
  `ERROR: deadlock detected (SQLSTATE 40P01)` under full-suite concurrent load, then
  passed cleanly both in isolation and on a full-suite re-run immediately after. Not a
  regression from this kernel (nothing in `internal/identity` was touched) — consistent
  with the pre-existing Discord-test flakiness already noted in project memory from
  before Kernel 64's isolation work. Not fixed, per Kernel 64/65's own precedent of
  documenting rather than chasing pre-existing unrelated flakes.
- No screenshot evidence, same gap as Kernel 66 — `chromium-cli`/headless-browser
  tooling still unavailable in this environment.
- The Session Link UI (`show.html`) uses a plain session-ID text field, not a picker —
  no session-picker UI exists anywhere in the app yet, and the spec explicitly allowed
  keeping this minimal.
- Session-to-Show association remains entirely manual; `/session start` still has no
  automatic way to populate `sessions.show_id` (by design — deferred per the spec).

**Next recommended step:** Scene Configuration Model / Capture Scene — the displaced
VTT-core work the original roadmap called for, and now has a real container (Show) to
attach to. A Showing-scheduling extension (letting `showings` itself gain a
scheduled/pre-live status for the case where a Show's *current* live occurrence needs
richer state) and the older stale-test-user sweep remain smaller, independent
candidates.

## 13. Files changed or created

**New:**
- `database/migrations/040_kernel67_shows.sql`
- `backend/internal/shows/{types,shows,sessions,roster,http,shows_test}.go`
- `frontend/venues/show-runs/show.html`, `show-program.html`
- `Construction/Kernels/kernel-67-show-instance-model-run-bridge-mvp-v0.1.md`
- `Construction/OperatorLogs/kernel-67-reportback.md` (this file)

**Modified:**
- `backend/internal/showruns/authority.go`, `blocks.go`, `http.go`, `roster.go`,
  `showruns.go`, `showruns_test.go` (authority-function export rename)
- `backend/cmd/victory/main.go` (import + 6 new route registrations)
- `frontend/venues/show-runs/run.html` (new "Shows" section)
- `scripts/smoke/fresh-install.sh` (migration 040 added; 8 new assertions)
- `Construction/Canon/Dictionary.txt` (new Show entry; corrected Showing note)

## 14. Project-memory updates completed

- `Construction/OperatorLogs/operator-log.md` — Kernel 67 entry appended.
- `Construction/OperatorLogs/operator-notes.md` — authority-rename note (so a future
  kernel searching for the old lowercase `canManageShowRun`/`canViewShowRun` names
  doesn't fail), plus the showings-vs-shows data-model decision recorded as a durable
  fact.
