# Architecture Milestones

**Produced by:** Kernel 99, 2026-09-12. This file exists so a future agent reading old code or old docs understands *why the vocabulary changed* — a comment or spec written before a milestone below may use a model Victory has since deliberately replaced. See `Construction/History/Kernel Index.md` for full per-kernel evidence, and `Construction/History/Superseded Doctrine Map.md` for the "do not resurrect" companion to this file.

---

### Character separated from user account (Kernel 23, "Character Cards and Personas")
Before this, "who you are" and "your account" were not clearly distinct concepts. Kernel 23 establishes the Character card/persona as an entity a user account controls but is not identical to — the seed of every later rule that authority and identity are resolved through the account, while Show *participation* is resolved through the Character.

### Showing separated from Session (Kernel 22, "Showing Model"; matured through Kernels 66, 70A, 71, 92)
Kernel 22 first distinguishes "a scheduled/ticketed event" (Showing) from "a live connected session" (Session) — a distinction the codebase spent 70 more kernels correctly wiring end-to-end. Kernel 66 gives Show Run/Audience Program/Roster real schema; Kernel 70A closes the live-stage loop and fixes a real Audience rehearsal-banner leak from this exact seam; Kernel 71 adds two-punch tickets and Showtime; Kernel 92 finally unifies "start a Show" into one shared Showtime/Showing-scheduler operation. **Session is not the canonical Show owner** — this was true well before Kernel 97, which only reconfirmed and documented it.

### Show Run roster became the canonical seat of Cast/Player/Crew authority (Kernel 66, hardened by Kernel 97)
`show_run_roster_members` is introduced in Kernel 66 as the Audience Program/Roster MVP. It does not become fully authoritative until Kernel 97 (2026-09-12) fixes `participation.LegacyLookupVenueRole`, which had always passed an empty Show Run ID into the resolver — silently skipping the one step that reads this table — misclassifying real roster Players as bare Audience on every `/api/world/*` snapshot. Before Kernel 97's fix, the roster table existed but was not reliably consulted by the venue-facing code path.

### Selected Character became canonical for Show participation (confirmed Kernel 97, evidence traces to Kernel 71's participation resolver)
Kernel 97's audit (2026-09-12) confirmed directly: `current_session_personas` and sitewide "active Character"/presence are **not** used as participation authority anywhere live; every remaining reference to them resolves display (name/portrait), never permission. The real authority path is the Show Run roster (above) plus location membership, both resolved through `participation.ResolveParticipationContext`.

### Presence made explicitly non-canonical (confirmed Kernel 97 and Kernel 98)
Both the Kernel 97 role-authority audit and the Kernel 98 resilience hunt independently confirmed: presence (who is currently connected) is never used as durable authority anywhere in the inventory. It indicates connection state only.

### Visibility became object-state projection, not a fixed per-role table (Kernel 90, "Canonical Stage Object State, Visibility & Cue Control")
Before Kernel 90, stage-object visibility was closer to a set of role-keyed rules scattered across call sites. Kernel 90 establishes a canonical stage-object state/visibility/cue model — the direct ancestor of the `stageobjects.CanPerceive`/`stageobjects.IsBackstageRole` projection functions Kernel 97 later audited and confirmed correctly distinguish *visibility* (a broader question, includes Crew) from *management* (a narrower question, excludes Crew per an explicit 2026-08-29 decision).

### Audience admissions became Showing-scoped, not Session- or global-scoped (Kernel 71, hardened by Kernel 93)
Two-punch Show tickets (Kernel 71) first tie admission to a specific Showing. Kernel 93's full Audience-seat dress rehearsal (2026-08-30) proved this end-to-end from a genuine non-Operator seat and fixed a real remaining bug where `access_grants` (a legacy, non-Showing-scoped table) could still leak visibility that `audience_admissions` (the correct, Showing-scoped table) should have gated.

### Migrations became a single embedded, checksummed system (Kernel 72)
Before Kernel 72 (2026-07-19), First Theater and Catharsis had drifted into a ~5000-line forked implementation with a less rigorous migration story. Kernel 72 collapses the fork and establishes the embedded, checksummed migration runner (`internal/migrate`) still in use today — migration filenames are not required to be numerically contiguous (the runner sorts by filename and tracks a checksum ledger), which explains several small numbering gaps documented in the migration chronology.

### Vue entered the frontend, starting with Storyboards (Kernel 94)
Every earlier venue is hand-rolled JS/DOM or Pixi. Kernel 94 (2026-08-31) is the first Vue rebuild, chosen deliberately as "the visual proving ground for the next generation of Victory's interface" — explicitly scoped to go deep on one venue rather than shallow across all of them, with the intent that Kernel 95 would carry successful patterns sitewide. **As of this writing, Kernel 95 has not been executed** — Vue remains scoped to Storyboards only; do not assume sitewide Vue adoption without checking the current state of `Construction/Kernels/Kernel 95` first.

### Windows consumer distribution architecture established (Kernel 100)
Kernel 100 (2026-09-11, PASS for alpha) establishes Victory's entire consumer packaging model in one kernel: a Velopack-based Windows installer/updater, Cloudflare Tunnel-based secure remote access (Quick Tunnel by default, loopback-only local backend binding, outbound-only tunnel — no port forwarding), and a mailto-based invite flow. This is the first point in Victory's history where "install this on a stranger's machine" became a real, tested path rather than a developer-only Linux/Docker deployment.

### Security and authority given dedicated, systemic audit kernels (Kernels 96, 97, 98)
Earlier eras fixed security and authority bugs as they were found within feature kernels (e.g., Kernel 76's password-reset log leak, Kernel 79's image-visibility fix, Kernel 86's dice-roll privacy fix). Kernels 96-98 (all 2026-09-12) are the first kernels whose entire purpose is systemic: a full adversarial security sweep (96), a canonical reconciliation of every role/authority resolver in the codebase against one source of truth per question (97), and a deliberate hunt for failure modes nobody had thought to ask about yet (98). This marks the shift from "fix bugs as found" to "systematically prove the absence of a class of bug" as Victory approached release.
