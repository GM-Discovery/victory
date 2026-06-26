# Kernel Report Back - Kernel 52

## 1. Status
PASS WITH PENDING LIVE MULTI-CLIENT VERIFICATION

Kernel 52 adds Victory’s canonical dice system. The backend now owns result generation, validation, and persistence for `roll/dice`; the First Theater shell exposes a simple Dice tray; `/roll` chat commands route through the canonical roll path; and the browser stays out of canonical randomness entirely. The system is server-generated, append-only, and ready for later consumers such as Socio character creation and character sheets.

## 2. What Was Built

- Added a backend dice parser and roller in `backend/internal/dice/dice.go`.
- Used `crypto/rand` for canonical production rolls and kept the randomness source injectable for tests.
- Enforced server-side limits for expression length, dice count, sides, modifier size, and exploding-roll safety.
- Added canonical `roll/dice` action storage in `backend/internal/actions/dice.go`.
- Added authority gating for `roll/dice` in `backend/internal/actions/authority.go` for Director, Producer, and Operator access.
- Wired websocket handling for dice requests in `backend/internal/network/ws.go`.
- Added a First Theater Dice tray module in `frontend/venues/first-theater/runtime/dice.js`.
- Wired the tray into the First Theater shell in `frontend/venues/first-theater/runtime.js` and `frontend/venues/first-theater/index.html`.
- Routed `/roll ...` chat commands through the canonical dice path instead of letting the browser invent results.
- Added focused backend and frontend regression coverage for the dice path.

## 3. Evidence

Checks run:

- `cd /opt/victory/backend && GOCACHE=/tmp/victory-gocache go test ./internal/dice ./internal/actions ./internal/network`
- `node --test /opt/victory/tests/first-theater/*.test.js`
- `git diff --check`

Result summary:

- Backend dice, authority, and websocket tests passed.
- The full First Theater Node suite passed.
- Diff hygiene passed.

Safe test strategy:

- I used an injectable random source in the backend dice package for deterministic tests.
- I avoided client-side randomness for canonical results.
- I verified the tray as a renderer for canonical actions, not as a result generator.

## 4. How to Run

1. Open First Theater.
2. Use the Dice tray to submit an expression such as `d20`, `2d6+3`, or `1d10!`.
3. Confirm the roll appears as a canonical session action and renders in the tray history.
4. Send `/roll d20` through chat and confirm it routes through the same canonical path.
5. Re-run the First Theater Node suite if you change the tray or session-sync wiring.

## 5. Operator Notes

- `roll/dice` is now a first-class append-only action type.
- The browser does not generate canonical die results.
- The tray is textual only; Kernel 52 does not add a visual dice renderer, physics, 3D dice, or sound.
- The backend authority seam for future controlled-character rolling lives in `canActDiceRoll`.
- `OPERATOR_HANDLE` and `OPERATOR_USER_ID` still govern operator identity elsewhere in the system, but the dice kernel does not depend on a special browser-side operator mode.

## 6. Blockers & Workarounds

BLOCKER:
Victory needed a canonical dice system without moving result generation into the browser.

CAUSE:
The prior stack had no dedicated server-owned dice action path.

WORKAROUND:
Added backend parsing, server-generated random results, append-only `roll/dice` persistence, and a simple First Theater Dice tray that renders canonical actions.

BLOCKER:
The kernel needed to support arbitrary positive integer die sizes, not just conventional physical dice.

CAUSE:
Older gameplay surfaces were not built around a reusable canonical dice grammar.

WORKAROUND:
Implemented a bounded parser for `[N]dS[!][+M|-M]` with support for custom die sizes and exploding chains.

## 7. Deviations from Kernel

- I did not add a visual dice renderer, physics, 3D dice, or sounds.
- I kept dice visibility as public-only in this pass; private/director recipient filtering can be added later if the product needs it.
- I did not implement controlled-character permissions; I only left the authority seam in place for that later work.

## 8. Known Issues

- The Dice tray is intentionally textual, so it is not a visual animation surface.
- Multi-client live verification has not yet been exercised for the dice tray in the same way as the underlying Node and Go tests.
- Private/director-specific dice visibility is not implemented yet.

## 9. Next Recommended Step

- Exercise the Dice tray in a live two-client session.
- Add private/director visibility if the product needs non-public dice results.
- Reuse the canonical `roll/dice` action for future Socio character creation and character sheet workflows.

