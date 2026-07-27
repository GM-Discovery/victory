# Kernel 75 — Tutorial Completion, Aftercare, and Director-Controlled Continuation

**Revision:** 0.1 (drafted by the builder at the close of Kernel 74; operator to confirm or re-cut)
**Primary track:** Golden Journey / playable Socio tutorial
**Depends on:** Kernel 70/70A, 72/72A, 73/73A, **74**
**Expected first migration:** confirm against both `backend/migrations/` and the live `schema_migrations` ledger immediately before naming. Kernel 74 ended at `068` (ledger at 69 entries).

---

## 0. Purpose

Kernel 74 got the Player *out the door*. They are standing on a placeholder map with placeholder copy, and nothing tells them what they accomplished or what happens next.

Kernel 75 makes the ending feel like an ending, and hands control back to the Director cleanly.

```text
Player reaches the tutorial-handoff projection
→ sees a finished Scene, not a placeholder
→ is told plainly what they just did and what they now have
→ their progression is acknowledged
→ Aftercare is offered
→ the Director advances the shared Show when the table is ready
```

This is deliberately the *smaller, finishing* half of the pair. Kernel 74 built machinery; Kernel 75 should mostly build presentation, on top of machinery that already exists.

---

## 1. What Kernel 74 already provides (do not rebuild)

Confirmed working and live:

- `participant_local_projections` — the per-Player handoff projection, Character-scoped, cleared on shared-Scene advance. The Scene it points at is an ordinary Scene Library row (`tutorial-handoff`), editable in Scene Setup.
- `participant_tutorial_progress` — five durable milestones per (user, Character, Show).
- `tutorial.ListShowProgress` + `GET /api/shows/{show_id}/tutorial-progress` — the Directors+ status list, names and statuses only, **no denominator**.
- `messages` `backstage_note` type + `GET /api/backstage-notes` + the Catharsis Backstage tab.
- `dialogue_packets`/`dialogue_topics` — reusable for any second NPC with no code change.
- `interaction_hotspot` composition elements with drag, resize, and milestone gating.

Kernel 75 should consume these, not extend them, unless it discovers a real gap.

---

## 2. Locked decisions carried forward from Kernel 74

These do not get re-litigated:

- **No Director GO inside the Player's own sequence.** Director control resumes only at the shared-Scene advance.
- **No Player-count denominators.** No "3 of 4 ready". Each Player finishes independently; the Directors+ list is names and statuses.
- **The handoff destination stays setting-neutral.** Ra may introduce Niava's Crown Bet; Victory must not assume the campaign goes there.
- **One object model.** Whatever Kernel 75 renders goes through the existing element/visibility filter.

---

## 3. Required scope

### 3.1 Finish the tutorial-handoff Scene

Replace placeholder art and copy. `frontend/assets/tutorial-handoff.png` is a generated placeholder; the Scene itself is `scenes.slug = 'tutorial-handoff'` at the `amurray-family` Location with a `map_backdrop` and one `index_card` in its Base-layer composition.

Requirements:
- Real backdrop art, authored through the normal asset path.
- Final copy replacing *"Tutorial complete. Your Character is equipped and ready. The Narrator will take it from here."*
- Must remain setting-neutral.

### 3.2 Ra's prose rewrite

Ra's six authored responses are seed text in migration `066`, each marked `-- canonical source content` or `-- Victory onboarding adaptation`. The operator flagged the register as too refined during the Kernel 74 walkthrough.

This is a **content** change, not a code change. Decide whether it ships as a new migration (preserving the append-forward canon rule) or as a dialogue-packet editing surface — see §4.1.

### 3.3 Tutorial-complete presentation

What the Player sees on arrival. At minimum:
- What they just did, in plain language.
- What their Character now has (inventory acquired from Kessa is already durable and queryable).
- That the Narrator takes it from here.

Reuse the Program Panel. Do not build a second modal system.

### 3.4 Progression acknowledgement

The narrowest useful version. Kernel 74 records five milestones; this should surface them as an accomplishment rather than a debug list.

**Explicitly not** XP, levels, or a rewards economy — those remain excluded.

### 3.5 Aftercare

Deferred from Kernel 74 with no design attached yet. Needs its own decision before implementation: what Aftercare *is* in Victory, who initiates it, whether it is private to the Player or shared with the table, and whether it is durable.

**This is the largest genuine unknown in Kernel 75 and should be specified before anything is built.**

### 3.6 Director-controlled continuation

The Director moves the table off the tutorial into the campaign's first real Scene. The mechanism exists (`SetCurrentScenePlacement` clears local projections and returns Players to the shared stage). What is missing is the Director-facing affordance: knowing *when* the table is ready.

The status list from §1 is the raw material. Consider a Stage Management panel showing it, with no denominator and no readiness gate — informational only, matching Kernel 74's treatment of the door-intention note.

---

## 4. Open questions for the operator

These change what gets built and should be answered before implementation:

### 4.1 Does authored NPC dialogue get an editing surface?

Kernel 73 deferred a merchant-packet editor; Kernel 74 deferred a dialogue-packet editor. Both are now seed-only, so **every wording change is a migration**. Ra's rewrite makes this concrete for the second time.

Options: (a) keep seed-only and accept a migration per rewrite; (b) build a bounded packet editor for `dialogue_packets`/`dialogue_topics`; (c) build one shared authoring surface covering both merchant and dialogue packets.

### 4.2 What is Aftercare?

See §3.5. No prior kernel defines it.

### 4.3 Should the tutorial be replayable?

Milestones are keyed per (user, Character, Show). A second Character replays it naturally. There is currently no way to reset a Character who has already finished. Is that needed?

### 4.4 Does First Theater get any of this?

Kernel 73/74 are Catharsis-only by capability flag. The Program Panel, dialogue packets, and projection machinery are all venue-agnostic.

---

## 5. Explicit exclusions (carried forward)

Do not build: Director GO inside the Player sequence; group synchronization or readiness denominators; general quest tracking; AI NPC dialogue; arbitrary dialogue graphs; workflow scripting; ratings; Character leveling; a rewards economy; Audience tickets; the Agent venue; a non-WebGL renderer fallback; a full grid editor; a broad renderer rewrite.

---

## 6. Standing debts Kernel 75 could close cheaply

Not required, but adjacent and small:

- **The nine `dice.test.js` failures** — tracked as a named exception in `alpha-gate.sh` and `roadmap.md` since Kernel 70A. Any change to the count in either direction fails the gate, so a fix must update the script and roadmap together.
- **`scripts/smoke/kernel74-tutorial-browser.js` has never completed a full run** against live Catharsis, because another Show holds a Session there. It aborts safely with a PRECONDITION message. Running it once end-to-end would convert Kernel 74's §16.21 from operator attestation to reproducible automated evidence.
- **Two `memberships` tables still coexist** (`location_memberships` and legacy `memberships`), unioned ad hoc by three packages instead of routing through `participation`. Standing since Kernel 71.
- **The `merchant` package holds the tutorial flow** because `ResolveEligibleContext` lives there. If a fourth interaction type appears, extracting a `participantinteractions` package becomes worth the call-site churn.

---

## 7. PASS standard (draft)

Kernel 75 passes only if:

1. the tutorial-handoff Scene shows final art and copy, not placeholders;
2. it remains setting-neutral and names no campaign destination;
3. Ra's prose is the operator's final text;
4. a Player arriving at the handoff is told what they did and what they have;
5. progression is acknowledged without introducing XP/levels/rewards;
6. Aftercare is specified before it is built, and matches that specification;
7. Directors+ can see who has finished, with no denominator and no readiness gate;
8. advancing the shared Scene still clears every local projection and returns Players to the shared stage;
9. no Director GO is introduced anywhere inside the Player's own sequence;
10. full alpha gate passes;
11. the operator walks the completed tutorial end to end in a browser.
