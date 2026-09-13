# Kernel 31 Reportback

## Status
PARTIAL, but ready to continue.

The clean stage-shell exists and the repo now has the right kernel framing for it. The shell is intentionally not a full venue product yet.

## What Was Built
- Added a new producer-only venue shell at `frontend/venues/middle-school-stage/index.html`
- Added a map entry and routing for `middle-school-stage` in `frontend/app.js`
- Seeded the venue in `backend/internal/access/kernel16_venue_bootstrap.go`
- Made the venue visible to producers in `backend/internal/access/visibility.go`
- Updated the canon in `Construction/Canon/current-state.md`, `Construction/Canon/roadmap.md`, and `Construction/Process/kernel-maker-field-guide.md`
- Kept The Cave untouched
- Kept Pixi out of The Cave

## Evidence
- `GOCACHE=/tmp/victory-gocache go test ./...` passed from `/opt/victory/backend`
- `node --check frontend/app.js` passed
- Inline script check for `frontend/venues/middle-school-stage/index.html` passed
- `git diff --check` passed
- `curl -s http://127.0.0.1:8081/health` returned `ok: true` from the running backend
- `curl -s http://127.0.0.1:8081/api/session/me` returned `signed_in: false` without a cookie
- Anonymous `curl -s http://127.0.0.1:8081/api/map/visibility` did not return `middle-school-stage`

## How To Run
- Start the backend with the normal dev workflow from `Construction/Process/workflow/dev-workflow.md`
- Open the map and navigate to `Middle School Stage`
- Sign in with a producer or operator account to pass the shell bootstrap

## Operator Notes
- The Cave still works and remains the proving ground
- The shell is producer-only by design
- Pixi remains outside The Cave and is not made global by this kernel
- The live Cave action system is still Cave-specific; this kernel did not generalize it into a full multi-venue live protocol

## Blockers & Workarounds
- Full live-control parity is still Cave-specific, so the shell currently uses preview/placeholder behavior for several controls
- No generalized view-as switching was built
- No dynamic sheets, HP, dice, or game systems were added
- No Workshop cleanup was attempted

## Deviations From Kernel
- The kernel asked for portable controls where feasible; the shell includes the layout and control grammar, but some controls remain preview-only because the backend live protocol is still centered on The Cave
- The kernel asked for a center-first shell; the implementation keeps the center primary and moves controls to the edges, with a placeholder context menu at the stage object

## Known Issues
- Target info is a shell placeholder rather than a live selected-object inspector
- Presence and network drawers are present, but they are mostly placeholder content until generalized session plumbing exists
- Chat is collapsed by default and currently acts as a shell placeholder
- Right-side character/game content is intentionally sparse

## Next Recommended Step
- Generalize only the smallest portable primitives needed for Middle School Stage if that can be done safely
- Otherwise keep the shell as a layout proof and continue in a later kernel

## Files Changed / Created
- `backend/internal/access/kernel16_venue_bootstrap.go`
- `backend/internal/access/visibility.go`
- `frontend/app.js`
- `frontend/venues/middle-school-stage/index.html`
- `Construction/Canon/current-state.md`
- `Construction/Canon/roadmap.md`
- `Construction/Process/kernel-maker-field-guide.md`
- `Construction/Kernels/kernel-31-middle-school-stage-edge-drawer-layout-v1.md`

## Direct Questions
- Whether Middle School Stage exists: yes
- Whether producer-only visibility works: implemented in code and gated in bootstrap/visibility logic; this run only verified the anonymous-blocked path, not an authenticated producer session
- Whether direct unauthorized access is blocked: yes for anonymous access, by signed-in bootstrap plus server visibility gating; authenticated producer verification was not live-tested in this pass
- Whether The Cave still works: yes, no Cave code was changed in this kernel
- Which controls work in Middle School Stage: layout shell controls, edge drawer toggles, right-click preview menu
- Which controls are partial/deferred: live stage actions, generalized presence/network data, live chat, live persona selector, live target inspector
- Whether any backend route was generalized: no
- Whether any Cave code was changed: no
- Test results: pass
- Inline script check results: pass
- `git diff --check` result: pass
