# Kernel 91 Report Back — Victory Campus Tours & Guided Onboarding

**Kernel spec:** `Kernel_91_Victory_Campus_Tours_and_Guided_Onboarding.md` (provided directly, not committed to `Construction/Kernels/`)
**Status:** PASS for the confirmed scope (see §2); PARTIAL against the full spec by deliberate agreement, not omission
**Date:** 2026-08-18
**Deployed:** live, `victory.amurray.family` — migrations 107 and 108 both applied at boot with the usual pre-apply backup

---

## 1. What this kernel is

Before this kernel, Victory taught a new user through two disconnected, unrelated pieces: a static one-shot "Welcome to Victory" modal on the campus map (`victory:map-onboarding-v1` in localStorage, no server memory, no spotlight, dismiss-only), and the Kernel 74/75 Catharsis character-creation wizard (`onboarding.js`) — a different concern entirely, walking a user through building a character rather than orienting them to a venue's live controls. There was no reusable "point at a real control, dim everything else, remember what this account has already been shown" system anywhere in the codebase, and no server-side record of onboarding progress outside the narrow, Character/Show-scoped `participant_tutorial_progress` table.

Kernel 91 builds that system: one shared frontend tour engine (dim/spotlight/arrow/click-gating), one backend package with server-authoritative role eligibility, and real content for the mandatory campus orientation (Audition Hall + Trailer), the skippable campus continuation (Catharsis + Greenroom), the Catharsis Cast venue tour, and the Director's Chair role-overlay tour — the kernel's own §42-43 acceptance proof (a Cast member who later becomes Director gets shown the Director toolbox without replaying what they already learned as Cast).

---

## 2. Scope confirmed with Grant before implementation

The full spec also asks for authored tour content for Producer, Crew, Operator, and Audience, matching the spec's own §54 pass criteria. Before writing code, Grant was asked and confirmed a narrower first pass: build the engine, schema, and API completely (so eligibility plumbing for every role already exists), but author real step content only for the mandatory tour, the campus continuation, Catharsis Cast, and the Director's Chair role-overlay — the spec's own explicit acceptance proof. Producer/Crew/Operator/Audience are eligible-but-empty: adding their content later is a migration (extending `tour_key`'s CHECK constraint) plus a new `Definitions` map entry, deliberately the same friction `tutorial.allMilestones` uses.

This matches the spec's own §50-52 boundaries, which explicitly defer deep Audience UX to Kernel 93 and deep Operator/slash-command training to Kernel 95.

---

## 3. What was built

**Database** — two migrations:
- `backend/migrations/107_kernel91_tour_completions.sql` — `tour_completions`, the durable completed/skipped record. `tour_key` is CHECK-constrained (4 values) so "which tours exist" is a schema fact, not a config toggle. Idempotency is a `CREATE UNIQUE INDEX` over `(user_id, tour_key, COALESCE(venue_slug,''), COALESCE(role_key,''))` — not a table-level `UNIQUE(...)`, which Postgres rejects when the key contains an expression; caught by the migration itself failing on first apply against the real test database, not by review.
- `backend/migrations/108_kernel91_tour_progress.sql` — `tour_progress`, a resumable cursor added **after** initial deployment, once Grant reported a real bug (see §5). Deliberately a separate table from `tour_completions` rather than a third status value on it: a cursor is mutable, overwritten-in-place state; a completion is an append-once historical fact, and conflating the two would have made `tour_completions`' own idempotency comment a lie.

**Backend** — new package `backend/internal/tour/`:
- `tour.go` — the 4 tour-key constants, `Subject` (session-derived identity only, never client-supplied), `RecordCompletion`/`HasCompletion`/`LoadCompletions`, `RecordProgress`/`LoadProgressStep`. `RecordCompletion` now also deletes any leftover `tour_progress` row for that scope on completion.
- `definitions.go` — the 4 authored `Definition`s as Go struct data (following `tutorial`'s own precedent — this codebase has no JSON-config-driven-content convention, and 4 rows didn't justify inventing one). Each `Step` carries a semantic `Target` string (`venue:audition-hall`, `control:director-console`, ...) resolved client-side; nothing DOM-specific lives here.
- `eligibility.go` — `EligibleTours`/`ResumeMandatory`, the one place role/venue eligibility is decided, using `access.CurrentLocationRoleForLocation`/`access.IsOperatorUser` (never a client-supplied role). `resumeFromProgress` slices a `Definition`'s `Steps` to start after the last `step_reached` in `tour_progress`, added in the follow-up fix.
- `http.go` — `GET /api/tours/state`, `GET /api/tours/{tour_key}/replay`, `POST /api/tours/{tour_key}/complete`, `POST /api/tours/{tour_key}/skip`, `POST /api/tours/{tour_key}/progress`, `GET /api/tours/history`. `complete`/`skip`/`progress` derive the user solely from the session cookie — there is no `user_id` field in any request body for them to forge.
- `backend/internal/venues/map.go` — `resolveVenueLocation` exported as `ResolveVenueLocation` so `tour` can resolve venue→location without duplicating the lookup.
- `backend/cmd/victory/main.go` — routes registered; GET routes bare (matching `/api/map/visibility`'s existing style, added to `kernel77_csrf_posture_test.go`'s allowlist), POST routes method-prefixed with `actionLimiter`.

**Frontend:**
- `frontend/lib/tour-engine.js` (new) — the one shared tour engine, `window.VictoryTourEngine`. Public API: `registerTargets`, `autostart`, `start` (replay), `isActive`. Renders four dimming rectangles framing the resolved target rather than one full-screen overlay (see §5 bug #1), attaches click-advance directly to the resolved target element, and posts progress/completion via `navigator.sendBeacon` from that same click handler (see §5 bug #3).
- `frontend/index.html` / `frontend/app.js` — old `#onboarding-overlay`/`maybeShowMapOnboarding()`/`ONBOARDING_KEY` removed outright (replaced, not run alongside); venue pins gained `data-venue-slug`; campus mandatory/continuation tours wired through `registerCampusTourTargets`/`maybeStartCampusTour`.
- `frontend/venues/catharsis/tour.js` (new) — Cast venue tour wiring, deliberately separate from `onboarding.js` (the Kernel 74/75 wizard), with a guard against autostarting while that wizard's overlay is open.
- `frontend/venues/directors-chair/index.html` — Director-toolbox role-overlay tour wired at the exact point the existing role gate already resolves Producer/Director/Operator.
- `frontend/account/index.html` — new Tours panel, `GET /api/tours/history`, Replay links using a `?replay-tour=<key>` convention (navigates to the tour's owning page rather than inventing a new Help-menu system).
- `frontend/styles.css` — new `.tour-*` rules only, no new breakpoints.

---

## 4. Evidence

### Backend tests

```bash
cd /opt/victory/backend
go build ./...
go vet ./...
TEST_DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory_test?sslmode=disable" go test ./...
```

Full suite green, `gofmt -l` clean on every touched file. New: 13 tests in `internal/tour/` (`tour_dbtest_test.go`, `http_dbtest_test.go`) — idempotent `RecordCompletion`, per-user/per-venue scoping, role-gating (Audience refused `catharsis_cast`, Cast granted it), Operator bypassing `location_role` entirely via `IsOperatorUser`, mandatory-tour skip rejection at the data layer, `campus_continuation` requiring `campus_mandatory` first, unauthenticated `/api/tours/state` → 401, a forged `user_id` in a `complete` request body structurally ignored (there is no field for it to land in), skip on a mandatory tour → 400, and (added with migration 108) `ResumeMandatory` correctly slicing past an already-reached step and `RecordCompletion` clearing the stale progress cursor.

### Deployment

```bash
docker compose build backend && docker compose up -d backend
```

Both migrations applied cleanly at boot with pre-apply backups (`victory_pre_migrate_20260818_044041_2pending.dump`, `victory_pre_migrate_20260818_061959_1pending.dump`); Discord gateway reconnected normally; `/health` and `/api/tours/state` (401 unauthenticated, correct) verified live over the real domain after each deploy. Frontend is served directly from `/opt/victory/frontend` via the shared Caddy bind mount (`/opt/bread-exchange/`), so JS/CSS/HTML fixes are live the moment the file is saved — no build or restart step, confirmed by re-fetching `/lib/tour-engine.js` and `/styles.css` over the real domain after each edit.

### Browser proof

Not run as a scripted Playwright suite this pass. Instead, real production usage by Grant surfaced three real bugs (below) that a scripted proof against a fresh throwaway account likely would not have caught as clearly, since two of them only manifest when a click-gated step's target causes a real page navigation mid-tour. All three are fixed and re-verified live.

---

## 5. Bugs found in production, not in review

**Bug 1 — the dimming overlay physically blocked every click, tour-active or not.** `.tour-scrim` was a single full-viewport `<div>` at `z-index: 200`. The browser's own hit-testing resolves clicks to whatever element is physically on top at that point — since the scrim sat over the entire page, `event.target` for any click was the scrim itself (or one of its children), never the real venue pin underneath, no matter what click-gating logic ran afterward. Grant's report: "I can't click anything on the map." Fixed by replacing the single scrim with four dimming rectangles that frame the target and leave it structurally unclipped — a real hit-testable hole in the DOM, not a visual illusion over an intercepting element.

**Bug 2 — the position-tracking `MutationObserver` triggered itself in an infinite loop.** It watched `attributes: true` on the same `document.body` subtree that `positionOverlay()` was writing inline `style.top/left/width/height` onto (the scrim is a child of `body`). Every reposition produced a mutation record, which re-triggered the observer, which repositioned again. Visible to Grant as a uBlock Origin "this page is slowing down your browser" warning — exactly the runaway-script signature that heuristic looks for. Fixed by dropping `attributes` from the watched mutation types (kept `childList`/`subtree`, still catching real structural page changes), ignoring mutations that originate inside the tour's own overlay, and coalescing repositions to at most one per animation frame.

**Bug 3 — a click-gated step whose target causes real navigation raced an ordinary `fetch()` and lost.** After fixing bugs 1 and 2, Grant reported the tour "stuck in a loop of resetting to Audition Hall": clicking the Audition Hall pin correctly advances the local tour state to the Trailer step *and* triggers the pin's own `window.location.href` navigation to `/venues/audition-hall/` — the ordinary `fetch()` call recording that progress could be, and was, cancelled by the page unload before it landed. Nothing was ever persisted, so every return to the map restarted the mandatory tour from step 0. Same failure mode was latent on the Trailer step too, since it's also a navigating pin and also the tour's last step (so `finish()`'s own completion `fetch()` was equally exposed). Fixed with `navigator.sendBeacon` — purpose-built for exactly this, guaranteed by the browser to deliver even if the page unloads immediately after the call returns — plus the new `tour_progress` table (migration 108) and `resumeFromProgress`, so `EligibleTours`/`ResumeMandatory` now return only the steps after the last one actually reached, instead of always the full list.

All three are recorded as reusable lessons in `operator-notes.md`.

---

## 6. Known gaps, honestly stated

- **No authored tour content for Producer, Crew, Operator, or Audience** — by agreed scope (§2), not oversight. The eligibility plumbing (`RequiredRoles` already accepts any `location_role` value plus the `"operator"` sentinel) needs no further backend work to add them; each is a migration extending the `tour_key` CHECK plus a `Definitions` map entry.
- **No scripted browser/Playwright proof committed for this pass.** The kernel's core acceptance claims (mandatory tour, campus continuation, Director role-transition) were exercised live by Grant rather than by an automated harness; the three bugs above were found and fixed through that real usage. A follow-up scripted proof (matching the spec's §39-43 required scenarios, including the Director role-transition case using the `k89fixture`/`k90fixture` role-granting pattern) would be the natural next-step evidence gap to close.
- **No dedicated `Construction/Operations/*.md` operator guide was written** for this kernel, unlike Kernel 89/90 — the reportback's §3/§5 are intended to carry that role for now.

---

## 7. How to run from a clean state

1. Ensure `.env` has `POSTGRES_PASSWORD` set.
2. `docker compose up -d postgres` (already running in production).
3. `docker compose build backend && docker compose up -d backend` — migrations 107/108 apply automatically at boot with a pre-apply `pg_dump` backup to `/opt/victory/backups/`.
4. Frontend needs no build step — `/opt/victory/frontend` is bind-mounted directly into the shared Caddy container at `/srv/web2`.
5. Sign in, visit the campus map — the mandatory tour should autostart for any account with no `campus_mandatory` row in `tour_completions`.

---

## 8. Files changed or created

**Backend:** `backend/migrations/107_kernel91_tour_completions.sql`, `108_kernel91_tour_progress.sql`; `backend/internal/tour/` (new package: `tour.go`, `definitions.go`, `eligibility.go`, `http.go`, `tour_dbtest_test.go`, `http_dbtest_test.go`); `backend/internal/venues/map.go`; `backend/cmd/victory/main.go`; `backend/cmd/victory/kernel77_csrf_posture_test.go`.

**Frontend:** `frontend/lib/tour-engine.js` (new); `frontend/app.js`; `frontend/index.html`; `frontend/venues/catharsis/tour.js` (new); `frontend/venues/catharsis/index.html`; `frontend/venues/directors-chair/index.html`; `frontend/account/index.html`; `frontend/styles.css`.

**Docs:** this reportback; `operator-log.md`; `operator-notes.md`; `current-state.md` top section.

---

## 9. Required project-memory updates completed

- [x] Reportback saved in repository (this file)
- [x] `operator-log.md` appended
- [x] `operator-notes.md` updated (three-bug lesson)
- [ ] `kernel-maker-field-guide.md` — not updated; no repo-layout/test-command/runtime-mode change this kernel introduced
- [ ] `dev-workflow.md` — not updated; no startup/port/service/migration-step change beyond the two new migration files, which follow the existing convention exactly
- [x] Master/current-state guide updated (`current-state.md` top section, `Victory_Canonical_Roadmap_v2.md` dated note)
- [x] Fresh-install/bootstrap migration list — automatic (`//go:embed *.sql`), no manual step
- [ ] Help/command documentation — not applicable, no new slash command or Director-facing command surface

---

## 10. Next recommended step

A scripted Playwright proof matching the spec's §39-43 required scenarios, particularly the Director role-transition acceptance proof (grant Director access mid-session via a fixture, confirm the toolbox tour becomes eligible without replaying the Cast tour) — the one claim in this kernel that has been exercised by real usage but not by an automated, repeatable check. After that, authoring Producer/Crew/Operator/Audience tour content is a small, mechanical follow-up given the eligibility plumbing already in place.
