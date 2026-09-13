# Kernel Lineage — Victory's Construction Eras

**Produced by:** Kernel 99, 2026-09-12. Companion to `Construction/History/Kernel Index.md`, which has the per-number detail; this file tells the story of how one era's architecture became the next one's legacy constraint. Eras are drawn from repository evidence (reportbacks, migrations, operator-log entries), not projected backward from current vocabulary — where a term wasn't in use yet at the time, that's noted.

---

## Era 1 — The founding slice (Kernels 1–13)

Victory begins as a single venue: the-cave. Kernel 1 (surviving as "Kernel 1.1 — First Production Slice") is the earliest formal spec in the repository; the earliest git commits (`a1ff569`/`428c181`, "Add construction docs and kernel specs" / "initial DB migration") land the same era, one day before the first real backend commit (`c69fcd6`, "Built Core"). There is no surviving "Kernel 1.0" — 1.1 is the first traceable document, and its own reportback is titled "Kernel 1.2," implying the numbering was fluid from the very start.

This era establishes identity/invites/role authority (Kernel 2), the map/session/access flow (Kernel 3), role-filtered visibility (Kernel 4), and the first action-authority UI (Kernels 5–6). Presence attribution (Kernel 7) and identity/presence repair (Kernel 8) follow quickly — early signs that "who is here and what can they do" was a recurring, not one-time, problem. Greenroom and Trailers (Kernel 9), Info Booth/Mailbox (Kernel 10), note-card delivery (Kernel 11), the index-card element (Kernel 12), and Workshop venue placement (Kernel 13) round out the founding venue set.

**What this era handed forward:** the index card as a recurring UI primitive (still load-bearing many eras later), and the first version of "role determines what you see" — a question Victory would keep re-answering, with increasing rigor, all the way through Kernel 97.

---

## Era 2 — Early expansion and the first documented gap (Kernels 14–30)

Kernels 15, 18, 21, 22, 23, 24, 25, 29, and 30 have surviving specs; Kernels 16, 27, and 28 were reconstructed from code citations and the field guide's own historical narrative. **Kernels 14, 17, 19, 20, and 26 have no recoverable evidence anywhere** — not in git history, code comments, or the field guide's own kernel-by-kernel account, which itself skips from 24 to 27 in prose. This is the single largest unresolved gap in Victory's history, and it is preserved honestly rather than papered over (see `Historical Uncertainty Report.md`).

What does survive from this era: Producer Office and Director Chair (Kernel 15) establish the first production-authority venues; the context menu and move tool (Kernel 18) become foundational stage-interaction primitives; venue chat (Kernel 21) is the first real-time communication lane; the Showing model (Kernel 22) first separates "a session" from "a scheduled event," a distinction that would be fought over and re-fought for the next 70 kernels; Character cards and personas (Kernel 23) and Greenroom character dressing (Kernel 24) establish Character as an entity distinct from the user account that controls it. Kernel 25 is explicitly a documentation-reconciliation pass — the first of many, and a direct ancestor of this very kernel. PixiJS arrives as a stage-rendering spike (Kernel 29) before Cave tooling is reorganized around it (Kernel 30).

**What this era handed forward:** Character-as-distinct-from-account (the seed of the "selected Character is canonical for Show participation" doctrine locked in much later), and the Showing/Session distinction that Kernels 66, 70A, 71, and 92 would spend real effort correctly resolving.

---

## Era 3 — Discord, and the character-creation pipeline (Kernels 31–60)

This era has almost no surviving original spec documents but excellent reportback and operator-log coverage — reconstruction here was compilation, not invention. Discord integration is built in one deliberate arc: OAuth login (Kernel 32), operator bootstrap (Kernel 33), account authority surface (Kernel 34), server link/bootstrap (Kernel 35), channel mappings (Kernel 36), the "Discord mic" feature and session threads (Kernel 37), chat bridging (Kernel 38), and finally live Gateway ingestion (Kernel 39) — five kernels (35–39) whose only surviving evidence is migration filenames and a still-live bootstrap function, with no reportback or spec at all. Runtime hardening (Kernel 40) and a fresh-install/deployment proof (Kernel 41) follow immediately — Victory's first attempt at exactly the kind of proof this kernel (99) now does at much greater scale.

The middle of this era is Pixi/map tooling maturing (Kernels 45–48: shell rebasing, map layers, grid alignment, personal camera and card pinning) into the Warehouse asset model (Kernel 49) and First Theater's first persistent tokens (Kernel 50). Character creation gets a dedicated, multi-kernel pipeline: the Workbook foundation (Kernel 53), Chapter 2 Lifepath (54), Chapter 3 Archetypes (55), Chapter 4 first skill (56) — building toward Socio's skill/progression system (Kernel 60), the living character sheet, and a server-owned canonical dice system (Kernel 52) that every later roll-privacy fix (Kernel 86) would build on.

**What this era handed forward:** Discord as a genuinely optional, three-piece system (sign-in / server-link / gateway) rather than one monolith — a distinction Kernel 99's own Discord guide documents directly; the character-creation chapter pipeline that later kernels' Socio work assumes exists; and the first server-authoritative dice model.

---

## Era 4 — Show becomes the product center (Kernels 61–75)

This era is where "Show" stops being a loose session concept and becomes real architecture. Third Place and the Trailer/Face system mature (Kernels 61/61A, 65), then the Show Run primitive, Audience Program, and Roster MVP arrive together (Kernel 66) — the direct ancestor of the `show_run_roster_members` table Kernel 97 spent real effort making authoritative. The Show Instance model bridges Show Run to live sessions (Kernel 67); venue visibility gates and production onboarding follow (Kernel 68); Scene becomes a first-class domain object with a real staging model (Kernel 69).

Kernel 70 gives Shows a persistent stage, a rehearsal workspace, and the Go Cue foundation; 70A closes the live-stage loop and fixes a real Audience rehearsal-banner leak — an early instance of the exact visibility-leak class Kernel 97 would later reconcile system-wide. Two-punch Show tickets, Character participation, and Showtime itself arrive in Kernel 71, riding on a newly canonical participation resolver — the direct ancestor of `participation.ResolveParticipationContext`, the very function Kernel 97 fixed tonight. Kernel 72 is a major consolidation: one embedded, checksummed migration system replacing a ~5000-line First-Theater/Catharsis fork, plus real session-security hardening. The era closes with Catharsis's equip mode and Kessa's onboarding packet (73), the Locked Door/Ra guided-dialogue tutorial (74), and Tutorial Completion / Story So Far / Aftercare (75) — Victory's first attempt at a full new-player emotional arc, left PARTIAL pending Grant's own art and walkthrough review.

**What this era handed forward:** Show Run and its roster as the real seat of Cast/Player/Crew authority (later made fully canonical in Kernel 97); the participation resolver itself; and the first documented visibility leak of the exact shape Kernel 97 would spend a whole kernel eliminating system-wide.

---

## Era 5 — Documents, Storyboards, and shared authoring surfaces (Kernels 76–84)

Security and hosted-readiness get their first dedicated audit (Kernel 76: closed a password-reset log leak, gated public signup, shipped the `victory-recover` break-glass tool), followed immediately by full private-client readiness — account deletion, export, and encrypted off-host backup (Kernel 77) — and a canonical venue-seed repair (77A) after the rebuild that 76 required. eWrite is built from nothing: a Markdown+full-text-search pipeline, Writer's Room and Library venues (78), proven against a real 357KB manuscript, then extended across four passes in one spec (79/79A/Goal F/Goal C&E) into a Skill Directory, ruleset-wide navigation, and hierarchical export.

Storyboards follows the same pattern — a working core first (80), then a full presentation rework with real drag-and-drop and Playwright-verified bug fixes (81), structural slugs (81A), and a genuine second creative mode, Timeline, built as a code-defined immutable template rather than a database row (82). Kernel 83 builds venue-agnostic leadership/turn-state coordination and a Presence Tray from scratch, wiring Storyboards as its first consumer. Kernel 84 — this era's own closing kernel — is itself a reconciliation pass: it fixed a real WebSocket context-lifetime bug, and produced the evidence-cross-checked ledger (`kernel-history-reconciliation-through-83.md`) that Kernel 99 reused directly rather than re-investigating.

**What this era handed forward:** eWrite and Storyboards as Victory's first genuinely general-purpose authoring surfaces (not venue-specific tools); the checksummed-migration discipline from Kernel 72 proven safe under continued heavy schema growth; and a second, real precedent (Kernel 84) for exactly what Kernel 99 does — an agent stopping to reconcile the record rather than letting drift accumulate.

---

## Era 6 — Sustained play, dice, and cartography (Kernels 85–87)

A short but dense era: cohort-scoped Scene progression and a canonical HP/status layer for sustained Socio play (85); real audience-resolved dice-roll visibility, fixing a genuine privacy bug where rolls weren't correctly gated by audience membership (86), then giving dice real spatial landing coordinates that survive reconnect (86A); and a full shared cartographic drawing-object model with in-character chat (87), left PARTIAL because live-stroke streaming and a full 50-item browser matrix weren't completed.

**What this era handed forward:** the audience-resolved dice-visibility model that Kernel 98's carried-in nil-pool investigation traced directly back to, and confirmed still correct in production.

---

## Era 7 — Guided play and the Showtime button (Kernels 88–92)

Director Prepared Play and a Training Arena (89) give Directors real authoring tools; canonical stage-object state, visibility, and cue control (90) close out the last major "who can see/control what" model before Kernel 97's full reconciliation; Victory Campus Tours (91) give new users guided onboarding; and Showtime Composition with a Showing scheduler (92) unifies "start a Show" into one shared domain operation. All five of these kernels are reconstructed from reportbacks alone — no operator-issued spec files for 88–92 were ever committed to the repository, though the reportbacks themselves are detailed and self-consistent enough to support HIGH CONFIDENCE.

**What this era handed forward:** the Showtime button as the real, single entry point for starting a Show — the surface Kernel 97's `LegacyLookupVenueRole` fix (deriving a live Show Run from a venue's current session) was built to serve correctly.

---

## Era 8 — Dress rehearsal, visual maturity, and hardening for release (Kernels 93–100)

The final era is where Victory stops asking "does this feature exist" and starts asking "is this safe to hand to a stranger." Kernel 93 puts a real, non-Operator Audience seat through a full dress rehearsal, finding and fixing a genuine `audience_admissions`-vs-`access_grants` visibility bug along the way. Kernel 94 rebuilds Storyboards in Vue with real visual design investment — built and live-reviewed positively pass-by-pass, though its own formal acceptance gate was never separately invoked, leaving it PARTIAL by the letter of its own spec. **Kernel 95, the shared-shell unification kernel meant to carry that visual language sitewide, was never executed at all** — a real, named gap in the sequence, not yet built.

Kernel 96 runs a full adversarial security sweep and fixes real findings in production. Kernel 97 reconciles every role/authority resolver in the codebase against a single canonical source per authority question, fixing three real bugs along the way — including the exact Operator-role client-side override that had made "every other view feels broken" a literal, mechanical fact rather than an impression. Kernel 98 hunts unknown unknowns across the entire product and finds zero launch blockers. Kernel 99 — this kernel — is the historical archive and documentation kernel you are reading now. Kernel 100 ships a working Windows consumer installer, updater, and secure remote-access path, verified on real hardware, with three items still requiring Grant's own hands-on action rather than more engineering.

**What this era handed forward:** a product that has been genuinely audited for security, authority correctness, and resilience — not merely built — plus one clearly named piece of unbuilt scope (Kernel 95) that a future kernel should either execute or formally retire rather than let linger silently.
