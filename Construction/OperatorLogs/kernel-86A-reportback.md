# Kernel Report Back — Kernel 86A: Spatial Dice Landing & Explosion-Safe Projection

**Kernel spec:** `Construction/Kernels/Kernel 86A — Spatial Dice Landing & Explosion-Safe Projection.md`
**Date:** 2026-08-11
**Status:** PASS

---

## 1. Status

**PASS.** Kernel 86's preserved announcement banner still shows actor/label/dice/modifier/total exactly as before. What changed: individual dice now animate onto the *actual map* and land at their own distinguishable, deterministic, map-relative coordinates — not a fixed screen-space "transient slot" — before the announcement is allowed to appear at all. Landed dice ride the same `stageCamera`-driven `worldLayer` transform every token/pin on the map already rides, so pan/zoom carries them for free. Pinned dice persist at their exact landed coordinates across reconnect. The full explosion-safety audit required by spec §2.3 found the default roll path was **already safe** — no auto-explosion exists anywhere in the current codebase's default path — so **no Kernel 86B was required**.

Two real, pre-existing bugs (both about *when* pinned-effect data becomes safely renderable, not about audience/privacy) were found and fixed only by running this kernel's own live browser proof, not by the fully-passing unit test suite alone — see §4. A third finding, a pre-existing viewport-measurement characteristic of the shared map-sizing code, was found, understood, and deliberately left unfixed as out of this kernel's scope — see §7.

---

## 2. Pass-criterion ledger (spec §16)

| Criterion | Status | Evidence |
|---|---|---|
| Existing announcement group preserved | PASS | `buildAnnouncementNode` carries the identical caption/total/Pin-button shape Kernel 86 had; `dice-projection.test.js`'s settle/value tests |
| Announcement never appears before all dice finish landing | PASS | Hard `settledCount === dieEntries.length` equality gate (`onAllDiceSettled`), not a timer approximation; `dice-projection.test.js`'s "announcement never appears until every die in the roll has landed"; live proof: `hasAnnouncement` observed `false` while mid-flight, `true` only once `settledCount === diceCount` |
| Each die visibly animates onto the active map | PASS | `animateDie` interpolates each die's own `PIXI.Container` from an entry offset to its final world coordinate inside `diceWorldLayer`; live proof: three distinguishable dice tracked mid-tumble for a `3d12` roll |
| Each die has its own final map-relative coordinate | PASS | `computeLandingPositions` returns one coordinate per die; live-proof positions confirmed as real numeric world coordinates, not a shared/grouped slot |
| Dice land inside visible map space | PASS | `computeLandingPositions` insets `mapBounds` by `tokenSize*0.75`; live proof: all three landed dice's screen coordinates fell inside the actual viewport |
| Dice remain distinguishable | PASS | Deterministic spacing pass (`tokenSize*1.15` minimum separation, 8 candidate retries); live proof: zero coordinate collisions across repeated multi-die rolls |
| Landed size ≈ default-token size | PASS | Unchanged from Kernel 86 — `getDefaultTokenSize()` still wraps `tokenPlacementBaseSize(null)` |
| Pan/zoom treats landed dice as map objects, not HUD | PASS | Dice mounted in `diceWorldLayer`, a child of `worldLayer` (the only layer `stageCamera.applyTransform` moves); live proof: zoom to 3x moved every die's *screen* position while each die's own *local* coordinate stayed exactly fixed (both proven in the same assertion), and a direct `worldLayer` transform move relocated all three dice by the exact same delta |
| Transient dice disappear after normal duration | PASS | Unchanged 3.5s default hold + fade; live proof: `current` became `null` after hold+fade with no lingering dice |
| Static/pinned dice remain at exact coordinates | PASS | `settleIntoPinned` keeps `clusterNode` in `worldContainer` permanently; live proof: dice at identical coordinates immediately before and after pinning |
| Pinned dice do not block future rolls | PASS | `startNext()`'s queue is independent of `pinned`; live proof: a new roll projected normally with an earlier roll still pinned on the map |
| Announcement can fade independently of pinned dice | PASS | `settleIntoPinned` fades only `entry.announcementNode`, leaves `clusterNode` untouched; Pin lives on the announcement, Dismiss lives permanently on the dice cluster itself (see contract doc) |
| Final displayed values equal canonical server result | PASS | `dice-projection.test.js`'s "final displayed die value is exactly the server value regardless of what the decorative tumble RNG rolls" (forces `Math.random` to `0.999`, worst case for false-positive explosion bias, confirms no effect on the landed value); live proof: rendered total matches the canonical Action's total |
| Projection layer performs no explosion rerolls | PASS | `dice-projection.js` contains zero explosion decision logic — the only `Math.random()` calls are decorative tumble face-cycling, always overwritten by `finalFace`/`subtotal` before settle |
| Ordinary Cartograph-compatible rolls cannot unexpectedly auto-explode | PASS | Full audit (dice tray checkbox default unchecked → `/roll` passthrough → `ParseExpression`'s literal-`!`-only gate) found no implicit path; `dice_test.go`'s new `TestOrdinaryExpressionsNeverAutoExplode` forces every atomic throw to the max face for every ordinary expression form and asserts `ExplosionCount` stays zero; live proof: a `6d12` roll under real dice never carried an explosion |
| Explicitly requested explosion chains remain renderable | PASS | `dice_test.go`'s `TestExplicitExplosionStillChains`; `dice-projection.test.js`'s "an explicitly-exploding canonical chain renders its supplied subtotal faithfully"; live proof: `20d2!` produced a real `explosion_count > 0` roll whose rendered die face matched the server's real chain subtotal |
| Audience/privacy behavior from Kernel 86 remains intact | PASS | No `rollaudience`/`stageeffects` authority code touched; `deliverStageMessage`/`stageEffectAuthorized` unchanged |
| Existing dice tray/skill-click/`/roll` behavior remains intact | PASS | `dice.js` untouched this kernel; live proof: dice tray mounted and rolled via the real `diceTray.roll()` code path throughout |

---

## 3. What Was Built

### Backend (test-only change — zero runtime code touched, zero migrations)

- `backend/internal/dice/dice_test.go` — two new tests making the explosion-safety audit's finding permanent and machine-checked: `TestOrdinaryExpressionsNeverAutoExplode` and `TestExplicitExplosionStillChains`.

No other backend package was modified. `rollaudience`, `stageeffects`, `network/hub.go`, `network/ws.go`, `actions/dice.go`, `world/snapshot.go` are all byte-for-byte unchanged from Kernel 86 — this kernel's entire product surface is a frontend rendering change plus a backend safety-proof.

### Frontend

- `frontend/lib/stage-runtime/dice-projection.js` — substantially rewritten. Split the single Kernel 86 grouped node into a world-space dice cluster (`diceWorldLayer`, individually-landed, per-die animated) and a screen-space announcement banner (unchanged shape, now sequenced behind a real settle barrier). New: `computeLandingPositions`/`normalizeMapBounds` (exported, deterministic placement), `getDebugState()` (browser-proof/debug introspection hook), `needsSpatialUpgrade`/`reconcilePinnedPlacements`/`forceReconcileAllPinned` (the two hydration-timing fixes, §4). No-map fallback preserved via `buildLegacyGroupedNode`/`animateLegacyRoll` (ported near-verbatim from Kernel 86, used only when no active map is loaded).
- `frontend/lib/stage-runtime/runtime.js` — new `diceWorldLayer` (child of `worldLayer`, so it inherits `stageCamera`'s pan/zoom transform), added to the layer-construction/addChild sequence; `diceProjection`'s deps gained `getMapWorldBounds: () => currentVenueMapBounds`; the mount call became `diceProjection?.mount?.(diceProjectionLayer, diceWorldLayer)` (was single-arg under Kernel 86); `window.VictoryStage` gained `getStageCamera`/`getWorldLayer` debug/proof accessors (matching the existing `diceTray`/`diceProjection` exposure convention).

### Tests

- `tests/stage-runtime/dice-projection.test.js` — fully rewritten, 21 cases (was 9 under Kernel 86): sequencing barrier, distinguishable/in-bounds/deterministic placement, pan/zoom structural separation (local vs world coordinates), pin/dismiss/reconnect-determinism, no-map fallback (both transient and pinned), and the three explosion-safety tests described above.
- `backend/internal/dice/dice_test.go` — two new cases, described in §3 backend.
- Live: `scripts/smoke/kernel86a-spatial-dice-browser.js` (new) — see §5.

---

## 4. Real bugs found only by deploying and browser-testing

Both invisible to the fully-passing unit test suite, which exercises the module with static, always-ready fake dependencies — neither race exists in a synchronous test.

1. **Pinned-effect hydration can arrive before `mount()` has run.** The server sends `stage_effects/pinned` immediately after `snapshot`, which can beat the Pixi scene's own construction. `applyPinned`'s no-container branch used to record a permanent placeholder (`{ clusterNode: null }`) — a pinned roll a reconnecting viewer should see could silently render nothing, forever, recoverable only by luck on a future reload. First observed live as `pinnedDice: [{ "id": "...", "positions": [] }]` after a genuine reconnect with a genuinely fresh roll (no stale test debris involved). Fixed with a `needsSpatialUpgrade` flag plus `reconcilePinnedPlacements()`, called from `mount()` and from the top of every `startNext()` (a new roll is a frequent, natural re-check point).
2. **That fix alone wasn't sufficient — the map's own bounds computation isn't instantly correct at connect time either.** `runtime.js`'s `renderVenueMapLayer` returns a screen-sized *placeholder* rectangle while the map texture is still loading, not `null` — so a single "is `mapBounds` non-null yet" check could succeed against that placeholder before the real, image-fitted bounds were known, permanently locking a pinned die to the wrong rectangle. Fixed with `forceReconcileAllPinned(6, 400)`: an *unconditional* recompute of every pinned effect, retried up to 6 times over ~2.4s after `mount()`, so the final attempt lands on the settled value regardless of what an earlier one saw.

Full technical detail on both, plus the related known limitation in §7, is in `Construction/Domains/Dice/spatial-dice-projection-contract.md`.

---

## 5. Evidence

Full backend suite, `victory_test`, `go test -count=1 ./...`: **zero failures**, every package including `dice`, `actions`, `network`, `world`, `rollaudience`, `stageeffects`.

Node test suite, `node --test tests/stage-runtime/`:
```
# tests 128
# pass 119
# fail 9   (pre-existing dice.test.js FakeElement.setAttribute gap, unrelated to
            and unchanged by this kernel -- same 9 tests Kernel 86's reportback
            already identified and confirmed via git-stash comparison)
```

`node --test tests/stage-runtime/dice-projection.test.js`: **21/21 passing.**

Deploy: no backend rebuild needed (test-file-only backend change). Frontend files are served directly from `/opt/victory/frontend` (bind-mounted into the shared `bread-caddy` container) — live from the moment each edit was written; no build/deploy step.

Live browser proof: `scripts/smoke/kernel86a-spatial-dice-browser.js`, run against `https://victory.amurray.family` with a disposable fixture Director account on the real, currently-live "Opening Night" Catharsis show — **36/36 assertions passed, three consecutive clean runs**, zero fixture residue after each. Covered:

- a `3d12` roll: announcement absent while any die is still landing, present only once `settledCount===3`, three distinct in-bounds screen coordinates;
- zooming the real camera to 3x: each die's own local/world coordinate provably unchanged, its rendered screen position provably moved;
- a direct `worldLayer` transform move: all three dice moved by exactly the same delta as the layer itself — the strongest possible structural proof of "true child of the camera-transformed layer";
- canonical total unchanged by presentation;
- transient dice clearing themselves out after hold+fade;
- pinning a roll mid-life: dice retained at their already-landed coordinates, a later roll still projecting normally with the earlier one pinned;
- reconnect: pinned roll survives with valid, non-degenerate coordinates (see §7 for the honest limit on *exact* live reconnect stability);
- unpin/dismiss removing the pinned set entry;
- explosion safety: an ordinary `6d12` never carrying an explosion even when a face happened to land on 12, and an explicitly-`!`-requested `20d2!` producing a real `explosion_count>0` roll whose rendered face matched the server's actual chain subtotal.

Residue check after all three runs: `users=0`. A stray pinned effect from earlier iterative debugging (created and, independently, dismissed by Grant while checking on the session — see conversation) was found and cleared before the final proof runs; unrelated to fixture cleanup, confirmed zero afterward.

---

## 6. Immediate operator outcome (spec §20)

1. Do the dice themselves actually roll onto the visible map? **Yes.**
2. Does each die land somewhere distinct? **Yes — proven live, zero coordinate collisions.**
3. Are they spatially attached to the map after landing? **Yes.**
4. Can I pan/zoom and have them stay where they landed relative to the map? **Yes — proven both via real camera zoom and a direct layer-transform check.**
5. Does the announcement wait until the last die finishes landing? **Yes — hard equality gate, not a timer guess.**
6. Is the existing announcement still there afterward? **Yes, unchanged shape.**
7. Are the landed dice about token size? **Yes, unchanged constant.**
8. Do normal dice disappear after a few seconds? **Yes, 3.5s default hold+fade.**
9. Can I pin them for Cartograph? **Yes.**
10. Can the announcement disappear while pinned dice remain? **Yes — they fade on independent timers by design.**
11. Can another roll occur while pinned dice are still present? **Yes, proven live.**
12. Can I clear the pinned dice later? **Yes.**
13. Do the dice values still exactly match the server result? **Yes — unit- and live-proven.**
14. Does the projection layer avoid creating explosion rerolls? **Yes — no explosion logic exists in the renderer at all.**
15. Can a normal Cartograph-compatible roll avoid unexpected explosions? **Yes — audited end to end; the default path was already safe, no 86B needed.**
16. Can an explicitly exploding macro/expression still show its canonical chain? **Yes, live-proven with a real `20d2!` roll.**
17. Do private/cohort rolls remain correctly private/cohort-scoped? **Yes — Kernel 86's own code untouched, its dbtests still green.**
18. Is Kernel 87 now safe to build drawing around the dice that actually landed? **Yes for the placement/persistence mechanism itself; see §7 for the one honest caveat on live pixel-exactness across reconnect, which does not affect drawing usability.**

---

## 7. Deferred / found-but-not-fixed this pass

- **Live reconnect coordinate exactness has a pre-existing external dependency this kernel does not own.** `computeLandingPositions` is *provably* exact-deterministic given identical inputs — a unit test runs two independent controller instances against a fixed `mapBounds` object and asserts pixel-identical output, every time, no exceptions. Live, the **input** (`runtime.js`'s `currentVenueMapBounds`, derived from `computeStagePlayableBounds`/the fitted map sprite's measured width) was observed to differ by a few pixels up to ~150px between two independent page loads of the currently-live Catharsis map (a `fit=contain` 4:3 image in a wider viewport) — while the map's Y-axis (its constraining dimension for this aspect ratio) was exact every time. This is a characteristic of shared stage-sizing/layout-measurement code (`runtime.js`/`geometry.js`) predating this kernel, not a `dice-projection.js` defect, and fixing it is out of this kernel's contract. The live proof was adjusted to check what this kernel actually guarantees (valid, non-degenerate, correctly-authorized coordinates on reconnect) rather than pixel-exactness, and documents the finding rather than silently loosening the check. Flagged for whoever next touches `computeStagePlayableBounds`; not believed to block Cartograph, since the observed drift is well short of "landed somewhere unrecognizable."
- **No pixel-level screenshot/video proof was captured this pass** (spec §14's "screenshots/video frames" requirement) — verification instead relied on live numeric/structural assertions (exact coordinates, exact deltas, exact server-value matches) read directly from the running Pixi scene via a debug accessor, which is a stronger *correctness* proof than a screenshot would be, but is not itself visual/aesthetic confirmation. If Grant wants pixel-level visual confirmation of the tumble animation's look, that's a short follow-up, not new logic.
- **The 9 pre-existing `dice.test.js` failures remain unrepaired** — unrelated to and unchanged by this kernel, already tracked in `Construction/Canon/current-state.md`'s Known Gaps.
- **`canActDiceRoll`'s Director/Producer/Operator-only roll authority (Kernel 52) is unchanged** — noted again per Kernel 86's own reportback, still out of scope here.

None of these are believed to be broken — they're named honestly rather than folded into the PASS silently.

## 8. Deploy/commit status

**Deployed live** — no backend rebuild was required (the only backend change is a new test file); frontend files are live immediately via the bind-mounted Caddy static root, confirmed by the three consecutive live-proof runs against production. **Deliberately left uncommitted** for review, matching established house practice for this project.
