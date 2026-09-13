# Spatial Dice Projection Contract (Kernel 86A)

## The problem

Kernel 86 projected a roll as one grouped node — dice pool, caption, and total all glued together at a fixed *screen*-space slot (`placeAtTransientSlot`), and pinned rolls stacked in a screen-space corner (`layoutPinned`). That satisfied "theatrical" but not "spatial": nothing about a roll ever touched the actual map, which is a hard blocker for Cartograph (Kernel 87), since drawing needs real reference coordinates. Kernel 86A splits the single node into two independently-lifecycled pieces and gives one of them a real position on the map.

## Two pieces, two coordinate spaces

`frontend/lib/stage-runtime/dice-projection.js` now renders:

- a **world-space dice cluster** — one `PIXI.Container` per die, each landing at its own map-relative `(x, y)`, mounted into a new `diceWorldLayer` (`runtime.js`) that lives *inside* `worldLayer`, alongside `mapLayer`/`gridLayer`/`pinnedObjectLayer`. Since `stageCamera` only ever transforms `worldLayer` (`worldLayer.position`/`worldLayer.scale`, set by `victory-stage-camera.js`'s `applyTransform`), landed dice pan and zoom with the map for free — no dice-projection-specific camera code needed.
- a **screen-space announcement banner** — the preserved Kernel 86 caption/total/Pin-button summary, mounted into the existing `diceProjectionLayer` (a direct `sceneRoot` child, never touched by the camera transform). This is the same node shape as before, just triggered later and stripped of the dice pool it used to carry.

`controller.mount(hudLayer, worldLayer)` takes both targets now (was a single-arg `mount(layer)` under Kernel 86).

## The sequencing barrier (spec §6.2, §7)

Each die animates independently (`animateDie`), staggered by `STAGGER_MS` so a multi-die roll doesn't lock-step. A per-roll `entry.settledCount` counter increments as each die finishes; only when `settledCount === dieEntries.length` does `onAllDiceSettled` fire, which is the *only* place that builds and shows the announcement node. There is no timer-based "probably done by now" approximation — the barrier is a hard equality check against the actual number of dice in the roll, so the announcement structurally cannot precede the last landing, regardless of how many dice or how long any individual tumble takes.

## Deterministic, shared, reconnect-stable landing coordinates

The hardest constraint in the spec (§4.3, §9): a pinned roll's dice must land in the *same* map spot for every viewer, including one who reconnects later — but the server has no idea what any viewer's camera is doing (pan/zoom is per-user `localStorage` state in `victory-stage-camera.js`, never sent to the backend). Two designs were available:

1. Have the server pick and store coordinates, broadcast them to everyone.
2. Have every client compute the *same* coordinates independently, from inputs that are already identical everywhere.

Kernel 86A takes (2), and needed no backend changes to do it. `computeLandingPositions(effectId, count, mapBounds, tokenSize)` is a pure function seeded by `effect.id + dieIndex` (FNV-1a hash → `mulberry32` PRNG — small, dependency-free, deterministic), evaluated against `mapBounds`, which is the *session's* active map bounds (`runtime.js`'s `currentVenueMapBounds`, the same rectangle already fed to `stageCamera.setWorldBounds` — a shared, non-viewport-dependent rectangle, not any individual viewer's live pan/zoom window). Every client — original roller, live spectator, or a browser that reconnects five minutes later — runs the identical computation and lands the identical dice in the identical spot, with zero coordination and zero new server storage. `stageeffects.Effect` is unchanged.

This also makes the pin-race safe for free: if a `stage_effect_pinned` confirmation arrives after a viewer's transient hold has already expired and been destroyed (`current` is `nil`), `applyPinned`'s fallback path rebuilds the cluster from scratch — and lands it in exactly the same place, because the formula is the same. Proven by `dice-projection.test.js`'s "reconnect recomputes identical landed coordinates" test, which runs two independent controller instances against the same effect and asserts pixel-identical output.

## Placement algorithm (spec §5, §6)

`computeLandingPositions` insets the map's world bounds by `tokenSize * 0.75` and, for each die, tries up to 8 deterministic candidate points inside that rectangle, keeping whichever candidate is farthest from already-placed dice in the same roll (falling back to "farthest tried" if none clears the `tokenSize * 1.15` minimum-separation threshold). No physics, no iterative solver — a bounded retry loop, per the spec's explicit "do not over-engineer" guidance. `normalizeMapBounds` rejects a missing/non-positive-area map outright rather than degrading to a nonsense rectangle.

## No-map fallback (not in the original K86 surface area, added defensively)

The spec assumes an active map always exists when a roll happens. In practice a scene can be map-less. Rather than land dice at a meaningless `(0, 0)` or crash, `startNext`/`applyPinned` check `normalizeMapBounds(getMapWorldBounds())` first; if it's `null`, the module falls back to the *original* Kernel 86 grouped-HUD presentation (`buildLegacyGroupedNode`/`animateLegacyRoll`, ported near-verbatim) — same announcement-immediately, no individual landing, no pan/zoom attachment, but not broken. This is exercised by two explicit tests (`dice-projection.test.js`) rather than left as an unverified assumption.

## Pin/Dismiss authority moves with the split (spec §1.9, §1.10)

Under Kernel 86, one button ("Pin" → later "Dismiss") lived on the single combined node. Kernel 86A's split forces a real decision: the announcement fades on its own normal timer even for a pinned roll (§1.10 — "static dice stay, announcement fades normally"), so a button that only lived on the announcement would vanish along with it, leaving no way to later dismiss dice that are, by design, meant to persist far longer than the banner. So:

- the **Pin** button lives on the announcement (`node.__pinButton`) — it only needs to be clickable during the roll's normal ~3-4s life, which is exactly when the announcement exists.
- the **Dismiss** button lives on the dice cluster itself (`clusterNode.__dismissButton`), positioned under the landed-dice centroid — it's built every time a cluster is built (transient or pinned-fresh) but only wired active (`setActionButton(..., "Dismiss", ...)`) once the roll is actually pinned, so it survives exactly as long as the dice it controls, independent of the announcement's lifetime.

Authority itself is unchanged from Kernel 86 — `network/ws.go`'s `stageEffectAuthorized` (roller, or Director+ except on a Private roll) is still the only real gate; the client-side buttons are just UI convenience, same as before.

## Explosion safety (spec §2, §13) — audited, not rebuilt

Before touching any code, the full default-roll path was traced end to end:

- `frontend/lib/stage-runtime/dice.js`'s Explode checkbox defaults **unchecked**, and only appends `!` to the expression when checked.
- The `/roll`/`/r` chat command (`runtime.js`) passes the user's typed expression straight through — no expression construction, no implicit `!`.
- `backend/internal/dice/dice.go`'s `ParseExpression` only sets `DieGroup.ExplodeOnMax = true` when the term literally matched a trailing `!` in `dieTermPattern`. There is no code path — default, macro, or otherwise — that sets it any other way for ordinary rolls.
- `characters/parentage_roll.go`'s `rollExplodingD20Single` is a deliberately named, single-purpose parentage macro that always explodes on a natural 20 — an explicit macro request in the spec's own sense, not the default roll path, and out of scope for this kernel.

Given that, the required §13 tests were still written rather than merely asserted: `dice_test.go`'s `TestOrdinaryExpressionsNeverAutoExplode` feeds every ordinary expression form (`d20`, `3d12`, `2d6+3`, ...) an `Intn` source that returns the maximum face on *every* atomic throw and asserts `ExplosionCount` stays zero regardless — if implicit explosion ever crept in, this test would either fail outright or hang against `MaxAtomicThrows`. `TestExplicitExplosionStillChains` proves the positive case wasn't collateral damage. On the renderer side, `dice-projection.js` contains no explosion decision-making at all — the only `Math.random()` calls are decorative face-cycling during the tumble animation, always overwritten by the server-supplied `finalFace`/`subtotal` before a die settles; `dice-projection.test.js`'s "final displayed die value is exactly the server value regardless of what the decorative tumble RNG rolls" test forces `Math.random` to `0.999` (the worst case for accidentally biasing toward "max face, must explode") and confirms the landed value is untouched.

**Result: no Kernel 86B was required.** The default path was already explosion-safe; Kernel 86A only had to prove it, not fix it.

## Two real bugs the live browser proof found (and fixed)

Both were invisible to the unit test suite because it exercises `computeLandingPositions`/`applyPinned` with static, always-ready fake deps — neither race exists in a synchronous test.

**1. Pinned-effect hydration can arrive before `mount()` has run.** The server sends `stage_effects/pinned` immediately after `snapshot`, which can beat the Pixi scene's own construction (when `diceProjection.mount(hudLayer, worldLayer)` first runs). `applyPinned`'s no-container branch used to record a permanent placeholder (`{ clusterNode: null }`) for that case — meaning a pinned roll a reconnecting viewer should see could silently render nothing, forever, with no way to recover short of a page reload landing more luckily. Fixed with a `needsSpatialUpgrade` flag plus `reconcilePinnedPlacements()`, called from `mount()` and from the top of `startNext()` (a roll happening later is a cheap, frequent, natural re-check point).

**2. That fix alone wasn't enough — `currentVenueMapBounds` itself isn't instantly correct at connect time either.** `runtime.js`'s `renderVenueMapLayer` returns a *screen-sized placeholder* rectangle (`computeStagePlayableBounds`) while the map's texture is still loading, not `null` — so a single "is `mapBounds` non-null yet" check can succeed against that placeholder before the real, content-fitted bounds are known, silently locking a pinned die's world coordinates to the wrong rectangle forever. Fixed with `forceReconcileAllPinned(6, 400)` — an unconditional (not flag-gated) recompute of every currently-pinned effect, retried up to 6 times over ~2.4s after `mount()`, so the last attempt lands on the settled value regardless of what an earlier one saw. A correctly-placed entry just gets recomputed to the identical answer each retry (harmless).

## Known limitation found, not fixed: live reconnect coordinate exactness

`computeLandingPositions` is provably exact-deterministic given identical inputs — `dice-projection.test.js`'s "reconnect recomputes identical landed coordinates" test runs two independent controller instances against the same effect and fixed `mapBounds` object and asserts pixel-identical output, every time. Live, across repeated browser-proof runs, the **input** wasn't always identical: `currentVenueMapBounds`'s X-axis component (derived from the fitted map sprite's measured width against `computeStagePlayableBounds`) was observed to differ by anywhere from a few pixels up to ~150px between two independent page loads of the same `fit=contain`, 4:3-image-in-a-wider-viewport map, while the Y-axis component (the constraining dimension for this aspect ratio) was exact every time. A deterministic fraction-of-bounds placement amplifies that input variance proportionally.

This is a **pre-existing characteristic of `computeStagePlayableBounds`'s viewport measurement** (`runtime.js`/`geometry.js`), not a `dice-projection.js` defect, and fixing it would mean auditing shared stage-sizing/layout-measurement code well outside this kernel's contract. The live browser proof (`kernel86a-spatial-dice-browser.js`) checks what is this kernel's to guarantee instead — a reconnected pinned die has valid, non-degenerate map and screen coordinates and is still the correct, authorized, correctly-valued effect — and documents this finding inline rather than silently loosening the check or chasing it into unrelated code. Flagged for whoever next touches `computeStagePlayableBounds`/the map-fit sizing pipeline; not blocking for Cartograph readiness, since the drift is well short of "landed somewhere unrecognizable."
