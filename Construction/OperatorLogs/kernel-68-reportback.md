# Kernel 68 Reportback — Venue Visibility Gates, Stage Management Surface, and Production Onboarding

**Status: PASS** (2026-07-13). Deployed live (migration 041 applied, backend rebuilt, `/health` OK). Changes left uncommitted for operator review, same as Kernels 62–67.

## 1. Readiness rule implemented

`playerprofile.TrailerFaceReady(ctx, pool, userID)` (`backend/internal/playerprofile/readiness.go`) — **computed, not a durable marker**. It reuses the existing Kernel 61 `ProjectTrailerFace` projection (already visible-only, via `BuildTrailerFace` → `VisibleSortedFields`) rather than adding a new `face_ready_at` column: since a Player Workbook fact only ever exists if the owner explicitly committed it (append-only event-sourced facts, no auto-populated defaults), field-presence is a reliable signal on its own.

Ready requires **both**:
- a stage name (`player_stage_name_history`, via `CurrentStageName`);
- at least one visible Face field (`len(face.Regions[...]) > 0` across all regions — already visibility-filtered).

`reason_code`: `face_ready`, `missing_face_commit` (no stage name), `missing_visible_face_field` (stage name set, zero visible facts), `not_authenticated`. Exposed at `GET /api/player-profile/me/face-readiness`.

## 2. Third Place visibility — a real fix, not just a new gate

Audited `access.ResolveVisibleVenues` before touching it: **Trailers and Third Place were already both gated to `producer/director/cast/crew` location role**, identical rules, since before this kernel. A brand-new self-signup account gets `location_memberships.role = 'audience'` with zero friction (`identity/auth.go:HandleSignup`) — meaning under the old rule, a plain new account could never see *either* tile, and the required test "new account sees Trailers" could not pass without also touching Trailers' own rule.

Confirmed via the fixture scripts (`fresh-install.sh`) that the actual `/api/third-place/headshots*` API has never required a performer role — only authentication — so the map-visibility role restriction was a stale carryover from the old Trailers-only rule, not a deliberate Third Place design choice. **Operator decision, confirmed via AskUserQuestion before implementing: readiness *replaces* the role gate for Third Place; Trailers itself is now open to any authenticated user** (moved into the same `authenticated_surface` bucket as `audition-hall`).

Implementation: `access.SetThirdPlaceReadinessChecker` — a callback injected from `main.go` at startup (same avoid-import-cycle pattern as `playerprofile.ProjectionChangeNotifier`; `access` sits below `playerprofile` in the import graph, so `access` cannot import it directly). `ResolveVisibleVenues` appends the `third-place` venue row with `visible_because: "trailer_face_ready"` only if the checker returns true; Operator bypass is unaffected (already sees every venue).

Direct API access: `thirdplace.HandleMe`'s **POST** (leave/place a Headshot — the actual "enter Third Place" action) now calls `TrailerFaceReady` first and returns a clean `403 {"error":"trailer_face_not_ready"}` if unready. GET/DELETE and the commons list/history stay open to any authenticated user regardless of readiness — nothing not-yet-ready is ever placed into the commons, so there's nothing to protect there.

## 3. Stage Management visibility gate

Renamed **user-facing only** — internal slug, routes, and Go package all remain `show-runs`/`backend/internal/showruns`, per the spec's own instruction to avoid churn. Map tile label, page `<title>`s, and eyebrow breadcrumbs across `index.html`/`run.html`/`roster.html`/`show.html` now read "Stage Management"; the two audience-facing pages (`program.html`, `show-program.html`) were deliberately left unchanged — an Audience member reaching them via a direct/curated link has no reason to see backstage-coded language.

Visibility rule (`access.ResolveVisibleVenues`): **Operator, OR active `producer`/`director` location role at the venue's location, OR an active `crew` roster row on any Show Run at that location** (`show_run_crew_surface` reason). Plain `audience`-role membership — which previously cleared this bar under Kernel 66's "Audience gets the best seats" rule for the *tile* — no longer does. That rule's actual purpose (curated Audience Program access) is preserved separately: it never depended on map-tile visibility.

Backstage API tightened to match, so a hidden tile can't be worked around by hitting the API directly: added `showruns.CanViewBackstage` (`CanManageShowRun` OR active crew roster row — visibility only, **not** manage authority) and swapped it in for `CanViewShowRun` on the two backstage-only reads: `HandleByID` GET (`/api/show-runs/{id}`) and `HandleShowRunShows`/`HandleShowByID` GET in the `shows` package (`/api/show-runs/{id}/shows`, `/api/shows/{id}`). `ListShowRunsVisibleToUser`'s underlying SQL was updated the same way. **Left untouched**: `CanViewShowRun` itself (still any active membership — it's what the Audience Program route, `HandleAudienceProgram`/`HandleShowProgram`, and self-join continue to use), and `HandleRosterCollection`'s internal roster view (stays Producer/Director/Operator-only, no crew exception — crew get backstage-listing visibility, not full roster access).

## 4. Crew editor: visibility implemented, edit authority deliberately deferred

Per §1.5's own permission to defer: implemented the **visibility-only** half (crew roster row unlocks the Stage Management tile and the backstage listing/detail reads, both zero-schema-change reuses of the already-existing `show_run_roster_members.role = 'crew'` enum value). **Did not** implement crew edit authority (PATCH-level access to non-destructive Show/Show Run fields) — doing that safely would require splitting `CanManageShowRun` into a full-authority check (archive/block/appoint — must stay Producer/Director/Operator-only) and a narrower crew-safe check threaded through every mutating handler individually, which is exactly the "full permission matrix" shape the spec says to avoid building. Documented here as the explicit deferral the spec asks for; a future kernel could add a `CanEditShowRunNonDestructive` check scoped to the specific PATCH fields (title/description/audience blurbs) if this becomes a real ask.

## 5. Production creation route/UI

Audited first: `backend/internal/identity/permissions.go` had exactly one route, `HandleListProductions` (GET-only), and zero `INSERT INTO productions` anywhere outside tests/fixtures/the smoke script — confirming Kernel 66's own documented gap. Added:

- `backend/internal/identity/permissions.go`: `HandleProductionsCollection` (GET/POST dispatch, same pattern as `showruns.HandleCollection`), `handleCreateProduction`. Authority: Operator (any location, resolved via an optional `location_slug` input — still resolved to an id server-side, never trusted verbatim), or Producer/Director creating for their own resolved location (`resolveInviteAuthorityScope`, unchanged). Slug: client-supplied or derived from name via a small local `slugifyProductionCandidate`; unique-constraint violation returns a clean `409 slug_already_used`.
- `database/migrations/041_kernel68_productions_created_by.sql`: additive `productions.created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL` — the table never had this column even though the spec's minimum-fields list calls for it.
- `frontend/venues/show-runs/index.html`: a "Create a Production" card, shown exactly when the Production picker would otherwise be empty (`canCreate` false), with Name/Slug inputs and a button that POSTs, then re-fetches the picker and reveals the Show Run creation form once a Production exists.

## 6. Cloud/fog implementation

Per §3.4's own allowance for a simpler model: a base radial-gradient fog layer (`#map-fog-base`) whose opacity recedes as the *count* of currently-visible venues grows (`Math.max(0.15, 0.55 - venues.length * 0.02)`), plus **targeted** cloud puffs for the two gateable venues (`third-place`, `show-runs`) positioned exactly at their known fixed map coordinates (`fallbackPositions`, unchanged whether the venue is visible or not) — each puff is present with `opacity: 1` while its slug is absent from the server's visibility response, and gets a `--receded` (opacity 0, CSS-transitioned) class once the slug appears. No venue name is ever rendered in the fog layer (it's decorative gradients only, `pointer-events: none`, z-index below the pin layer) — nothing hidden is named, and nothing visible is ever obscured, so there's no broken click target under the cloud by construction (a hidden venue has no pin to click in the first place).

## 7. Tests and proof

- `go build ./...` / `go vet ./...` clean.
- `git diff --check` clean.
- New tests: `playerprofile/readiness_test.go` (5 cases: not-authenticated, brand-new-not-ready, stage-name-alone-not-enough, stage-name-plus-field-is-ready, hidden-field-doesn't-count), `access/visibility_dbtest_test.go` (4 cases: Trailers open to plain audience, Third Place gated on injected readiness checker true/false, Stage Management backstage-only vs. Producer, crew roster exception), `identity/productions_test.go` (4 cases: anonymous 401, audience-only 403, producer 200 + appears in list, blank name 400), `showruns/showruns_test.go` new case `TestCanViewBackstageExcludesPlainAudienceButIncludesCrew`, `thirdplace/http_test.go` new case `TestHandleMePostRejectsUnreadyTrailerFace` (403 → ready → 200) plus fixed two pre-existing tests that had never set up a Trailer Face (see Known Issues).
- Full `go test -count=1 ./...` green with `TEST_DATABASE_URL` set. One flake (`TestDiscordMicRegisterAndStatusDispatch`, unrelated Discord dispatch timing) reproduced once, then passed 3/3 in isolation and on a full-suite re-run — same class of pre-existing flake Kernel 67 documented, not chased further.
- Confirmed hard-fail without `TEST_DATABASE_URL` for the new `access` DB test.
- `scripts/smoke/fresh-install.sh --local`: migration 041 added to the hardcoded array (applying Kernel 66's own documented lesson about it not being a glob), extended with 11 new assertions, full run PASS end-to-end from an empty database. New assertions cover: Face-ready Producer A sees both Third Place and Stage Management; Face-unready Audience-only B sees Trailers but neither gated venue; direct Third Place POST 403 for unready; direct Stage Management backstage-detail 403 for Audience-only; curated Audience Program still reachable for that same Audience-only user; Create Production rejects anonymous; Create Production succeeds for Producer and the new Production is immediately usable to create a Show Run.

## 8. Visual proof

No `chromium-cli`/headless browser tooling available in this environment (same gap Kernels 66/67 flagged). Manual checklist for the operator:

1. Sign up a brand-new account, do not touch Trailer Face → map shows Trailers tile, no Third Place tile, heavier fog puff sits over Third Place's map position (upper-right area near Trailers).
2. Visit `/venues/trailers/face.html` while unready → a banner reads "Set your Trailer Face to enter Third Place."
3. Set a stage name (Account or My Face → Edit) and add one visible Workbook fact (e.g. Short Introduction) → reload the map → Third Place tile appears, its fog puff is gone; the Trailers banner now reads "Third Place unlocked."
4. As that same plain-audience account, confirm the Stage Management tile is absent (fog puff present near the top of the map).
5. As a Producer/Director (or via `victory-bootstrap`), confirm the Stage Management tile is visible, labeled "Stage Management," and its fog puff is gone.
6. Open Stage Management with no Production yet at that location → "Create a Production" card appears instead of the Show Run form; create one → the Show Run form appears with the new Production selectable.

## 9. Known issues

- **Two pre-existing `internal/thirdplace` HTTP tests never actually set up a ready Trailer Face** (`TestHandleMeFullLifecycle`, `TestHandleMeIgnoresClientSuppliedUserID`) — they relied on the POST endpoint having no readiness gate at all. Fixed by supplying a visible field via the existing `setStageNameAndFace` test helper's optional `shortIntro` parameter (previously called with blank strings). Not a regression in behavior, just a test fixture that needed to catch up to the new gate it's exercising.
- Crew edit authority (PATCH-level) remains deferred per §1.5/§3.8 — see §4 above for the exact reasoning and what a follow-up would need to do.
- No screenshot evidence — see §8.

## 10. Files changed

**New:**
- `backend/internal/playerprofile/readiness.go`, `readiness_test.go`
- `backend/internal/access/visibility_dbtest_test.go`
- `backend/internal/identity/productions_test.go`
- `database/migrations/041_kernel68_productions_created_by.sql`
- `Construction/Kernels/kernel-68-venue-visibility-stage-management-production-onboarding-v0.1.md`
- `Construction/OperatorLogs/kernel-68-reportback.md` (this file)

**Modified:**
- `backend/cmd/victory/main.go` (readiness checker wiring, new route)
- `backend/internal/access/visibility.go` (Trailers/Third Place/Stage Management rules, readiness checker injection point)
- `backend/internal/identity/permissions.go` (Create Production)
- `backend/internal/playerprofile/http.go` (`HandleFaceReadiness`)
- `backend/internal/showruns/authority.go` (`CanViewBackstage`), `http.go`, `showruns.go` (listing SQL), `showruns_test.go`
- `backend/internal/shows/http.go` (backstage gating)
- `backend/internal/thirdplace/http.go` (readiness gate on POST), `http_test.go`
- `Construction/Canon/Dictionary.txt` (Trailer Face Ready, Stage Management, Third Place unlock rule)
- `frontend/app.js` (Trailers-open rule effects on menu grouping, display-name override, map fog)
- `frontend/index.html`, `frontend/styles.css` (fog layer)
- `frontend/venues/show-runs/index.html` (Stage Management labels, Create Production UI), `run.html`, `roster.html`, `show.html` (labels only)
- `frontend/venues/trailers/face.html` (unlock prompt)
- `scripts/smoke/fresh-install.sh` (migration array, new fixtures/assertions)

## 11. Next recommended step

Scene Configuration Model / Capture Scene — the roadmap has pointed here for three kernels running, and now has a Show container, a gated Third Place, and a labeled backstage surface all in place. A follow-up crew-edit-authority kernel (see §4) and the older stale-test-user sweep remain smaller, independent candidates.
