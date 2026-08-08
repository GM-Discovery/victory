# Victory VTT — Canonical Roadmap

**Version:** 2.0  
**Status:** Active planning control document  
**Adopted:** 2026-07-30  
**Replaces:** `victory-track-roadmaps-v1.md` and `victory-master-actual-implementation-guide-v1.md`

---

## 1. Purpose and authority

This is the single canonical roadmap for Victory VTT.

It exists to answer five questions:

1. What Victory is becoming.
2. What has actually been built.
3. What remains incomplete, skipped, deferred, or unknown.
4. What the next three committed kernels are.
5. What provisional work follows after those kernels.

This roadmap controls planning. It does not replace:

- kernel specifications;
- kernel reportbacks;
- the operator log;
- operator notes;
- the kernel-maker field guide;
- the project dictionary;
- security notes;
- deployment and development instructions.

Those files record execution evidence and durable implementation knowledge. They do not compete with this roadmap for sequence control.

### 1.1 Source-of-truth order

When documents disagree, use this order:

1. **Repository behavior and current database state**
2. **Current kernel reportbacks and browser-visible evidence**
3. **Operator log and current operator decisions**
4. **This roadmap**
5. **Older roadmap text and provisional kernel plans**

A roadmap entry never proves that a feature exists. A PASS reportback without adequate evidence must be treated as questionable until reproduced. A feature that exists in the repository but was built under a different number or name remains real and must not be rebuilt merely to satisfy an old plan.

### 1.2 One-file rule

Victory will maintain one roadmap file.

Historical plans may remain archived, but they are not active control documents. New planning must be incorporated here rather than creating another parallel roadmap.

---

## 2. Product doctrine

Victory is Grant Murray’s dream virtual tabletop: a **rules-native, theatrical, persistent environment for play, performance, creation, and teaching**.

It is not a kitchen-sink productivity application. It may gain broad capabilities when they serve real tabletop, theatrical, writing, storyboarding, or educational work, but it should not accumulate unrelated business software merely because it can.

Socio: Stories of Us is Victory’s flagship rules integration and living reference implementation.

The platform split remains:

```text
Victory owns:
- identity and authority
- persistence
- locations, lots, venues, productions, shows, scenes, and sessions
- elements and placements
- projection and audience differences
- theatrical runtime and presentation
- documents and reusable creative surfaces
- integration contracts

A rules integration owns:
- game-specific terminology
- character schemas
- values, resources, health systems, moves, oracles, and procedures
- rules-specific presentation
- game-specific tutorials and guided play

Socio is the first deep proof of that contract.
Anthology packages prove the contract is reusable.
```

### 2.1 Implementation rhythm

The preferred rhythm is:

```text
Build or repair a reusable Victory primitive
→ exercise it through a real game or production need
→ prove it in the browser with the correct roles
→ record the reusable integration seam
→ preserve skipped and deferred work explicitly
```

Do not build generic infrastructure for months without a lived use. Do not solve a Socio requirement by creating a second authority model, action stream, character truth system, Scene system, or document universe.

### 2.2 Dream-VTT boundary

Victory’s scope is discovered through use.

New major systems may be added when Grant encounters a real need while writing, running, teaching, staging, or publishing through Victory. Unknown future needs are legitimate, but they are not advance permission to build speculative systems.

---

## 3. Governing project truths

The following decisions are stable unless this roadmap explicitly records a change.

### 3.1 World and production hierarchy

```text
Location
→ Lot
→ Venue

Location
→ Production
→ Show Run
→ Show
→ Show Scene Placement
→ Scene

Show
→ persistent theatrical state

Session
→ temporary live attendance and interaction
```

A Scene is reusable. A Show Scene Placement stages a Scene for one Show. A Show owns persistent performance state. A Session is a temporary live occurrence and must not silently become the durable owner of Show truth.

### 3.2 Identity and character

- User identity is accountable and server-owned.
- A Character is a persona, not an alternate account.
- A player may choose and switch among eligible Characters.
- Character-local projection clears or changes appropriately when shared Scene state advances.
- Greenroom work and venue trays must consume shared character truth rather than copying it.
- Private Journal content must not leak into public projection or game events.

### 3.3 Authority

- The server is authoritative.
- Client payloads are requests, not facts.
- Role, identity, membership, Character access, and trusted game events are resolved server-side.
- Operator authority is infrastructure authority outside the normal app.
- Producer is the highest ordinary in-app authority, not server ownership.
- Venue capabilities fail closed.
- Access is granted rather than assumed.

### 3.4 Event and history model

- Recorded Actions are append-forward.
- Past recorded Actions are not destructively rewritten.
- Presence is ephemeral and not canonical history.
- Participant Interaction may remain local or private.
- Cues are shared theatrical operations.
- Buttons, forms, and commands should invoke the same domain operations when they represent the same action.

### 3.5 Consent and audience

- Two-punch consent is the default.
- Open enrollment is optional and explicit.
- Audience access and projection are curated.
- No ticket should mean no private Venue visibility or entry unless a specific admission policy says otherwise.

### 3.6 Equipment and Socio baseline

- Equipment remains inventory truth rather than automatic currency arithmetic.
- The current Socio equipment baseline includes the larger equipment library and Kessa’s smaller sale inventory.
- Purchases do not automatically deduct coins unless a later explicit mechanic is designed and approved.
- Catharsis remains the live Socio tutorial and playable onboarding environment.

### 3.7 Runtime reuse

Shared stage behavior belongs in the shared stage runtime rather than being copied across Venues. New theatrical surfaces should reuse existing stage, Scene, Cue, Element, placement, authority, and projection systems whenever those abstractions fit.

---

## 4. Sequence management

### 4.1 Three-kernel committed horizon

Only the next three kernels are committed.

A committed kernel may be renumbered if the repository already contains that number. It may be split or given a continuation suffix when evidence shows that is necessary. Its purpose may not silently change.

### 4.2 Provisional horizon

Everything after the next three kernels is provisional.

Provisional ordering expresses dependency and current product judgment. It may change when:

- a real user exposes a more urgent need;
- a security audit exposes material risk;
- a kernel reportback proves an assumption wrong;
- a game integration reveals a missing primitive;
- Grant changes product direction.

### 4.3 Numbering

Before issuing any kernel:

1. inspect the repository;
2. inspect the operator log;
3. identify the next unused real kernel number;
4. confirm no uncommitted or side-branch kernel already uses it.

This roadmap currently refers to the next likely sequence as Kernels **76–78**, because the reported implementation has advanced through Kernel 75. The kernel maker must verify those numbers before creating files.

### 4.4 Continuation kernels

Use `A`, `B`, or a clearly named repair kernel when finishing or proving already-shipped work.

A continuation does not rewrite the original reportback. It records the additional work and evidence.

### 4.5 Explicit skip rule

Old requirements may be skipped when they are obsolete, replaced, unnecessary, or poorly timed.

Every skip must record:

- the skipped requirement;
- the source plan or kernel that contained it;
- why it is being skipped;
- whether it is replaced, deferred, or abandoned;
- any consequence or open risk.

Silence is not a skip decision.

---

## 5. Actual implementation baseline

This baseline reconciles the reported repository state through **Kernel 84** (Canonical Reconciliation & Runtime Cleanup, 2026-08-08). §5.1–§5.6 reconcile through Kernel 75, unchanged since v2.0; §5.7 is new. Full per-kernel evidence (titles, dates, reportback paths, capability summaries, and self-flagged open findings) for Kernels 76–83 lives in `Construction/OperatorLogs/kernel-history-reconciliation-through-83.md` — this section summarizes capability bands only. The next audit must verify this baseline against the live repository.

### 5.1 Foundation through Kernel 52

Victory established:

- Go backend and PostgreSQL persistence;
- Docker as deployment truth;
- host-Go plus Docker-Postgres as a development convenience;
- Locations, Lots, Venues, Libraries, Elements, placements, Sessions, participants, and Actions;
- authentication, sessions, password credentials, password reset, invites, memberships, and roles;
- server-resolved identity;
- WebSocket presence and action delivery;
- speech, reactions, chat, note cards, mailbox, index cards, reveal/hide, overlays, and stage placement;
- Greenroom and Trailers profile/character surfaces;
- early production, invitation, and authority surfaces.

These early systems contain known old assumptions and must not be presumed secure or architecturally complete merely because later kernels use them.

### 5.2 Kernels 53–60 — Character and command spine

**Kernel 53–56:** Character onboarding vertical established.

**Kernel 57:** Partial. Some multi-Character and onboarding repair work remained unproven or incomplete.

**Kernel 58:** Partial in its original form. Face and projection semantics were specified, but the visible behavior was not fully completed there.

**Kernel 59:** Infrastructure subset. Command registry, resolver, execution routes, and receipts shipped, but the larger value/override/event vision did not all ship under this number.

**Kernel 59A:** PASS. Greenroom Face controls, shared venue projection, projection invalidation, Face/Mechanics tray tabs, Director locks and overrides, and browser evidence completed the most important Character projection gap.

**Kernel 60:** Backend vertical largely proven. Browser sheet/click/event authority and trusted `game/event` boundaries remained a reported concern and must be included in the current audit rather than assumed resolved.

### 5.3 Kernels 61–65 — Player identity and social spine

**Kernel 61 / 61A:** PASS. Trailer Player Workbook, Face Compiler, legacy profile migration, public player Face linking, live profile updates, and reauthenticated email change.

**Kernel 62:** PASS. My People / private directional relationship records, notes, journals, follow-ups, archive state, and shared-Production context.

**Kernel 63:** PASS. Fixture/navigation repair work.

**Kernel 64:** PASS. Dedicated database-test isolation and live-database safety gate.

**Kernel 65:** PASS. Third Place Headshot Commons and associated shared-profile projection.

These kernels advance Track V8 and social identity. They are not the stale Character or Scene kernels that older plans assigned to the same numbers.

### 5.4 Kernels 66–70 — Production, Shows, and Scenes

**Kernel 66:** PASS. Show Runs, roster, audience blocks, and Audience Program.

**Kernel 67–68:** PASS. Shows, Production onboarding, and Location authority.

**Kernel 69:** PASS. Reusable Scene Library and Show Scene Placements.

**Kernel 70:** PASS. Persistent Show-owned stage state, current Scene and variables, Cues, idempotent GO, rehearsal metadata, and curated player stage buttons.

Still open from this band:

- visual Scene composition and capture;
- broader browser proof where reportbacks relied mainly on HTTP or test evidence;
- stale fixture cleanup;
- any remaining authority assumptions found by audit.

### 5.5 Kernels 71–75 — Catharsis: The Golden Journey

Catharsis became the main branch temporarily and produced a complete guided tutorial journey.

**Kernel 71:** Account, Show access, onboarding, Profile, Character selection/creation, open enrollment, and Showtime entry.

**Kernel 72:** Shared runtime and Scene composition for the guided journey.

**Kernel 73:** Kessa and equipment interaction.

**Kernel 74:** Locked door, intention, Ra, and local projection behavior.

**Kernel 75:** Completion, Story So Far, Aftercare, My People, and waiting-for-humans handoff.

This work materially advanced:

- V3 Scene activation and composition;
- V4 Cues and guided transitions;
- V6 Shows, admission, and tutorial runtime;
- V8 Player identity and social continuity;
- S2 Courtyard;
- S3 Tutorial;
- part of S5 Equipment and bartering.

### 5.6 Catharsis follow-up ledger

Catharsis is no longer the main branch, but side repairs remain valid:

- roster auto-creation;
- aspect-ratio-safe door hotspots;
- final Kessa portrait handling;
- Ra dialogue refinement;
- gate reveal;
- backstage handoff;
- completion-panel refinement;
- fog, emphasis, and cache repairs;
- Profile redesign;
- favorite-TTRPG selectors;
- status levels;
- migration 080 verification;
- dice-test debt;
- other issues demonstrated by real tutorial play.

These should be handled as bounded side repairs or absorbed by later generic kernels. They must not retake the main roadmap indefinitely.

### 5.7 Kernels 76–84 — Hosted-readiness, eWrite/Documents, Storyboards, Venue Coordination, reconciliation

This band replaced the entire old six-kernel committed horizon (§14) with real, evidence-backed work under different titles. Grouped by capability, not by number — see `kernel-history-reconciliation-through-83.md` for the exact per-kernel ledger this summary was built from.

**76–77A — Hosted readiness, security, private-client readiness.** PASS/PASS/PASS. Closed the password-reset log leak and gated public signup behind `PASSWORD_SIGNUP_ENABLED` (now closed in production — Discord is the only account-creation path); WS auth-before-handshake, rate limiting, session-revocation sweep; `victory-recover` break-glass tool. Account deletion (tombstone model), self-service export, and encrypted off-host backup with a *measured* restore (RTO≈69s) are all real and proven — not aspirational. Self-service recovery email is code-complete and tested but delivery is blocked on a pending Brevo account-approval gate (re-checked Kernel 79A and again during Kernel 84's own reconciliation pass; still blocked as of 2026-08-08 — see `Security-notes.md` §3). Repaired the canonical Audition Hall venue seed gap (77A).

**78–79 (all passes) — eWrite Foundation / Victory Documents, realized.** PASS across every pass. This *is* what the old roadmap called "Victory Documents" (§V9) — built and shipped under the product name "eWrite." Real capability, not a foundation stub: Markdown pipeline (goldmark+bluemonday) and Postgres FTS search; typed collection hierarchy (Ruleset→Series→Module→Publication→Section); append-forward revisions with 409 save-conflict handling; stable section anchors; named editor grants; a generic reusable "directory" abstraction (proven via a 100-entry Skill Directory); object links from Cues, index cards, Scene elements, and dialogue Topics into rule sections; hierarchical zip export with per-file checksums; ruleset-wide landing pages, breadcrumbs, and scoped search. Proven at real scale against the actual Socio v1.1 manuscript (357KB / 43,207 words / 763 headings), not a toy fixture. Anonymous public reading remains a deliberate, operator-approved scope cut (schema-ready, not exposed) — see V9 below.

**80–83 — Storyboards and Venue Coordination.** PASS across all four. This is what the old roadmap's P2 called "semantic banded board foundation" and the intended foundation for a Microscope-style anthology proof (A1) — built as a generic, reusable, non-Microscope-specific capability on purpose (see A1 below). Ownership/sharing/role-capability matrix, columns/bands/rows/cards with per-viewer hidden-card WS filtering; CSS Grid presentation with drag-and-drop and a full keyboard/modal fallback; a built-in immutable Timeline template with server-enforced Beginning/Ending boundary protection and a configurable Reference Panel; structured JSON export. Kernel 83 added a second, independent generic platform primitive on top of it — live per-venue-session Group Leader/Current Turn coordination (`backend/internal/venuecoordination`), explicitly *not* a game-rule/turn-order system (see the new V11 below) — with Storyboards as its first, but not only intended, consumer.

**84 — Canonical Reconciliation & Runtime Cleanup.** This kernel. Repaired the WS context-lifetime bug Kernel 83 flagged (`storyboards/ws.go` was reusing a 5-second handshake-scoped context for a connection's entire life); fixed a genuine test-isolation bug in the long-flagged `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent` flake (a hardcoded absolute revision-number assertion, not database drift as previously assumed); found and documented (not fixed — see the skip/defer ledger) a real Timeline boundary-invariant gap in `AddColumn`; brought this roadmap, `current-state.md`, and the accessibility/security operator docs current through Kernel 83. Full ledger: `Construction/OperatorLogs/kernel-84-reportback.md`.

---

## 6. Track map

Victory retains five planning tracks. They now live in this one file.

| Track | Purpose | Current success condition |
|---|---|---|
| **V — Victory Core** | Reusable VTT, production, creative, document, and teaching primitives | A dream VTT with durable shared systems rather than isolated feature pages |
| **S — Socio** | Make Socio teachable and sustainably playable | Players can learn, create, play, track all major systems, and continue beyond the tutorial |
| **A — Anthology** | Prove the integration boundary through bounded game packages | Three meaningfully different small packages run through reusable Victory capabilities |
| **C — Concierge and Distribution** | Make paid hosted/private use and personal self-hosting practical | A client can receive a bounded offer, and a personal user can run Victory without private support |
| **O — Operational Integrity** | Preserve security, evidence, migrations, performance, and operator sanity | Strangers can use Victory without unexamined trust gaps, and future builders can reproduce it |

Operational Integrity is cross-cutting. It is not deferred until the product is “finished.”

---

# 7. Track V — Victory Core

## V1. Character truth and projection

### Established

- Character onboarding and ownership;
- active Character/persona selection;
- Face and Mechanics projection;
- Director locks and overrides;
- player and Character identity separation;
- venue tray projection;
- player workbook/profile foundations.

### Open

- audit trusted game-event authorship;
- complete any missing manual and Director override behavior exposed by current UI;
- verify shared projection in all modern stage-runtime Venues;
- support game integrations with multiple health/resource systems;
- preserve private Journal isolation under every projection;
- normalize Character links into Documents, rules, images, and future learning surfaces.

## V2. Command, action, and rules-operation surface

### Established

- command registry and execution spine;
- HTTP command surfaces;
- Action persistence and broadcast;
- chat, speech, reactions, reveal/hide, placement, overlays, Cues, and GO;
- append-forward history.

### Open

- complete ruleset-defined values and operations where still missing;
- reject direct client forgery of trusted game events;
- ensure buttons/forms/commands converge on domain operations;
- expose integration-friendly registries for values, moves, oracles, and structured outcomes;
- complete contextual help and discoverability without making arbitrary database editing possible.

## V3. Scenes, composition, and storyboarding

### Established

- reusable Scenes;
- Show Scene Placements;
- Show-owned current Scene and variables;
- Cues and guided Scene transitions;
- Catharsis Scene composition through the shared stage runtime;
- **semantic banded card boards (Kernels 80–82, PASS)** — bands stacked vertically, titled rows/bands, a finite horizontal sequence of columns, lockable bands, custom column labels, cards with front/back content and images, and structured JSON export preserving order/hierarchy/placement. Live per-viewer hidden-card filtering. A built-in immutable Timeline template (three columns, server-enforced Beginning/Ending boundary protection, a configurable four-field Reference Panel) ships alongside plain Blank boards. This *is* the generic capability the old plan (§13 P2) described — proven, not still-required. See `Construction/Storyboards/` for the full design contracts.

### Explicitly not built (see skip/defer ledger for why)

- **nested/beneath cards and merged cells/regions** — the original P2 wish-list item. Kernel 80's spec asked for it; Grant's own post-deploy review (Kernel 80 reportback) confirmed a flat one-card-per-cell model matches how he actually wants to use it, and a "multi-card-per-cell" model was not carried into Kernel 81's presentation rework. Not a gap to fill later by default — see skip/defer ledger.
- **user-created Storyboard templates** (save-a-board-as-a-reusable-template, beyond the two built-in Blank/Timeline templates) — never requested, never speced.

### Required next systems

#### Visual Scene composition and capture

- compose a Scene from reusable Elements and placements;
- save stable transforms, visibility, labels, locks, and context;
- capture or revise a reusable Scene without duplicating runtime truth;
- preview role-specific projection.

#### Richer index cards

Index cards remain bounded Elements. **Established (Kernel 79 Goal C&E):** links to eWrite Documents/anchored sections. **Established (Kernels 80–81):** image references/attachments, thumbnail display, click-to-full-size preview (lightbox), structured front/back content and export. **Still open:** links to other cards and game objects (beyond the eWrite object-link types Kernel 79 already covers — Cues, Storyboard cards, Scene elements, dialogue Topics).

They should not become unrestricted miniature word processors.

## V4. Cues, transitions, and theatrical control

### Established

- persistent Show state;
- current Scene;
- Cues;
- idempotent GO;
- guided tutorial transitions;
- shared stage runtime;
- role-specific player controls.

### Open

- broader transition types;
- Cue grouping and rehearsal tools where real use requires them;
- stage-management views;
- reliable visual and browser proof across role combinations;
- portable Scene/Cue package export;
- integration-driven triggers without hard-coding game meaning into Victory.

## V5. Elements, media, tokens, maps, and drawing

### Established

- Libraries, Elements, placements, context classes;
- index cards;
- stage movement;
- overlays and images;
- equipment content;
- shared stage runtime.

### Open

- reliable image and file pipeline;
- thumbnails and full-size previews;
- upload validation and re-encoding;
- token truth and visual state;
- fog and emphasis;
- shared vector drawing;
- terrain stamps and palettes;
- layers, erase, undo, and export;
- Scene-local or reusable map-authoring decisions.

The Cartograph-style package will prove the shared drawing and stamp layer.

## V6. Productions, Shows, admission, and persistence

### Established

- Location, Production, Show Run, Show;
- roster and Audience Program;
- Shows and Scene Placements;
- persistent Show stage state;
- open enrollment option;
- Showtime and guided entry;
- Story So Far and Aftercare handoff.

**Also established (Kernel 77):** reliable backup/restore, with a measured restore proof (RTO≈69s, RPO ≤15min DB / ≤24h assets) — this was "Open" as of v2.0 and is real now, not just built.

### Open

- stranger-facing onboarding;
- stronger data isolation verification;
- capacity and performance measurement;
- client-controlled Production or Lot boundaries;
- clearer show scheduling and status where needed;
- admin tools that do not grant infrastructure authority.

## V7. Rules integration boundary

This was the first major old milestone wholly skipped when Catharsis became the main branch. It remains important, but it should now be built through real packages rather than a speculative universal abstraction.

### Required

- package manifest and version;
- ruleset metadata;
- Character schemas;
- value/resource registries;
- game operations;
- oracle/table registries;
- Documents and source references;
- UI contributions and capability declarations;
- migration and compatibility expectations;
- licensing and attribution metadata;
- export/import contract;
- fail-closed behavior when a package is absent or incompatible.

### Principle

Victory should offer stable primitives. A package may rename or combine them for its game. Victory must not require every game to pretend it has Socio’s vocabulary.

## V8. Player identity and social continuity

### Established

- Trailer Player Workbook and public Face;
- My People relationships and private notes;
- Third Place Headshots;
- Profile and Character onboarding;
- Aftercare and waiting-for-humans handoff.

### Open

- Profile redesign and status levels;
- favorite-TTRPG selectors;
- privacy and discovery controls;
- invitation and blocking refinement;
- safe cross-Production discovery;
- notifications only when deliberately designed;
- no conversion into a social network engagement machine.

## V9. Victory Documents

Victory Documents is a rules-native publication and reference system, not a replacement for Google Docs.

### Product hierarchy

```text
Series
→ Ruleset
→ Module
→ Document
→ Section or anchored block
```

Current product language:

- rulesets are Series;
- scripts are Modules;
- a Series may contain long books, short rules, scripts, references, or educational material.

Other integrations may display different labels while using the same underlying organization.

Shipped under the product name **eWrite** (Kernels 78–79, all passes PASS). "Victory Documents" in this roadmap and "eWrite" in the repository are the same system — the repository name is authoritative going forward.

### Established

- long-form Documents (Publications) and short standalone articles, same model;
- Markdown paste/import with safe rendering (goldmark + bluemonday — the repository's only Markdown pipeline);
- headings with stable, alias-preserving identifiers;
- links within one Document, to another Document, and directly to anchored sections;
- links from Cues, index cards, Storyboard cards, and dialogue Topics (Character-sheet skill links also shipped — Kernel 79A);
- cover-to-cover reading and direct opening of one short rule;
- images and linked media, with publication-visibility-aware access control (a real gap closed in Kernel 79 Phase 1 — embedded images previously never checked the referencing publication's own visibility);
- drafts and publication, with append-forward revisions and 409 save-conflict protection;
- Ruleset-scoped search and navigation, collection landing pages, breadcrumbs;
- a reusable generic "directory" abstraction (proven via a 100-entry Skill Directory, 98 auto-linked to real manuscript sections);
- hierarchical (Module/Series/Ruleset) zip export with per-file checksums;
- proven at real scale against the actual Socio v1.1 core rulebook (357KB/43,207 words/763 headings), not a toy fixture.

### Open

- links from Character equipment/values/attributes/health systems (only skills are wired — no data model exists yet for the rest);
- anonymous public reading (schema-ready — `public` visibility exists — but deliberately deferred; currently serves authenticated readers only, an operator decision not a gap).

### Collaboration boundary

Real-time multi-author editing is not required for the first version. Named editors, ordinary save conflict handling, and version history may be added if they are inexpensive and safe. Victory does not need to rebuild Google Docs collaboration before Documents become useful.

## V10. Education

Victory education is a presentation-rich learning environment inspired by the flexibility of Schoology and the theatrical presentation strengths of VTTs.

It is not a student information system.

### Explicitly excluded

- institutional gradebook infrastructure;
- attendance bureaucracy;
- parent phone logs;
- discipline records;
- district records;
- comprehensive student-management administration.

### Required educational domain

- courses;
- lessons;
- assignments;
- assessment delivery;
- submissions;
- returned feedback;
- course sequencing;
- reusable Documents and media;
- interactive Scenes and Cues;
- package/plugin-style extensibility;
- live teaching and asynchronous work.

Education should reuse Victory primitives but remain a distinct domain. A classroom Venue must not become a junk drawer for every school function.

## V11. Live venue coordination

Added Kernel 84, reconciling Kernel 83's work into this track (spec instruction: record it here, "not a game-rule track" — it does not belong to Track S/Socio even though Socio will likely be its next real consumer).

### Established

- a generic, venue-agnostic, in-memory coordination primitive (`backend/internal/venuecoordination`) for two ephemeral live-session signals: **Group Leader** and **Current Turn**;
- no persistence across sessions by design — ephemeral live-session state only, cleared on session end and on backend restart;
- neither state is a Victory role, permission grant, or turn-order system — holding either never changes read/write/structural authority (Kernel 83's own required negative-authority tests prove this directly);
- explicit handoff only — no participant order, no automatic rotation, no software-inferred "next" participant;
- Storyboards wired as the first consumer (its whole family — Blank and Timeline both), including a Presence Tray (right-click authority-gated actions, server-reverified independent of what the UI shows) that did not exist in Storyboards before Kernel 83.

### Open

- a second real consumer beyond Storyboards (the generic package is unproven against a second venue's different authority model — see the open decision below);
- any keyboard-reachable entry point for the Presence Tray's context menu (currently right-click only, a named accessibility gap — see `Construction/Storyboards/storyboards-accessibility.md`);
- whether The Cave's existing (different-shaped) session/presence concept should eventually adopt this same coordination primitive, or remain separate.

---

# 8. Track S — Socio Playable Experience

## S1. Character creation, truth, and presentation

### Established

- substantial Character onboarding;
- Face and Mechanics projection;
- Character selection and activation;
- Catharsis tutorial Character flow;
- Profile and social continuity.

### Open

- verify every Socio Character field and authority path;
- track and present all eight health systems appropriately;
- finish value/resource help;
- eliminate remaining manual workaround gaps;
- provide direct links from Character mechanics to relevant rules Documents.

## S2. Courtyard and world entry

**Status:** Substantially complete through Catharsis.

Remaining work is side repair and polish rather than a new main-track milestone.

## S3. How to play Socio

Catharsis proves onboarding and a guided journey, but Socio still needs a complete learn-to-play path.

Likely required teaching includes:

- Social Stances;
- major and minor actions;
- reactions;
- values;
- skills and resolution;
- advancement;
- equipment;
- all health systems;
- habits, Resolve, Fate, and exhaustion where applicable;
- Director judgment and consequences;
- session continuity;
- audience projection.

The system should primarily assist the Director and players rather than rigidly automate every judgment.

## S4. Stance and social display

### Required

- clear stance wheel and current stance;
- rules-aware available actions;
- reactions and consequences;
- links to concise rule articles;
- Director assistance without forcing a single interpretation;
- public/private projection appropriate to Socio play.

## S5. Equipment, bartering, and hand cards

### Established

- Equipment library;
- Kessa sale inventory;
- purchase/barter tutorial;
- cards and stage placement;
- no automatic coin deduction.

### Open

- generic inventory ownership and transfer;
- clearer equipment use;
- links to item Documents and images;
- hand-card interaction where useful;
- Director correction and audit;
- continuity beyond Kessa.

## S6. Sustained adventure

### Goal

Move beyond a one-time tutorial into repeatable play.

Required:

- multiple connected Scenes;
- persistent consequences and resources;
- health tracking;
- Director tools;
- scene-to-scene continuity;
- Story So Far;
- re-entry after a break;
- meaningful player choice rather than only guided clicks.

## S7. Director and audience

### Required

- Director resolution assistance;
- curated audience projection;
- safe Cues and overrides;
- audience reactions and participation where invited;
- no leakage of private mechanics or journals;
- reliable multi-user browser proof.

---

# 9. Track A — Anthology and Integration Proof

Anthology packages are not giant alternate-platform eras. Each should be a bounded 1–3 kernel proof that packages a playable game while adding one reusable capability Victory already needs.

## A0. Selection and legal rule

Before using another creator’s exact text:

- identify the governing license or obtain permission;
- preserve required attribution;
- do not assume public availability equals permission to reproduce;
- separate mechanics, terminology, and copyrighted expression;
- record what Victory may distribute.

When exact game text cannot be used, build an original package proving the same class of capability.

## A1. Microscope-style package

**Order:** First. **Status reconciled Kernel 84 — the generic capability shipped; the actual licensed package did not, deliberately.**

### Generic Victory capability — proven (Kernels 80–83, PASS)

- semantic Storyboards (columns/bands/rows/cards, ownership/sharing, role-capability matrix);
- a built-in Timeline mode (server-enforced Beginning/Ending boundary columns, configurable Reference Panel);
- structured JSON export preserving order, hierarchy, and placement;
- long ruleset-scale navigation (eWrite, Kernels 78–79);
- live Group Leader (turn/authority flow's leadership half — Kernel 83);
- live, explicit Current Turn (turn/authority flow's turn half — Kernel 83, handoff-only, no automatic rotation).

This is everything A1's original "Generic Victory capability" line asked for. No further platform-primitive work is believed necessary before a Microscope-style package could be built.

### Actual Microscope package — not built, intentionally

- exact licensed terminology (Periods/Events/Scenes or a legally distinct equivalent);
- permission or license to use Microscope's specific trade dress/procedures;
- complete game-specific setup and turn procedures (Legacies, Palette, bang/question mechanics, etc.);
- exact trade dress.

Kernel 82 explicitly grepped its own code/docs/UI copy for Microscope-specific terms (Lens/Focus/Period/Event/Scene) and confirmed none are present — Storyboards/Timeline is generic on purpose, not a disguised Microscope implementation. **Decision recorded:** remain generic until permission/licensing is separately established; do not build the actual licensed package speculatively.

### Estimate

Unknown until the licensing/permission open decision (§15) resolves — the generic foundation is no longer the blocker.

## A2. Cartograph-style package

**Order:** Second.

### Generic Victory capability

Shared vector drawing, coastline/path tools, stamps, terrain palettes, layers, undo, and export.

### Architecture question to resolve in its kernel

Whether drawing is:

- a specialized Element;
- a vector layer on a Scene;
- an append-forward drawing Action stream;
- or a combination with snapshots.

### Estimate

Likely 1–3 kernels.

## A3. Solo oracle package

**Order:** Third.

Ironsworn may not be legally or practically distributable as a Victory package. Victory will instead create an original solo-play oracle game unless a suitable license is confirmed.

### Generic Victory capability

- versioned oracle registries;
- contextual table lookup;
- dice resolution;
- structured results;
- prompts;
- progress or tracks;
- journal linkage;
- rules-document links;
- replayable history.

### Estimate

Likely 1–3 kernels.

## A4. Later proofs

Potential later packages:

- FitD-style clocks and shared organization workbook;
- triggered-move/playbook package;
- other small licensed or original systems;
- larger integrations only when commercially justified.

D&D, Pathfinder, and Socio-class integrations are major projects, not ordinary anthology proofs.

---

# 10. Track C — Concierge and Distribution

## C1. Hosted use first

Victory will initially remain on Grant’s hosted environment while real use, security, capacity, support burden, and willingness to pay are learned.

The first outside users may be family, friends, invited strangers, founding partners, or lessees. Public exposure must not precede the hosted-readiness audit and critical repairs.

## C2. Concierge before SaaS

Before building a generalized SaaS control plane, Victory may offer bespoke arrangements:

- access to a private Production Lot on Grant’s server;
- a customer-funded server operated by Grant;
- a private deployment the customer operates;
- installation and configuration;
- rules integration;
- branding or Venue packages;
- ongoing operation when the price justifies it.

The exact commercial model remains provisional. Engineering should preserve these options without prematurely building multi-tenant billing machinery.

## C3. Personal self-hosted edition

After stabilization, Victory should support a personal/self-hosted distribution path so Socio and Victory can spread without requiring Grant to host every table.

Likely first target:

- Windows;
- Docker Desktop;
- WSL 2;
- Linux containers;
- a simple launcher and browser opening;
- user-owned hardware, storage, backups, and networking;
- no private support obligation;
- community discussion only when useful;
- users obtain Docker separately and are responsible for its license.

A Linux private-server package should remain possible from the same Docker deployment truth.

## C4. Backup, restore, export, and handoff

### Established (Kernel 77, PASS)

- reproducible install (`scripts/smoke/fresh-install.sh`, repaired this era after being silently broken since Kernel 76);
- database backup (pre-migrate automatic dump, plus encrypted off-host backup);
- asset backup;
- **restore proof — measured, not assumed** (RTO≈69s, RPO ≤15min DB / ≤24h assets, rehearsed in an isolated environment);
- versioned migrations (embedded, checksummed ledger, auto-applied at boot — Kernel 72's design, unchanged);
- self-service account data export (JSON+Markdown+files archive);
- account deletion (tombstone-reassignment model, not a hard delete that breaks shared history).

### Open

- live-over-live restore specifically (§8 of the restore runbook) was documented but not separately drilled — only the isolated-environment path was rehearsed;
- operator handoff documentation for a *second* human beyond Grant (current docs assume the operator is also the developer);
- export scope for Storyboards/Timeline and eWrite content specifically has not been added to the same self-service export path Kernel 77 built for account data (Storyboards has its own per-board JSON export; eWrite has its own per-publication/hierarchical export; neither is bundled into the *account*-level export yet).

## C5. Branding and private configuration

Potential paid capabilities:

- customer identity and Production Lot branding;
- selected Venue packages;
- rules integration packages;
- client-specific configuration;
- domain and email configuration;
- no source-code or host authority implied by Producer role.

## C6. Capacity and business evidence

Measure:

- concurrent users;
- active Shows and Sessions;
- database growth;
- asset growth;
- bandwidth;
- CPU and memory;
- backup size and duration;
- restore duration;
- operator hours per customer;
- support load;
- integration labor;
- whether isolation requires separate instances.

These measurements decide whether Victory becomes:

- one shared hosted service;
- several managed servers;
- customer-operated private deployments;
- SaaS;
- or a smaller number of high-value managed clients.

---

# 11. Track O — Operational Integrity

## O1. Evidence-first delivery

PASS requires evidence appropriate to the feature:

- tests;
- commands and outputs;
- database proof;
- API proof;
- browser-visible behavior;
- multi-user proof;
- screenshots where presentation matters;
- negative authority tests;
- clean-install proof when deployment changes.

“No evidence” means “not proven.”

## O2. Canonical documentation

Maintain:

- this roadmap;
- kernel specs;
- reportbacks;
- operator log;
- operator notes;
- dictionary;
- field guide;
- development workflow;
- security notes;
- vendor acknowledgements.

Do not create another roadmap.

## O3. Migrations and data safety

- additive migrations;
- isolated test database;
- no destructive tests against live data;
- bootstrap and migration compatibility;
- explicit backfills;
- rollback or recovery plan for risky changes;
- fresh-install smoke coverage;
- backup before consequential production migrations.

## O4. Authority and security

### Established

- server-authoritative identity and role; no client-selected privilege (every Kernel 76–84 authority check re-derives the caller's tier server-side — a standing convention, not a one-time fix);
- route and WebSocket authorization, including auth-before-handshake (Kernel 77 K77-08) and periodic session-revocation sweeps for already-open sockets (Kernel 77);
- Location/Production/Show/Character isolation;
- rate limiting (credential endpoints since Kernel 72/76; WS connection/message limiters);
- safe cookies and CSRF posture (`SameSite=Lax`, regression-tested);
- secret management (rotated DB password into `.env`, Kernel 76);
- upload validation (size/type caps);
- safe Markdown and HTML rendering (goldmark + bluemonday, the repository's one sanitize path, proven at real manuscript scale);
- privacy-safe logs (the password-reset token-in-logs leak Kernel 76 closed was the concrete negative example);
- deletion/export policy (Kernel 77, see C4);
- database and storage isolation (dedicated test-database safety gate, Kernel 64, still enforced).

### Open

- dependency and license review (last done as part of the Kernel 76 audit inventory; no recurring process);
- self-service recovery email delivery (code-complete, blocked on Brevo account approval — see §5.7);
- a second, non-Storyboards consumer to prove `venuecoordination`'s authority model genuinely generalizes (see V11).

## O5. Browser and multi-user smoke

Every major player-facing or theatrical feature should eventually prove:

- correct operator/Producer/Director/player/audience behavior;
- reconnect;
- second-user projection;
- forbidden access;
- stale state;
- ordinary browser use rather than only direct API calls.

## O6. Performance and storage

Do not optimize abstract scale prematurely. Do measure before selling reliability.

Required later:

- representative load tests;
- large Document tests;
- asset and Scene storage tests;
- Action history growth;
- WebSocket fanout;
- backup/restore timing;
- database indexes;
- cache behavior;
- observability sufficient to diagnose client problems.

---

# 12. Immediate committed horizon

**Reconciled Kernel 84 (2026-08-08).** The three kernels this section originally committed (a hosted-readiness audit, hosted-user stabilization/security repair, and a Victory Documents foundation) are all **complete** — shipped as Kernels 76, 77/77A, and 78/79 respectively, under different titles and with real scope divergence from what this section originally specified. See §5.7 for the capability-band summary and `Construction/OperatorLogs/kernel-history-reconciliation-through-83.md` for the full per-kernel evidence ledger. Their original detailed specs are preserved verbatim in `Construction/Kernels/` (e.g. `Kernel 77 — Proposed Hosted-User Stabilization Scope.md`, marked superseded in favor of the canonical `Kernel 77 — Private-Client Readiness.md`) — not duplicated here, to avoid this file itself becoming the next thing that goes stale.

The following two kernels are now committed in purpose. Numbers must be verified against the repository before issue — per §4.3, always check the operator log for the true next number.

## Kernel 85 — Socio Sustained Play

**Primary tracks:** S  
**Secondary tracks:** V (reuses Documents/eWrite, Storyboards if useful, and the live coordination primitive)

### Goal

Move Socio beyond tutorial/onboarding (Catharsis: The Golden Journey, Kernels 71–75) into repeatable, connected, sustained play — the "P4. Socio learn-to-play continuation" item from the old provisional horizon, now promoted to committed.

### Direction, not full spec

Per Kernel 84 spec §10.8, this kernel is intentionally not fully specified here. Likely territory, subject to its own kernel-maker audit against current Catharsis/character/rules-integration state: stances, actions and reactions, values, skill resolution, health systems, sustained-play loop, Director assistance tools. Should reuse eWrite (rules reference/linking) and the live coordination primitive (V11) where a real fit exists rather than inventing parallel mechanisms.

## Kernel 86 — Cartograph-Style Drawing Foundation

**Primary tracks:** V5, A2  
**Secondary tracks:** V3

### Goal

Build the next meaningfully different reusable creative/game primitive — shared map/drawing authoring — suitable for an original or permission-safe Cartograph-style proof. The "P6. Shared drawing and Cartograph-style package" item from the old provisional horizon, now promoted to committed.

### Direction, not full spec

Per Kernel 84 spec §10.8, this kernel is intentionally not fully specified here. Likely territory: shared vector drawing, coastline/path tools, stamps, terrain palettes, layers, undo, export (matching A2's existing "Generic Victory capability" description). Open decision #7 (exact architecture of vector drawing storage) should be resolved as part of this kernel's own early investigation, not assumed here.

### Third committed slot

Per Kernel 84 spec §1.5/§10.8: no third slot is committed. Repository evidence does not make a specific third kernel's dependency clear yet — Kernel 84's own audit found real, bounded cleanup items (see the cleanup ledger) but none large enough to justify displacing Socio/Cartograph as the next two, and none of Track C/A's remaining items (concierge delivery, A3 solo oracle, personal distribution) have a clear forcing dependency yet either. This slot is explicitly decision-gated/provisional rather than filled to satisfy the roadmap's own three-kernel format — see §4.1's own three-kernel horizon rule and §15's open decisions for what would need to resolve first.

---

# 13. Provisional horizon after Kernel 84

This order is provisional. **Reconciled Kernel 84:** P4 and P6 are promoted to the committed horizon (§12, as Kernels 85 and 86); P1, P2, and P3 are updated below to reflect real shipped status rather than left as stale future-tense requirements.

## P1. Document linking and content integration — largely established

Connecting Characters/equipment/index cards/Cues/Scenes to Documents shipped as eWrite's object-link system (Kernel 79 Goal C&E: Cues, index cards, Storyboard cards, dialogue Topics; Kernel 79A: Character skill links). Thumbnail/full-image support, navigation, and search all shipped (V9). **Still open:** links from Character equipment/values/attributes/health systems specifically (no data model yet); concise in-context help surfaces; source attribution/package metadata for imported content.

## P2. Semantic banded board foundation — established

Shipped as Storyboards (Kernels 80–82): horizontal finite sequences, stacked bands, titles, locks, custom column labels, structured export. **Deliberately not built** (see skip/defer ledger): nested/beneath cards, merged regions — Grant's own review confirmed a flat model matches actual use, this is not a gap to fill by default.

## P3. Microscope-style package — generic capability established, actual package still gated

See A1. The semantic board (P2) is real and sufficient; the actual licensed package remains gated on the unresolved legal/permission open decision (§15#6), not on further platform work.

## P5. Scene composition and capture

Build the general visual Scene-authoring surface if the board or Socio work has not already forced it earlier.

## P7. Oracle registry and original solo package

Add contextual oracles, structured resolution, progress, prompts, and journal links; deliver an original solo game unless licensed integration becomes available.

## P8. Personal Windows distribution

- clean Windows test machine;
- Docker Desktop/WSL 2 flow;
- installer or PowerShell launcher;
- migration;
- restart;
- update;
- backup;
- restore;
- local multi-user proof;
- clear no-support boundary.

## P9. Education foundation

- Courses;
- Lessons;
- Assignments;
- Submissions;
- Assessments;
- returned feedback;
- Documents, Scenes, Cues, and packages as teaching material.

The first educational proof should be chosen when Grant has a real course or teaching need. It need not be decided now.

## P10. Concierge delivery and capacity evidence

- founding client or invited lessee;
- bounded onboarding;
- private Production Lot or customer-funded deployment;
- measured support and infrastructure cost;
- repeatable proposal, install, backup, and handoff;
- SaaS decision only after evidence.

---

# 14. Skip and defer ledger

This ledger must be updated whenever a planned item is not built.

| Item | Current disposition | Reason / replacement |
|---|---|---|
| Old six-kernel committed horizon | **Abandoned** | Too much plan becomes stale before evidence arrives; replaced by three committed kernels |
| Separate track roadmap and master guide | **Replaced** | Grant cannot track multiple roadmap files; replaced by this canonical roadmap |
| Old numbered Kernel 61–70 plans | **Superseded** | Real repository used those numbers for different work; retain only as historical ideas |
| V7 speculative universal integration layer before real packages | **Skipped in original order; retained as incremental track** | Build integration seams through Documents and anthology proofs rather than abstraction first |
| Catharsis as continuing main branch | **Ended** | Tutorial journey shipped; remaining work becomes side repair |
| Google-Docs-grade collaborative editor | **Deferred / probably unnecessary** | Victory needs linkable Markdown publication first |
| Automated story beats or narrative judgment | **Not planned** | Manual labels, cards, bands, and Scenes are sufficient |
| School information system features | **Explicitly excluded** | Education needs courses, assignments, assessments, and feedback, not administrative records |
| Automatic coin deduction for Kessa | **Not planned** | Equipment interaction remains Director/player-assisted unless a later explicit design changes it |
| Ironsworn package | **Deferred / replaced provisionally** | Licensing and fit uncertain; original solo oracle package will prove the capability |
| D&D/Pathfinder as anthology proofs | **Rejected** | Multi-source, major integrations; only commercially justified projects |
| SaaS before customer evidence | **Deferred** | Concierge/hosted use must reveal capacity, support, and payment behavior first |
| Separate Victory instance per five players | **Not assumed** | Measure real capacity before deciding tenancy/deployment model |
| Unlimited support inside annual license | **Rejected** | Community discussion may exist; private labor remains separately scoped |
| Tauri desktop package | **Later only if useful** | Docker-based personal distribution comes first |
| Nested/beneath cards and merged cells/regions in Storyboards | **Replaced** | Kernel 80 speced it; Grant's own post-deploy review confirmed a flat one-card-per-cell model matches actual use. Not carried into Kernel 81's rework — see V3 |
| Actual licensed Microscope terminology/mechanics in Storyboards/Timeline | **Deferred, intentionally** | Generic capability shipped (Kernels 80–83); the specific package remains gated on the unresolved licensing/permission open decision (§15) — see A1 |
| Tone/light-dark hardcoding in Storyboards | **Not planned** | Never speced for Storyboards; no product need identified |
| Participant turn order / automatic turn advancement in the coordination primitive | **Rejected by design, not merely unbuilt** | Kernel 83's spec explicitly required *no* participant order, *no* automatic rotation, *no* software-inferred "next" — explicit handoff only. This is a permanent product decision, not a future-work gap. See V11 |
| Persisted Group Leader/Current Turn across sessions | **Rejected by design** | Ephemeral live-session state is the whole point (spec §1.5) — persisting it would be a different, unrequested feature |
| User-created Storyboard templates (beyond the two built-in Blank/Timeline) | **Not planned** | Never requested; the two built-in templates cover current need |
| Old six-kernel committed horizon (v2.0's own §12: audit/stabilization/Documents) | **Completed and replaced** | All three shipped (as Kernels 76, 77/77A, 78) under different titles than originally planned — see §5.7. Replaced by the Kernel 85/86 horizon in §12 |
| Storyboards WS pump reusing a handshake-scoped context for a connection's whole life | **Repaired (Kernel 84)** | Was a real bug (late `watch_board`/board-switch operations silently failed past ~5s), not a documentation gap — see `Construction/Operations/websocket-context-lifecycle.md` |

---

# 15. Open decisions

These decisions are intentionally unresolved and should not block current kernels.

1. The next major Victory system after Documents, boards, and education may emerge through real use.
2. Whether all integrations display Series → Ruleset → Module labels or relabel the same underlying hierarchy.
3. Whether the first paid outsider receives a private Production, Production Lot, Location, or separate deployment.
4. The exact support, hosting, and commercial model.
5. The first real educational course.
6. Exact legal treatment and permission for a Microscope package (directly gates A1's actual-package build — the generic capability no longer blocks it, see A1).
7. Exact architecture of vector drawing storage (directly relevant to Kernel 86 — Cartograph-Style Drawing Foundation).
8. Capacity threshold for shared hosting versus isolated deployments.
9. Whether a future desktop wrapper adds enough value beyond Docker-based personal distribution.
10. Whether `venuecoordination` (V11, Kernel 83) should gain a second consumer venue (e.g. The Cave's existing session/presence model) now, or remain single-consumer until a concrete second need appears — the generic package is unproven against a differently-shaped authority model.

~~Whether Documents support named editors in the first kernel or immediately afterward~~ — resolved (Kernel 78): named editors shipped in the first version. Removed 2026-08-08.

Open decisions belong here until evidence or operator choice resolves them.

---

# 16. Roadmap update rules

Update this roadmap when any of the following occurs:

- a committed kernel passes, partially passes, or fails;
- the next three committed kernels change;
- a major architecture decision changes;
- a new system becomes part of the dream-VTT scope;
- a planned requirement is skipped, replaced, or abandoned;
- a security audit changes readiness;
- an anthology package changes order or legal status;
- concierge or SaaS strategy changes;
- repository evidence contradicts this baseline.

For each update:

1. update the implementation baseline;
2. update the committed horizon;
3. update the relevant track;
4. update the skip/defer ledger;
5. add a short change-log entry.

Do not preserve stale kernel numbers as though they still control implementation.

---

# 17. Change log

## 2026-07-30 — Version 2.0 adopted

- merged the former parallel-track roadmap and master implementation guide into one canonical file;
- reconciled the reported implementation through Kernel 75;
- ended Catharsis: The Golden Journey as the main branch while preserving side repairs;
- established a three-kernel committed horizon;
- committed hosted-readiness audit, stabilization/security repair, and Victory Documents foundation as the next sequence;
- defined Victory Documents, semantic banded storyboards, and Education;
- reframed anthology games as bounded 1–3 kernel packages;
- ordered Microscope-style, Cartograph-style, and original solo-oracle packages;
- preserved concierge-before-SaaS direction;
- added personal Windows/Docker distribution as a later free or low-cost lane;
- added an explicit skip/defer ledger;
- reaffirmed that repository evidence outranks roadmap text.

---

## 18. Immediate next action

After repository numbering verification, draft the full specification for:

> **Kernel 76 — Canonical State, Security, and Hosted-Readiness Audit**

The kernel specification must use this roadmap as planning authority and current repository evidence as implementation authority.
