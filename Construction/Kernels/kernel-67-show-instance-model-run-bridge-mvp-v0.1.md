# Kernel 67 — Show Instance Model and Show Run Bridge

## 0. Purpose

Build the missing container between a Show Run (Kernel 66) and the technical
live Session/Showing system (Kernel 22): `Production -> Show Run -> Show ->
Session(s)`. A **Show** is the concrete playable/viewable instance of a Show
Run — a planned future occurrence with no session yet, a live occurrence, a
multi-session arc, or a completed/cancelled/archived one.

## 1. Resolved operator decisions

1. User-facing term is **Show**.
2. **Data model: a brand-new `shows` table. `showings` (Kernel 22) is not
   modified, renamed, or rewired.** A live-code audit found
   `showings.session_id` is `NOT NULL UNIQUE REFERENCES sessions(id)`,
   lazily created on first action-write, and read/written from 20+ call
   sites (`backend/internal/actions/*.go`, `network/session_control.go`,
   `identity/discord_mic.go`, `identity/join.go`). Loosening that
   constraint to let a Show pre-exist without a session was assessed and
   explicitly rejected as too risky — this is a deviation from the original
   draft spec's default instruction to "prefer extending showings," made
   with operator confirmation after the audit.
3. `sessions` gains a nullable `show_id` FK. A session belongs to zero or
   one Show; no join table needed.
4. Session association is a minimal, manual, separate link/unlink action.
   It does **not** wire into the existing `/session start` flow
   (`network/session_control.go`) — that stays untouched.
5. A Show has no roster table of its own — it inherits its parent Show
   Run's roster and authority wholesale.
6. A Show's Audience Program is genuinely Show-specific (its own
   `audience_title`/`audience_program_blurb`), while the roster underneath
   it is still the shared Show Run roster.
7. Status set is `draft, scheduled, live, paused, completed, cancelled,
   archived` — 7 values, deliberately different from `show_runs.status`'s 5.
8. Dates are optional throughout; UI uses native `datetime-local` inputs,
   no calendar widget.

## 2. Data model

`database/migrations/040_kernel67_shows.sql`:
- **`shows`**: `id, show_run_id (FK show_runs CASCADE NOT NULL), slug,
  title, description, audience_title, audience_program_blurb, status (CHECK,
  7 values), scheduled_start_at, scheduled_end_at, actual_start_at,
  actual_end_at, created_by_user_id, created_at, updated_at, archived_at`.
  `UNIQUE(show_run_id, slug)`. CHECK constraints: status enum,
  archived/archived_at consistency (mirrors Kernel 66's pattern), and a new
  `scheduled_end_at >= scheduled_start_at` (when both set).
- **`sessions.show_id`**: nullable `ADD COLUMN IF NOT EXISTS ... REFERENCES
  shows(id) ON DELETE SET NULL`.

No venue seed, no Go-side bootstrap — Shows live inside the existing
`show-runs` venue.

## 3. Backend — `backend/internal/shows/`

- `types.go` — `Show`, `ShowSummary`, `ShowRunShowsSummary`,
  `CreateShowInput`, `UpdateShowPatch` (full `json` tags from day one —
  Kernel 66's reportback documents the missing-tags bug as a standing
  lesson).
- `shows.go` — `CreateShow`, `UpdateShow`, `ArchiveShow`, `LoadShowByID`,
  `ListShowsForRun` (returns a live/upcoming/completed bucket summary
  computed in Go).
- `sessions.go` — `LinkSessionToShow`/`UnlinkSessionFromShow`, manage-
  authorized, touch only `sessions.show_id`.
- `roster.go` — thin pass-throughs to `showruns.ListInternalRoster`/
  `ListAudienceProgramMembers` — no new roster logic.
- `http.go` — 6 routes (see §4).
- `shows_test.go` — dedicated-test-DB workflow.

**Authority reuse**: `backend/internal/showruns/authority.go`'s
`canManageShowRun`/`canViewShowRun` exported to
`CanManageShowRun`/`CanViewShowRun` (pure rename, ~16 in-package call sites
updated) so `shows` calls the same authority logic rather than duplicating
it — a deliberate exception to this codebase's usual small-helper
duplication convention, since authority logic drifting between two copies
is a real risk.

## 4. Routes

```
GET/POST  /api/show-runs/{show_run_id}/shows
GET/PATCH /api/shows/{show_id}
POST      /api/shows/{show_id}/archive
GET       /api/shows/{show_id}/program
POST      /api/shows/{show_id}/sessions/{session_id}/link
POST      /api/shows/{show_id}/sessions/{session_id}/unlink
```

## 5. Frontend

- `frontend/venues/show-runs/run.html` — new "Shows" card (list + bucket
  summary + manage-only inline create form).
- New `frontend/venues/show-runs/show.html` — detail/editor: title,
  backstage description, audience title/blurb, 7-value status select, two
  `datetime-local` inputs, and a manage-only Session Link row (plain
  session-ID text field — no session picker exists anywhere yet).
- New `frontend/venues/show-runs/show-program.html` — curated Audience
  Program view, Show's own title/blurb, same roster-card rendering as the
  Show Run's `program.html`.

## 6. Dictionary

`Construction/Dictionary.txt`: added a `Show (Show Instance)` entry, and
**corrected** the existing `Showing (live/runtime) vs. future scheduled
occurrence` note — it previously predicted scheduling would extend
`showings`; that was found unsafe, so it now documents that `shows` (not
`showings`) is the scheduling primitive going forward.

## 7. Tests and proof

Kernel 64 dedicated-test-DB workflow throughout. `backend/internal/shows/shows_test.go`
covers: location-scoped authority (positive/negative/Operator-bypass, same
pattern as Kernel 66); Show belongs to its parent Show Run and multi-Show
listing/bucketing; DB-level CHECK constraints (status enum,
archived/archived_at, scheduled-end-not-before-start); all scheduling
timestamps nullable; Audience Program is genuinely Show-specific while
excluding backstage fields; roster inheritance with zero Show-scoped roster
rows; session link/unlink including a mismatched-unlink no-op; anonymous
rejection. Full existing `internal/network`/`internal/actions` suites
re-run unchanged to confirm the new nullable `sessions.show_id` column
causes no regression.

`scripts/smoke/fresh-install.sh`: migration `040` added to the hardcoded
migration array (a glob it is not — the exact gap Kernel 66 hit); 8 new
assertions reusing the Kernel 66 block's fixtures.

## 8. Explicit exclusions

Scene Configuration, Capture Scene, Fly Scene, cue groups, attendance,
tickets, full calendar scheduling, Show-specific roster overrides, comments,
feeds, impressions, autographs, connection requests, circles, tags, daily
view/connection limits, social recommendation, public anonymous discovery.

## 9. PASS standard

Kernel 67 passes when: Shows belong directly to Show Runs; the user-facing
UI says Show; existing `/session`/`/mic` behavior is unbroken (verified via
full regression run, not assumed); a Show may span multiple Sessions via
the new nullable link; dates are optional; Audience Program is Show-specific
and curated; roster visibility inherits from Show Run without a second
roster system; the dictionary resolves Show/Showing/Session terminology
without contradiction; the dedicated test-DB workflow is followed; fresh
install passes; no deferred social-scarcity features are built.
