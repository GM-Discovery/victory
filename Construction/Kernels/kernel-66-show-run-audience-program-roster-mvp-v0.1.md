# Kernel 66 — Show Run, Audience Program, and Roster MVP

## 0. Numbering note (read first)

`Construction/roadmaps/victory-master-actual-implementation-guide-v1.md` contains a
**stale, unrelated document-internal "Kernel 66 — Scene Transitions, Cue Groups, and
Courtyard Tutorial Beat"** section, left over from earlier aspirational planning. It is
**not** this kernel. The operator log confirms Kernel 65 (Third Place Headshot Commons
MVP, PASS, 2026-07-11) is the last real kernel landed, and this document is the real,
authoritative Kernel 66. The roadmap doc is updated alongside this kernel to note the
collision, following the same precedent already used for kernels 62/64/65.

## 1. Objective

Build the first real Show Run primitive: a bounded container —
`Production → Show Format → Show Run → Roster / Audience Program` — for a
cohort/campaign/table series of a Production, without building Showing, Scene
configuration, scheduling, tags, comments, impressions/autographs, feeds, or a
site-wide moderation system.

Show Run is the code implementation of the dictionary's pre-existing "Production
Run" concept (never previously implemented). "Show Run" is its product-facing
name; the dictionary entry is merged, not duplicated (`Construction/Dictionary.txt`).

## 2. Why this kernel now

Kernel 65 (Third Place) gave Victory a live, always-current social presence
(Headshots) and Kernel 62 (My People) gave every user a private relationship
matrix. Neither has anywhere to attach a *specific bounded activity* — a
cohort, a playtest, a table series. Show Run is that container.

## 3. Resolved owner decisions (do not re-litigate)

These were resolved by the operator against the live repository, not assumed
from the original draft spec:

1. **`productions` already exists.** `show_runs.production_id` is
   `NOT NULL REFERENCES productions(id) ON DELETE RESTRICT`, resolved
   server-side from the given production, never trusted from the client —
   the exact pattern `backend/internal/showings/showings.go` already uses.
2. **"Showing" name collision.** Kernel 22's `Showing` (live, 1:1 runtime
   wrapper around an active `sessions` row) is *not* renamed or shadowed.
   A future pre-live scheduling capability must extend the existing
   `showings` table/package (e.g. a scheduled/pre-live status), not
   introduce a second same-named concept. This kernel does not implement
   scheduling; it only documents the correct future path in the dictionary.
3. **"Show Run" IS "Production Run."** The dictionary's existing
   `Production Run` entry is merged into a `Show Run (Production Run)` entry
   rather than duplicated under a new term. The draft spec's proposed
   "Work / Game Work" top-level hierarchy is dropped — it would shadow the
   existing, correct Production/Session/Production-Run trio.
4. Audience is first-class: "Audience gets the best seats." Audience section
   is ordered *first* in both internal roster and Audience Program views,
   never last or hidden behind other roles.
5. Audience does not see the full internal roster by default — the
   Audience Program is a curated projection (`program_visible = TRUE` rows
   only, and a narrower field set that omits `added_by_user_id` and other
   internal-only metadata).
6. Roster label uses **"Player," never "Cast."** Enforced in exactly one
   place in code (`roleDisplayLabel` in `backend/internal/showruns/projection.go`)
   so no call site can drift from this rule.
7. Show Run creation/roster management authority: Producer or Director,
   location-scoped (not global), or Operator (bypasses everything).
8. Audience self-join: opt-in per run (`audience_self_join_enabled`),
   blocked users cannot self-join, Producer/Director adding a blocked user
   as Audience is rejected unless the actor is Operator (explicit override).
9. Data model kept deliberately small: three tables
   (`show_runs`, `show_run_roster_members`, `show_run_audience_blocks`), no
   event/history/tag/journal tables.
10. Roster cards are always live Trailer Face projections — never a stored
    snapshot, matching Kernel 65's Headshot discipline exactly.

## 4. Data model

`database/migrations/039_kernel66_show_runs.sql`:

- **`show_runs`** — `id, location_id, production_id, title, slug,
  description, show_format (CHECK, not ENUM), custom_show_format,
  cohort_name, status (CHECK: planning/active/paused/completed/archived),
  audience_self_join_enabled, created_by_user_id, created_at, updated_at,
  archived_at`. `UNIQUE(location_id, slug)`.
- **`show_run_roster_members`** — `id, show_run_id, user_id, role (CHECK:
  producer/director/player/crew/audience/guest/observer/custom),
  custom_role_label, program_visible, added_by_user_id, added_at,
  removed_at`. Partial unique index `(show_run_id, user_id) WHERE removed_at
  IS NULL` — DB-level "one active row per user per run" enforcement, not
  just an application check.
- **`show_run_audience_blocks`** — `id, show_run_id, user_id,
  blocked_by_user_id, reason, created_at, lifted_at`. Same partial-unique
  pattern. Run-scoped only, never a site-wide moderation record (confirmed:
  no such primitive existed anywhere in the codebase before this kernel).

`show_format`/`status`/`role` use `TEXT CHECK` rather than a Postgres
`ENUM`, matching Kernel 65's choice — a `CHECK` is a plain additive
migration to extend later, an `ENUM` needs `ALTER TYPE`.

The `show-runs` venue is seeded with plain idempotent SQL in the same
migration (no Go-side `Ensure*Surface` bootstrap needed), following
`038_kernel65_third_place.sql`'s exact precedent.

## 5. Backend

New package `backend/internal/showruns/`:
`types.go`, `showruns.go` (CRUD/lifecycle), `roster.go` (add/update/remove/
self-join/list), `blocks.go` (block/unblock), `projection.go` (live roster
card + role-label mapping), `authority.go` (`canManageShowRun`/
`canViewShowRun`), `http.go` (handlers).

New location-scoped authority helpers in `backend/internal/access/`:
`CurrentLocationRoleForLocation` and `HasActiveLocationMembership`
(`location_role.go`) — the pre-existing `CurrentLocationRole` ignores which
location is in play and would have let a Producer at Location A pass a
check for Location B. Also `HasAnyManageableLocation`, used by the Third
Place integration point.

Roster-card projection (`ProjectRosterMember`/`ProjectAudienceProgramEntry`)
follows `thirdplace.ProjectHeadshot`'s exact live-projection pattern:
`playerprofile.ProjectTrailerFace` is called fresh on every read, nothing
Face-shaped is ever stored on a roster row.

Routes registered in `backend/cmd/victory/main.go` using the newer Go 1.22
method+path-param style (`showings`' precedent), not `thirdplace`'s older
flat style, since Show Runs has real path params (`{id}`, `{member_id}`,
`{block_id}`).

`ResolveVisibleVenues` (`backend/internal/access/visibility.go`) gets a new
UNION arm for `show-runs` with **no role filter** — any active location
membership, including `audience`, sees the venue tile, unlike Third
Place/Trailers which exclude Audience.

## 6. HTTP/API surface

```
GET/POST     /api/show-runs
GET/PATCH    /api/show-runs/{id}
POST         /api/show-runs/{id}/archive | /unarchive
GET/POST     /api/show-runs/{id}/roster
PATCH/DELETE /api/show-runs/{id}/roster/{member_id}
POST         /api/show-runs/{id}/roster/self-join
GET          /api/show-runs/{id}/audience-program
POST         /api/show-runs/{id}/blocks
DELETE       /api/show-runs/{id}/blocks/{block_id}
```

Anonymous requests are rejected (401) on every route. Authorization
failures return 403, not 404 — Show Run existence is not meant to be
secret, unlike Kernel 62's subject-invisibility rule.

## 7. Frontend

New venue `frontend/venues/show-runs/`: `index.html` (list + create),
`run.html` (metadata edit, archive/unarchive, links to roster/program),
`roster.html` (internal roster management — Producer/Director/Operator
only, sections ordered Audience-first, add-by-profile-ID-or-Trailer-link,
role change, remove), `program.html` (curated Audience Program view,
"Join as Audience" when eligible).

Third Place integration: `frontend/venues/third-place/index.html`'s
`renderCard` gets an "Add to Show Run" chip, gated on a new
`can_add_to_show_run` field on `HeadshotProjection`
(`backend/internal/thirdplace/headshots.go`, computed via
`access.HasAnyManageableLocation`). Clicking it opens a picker over the
viewer's own manageable, non-archived Show Runs and calls the existing
`POST /api/show-runs/{id}/roster` — no new endpoint.

## 8. Explicit exclusions

Showing implementation, Scene configuration/capture/fly, attendance
tracking, scheduling/calendar, comments, tags/circles,
impressions/autographs, feeds, discovery algorithms, DMs, site-wide bans,
tickets/payment, audience analytics, public anonymous access.

## 9. Tests

Kernel 64's dedicated-test-DB workflow (`TEST_DATABASE_URL`,
`scripts/test/setup-test-database.sh`, hard-fail without it — verified).
`backend/internal/showruns/showruns_test.go` covers: one-active-roster-row
DB-level enforcement and in-place role update; location-scoped authority
(same role at the *wrong* location is rejected — the specific bug
`CurrentLocationRoleForLocation` exists to prevent — and Operator bypass);
self-join respecting `audience_self_join_enabled` and blocks, including
idempotent repeat self-join; non-Operator vs. Operator-override block
behavior; Audience-first Audience Program ordering and `program_visible`
filtering; live Trailer Face re-projection on roster cards; role label
never rendering "Cast" for `role="player"`; Audience can view but not
manage.

## 10. PASS standard

Kernel 66 passes when: the three-table data model exists and enforces one
active roster row per user per run at the DB level; Producer/Director
authority is correctly location-scoped (not global); Operator bypasses;
Audience is first-class in both internal roster and Audience Program
ordering, without seeing full internal roster data by default; roster cards
are always live Trailer Face projections; "Player" is the only label shown
for that role; self-join/block behavior is safe; the dictionary reflects
the Show-Run/Production-Run merge and the Showing collision resolution
without inventing new competing terms; `go test ./...` is green with
`TEST_DATABASE_URL`; and DB-touching tests hard-fail without it.
