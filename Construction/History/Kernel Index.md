# Kernel Index — Kernels 1 through 100

**Produced by:** Kernel 99 (Canonical Construction Archive, Fresh Install Proof & Operator Documentation), 2026-09-12.
**Purpose:** the fastest way to understand Victory's construction history. For every kernel number 1–100: what it built, where the evidence lives, and what debt it handed forward.
**Method:** every surviving spec file, every reportback in `Construction/OperatorLogs/`, `operator-log.md`'s chronological entries, git history, and migration filenames were read directly. Nothing below is invented — provenance labels distinguish surviving originals from evidence-based reconstruction, and five kernel numbers are marked UNKNOWN after an exhausted search rather than guessed. See `Construction/History/Kernel Lineage.md` for the narrative version of this same history, and `Construction/History/kernels.json` for the machine-readable form.

**Provenance key:** ORIGINAL · ORIGINAL + AMENDED · RECONSTRUCTED — HIGH CONFIDENCE · RECONSTRUCTED — PARTIAL · NUMBER RESERVED / NOT EXECUTED · MERGED / SUPERSEDED · UNKNOWN.

---

## Kernels 1–30 — the earliest era (pre-formal-spec through first Cave tooling)

| Kernel | Name | Provenance | Status | Primary change | Evidence | Successor/debt |
|---|---|---|---|---|---|---|
| 1 | First Production Slice (the-cave) | ORIGINAL + AMENDED (1.1→1.2) | PASS | Founding venue slice: the-cave, capacity/role planning | `Kernels/kernel1`, `reportback1` | Basis for all later Cave work |
| 2 | Identity, Invites, Role Authority, Workshop Asset Floor | ORIGINAL | PASS | Identity/invite system, role authority baseline | `Kernels/kernel2`, `reportback2` | Feeds Kernel 3 access flow |
| 3 | Victory Map + Cave Session + Access Flow | ORIGINAL | PARTIAL PASS (strong) | Map GUI, session/access flow | `Kernels/kernel3`, `reportback3`, `reportbackGUImap` (companion) | — |
| 4 | Reveal, Role-Filtered Cave UI, First Illusion Layer | ORIGINAL | per reportback | Role-filtered visibility/reveal UI | `Kernels/kernel4`, `reportback4` | Basis for later visibility model |
| 5 | Action UI Layer (First Production Tooling) | ORIGINAL | PASS | Action UI, role-aware controls | `Kernels/kernel5`, `reportback5` | — |
| 6 | Action Authority | ORIGINAL | per spec | Action authority model | `Kernels/kernel-6-action-authority.md` | — |
| 7 | Presence Attribution | ORIGINAL | per spec | Presence attribution | `Kernels/kernel-7-presence-attribution.md` | — |
| 8 | Identity Surface Presence Repair | ORIGINAL | per spec | Identity/presence repair | `Kernels/kernel-8-identity-surface-presence-repair.md` | — |
| 9 | Greenroom, Trailers | ORIGINAL | per spec | Greenroom + Trailers venues | `Kernels/kernel-9-greenroom-trailers.md` | — |
| 10 | Info Booth, Mailbox | ORIGINAL | per spec | Info Booth + Mailbox | `Kernels/kernel-10-info-booth-mailbox.md` | — |
| 11 | Note Card Delivery | ORIGINAL | per spec | Note-card delivery | `Kernels/kernel-11-note-card-delivery.md` | — |
| 12 | Index Card Element | ORIGINAL | per spec | Index-card primitive | `Kernels/kernel-12-index-card-element.md` | — |
| 13 | Workshop Venue Placement | ORIGINAL | per spec | Workshop venue placement | `Kernels/kernel-13-workshop-venue-placement.md` | — |
| **14** | — | **UNKNOWN** | UNKNOWN | No trace found anywhere | git log, code comments, and field-guide narrative all searched — none found | Open gap |
| 15 | Producer Office, Director Chair | ORIGINAL | per spec | Producer Office + Director Chair venues | `Kernels/kernel-15-producer-office-director-chair.md` | — |
| 16 | Venue Seed and Bootstrap Expansion | RECONSTRUCTED — HIGH CONFIDENCE | IMPLEMENTED — status not recorded | Fresh-install venue seed (`EnsureKernel16VenueSurface`) | `Construction/History/Reconstructed Kernels/Kernel 16 — Venue Seed and Bootstrap Expansion.md`; `access/kernel16_venue_bootstrap.go` + 7 later citations | Every later venue-capability-flag kernel updated this seed |
| **17** | — | **UNKNOWN** | UNKNOWN | No trace found anywhere | git log, code comments, field guide all searched — none found | Open gap |
| 18 | Context Menu, Move Tool | ORIGINAL | per spec | Context menu + move tool | `Kernels/kernel-18-context-menu-move-tool.md` | — |
| **19** | — | **UNKNOWN** | UNKNOWN | No trace found anywhere | git log, code comments, field guide all searched — none found | Open gap |
| **20** | — | **UNKNOWN** | UNKNOWN | No trace found anywhere | git log, code comments, field guide all searched — none found | Open gap |
| 21 | Venue Chat v1 | ORIGINAL | per spec | First venue chat | `Kernels/kernel-21-venue-chat-v1.md` | — |
| 22 | Showing Model | ORIGINAL + AMENDED | per spec | Session-to-Showing link | `kernel-22-showing-model.md` + `-v1.md` | Superseded much later by Kernels 66/71/92 |
| 23 | Character Cards and Personas | ORIGINAL + AMENDED | per spec | Story-first character cards, persona equip | `kernel-23-character-cards-personas.md` + `-story-first-...md` | Feeds Kernel 24 |
| 24 | Greenroom Character Dressing | ORIGINAL + AMENDED | per spec | Moved character editing to Greenroom | `kernel-24-greenroom-character-dressing.md` + `-dressing-room.md` | — |
| 25 | Current State Canon | ORIGINAL | per spec | Doc reconciliation pass | `Kernels/kernel-25-current-state-canon.md` | Precedent for later doc-reconciliation kernels |
| **26** | — | **UNKNOWN** | UNKNOWN | No trace found anywhere; field guide's own narrative skips 25→27 too | git log, code comments, field guide all searched — none found | Open gap |
| 27 | Closed-Showing Review Surface | RECONSTRUCTED — PARTIAL | IMPLEMENTED — status not recorded | Director's Chair closed-Showing review | `Construction/History/Reconstructed Kernels/Kernel 27 — Closed-Showing Review Surface.md`; field guide only | Likely superseded by later Show/Showing rework, not confirmed |
| 28 | Live Director Console | RECONSTRUCTED — PARTIAL | IMPLEMENTED — status not recorded | Live current-Showing control console | `Construction/History/Reconstructed Kernels/Kernel 28 — Live Director Console.md`; field guide + `director_console.go:355` | Foundation for `director_console.go`, still live |
| 29 | PixiJS Stage Spike | ORIGINAL | per spec | Pixi rendering spike | `Kernels/kernel-29-pixijs-stage-spike.md` | — |
| 30 | Cave Organization, Renderer Containment | ORIGINAL | per spec | Cave tooling/panel organization | `Kernels/kernel-30-cave-organization-renderer-containment.md` | Kernel 30.2 audit follow-up |

---

## Kernels 31–60 — Discord foundation, character workbook, Socio, dice

| Kernel | Name | Provenance | Status | Primary change | Evidence | Successor/debt |
|---|---|---|---|---|---|---|
| 30.2 | Consolidation Audit | ORIGINAL (audit, not a build kernel) | Consolidation pass | Consolidated drift/unfinished work for next kernel-maker | `kernel-30.2-audit.md` | — |
| 31 | Middle-School Stage Edge/Drawer Layout | ORIGINAL | PARTIAL, continued later | Clean stage-shell framing | `kernel-31-middle-school-stage-edge-drawer-layout-v1.md`; `kernel-31-reportback.md` | Continued by later Cave/stage work |
| 32 | Discord OAuth Primary Login | RECONSTRUCTED — HIGH CONFIDENCE | Complete for the OAuth slice | Discord OAuth start/callback, identity linking | `kernel-32-reportback.md` (no surviving spec) | Folded into Kernel 33 |
| 33 | Operator Bootstrap + Canon Capture | RECONSTRUCTED — HIGH CONFIDENCE | Complete for this slice | Safe operator CLI for producer grants | `kernel-33-reportback.md` (no surviving spec) | — |
| 34 | Account Authority Surface | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | Account summary endpoint, Discord link status, producer authority in account UI | `operator-log.md` 2026-05-29; commit `a2d77a5` | — |
| 35 | Discord Server Link and Bootstrap | RECONSTRUCTED — HIGH CONFIDENCE | Implemented — status not recorded | Discord server link/bootstrap (install, callback, unlink) | `Construction/History/Reconstructed Kernels/Kernel 35 — Discord Server Link and Bootstrap.md`; migrations `019`/`020_kernel35_*.sql`; commit `adce59d` | — |
| 36 | Discord Channel Mappings | RECONSTRUCTED — HIGH CONFIDENCE | Implemented — status not recorded | Venue-to-Discord-channel mapping | `Construction/History/Reconstructed Kernels/Kernel 36 — Discord Channel Mappings.md`; migration `021_kernel36_*.sql` | — |
| 37 | Discord Mic and Session Threads | RECONSTRUCTED — HIGH CONFIDENCE | Implemented — status not recorded | "Discord mic" feature + session-threads table | `Construction/History/Reconstructed Kernels/Kernel 37 — Discord Mic and Session Threads.md`; migration `022_kernel37_*.sql` | — |
| 38 | Discord Chat Bridges | RECONSTRUCTED — HIGH CONFIDENCE | Implemented — status not recorded | Venue↔Discord chat bridging | `Construction/History/Reconstructed Kernels/Kernel 38 — Discord Chat Bridges.md`; migration `023_kernel38_*.sql` | — |
| 39 | Discord Gateway Surface | RECONSTRUCTED — HIGH CONFIDENCE | Implemented — status not recorded | Live Gateway ingestion (edits, sparse-payload backfill) | `Construction/History/Reconstructed Kernels/Kernel 39 — Discord Gateway Surface.md`; `EnsureKernel39DiscordGatewaySurface` (`main.go:190`) | Schema formalized later by Kernel 72's migration 054 |
| 40 | Runtime Hardening + House Mic Regression Harness | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | Runtime hardening, mic regression harness | `operator-log.md` 2026-06-08 | — |
| 41 | Fresh Install / Deployment Proof | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | Fresh-install/deployment proof pass | `operator-log.md` 2026-06-08 | Precedent for K99's own fresh-install work |
| 42 | Migration Baseline Cleanup + Documentation Audit | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | Migration baseline cleanup, doc audit | `operator-log.md` 2026-06-09 | — |
| 43 | Discord Audio Left Tray Foundation | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | Discord audio left-tray foundation | `operator-log.md` 2026-06-09 | — |
| 44 | Discord Audio Presence / Speaker Feasibility | RECONSTRUCTED — HIGH CONFIDENCE | Implemented (v1) | Discord audio presence/speaker feasibility | `operator-log.md` 2026-06-09 | Audio behavior later left untouched by K41 note |
| 45 | Venue Shell Rebase + Shared Helper Extraction | RECONSTRUCTED — HIGH CONFIDENCE | Complete for this slice | Tightened public map shell; shared venue-shell helpers | `kernel-45-reportback.md` | — |
| 46 | Pixi Map Layer + Workshop Map Upload | RECONSTRUCTED — HIGH CONFIDENCE | Complete for this slice | First Theater map workflow: upload/crop/fit/scale | `kernel-46-reportback.md` | — |
| 47 | Pixi Grid Primitive + Map Alignment | RECONSTRUCTED — HIGH CONFIDENCE | Implemented, promoted to canon | Pixi grid primitive, map alignment | `operator-log.md` 2026-06-20 | — |
| 48 | Personal Stage Camera and Card Pinning | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | Personal browser-local camera; world/overlay card pinning | `Construction/History/Reconstructed Kernels/Kernel 48 — Personal Stage Camera and Card Pinning.md`; `operator-log.md` 2026-06-21 | Fixed same-day by 48.1 |
| 48.1 | Card Attachment World/Overlay Fix | RECONSTRUCTED — HIGH CONFIDENCE | PASS | Fixed world-vs-overlay placement across move/pin/duplicate/create | `kernel-48.1-reportback.md` | — |
| 49 | Warehouse Asset Model | RECONSTRUCTED — HIGH CONFIDENCE | PASS | Workshop asset model extended into durable Warehouse model | `kernel-49-reportback.md` | — |
| 50 (+50+) | First Theater Persistent Tokens | RECONSTRUCTED — HIGH CONFIDENCE | PASS + hardening follow-on | Persistent token placement, grid snapping; 50+ fixed refresh-persistence | `kernel-50-reportback.md`, `kernel-50-plus-reportback.md` | "50+" folded into this row, not an independent number |
| 51 (+51A/51B) | First Theater Runtime Refactor | RECONSTRUCTED — HIGH CONFIDENCE | PASS, pending multi-client verification | Runtime responsibility split, warehouse/Discord fixes | `kernel-51-reportback.md` | Multi-client live verification pending at report time |
| 52 | Canonical Dice System | RECONSTRUCTED — HIGH CONFIDENCE | PASS, pending live multi-client verification | Server-owned dice generation/validation/persistence | `kernel-52-reportback.md` | Consumed later by Kernel 60 |
| 53 | Character Workbook Foundation | RECONSTRUCTED — HIGH CONFIDENCE | PARTIAL PASS | `character_cards` extended with workbook metadata | `kernel-53-reportback.md`; migration `031_kernel53_*.sql`; operator-log "Correction" entry | Correction folded into same kernel number |
| 54 | Chapter 2 Lifepath System | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | Chapter 2 lifepath character creation | `operator-log.md` | — |
| 55 | Chapter 3: Character Archetypes | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | 14 archetypes, quiz, projections | `operator-log.md` | Feeds Kernel 56 |
| 56 | Chapter 4: First Skill and Courtyard Entry | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | First-skill selection, Courtyard boundary | `operator-log.md` | Feeds Kernel 60 |
| 57 | Catharsis Onboarding Repair + Greenroom Polish | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | Onboarding repair, archetype fixes | `operator-log.md` 2026-07-03 | — |
| 58 | Greenroom Face Projection & Contrast Passes | RECONSTRUCTED — HIGH CONFIDENCE | Implemented | Visual refinement, face projection, workbook hero pass | `operator-log.md` 2026-07-03 (5 entries) | — |
| 59 | Command Registry + Execution | RECONSTRUCTED — HIGH CONFIDENCE | PASS, minor findings | Slash-command registry/execution surface | commit `f324901`; `kernel-59-reportback.md` | Kernel 60 builds on this surface |
| 59A | Face Curation / Shared Projector | RECONSTRUCTED — HIGH CONFIDENCE | Phase 1 PARTIAL → Phase 3/Full-Scope PASS | Owner-curation layer, shared Face/Mechanics projector | `kernel-59A-reportback.md` + phase3/full-scope variants; migrations `034`/`035` | `/quote add` multi-quote deferred |
| 60 | Socio Skills, Progression, Living Character Sheet | ORIGINAL | PASS, pending browser-level verification | Skill dice ladder, `character_skills`, right-tray sheet UI | `kernel-60-socio-skills-progression-and-sheet-DRAFT.md` (finalized Rev 0.3 despite filename); `kernel-60-reportback.md` | — |

---

## Kernels 61–92 — Trailers/Face, Show/Showing architecture, eWrite, Storyboards, Cartography, Guided Play

| Kernel | Name | Provenance | Status | Primary change | Evidence | Successor/debt |
|---|---|---|---|---|---|---|
| 61 | Trailer Player Workbook, Face Compiler, Legacy Profile Migration | RECONSTRUCTED — HIGH CONFIDENCE | SUPERSEDED by 61A (reached PARTIAL alone) | Schema/domain/HTTP/legacy-migration/read-only My Face page | `kernel-61-reportback.md` | Completed by 61A |
| 61A | Trailer Workbook UI, Social Viewing, Migration Closure | RECONSTRUCTED — HIGH CONFIDENCE | PASS (2026-07-09) | Workbook UI, Face compiler, WS live-invalidation, cross-user viewing | `kernel-61A-reportback.md` | None open |
| 62 | Player Relationship Matrix, Private Notes, Relationship Journals | ORIGINAL | PASS (2026-07-09) | Two-user private notes/journals with archive | `kernel-62-reportback.md` + spec | 2 pre-existing unrelated Discord test failures noted |
| 63 | Discord Test Fixture-Leak Cleanup, Back-to-Map Nav Consistency | RECONSTRUCTED — HIGH CONFIDENCE | PASS | Fixed fixture leak; consistent back-to-map nav | `kernel-63-reportback.md` | 1 pre-existing unrelated failure (confirmed via git stash) |
| 64 | DB Test Isolation and Live-DB Safety Gate | RECONSTRUCTED — HIGH CONFIDENCE | PASS | `TEST_DATABASE_URL` hard-required; destructive reset can't reach live DB | `kernel-64-reportback.md` | None open |
| 65 | Third Place Headshot Commons MVP | ORIGINAL | PASS, live | Headshot lifecycle, live Trailer Face projection, My People | `kernel-65-reportback.md` + spec | None open |
| 66 | Show Run, Audience Program, Roster MVP | ORIGINAL | PASS | Show Run primitive, Audience Program, Roster MVP | `kernel-66-reportback.md` + spec | Fixed a stale roadmap-doc numbering collision |
| 67 | Show Instance Model and Show Run Bridge | ORIGINAL | PASS, live | Show instance model bridging Show Run to live sessions | `kernel-67-reportback.md` + spec | None open |
| 68 | Venue Visibility Gates, Stage Mgmt Surface, Production Onboarding | ORIGINAL | PASS, live | Computed Trailer-Face readiness gate; Stage Mgmt surface | `kernel-68-reportback.md` + spec | None open |
| 69 | Scene Library and Show Staging Model | ORIGINAL | PASS, live | First real Scene domain object; Show staging model | `kernel-69-reportback.md` + spec | None open |
| 70 | Persistent Show Stage, Rehearsal Workspace, Go Cue Foundation | RECONSTRUCTED — HIGH CONFIDENCE | PASS | Location-scoped Scene ownership; Cues + execution | `kernel-70-reportback.md` | None open |
| 70A | Live Stage Closure and Alpha Path Alignment | RECONSTRUCTED — HIGH CONFIDENCE | PASS, live | StartShowSession; stage state survives resume | `kernel-70a-reportback.md` | None open |
| 71 | Two-Punch Show Tickets, Character Participation, Showtime | RECONSTRUCTED — HIGH CONFIDENCE | PASS, live | Canonical participation resolver; ticket schema; Audition Hall UI | `kernel-71-reportback.md` | — |
| 72 | Single Schema Truth, Shared Stage Engine, Session Security | RECONSTRUCTED — HIGH CONFIDENCE | PASS, live | Embedded checksummed migrations; collapsed FT/Catharsis fork; session hardening | `kernel-72-reportback.md` | None open |
| 72A | Post-Deploy Bug Fixes and Venue Capability Flags | RECONSTRUCTED — HIGH CONFIDENCE | PASS, live | Fixed 4 live bugs | `kernel-72A-reportback.md` | — |
| 73 | Catharsis Equip Mode, Character Inventory, Kessa Program Packet | RECONSTRUCTED — HIGH CONFIDENCE | PASS, live, 18/18 criteria | Equip mode; Character inventory; Kessa packet | `kernel-73-reportback.md` | None open |
| 74 | Locked Door Intentions, Ra Guided Dialogue, Tutorial Handoff | RECONSTRUCTED — HIGH CONFIDENCE | PASS, live, 21/21 criteria | Kessa→door→Ra→Crown Bet→handoff | `kernel-74-reportback.md` | 5 defects found+fixed same pass |
| 75 | Tutorial Completion, Story So Far, Aftercare, MVP Proof | ORIGINAL (operator-issued revision, authoritative) | PARTIAL — 2 criteria outstanding | Tutorial completion, Story So Far generation, Aftercare | `kernel-75-reportback.md` + spec | Criteria 25 (art)/27 (operator walk) open |
| 75 (draft) | Tutorial Completion, Aftercare, Director-Controlled Continuation | SUPERSEDED | Superseded pre-implementation | — | `kernel-75-tutorial-completion-aftercare-continuation-v0.1.md` (self-marked) | Kept for provenance only |
| 75 Followup | Golden Journey, Presentation, Profile UI | RECONSTRUCTED — HIGH CONFIDENCE | PARTIAL/operational | Character-selection auto-enrollment fix; UI fixes | `kernel-75-followup-reportback.md` | — |
| 76 | Canonical State, Security, Hosted-Readiness Audit | ORIGINAL | PASS (readiness Level 3) | Password-reset log leak closed; break-glass tool; full DB rebuild | `kernel-76-reportback.md` | 4 items → K77 |
| 77 | Private-Client Readiness | ORIGINAL + AMENDED | PASS (Level 4 for deletion/export/backup) | Account deletion, export, encrypted off-host backup+restore | `kernel-77-reportback.md` | Brevo auth blocked (account-approval gate) |
| 77A | Canonical Venue Seed Repair | ORIGINAL | PASS | Fixed Audition Hall zero-venue-row gap | `kernel-77a-reportback.md` | None |
| 78 | eWrite Foundation | ORIGINAL | PASS | Markdown+FTS pipeline; Writer's Room/Library | `kernel-78-reportback.md` | Anonymous public reading deferred |
| 79 (Phase 1/79A/Goal F/Goal C&E) | eWrite Integration and Rules Navigation | ORIGINAL (multi-pass, one spec) | PASS across all 4 passes | Image-visibility fix, Socio hierarchy, Skill Directory, Library nav, object rule links, hierarchical export | `kernel-79-reportback.md`, `kernel-79a-reportback.md`, `kernel-79-goal-f-reportback.md`, `kernel-79-goal-ce-reportback.md` | Raw-Postgres-error leak on malformed input, left unfixed |
| 80 | Storyboards Core | ORIGINAL | PASS vs written spec | New `storyboards` package, 39 tests, first Playwright automation | `kernel-80-reportback.md` | Presentation gaps scoped to K81 |
| 81 | Storyboards Presentation Rework | ORIGINAL | PASS | CSS Grid, drag-drop, lightbox | `kernel-81-reportback.md` | — |
| 81A | Storyboard Structural Slugs | RECONSTRUCTED — HIGH CONFIDENCE | PASS | Deterministic slugs, migration 094 | `operator-log.md` + migration 094 + test file | `sort_order` race documented, not fixed |
| 82 | Storyboards Timeline Mode | ORIGINAL | PASS | Code-defined Timeline template; Reference Panel | `kernel-82-reportback.md` | — |
| 83 | Venue Leadership & Turn State | ORIGINAL | PASS | New `venuecoordination` package; Presence Tray | `kernel-83-reportback.md` | WS-pump context-reuse bug flagged |
| 84 | Canonical Reconciliation & Runtime Cleanup | ORIGINAL | see own reportback | Fixed WS context bug; reconciled ledger/roadmap | `kernel-84-reportback.md` | Timeline boundary gap in `AddColumn` documented |
| 85 | Socio Sustained Play — Cohorts, Scene Progression, Game Status | ORIGINAL | PASS | Cohort-scoped Scene progression; HP/status layer | `kernel-85-reportback.md` + spec | None open |
| 86 | Theatrical Roll Projection | ORIGINAL | PASS | Real audience-resolved roll visibility (fixed a privacy bug) | `kernel-86-reportback.md` + spec | Explosion semantics unchanged |
| 86A | Spatial Dice Landing & Explosion-Safe Projection | ORIGINAL | PASS | Dice land at real map coordinates | `kernel-86A-reportback.md` + spec | 86B ruled unnecessary |
| 87 | Shared Cartographic Rendering & In-Character Play | ORIGINAL | PARTIAL | Drawing-object model, Detail View, PNG export, IC chat | `kernel-87-reportback.md` + spec | Live-stroke streaming not built |
| 88 | Socio Guided Play Surface | RECONSTRUCTED — HIGH CONFIDENCE | PARTIAL | Fate/Stance/Interrupt-Help stack | `kernel-88-reportback.md` | Player-roll path untested |
| 89 | Director Prepared Play & Training Arena | RECONSTRUCTED — HIGH CONFIDENCE | PASS, live | Director's Chair authoring; merchant-from-corpus | `kernel-89-reportback.md` + kernel-maker variant | 2 explicit scope decisions stated |
| 90 | Canonical Stage Object State, Visibility & Cue Control | RECONSTRUCTED — HIGH CONFIDENCE | PASS (1 evidence gap) | Canonical stage-object state/visibility/cue model | `kernel-90-reportback.md` | Browser-render evidence gap stated honestly |
| 91 | Victory Campus Tours & Guided Onboarding | RECONSTRUCTED — HIGH CONFIDENCE | PASS for confirmed scope | Reusable spotlight/tour system | `kernel-91-reportback.md` | Scope reduction was a deliberate agreement |
| 92 | Showtime Composition & Showing Scheduler | RECONSTRUCTED — HIGH CONFIDENCE | PASS for confirmed scope | Showtime button → pick/schedule → SHOWTIME | `kernel-92-reportback.md` | True/End-Showtime Go-tested, not browser-scripted |

---

## Kernels 93–100 — dress rehearsal through consumer release

| Kernel | Name | Provenance | Status | Primary change | Evidence | Successor/debt |
|---|---|---|---|---|---|---|
| 93 | Audience Seat Dress Rehearsal | ORIGINAL | PASS (2026-08-30, Grant's direct acceptance) | Full Audience-seat dress rehearsal; fixed `audience_admissions` vs `access_grants` visibility bug | spec + `kernel-93-audience-dress-rehearsal-ledger.md` | Handed visual-language groundwork to K94 |
| 94 | Storyboards Vue Rebuild + Major Beautification | ORIGINAL | PARTIAL — built and live-reviewed pass-by-pass with positive reactions, plus a PDF-export follow-up, but the spec's own combined Empty/Working/Full acceptance gate (§32/34) was never formally invoked as its own verdict | Full Vue rebuild of Storyboards, visual redesign, motion, PDF export | spec + `kernel-94-storyboards-vue-rebuild-ledger.md` + `Construction/Domains/Design/Kernel 94 Visual Language.md` | Handed reusable visual-language patterns + 2 documented Vue/CSS traps to K95 |
| 95 | Unified Victory Shell, Future Trays, Continuity & Guidance | ORIGINAL | **RECONSTRUCTED — HIGH CONFIDENCE, IMPLEMENTED — status not recorded.** Corrected 2026-09-17: earlier archived as "not executed" — wrong. Real, extensive, multi-pass implementation exists in shipped code (`frontend/lib/stage-runtime/runtime.js` and `frontend/venues/catharsis/index.html` cite "Kernel 95 Pass 2" through at least "Pass 7"; production cache-bust tag `runtime.js?v=k95-pass2a`), just with **no reportback or closure ledger ever written** for any pass | Shared live-shell rebuild (reactive runtime primitives, drawer/presence patterns, Guide command grouping, Fanmail styling, popup entrance patterns) across at least 7 passes | spec + code citations only, no reportback | Genuinely unclear how much of the spec's full scope those 7 passes cover, or whether more passes happened after — no closure record exists to check against |
| 96 | Security, Privacy Boundaries, Fresh-Install Trust & Internet Hardening | ORIGINAL | PASS (2026-09-12) | Trust/default audit + full adversarial security sweep, real findings fixed in production | spec + `kernel-96-security-hardening-ledger.md` | Deferred: CSP/HSTS, release cleanup, git history scrub — named, not hidden |
| 97 | Canonical Role, Authority & Experience Reconciliation | ORIGINAL | PASS (2026-09-12) | Reconciled role/authority resolvers; fixed 3 real bugs (Operator role-lie, roster misclassification, cross-Location leak) | spec + `kernel-97-role-authority-reconciliation-ledger.md` + `Construction/Domains/Identity/Canonical Role and Authority Resolution.md` | Residual: `isOperatorDiceRoller` duplicate; no general preview tool |
| 98 | Unknown Unknowns, Failure Modes & Product Resilience Hunt | ORIGINAL | PASS (2026-09-12), zero blockers | Broad resilience/failure hunt; 3 carried K96 findings resolved | spec + `kernel-98-unknown-unknowns-resilience-hunt-ledger.md` | Carry-forward list routed to K99/K101/K102+ |
| 99 | Canonical Construction Archive, Fresh Install Proof & Operator Documentation | ORIGINAL | IN PROGRESS (this kernel) | This archive, fresh-install proof, and documentation suite | spec; this file and its siblings under `Construction/History/` and `Docs/` | — |
| 100 | Windows Consumer Installer, Secure Remote Access & Automatic Updates | ORIGINAL | PASS for alpha (2026-09-11) | Windows installer, updater, secure remote access, invite flow, data-deletion loop verified on real hardware | spec + `kernel-100-windows-installer-ledger.md` | 3 open items requiring Grant's own action: code signing, external-network proof, A→B data-integrity proof |

---

## Open historical gaps (honest, not hidden)

Five kernel numbers have no recoverable evidence anywhere in the repository after an exhausted search (git log across all branches, code comments, migration filenames, `operator-log.md`, and the Kernel Maker Field Guide's own historical narrative — which itself skips from 24 to 27 in prose even though 25 has a surviving spec):

- **Kernel 14, 17, 19, 20, 26** — marked **UNKNOWN**. It is possible these numbers were reserved during early informal planning and never executed, were merged silently into an adjacent kernel before the project's documentation habits matured, or their evidence was lost in an early history rewrite. No basis exists to say which. See `Construction/History/Historical Uncertainty Report.md` for the full accounting.

No other gaps remain — kernels 1–13, 15–16, 18, 21–25, 27–100 all have at least one piece of concrete, cited evidence.
