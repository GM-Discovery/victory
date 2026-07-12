# Victory VTT — Master Actual Implementation Guide

**Version:** 1.0  
**Status:** Active planning and execution control document  
**Companion documents:**

- `victory-track-roadmaps-v1.md`
- `victory-kernel-spec-reportback-template-v1.md`
- existing `kernel-maker-field-guide.md`
- existing `dev-workflow.md`
- existing `operator-log.md`
- existing `operator-notes.md`

---

## 1. Purpose

This is not an aspirational feature list. It is the control document for deciding what is actually built next.

It exists to prevent three recurring failures:

1. roadmap numbering drifting away from the repository’s real state;
2. generic infrastructure consuming months without advancing Socio play;
3. a Socio feature creating one-off systems that cannot support a second rules integration or private client fork.

The master sequence is updated after every kernel reportback. A kernel maker must audit the repository and latest evidence before treating any row as current truth.

---

## 2. Product doctrine

Victory is a rules-native theatrical VTT. Socio is the flagship resident production and reference integration.

The implementation rhythm is:

```text
Build a reusable Victory primitive
→ use it immediately in Socio
→ harden what real play exposes
→ record the reusable integration seam
```

When practical, a kernel should also improve anthology readiness or concierge delivery. Those are benefits, not excuses to delay the flagship.

---

## 3. Actual baseline at adoption

The repository must be rechecked, but the current reported state is approximately:

| Work | Reported state | Required interpretation |
|---|---|---|
| Kernels 53–56 | Character onboarding vertical exists | Preserve and use as the first full rules-integration proof |
| Kernel 57 | Partial | Some onboarding/multi-character repairs remain unproven or incomplete |
| Kernel 58 | Partial in product behavior | Face semantics were specified, but Greenroom manual curation and shared right-tray projection are not visibly complete |
| Kernel 59 | Infrastructure subset shipped | Registry, resolver, basic commands, HTTP execution, and receipts exist; the broader value/override/event specification was not completed |
| Kernel 59A | PASS | Greenroom Face controls, shared venue projection, projection invalidation, Face/Mechanics tray tabs, Director locks/value overrides, and screenshot-backed two-venue browser acceptance are complete |
| Kernel 60 | Backend vertical largely proven | Browser-level sheet/click/event proof and trusted `game/event` authority still need verification; venue/session identity issue was reported |
| Kernel 61 / 61A | PASS (per `kernel-61A-reportback.md`, 2026-07-09) | Trailer Player Workbook, Face Compiler, and legacy `performer_profiles` migration (real-user identity, track V8 — unrelated to the stale "Kernel 61" planning entry below). Single-account Workbook/History/Face-Compiler/Stage-Name UI, cross-user Face viewing via a copyable link, targeted websocket live updates, password-reauthenticated email change, and legacy-route closure are done and browser-proven. Global Trailer discovery still deliberately does not exist; provider-only email step-up remains a known deferred gap. |
| Kernel 62 | PASS (per `kernel-62-reportback.md`, 2026-07-09) | Player Relationship Matrix / My People (track V8 continuation): private, directional relationship records about other real users — relationship workbook (events→facts), qualitative dropdowns, categories, private journal, stored follow-ups (no notifications), archive/unarchive, verified-shared-productions context panel, and hard subject-invisibility (all non-owner access 404s). First persistent cross-user surface, built on the Kernel 61A opaque profile-ID contract. |

No future kernel may silently call these items complete merely because a later feature depends on them.

---

## 4. Sequence management

### 4.1 Committed horizon

The next six kernels are the committed horizon. Do not reorder them without recording why.

### 4.2 Provisional horizon

Later kernels are sequenced by dependency and strategic pairing, but may be reordered when actual evidence exposes a better path.

### 4.3 Numbering

The next unused repository kernel number controls the actual filename. This guide assumes Kernel 61 follows Kernel 60. If the repository already used a number, advance the number and update this guide rather than creating collisions or retroactive aliases.

**Collision note (2026-07-11, updated 2026-07-12):** the real repository sequence has advanced past this guide's own Socio-track numbering below. Real Kernels 62 (My People), 63 (Discord fixture cleanup + Back to Map), 64 (DB test isolation and live-DB safety gate), 65 (Third Place Headshot Commons — see `Construction/OperatorLogs/kernel-65-reportback.md`), and now **66 (Show Run, Audience Program, and Roster MVP — see `Construction/OperatorLogs/kernel-66-reportback.md`)** were all built to real operator briefs, outside this document's own "Kernel 62 — Player Relationship Matrix," "Kernel 64 — Scene Configuration Model," "Kernel 65 — Capture and Atomic Fly Scene," and "Kernel 66 — Scene Transitions" headers further down, which remain *unbuilt Socio-track plans*, not what actually shipped under those numbers. The next real kernel is **67**, regardless of what this document's own section headers below are numbered — do not reuse a document-internal number as a real kernel filename without first checking `Construction/OperatorLogs/*-reportback.md` for the true next-unused number, per §4.3 above.

### 4.4 Continuation suffixes

Use `A`, `B`, or a clearly titled repair kernel only when it completes or verifies an already-shipped kernel without pretending to be a new feature band.

A continuation kernel does not rewrite the earlier report. It appends the missing proof or behavior.

---

# 5. Committed implementation horizon

> **Numbering correction (2026-07-09):** this document's planned "Kernel 61" below (Greenroom Face Curation) is stale — its own status line already says the work shipped as **Kernel 59A**, before this document's Kernel 61 slot was ever built. The repository's real, already-shipped Kernel 61 is a *different* feature entirely: **Trailer Player Workbook, Face Compiler, and Legacy Profile Migration** (real-user identity/social-profile, not the fictional Character Workbook this section describes) — see `Construction/OperatorLogs/kernel-61-reportback.md` and its closure pass `kernel-61A-reportback.md` for status (PARTIAL as of 61A — see §3 baseline table below). Track home is `V8` in `victory-track-roadmaps-v1.md`. The section immediately below is kept as a historical record of the original plan and should not be treated as describing what "Kernel 61" is in the actual repository.

## Kernel 61 (superseded planning entry) — Greenroom Face Curation and Shared Venue Character Sheet

**Primary tracks:** V1, S1  
**Complementary payoff:** C3, O2/O5  
**Source:** Kernel 59A repair specification
**Current status:** PASS as Kernel 59A. Core code is built and verified with screenshot-backed Catharsis plus First Theater browser evidence, Director fixture proof, reconnect proof, and privacy checks. `/quote add` / multi-quote collection remains a separate future feature.

### Build

- owner controls for Show on Face, Hide from Face, and Return to Inferred;
- manual integer priority and inferred/manual mode;
- independent Director value, visibility, and priority overrides/locks;
- fixed semantic Face arrangement;
- shared canonical character projection for Greenroom and venue right trays;
- compact Face and Mechanics right-tray views;
- refresh propagation after Greenroom changes;
- preserve Kernel 60 clickable skills.

### Socio use

The active Socio character becomes readable and curatable both before and during play.

### Do not pull in

- new slash-command families;
- multi-owner character access;
- Scene infrastructure;
- arbitrary drag-and-drop Face layout.

### Required proof

- owner promotes and hides facts in Greenroom;
- Director independently locks value and priority;
- right tray updates from the same projection;
- second browser sees the correct public sheet;
- private Journal content never appears.

---

> **Numbering correction (2026-07-09):** the repository's real, already-shipped **Kernel 62** is **Player Relationship Matrix, Private Notes, and Relationship Journals** (My People — track V8 continuation), spec at `Construction/Kernels/kernel-62-player-relationship-matrix-private-notes-v0.1.md`, status **PASS** per `Construction/OperatorLogs/kernel-62-reportback.md`. The planned kernel below (Venue Action Integrity / Kernel 60 browser completion) was **not** built as Kernel 62 — it remains fully open and should take the next unused kernel number when picked up. Kept below as the historical planning entry.

## Kernel 62 (superseded planning entry) — Venue Action Integrity and Kernel 60 Browser Completion

**Primary tracks:** O4/O5, V2, S1  
**Complementary payoff:** C1

### Build and verify

- prove ordinary clients cannot directly author trusted `game/event` actions;
- repair if necessary by restricting trusted Game Event production to server-owned domain paths;
- remove hard-coded venue resolution such as `the-cave` from shared session identity;
- repair the showing/session startup dependency if a valid normal startup path can deadlock;
- normalize the command-facing active-character summary shape;
- add tests for session-persona precedence and account-active fallback;
- complete real browser verification of the venue sheet, click-to-roll, advancement, and second-user Game Event visibility.

### Socio use

Skill use and growth become trustworthy in actual venue play rather than only through HTTP and database tests.

### Do not pull in

- new advancement systems;
- helper-card selection;
- production management;
- unrelated visual redesign.

### Required proof

A two-user browser run demonstrates add skill, display, click-to-roll, advance, updated die, and public Game Event. A direct client-forgery attempt is rejected.

---

## Kernel 63 — Material Facts, Values, Journal, and Director Overrides

**Primary tracks:** V2, S1  
**Complementary payoff:** A3/A4 readiness, O4

### Build

- finish `/value`, `/add`, and `/subtract` against a ruleset-defined value registry;
- finish `/override` with reason, preview, audit, and optional Director-only silent mode;
- expose material-fact mutations that call Kernel 58/61 domain operations;
- preserve `/journal` owner privacy;
- broadcast public mechanical value changes into Game Events and All;
- complete contextual `/` palette behavior from the canonical registry;
- stale-preview and active-character revalidation;
- no arbitrary JSON or database-field editing.

### Socio use

A player can receive or lose HP, Fate, or another allowed value during a venue session, with the change projected to the character, History, and public Game Events.

### Do not pull in

- cross-user Director targeting;
- full shared ownership;
- ruleset-agnostic arbitrary numeric fields;
- helper cards.

### Required proof

`/value add HP 1` targets the issuer’s active character, shows a preview, applies exactly once, updates the shared sheet, appends History, and broadcasts to another client. A protected value rejects ordinary player mutation. A Director override records its reason and lock.

---

## Kernel 64 — Scene Configuration Model v1 and Socio Courtyard Seed

**Primary tracks:** V3, S2  
**Complementary payoff:** A1/A2, C3, O3

### Build

- canonical Scene and Scene revision records;
- production or explicit temporary-production ownership;
- venue/surface compatibility;
- background/map reference;
- ordered placement records with supported transforms, visibility, locks, labels, and context metadata;
- version/status/creator/timestamps;
- read projection suitable for runtime activation;
- seed the Courtyard content as a stored Scene revision without yet deleting the legacy descriptor.

### Socio use

The Courtyard becomes real data in the generic Scene model.

### Do not pull in

- stage capture UI;
- scene activation;
- cue lists;
- transitions;
- full production manager.

### Required proof

A clean database seeds or creates a Courtyard Scene whose projection contains the expected background, walls, booths/crowd, Kessa, locked door, and stable placement ordering.

---

## Kernel 65 — Capture and Atomic Fly Scene

**Primary tracks:** V3, S2  
**Complementary payoff:** C3, O4/O5

### Build

- capture supported current stage state as a new Scene, new revision, or named duplicate;
- preview captured content;
- Director selects and previews a Scene revision;
- one idempotent server-authoritative Fly Scene operation;
- atomic resulting projection;
- reconnect and late-join consistency;
- historical activation event;
- no dozens of browser-authored placement mutations.

### Socio use

Character onboarding activates the stored Courtyard Scene through the generic activation path.

### Do not pull in

- scene folders;
- advanced transition presentation;
- cue sequences;
- fog;
- production scheduling.

### Required proof

Save an arrangement, alter the stage, fly the saved Scene, verify two connected clients and one late joiner receive the same projection, then complete Socio onboarding and land in the stored Courtyard.

---

## Kernel 66 — Scene Transitions, Cue Groups, and Courtyard Tutorial Beat

**This document-internal number collided with a real kernel.** The real Kernel 66 that
shipped (2026-07-12) is **Show Run, Audience Program, and Roster MVP** — see
`Construction/OperatorLogs/kernel-66-reportback.md` and §4.3's collision note above. This
section's plan (Scene Transitions/Cue Groups/Courtyard Tutorial Beat) remains an unbuilt
Socio-track idea; it will need a different real kernel number whenever it's actually built.

**Primary tracks:** V4, S3  
**Complementary payoff:** A1, C3

### Build

- cut, fade, and curtain presentation around canonical Scene activation;
- coordinated reveal/hide cue groups;
- append-forward cue execution;
- basic cue labels and Director notes;
- one Socio Courtyard tutorial beat using cues, such as crowd/booth emphasis or locked-door reveal;
- presentation never becomes the authority mechanism.

### Socio use

The Courtyard starts behaving like a directed playable scene rather than a static arrival card.

### Required proof

A Director executes a cue observed identically by a second client, and the tutorial beat remains correct after reconnect.

---

# 6. Provisional implementation horizon

These are sequenced recommendations, not permission to skip the committed horizon.

| Kernel | Working title | Primary tracks | Immediate Socio consumer | Complementary payoff |
|---:|---|---|---|---|
| 67 | Scene Library, Revisions, and Scene Sets | V3/V4 | Organize Courtyard and next Socio location | Client production libraries |
| 68 | Element Transform, Grouping, and Token Authority | V5 | Movement and interaction tutorial in Courtyard | Conventional VTT usability |
| 69 | Production Context v1 | V6 | Socio becomes a real Production owning its Scenes | Private client fork structure |
| 70 | Showing Lifecycle and Session Identity | V6/O4 | Rehearsal/live Socio sessions | Actual-play and client operations |
| 71 | Character Access, Shared Ownership, and Venue-Director Grants | V1/V6 | Directors access characters present in their venue | Crew/shared-character modules |
| 72 | Ruleset Manifest and Integration Boundary v1 | V7/S1 | Extract Socio attributes, values, skills, and sheet definitions from accidental hard-coding | Concierge integration method |
| 73 | Socio Stance and Social Encounter v1 | S4/V7 | First distinctively Socio encounter loop | Playbook/move UI lessons |
| 74 | Microscope-Style Timeline Prototype | A1/V7 | None required beyond shared scene/card primitives | Proves non-character-centered play |
| 75 | Socio Bartering, Equipment, and Hand Cards | S5/V5 | Courtyard tutorial continuation | Generic inventory/transfer primitives |
| 76 | Cartograph-Style Authored Map Prototype | A2/V5 | Socio maps and relationship diagrams benefit from authoring tools | Proves map-making-as-play |
| 77 | Self-Hosted Install, Backup, and Restore v1 | C1/C2/O3 | Protect Socio campaigns and character data | Makes free distribution credible |
| 78 | Socio First Sustained Adventure Arc | S6/V3/V4/V5 | Play beyond tutorial | Demonstrates flagship viability |
| 79 | Ironsworn-Style Moves, Tracks, and Oracles Prototype | A3/V7 | Socio can reuse progress and prompt primitives selectively | Strong solo/co-op integration proof |
| 80 | Private Fork Branding, Venue Package, and Handoff | C3/C4 | Victory/Socio remains upstream reference | First concierge-ready delivery shape |

---

# 7. Later horizon

After Kernel 80, choose from the following based on real usage rather than numerical inertia:

- production staffing, admission, will-call, and audience access;
- richer cue sheets and rehearsal notes;
- fog and segmented projections;
- shared organizational workbooks and clocks;
- Forged-in-the-Dark-style operational prototype;
- operator dashboard and supported updates;
- marketplace/licensing only when a real distribution need exists;
- Tauri evaluation only after self-hosted installation, backup, and update are stable;
- 3D only after rich 2D play is stable and performance-tested.

---

## 8. Kernel selection checklist

Before promoting a provisional row into an actual kernel, answer:

1. **Actual state:** What is demonstrably present in the repository now?
2. **Primary track:** Which track milestone does this advance?
3. **Socio consumer:** What concrete Socio behavior uses the new primitive now or in the immediately following kernel?
4. **Reusable primitive:** What belongs to Victory rather than Socio?
5. **Second-system value:** Does this remove a known Socio assumption or prepare a real anthology prototype?
6. **Concierge value:** Does it improve private-fork integration, installation, ownership, or handoff?
7. **Authority boundary:** Who may do what, to which active object, in which venue/session/production?
8. **Canonical truth:** Which table/event/projection is authoritative?
9. **Evidence:** What test, browser run, multi-user observation, or negative security proof establishes completion?
10. **Restraint:** What tempting adjacent work is explicitly excluded?

If these cannot be answered, the kernel is not ready to draft.

---

## 9. Status update protocol

After every kernel reportback:

1. update the corresponding row to `PASS`, `PARTIAL`, or `FAIL`;
2. record the report path and commit hash;
3. move unfinished acceptance criteria into a named continuation kernel;
4. do not silently narrow the original kernel and call it complete;
5. update the actual baseline section when a capability becomes reliable;
6. reassess only the next three provisional kernels unless a major architectural discovery requires broader change;
7. append the operator log and any durable operator notes;
8. preserve the historical roadmap rather than rewriting what was previously believed.

---

## 10. Churn alarms

Pause and reassess when any of these occur:

- three consecutive kernels advance Victory Core without a concrete Socio use;
- three consecutive Socio kernels add one-off storage, authority, projection, or scene logic;
- a kernel is reported PASS without the browser or multi-user proof named in its own acceptance criteria;
- a retroactive report changes the accepted scope instead of marking the work partial;
- a second rules integration requires copying a Socio package rather than registering different rules data/behavior;
- a concierge feature assumes Victory will pay ongoing hosting;
- Tauri, 3D, marketplace, billing, or institutional features begin before their dependency gates are met.
