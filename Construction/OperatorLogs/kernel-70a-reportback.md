# Kernel Report Back — Kernel 70A: Live Stage Closure and Alpha Path Alignment

## Status

**PASS** — completed and committed as `2611840` on 2026-07-16, then **deployed live to production** the same day. `current-state.md` records the live deploy details under Kernel State.

## Delivered

- Server-side `shows.StartShowSession` (`backend/internal/shows/sessions.go`) + `POST /api/shows/{show_id}/sessions/start` + `show.html` UI — starts/resumes a venue Session and links it to a Show, with no manual Session ID pasting. Handles `venue_busy_with_other_show` for reattach confirmation.
- Proved Show-owned stage state (current Scene, variables) survives a full session end/resume cycle at Catharsis with no rebuild step.
- Fixed `go_to_scene`/`set_show_variable` to work with no active session; `emit_game_event` now fails cleanly and visibly instead of silently.
- Fixed `emit_game_event`'s hardcoded `"the-cave"` venue coupling via `identity.ResolveSessionVenueSlug` — now works at any venue, including Catharsis.
- Fixed `world.LoadVenueSnapshot`'s NULL-scan crash (COALESCE + nullable `*time.Time`) so an idle venue returns a graceful empty snapshot instead of erroring.
- New `Snapshot.TheaterContext{Kind, Message}`, computed server-side with exact required strings for `venue_open`, `participant` (no character), `audience`, and `backstage` (no message). Priority: registered player roster role > backstage-tier viewer role > plain audience.
- Closed an Audience-facing leak: the Rehearsal banner and Cue buttons in both venues' `runtime.js` were rendering regardless of viewer role; now gated on `normalizeRole(currentRole) === "audience"`.
- First Theater got its own independent, real venue wiring (`/api/world/first-theater`, `/api/session/first-theater/join`, `/ws/first-theater`), additive only — the-cave's routes untouched. First Theater remains investor/demo-only.
- Character-to-Show linkage audit (docs only, no schema change): confirmed no durable Character↔Show/Show-Run/Session FK exists; recommended attachment point for a future kernel is a nullable `character_card_id` on `show_run_roster_members`.
- Honest relabeling: "Scene Setup", "Edit Scene Details" (replaced a bare `prompt()` with an inline form), "Update Base Scene", "Save to This Show", "Cue Setup".
- `tests/catharsis/` — full mirror of `tests/first-theater/`'s 13 test files, same 46-pass/9-known-dice-failure shape. New `tests/contract/scene-nodes.contract.test.js` proving shared `scene-nodes.js` behavior between venues, explicitly excluding Catharsis's extra token-aura layer (documented, accepted drift).
- `scripts/test/alpha-gate.sh` added as the orchestrating gate; passes end-to-end.

## Bugs Found During Live Deployment And Verification

- **Kernel 70's own migrations had never reached production.** Migrations 043–045 (`actions.show_id`, `cues`, `cue_executions`) were committed in `920aeb7` on 2026-07-14 but never applied to the live `victory` database. The live backend was silently erroring on every `/ws/catharsis` snapshot (`column a.show_id does not exist`) for two days until this deploy. Fixed as part of the Kernel 70A deploy: took a `pg_dump` backup (`backups/victory_pre_kernel70a_migrations_20260716_031420.dump`, gitignored), replayed all 46 migration files against live `victory` (idempotent, `ON_ERROR_STOP=1`, zero errors), rebuilt and restarted the `victory-backend` container. Confirmed clean post-deploy via `/health`, `/ws/catharsis`, and `/api/director-console/current`. **Lesson recorded in `current-state.md`**: committed is not deployed in this repo — verify migrations actually reached production as part of closing out a kernel.
- **Two independent, non-interchangeable membership tables both claim to answer "what is this user's role."** `location_memberships` (Kernel 66+, used by Show-Run/Cue authority) vs. `memberships` (older, venue-scoped, used only by `main.go`'s `lookupVenueRole`, which feeds `viewerRole` into `world.LoadVenueSnapshot`). A real Producer with only `location_memberships` gets `viewerRole = "none"` at their own venue. Reaching Catharsis's live snapshot at all also requires a separate `access_grants` row. This is a genuine pre-existing gap, not caused by Kernel 70A — discovered during manual live verification. **Not fixed here** (out of approved scope; needs a real authority-model unification). Flagged for Kernel 71+.

## Validation

- `go build`/`go vet`: clean. `scripts/test/alpha-gate.sh`: PASS end-to-end (exit 0).
- No browser automation tooling (chromium/playwright) exists in this environment and no project run-skill was found at the time, so verification was a full manual Director/Player/Audience checklist driven directly over HTTP against a real, running, `go build`-compiled backend instance: signup → bootstrap producer → Production/ShowRun/Show/Scenes/Cue → StartShowSession → GO → end → resume → confirm persistence; separate Player/Audience accounts confirmed `theater_context` strings; confirmed a forged Cue execution returns 403.
- Rendered-UI/visual proof (banners, buttons actually painting correctly in a browser) was **not** captured and remains unverified — flagged for a spot-check.

## Explicitly Deferred

- The first playable Socio show.
- A casting/attendance system.
- Any participant-facing First Theater work (remains investor/demo-only).
- Unifying `location_memberships` and `memberships` into a single authority model (see bug above).
- Browser screenshot proof of Kernel 70/70A's GO and Start Show Session controls.
- Repairing the nine pre-existing frontend dice-test failures (now duplicated in both `tests/first-theater/` and `tests/catharsis/`).

## Next Recommended Step

Visual Scene composition/capture: define how reusable Base Scene content and a Show Placement's overrides bind to the existing stage-object/action model without creating a second renderer authority system. Before or alongside that, resolve the `location_memberships`/`memberships` split — it's a live authority bug, not just tech debt, and it will keep surfacing as more venues gate on `viewerRole`.
