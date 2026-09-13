# Kernel Report Back — Kernel 66: Show Run, Audience Program, and Roster MVP

## 0. Numbering note

`Construction/Canon/roadmaps/victory-master-actual-implementation-guide-v1.md` has a stale,
unrelated document-internal "Kernel 66 — Scene Transitions, Cue Groups, and Courtyard
Tutorial Beat" section left over from earlier aspirational planning. It is not this
kernel. Kernel 65's own reportback (§12) named "Show Run primitive / Run Roster MVP" as
the next recommended step, and the operator log confirms Kernel 65 (PASS, 2026-07-11) is
the last real kernel landed before this one. This document is the real, authoritative
Kernel 66. The roadmap doc is updated alongside this reportback to flag the collision,
per the same precedent already used for kernels 62/64/65.

## 1. Status

**PASS.** Deployed live: migration 039 applied to the live `victory` database, backend
rebuilt and restarted, `/health` OK. Uncommitted, per the same operator convention used
for kernels 62–65 (the operator reviews before committing).

## 2. Acceptance-criterion ledger

| # | Criterion | Result |
|---|---|---|
| 1 | Small, understandable 3-table data model (`show_runs`, `show_run_roster_members`, `show_run_audience_blocks`) | PASS |
| 2 | `production_id` resolved server-side, never trusted from client | PASS |
| 3 | Producer/Director authority correctly **location-scoped**, not global | PASS — new `access.CurrentLocationRoleForLocation`, tested with a same-role-wrong-location negative case |
| 4 | Operator bypasses all authority checks | PASS |
| 5 | Audience is first-class: ordered first in internal roster and Audience Program, never hidden | PASS |
| 6 | Audience Program is curated (no `added_by_user_id`/internal-only fields) | PASS, verified live |
| 7 | Roster label is always "Player," never "Cast" | PASS — single `roleDisplayLabel` chokepoint, tested and live-verified |
| 8 | Roster cards are live Trailer Face projections, never stored snapshots | PASS |
| 9 | Self-join respects `audience_self_join_enabled` and blocks | PASS |
| 10 | Non-Operator adding a blocked user as Audience is rejected; Operator override works | PASS |
| 11 | Dictionary reflects Show-Run/Production-Run merge and Showing-collision resolution, no new competing "Work/Game Work" hierarchy | PASS |
| 12 | Anonymous requests rejected on every route | PASS |
| 13 | Third Place → Add to Show Run integration reuses the existing roster-add endpoint | PASS, no new endpoint |
| 14 | Kernel 64 dedicated-test-DB workflow followed throughout | PASS |
| 15 | `go test ./...` green with `TEST_DATABASE_URL`; hard-fails without it | PASS |
| 16 | `fresh-install.sh --local` passes with new migration + assertions | PASS |
| 17 | Live deployment healthy, zero residue after proof | PASS |

## 3. What was built

### Backend
- `database/migrations/039_kernel66_show_runs.sql` — `show_runs`, `show_run_roster_members`,
  `show_run_audience_blocks`, plus the `show-runs` venue seed (same idempotent pattern as
  Kernel 65's `third-place` seed — no Go-side `Ensure*Surface` bootstrap needed).
- New package `backend/internal/showruns/`: `types.go`, `showruns.go` (create/update/
  archive/list, all authority-checked against a server-resolved location), `roster.go`
  (add/update-role/remove/self-join/list-internal/list-program), `blocks.go`
  (block/unblock/is-blocked), `projection.go` (live roster-card projector + the single
  `roleDisplayLabel` mapping), `authority.go` (`canManageShowRun`/`canViewShowRun`),
  `http.go` (13 handlers).
- New `backend/internal/access/location_role.go`: `CurrentLocationRoleForLocation`,
  `HasActiveLocationMembership`, `HasAnyManageableLocation` — the pre-existing
  `CurrentLocationRole` ignores which location is in play; using it unmodified here would
  have let a Producer at Location A pass an authority check for a Show Run at Location B.
- `backend/internal/access/visibility.go`: new `show-runs` UNION arm in
  `ResolveVisibleVenues` with **no role filter** — unlike Third Place/Trailers, this
  surface must be visible to `audience`-role members too. Verified live: an
  audience-only-role disposable account saw the `show-runs` venue tile.
- `backend/internal/thirdplace/`: `HeadshotProjection` gained `can_add_to_show_run`
  (computed via `access.HasAnyManageableLocation`), the integration point for the
  Third-Place-to-Show-Run chip.
- 13 new routes registered in `backend/cmd/victory/main.go`.

### Frontend
- New venue `frontend/venues/show-runs/`: `index.html` (list + create, production picker
  via the existing `/api/productions`), `run.html` (metadata edit, archive/unarchive),
  `roster.html` (internal roster management, Audience-first sectioning, add-by-profile-ID-
  or-Trailer-link), `program.html` (curated Audience Program, self-join button).
- `frontend/venues/third-place/index.html`: new "Add to Show Run" chip + lightweight
  picker overlay, calling the existing `POST /api/show-runs/{id}/roster` — no new endpoint.

### Documentation
- `Construction/Canon/Dictionary.txt`: merged `Production Run` → `Show Run (Production Run)`;
  added a `Showing (live/runtime) vs. future scheduled occurrence` note so a later kernel
  doesn't reintroduce the same-named collision with Kernel 22's `Showing`.
- `Construction/Kernels/kernel-66-show-run-audience-program-roster-mvp-v0.1.md` — the
  finalized spec, written to already contain every resolved decision below (not re-askable).

## 4. Resolved decisions this kernel encoded (not re-askable by a future kernel)

The original draft spec assumed `productions` didn't exist yet and left `production_id`
nullability ambiguous, and it proposed inventing a new "Work / Game Work" dictionary
hierarchy plus a future "Showing" definition that would have collided with Kernel 22's
already-shipped `Showing`. Before writing any code, this was checked against the live
repository and corrected:

1. **`productions` already existed** (`database/migrations/000_kernel42_productions_baseline.sql`,
   already consumed by `showings.go`) — `show_runs.production_id` follows that exact
   `NOT NULL REFERENCES productions(id) ON DELETE RESTRICT` pattern, resolved server-side.
2. **"Showing" name collision** — Kernel 22's `Showing` is a live, 1:1 runtime wrapper
   around an active `sessions` row; it is not renamed or shadowed. A future pre-live
   scheduling capability must extend that existing table/package, documented in the
   dictionary, not built as a second same-named concept. Not implemented in this kernel.
3. **"Show Run" IS "Production Run"** — the dictionary's pre-existing, never-implemented
   `Production Run` entry was merged into `Show Run (Production Run)` rather than
   duplicated under a new term; the draft's "Work / Game Work" hierarchy was dropped.

## 5. Evidence

### Automated checks
```
$ go build ./...          # clean
$ go vet ./...             # clean
$ gofmt -l ./internal/showruns/*.go   # clean after formatting
```

### Database/domain proof (dedicated test DB, Kernel 64 workflow)
```
$ TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
    scripts/test/setup-test-database.sh
... PASS: victory_test is set up, migrated, and Go-side bootstrapped.

$ TEST_DATABASE_URL=... go test -count=1 -v ./internal/showruns/...
--- PASS: TestCreateShowRunRequiresProducerOrDirectorAtCorrectLocation
--- PASS: TestAddRosterMemberEnforcesOneActiveRowPerUser
--- PASS: TestSelfJoinAsAudienceRespectsFlagAndBlocks
--- PASS: TestListAudienceProgramMembersOrdersAudienceFirstAndRespectsVisibility
--- PASS: TestProjectRosterMemberReflectsLiveTrailerFace
--- PASS: TestAudienceCanViewButNotManage
PASS

$ TEST_DATABASE_URL=... go test -count=1 ./...
ok all 17 packages (including internal/showruns, internal/thirdplace, internal/access)
```

Hard-fail without `TEST_DATABASE_URL` confirmed:
```
$ unset TEST_DATABASE_URL; go test ./internal/showruns/...
--- FAIL: ... dbtest: TEST_DATABASE_URL is required for database-touching tests
(all 6 tests fail this way — no silent skip)
```

### A real bug caught by this proof process
`ShowRun`, `RosterMember`, and `AudienceBlock` structs initially had no `json` tags,
so `json.Marshal` emitted capitalized Go field names (`"ID"`, `"ShowFormat"`) instead of
the snake_case the frontend and smoke assertions expect. Unit tests (which use struct
fields directly, not JSON) did not catch this — it only surfaced when
`fresh-install.sh --local`'s new Show Run assertions tried to parse the real HTTP
response and failed with a UUID-parse error one level down (the *next* request received
a malformed value). Fixed by adding full `json` tags to all three raw-row structs.
Recorded in `operator-notes.md` as a standing lesson (§6 below).

### Fresh-install proof
```
$ scripts/smoke/fresh-install.sh --local
... (all pre-existing Kernel 61A/62/63/65 assertions still PASS) ...
PASS Show Runs list rejects anonymous requests
PASS fixture production created for Show Run smoke checks (...)
PASS Producer A created a Show Run (...)
PASS roster member added with role_label Player, never Cast
PASS B self-joined as Audience once self-join was enabled
PASS Audience Program is curated and excludes internal-only roster fields
PASS Show Runs page files exist
PASS clean-install smoke complete
```
`scripts/smoke/fresh-install.sh`'s hardcoded migration array (not a glob — a real gap
this kernel's first fresh-install run caught) needed `039_kernel66_show_runs.sql` added
explicitly; fixed.

### Live deployment proof
```
$ docker exec -i victory-postgres psql -U victory -d victory < database/migrations/039_kernel66_show_runs.sql
BEGIN ... COMMIT   # additive-only, IF NOT EXISTS throughout
$ docker compose up -d --build backend
 Container victory-backend Started
$ docker exec victory-backend wget -qO- http://127.0.0.1:8081/health
{"ok":true,"service":"victory-backend", ...}
$ curl -s https://victory.amurray.family/venues/show-runs/   → 200
$ curl -s https://victory.amurray.family/api/show-runs        → {"ok":false,"data":{"error":"not_authenticated"}}
```

Live authenticated proof (two disposable accounts, `k66live_producer_*` bootstrapped as
Producer at `amurray-family`, `k66live_audience_*` plain signup; real existing "Main
Production"):
- Producer created a Show Run — response used correct snake-case JSON.
- Producer added themselves to the roster as `player` → `role_label: "Player"` (never
  "Cast") confirmed in the live response.
- Producer enabled `audience_self_join_enabled`; audience account self-joined
  successfully (already held an active `audience` location membership from signup) →
  `role: "audience"`, `role_label: "Audience"`.
- Audience Program view (as the audience account) showed **Audience entry first**, then
  Player — confirmed the "best seats" ordering live, not just in unit tests — and
  contained no `added_by_user_id` or other internal-only field.
- Internal roster view (as the producer) showed the full 2-row roster with internal
  metadata.
- Anonymous request to the internal roster route → 401.
- `GET /api/map/visibility` for the audience-only account included the `show-runs` venue
  tile — confirming the new no-role-filter visibility arm works for a real audience
  member, not just Producer/Director/Crew.
- Third Place: after the audience account left a Headshot, the producer account's view
  of that Headshot included `"can_add_to_show_run": true`, confirming the integration
  point's authority check works live.

No screenshot-based browser proof was produced — `chromium-cli` was not available in
this environment. The live proof above exercises the exact same authenticated-session,
real-HTTP-request path a browser session would use (real login cookies via
`/api/auth/signup`, real `victory_session` cookie on every subsequent request), just
without pixel rendering. This is a real gap relative to Kernel 65's screenshot evidence;
flagged in §9.

### Cleanup
All disposable live-proof data removed and verified zero-residue:
```sql
-- after proof:
SELECT 'users', count(*) FROM users WHERE handle LIKE 'k66live%';              -- 0
SELECT 'show_runs', count(*) FROM show_runs WHERE slug = 'kernel66-live-proof-run'; -- 0
SELECT 'location_memberships', count(*) FROM location_memberships lm
  JOIN users u ON u.id = lm.user_id WHERE u.handle LIKE 'k66live%';            -- 0
```
The real "Main Production" and `amurray-family` location used as the fixture's parent
were read-only throughout — no production/location rows were created or modified.

## 6. Operator notes

- Struct JSON tags are load-bearing, not cosmetic: a struct returned directly through
  `writeOK(w, map[string]any{"show_run": sr})` must have `json:"snake_case"` tags on
  every field, or `encoding/json`'s default (capitalized Go field name) ships to the
  frontend silently wrong. Unit tests that assert on Go struct fields directly will not
  catch this — only an actual HTTP round-trip (fresh-install, live proof, or an
  HTTP-layer test asserting on raw response bytes) will.
- `scripts/smoke/fresh-install.sh`'s migration list is a hardcoded array, not a glob —
  every new migration must be added to it explicitly or it is silently skipped on a
  fresh install (no error, migrations just stop one short).
- There is currently no in-app "create a Production" flow anywhere in Victory — both
  `producers-office` and this kernel's own Show Run creation UI only ever *consume*
  `/api/productions`. On a genuinely fresh install this list is empty. Fresh-install and
  live proof both worked around this by inserting a fixture/using a real pre-existing
  production directly; this is a pre-existing gap, not something Kernel 66 introduced or
  is responsible for closing.
- New users appear to receive an active `audience` `location_memberships` row at the
  neutral install location automatically on signup (observed live: the disposable
  audience account could self-join and saw the `show-runs` venue tile with zero manual
  setup). This kernel did not investigate or change that behavior, only relied on it
  being consistent with `HasActiveLocationMembership`'s existing semantics.

## 7. Blockers and workarounds

- `chromium-cli` (or any headless-browser tool) was not available in this environment.
  Worked around with authenticated real-HTTP-session proof against the live server
  instead of pixel screenshots (§5). Recommend a future kernel or session set up browser
  tooling if pixel-level UI regressions become a concern.

## 8. Deviations from kernel

None. All resolved operator decisions (§4) were followed as specified.

## 9. Known issues

- No screenshot evidence for the four new frontend pages or the Third Place chip — only
  HTTP-level proof that the APIs they call behave correctly. The pages were also
  syntax-checked (`node --check` on extracted inline scripts) but never rendered in an
  actual browser in this session.
- The "no in-app Production creation flow" gap (§6) means a real Producer cannot create
  their first Production without direct database access today. Out of scope for this
  kernel; worth a future kernel if it becomes an actual blocker for onboarding.

## 10. Files changed or created

**New:**
- `database/migrations/039_kernel66_show_runs.sql`
- `backend/internal/showruns/{types,showruns,roster,blocks,projection,authority,http,showruns_test}.go`
- `backend/internal/access/location_role.go`
- `frontend/venues/show-runs/{index,run,roster,program}.html`
- `Construction/Kernels/kernel-66-show-run-audience-program-roster-mvp-v0.1.md`
- `Construction/OperatorLogs/kernel-66-reportback.md` (this file)

**Modified:**
- `backend/internal/access/visibility.go` (new `show-runs` UNION arm)
- `backend/internal/thirdplace/types.go`, `headshots.go` (`can_add_to_show_run` field)
- `backend/cmd/victory/main.go` (13 new route registrations)
- `frontend/venues/third-place/index.html` ("Add to Show Run" chip + picker)
- `scripts/smoke/fresh-install.sh` (migration 039 added to array; 8 new assertions)
- `Construction/Canon/Dictionary.txt` (Show Run / Production Run merge; Showing collision note)

## 11. Project-memory updates completed

- `Construction/OperatorLogs/operator-log.md` — Kernel 66 entry appended.
- `Construction/OperatorLogs/operator-notes.md` — JSON-tag lesson, fresh-install
  hardcoded-migration-array lesson, and no-Production-creation-flow gap recorded.
- `Construction/Canon/roadmaps/victory-master-actual-implementation-guide-v1.md` — collision
  note added next to the stale internal "Kernel 66" section.

## 12. Next recommended step

Show Run now gives Third Place Headshots and My People relationships somewhere bounded
to attach to. The natural next steps, in rough order of dependency:

1. **Scene Configuration Model / Capture Scene** — the displaced VTT-core work the
   original roadmap called for after the identity/social spine stabilized.
2. **Showing scheduling extension** — if the operator wants pre-live scheduling, this is
   the kernel that extends Kernel 22's `showings` table with a scheduled/pre-live status,
   per the dictionary note this kernel added. Not urgent unless requested.
3. A small, deliberately separate stale-test-user sweep remains an independent,
   low-priority candidate (named in kernels 62/63/64, still untouched).
