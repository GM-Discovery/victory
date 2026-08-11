# Everything Implemented — current-repository audit

**Audit date:** 2026-08-11  
**Repository state examined:** `b5a8787` plus the current uncommitted Kernel 86 working tree.  
**Method:** source, embedded migrations, registered routes, frontend runtime, tests, seeds, current-state/operator material, kernel specs/reportbacks, and recent Git history were cross-checked. Current code and schema take precedence over documents.

## How to read this document

This is an inventory of current behavior, not a roadmap. A status applies to the immediately described capability, not to all of Victory or its UI polish.

- **IMPLEMENTED** — usable current capability.
- **IMPLEMENTED WITH KNOWN LIMITATIONS** — usable core with a bounded material gap.
- **PARTIAL / INCOMPLETE** — code/UI exists but not an end-to-end product capability.
- **SUPERSEDED** — an older model has been replaced.
- **DEFERRED / NOT IMPLEMENTED** — explicitly contemplated but absent from current behavior.

The audit records 44 capability areas: **7 IMPLEMENTED**, **32 IMPLEMENTED WITH KNOWN LIMITATIONS**, **1 PARTIAL / INCOMPLETE**, **3 DEFERRED / NOT IMPLEMENTED**, and **1 SUPERSEDED**. “Kernel 86/86A” below is verified in the uncommitted working tree; it should not be mistaken for a committed revision.

## 1. Identity, authentication, and people

### Account and session identity
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Local login, logout, password reset, email verification/change, invites, account export, deletion/tombstoning, and session identity are registered in `backend/cmd/victory/main.go` and implemented in `backend/internal/identity/`.
- Discord OAuth is a real provider path; Discord server bootstrap, gateway, channel mapping and mic-control surfaces also exist.
- Migration spine: `001`, `003`, `005`, `018`–`024`, `081`–`083`; tests include `identity/account_test.go`, `account_deletion_test.go`, and `kernel76_auth_closure_test.go`.
- Password signup is deliberately closed in production via `PASSWORD_SIGNUP_ENABLED=false`; Discord is the supported production account-creation route. Provider-only accounts have no provider step-up path for secure email change, and recovery-email delivery is code-complete but blocked on mail-provider approval.

### Player Workbook, Face, and active Character
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- A user can maintain a Trailer Player Workbook, publish/shape a Face, set visibility/priority/stage name, and use an active Character. Greenroom, Trailers, Catharsis, and First Theater consume projections.
- Evidence: migrations `007`, `016`, `017`, `031`–`035`, `036`; `backend/internal/playerprofile/`, `backend/internal/characters/`, `frontend/venues/trailers/`, `frontend/venues/greenroom/`; corresponding package tests.
- Character selection for participation is Show-Run scoped (`show_run_roster_members.character_card_id`), while legacy session and site-wide active-character concepts remain for other paths. That distinction is real and is a source of integration complexity.

### Session-scoped persona as participation authority
**Status:** SUPERSEDED

- The old `current_session_personas` signal is no longer the canonical basis for a Player entering a Show Run. Kernel 71 made the active Player roster row and its selected Character authoritative.
- Evidence: migration `047_kernel71_roster_character_selection.sql`, `backend/internal/participation/`, `showruns/roster.go`, and the Kernel 71 participation model in `Construction/current-state.md`.
- Legacy persona and site-wide active-character rows remain read by unrelated paths; they were not deleted.

### Audition Hall and Show participation
**Status:** IMPLEMENTED

- Players request a Show Run place or Directors invite them; the second acceptance atomically creates/revives the Player roster row. Players then select an owned active Character.
- Evidence: migrations `046`–`048`, `083`; `backend/internal/tickets/`, `showruns/roster.go`, `participation/`; `/api/show-runs/{id}/tickets/*`, `/api/tickets/*`, `/api/show-runs/{id}/roster/me/character`; `tickets_test.go`, `showruns_test.go`; Kernel 71 reportback/current-state.
- This is generic Victory participation functionality, not Socio-specific.

### My People and Third Place
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- My People provides private, directional relationships, notes, follow-ups, journal context and archive filtering. Third Place provides opt-in Headshot commons gated by Trailer Face readiness.
- Evidence: migration `037`, `038`; `backend/internal/playerrelationships/`, `thirdplace/`; `frontend/venues/trailers/people.html`, `frontend/venues/third-place/`; privacy and HTTP tests.
- My People deliberately returns `404` to non-owners; the People Picker combines these canonical profile identities for grants/invites. This does not create a general public people directory.

## 2. Roles, permissions, and authority

### Layered authority
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Operator is installation-wide; Location membership (`producer`, `director`, `cast`, `crew`, `audience`) is the primary scoped authority; Show Run roster is a separate participation fact. Audience/Cast/Crew/Director/Producer distinctions are enforced server-side in relevant handlers.
- Evidence: `backend/internal/access/`, `participation/`, `shows/authority.go`, `scenes/authority.go`, `cues/authority.go`, action authority tests, and `Construction/Security/kernel-76-*` matrices.
- Older `memberships`/`access_grants` are still an upgrade fallback, and a live stage additionally depends on some legacy access-grant wiring. The latter is documented in Kernel 85’s live proof.

### Server-authoritative stage and action control
**Status:** IMPLEMENTED

- Token/element movement, placement, overlay, chat, dice, scene changes, Cues, cohorts, game status and ticket actions derive identity on the server and re-check authority. Client-provided identity/role is not authority.
- Evidence: `backend/internal/actions/authority.go`, `actions/*.go`, `scenes/`, `cues/`, `cohorts/`; authority/db tests across those packages.

### Group Leader and Current Turn
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- A generic, ephemeral per-live-session coordination registry supports explicit Group Leader and Current Turn handoff; Storyboards provides its Presence Tray consumer.
- Evidence: `backend/internal/venuecoordination/registry.go`, `backend/internal/storyboards/coordination.go`, `kernel83_coordination_dbtest_test.go`, `frontend/venues/storyboards/board.html`, Kernel 83 reportback.
- It is not initiative/order automation, persists only while the live session has watchers, and the context menu has no keyboard entry point.

## 3. Shows, venues, scenes, and navigation

### Production, Show Run, Show, and Showing spine
**Status:** IMPLEMENTED

- Victory has Location → Production → Show Run → Show → Show Scene Placement, with Sessions as temporary runtime windows and Showings as reviewable live wrappers. Shows can be scheduled/archived, rostered, given short codes and linked to sessions.
- Evidence: migrations `000`, `039`–`041`, `044`, `048`; `backend/internal/showruns/`, `shows/`, `showings/`, `showtime/`; Show Run and Show routes in `main.go`; package tests.

### Scene library and current-scene switching
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Reusable location-scoped Scenes can be created, staged into Shows, edited/archived, selected as a Show’s current placement, and presented through a curated audience program. `/showtime` starts/resumes the derived venue session from a Show code.
- Evidence: migrations `042`–`045`; `backend/internal/scenes/`, `shows/stage.go`, `showtime/`; `scenes_test.go`, `show_stage_test.go`, `showruns` tests; First Theater/Catharsis runtime.
- Cue support is intentionally bounded to `go_to_scene`, `emit_game_event`, and `set_show_variable`; a Cue failure stops later actions without rolling back earlier successful ones.

### Scene composition and capture
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Base Scene elements and Show-placement overrides/bindings resolve into the snapshot; a Director can update the current Scene or save the arrangement as a new Scene.
- Evidence: migration `064_kernel73a_scene_stage_composition.sql`, `backend/internal/scenes/composition.go`, `capture.go`, `http_composition.go`, and composition/capture tests; routes `/stage-composition`, `/update-current-scene`, `/save-as-new-scene`.
- `Construction/current-state.md` still says visual composition/capture is not implemented, but that conflicts with this current code/migration/routes/tests and is stale. Kernel 85’s browser run did not separately click the two capture buttons, so their UI wiring has Go-test rather than browser proof.

### Cohort scene progression
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Directors create serialized Cohorts, assign Show participants, and set independent current Scenes without moving other cohorts or Ungrouped users. Snapshot resolution restores the proper cohort scene on reconnect.
- Evidence: migrations `096_kernel85_cohorts.sql`, `backend/internal/cohorts/`, `backend/internal/world/snapshot.go`, `frontend/lib/stage-runtime/kernel85-cohort-tools.js`, and cohort tests; Kernel 85 live two-browser proof.
- Ungrouped is deliberately not part of sustained-play progression. The Kernel 85 smoke script has unsafe cleanup that can delete a real Show’s serial-counter row; do not run it against persistent Shows until repaired.

## 4. Shared stage / VTT runtime

### Pixi shared stage, maps, pan/zoom, fullscreen fit
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- First Theater and Catharsis share the Pixi engine. It renders maps, stage nodes/elements, camera pan/zoom, theater/fullscreen modes, fit/reset, token UI, overlays and real-time snapshot synchronization.
- Evidence: `frontend/lib/stage-runtime/runtime.js`, `scene-nodes.js`, `token-ui.js`, `state.js`, `session-sync.js`, `socket.js`, `frontend/lib/victory-stage-camera.js`, plus `tests/stage-runtime/`; `backend/internal/world/`, `venues/map.go`.
- The Cave remains a dense proving-ground interface; current tests cover behavior, while exact visual/theatrical quality is not comprehensively screenshot-proven.

### Tokens, movement, and persistent stage actions
**Status:** IMPLEMENTED

- Token/element placement and move/remove/reveal/overlay actions persist as Actions and are reconstructed in venue snapshots; Show-owned state survives linked Session replacement.
- Evidence: `backend/internal/actions/token.go`, `place.go`, `remove.go`, `stage_state.go`, `backend/internal/world/snapshot.go`, `actions/token_dbtest_test.go`, `stage_elements_dbtest_test.go`.

### Grid configuration
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Supported theater venues expose persisted square/hex/none grid configuration: cell size, origin offsets, hex orientation, line width/style/opacity and visibility.
- Evidence: migration `028_kernel47_grid_config.sql`, `backend/internal/venues/grid.go`, `frontend/lib/stage-runtime/map-grid.js`, `grid_test.go`.
- This is a display/configuration grid, not a declared physical scale/unit model.

### Measurement, distance, diagonals, fog, and per-token visibility
**Status:** DEFERRED / NOT IMPLEMENTED

- No scale-aware ruler, distance display, diagonal/path rule, map-unit model, fog-of-war, or generalized per-participant object visibility was found in the runtime, API, schema, or tests.
- Evidence: grid schema/config above contains no units; `backend/internal/projection/projection.go` explicitly says per-participant object visibility is deferred; no measurement/fog implementation was located by repository search.
- Local participant projections exist, but they are scene presentation layers, not fog/visibility.

## 5. Dice, actions, and commands

### Canonical dice, command entry, history
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- The Go parser/RNG persists canonical `roll/dice` Actions; `/roll`/`/r`, dice tray, and skill-linked rolls use it. Rolls are rate-limited and idempotency-aware.
- Evidence: `backend/internal/dice/`, `backend/internal/actions/dice.go`, `commands/`, `network/dice_roll_test.go`, `actions/dice_test.go`, `frontend/lib/stage-runtime/dice.js`.
- Current roll submission is Director/Producer/Operator-only (`canActDiceRoll`); ordinary Cast cannot submit a personal Cohort roll. Ordinary parsed expressions do not explode unless they contain `!`; however, venue skill-click rolls deliberately generate `!` expressions, so that product decision should be understood as an explicit system-generated operation rather than an engine default.

### Targeted/private and spatial theatrical dice
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Kernel 86 resolves Show/Cohort/Director/Private audiences server-side, filters restricted historic `roll/dice` Actions from snapshots, sends only to recipients, and projects queueable transient/static dice effects that can be pinned/dismissed by a roller or Director+.
- The current Kernel 86A working tree further renders individual dice into deterministic, map-relative world positions, delays the HUD announcement until settling, preserves pinned positions through pan/zoom, and renders only the canonical explosion chain it receives.
- Evidence: current uncommitted `backend/internal/rollaudience/`, `stageeffects/`, `actions/dice.go`, `network/hub.go`, `world/snapshot.go`; `frontend/lib/stage-runtime/dice-projection.js`, `runtime.js`; `kernel86_*` Go tests, `tests/stage-runtime/dice-projection.test.js`, and `backend/internal/dice/dice_test.go`.
- Kernel 86 has a PASS reportback; 86A has source/tests/smoke script but its own spec still says “READY FOR IMPLEMENTATION” and no reportback was found. Therefore spatial behavior is verified from current implementation but its live/browser completion evidence and commit status are uncertain.

### Cues and game events
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Director/Crew authoring and idempotent GO support the three current Cue action types, player-curated enabled Cue buttons, durable execution history, and stage invalidation.
- Evidence: migration `045`, `backend/internal/cues/`, `actions/game_event.go`, Cue tests, `frontend/lib/stage-runtime/action-router.js`.
- Object reveal/hide and interaction enable/disable Cue types are expressly deferred pending object/state mapping.

## 6. Characters and game state

### Character sheets, mechanics, history, and journals
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Character Cards, Socio creation chapters, parentage/archetype/skill flow, Face/Mechanics/History presentation, values/quotes/bio overrides, private journals and Character skills are implemented.
- Evidence: migrations `016`, `031`–`035`; `backend/internal/characters/`; `frontend/venues/greenroom/`, `first-theater/`, `catharsis/`; Character tests.
- A Greenroom placeholder reports “This page is not implemented yet” for an uncovered page/surface; skill rule-links are implemented but broader mechanics linking is not.

### Inventory and durable items
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Durable equipment catalog and Character inventory exist, including idempotent purchases and a Greenroom inventory view.
- Evidence: migrations `059`, `062`, `063`; `backend/internal/merchant/`, `/api/venues/{slug}/equipment`, `/api/characters/{id}/inventory`, `frontend/venues/greenroom/inventory.html`; Kernel 73 integration evidence.
- Equipment CRUD is HTTP-only; no dedicated frontend item-editor page is wired.

### Socio HP/status mechanics
**Status:** IMPLEMENTED

- Eight canonical HP pools, status registry/application/clear, Character state and cohort-filtered movable Game Status are real persisted mechanics.
- Evidence: migration `097`, `backend/internal/socio/`, `/api/socio/statuses`, `/api/shows/{id}/characters/{id}/socio/*`, `socio_test.go`, Kernel 85 proof.

## 7. Socio rules-native support

### Tutorial, Kessa, locked door, Ra, and local progression
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Catharsis supports a staged Socio tutorial: Kessa participant interaction, five authored stance choices, server-authoritative Haggle roll, inventory purchase, milestone-gated locked-door hotspot, private Director note, Ra guided dialogue, and participant-local stage projection.
- Evidence: migrations `057`–`080`; `backend/internal/merchant/`, `dialogue/`, `projection/`, `storysofar/`; `frontend/lib/stage-runtime/participant-interactions.js`; Kernel 73–75 reportbacks and DB/integration tests.
- The merchant packet is seed-authored rather than Director-editable, Equip Mode is deliberately Catharsis-only, and this family lacks browser screenshot proof despite strong DB-backed flow tests.

### Sustained Socio play
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Cohorts, canonical HP/status, scene progression, aftercare, Story So Far, Director journals and tutorial completion establish persistent play beyond the tutorial.
- Evidence: migrations `066`–`080`, `096`–`097`; `cohorts/`, `socio/`, `storysofar/`, merchant aftercare routes; Kernel 75 and 85 reportbacks.
- There is no audited generalized Socio turn/economy engine for Major/Minor actions, reactions, barter currency accounting, or automatic status rules. Existing Kessa Haggling is a bounded authored packet, not proof of all Socio economy/rules automation.

### Stance and social action support
**Status:** PARTIAL / INCOMPLETE

- Five Kessa conversational stances and a Haggle skill check are implemented for the seeded Catharsis packet.
- Evidence: `merchant_packets`, `participant_interactions`, `backend/internal/merchant/`, Kernel 73 reportback.
- No generic stance system, reusable social-action/reaction framework, or director authoring UI for the packet was found; do not describe Socio’s full social rules as generally automated.

## 8. Storyboards and timeline

### Boards, cards, permissions, sharing, export
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Blank boards support ownership/grants, capability matrix, columns/bands/rows/cards, hidden-card filtering, locks, drag/drop plus keyboard/modal fallback, card images/lightbox, live WS updates and structured JSON export.
- Evidence: migrations `090`–`092`, `094`; `backend/internal/storyboards/`, `frontend/venues/storyboards/`, extensive `*_dbtest_test.go`; Kernels 80–81 reportbacks.
- Concurrent structural creation retains a documented `sort_order` race; UI/live visual review remains a human-judgment area.

### Timeline template and reference panel
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- The immutable Timeline template provides Beginning/Middle/Ending boundary columns, configurable four-field Reference Panel, Crew-content/Director-structure split and middle-mouse pan.
- Evidence: migration `095`, `storyboards/timeline.go`, `reference_panel.go`, `frontend/venues/storyboards/board.html`, Kernel 82 tests/reportback.
- Generic “+ Column (end)” can append after Ending because the server lacks boundary-aware insertion; the specialized UI works around it with a second reorder. Changing a Reference Panel field type requires that field to be empty.

## 9. Documents / eWrite

### Authoring, revisions, reading, and search
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- eWrite provides typed Ruleset→Series→Module→Publication→Section hierarchy, Markdown sanitization, append-forward revisions with 409 conflict protection, anchors/aliases, editor grants, Writer’s Room, Library, and Ruleset-scoped full-text search.
- Evidence: migrations `084`–`087`; `backend/internal/ewrite/`, `frontend/venues/writers-room/`, `library/`, eWrite tests and contracts; Kernels 78–79 reportbacks.
- Anonymous/public reading is schema-ready but deliberately unavailable; malformed object-link IDs have a documented raw-Postgres-error leak.

### Rule links and hierarchical export
**Status:** IMPLEMENTED

- Character skills, Cues, index cards, Storyboard cards, scene elements and dialogue Topics can link to rule sections; Module/Series/Ruleset export produces checksummed zip hierarchy.
- Evidence: migration `089`, `backend/internal/ewrite/`, link UI surfaces, `Construction/eWrite/ewrite-import-export.md`, Kernel 79 Goal C&E reportback.

### Socio publication seed
**Status:** IMPLEMENTED

- The install bootstrap seeds and reconciles the Socio v1.1 manuscript, hierarchy, Quickstart/Niava material, publication asset references and a 100-entry Skill Directory.
- Evidence: `main.go` bootstrap, `backend/internal/ewrite/`, `characters.Chapter4Skills`, `frontend/assets/rulesets/Sociov1_1.md`, eWrite seed tests.

## 10. Chat, presence, and communication

### Venue chat, OOC, messages, and note cards
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Venue chat/OOC commands, current speaker attribution, durable messages/backstage notes, Mailbox and note cards are implemented, with WebSocket delivery and Discord chat bridge support.
- Evidence: migrations `008`, `009`, `014`, `023`, `050`; `backend/internal/messages/`, `commands/ooc.go`, `actions/chat.go`, `network/discord_chat_bridge.go`; `frontend/lib/victory-mic-chat.js`, Mailbox UI and tests.
- No verified generic “In Character” tab was located; current IC identity behavior is spread across persona/character projections and venue interfaces rather than a unified documented tab.

### Presence and Discord voice integration
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- WebSocket presence, Storyboards Presence Tray and Discord gateway/voice-state/mic-control status exist.
- Evidence: `backend/internal/network/presence.go`, `identity/discord_*`, `network/discord_gateway.go`, related tests/routes.
- Victory does not carry Discord audio. Active-speaker detection and per-user audio volume are not truthfully available.

## 11. Commerce, cards, and transfers

### Kessa equipment commerce
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Kessa’s participant-local purchase flow, starting equipment seed, idempotency ledger and inventory persistence are implemented.
- Evidence: migrations `059`–`063`, `merchant/`, Kernel 73 full-chain test/reportback.
- It has no currency transfer/accounting and is not a generic merchant authoring system.

### Hands, deck/cards, and general transfers
**Status:** DEFERRED / NOT IMPLEMENTED

- No generalized hand/deck/card-zone system, player-to-player inventory transfer, or generic commerce ledger was found in schema/runtime/tests.
- Existing index cards and Storyboard cards are editorial/stage artifacts, not a player hand.

## 12. Audience, projection, and theatrical controls

### Audience program and spectator-safe projections
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Audience Program/Roster views use curated response types, while current scene, character/Face projection, player Cue buttons and map/stage snapshots provide spectator-facing experience.
- Evidence: `showruns/projection.go`, `shows.HandleShowProgram`, `scenes.HandleShowSceneProgram`, `world/snapshot.go`, Show Run/frontend program pages, security matrices.
- The audience experience is primarily venue/runtime-specific; there is no separately audited general public broadcast/streaming product.

### Local and theatrical stage effects
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Participant-local stage projection gives one player temporary presentation without changing the Show scene; Kernel 86 adds targeted dice Stage Effects (transient/static/pinnable).
- Evidence: migrations `067`–`068`, `projection/`, `stageeffects/`, `dice-projection.js`, Kernel 74/86 evidence.
- Effects currently cover the implemented local projection and dice use cases, not an open-ended Director stage-effects authoring system.

## 13. Persistence, recovery, and continuity

### Persistent state and reconnect
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- PostgreSQL persists core identity, Character, Show, Scene, Action, inventory, tutorial, Cohort and Socio state; snapshots rehydrate stage state and current cohort placement. Sessions can be resumed/revalidated.
- Evidence: migrations, `world/snapshot.go`, `shows/session_resume_test.go`, `network/session_control.go`, Cohort reconnect proof, `migrate.Run` in `main.go`.
- Venue coordination is intentionally in-memory and clears when its last live watcher leaves; it is not durable turn state.

### Story So Far, aftercare, account export/deletion
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Story So Far, face sheets, aftercare/review CSV, account export/download and tombstone deletion exist.
- Evidence: migrations `070`, `073`–`076`, `081`–`082`; `storysofar/`, merchant aftercare endpoints, `identity/account_export.go`, deletion tests, Kernel 75/77 reportbacks.
- These are distinct exports: account export and Storyboard/eWrite exports are implemented; there is no verified full-Show/whole-install content export product.

### Backup and restore operations
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Encrypted off-host backup and measured restore procedures, backup status and break-glass recovery tooling are documented and code-supported.
- Evidence: `Construction/Operations/victory-backup-runbook.md`, `victory-restore-runbook.md`, Kernel 77 proof, `identity/backup_status.go`, deployment/fresh-install scripts.
- Recovery email is still operationally blocked by Brevo approval; this is a hosted-readiness limitation.

## 14. Operator, hosted readiness, and security

### Migrations, bootstrap, and deployment
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Embedded checksummed migrations run before bootstrap; installation uses Docker Compose/Caddy/Postgres with fresh-install proof and known venue/reconciliation bootstrap.
- Evidence: `backend/internal/migrate/`, `backend/migrations/`, `Construction/deployment/fresh-install.md`, `scripts/smoke/fresh-install.sh`, `main.go`.
- Migration number `088` is absent but harmless under the checksum ledger; migration `093` is an unattributed hotfix. Neither should be silently assigned to a historical kernel.

### Security and privacy hardening
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Credential/API rate limits, body/frame caps, session WS revalidation, secure deletion/export, scoped serialization, image visibility and privacy boundary tests exist.
- Evidence: `backend/internal/ratelimit/`, `network/kernel77_*`, `identity/kernel76_auth_closure_test.go`, `Construction/Security/`, eWrite/image-visibility migration `086`, Kernel 76–77 reports.
- The new Kernel 86 roll privacy repair is uncommitted. A malformed eWrite object-link error remains cosmetic information leakage; provider email recovery is not operationally live.

## 15. Resident productions and reusable primitives

### Socio
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Socio is the fullest resident production: ruleset publication, Character creation/skills, Catharsis tutorial, Kessa, inventory, local projection, Story So Far/aftercare, Cohorts, HP/status and dice projection.
- Evidence spans `backend/internal/characters`, `merchant`, `socio`, `cohorts`, eWrite seed, Catharsis frontend, migrations `057`–`080`, `096`–`097`.
- It proves several reusable primitives, but not every Socio tabletop rule is automated (see §7).

### Timeline / Storyboards
**Status:** IMPLEMENTED WITH KNOWN LIMITATIONS

- Timeline is a reusable Storyboards template rather than a production-specific database artifact; generic board sharing, export, coordination and Reference Panel are reusable Victory primitives.
- Evidence: `storyboards/`, migrations `090`–`095`, timeline contracts and Kernels 80–83 reportbacks.

### Cartograph
**Status:** DEFERRED / NOT IMPLEMENTED

- No Cartograph resident production or drawing/toolset was found. Kernel 86’s pinned dice are a declared future seam for drawing, not a Cartograph implementation.

## 16. Implemented-but-rough areas

- **Dice projection:** IMPLEMENTED WITH KNOWN LIMITATIONS — actual Pixi projection and targeted privacy exist, but pixel-quality/animation screenshot proof is absent, and auto-explosion needs product semantics.
- **Stage runtime/Cave:** IMPLEMENTED WITH KNOWN LIMITATIONS — functional shared engine and controls, but Cave is explicitly a dense proving-ground UI.
- **Storyboards:** IMPLEMENTED WITH KNOWN LIMITATIONS — capable and tested, but structural concurrent-creation race, Timeline end-column loophole, and keyboard-inaccessible presence menu remain.
- **Socio operation UI:** IMPLEMENTED WITH KNOWN LIMITATIONS — core sustained play works; some capture and merchant/authoring interfaces are Go/API proven rather than browser-polished/proven.
- **Responsive/accessibility:** PARTIAL / INCOMPLETE — evidence screenshots show some desktop/mobile surfaces and Storyboards has keyboard fallback for card operations, but no repository-wide responsive or accessibility conformance implementation/test suite was found. Presence Tray’s action menu is specifically mouse-only.

## Appendix A — Kernel capability index

The repository does not contain recoverable formal reportbacks for every early kernel. The index therefore names only recoverable evidence and does not invent gaps.

| Kernel(s) | Verified introduction/material change | Later state |
|---|---|---|
| 1–5 | Early world/bootstrap, identity, GUI-map and reportback artifacts | Superseded/extended by later production and stage systems; thin historical evidence only. |
| 6 | Server action authority | Still foundational in `actions/authority.go`. |
| 7–8 | Presence attribution and identity surface repair | Extended by modern WS presence/Profile paths. |
| 9–11 | Greenroom/Trailers, Mailbox, note-card delivery | Still implemented, extended by newer Character/people surfaces. |
| 12–18 | Index cards, Workshop placement, production/venue roles, move-tool context | Still present; later Scene composition/runtime supersedes narrow early stage assumptions. |
| 21–24 | Venue chat, Showing model, Character cards/personas, Character-sheet links | Still implemented; Show/Show Run and Character selection later formalized it. |
| 29–33 | Pixi spike, Cave organization, canonical command/dice groundwork | Shared stage runtime and Kernel 86 later extend it. |
| 35–39 | Discord server/OAuth/chat/gateway | Still implemented with stated audio limitations. |
| 46–53 | First Theater map, grids, warehouse, capacity, canonical dice | Grid persists; K86/86A add real roll audiences and spatial projection. |
| 59A–65 | Character Face overrides/skills, Workbook, My People, Third Place | Still implemented, reused by People Picker. |
| 66–71 | Show Runs, Shows, Scenes, stage/Cues, tickets/roster/short codes | Current production spine; K85 adds cohort progression. |
| 72 / 72A | Shared stage runtime; embedded migrations; capability flags | Superseded duplicated venue runtime trees and slug allowlists. |
| 73 / 73A | Catharsis Kessa, equipment, local interactions; Scene composition | Current, bounded Catharsis/Scene capability. |
| 74–75 | Locked Courtyard/Ra tutorial; completion, Story So Far, aftercare | Current Socio tutorial/continuity. |
| 76–77A | security/hosted audit, export/deletion/backups, venue seed repair | Current hardening; recovery mail remains blocked. |
| 78–79A / goals | eWrite, Socio hierarchy, directory, links, navigation, export | Current document system. |
| 80–82 | Storyboards core/presentation, slugs, Timeline | Current boards/timeline; boundary and sort-order limits remain. |
| 83–84 | Coordination/presence, WS cleanup/reconciliation | Current ephemeral leader/turn; keyboard menu and Timeline add loophole remain. |
| 85 | Cohorts, independent scenes, Socio HP/status | Current sustained-play layer; smoke cleanup bug remains. |
| 86 / 86A | Targeted dice audience/projection; map-relative spatial landing and explicit-explosion safety | Both are present in the working tree. K86 has a PASS reportback; 86A lacks a completed reportback/commit, so live completion is uncertain even though source/tests verify its current behavior. |

## Appendix B — What earlier kernel makers do **not** need to report again

- **Private dice visibility was deferred in Kernel 52.** Kernel 86 now verifies Show/Cohort/Director/Private resolution, targeted WS delivery and filtered reconnect history. They should only discuss the still-open explosion semantics or player roll authority.
- **Independent participant scene progression was missing.** Kernel 85 implements Cohorts with server-authoritative independent current Scenes and reconnect restoration.
- **Group Leader / Current Turn was missing.** Kernel 83 provides explicit ephemeral handoff and live Presence Tray delivery; only initiative automation and keyboard entry remain open.
- **A rules/document system was missing.** Kernels 78–79 implement eWrite, manuscript seed, search, revisioning, rule links and hierarchical export; public anonymous reading is a deliberate scope decision.
- **Storyboard drag/images/timeline were absent in the first Storyboards pass.** Kernels 81–82 added those; report only the remaining structural race/boundary issue.
- **Account deletion/export/backup were absent at Kernel 76 audit time.** Kernel 77 implements them; recovery-email operations remains the exception.
- **Duplicated First Theater/Catharsis runtimes were a maintenance risk.** Kernel 72 replaced them with `frontend/lib/stage-runtime/`.

## Appendix C — Questions worth sending back to earlier kernel makers

### Grid / map measurement
Current state:
- persisted square/hex display grid exists;
- no verified map scale, ruler, path distance or diagonal model.

Ask:
- Was physical distance measurement deliberately deferred rather than intentionally excluded from Victory?
- Was a map-unit/scale model or diagonal rule selected for a later stage/drawing kernel?

### Dice authority and skill-roll explosion policy
Current state:
- canonical dice and private/projected spatial delivery exist;
- only Director/Producer/Operator submits rolls; normal expressions require `!`, but venue skill clicks generate it.

Ask:
- Was controlled-Character/player roll authority deliberately deferred, and what authority boundary was intended?
- Was the skill-sheet’s explicit exploding expression the intended rules policy for every venue skill roll, or should it be user/macro-selectable?

### Fog / hidden stage information
Current state:
- role- and roll-audience filtering exists; local projections exist;
- no generic per-token fog/visibility implementation.

Ask:
- Were Director-hidden objects/fog meant to be solved through a defined projection model, or deliberately kept out of the VTT scope?

### Scene composition
Current state:
- composition and capture implementation exists despite a stale current-state gap statement.

Ask:
- Did the older “visual composition not implemented” statement predate migration 064/Kernel 73A, or was another intended visual-composer behavior left out of the implementation?

### Socio generic rules
Current state:
- Cohorts, HP/status, Character skills and the Kessa packet work;
- Major/Minor economy, reactions and general barter bookkeeping are not verified as generic engines.

Ask:
- Which Socio mechanics were intentionally meant to remain authored/narrative rather than rules-engine automation?

### Timeline structure
Current state:
- protected boundaries exist, but generic append can put a column after Ending.

Ask:
- Was the generic append action knowingly left as an API loophole, or should boundary preservation have been universal?

## Appendix D — Remaining release-relevant gaps found during audit

| Gap | Classification | Evidence / why incomplete | Earlier maker consultation? |
|---|---|---|---|
| No scale-aware ruler, units, path/diagonal measurement | Basic VTT expectation | Grid config has visual cell size only; no measurement implementation | Yes: grid/stage intent. |
| No generalized fog/per-object player visibility | Basic VTT expectation / Victory-specific choice | `projection.go` declares it deferred; no runtime/schema path | Yes: intended theatrical replacement. |
| Cast cannot submit own rolls; skill sheets always generate exploding expressions | Socio / basic play expectation | `canActDiceRoll` boundary; `characters/venue_sheet.go` uses `StepExpression(..., true)` | Yes: dice authority/semantics. |
| Timeline can append after Ending | Presentation/product correctness | `storyboards.AddColumn` lacks boundary-aware insertion; K84 finding | Yes: Timeline maker intent. |
| Storyboard structural `sort_order` race | Operational/product correctness | K81A reconciliation and concurrency tests document hazard | Possibly; implementation owner. |
| Presence action menu inaccessible by keyboard | Accessibility | Kernel 83 and Storyboards accessibility contract | Yes: K83 scope intent. |
| K85 smoke cleanup can corrupt cohort serial counter | Operational/release blocker | `current-state.md`, K86 reportback | No; direct regression fix is clear. |
| No Director UI for merchant packet/equipment management | Socio requirement / polish | Seed packet; equipment CRUD has no page | Yes: K73 authoring intent. |
| Recovery email delivery blocked | Operational/release blocker | K77 reportback/Security notes: provider approval | No; external provider state. |
| No comprehensive visual/browser proof for recent Socio/dice UI | Presentation/polish | K73, K85 capture, K86 reportbacks | No; verification gap, not absent architecture. |

**This document describes verified current implementation, not the desired final product.**
