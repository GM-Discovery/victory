# Kernel 48 — Personal Stage Camera and Card Pinning

**Provenance:** RECONSTRUCTED — HIGH CONFIDENCE
**Original date/window:** 2026-06-21
**Implementation status:** IMPLEMENTED — status not recorded (no dedicated reportback survives; immediately followed same-day by the Kernel 48.1 fix pass, which does have a full reportback and treats Kernel 48 as its established baseline)
**Evidence sources:** `Construction/OperatorLogs/operator-log.md:592-609` ("Kernel 48 Personal Stage Camera + Card Pinning v1"); corroborated by `Construction/OperatorLogs/kernel-48.1-reportback.md`, which explicitly repairs a world-vs-overlay placement bug in the exact model this entry describes.

## Reconstructed purpose

Give First Theater a personal, browser-local camera (pan/zoom) and let index cards be pinned into world space (moving with the map) versus left in overlay/screen space (fixed regardless of camera position).

## What evidence proves was built

- A reusable camera helper (`frontend/lib/victory-stage-camera.js`) mounted in First Theater: middle-mouse pan, cursor-centered wheel zoom, edge scrolling, `− / 100% / + / Fit` controls.
- Pixi scene split into shared world layers versus fixed overlay layers so the map/grid/pinned cards move together while unpinned cards stay readable above the interactive view.
- Card pin/unpin context-menu actions and drag behavior storing shared pin metadata on the card element.
- Backend: `update/index_card` and `act/duplicate_element` extended to preserve/write `pin_mode`, `world_x`, `world_y`, `screen_x`, `screen_y` without a separate pin route.
- Camera state is explicitly documented as personal browser storage, not shared Victory truth.

## Files/systems affected

`frontend/lib/victory-stage-camera.js` (new); First Theater Pixi scene/layering; index-card move/duplicate handlers (backend).

## Known deviations / later corrections

Kernel 48.1 (same day, 2026-06-21) found and fixed a real bug in this exact model: card move/pin/duplicate/create flows didn't consistently respect world-vs-overlay placement, and Director focus ping leaked across venues. Treat Kernel 48 as the baseline model and 48.1 as its immediate correction, not a separate feature.

## What this kernel handed to the next kernel

The world/overlay placement distinction this kernel introduced remained load-bearing for all later First Theater token/card work (Kernel 49 Warehouse assets, Kernel 50 persistent tokens).

## Confidence / unresolved gaps

High confidence — the operator-log entry is detailed and self-consistent, and is directly corroborated by a full, independently-written reportback (48.1) that treats it as settled prior fact. No original spec document survives; touch controls and Director focus/broadcast were explicitly deferred per the entry's own notes.
