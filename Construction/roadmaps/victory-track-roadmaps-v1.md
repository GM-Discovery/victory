# Victory VTT — Parallel Track Roadmaps

**Version:** 1.0  
**Status:** Working control document  
**Purpose:** Keep Victory, Socio, anthology modules, concierge delivery, and operational integrity moving together without pretending they are one undifferentiated backlog.

---

## 1. Governing doctrine

Victory is a **rules-native theatrical VTT platform**. Socio is its flagship rules integration and living reference implementation.

The platform split is:

```text
Victory owns the container, authority, persistence, projection, runtime, and presentation.
A rules integration owns the game-specific meaning.
Socio is the first complete proof of that contract.
```

Every generic Victory capability should be exercised by a real Socio behavior as soon as practical. Every Socio feature should reuse a Victory primitive when the concept is genuinely reusable.

The project should not alternate between two bad extremes:

- abstract VTT infrastructure with no playable reason to exist;
- Socio-only features that quietly create a second scene system, second action log, second permission model, or second character truth system.

---

## 2. Track summary

| Track | Purpose | Immediate success condition |
|---|---|---|
| **V — Victory Core** | Reusable runtime, character, scene, production, authority, and presentation primitives | Socio can play through the same generic systems another rules integration could consume |
| **S — Socio Playable Experience** | Advance from character creation into sustained guided play | A player can move beyond the Courtyard into a meaningful repeatable play loop |
| **A — Anthology and Integration Proof** | Prove Victory is not accidentally hard-coded to Socio | At least two substantially different games run through reusable Victory primitives |
| **C — Concierge and Distribution** | Make self-hosting, private forks, installation, branding, and handoff practical | An operator or client can receive, run, back up, and own an instance without Victory paying hosting costs |
| **O — Operational Integrity** | Preserve evidence, migration safety, security, tests, and operator memory | A new builder can reproduce behavior from documentation and a clean install |

Operational Integrity is cross-cutting. It is never an optional fifth project that happens after feature work.

---

# Track V — Victory Core

## V1. Character truth and projection

### Goal
One canonical character truth system supports Greenroom workbooks, venue right trays, commands, public game events, and future rules integrations.

### Required capabilities

- typed current facts projected from append-forward workbook events;
- singleton, collection, derived, ledger, temporary-state, and custom-field contracts;
- owner manual visibility and integer priority;
- Director value, visibility, and priority overrides with independent locks;
- Token Aura as a real token visual fact;
- one shared projection consumed by Greenroom and venue right trays;
- active-character targeting resolved server-side;
- private Journal isolated from public Face, Mechanics, History, and Game Events;
- ruleset-defined value and skill authority.

### Completion proof

- A Greenroom change appears in the right tray without creating a second data source.
- A Director-locked fact cannot be silently replaced by owner or system refresh.
- A low-authority generated value cannot replace a better explicit fact.
- A two-user browser test proves public mechanical events while Journal content remains private.

---

## V2. Command and event surface

### Goal
Buttons, forms, and slash commands call the same domain operations.

### Required capabilities

- canonical command registry;
- active-character resolver;
- `/char`, `/bio`, `/quote`, `/journal`, `/value`, `/override`, `/help`, and existing venue/session commands;
- `/` contextual palette;
- previews for material or mechanical changes;
- idempotent mutation receipts;
- typed public Game Events;
- server-only production of trusted system events;
- role-, venue-, and ruleset-aware command availability.

### Completion proof

The same value change made through UI and slash command produces the same canonical event, projection, History entry, authorization result, and public Game Event.

---

## V3. Scene configuration and activation

### Goal
A Scene becomes a reusable, versioned, production-owned configuration that can be atomically projected into a live Session.

### Required capabilities

- stable Scene identity and revision;
- map/background asset reference;
- ordered placements;
- surface, grid, camera, visibility, lock, label, and transform settings;
- capture current supported stage state;
- preview and atomically activate a revision;
- reconnect and late-join consistency;
- one canonical scene-activation event;
- no chat, presence, selections, or transient browser state captured.

### Completion proof

Arrange a stage, save it, alter the live arrangement, activate the saved Scene, and verify every connected and later-joining client receives one coherent restored projection.

---

## V4. Scene cues, sets, and transitions

### Goal
Directors can perform scenes rather than merely load maps.

### Required capabilities

- cut, fade, and curtain presentation;
- reveal/hide cue groups;
- scene sequences and acts;
- next, previous, and jump;
- cue labels and Director notes;
- server-authoritative execution state;
- bounded transition duration.

### Completion proof

A Director can run a short Socio sequence with at least two Scenes and one coordinated cue while a second client observes the same state.

---

## V5. Element, placement, and token authority

### Goal
Stage editing and map play are persistent, reconnect-safe, and reusable across rulesets.

### Required capabilities

- position, size, rotation, z-order, visibility, lock, and label authority;
- multiselect and grouping;
- duplication and alignment;
- backstage inventories and placement templates;
- character-to-token association;
- square, hex, and gridless surfaces;
- snapping, measurement, pan, zoom, and optional camera following;
- Token Aura and role-dependent projections.

### Completion proof

A Director prepares a Scene, moves and groups placements, a player moves an authorized token, and refresh/late join reproduce the same state.

---

## V6. Production, showing, and admission spine

### Goal
Creative ownership, scheduling, performance occurrences, live sessions, and audience access are distinct but connected.

### Required capabilities

- Production creation, ownership, staffing, and default ruleset;
- Scene ownership by Production;
- production runs;
- Showing lifecycle;
- Showing configuration and duplication;
- Session start/end tied to Showing without collapsing the concepts;
- cast, crew, and audience invitations;
- will-call and admission records;
- venue/map visibility derived from durable grants.

### Completion proof

A Production schedules a Showing, assigns a Scene sequence and cast, admits an audience member, starts a Session, and archives the Showing without copying live attendance or chat into a duplicate.

---

## V7. Rules-integration boundary

### Goal
Victory can host a second system without copying Socio packages or hiding game logic in venue scripts.

### Required capabilities

- versioned ruleset identity and manifest;
- attributes, values, skills, resources, dice expressions, advancement, sheet sections, commands, and creation workflow registration;
- rules-specific data packages isolated from generic character/workbook/runtime packages;
- stable internal extension points proven by at least two integrations;
- no premature public plugin SDK promise.

### Completion proof

A second game supplies different character objects and play structures while reusing Victory identity, workbooks, actions, commands, scenes, permissions, and presentation.

---

## V8. Player identity and social profile

### Goal
Every real Victory account (not a fictional character) has one server-authoritative Player Workbook and one owner-curated Trailer Face, cleanly separated from the Character Workbook system in V1.

### Required capabilities

- versioned page/field catalogue driving both backend validation and frontend rendering, independent of any Character Workbook contract;
- typed profile events → recomputed current facts, with owner-deletable ordinary history and an append-only stage-name ledger tied to the account UUID;
- an owner Face compiler (show/hide/inferred visibility, manual/inferred priority) reusing the Kernel 59A override-dimension pattern rather than inventing a second one;
- a compiled social Face reachable by another authenticated user via a stable, copyable link, with no workbook/history/email/handle/UUID/edit-control leakage and no route-ID-based cross-account mutation;
- targeted (not global) websocket projection invalidation, so owner and viewer tabs refetch live without polling;
- a secure, real-reauthentication account-email-change path, scoped to accounts that actually have a safe reauth mechanism (password) rather than a weak stand-in for accounts that don't (provider-only);
- a closed legacy profile surface: old routes typed-deprecated, no writes reach the pre-Workbook table, no live UI reads stale data from it.

### Completion proof (Kernel 61 / 61A)

- A real existing account's legacy profile data migrates into typed facts without changing its UUID, handle, memberships, grants, or character ownership.
- The owner can complete Workbook pages, curate a Face, rename their stage name (ledger-backed, idempotent, undeletable), and change their email (password-reauthenticated) entirely through the UI.
- A second authenticated browser opens the first user's Trailer Face via a copied link, sees only the compiled Face, and cannot mutate anything on that account through any request shape.
- A Face-visibility/stage-name change made in one owner tab appears in a second owner tab and in a viewer's open Trailer tab without a reload.
- `scripts/smoke/fresh-install.sh --local` proves the whole loop (workbook → catalogue → commit → Face → delete history) on a brand-new account from an empty database.

**Status:** PARTIAL as of Kernel 61A (2026-07-09) — see `Construction/OperatorLogs/kernel-61-reportback.md` and `kernel-61A-reportback.md` for the exact per-criterion ledger. Single-account functionality (Workbook, History, Face Compiler, Stage Name, legacy closure), cross-user Face viewing, and targeted live updates are done; secure email is done for password-holding accounts only (provider-only accounts are explicitly blocked, not weakly confirmed); still open: any actual Trailer-discovery mechanism beyond a copied link, and the full operator-log/field-guide documentation pass.

---

# Track S — Socio Playable Experience

## S1. Character vertical stabilization

### Goal
The completed character maker produces a trustworthy character that remains usable after onboarding.

### Required capabilities

- Catharsis venue orientation separated from game onboarding;
- multiple character workbooks;
- Greenroom draft/resume and active-character selection;
- parentage and inheritance rules corrected;
- custom archetype and custom skill paths;
- Face curation and shared right-tray projection;
- first skill backfill and ongoing skill growth;
- commands and public game events;
- browser-level verification of the whole chain.

### Completion proof

A user creates two characters, completes one, resumes another, selects an active persona, changes a current fact, rolls and advances a skill, and sees consistent Greenroom/right-tray projections.

---

## S2. Courtyard as a real Scene

### Goal
The current static Courtyard descriptor becomes the first genuine Scene-based Socio location.

### Required content

- Courtyard map/background;
- walls;
- booths and crowd;
- Kessa;
- locked door;
- onboarding handoff;
- initial camera and visibility;
- no invented dialogue merely to satisfy infrastructure tests.

### Completion proof

Completing character onboarding activates the stored Courtyard Scene through the generic Scene system, and reconnect restores it.

---

## S3. Courtyard tutorial loop

### Goal
The Courtyard teaches Victory and Socio by letting the player do meaningful things rather than reading a manual.

### Candidate lessons

- movement and observation;
- conversation and journal use;
- skill roll and public Game Event;
- gaining or spending a value;
- interacting with Kessa;
- discovering that the door is locked;
- booth or crowd interaction;
- bartering and equipment introduction.

### Completion proof

A new player completes a short guided sequence that exercises character, scene, command, value, event, and inventory primitives without leaving the fiction.

---

## S4. Social stance and encounter display

### Goal
Socio’s distinctive social system becomes visible and playable in Victory.

### Required capabilities

- stance selection and current stance display;
- shared tempo and reaction availability;
- stance-aware actions;
- environmental/social modifiers;
- player-facing rules references;
- Director-facing resolution support;
- public event presentation that remains readable.

### Completion proof

A small social encounter can be run with two players and a Director using native stance and reaction surfaces rather than freeform notes.

---

## S5. Equipment, bartering, and hand/index cards

### Goal
The player leaves the Courtyard with tangible choices and reusable play objects.

### Required capabilities

- item/equipment records;
- bartering tutorial;
- ownership and transfer;
- hand/private tray where appropriate;
- rules hyperlinks;
- index/helper cards;
- no separate Socio-only inventory if Victory inventory primitives suffice.

### Completion proof

The player negotiates for or acquires an item, sees it on the character, and uses or references it later in play.

---

## S6. First sustained adventure arc

### Goal
Socio supports a repeatable play session beyond tutorial onboarding.

### Required capabilities

- multiple stored Scenes;
- transitions and cues;
- meaningful objectives;
- skill use and advancement;
- social and physical challenges;
- character Journal and History continuity;
- Director tools;
- session close and resume.

### Completion proof

A group can play a complete short Socio session, close it, return later, and continue from canonical state.

---

## S7. Director and audience modes

### Goal
Socio takes advantage of Victory’s theater identity.

### Required capabilities

- Director preparation and overrides;
- audience-safe projection;
- hidden information;
- audience watching and participation controls;
- showing/cue integration;
- rehearsal and performance modes.

### Completion proof

A Director can rehearse and then perform the same Socio sequence with an audience projection that does not expose private character or preparation data.

---

# Track A — Anthology and Integration Proof

## A0. Selection rule

A module belongs in the anthology only when at least one is true:

- the project genuinely wants to play it;
- it proves an architectural capability Victory needs;
- a client pays for it.

Prefer two. Never build a shallow module merely to increase a module count.

---

## A1. Microscope-style history production

### Why it matters
It challenges the assumption that the central object is always one persistent adventuring character on one tactical map.

### Victory capabilities exercised

- collaborative timeline/board;
- nested periods, events, and scenes;
- shared authorship;
- turn authority;
- nonchronological navigation;
- card-like persistent objects;
- roleplayed scenes inside a broader authored history.

### Integration restraint
Do not publicly distribute proprietary rules text without appropriate permission. A private prototype can prove architecture before licensing is settled.

---

## A2. Cartograph-style authored map game

### Why it matters
It makes map creation itself the game.

### Victory capabilities exercised

- collaborative or solo map authoring;
- drawing/placement tools;
- journal prompts;
- resources;
- map revision history;
- campaign artifact export;
- board state rather than tactical token movement as the primary play surface.

---

## A3. Ironsworn-style procedural quest module

### Why it matters
It tests solo, cooperative, and guided play with moves, progress tracks, vows, assets, and oracles.

### Victory capabilities exercised

- move registry;
- progress tracks;
- structured prompts;
- oracle tables;
- solo Journal workflow;
- public or private resolution modes;
- asset cards;
- campaign state without a traditional GM.

---

## A4. Forged-in-the-Dark-style operational module

### Why it matters
It tests clocks, crews, factions, stress, harm, and a shared organizational character.

### Victory capabilities exercised

- shared workbook ownership;
- organization/crew sheets;
- clocks;
- score/downtime cycles;
- faction state;
- position/effect presentation;
- group resources.

---

## A5. Conventional playbook-and-moves module

### Why it matters
It tests whether the rules-integration boundary can support playbooks, move triggers, partial-success choices, conditions, and advancement without assuming Socio’s attributes and skills.

### Constraint
Implement one actual licensed/open game before claiming a generic PbtA framework.

---

# Track C — Concierge and Distribution

## C1. Reproducible self-hosted installation

### Goal
A competent operator can install Victory from the repository without your intervention.

### Required capabilities

- documented prerequisites;
- Docker-based install contract;
- environment configuration;
- migrations and seeds;
- health checks;
- admin/operator bootstrap;
- persistent volumes;
- upgrade procedure;
- clean uninstall boundaries.

### Completion proof

A clean machine reaches a functioning Victory instance by following the published installation guide.

---

## C2. Backup, restore, export, and handoff

### Goal
A private-fork owner can truly possess the delivered product.

### Required capabilities

- database backup and restore;
- asset backup;
- instance configuration export;
- character/workbook export where appropriate;
- version and migration reporting;
- handoff checklist;
- recovery test.

### Completion proof

A fresh instance can be restored from an exported backup and reproduce the expected characters, productions, scenes, and assets.

---

## C3. Private-fork branding and venue package

### Goal
A client can receive an owned edition without contaminating the Victory core with one-off branding.

### Required capabilities

- instance identity;
- logo and theme configuration;
- dedicated venue package;
- ruleset package;
- licensed/private assets;
- fork and upstream relationship documented;
- generic improvements separable from client proprietary content.

---

## C4. Integration delivery method

### Goal
Turn bespoke rules integration into a repeatable professional service without pretending it is push-button automation.

### Required artifacts

- rules analysis questionnaire;
- data/content intake format;
- integration scope matrix;
- ownership/licensing schedule;
- acceptance test plan;
- installation and training plan;
- maintenance and update options;
- explicit exclusions.

---

## C5. Operator and update surface

### Goal
A client operator can understand instance health and apply supported updates without shell archaeology.

### Required capabilities

- version/build display;
- migration status;
- storage summary;
- backup status;
- service health;
- update instructions;
- log access appropriate to the operator role;
- no accidental promotion of in-app Producer authority to infrastructure authority.

---

## C6. Tauri/local package evaluation

### Goal
Determine whether a desktop wrapper materially improves installation or local/private play.

### Gate
Do not begin until the self-hosted web install, backup, and update path is stable.

### Questions to prove

- Can the Go backend be bundled or launched safely?
- What database mode is appropriate for a local installation?
- How does remote multiplayer work?
- How are updates and backups handled?
- Does Tauri reduce operator burden enough to justify another distribution target?

---

# Track O — Operational Integrity

## O1. Evidence-first completion

A kernel is not complete because code exists. Completion requires the evidence named in its acceptance criteria.

- Required browser behavior not browser-tested means **PARTIAL**.
- Required two-user behavior not exercised with two users means **PARTIAL**.
- Required clean-install behavior not exercised from a clean database means **PARTIAL**.
- Security claims require a negative test proving forbidden behavior is rejected.

---

## O2. Canonical documentation updates

Every completed or partial kernel must update the relevant project memory:

- kernel specification status;
- kernel reportback;
- operator log;
- operator notes when a durable decision or trap changed;
- field guide or dev workflow when runtime/test/install instructions changed;
- master implementation guide status;
- fresh-install migration list when schema changed.

---

## O3. Migration and data safety

- additive migrations by default;
- append-forward history;
- idempotent bootstrap where the project pattern requires it;
- clean-install validation;
- upgrade validation on an existing database for material migrations;
- no browser-only canonical state.

---

## O4. Authority and trust boundaries

- server resolves actor identity;
- server resolves active character;
- clients cannot forge trusted Game Events;
- Director authority is not Operator authority;
- access and active targeting are distinct;
- venue-derived access expires when venue authority ends;
- public projections never include private Journal data.

---

## O5. Browser and multi-user smoke

Every user-facing runtime kernel should name the browsers, accounts, venues, sessions, and expected observations required for proof.

A code review and `node --check` are not substitutes for DOM behavior when DOM behavior is the feature.

---

## O6. Performance and storage awareness

- list endpoints return summaries, not entire journals;
- paginate growing History and Journal surfaces;
- do not copy asset bytes into Scenes;
- make storage-heavy features visible to operators;
- treat character-count limits as interface/abuse guards, not complete storage policy.

---

# 3. Track matching rule

Before drafting a kernel, identify:

1. the **primary track** whose milestone is being advanced;
2. at least one **complementary track payoff**, when honest;
3. the concrete **Socio consumer** for any new generic primitive;
4. the reusable **Victory primitive** for any new Socio feature;
5. the required **Operational Integrity proof**.

A kernel may advance only one product track when necessary, but repeated single-track kernels require explicit justification in the master guide.

