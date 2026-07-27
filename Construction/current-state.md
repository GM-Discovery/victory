# Victory Current State

## Purpose

This document is the current-state canon for Victory as of **Kernel 74**. Historical kernel specifications and reportbacks describe what was true when they were written; this file wins when an older current-tense statement conflicts with the implemented repository.

For detailed vocabulary use `Construction/Dictionary.txt`. For durable implementation traps use `Construction/OperatorLogs/operator-notes.md`. For chronological kernel history use `Construction/OperatorLogs/operator-log.md`.

## Kernel State

- Current completed kernel: **Kernel 74 — Locked Door Intentions, Ra Guided Dialogue, and Participant-Local Tutorial Handoff**
- Completion date: **2026-07-27**
- Status: **PASS, deployed live, uncommitted** — migrations `066`–`068` applied to production with pre-apply backups (ledger at 69), backend rebuilt, `scripts/test/alpha-gate.sh` green, and the operator walked the tutorial in a real browser. Not committed; pending review, matching project practice.
- Product state: Kernels 53–74 are implemented. Visual Scene composition shipped in Kernel 73A; Scene *capture* remains future work.
- Next kernel: **Kernel 75 — Tutorial Completion, Aftercare, and Director-Controlled Continuation** (drafted at `Construction/Kernels/kernel-75-tutorial-completion-aftercare-continuation-v0.1.md`; Aftercare needs an operator decision before implementation).
- Kernel numbers are stable historical labels. Always check the operator log before assigning the next number.
- Previous completed/live kernel: **Kernel 70A — Live Stage Closure and Alpha Path Alignment**, deployed live 2026-07-16, commit `2611840`. That deploy also retroactively applied Kernel 70's own migrations (`043`–`045`), which had never reached the live database despite being committed since `920aeb7` on 2026-07-14.

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
- **Show Run roster role** (`show_run_roster_members`): per-user, per-Show-Run role — `producer`, `director`, `player`, `crew`, `audience`, `guest`, `observer`, or `custom`. This is a distinct fact from Location role and can diverge from it: a Location Producer need not be on a given Show Run's roster at all, and a roster `crew` member need not hold any Location membership. **As of Kernel 71, an active Player roster row is the canonical, sole reason a Player may participate in a Show Run — see "Kernel 71 Participation Model" below.**
- **Selected Character** (`show_run_roster_members.character_card_id`): per-user, per-**Show Run** — which Character this Player is presenting as for that Show Run, freely switchable, no Director approval required (Kernel 71). This superseded the old Session-scoped `current_session_personas` as the canonical "has this Player chosen a Character" signal (`world.LoadVenueSnapshot`'s `TheaterContext` now reads this column); `current_session_personas` and the separate, site-wide `active_user_characters` table both still exist and are read by other, unrelated code paths (persona-equip actions, Greenroom's own "current Character" concept), but neither is authoritative for Show Run participation.

## Kernel 71 Participation Model

Kernel 71 closed two gaps: there was no mutual-consent path onto a Show Run's roster (a Director could unilaterally add anyone as Player), and a real authority-resolution bug meant a Producer/Director whose only grant was a `location_memberships` row could resolve as `viewerRole = "none"` at their own venue (see the former "Known Gaps" entries below, now resolved).

- **Canonical participation resolver** (`backend/internal/participation`, `ResolveParticipationContext`): the one function venue-role/theater-entry/backstage-gate call sites should converge on. Precedence: Operator → `location_memberships` → `show_run_roster_members` → any active `location_memberships` row (audience) → legacy `memberships`/`access_grants` (last-resort upgrade only, never a downgrade). `main.go`'s `lookupVenueRole` (feeding `/api/world/*`'s `viewerRole`) now delegates here instead of querying `memberships` alone — this is the fix for the "none" bug.
- **Two-punch Show tickets** (`show_run_tickets`, `backend/internal/tickets`): the only ordinary path to an active Player roster row. Either side may punch first (Player requests via Audition Hall, or a Director invites); one punch grants nothing; the second punch is one atomic transaction (`tickets.SecondPunch`) that marks the ticket valid and creates/reactivates exactly one Player roster row, reusing `show_run_roster_members`'s existing partial-unique-index `ON CONFLICT` so a duplicate row or a double-processed concurrent punch is structurally impossible. `showruns.AddRosterMember` and `UpdateRosterMemberRole` both reject a direct/promoted `role="player"` for non-Operator actors — the two-punch ticket is the only ordinary route to that role; Operator retains an explicit, auditable override.
- **Character selection**: `POST /api/show-runs/{id}/roster/me/character` (always acts on the caller's own roster row). Character must be active (`is_deleted = FALSE`) and owned by the caller.
- **Show short codes**: every Show gets an auto-generated, human-typeable, location-scoped-unique code (`shows.short_code`, confusable-excluding charset) at creation, editable by Director/Producer/Operator.
- **`/showtime`**: one Director action (`POST /api/showtime/control`, in-app legacy command matching `/session`'s precedent) that resolves a Show by short code, derives its venue from staged Scene Placements (asking only if none or multiple distinct venues are staged), and starts/resumes the live Session via the unmodified Kernel 70A `shows.StartShowSession` — no manual venue slug, Session ID, or separate Production/Show-Run/Show start steps. Mic always starts off; the response suggests `/mic hot`. `/showtime end` ends only the technical Session — current Scene, roster, and Character selections are untouched by construction (no write path in `showtime.End` reaches them).

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
- Two-punch Show tickets (Audition Hall Player-request and Director-invitation directions) as the sole ordinary path to active Player roster participation
- Per-Show-Run Character selection and free switching among a Player's own active Characters
- Shows with scheduling/status fields, auto-generated editable short codes, and optional Session links
- `/showtime <code>` one-action start/resume/status/end orchestration
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

Kernel 71 route additions:

- `POST /api/show-runs/{id}/tickets/request`, `POST /api/show-runs/{id}/tickets/invite`, `GET /api/show-runs/{id}/tickets/incoming` — two-punch ticket first-punch/inbox
- `POST /api/tickets/{ticket_id}/punch|decline|withdraw`, `GET /api/tickets/mine` — second punch and a Player's own ticket list
- `GET /api/show-runs/{id}/roster/me`, `POST /api/show-runs/{id}/roster/me/character` — a Player's own roster status and Character selection
- `PATCH /api/shows/{show_id}/short-code`, `GET /api/shows/by-code?code=` — short-code edit and lookup (the latter deliberately narrow: show/show-run/location identifiers only, no backstage data)
- `POST /api/showtime/control` — the bespoke endpoint the in-app `/showtime` legacy command is bound to (`{short_code, action: start|status|end, force_reattach, venue_slug}`)

## Authority And Privacy Rules

- Discord authenticates; Victory authorizes.
- Operator is installation authority. Producer is the highest normal in-app role.
- Production, Show Run, Show, Scene, and Cue authority is resolved against the relevant Location; do not use a globally best role for location-scoped decisions.
- Audience endpoints use curated response types or explicit field exclusion. Backstage Show variables and current-placement identifiers must not serialize through the audience Show Program.
- My People is private and directional. Non-owner access returns `404`, not `403`, so record existence is not disclosed.
- Player-facing Cue lists contain only `{id, label}` for enabled Cues the viewer can currently trigger. Audience can never trigger a Cue.
- Client identity, role, current character, dice results, and stage authority are requests to the server, never client-authored facts.
- Player Show Run participation is granted only by a valid two-punch ticket's atomic second punch (`tickets.SecondPunch`), never by a direct roster insert/promotion outside Operator's explicit override. A client-supplied user ID, roster role, or Character ID is never trusted where it can be derived server-side instead (a Player can only punch/select for themselves; a Director's punch authority is re-checked against the locked ticket row's own Show Run, not a client-supplied one).
- Show short-code lookup (`GET /api/shows/by-code`) returns only show/show-run/location identifiers — never roster, variables, or other backstage state.

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
- Production/runtime: locations, venues, productions, Show Runs/rosters/blocks, Shows (with `short_code`), Scenes, Show Scene Placements, Sessions, Showings, actions, Cues, Cue executions, `show_run_tickets`
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
- Player requests to join a Show Run via Audition Hall, a Director approves, and the Player selects a Character and enters the linked theater with no manual access grant (Player-request direction)
- Director invites a specific Player to a Show Run, the Player accepts in Audition Hall, and the same participation/Character-selection/theater-entry flow follows (Director-invitation direction)
- Director types a Show's short code into `/showtime` and the venue Session starts or resumes with the venue auto-derived, mic off, and the Show's persistent current Scene intact; `/showtime end` closes only the technical Session
- **Kernel 72** replaced First Theater's and Catharsis's separately-copied stage runtimes with one shared engine (`frontend/lib/stage-runtime/`), configured per venue via a small `venue.js`; embedded, checksummed, auto-backed-up migrations (`backend/internal/migrate`) are now the single schema source of truth, ending the manual-apply/drift risk that bit Kernel 70A. **Kernel 72A** replaced the hardcoded per-feature venue-slug allowlists that risk (element actions, session control) with `venues.config` capability flags, fail-closed for unknown venues.
- **Kernel 73** adds Catharsis's first participant-local gameplay packet: a Director-authored interaction (seeded example: "Visit Kessa's Shop") opens a private Program Panel for the triggering Player only, without changing the Show's shared current Scene. The Player picks one of five authored conversational stances (disposition is fixed per stance, never flipped by the roll), attempts a Haggle skill check (server-authoritative d6-skilled/d4-unskilled roll against Target Value 5, narrative-only discount, no currency touched), and purchases seeded starting equipment that persists as durable Character inventory (viewable on its own page, `frontend/venues/greenroom/inventory.html`). Built on new reusable primitives: `equipment_items`/`character_inventory_items` (idempotent purchase, ledger-based retry safety mirroring Cues' `cue_executions` pattern), `merchant_packets` (a bounded, reusable non-dialogue-graph packet format), and `participant_interactions` (its own table, participant-scoped rather than role-scoped like Cues, attached to a Show Scene Placement through a Director authoring panel in Stage Management). The Program Panel itself (`frontend/lib/stage-runtime/program-panel.js`) is venue-agnostic and reusable for a future second merchant/program packet.
- **Kernel 74** closes the Player-controlled portion of the Locked Courtyard tutorial. A Player finishes with Kessa (no purchase required), which reveals a milestone-gated **interaction hotspot** aligned over the door already painted into the Courtyard map — not a duplicate door token. Clicking it opens one neutral freeform field: the Player writes what their Character tries, Victory stores those words verbatim, reports them privately to Directors+ as a durable backstage note (informational only, never a GO), and Ra interrupts before the attempt resolves. Ra is a bounded **guided-dialogue packet** — authored topics with prerequisites and per-Character seen-state, no AI and no dialogue graph — delivering the Crown Bet. `Leave Ra` is Player-controlled and moves **only that Player** onto a **participant-local stage projection**: a temporary per-Player presentation layered over the shared stage, which never writes `shows.current_show_scene_placement_id` and is cleared when a Director later flies a shared Scene. There is no Director GO anywhere inside the sequence.

## Known Gaps And Deferred Work

- **Visual Scene composition/capture is not implemented.** Base Scene versus This Show's Version currently covers metadata/configuration, not a bound visual composition over `elements`, `venue_layout_elements`, and stage actions.
- `reveal_object`, `hide_object`, `enable_interaction`, and `disable_interaction` Cue actions are deferred until that object/state mapping is designed.
- Kernel 70 lacks browser screenshot evidence for rehearsal messaging and Cue buttons; code paths, syntax, backend tests, and existing Node tests were used instead.
- The First Theater Node suite (`tests/first-theater/`) has 46 passing tests and 9 pre-existing `dice.test.js` failures; do not describe that suite as wholly green until repaired. Kernel 70A added a byte-for-byte mirror at `tests/catharsis/` (import-path/fixture changes only) with the identical 46-pass/9-known-fail shape, plus `tests/contract/scene-nodes.contract.test.js` asserting the behavior both venues' `scene-nodes.js` must share.
- **Resolved in Kernel 72** (documented here for history, no longer a gap): First Theater and Catharsis no longer retain parallel runtime trees. Both now load the single shared engine at `frontend/lib/stage-runtime/`, configured per venue via a small `venue.js`; the old per-venue `runtime/*.js` copies and the duplicated `tests/first-theater/`/`tests/catharsis/` suites were deleted and merged into `tests/stage-runtime/`. Catharsis's token-aura rendering became the shared engine's behavior (First Theater gained it too) rather than a documented drift exception.
- **Kernel 73's Equip Mode is Catharsis-only by design.** `participant_interactions_enabled` is seeded true only for `catharsis`; First Theater integration is explicitly deferred (matches the Program Panel module itself being venue-agnostic and ready to reuse, not a scope gap in the module).
- **Kernel 73's merchant packet has no dedicated authoring editor.** Kessa's packet (five stance dispositions, Haggle die/Target Value, success/failure text) is authored via a seed migration, not a UI form — spec-permitted ("a bounded form or seed configuration"), but a Director cannot currently edit Kessa's dialogue without a new migration. The minimal equipment *item* editor (name/image/description/active) does exist as a real HTTP CRUD surface (`/api/venues/{venue_slug}/equipment`), just with no frontend page wired to it yet.
- **No browser/screenshot proof for Kernel 73's UI** (the Equip Mode Program Panel, the Director's Participant Interactions authoring panel, the Inventory page) — same standing gap as every kernel since 65; verification is the real compiled backend plus a full DB-backed Go integration test driving the entire ticket→character→Show→Scene→Kessa→purchase→inventory chain end-to-end, not a synthetic mock.
- The Cave remains a dense proving-ground UI rather than a polished player product.
- Some older consolidated roadmaps contain superseded kernel numbers; their collision notes are historical planning records, not the actual kernel sequence.
- Provider-only accounts still lack a provider step-up path for secure email change.
- Discord active-speaker detection and per-user audio volume are not truthfully available through the current integration.
- Video recording/capture does not exist and is not implied by Showing Review or Scene capture.
- **The campus entry funnel is not the aspirational one-path journey.** Info Booth (a map-tile modal in `frontend/app.js`/`index.html`) and Audition Hall (`frontend/venues/audition-hall/`) are both real, shipped surfaces, but they are general orientation/onboarding points, not a guided "join a Show" flow. A registered participant reaching their Show today still depends on Stage Management/roster setup done ahead of time, not a single campus path from map to stage.
- **First Theater gained its own independent, real venue wiring in Kernel 70A** (`GET /api/world/first-theater`, `POST /api/session/first-theater/join`, `/ws/first-theater`, plus widened stage-action authority) rather than remaining a themed skin over the-cave's backend. It is still investor/demo-only — no participant-facing map-visibility work targets it, and the-cave itself (a deliberate, permanently hidden, DOM-only test harness) was left completely untouched.
- **Resolved in Kernel 71** (documented here for history, no longer a gap): Character-to-Show/Show-Run linkage now exists via `show_run_roster_members.character_card_id`, and the `location_memberships`/`memberships`/`viewerRole="none"` bug is fixed by the canonical `participation.ResolveParticipationContext` resolver — see "Kernel 71 Participation Model" above.
- **Audience tickets are not implemented.** Kernel 71's two-punch ticket (`show_run_tickets.requested_role`) is CHECK-locked to `'player'` only. A future Audience-ticket kernel is expected to be an additive migration (widen the CHECK), not a redesign; Audience tickets would be single-Show-scoped (not Show-Run-scoped like Player tickets) and acquired through a future audience/marketing venue, not Audition Hall.
- **Legacy `memberships`/`access_grants` tables are not removed.** Kernel 71 deliberately did not do a wholesale deletion; they remain read as a last-resort compatibility upgrade inside the canonical resolver, and the ticket second-punch writes a compatibility `access_grants` row for any not-yet-migrated reader. Several packages (`characters`, `assets/read.go`, `showings/review.go`) still independently UNION `location_memberships`+`memberships` rather than calling a shared helper — left as-is this kernel (already-correct, just duplicated) rather than risking a broader refactor; `assets/upload.go`'s `resolveProducerScope` is the one reader that does not union both tables at all, flagged but not changed (no demonstrated bug against it).
- **No casting/attendance system.** A valid ticket is a one-time mutual-consent event; there is no recurring-attendance tracking, no "who's coming to Thursday's Show" roster distinct from the Show Run roster itself.
- **No real Discord-native `/showtime` slash command.** `/showtime` is an in-app "legacy" command (`commands/registry.go`, matching `/session`'s existing precedent) reachable from Victory's own command UI/Stage Management button, not typeable directly into a Discord channel the way `/mic` is — a deliberate, confirmed scope decision, not an oversight.
- **No browser/screenshot proof for Kernel 71's UI** (Audition Hall panels, Stage Management's Invite/Incoming-Tickets panels, the Character-selection page). No browser automation tooling exists in this environment; verification was full HTTP-level proof against a real compiled backend (`scripts/smoke/fresh-install.sh`/`scripts/test/alpha-gate.sh`), the same substitution used since Kernel 65.

## Next Recommended Direction

**Kernel 73 (Catharsis Equip Mode, Character Inventory, and Kessa Program Packet)** is implemented and verified end-to-end against the real migrated test database (a single Go integration test drives ticket→roster→character→Show→Courtyard placement→Kessa attachment→all five stances→Haggle preview/attempt→idempotent purchase→inventory persistence, plus a dedicated security-proof test file). It establishes three reusable primitives for future bounded packets: `merchant_packets`/`equipment_items`/`character_inventory_items`, `participant_interactions` (participant-scoped, distinct from Cues' role-scoped model), and the venue-agnostic `frontend/lib/stage-runtime/program-panel.js`. **Explicitly deferred, by locked scope**: NPC AI/freeform dialogue, an economy/currency system, First Theater integration, and the door/Ra-interruption/backdrop-transition/first-Show-completion sequence the Kessa Scene is a prelude to.

The natural next kernel is the **door/Ra-interruption/backdrop-transition sequence** the Kessa Equip Mode packet was built as a prelude to, or **visual Scene composition/capture**: define how reusable Base Scene content and a Show Placement's overrides bind to the existing stage-object/action model without creating a second renderer authority system. Before or alongside either: repair the nine pre-existing `dice.test.js` failures (now tracked once in `tests/stage-runtime/`, not duplicated); add browser screenshot evidence for the stage/participation/Equip-Mode controls (no browser automation tooling exists in this environment yet); finish deduplicating the three packages (`characters`, `assets/read.go`, `showings/review.go`) that each independently UNION `location_memberships`+`memberships` instead of calling one shared helper; and build a Director-facing editor for merchant packet dialogue (currently seed-only).

## Recording Language

- **Showing Review**: review of durable Session/showing actions and history.
- **Scene capture/composition**: authoring reusable stage content and its Show-specific overrides.
- **Video recording**: audiovisual output capture; still future work and a separate concept.
