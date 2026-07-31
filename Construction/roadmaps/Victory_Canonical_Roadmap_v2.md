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

This baseline reconciles the reported repository state through Kernel 75. The next audit must verify it against the live repository.

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
- Catharsis Scene composition through the shared stage runtime.

### Required next systems

#### Visual Scene composition and capture

- compose a Scene from reusable Elements and placements;
- save stable transforms, visibility, labels, locks, and context;
- capture or revise a reusable Scene without duplicating runtime truth;
- preview role-specific projection.

#### Semantic banded card boards

A storyboard or timeline may contain:

- a finite horizontal sequence;
- multiple bands stacked vertically;
- titled rows or bands;
- one Scene or custom unit per band;
- cards beneath or nested inside other cards;
- lockable bands;
- merged cells or regions;
- custom labels such as Beat;
- manual rather than algorithmic story structure;
- structured export preserving order, hierarchy, front/back content, and placement.

This is the generic capability later proven by the Microscope-style package.

#### Richer index cards

Index cards remain bounded Elements. They should gain:

- links to Documents and anchored sections;
- links to other cards and game objects;
- image references or attachments;
- thumbnail display;
- click-to-full-size preview;
- structured front/back export.

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

### Open

- stranger-facing onboarding;
- stronger data isolation verification;
- capacity and performance measurement;
- reliable backup/restore;
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

### Required capabilities

- long-form Documents;
- short standalone articles;
- Markdown paste/import;
- safe Markdown rendering;
- headings with stable identifiers;
- links within one Document;
- links to another Document;
- links directly to anchored sections;
- links from Character sheets, cards, equipment, Cues, lessons, and other objects;
- cover-to-cover reading;
- direct opening of one short rule;
- images and linked media;
- drafts and publication;
- search and navigation;
- export without trapping content in Victory.

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

**Order:** First.

### Generic Victory capability

Semantic banded card boards and structured export.

### Package scope

- timeline/bands;
- cards and nesting;
- periods/events/scenes or legally permitted equivalent terminology;
- turn and authority flow;
- links, images, and card detail;
- export preserving semantic order and layout;
- complete playable package.

### Estimate

Likely 1–3 kernels after the underlying board foundation is ready.

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

Required before serious client dependence:

- reproducible install;
- database backup;
- asset backup;
- restore proof;
- versioned migrations;
- export of Documents, Characters, Scenes, and relevant package data;
- operator handoff documentation;
- recovery from a clean environment.

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

- server-authoritative identity and role;
- no client-selected privilege;
- route and WebSocket authorization;
- Location/Production/Show/Character isolation;
- rate limiting;
- audit events;
- safe cookies and CSRF posture;
- secret management;
- upload validation;
- safe Markdown and HTML rendering;
- privacy-safe logs;
- deletion/export policy;
- dependency review;
- database and storage isolation.

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

The following three kernels are committed in purpose. Numbers must be verified against the repository before issue.

## Kernel 76 — Canonical State, Security, and Hosted-Readiness Audit

**Primary tracks:** O, V  
**Secondary tracks:** C, S  
**Type:** Audit with bounded critical repair

### Goal

Determine, from current repository and deployment evidence, whether Victory can responsibly accept strangers and their data.

### Required audit areas

- real repository/kernel inventory;
- route inventory;
- anonymous and public exposure;
- authentication and session cookies;
- password reset and invite handling;
- authorization across Locations, Productions, Shows, Sessions, Scenes, Characters, profiles, relationships, messages, and assets;
- WebSocket authentication and subscription isolation;
- trusted game-event authorship;
- CSRF posture;
- brute-force and rate limiting;
- secrets and environment variables;
- database network exposure;
- logs containing tokens, credentials, private data, or sensitive payloads;
- uploads, file types, image decoding, storage paths, and size limits;
- Markdown/HTML sanitization requirements before Documents;
- backup confidentiality and restore viability;
- account deletion and data export expectations;
- dependency and license inventory;
- denial-of-service limits;
- stale security documentation;
- stale or misleading operator documentation.

### Allowed implementation

The kernel may repair a clearly bounded critical defect when:

- the cause is understood;
- the repair is low-risk;
- evidence can be completed;
- it does not consume the audit.

Larger repairs go into the Kernel 77 ledger.

### Required deliverables

- current architecture and exposure map;
- severity-ranked finding ledger;
- explicit “safe for what kind of user” classification;
- bounded repairs and evidence;
- updated security notes;
- explicit skipped/deferred findings;
- recommended Kernel 77 scope;
- confirmation or correction of future kernel numbering.

### Non-goals

- compliance certification;
- generalized SaaS;
- Victory Documents;
- unrelated redesign;
- endless theoretical hardening.

## Kernel 77 — Hosted-User Stabilization and Critical Security Repair

**Primary tracks:** O, V, C  
**Scope source:** Kernel 76 findings

### Goal

Repair the most important blockers to stranger-facing hosted use and establish a sustainable maintenance floor.

### Likely work, subject to audit

- rate limiting;
- audit logging;
- route and WebSocket isolation repairs;
- trusted-event repair;
- secret cleanup;
- cookie/CSRF corrections;
- upload hardening;
- privacy-safe logging;
- backup and restore proof;
- account deletion/export minimum;
- operator health visibility;
- browser and multi-user regression suite;
- stale documentation repair.

### Scope rule

Kernel 77 must be bounded. Findings that are important but too large become named follow-up work, not silent omissions.

### Required outcome

A documented readiness classification such as:

- suitable for Grant only;
- suitable for family/friends;
- suitable for invited strangers with limited data;
- suitable for paid hosted Productions;
- not yet suitable for broader public signup.

The classification must be evidence-based.

## Kernel 78 — Victory Documents Foundation

**Primary tracks:** V9  
**Secondary tracks:** S, A, O

### Goal

Create the safe, linkable document foundation needed for rules, scripts, short articles, education, cards, and integrations.

### Build

- Series, Ruleset, Module, Document, and anchored-section model;
- draft/published state;
- owner and named-editor authority as appropriate;
- Markdown paste/import;
- safe parsing and rendering;
- stable heading/section identifiers;
- internal links;
- cross-Document links;
- direct anchored links;
- long-form reading view;
- short-article view;
- image/media references;
- search/navigation minimum;
- export to Markdown or another open representation;
- links from at least one existing Victory object;
- Socio rules content as the first real consumer.

### Do not pull in

- Google-Docs-grade simultaneous editing;
- full educational Courses;
- semantic storyboard boards;
- arbitrary executable HTML;
- universal package marketplace;
- large-scale content entry by the kernel maker.

### Required proof

- paste a nontrivial Markdown rules document;
- publish it;
- follow an internal heading link;
- follow a link to a short separate rule;
- link from an existing Victory object into the correct anchored rule;
- verify unauthorized draft access is denied;
- export the content without losing its basic structure;
- prove unsafe HTML/script content does not execute.

---

# 13. Provisional horizon after Kernel 78

This order is provisional.

## P1. Document linking and content integration

- connect Characters, equipment, index cards, Cues, and Scenes to Documents;
- thumbnail and full-image support;
- better navigation and rules lookup;
- concise help surfaces;
- source attribution and package metadata.

## P2. Semantic banded board foundation

- horizontal finite timelines;
- stacked bands;
- titles;
- nested cards;
- merged regions;
- locks;
- custom labels;
- structured export.

## P3. Microscope-style package

Use the semantic board to deliver the first complete anthology proof, subject to licensing or permission.

## P4. Socio learn-to-play continuation

Use Documents, links, and existing Catharsis systems to teach the broader Socio rules:

- stances;
- actions and reactions;
- values;
- skill resolution;
- health systems;
- sustained play;
- Director assistance.

## P5. Scene composition and capture

Build the general visual Scene-authoring surface if the board or Socio work has not already forced it earlier.

## P6. Shared drawing and Cartograph-style package

Add vector drawing, stamps, layers, undo, and export; package a complete map-making game.

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

---

# 15. Open decisions

These decisions are intentionally unresolved and should not block current kernels.

1. The next major Victory system after Documents, boards, and education may emerge through real use.
2. Whether Documents support named editors in the first kernel or immediately afterward.
3. Whether all integrations display Series → Ruleset → Module labels or relabel the same underlying hierarchy.
4. Whether the first paid outsider receives a private Production, Production Lot, Location, or separate deployment.
5. The exact support, hosting, and commercial model.
6. The first real educational course.
7. Exact legal treatment and permission for a Microscope package.
8. Exact architecture of vector drawing storage.
9. Capacity threshold for shared hosting versus isolated deployments.
10. Whether a future desktop wrapper adds enough value beyond Docker-based personal distribution.

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
