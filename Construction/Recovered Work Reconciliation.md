# Recovered Work Reconciliation

**Audit date:** 2026-08-11  
**Authority:** current repository and schema first; current uncommitted Kernel 86/86A work is included.  
**Companion:** `Construction/Everything Implemented.md` answers what exists. This document reconciles historical unfinished intent to the smallest current release set.

## 1. Executive Summary

Most historical deferrals are closed. The recovered record collapses into six material pre-freeze concerns: player-controlled Character rolls; a measured-tabletop decision (if physical maps are a launch promise); bounded scene-object state/visibility; Director interaction authoring; Storyboard correctness/accessibility repairs; and operational/build hygiene. The first four are product choices as much as engineering work. Existing work already supplies the foundations; none calls for a replacement VTT architecture.

The largest asymmetry is authoring: Scenes, Cues, stage composition, participant interaction execution, equipment, dialogue, and spatial dice are real, but several intended authored experiences still require seeds, raw HTTP, or developer configuration. Socio’s tutorial is therefore a proof of the primitives, not proof that a Director can author a second short encounter unaided.

The serious technical integrity finding is the Kernel 85 smoke cleanup that can delete a persistent cohort serial counter. The immediate process integrity finding is that Kernels 86 and 86A are deployed/verified but deliberately uncommitted. Neither should be hidden behind feature planning.

## 2. Repository State Audited

- Git `HEAD`: `b5a8787` (Kernel 85); current working tree contains Kernel 86 and 86A changes, documents, tests, and smoke scripts.
- Kernel 86A now has `Construction/OperatorLogs/kernel-86A-reportback.md`, PASS, live proof, but remains uncommitted. It supersedes the earlier audit’s uncertainty caused by the spec’s stale “READY FOR IMPLEMENTATION” header.
- Schema source is embedded `backend/migrations/`; current relevant increments include `064` Scene composition, `096` Cohorts, and `097` Socio mechanics.
- DB-backed tests are present but require `TEST_DATABASE_URL`; local audit execution could only run non-DB dice and Node spatial-projection tests. The reportbacks record complete DB/live proof; that is historical corroboration, not a substitute for a configured local run.

## 3. Method and Source Priority

I read current source/routes/schema/tests first, then the uncommitted implementation, Kernel 85/86/86A reports, `Everything Implemented.md`, current-state and reconciliation/operator records, then recoverable kernel specs/reports from the early action/identity/map era through Storyboards and Socio. Keyword hits were read in context where they described a real intended capability; plans and TODOs alone were excluded.

## 4. Historical Recovery Index

| Era / sets inspected | Recovered intent that still matters | Reconciliation result |
|---|---|---|
| K6–18 action, venue, map, cards | server action authority; early map/grid; placements | Authority and placement are closed; measurement was never completed. |
| K21–33 chat, Showing, Character cards, Pixi/Cave | role-aware chat, canonical Character, stage runtime | Modern Show/Scene/shared Pixi model supersedes early Session/Cave center. |
| K46–53 map/grid/warehouse/dice | configurable grid and canonical server dice | Grid/dice closed; physical measurement and player roll authority remain. |
| K59A–65 Sheets, Workbook, People | Character projection, skills, identity/social surfaces | Mostly closed; acting identity is still not converged through chat/rolls. |
| K66–75 Show/Scene/Cues/tutorial | Show-run participation, scene lifecycle, Cues, local projection, Kessa/Ra/continuity | Core closed; Cue object-state and authoring remain bounded/open. |
| K76–79 eWrite/security/hosted | security, deletion/export/backups, documents and links | Mostly closed; recovery email external blocker, raw DB error, public reading deliberately deferred. |
| K80–84 Storyboards/coordination | usable boards, Timeline, leader/turn, WS hardening | Board correctness/accessibility repairs remain. |
| K85–86A Cohorts/Socio/dice | sustained play, scoped dice/privacy, spatial dice | Core closed; smoke cleanup, Cast rolls, visual proof/polish remain. |

## 5. Recovered Work Reconciliation Matrix

| Recovered request | Classification | Current result / remaining truth |
|---|---|---|
| Canonical server action authority | CLOSED — CURRENT IMPLEMENTATION SATISFIES INTENT | `actions/authority.go` and scope-specific checks now govern mutations. |
| Persistent production spine rather than Session-only play | CLOSED — SUPERSEDED BY BETTER CURRENT MODEL | Show Run/Show/Scene placement owns durable state; Session is runtime. |
| Duplicated venue runtime trees | CLOSED — SUPERSEDED BY BETTER CURRENT MODEL | K72 shared `frontend/lib/stage-runtime/` is the behavior path. |
| Show participation through consent/roster | CLOSED — CURRENT IMPLEMENTATION SATISFIES INTENT | K71 two-punch tickets and selected Character provide it. |
| Scene library, staging, recall/capture | CLOSED — CURRENT IMPLEMENTATION SATISFIES INTENT | K73A composition/capture exists; prior current-state denial is stale. |
| Private roll visibility | CLOSED — CURRENT IMPLEMENTATION SATISFIES INTENT | K86 audience resolver, targeted delivery, snapshot filtering. |
| Spatial dice/pinning and explicit explosion safety | CLOSED — CURRENT IMPLEMENTATION SATISFIES INTENT | K86A world-space projected dice; normal expressions only explode with `!`. |
| eWrite/revisions/search/rule links/export | CLOSED — CURRENT IMPLEMENTATION SATISFIES INTENT | K78–79 delivered the intended document system. |
| Storyboard baseline cards, sharing, drag, images/timeline | CLOSED — CURRENT IMPLEMENTATION SATISFIES INTENT | K80–82 completed these historical gaps. |
| Account deletion/export/backup | CLOSED — CURRENT IMPLEMENTATION SATISFIES INTENT | K77; recovery mail is separate. |
| Grid display/configuration | CLOSED — CURRENT IMPLEMENTATION SATISFIES INTENT | Square/hex configuration, origin, cell size and display options persist. |
| Measured tabletop: scale/unit/ruler/path/diagonal policy | OPEN — IMPORTANT BUT NOT RELEASE-BLOCKING | No model beyond visual grid; launch importance depends on whether maps represent distance. |
| Generalized object visibility / Director-only / selected-player / Cohort | PARTIAL — SMALL BOUNDED GAP | Role/private visibility exists per element and rolls; no unified dynamic projection state. |
| Spatial fog/exploration | DEFER — POST-RELEASE | No foundation beyond scene projection; distinct from object visibility and no launch production dependency. |
| Cue reveal/hide and enable/disable object interaction | OPEN — IMPORTANT BUT NOT RELEASE-BLOCKING | Explicitly deferred in `cues/types.go`; direct stage hide/reveal exists but not canonical Cue state. |
| Director-authored interactions/merchant/dialogue/clue/door | PARTIAL — PRODUCTIZATION GAP | Runtime/data primitives exist, but merchant packet is seed-authored and equipment UI is HTTP-only. |
| Scene capture/recall affordances | PARTIAL — PRODUCTIZATION GAP | Backend/UI exist; capture UI has no browser proof and needs release usability review. |
| Stage-effects authoring | DEFER — POST-RELEASE | Dice effects prove consumption; no evidence that open-ended effect authoring is launch-required. |
| Canonical acting identity across stage/chat/rolls | OPEN — REQUIRED BEFORE FIRST RELEASE | Show-selected Character is canonical for entry, but IC chat and player rolls do not consistently consume it. |
| Ordinary Cast/Player Character roll authority | OPEN — REQUIRED BEFORE FIRST RELEASE | `canActDiceRoll` permits only Director/Producer/Operator. |
| Socio generic stance/action economy | DEFER — POST-RELEASE | Kessa stance choices are a bounded packet, not an intended CRPG engine. |
| Socio short encounter authoring/setup | OPEN — REQUIRED BEFORE FIRST RELEASE | If Socio is launch production, a Director cannot credibly make a second encounter without code/seeds. |
| Story So Far meaningful-event coverage | PARTIAL — SMALL BOUNDED GAP | Story So Far works, but no clear evidence newer Cohort/status/item events are curated into biography rather than omitted/telemetry. |
| Generic In Character chat lane | PARTIAL — PRODUCTIZATION GAP | Chat/Character identity exist; no generic IC tab/path found. |
| Timeline Ending append loophole | PARTIAL — SMALL BOUNDED GAP | `AddColumn` can append after Ending; client workaround is not universal enforcement. |
| Storyboard structural sort-order race | PARTIAL — SMALL BOUNDED GAP | Documented K81A concurrent creation hazard. |
| Presence Tray keyboard entry | PARTIAL — SMALL BOUNDED GAP | Current essential coordination menu is right-click only. |
| K85 serial-counter smoke cleanup | OPEN — REQUIRED BEFORE FIRST RELEASE | Can corrupt a real persistent Show’s cohort creation. |
| Recovery email delivery | OPEN — REQUIRED BEFORE FIRST RELEASE | Code exists but provider account approval blocks delivery; account recovery claim must be truthful before public release. |
| Raw eWrite malformed object-ID database error | PARTIAL — SMALL BOUNDED GAP | Known error-surface leak; bounded validation repair. |
| Public anonymous eWrite reading | DEFER — POST-RELEASE | Explicit operator scope decision; authenticated Library satisfies current intent. |
| Hands/decks/transfers/generic commerce | DEFER — POST-RELEASE | No resident production requires it; inventory/Kessa is not this system. |
| Audience polls/applause/voting | DEFER — POST-RELEASE | Audience Program and safe observation exist; no evidence interaction is launch dependency. |
| Legacy session persona as participation authority | RETIRE — LEGACY CONCEPT | Superseded by K71 selected roster Character; retain compatibility only. |
| Site-wide active Character as Show authority | RETIRE — LEGACY CONCEPT | Preference/default only; must not become Show identity again. |
| Raw Cave Session as product center | RETIRE — LEGACY CONCEPT | Cave is a proving ground; Show-owned persistence is current model. |
| Legacy memberships/access grants as primary authority | RETIRE — LEGACY CONCEPT | Resolver uses them only additive fallback; do not extend them. |
| Session as durable performance owner | RETIRE — LEGACY CONCEPT | Current Show-owned state supersedes it. |

## 6. Detailed Findings by Band A–Q

### A — Map, grid, and spatial tabletop

The verified chain is **grid → configuration** only. `venues/grid.go` persists square/hex shape, cell size, offsets and visual treatment; the shared Pixi runtime renders it. There is no physical scale, unit, ruler, straight/path distance, or diagonal policy. New spatial work must converge on the shared Pixi world layer; DOM/Cave math is not a behavioral foundation. This is a single measured-tabletop decision, not six unrelated gaps.

### B — Visibility, hidden information, and projection

There is one canonical Scene/snapshot, then role/private filtering at several layers: action visibility, Scene element `toRoles`/`privateTo`, Director layers, roll audiences, and participant-local Scene substitution. This is not yet one reusable visibility primitive. `projection` correctly avoids creating alternate Scene state; it changes which authored Scene a participant sees. Generalized object visibility is a bounded convergence problem. Fog/exploration is a different spatial system and should not be smuggled into it.

### C — Scene-element state and Cues

Elements already carry visibility, lock/nameplate and interaction binding metadata; direct stage actions can hide/show elements. Cues intentionally implement only scene change, game event and Show variable. `reveal_object`, `hide_object`, `enable_interaction`, and `disable_interaction` are explicitly rejected by `cues/types.go` because no Show-scoped state derivation exists. The coherent missing capability is bounded canonical element state, then Cues operating it—not scripting.

### D — Director authoring and productization

Scene composition/capture and Cue CRUD have Director surfaces. Participant interaction execution, Kessa, Ra, door prompts and equipment show credible runtime primitives, but authoring is asymmetric: Kessa packet is migration seed content, equipment editing is API-only, and no single Director flow creates an interaction + binding + state/Cue path. This should consolidate as a bounded interaction-authoring/Director-control bundle, not independent merchant, door, NPC, dialogue and clue features.

### E — Identity, participation, roles, and access

`participation.ResolveParticipationContext` gives canonical roles priority, with legacy membership/access grants only additive fallback. Current tests expressly cover the historical "new canonical role resolves none" bug. I found no evidence that old rows bypass ticket-derived Player participation or grant elevated management. Risk remains integration complexity: the Catharsis stage bootstrap’s legacy access-grant requirement means ordinary modern membership alone may not establish a usable stage. That is a product/access reconciliation issue worth testing before launch, not grounds to revive legacy authority.

### F — Character participation identity

For a Show participant, the answer is selected Character on `show_run_roster_members`. `world/snapshot.go` uses it for theater context and local-projection lookup. Session personas and site-active Character remain compatibility/preference paths. The answer becomes inconsistent at chat and dice: no generic IC speaker mode was found, and Cast cannot roll their selected Character. This is one participation-identity consolidation group.

### G — Dice and roll authority

Server RNG, parser, tray, `/roll`, skill clicks, canonical Action history, private/show/cohort/director audience, history filtering, spatial map landing, pinning and banner timing are current and live-proven through 86A. Parser behavior is correct: only `!` explodes; renderer never creates explosions. Venue skill sheets deliberately call `StepExpression(..., true)`, so their always-exploding behavior is a rules-operation policy to confirm, not a parser defect. The live product hole is ordinary player submission authority, not dice mechanics.

### H — Socio rules-native play

HP pools/statuses, skills, Character flow, inventory, Kessa, Ra, locked door, Cohorts, Game Status, Story So Far and aftercare all exist. There is no reusable current stance, stance wheel/relationship layer, major/minor/reaction spending, modifier/assistance model, or generic barter ledger. Two Players and a Director can run the seeded tutorial/sustained cohort flow; they cannot author and run a short new Socio encounter through native Director controls without seed/code work. The concrete launch decision is whether Socio is marketed as a playable resident production beyond that seed.

### I — Communication lanes

Venue/OOC chat, durable Mailbox/note cards/backstage notes, Director communication and Discord bridge are separate implemented lanes. Their identity presentation is not yet coherent as a player-facing IC lane. A bounded IC mode should derive the current Show-selected Character server-side and retain accountable user ID; it must not accept a client-selected Character.

### J — Show lifecycle

The current model is sound: Show owns persistence, a Session is live runtime, a Showing is reviewable wrapper; current Scene and Action history survive Session replacement. No evidence requires a naming-only refactor. Rehearsal/live/closed behavior does not appear to be a universally formalized mode machine, but existing authority and Show/session control enforce the key operations. Treat mode refinement as later unless a concrete bypass is found.

### K — Story So Far and continuity

Story So Far, aftercare and history are implemented. The audit cannot prove that later Cohort shifts, significant Socio status changes, purchases or milestones emit intentionally curated biography events. This is a small integration audit: do not log movement/chat telemetry, but identify only events a player would expect in a story summary.

### L — Storyboards / Timeline hardening

The Ending append loophole, structural ordering race and mouse-only Presence Tray are real retained obligations. They are bounded correctness/accessibility work, not another board architecture. Timeline’s normal UI workaround does not protect API users.

### M — Library, assets, placement, and promotion

Modern Scene composition preserves reusable asset versus placement: assets/maps/elements are not equivalent to a specific Scene placement. A historical `promote_to_library` idea was not found as a current user capability. It should remain retired: current composition/capture is a more direct Scene-level promotion model. No evidence requires a general asset marketplace/lifecycle/licensing feature before release.

### N — Cards, hands, transfers, and commerce

Index Cards, Storyboard cards and equipment inventory have different meanings. No generic player hand/deck/zone/transfer engine exists. Nothing in Socio seed proof makes it a release dependency.

### O — Audience

Audience viewing and curated safe projections are real; audience is explicitly prevented from Cue-trigger/player authority. No historical evidence makes polls, voting or applause necessary for first release. Preserve the boundary.

### P — Accessibility, responsiveness, usability

Concrete evidence supports some mobile surfaces and Storyboard keyboard fallback for card operations. The coordination context menu has no keyboard entry, and multiple recent visual surfaces have no screenshot/aesthetic proof. This is enough to require focused release usability passes, not enough to claim global accessibility conformance failure.

### Q — Operational / release integrity

Backups/restore, deletion/export, migration checksum and WS hardening are substantial closes. Remaining concerns are: K85 smoke corruption hazard; recovery-email external provider block; malformed eWrite object-link errors; absent migration 088/unattributed 093 (bookkeeping, not behavior); and uncommitted 86/86A released code. These must be kept separate—application defects, external dependency, harmless ledger anomaly, and release hygiene respectively.

## 7. Consolidated Missing Primitives

1. **Participation identity in live play** — selected Show Character → IC speaker and Player-controlled canonical rolls. Do not merge with legacy persona migration.
2. **Bounded Scene object state and visibility** — role/private/static state plus Cue-driven reveal/hide and enabled/disabled binding. Fog remains distinct.
3. **Director interaction authoring** — configure an authored interaction, bind it to a Scene element, choose bounded outcome/state/Cue behavior, and prove a Director can run it without seed editing.
4. **Measured tabletop (conditional)** — map scale/unit/ruler/path/diagonal policy in shared Pixi. Build only if public launch claims tactical physical maps.
5. **Release integrity/hardening** — smoke repair, recovery email resolution/claim adjustment, validation leak, Timeline/order/accessibility repairs, and commit/release reconciliation.

## 8. Closed / Already Solved Historical Requests

- Private dice and reconnect history leak: solved by K86, extended by K86A spatial projection.
- Duplicated per-venue stage engines: replaced by shared stage runtime in K72.
- Player Show admission and Character selection: solved by K71 ticket/roster model.
- Rules/handouts/revision/search/export: solved by eWrite K78–79.
- Storyboard presentation, images, drag/drop and Timeline: solved by K81–82; only bounded hardening remains.
- Persistent Socio cohorts and HP/status state: solved by K85.
- Account deletion, export and backup/restore: solved by K77.

## 9. Legacy Concepts to Retire or Contain

| Concept | Status | Why |
|---|---|---|
| `current_session_personas` as authority | Retire after release | K71 selected roster Character is canonical. |
| Site-wide active Character as Show identity | Harmless compatibility | May serve a default/preference; must not authorize Show actions. |
| Raw Cave/Session admission as product center | Retire after release | Current product center is Show/Show Run; Cave is a proving ground. |
| Legacy memberships/access grants as primary role model | Contain before release | Resolver limits them to fallback; test stage bootstrap for legacy grant dependency. |
| Presence as durable attendance/turn history | Retire after release | Coordination deliberately clears when live watchers leave. |
| Operator/dev mechanisms as normal product routes | Contain before release | Operator override is legitimate; ordinary ticket/role flows must remain sufficient. |
| DOM-era stage math | Retire after release | Shared Pixi runtime owns new spatial behavior. |
| Index Card as universal card abstraction | Retire after release | Storyboard cards and inventory have separate semantics. |
| Session as durable performance owner | Retire after release | Show-owned persistent state is the current model. |

## 10. Corrections to Everything Implemented

1. **K86A status:** `Everything Implemented.md` said 86A had source/tests but no completed reportback and called live completion uncertain. Current repository now contains `kernel-86A-reportback.md`, PASS with 36/36 live assertions. Correct status: verified, deployed, uncommitted.
2. **Explosion behavior:** the current default parser/tray path is not auto-exploding. `backend/internal/dice/dice_test.go` proves ordinary expressions never explode without `!`. Skill-click expressions deliberately include `!`; distinguish this policy from parser behavior.
3. **Visibility wording:** generalized dynamic visibility remains absent, but static Scene-element role/private visibility and direct hide/show actions do exist. The missing piece is convergence/Cue-driven canonical state, not zero visibility support.

## 11. Authoring Asymmetry Table

| Capability | Runtime exists? | Backend exists? | Director UI exists? | Usable without code? | Release importance |
|---|---:|---:|---:|---:|---|
| Scene composition / stage elements | Yes | Yes | Yes | Mostly | High |
| Update Current Scene / Save as New | Yes | Yes | Yes | Mostly; needs UX proof | High |
| Cues: scene/event/Show variable | Yes | Yes | Yes | Yes | High |
| Cue element state / reveal/hide / enable | Partial | No canonical Cue writer | No | No | High if interactive scenes launch |
| Participant interactions | Yes | Yes | Thin | Not for a new rich interaction | High for Socio launch |
| Merchant packet / Kessa | Yes | Yes | No packet editor | No; seed-authored | High for second encounter |
| Equipment catalog | Yes | Yes | No dedicated page | HTTP-only | Medium |
| Dialogue packets / Ra | Yes | Yes | Thin/seed-led | No clear full authoring flow | High for Socio launch |
| Visibility / selected viewer control | Partial | Partial | Basic element controls | Not dynamic/authorable | High if hidden play promised |
| Stage effects | Dice only | Dice only | Pin/dismiss only | No general authoring | Low |
| Socio encounter setup | Seeded tutorial | Partial | Cohort/HP status controls | No | Required if Socio is launch claim |

## 12. Tier 0 — Immediate Blockers

1. **Repair and regression-proof the Kernel 85 smoke cleanup** — direct persistent-state corruption risk. XS.
2. **Reconcile/review/commit or explicitly roll back deployed K86/86A work** — current release provenance is ambiguous. XS operational action.
3. **Resolve recovery-email delivery or remove/qualify the public recovery promise** — a public account system cannot claim a recovery path that fails at provider delivery. S; external dependency.

## 13. Tier 1 — Required for Victory 1.0

1. **Player-controlled canonical rolls for a valid selected Show Character** — expected agency for serious shared play; build on K71 participation and K86 audience resolution. S.
2. **Participation identity completion: generic IC chat speaker derived server-side** — paired with player rolls, makes “who acts/speaks” coherent and accountable. S.
3. **Director-authored bounded interactions for Socio** — required only if Socio is publicly presented as playable beyond its one seeded tutorial; combine packet/binding/outcome controls, not an arbitrary dialogue engine. M.
4. **Bounded Scene object state + Cue controls** — required if release promises hidden/revealable/interactable Scenes; otherwise explicitly narrow launch claim to static shared staging. M.

## 14. Tier 2 — Strong Pre-100 Candidates

1. **Measured tabletop** — S/M, conditional on tactical physical-map promise; make one product decision on scale/unit/path/diagonal rules.
2. **Story So Far biography-event review** — S; wire only significant intended events.
3. **Scene capture and Director authoring usability proof** — S; browser/real-play validation rather than architecture.
4. **Legacy stage-access reconciliation** — S; establish that canonical ticket/membership participation actually reaches the stage without accidental legacy grant dependence.

## 15. Tier 3 — Polish / Hardening

1. Timeline Ending append enforcement (XS).
2. Storyboard structural ordering race (S).
3. Keyboard entry/focus behavior for Presence Tray actions (XS/S).
4. eWrite malformed object-link validation/error shaping (XS).
5. Spatial dice and interaction visual screenshot/reduced-motion/usability review (S).
6. Full responsive/accessibility pass over public launch surfaces (M, bounded by actual release routes).

## 16. Tier 4 — Post-Release

- Spatial fog/exploration.
- Generic stage-effects authoring.
- Generic deck/hand/zones and item/card transfer engine.
- Audience polls/voting/applause product.
- Broad asset marketplace/lifecycle/licensing.
- Analytics/gamification/XP.
- Full Socio CRPG-style stance/action/reaction/economy automation.
- Anonymous public Library reading, absent a launch publishing promise.

## 17. Dependency Graph

```text
K71 selected Show Character
  ├─ Player roll authority ──┐
  └─ Server-resolved IC chat ┼─> coherent accountable participation
                             │
Scene composition + element bindings
  ├─ bounded object state ───┼─> Cue reveal/enable controls
  └─ interaction authoring ──┴─> Director-authored Socio encounter

Shared Pixi map/grid ─> scale/unit policy ─> ruler/path/diagonal (conditional)

Release hygiene: K85 smoke repair + K86/86A provenance + recovery delivery
  └─> credible public operations
```

## 18. Candidate Kernel Bundles

### Candidate Bundle — Accountable Player Agency

Includes server-derived selected Character for Cast roll submission and IC chat, with audience/privacy behavior preserved. Excludes initiative, automated movement and arbitrary Character switching.

### Candidate Bundle — Authored Scene Interaction

Includes bounded element state, Cue reveal/hide/enable/disable, and a Director flow to author/bind a participant interaction with constrained outcomes. Excludes scripting, fog and alternate Scene state.

### Candidate Bundle — Socio Second Encounter Proof

Includes only the authoring controls needed to create a small non-seeded Socio encounter using existing Cohorts, HP/status, Character skills, inventory and dialogue/interaction primitives. Excludes universal rules automation, economy ledger and stance engine.

### Candidate Bundle — Measured Tabletop (conditional)

Includes physical map scale, unit, ruler, path measurement and declared diagonal policy in the shared Pixi runtime. Excludes fog, initiative and movement automation.

### Candidate Bundle — Release Integrity Sweep

Includes K85 smoke cleanup, recovery-delivery truth, K86/86A commit/provenance reconciliation, Timeline append repair, ordering race decision, keyboard coordination access, error shaping, and launch-route visual checks. Excludes broad new features.

## 19. Smallest Credible Public-Release Set

Victory can credibly release as a persistent theatrical VTT if it keeps its product claim precise: shared stage, Scenes, permissions, Show participation, Character sheets/rules, chat/OOC, documented access, player agency, persistence/reconnect, backups and usable Director configuration. The smallest additional set is:

1. fix the Tier 0 integrity/provenance/recovery issues;
2. let an ordinary valid Player act as their selected Character through canonical rolls and IC communication;
3. make at least one complete Director-authored interaction path usable without seeds/source (needed if Socio is launch production beyond the tutorial);
4. either provide bounded object-state/Cue control or explicitly state that launch Scenes do not support dynamic hidden/revealable interactables;
5. decide whether physical measurement is a launch expectation. If yes, ship the measured-tabletop bundle; if no, state grids are theatrical/configurational.

Everyday expectations already met: token movement, pan/zoom, Scene switching, rules/handouts, Character sheets, inventory, OOC/durable communication, permissions, snapshots/reconnect, save/restore, audience-safe views and targeted/private rolls. Fog, decks, generic commerce and Audience interaction are not necessary merely because conventional VTTs offer them.

## 20. Explicit Post-Release / Do-Not-Build-Before-100

Do not consume the remaining feature window on full fog/exploration, generic decks/hands, universal commerce transfers, Audience participation, asset marketplace, analytics, XP/gamification, arbitrary scripting, open-ended stage effects, or a universal Socio CRPG executor. Each is historically interesting or commercially plausible, but repository evidence does not establish it as a prerequisite for the current launch productions.

## 21. Contradictions / Uncertainties Requiring Grant

1. Is Socio being released as a play-ready product beyond the seeded tutorial? If yes, the Director interaction/second-encounter bundle is Tier 1; if no, defer it and accurately scope Socio.
2. Are grids meant to represent physical distance in the first public release? If yes, measured tabletop is Tier 1; if no, retain visual/theatrical grid language.
3. Does first release promise hidden/revealable individual Scene objects? If yes, bounded object state/Cue visibility is Tier 1; if no, do not imply fog/GM secrecy beyond current role/private layers.
4. Is ordinary Player self-rolling a non-negotiable play expectation, or is Director-mediated rolling an intentional theatrical format? Repository history calls it a deferral, not a settled product decision.
5. Should skill-click rolls always explicitly explode? Source does this intentionally, but no evidence settles whether it is Socio policy or inherited convenience.
6. What exact recovery promise will be public while the email provider remains blocked?

## 22. Appendix — Evidence Index

- Current implementation: `backend/internal/participation/resolver.go`, `world/snapshot.go`, `scenes/composition.go`, `scenes/capture.go`, `cues/types.go`, `projection/projection.go`, `actions/dice.go`, `rollaudience/`, `stageeffects/`, `socio/`, `cohorts/`, `storyboards/`, `ewrite/`.
- Frontend/runtime: `frontend/lib/stage-runtime/runtime.js`, `dice.js`, `dice-projection.js`, `kernel85-cohort-tools.js`, `participant-interactions.js`, Storyboards board UI.
- Schema: migrations `028`, `042`–`048`, `057`–`080`, `084`–`097`.
- Historical reconciliation: `Construction/current-state.md`, `OperatorLogs/kernel-history-reconciliation-through-83.md`, reportbacks K71, K73–77, K79–86A, and kernel specs K69–86A.
- Tests/proofs: package DB tests, `tests/stage-runtime/dice-projection.test.js`, `backend/internal/dice/dice_test.go`, K85/86/86A browser smoke scripts and reports.

**This audit reconciles recovered historical intent against the current Victory implementation. It is a release-prioritization aid, not a new roadmap.**
