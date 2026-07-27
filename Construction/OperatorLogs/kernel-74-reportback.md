# Kernel 74 Reportback — Locked Door Intentions, Ra Guided Dialogue, and Participant-Local Tutorial Handoff

## 1. Status

**PASS. Deployed live 2026-07-26/27.** All 21 PASS-standard criteria in the spec are met, including §16.21 — the operator walked the tutorial in a real browser against live production and confirmed the flow, after five real defects found *by that walk* were fixed and redeployed.

The Player-controlled portion of the Locked Courtyard tutorial is closed: a Player goes Kessa → door → intention → Ra → Crown Bet → handoff with **no Director GO anywhere inside the sequence**.

## 2. Actual migration numbers and runner proof

Confirmed next-available against both `backend/migrations/` and the live `schema_migrations` ledger immediately before naming (`065` was the ceiling in both).

- `066_kernel74_tutorial_progress_and_dialogue.sql` — `participant_tutorial_progress`, `participant_freeform_submissions`, `dialogue_packets`/`dialogue_topics`/`dialogue_topic_prerequisites`, `participant_dialogue_topic_views`, `participant_local_projections`; extends the `participant_interactions` and `scene_stage_elements` CHECK constraints; adds `scene_stage_elements.width/height` and `stage_element_bindings.requires_milestone`; seeds Ra's packet (6 topics, 3 prerequisite edges) and the setting-neutral `tutorial-handoff` Scene.
- `067_kernel74_local_projection_capability.sql` — `participant_local_projection_enabled` venue flag (055/058/064 pattern), plus the matching entry in `access/kernel16_venue_bootstrap.go` for fresh installs.
- `068_kernel74_projection_per_character.sql` — **bug fix found in live play**, see §9.

**Live deploy logs**: `2 pending → pre-apply backup (victory_pre_migrate_20260726_202152_2pending.dump, 3.2 MB) → both applied (<260ms) → schema current`; second boot reported `schema current (68 migrations recorded)`. `068` deployed separately with its own backup (`victory_pre_migrate_20260727_043630`); ledger now at 69.

## 3. Pre-implementation audit (spec §3 deliverables)

1. **Kessa completion**: `participant_tutorial_progress`, keyed `(user, character, show, milestone_key)` with a CHECK-constrained enum of exactly five milestones. UNIQUE is the idempotency mechanism, not a read-then-write.
2. **Hotspot availability, server-side**: resolved in `world.loadCompositionRows` via a `milestoneGate`. A gated element is **omitted from the payload entirely** rather than shipped flagged-hidden, so there is nothing for a client to un-hide. Backstage roles bypass for authoring.
3. **Directors+ notes**: durable `messages` rows (`message_type='backstage_note'`), one per resolved recipient, read via `GET /api/backstage-notes` and a new Catharsis chat tab. Live push is per-recipient `BroadcastToSessionUser` — never session-wide, because the hub does no role filtering.
4. **Ra topic progress**: `participant_dialogue_topic_views`; unlock and completion are recomputed from rows on every response, never restored from client state.
5. **Local projection vs shared Scene**: `participant_local_projections` substitutes *which Scene's composition one viewer resolves*. `shows.current_show_scene_placement_id` is never written and is still reported verbatim in that Player's own snapshot — which is what makes "the shared Scene did not move" assertable from the client, not just the database.
6. **Projection clearing**: `ClearForSharedSceneAdvance`, in the single `shows/stage.go` write path so both the direct Director route and the Cue route get it. The rule needs no tutorial-sequence registry: a projection survives while the shared stage sits on its origin placement and clears the moment the Director moves anywhere else.
7. **Deliberately kept bounded**: CHECK-constrained milestone list (a sixth milestone is a migration, on purpose); fixed freeform config keys, not a form builder; acyclic-validated topic DAG, not a scripting engine; destination resolved from the packet's Scene slug, never client-supplied.

## 4. The §4 Kernel 73A interaction defect

**Already fixed in the tree before this kernel started.** `selectObject` → `syncSelectedActions()` → bound-action button (`runtime.js:2889-2923`) was present and working. What was missing was the test. Added a regression test against a **bound `scene_composition` token specifically** — not a warehouse token — asserting it normalizes to a selectable token node with `live: false` and carries its binding through to `source.data.binding`, which is the exact path the action bar reads.

## 5. Architecture: the one genuinely new idea

`participant_local_projections` is the only structurally new concept, and its boundary was the main design risk:

```
shows.current_show_scene_placement_id  → the shared, authoritative Scene for the table
participant_local_projections          → which Scene ONE viewer resolves instead
```

Critically this does **not** fork the element/visibility model. `cues/types.go:43-48` recorded that per-participant object visibility was deferred precisely because it would require a second show-scoped source feeding `world/snapshot.go`'s visibility derivation. This kernel does not do that: it substitutes the Scene whose composition is loaded, and every element inside then passes through the one existing filter unchanged. There is still exactly one object model.

## 6. Ra's packet — canonical versus adapted

Six topics, seeded in `066` with a `-- canonical source content` or `-- Victory onboarding adaptation` marker on **every** response. Source grounding: *Locked Courtyard* pp.13-14 (Ra), p.15 (Beyond the Door), p.32 (The Crown Bet).

Required for completion: `why-looking`, `crown-bet`. Optional: `why-locked`, `who-are-you`, `what-if-succeed`, `beyond-the-door`. Dependency edges: `why-looking → crown-bet → {what-if-succeed, beyond-the-door}`.

The "Nature called" line is marked adaptation, not quotation. The lock reveal (`closing_narration`) describes a recessed iron plate — consistent with, never contradicting, the door's "no obvious lock or keyhole" opening description.

**Operator note**: Ra's prose was flagged as too refined during the walkthrough and is the operator's to rewrite. It is all seed text; no code reads it.

## 7. Evidence

- **Alpha gate**: `scripts/test/alpha-gate.sh` — all automated steps PASS, confirmed across multiple full runs after each fix. The tracked `dice.test.js` exception count is unchanged at 9; no gate changes were needed.
- **Backend**: `kernel74_tutorial_dbtest_test.go` (3 tests) drives the entire PASS standard against the real migrated schema — Kessa completion, per-Character door reveal, Character-switch isolation in **both** directions, verbatim submission storage, empty/oversized rejection, retry idempotency for both the intention and the note, note visibility by role (director/producer see it; submitting Player, other Player, and Audience do not), topic prerequisites, required-vs-optional, direct-Leave bypass refusal, projection scoped to the caller, shared Scene unchanged, refresh persistence, and shared-advance clearing. Plus anonymous-refusal and milestone-validation tests.
- **Frontend**: `tests/stage-runtime/kernel74-tutorial.test.js` (13 tests) — the 73A regression, hotspot node kind and sizing, `map_backdrop` still producing no node, the reveal gate, local projection on `session`, socket routing, and four regression tests for the live-play defects in §9.
- **Browser**: the operator walked the flow live (Kessa → completion → door → intention → Ra → topics → Leave → handoff → Character switch). §16.21 satisfied by operator attestation.

## 8. Security proofs (spec §13)

Anonymous → `not_authenticated` on all five entry points; audience and non-roster refused by `ResolveEligibleContext` (no backstage branch exists); archived/unowned Character refused; forged client progress cannot reveal the door; locked topics and premature Leave refused server-side; the Player never names a projection destination; notes never reach Player or Audience in either snapshot or history rehydration; Player text is escaped at render (`textContent`, never `innerHTML`) in both the Program Panel and the backstage tab.

## 9. Real defects found and fixed — five of them, all by the operator's live walk

None of these were discoverable by reading the spec or by the test suite as originally written. Recording them in full because each represents a class of mistake worth not repeating.

**(a) The reveal gate protected discovery only — a genuine authority hole.**
The snapshot omitted the door for Players without `kessa_intro_completed`, but a Player who learned the interaction id another way could POST straight to it. S13 requires both "client-forged progress does not reveal the door" *and* "Player cannot unlock the door before Kessa completion." Fixed with `merchant.requireBindingMilestone`, re-checking the same milestone on **invocation** for open/submit/dialogue. **Lesson: an "omit from payload" gate is never sufficient on its own.**

**(b) An unbound hotspot rendered a live, pointer-capturing box.**
Left unbound, the door hotspot still drew a full-size interactive region. On the live Courtyard it sat over Kessa's token, swallowed every click meant for her, and answered "cannot be used right now." Fixed: unbound or disabled hotspots are fully inert — no pointer events, invisible to non-backstage viewers, faint authoring outline for Directors. **A control that cannot act must not be able to block.**

**(c) Hotspots rendered above tokens.**
Even correctly bound, a hotspot overlapping a token won the click. Hotspots now render at z-index 16, below public tokens at 24 — a hotspot is a region over *map art*, and map art belongs behind the people standing on it.

**(d) A hotspot's authored position was never read.**
`geometry.js` converted normalized 0-1 composition coordinates to pixels only for `kind === "token"`. A hotspot fell through to a generic mid-stage fallback, so its stored position was ignored entirely — `y: 0.35` and `y: 0.075` rendered identically, on top of Kessa, and re-authoring appeared to do nothing. Fixed by extending that branch to hotspots. **This one masked (b) and (c): all three presented as "the box is in the wrong place."**

**(e) The local projection was keyed on `(user, show)`, ignoring Character.**
A Player who finished the tutorial with one Character and switched stayed stranded on the handoff map with a Character who had never met Kessa. Every *other* Kernel 74 table was correctly Character-keyed; the projection carried the column and didn't use it. Fixed by migration `068` widening the partial unique index and threading the selected Character through `LoadActiveForViewer`.

**My tests missed (e) specifically** because the Character-switch assertion I wrote only covered the door hotspot. I checked the thing I had just built a gate for and did not ask the same question of the projection. The regression test now asserts both directions.

## 10. Deviations from the kernel

- **A sixth change the spec did not ask for**, made on the operator's explicit approval during the walkthrough: `ActivateCharacterCard` now writes through to `show_run_roster_members.character_card_id`. Victory had two independent "current Character" values — the Greenroom picker (`active_user_characters`) and the Show-Run roster selection (Kernel 71's canonical one, which every Kernel 74 surface keys off). Switching in Greenroom left the roster row untouched, so the tutorial state did not follow. Scoped narrowly: `player` roster rows only, non-archived Show Runs only, same Location as the Character.
- **Kessa's two exit buttons collapsed to one.** The first pass added "Leave Kessa's Stall" beside the packet's existing "Leave the Shop" — the distinction (dismiss vs. record) is an implementation detail no Player should parse. One button now records completion and closes; the `×` and Esc remain a pure dismissal.
- **Clicking the door as a Director** now says it is a Player interaction and points at Preview as Player, rather than returning a generic failure. Participant interactions are Player-only by construction; the previous message read like a bug.

## 11. Known limitations

- **Placeholder art**: `frontend/assets/ra.png` and `frontend/assets/tutorial-handoff.png` are generated placeholders, swappable as files with no code change. Kernel 75 finalizes.
- **Ra's prose** is authored seed text flagged for operator rewrite (§6).
- **The door hotspot's default position** (x 0.503, y 0.124, 0.06 × 0.075) is measured off the current `courtyard.png` and is a *starting* position — a Director drags/resizes it in Scene Setup, which now actually works.
- **The Ra Program shows the opening line on resume**, not the last response read. Topic seen-state, unlock state, and earned Leave all restore fully; replaying the last line would mean storing transcript state this kernel does not need.
- **`scripts/smoke/kernel74-tutorial-browser.js` cannot run against live Catharsis** while another Show holds an active Session there — it preflights and aborts with a clear PRECONDITION rather than ending a Session it did not open. Set `K74_VENUE_SLUG` to a free venue, or end the other Session first.
- **The `merchant` package name is now narrower than its contents.** The tutorial flow lives there because `ResolveEligibleContext` does; a rename would touch every call site. Documented in `tutorial_flow.go`'s header rather than left implicit.

## 12. Live deployment status

Deployed live 2026-07-26/27. Migrations `066`–`068` applied with pre-apply backups; ledger at 69; second-boot idempotency confirmed. Backend rebuilt/restarted. **Uncommitted**, matching project practice pending review.

Live production side effects: throwaway `k74-browser-*` rows from smoke iterations were fully cleaned up (verified zero remaining); 39 `k74_*` throwaway accounts remain with no elevated privileges, per Kernel 62/63/65 precedent. The operator's own pre-existing Catharsis Session was never touched.

## 13. Files changed / created

- **New**: `backend/migrations/066`–`068`; `backend/internal/tutorial/`, `backend/internal/dialogue/`, `backend/internal/projection/` (new packages); `backend/internal/merchant/{tutorial_flow,http_tutorial,kernel74_tutorial_dbtest_test}.go`; `backend/internal/messages/backstage_notes.go`; `backend/internal/world/kernel74_local_projection.go`; `frontend/assets/{ra,tutorial-handoff}.png`; `tests/stage-runtime/kernel74-tutorial.test.js`; `scripts/smoke/kernel74-tutorial-browser.js`; this reportback.
- **Modified**: `backend/cmd/victory/main.go` (routes); `backend/internal/access/kernel16_venue_bootstrap.go` (fresh-install flag); `backend/internal/merchant/{interactions,types,http,prepare}.go`; `backend/internal/scenes/{composition,composition_types,http_composition}.go` (hotspot kind, width/height, `requires_milestone`); `backend/internal/shows/stage.go` (projection clearing); `backend/internal/world/snapshot.go`; `backend/internal/characters/characters.go` (roster write-through); `frontend/lib/stage-runtime/{runtime,scene-nodes,state,geometry,session-sync,socket,participant-interactions}.js`; `frontend/venues/catharsis/index.html` (Backstage tab); `frontend/venues/show-runs/show.html` (hotspot authoring, resize, milestone gate).

## 14. Recommended Kernel 75 scope

See `Construction/Kernels/kernel-75-tutorial-completion-aftercare-continuation-v0.1.md` for the drafted spec.
