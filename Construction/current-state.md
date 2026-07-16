# Victory Current State

## Purpose

This document is the current-state canon for Victory as of **Kernel 70A**. Historical kernel specifications and reportbacks describe what was true when they were written; this file wins when an older current-tense statement conflicts with the implemented repository.

For detailed vocabulary use `Construction/Dictionary.txt`. For durable implementation traps use `Construction/OperatorLogs/operator-notes.md`. For chronological kernel history use `Construction/OperatorLogs/operator-log.md`.

## Kernel State

- Current completed kernel: **Kernel 70A — Live Stage Closure and Alpha Path Alignment**
- Completion date: **2026-07-16**
- Commit: **`2611840`**
- Live deployment date: **2026-07-16**. Applying Kernel 70A's own code was a container rebuild only (no new migrations), but the deploy surfaced that **Kernel 70's three migrations (`043`–`045`) had never actually reached the live `victory` database** despite being committed since `920aeb7` on 2026-07-14 — `actions.show_id`, `cues`, and `cue_executions` did not exist in production until this deploy. A `pg_dump` backup was taken first (`backups/victory_pre_kernel70a_migrations_20260716_031420.dump`); all 46 migration files were then replayed against the live database (idempotent, `ON_ERROR_STOP=1`, zero errors) before the backend restart. Verified clean post-deploy: `/health`, `/ws/catharsis`, and `/api/director-console/current` all succeeded with no schema errors.
- Product state: Kernels 53–70A are implemented and live. Visual Scene-composition/capture remains future work.
- Kernel numbers are stable historical labels. Always check the operator log before assigning the next number.

## Product Shape

Victory is a persistent, theatrical TTRPG community platform. Its current spine is:

`Location → Production → Show Run → Show → Show Scene Placement → Session / Showing`

- A **Location** is the authority and venue boundary.
- A **Production** organizes work at one Location.
- A **Show Run** owns roster, audience-program, and admission context.
- A **Show** is one schedulable/playable instance of a Show Run.
- A reusable **Scene** belongs to a Location and may retain optional source-Production provenance.
- A **Show Scene Placement** is one Show's staged use and override layer for a Scene.
- A **Session** is a temporary live runtime window and may link to a Show.
- A **Showing** is the reviewable live wrapper around one active Session, not the scheduling primitive.
- The **Show owns persistent stage state**. Linked Sessions project and add to that state; they do not become its durable owner.

This is the **Domain Hierarchy** — the production/scheduling spine above. A separate, only loosely-connected **Identity Hierarchy** governs who a given user is and what they may do, at up to three independent layers:

- **Operator**: global installation authority (`access.IsOperatorUser`), independent of any Location.
- **Location role** (`location_memberships`): per-user, per-Location role — `producer`, `director`, `cast`, `crew`, or `audience`. This is the primary authority scope for Production/Show Run/Show/Scene/Cue decisions; do not substitute a user's best role at any other Location.
- **Show Run roster role** (`show_run_roster_members`): per-user, per-Show-Run role — `producer`, `director`, `player`, `crew`, `audience`, `guest`, `observer`, or `custom`. This is a distinct fact from Location role and can diverge from it: a Location Producer need not be on a given Show Run's roster at all, and a roster `crew` member need not hold any Location membership.
- **Equipped Character** (`current_session_personas`): per-user, per-**Session** — which Character this user is currently playing, if any. This is ephemeral (`ON DELETE CASCADE` from `sessions`, primary key `(session_id, user_id)`), reset every time a Session ends. There is currently no durable link from a Character to a Show, Show Run, or roster entry; see "No durable Character-to-Show/Show-Run/Session linkage exists" below.

## Current Stack

- Frontend: static HTML, CSS, and JavaScript under `frontend/`
- Stage rendering: PixiJS plus DOM overlays and controls
- Shared frontend helpers: `frontend/lib/` and `frontend/venues/shared/venue-shell.js`
- First Theater and Catharsis runtime entry points: sibling `runtime.js` files backed by modules in each venue's `runtime/` directory
- Backend: Go 1.25 modular monolith under `backend/`
- Database: PostgreSQL 16
- Live transport: Gorilla WebSocket
- Reverse proxy/static serving: Caddy
- Container orchestration: Docker Compose
- Durable uploads: filesystem warehouse plus PostgreSQL metadata
- Discord: OAuth, server bootstrap, Gateway presence/chat integration, and voice-state/mic control; Victory does not transport Discord audio

## Runtime Modes

- Dev: PostgreSQL in Docker; backend on the host, normally port `8081` or `18081`
- Install: `victory-backend` and `victory-postgres` in Docker; Caddy proxies `/api/*`, `/ws/*`, and `/auth/*`
- Test: dedicated `victory_test` database selected only through `TEST_DATABASE_URL`
- Fresh-install proof: `scripts/smoke/fresh-install.sh --local` creates and destroys its own disposable database

## Current Major Surfaces

Identity and community:

- Local and Discord authentication, sessions, password reset, invites, authority requests, and operator bootstrap
- Trailer Player Workbook and projected Trailer Face
- Private, directional My People relationships, notes, follow-ups, and shared context
- Third Place opt-in Headshot commons
- Mailbox and note cards

Character and rules:

- Character Workbook rooted at `character_cards`
- Active-character selection and venue character-sheet projection
- Face visibility, priority, Director locks, and displayed-value overrides
- Private character journals
- Socio character flow through parentage, Chapter 2, archetype selection, and Chapter 4 skill/group selection
- Command registry, canonical dice, and game-event mirroring

Production and performance:

- Production creation and location-scoped authority
- Show Runs, rosters, self-join, audience blocks, and curated audience programs
- Shows with scheduling/status fields and optional Session links
- Location-scoped reusable Scenes and per-Show Scene Placements
- Current-Scene pointer and Show variables
- Persistent Show-owned action/state replay across linked Sessions
- Cues with ordered actions, idempotent GO execution, execution ledger, crew/player trigger boundaries, and curated player-facing stage buttons
- Showing review and Director Console controls

Stage and assets:

- Maps, square/hex grids, camera, index cards, token assets, snapping, and persistent placements
- Warehouse storage policy, generated token variants, tombstone-safe asset reads, and capacity guardrails
- First Theater and Catharsis receive Show/current-Scene snapshot context, rehearsal-availability messaging, and player Cue controls

## Current API Families

The canonical registrations live in `backend/cmd/victory/main.go`. Major route families are:

- `/api/auth/*`, `/auth/discord/*`, `/api/account/*`, `/api/session/me`
- `/api/discord/*`, `/api/requests/*`, `/api/invites*`, `/api/productions`
- `/api/player-profile/*`, `/api/player-relationships*`, `/api/third-place/headshots*`
- `/api/character-cards*`, `/api/character-workbooks/*`, `/api/character-journals`, `/api/characters/*`
- `/api/commands/*`, `/api/messages*`, `/api/note-cards`, `/api/index-cards`
- `/api/show-runs*`, `/api/shows*`, `/api/scenes*`, `/api/cues/*`
- `/api/showings*`, `/api/director-console/current`
- `/api/world/{venue}`, `/api/session/{venue}/join`, `/api/venues/*`
- `/api/workshop/*`, `/api/warehouse/*`, `/api/assets/*`
- `/ws/the-cave`, `/ws/catharsis`, `/ws/first-theater`, `/ws/player-profile`, `/health`

Kernel 70 route additions:

- `POST /api/shows/{show_id}/current-scene`
- `GET|POST /api/shows/{show_id}/scenes/{placement_id}/cues`
- `GET /api/shows/{show_id}/scenes/{placement_id}/player-cues`
- `GET|PATCH /api/cues/{cue_id}`
- `POST /api/cues/{cue_id}/go`

Kernel 70A route additions:

- `GET /api/world/first-theater`, `POST /api/session/first-theater/join` — First Theater's own independent venue wiring, additive alongside the-cave and Catharsis (the-cave's own routes are untouched)
- `POST /api/shows/{show_id}/sessions/start` — Start Show Session: starts-or-resumes a venue's live Session and links it to a Show server-side, with no manual Session ID handling by the caller

## Authority And Privacy Rules

- Discord authenticates; Victory authorizes.
- Operator is installation authority. Producer is the highest normal in-app role.
- Production, Show Run, Show, Scene, and Cue authority is resolved against the relevant Location; do not use a globally best role for location-scoped decisions.
- Audience endpoints use curated response types or explicit field exclusion. Backstage Show variables and current-placement identifiers must not serialize through the audience Show Program.
- My People is private and directional. Non-owner access returns `404`, not `403`, so record existence is not disclosed.
- Player-facing Cue lists contain only `{id, label}` for enabled Cues the viewer can currently trigger. Audience can never trigger a Cue.
- Client identity, role, current character, dice results, and stage authority are requests to the server, never client-authored facts.

## Kernel 70 Stage And Cue Semantics

- `scenes.location_id` is canonical scope. `source_production_id` is nullable provenance, not an ownership restriction.
- `shows.current_show_scene_placement_id` identifies the Show's active staged Scene.
- `shows.variables_json` is a denormalized backstage cache updated by canonical Show-scoped actions.
- `actions.show_id` permits Show-owned state to survive Session replacement.
- World snapshots fold both the current Session's actions and linked Show actions. An unlinked Session retains pre-Kernel-70 behavior.
- Implemented Cue actions: `go_to_scene`, `emit_game_event`, `set_show_variable`.
- Cue actions run in order, one transaction per action, and fail stop. Earlier successful actions are not rolled back if a later action fails.
- GO idempotency is enforced by `UNIQUE(cue_id, idempotency_key)` and recorded in `cue_executions`.
- Crew may create/edit non-destructive Cues and press GO, but cannot directly change the current Scene, edit a Base Scene, archive a Show, or perform destructive management.

## Current Database Spine

- Identity/social: users and auth tables, memberships, Player Workbook/events/facts, player relationships/journal/follow-ups, Third Place Headshots
- Character/rules: character cards, active characters, workbook entries/modules, journals, skills, Face overrides
- Production/runtime: locations, venues, productions, Show Runs/rosters/blocks, Shows, Scenes, Show Scene Placements, Sessions, Showings, actions, Cues, Cue executions
- Stage/assets: elements, placements, venue layout elements, maps, grids, assets, warehouse policy

## Known Working End-To-End Flows

- Signup/login → Trailer Workbook → publish a Face → leave a Third Place Headshot
- Create and privately maintain a My People relationship without exposing it to the subject
- Create a Production → Show Run → roster/audience program → Show
- Create a reusable Scene → stage independent versions in Shows at the same Location
- Link a Session to a Show and project Show-owned stage actions across Session replacement
- Select or clear a Show's current Scene
- Author a Cue, press GO idempotently, change Scene / emit an event / set a Show variable, and inspect the execution outcome
- Render curated player Cue buttons in First Theater and Catharsis without exposing backstage actions
- Create and project a character through Greenroom, Catharsis, and First Theater
- Upload/reuse maps and tokens, configure grids, move stage objects, and submit canonical dice rolls
- Start/control/close a Showing and review its durable action history

## Known Gaps And Deferred Work

- **Visual Scene composition/capture is not implemented.** Base Scene versus This Show's Version currently covers metadata/configuration, not a bound visual composition over `elements`, `venue_layout_elements`, and stage actions.
- `reveal_object`, `hide_object`, `enable_interaction`, and `disable_interaction` Cue actions are deferred until that object/state mapping is designed.
- Kernel 70 lacks browser screenshot evidence for rehearsal messaging and Cue buttons; code paths, syntax, backend tests, and existing Node tests were used instead.
- The First Theater Node suite (`tests/first-theater/`) has 46 passing tests and 9 pre-existing `dice.test.js` failures; do not describe that suite as wholly green until repaired. Kernel 70A added a byte-for-byte mirror at `tests/catharsis/` (import-path/fixture changes only) with the identical 46-pass/9-known-fail shape, plus `tests/contract/scene-nodes.contract.test.js` asserting the behavior both venues' `scene-nodes.js` must share.
- First Theater and Catharsis retain parallel runtime trees. Changes to shared stage behavior must inspect and test both — `frontend/venues/{catharsis,first-theater}/runtime/*.js` — and update both `tests/catharsis/` and `tests/first-theater/`. One confirmed, intentional exception: Catharsis's `scene-nodes.js` renders an extra token-aura layer First Theater's does not; the contract test in `tests/contract/` documents this as accepted drift, not a bug.
- The Cave remains a dense proving-ground UI rather than a polished player product.
- Some older consolidated roadmaps contain superseded kernel numbers; their collision notes are historical planning records, not the actual kernel sequence.
- Provider-only accounts still lack a provider step-up path for secure email change.
- Discord active-speaker detection and per-user audio volume are not truthfully available through the current integration.
- Video recording/capture does not exist and is not implied by Showing Review or Scene capture.
- **The campus entry funnel is not the aspirational one-path journey.** Info Booth (a map-tile modal in `frontend/app.js`/`index.html`) and Audition Hall (`frontend/venues/audition-hall/`) are both real, shipped surfaces, but they are general orientation/onboarding points, not a guided "join a Show" flow. A registered participant reaching their Show today still depends on Stage Management/roster setup done ahead of time, not a single campus path from map to stage.
- **First Theater gained its own independent, real venue wiring in Kernel 70A** (`GET /api/world/first-theater`, `POST /api/session/first-theater/join`, `/ws/first-theater`, plus widened stage-action authority) rather than remaining a themed skin over the-cave's backend. It is still investor/demo-only — no participant-facing map-visibility work targets it, and the-cave itself (a deliberate, permanently hidden, DOM-only test harness) was left completely untouched.
- **No durable Character-to-Show/Show-Run/Session linkage exists.** Audited (Kernel 70A, schema + Go-type check, no build): `character_cards` carries `owner_user_id`/`location_id`/optional `production_id` provenance only, no Show/Show-Run/Session foreign key. `show_run_roster_members` is account-level (`user_id`), with no `character_card_id` column. `current_session_personas` is the only table connecting a Character to live play, and it is session-scoped/ephemeral (`ON DELETE CASCADE` from `sessions`, primary key `(session_id, user_id)`) — it answers "who is this user playing right now" for one live Session, not "which Character is this roster member's Character for this Show." A Show's roster and its participants' Characters are today two unconnected facts a Director must reconcile by memory. Greenroom's own unlock condition is unrelated to this gap: it checks only `EXISTS(character_cards WHERE owner_user_id = $1 AND is_deleted = FALSE)` (`backend/internal/access/visibility.go`) — any owned, non-deleted Character card unlocks it, regardless of workbook completion. Recommended smallest attachment point for Kernel 71, not built here: a nullable `character_card_id` on `show_run_roster_members`, letting a roster entry optionally declare its Character without touching the session-scoped persona-equip mechanism.
- **Two independent, non-interchangeable membership tables both claim to answer "what is this user's role."** Discovered live during Kernel 70A's manual checklist walkthrough (a fresh Producer, correctly set up via `location_memberships`, was rejected as `viewerRole = "none"` at their own Catharsis snapshot). `location_memberships` (per-Location, used by `access.CurrentLocationRole*`, `showruns.CanManageShowRun`, and everything Show-Run/Show/Scene/Cue-authority-related) is the current, Kernel-66+ system. `memberships` (per-venue-or-production-or-location, used only by `main.go`'s `lookupVenueRole` to compute the `viewerRole` passed into `world.LoadVenueSnapshot`) is an older, still-live table that drives both the pre-Kernel-70 stage-element-visibility filtering and Kernel 70A's new `theater_context` "backstage" classification. A user can hold the correct `location_memberships` producer/director role and still be treated as role `"none"` for live-venue viewing purposes if nobody separately granted them a `memberships` row. Reaching Catharsis's live snapshot at all additionally requires an `access_grants` row (`venue_access`, tied to `location_memberships` role IN producer/director/cast/crew) — a third, also-separate gate. Not fixed here (a real authority-model unification, out of Kernel 70A's approved scope); flagged so Kernel 71+ treats `lookupVenueRole`/`memberships` as a known trap, not an oversight.

## Next Recommended Direction

**Kernel 70A (Live Stage Closure and Alpha Path Alignment)** shipped and is now live in production (2026-07-16): a server-side Start Show Session action (no manual Session ID handling), proof that Show-owned stage state survives a full start/end/resume cycle at Catharsis with no rebuild step, `go_to_scene`/`set_show_variable` working with no active session while `emit_game_event` fails cleanly and visibly, a closed Audience-facing leak (Rehearsal banner/Cue buttons were previously visible regardless of role), backend-computed theater-context empty states with the exact required strings, honest Scene/Cue setup labels, First Theater's own independent (still investor/demo-only) venue wiring, a Character-to-Show linkage audit (no build), and a `tests/catharsis/` mirror plus a scene-nodes contract test. The live deploy also retroactively applied Kernel 70's own migrations, which had never reached production despite being committed since 2026-07-14 — see Kernel State above. **Still deferred, by explicit scope decision**: the first playable Socio show; a casting/attendance system; and any participant-facing First Theater work, since First Theater remains investor/demo-only for this pass.

The natural next kernel is **visual Scene composition/capture**: define how reusable Base Scene content and a Show Placement's overrides bind to the existing stage-object/action model without creating a second renderer authority system. A smaller independent closure pass can add browser screenshots for Kernel 70/70A's GO and Start Show Session controls and repair the nine pre-existing frontend dice-test failures (now duplicated in both `tests/first-theater/` and `tests/catharsis/`). A good practice to establish going forward: confirm each kernel's migrations actually reached the live database as part of closing it out, not just that they're committed — this gap sat unnoticed for two days.

## Recording Language

- **Showing Review**: review of durable Session/showing actions and history.
- **Scene capture/composition**: authoring reusable stage content and its Show-specific overrides.
- **Video recording**: audiovisual output capture; still future work and a separate concept.
