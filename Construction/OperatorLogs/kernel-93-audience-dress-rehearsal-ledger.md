# Kernel 93 — Audience Seat Dress Rehearsal Ledger

**Kernel spec:** `Construction/Kernels/Kernel 93 — Audience Seat Dress Rehearsal.md`
**Status:** Setup complete, acceptance gate reached. STOPPED per spec §0/§34 — handing control to Grant for the live passes (A–D). Not autonomously continued past setup.
**Show under test:** JG5X (per Grant, 2026-08-19), venue Catharsis.

---

## 0. Repository audit (spec §33)

Delegated to two research passes before writing any code. Findings that shaped the design:

- **No existing single-Showing Audience admission primitive.** `show_run_tickets` (Kernel 71's two-punch ticket) is Show-Run-scoped, `requested_role` CHECK-locked to `'player'`, and its own migration comment anticipated-but-didn't-build a future Audience widening. Deliberately did **not** widen it — Kernel 93 needs Showing scope (one live Showing), which a Show-Run-scoped table can't express without conflating the two systems (spec §2). Built `audience_admissions` as a new, clearly separate table instead.
- **The actual K90 blocker:** `access.ResolveVisibleVenues`'s Catharsis branch only admits `location_memberships.role IN ('producer','director','cast','crew')`. A plain `audience`-role account with zero location memberships cannot see the Catharsis tile at all, and both the page-load snapshot route and the `/api/session/catharsis/join` route gate on this same function — so this one query is the entire chokepoint for entry.
- **Once past that gate, the rest of the pipeline already works correctly for a pure Audience account with no location membership:** `identity.JoinVenue`'s `resolveRoleForUser` defaults to `"audience"` when no location role is found; `stageobjects.ResolveViewer`'s `Audience = !cast` is already the correct complement regardless of how the viewer got in; `session_participants.role` (not the participation resolver) is what actually reaches the live WS session. This meant the fix could stay small instead of touching the whole role-resolution stack.
- **Kernel 92 Showtime's own selector** (`sessions WHERE show_id = $1 AND status IN ('rehearsal','live') ORDER BY started_at DESC LIMIT 1`) is what "the currently live Showing for a Show" already means everywhere else in this codebase (showtime.Start/Status/End). Every new endpoint reuses this exact query rather than inventing a second live-event selector (spec §25).
- **rollaudience.ModeShow already returns true unconditionally** — i.e. current public dice behavior is "on" — which is why the new Audience Dice Rolls toggle defaults ON (spec §5).
- **No Audience tier exists in `socio.ResolveTier`** — Socio's own doc comment says Audience-facing output is meant to come from "the existing stage/Cave projection pipeline," not the Player/Director mechanical query surface. No qualitative per-character health data pipeline reaches any client today.
- **Presence has no Audience-safe read endpoint** — the existing preview UI reads `/api/director-console/current`, which is Director/Producer-only. No fork of the Presence system was built to work around this (spec §17); see the open item below instead.
- **Reactions (`react/emote`) are fully wired server-side** (`actions/react.go`, `network/ws.go`) but have **zero frontend caller or renderer** anywhere in the repo today. Out of scope for this setup pass — recorded as a Pass B/FUTURE FEATURE candidate, not built here.

---

## 1. Setup work (pre-Pass-A)

| ID | Area | Decision / Action | Files | Status |
|----|------|--------------------|-------|--------|
| S1 | Schema | New `audience_admissions` (user_id, showing_id, issued_by, issued_at) and `audience_projection_configs` (showing_id PK, three booleans, updated_by/at) tables. Kept structurally separate from `show_run_tickets`. | `backend/migrations/109_kernel93_audience_admission.sql` | Done |
| S2 | Backend | `audienceadmission` package: `IssueForShowCode`/`ListForShowCode`, resolves Show by short code → Show-Run authority (`showruns.CanManageShowRun`) → current live/rehearsal Showing via Showtime's own selector → upserts admission by target handle (handle-paste, matching `storyboards/grants.go`'s established no-user-search convention). | `backend/internal/audienceadmission/*.go` | Done |
| S3 | Backend | `audienceprojection` package: `Config` (dice/presence/health booleans), `ForShowing`/`ForShowCode`/`UpdateForShowCode`, same Showtime-selector + authority pattern. Defaults: dice **ON** (matches current public behavior), presence **OFF**, health **OFF** (Grant's explicit answers, 2026-08-19). | `backend/internal/audienceprojection/*.go` | Done |
| S4 | Backend | Venue-gate fix: added a narrow `audience_admissions`-scoped UNION branch to `access.ResolveVisibleVenues`'s Catharsis query, and a matching Step 3.5 in `participation.ResolveParticipationContext`. Both scoped through the admission's own Showing's still-live session — never a blanket venue grant. | `backend/internal/access/visibility.go`, `backend/internal/participation/resolver.go` | Done |
| S5 | Backend | Confirmed (no code change needed) that `stageobjects.ResolveViewer`'s Cast-complement logic and `identity.JoinVenue`'s role fallback already handle a pure Audience account correctly once S4 lets them past the gate. | — | Verified, no change |
| S6 | Backend | Audience Dice Rolls enforcement: gated Show-mode roll visibility (snapshot re-fetch, pinned-effect reconnect, **and live broadcast delivery**) on the Showing's `show_dice_rolls` toggle, for Audience viewers only, always excluding nothing for the roller's own roll. Three call sites (`world/snapshot.go`, `network/ws.go` ×2) each carry a small local duplicate of the config read rather than importing `audienceprojection`, to avoid an import cycle (`network`→`audienceprojection`→`shows`→`network`). | `backend/internal/world/snapshot.go`, `backend/internal/network/ws.go` | Done |
| S7 | Backend | Embedded the resolved `AudienceProjectionConfig` directly into `world.Snapshot` (`audience_config` field) so any client already fetching the venue snapshot gets the current toggle state for free, no second authorized fetch needed. | `backend/internal/world/snapshot.go` | Done |
| S8 | Frontend | Director UI: three toggle buttons (Dice Rolls / Presence / Health Statuses, `chip` button convention, `aria-pressed`) plus a handle-input "Admit" control, added to the existing Showtime card (`show.html`) rather than the legacy `directors-chair` Director Console — confirmed that console is hardcoded to `"the-cave"` venue's session resolution and cannot see Catharsis Showings at all. | `frontend/venues/show-runs/show.html` | Done |
| S9 | Frontend | Audience overlay: new `#audience-drawer` in `catharsis/index.html`, structurally matching the existing left-drawer's drawer/rail language but simpler (Diagnostic / Presence / Health Status only, no Director/Cast controls). New standalone module `kernel93-audience-overlay.js` (independent of runtime.js's internals, own `/api/world/catharsis` poll) hides the Cast/Director drawers and shows this one when the server-resolved `theater_context.kind === "audience"`. | `frontend/venues/catharsis/index.html`, `frontend/lib/stage-runtime/kernel93-audience-overlay.js` | Done, with an open gap (S10) |
| S10 | Frontend | **Open gap, disclosed rather than papered over:** Presence and Health Status overlay sections currently show a static "this is on for the Showing" note, not live per-connection Presence data or live per-character qualitative health. Wiring those needs (a) a Presence read path that doesn't require Director authority, and (b) deciding which characters' health Audience should see and how — both are Pass A/B-shaped product questions (spec §13, §17) better answered after Grant is actually watching, not guessed here. | — | **Deferred to live pass** |
| S11 | Verification | `go build ./...` and `go vet ./...` clean across the whole backend after every change. | — | Done |
| S12 | Verification | **Could not run the DB-backed Go test suite in this environment** — no Docker/Postgres access in this sandbox (`TEST_DATABASE_URL` unavailable). Pre-existing `dbtest`-tagged tests fail with a clear "TEST_DATABASE_URL is required" message, not a regression from this work. Recommend running `TEST_DATABASE_URL=... go test ./...` (see `Construction/kernel-maker-field-guide.md`) before or during the live dress rehearsal. | — | **Not run — needs a real Postgres** |

---

## 2. Setup acceptance gate (spec §34) — self-assessment

1. A Showing exists — yes, once Grant runs Showtime for JG5X.
2. Director can issue/assign Audience admission — yes, S2 + S8 (handle-paste on the Show page).
3. Audience account can enter the correct venue — yes, S4 (needs live verification — see §3 below).
4. Audience account is truly resolved as Audience — yes, S4/S5 (`theater_context.kind == "audience"`, `session_participants.role == "audience"`).
5. Audience sees no Director/Cast-only controls — yes structurally, S9 (Cast/Director drawers are hidden outright for Audience, not CSS-dimmed).
6. Director can toggle Dice/Presence/Health Status — yes, S3 + S8.
7. Audience projection changes accordingly — **Dice: yes (S6, real server-side enforcement, including live broadcast). Presence/Health: toggle persists and gates overlay visibility, but no live data is streamed yet (S10).**
8. Presence begins collapsed if enabled — yes, S9 (native `<details>`, closed by default).

Items 1–6 and dice-in-7 and 8 are real. Presence/health-in-7 is a known, disclosed partial — see S10.

---

## 3. What I could not verify myself

No browser and no live database in this environment, so none of the following has actually been exercised end to end:

- Signing in as an admitted Audience account and confirming the venue actually opens (the SQL is written and mirrors working query shapes elsewhere, but is unexecuted).
- Confirming a Show B admission genuinely cannot enter Show A's Showing live (§28's core security proof) beyond reading the join logic by eye.
- Any visual/UX judgment about the overlay, toggle placement, or whether this is pleasant to watch — that is explicitly Grant's call per spec §0.

This is exactly what Pass A is for. Recommend Grant runs the DB-backed test suite first if convenient, then the live setup acceptance gate (§34) as the very first live action, before starting Pass A.

---

## 4. Decisions recorded (not to be re-asked)

- Test target: a new/dedicated environment is not being stood up — Show **JG5X** is the live target (Grant, 2026-08-19).
- Audience Presence toggle default: **OFF** at Showing creation (Grant, 2026-08-19).
- Audience Health Statuses toggle default: **OFF** at Showing creation (Grant, 2026-08-19).
- Audience Dice Rolls toggle default: **ON**, derived from repository evidence (current public `ModeShow` behavior), per spec §5's instruction to prefer existing behavior over asking.
- (Carried from the kernel spec §32, unchanged.)

---

## 5. Pass A–D

Not started. Per spec §0, this agent stops here and hands control to Grant. Entries below will be filled in live.

| ID | Pass | Observation | Classification | Decision | Action | Status | Files changed | Retest result | Deferred kernel |
|----|------|-------------|-----------------|----------|--------|--------|----------------|----------------|------------------|
| — | A | *(pending Grant's live session)* | | | | | | | |
