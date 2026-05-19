# Kernel 32 Reportback

## Status
Complete for the intended extraction slice.

Kernel 32 was implemented as a template-venue extraction pass, not a Discord OAuth pass.

## What Was Built
- Added a hidden internal reusable shell venue at `frontend/venues/stage-template/index.html`.
- Mounted the portable overlay from the shared venue shell helper in `frontend/venues/first-theater/index.html` so Pixi and overlay can be compared together.
- Refactored `frontend/venues/middle-school-stage/index.html` to reuse the shared venue-shell helper for preferences and presence rendering.
- Extended `frontend/venues/shared/venue-shell.js` so shell preferences can keep per-venue storage prefixes instead of sharing one generic bucket.
- Updated the canon docs so Kernel 32 is the template extraction pass and Kernel 33 is Discord OAuth.

## Evidence
- Backend health responded successfully:
  - `curl -s http://127.0.0.1:8081/health`
  - Result: `{"ok":true,"service":"victory-backend","time":"2026-05-17T21:17:20.048032504Z"}`
- Anonymous session check stayed anonymous:
  - `curl -s http://127.0.0.1:8081/api/session/me`
  - Result: `{"ok":true,"signed_in":false}`
- Public map visibility stayed narrow and did not expose the hidden template scaffold:
  - `curl -s http://127.0.0.1:8081/api/map/visibility`
  - Result included public venues only: `construction`, `first-theater`, and `info-booth`
  - `stage-template` did not appear in the visible map payload
- Backend tests passed:
  - `cd /opt/victory/backend && GOCACHE=/tmp/victory-gocache go test ./...`
- Inline script checks passed:
  - `node --check` on the extracted First Theater inline script
  - `node --check` on the Middle School Stage inline script
  - `node --check` on the Stage Template inline script
  - `node --check frontend/venues/shared/venue-shell.js`
- Whitespace/diff check passed:
  - `git diff --check`

## How To Run
- Open `Middle School Stage` for the source shell.
- Open `First Theater` to see Pixi with the portable overlay mounted above it.
- Open the hidden `Stage Template` shell directly if you are signed in as producer or operator.

## Operator Notes
- The Cave was left untouched.
- The new template venue is hidden/internal and is not map-visible.
- The portable overlay helper is now shared between the template path and First Theater.
- Middle School Stage still serves as the source shell for the edge-drawer grammar.

## Blockers & Workarounds
- No backend route generalization was needed for this extraction slice.
- The hidden template venue uses client-side session/role gating instead of a new backend 403 route.
- Presence preview reuses the existing director-console current state payload instead of adding a new presence API.

## Deviations From Kernel
- Discord OAuth was not started here; it remains the next kernel.
- The extraction stayed frontend-only aside from doc updates and the shared shell helper reuse.
- The hidden template venue was intentionally kept out of the map registry so it stays internal.

## Known Issues
- The hidden template shell is scaffold-level only.
- First Theater still uses the existing Pixi renderer model; only the overlay presentation was shared.
- Middle School Stage remains shell-first and does not gain new backend parity from this pass.

## Next Recommended Step
- Kernel 33: Discord OAuth Primary Login v1

## Files Changed / Created
- `frontend/venues/shared/venue-shell.js`
- `frontend/venues/stage-template/index.html`
- `frontend/venues/first-theater/index.html`
- `frontend/venues/middle-school-stage/index.html`
- `Construction/current-state.md`
- `Construction/roadmap.md`
- `Construction/kernel-maker-field-guide.md`
- `Construction/OperatorLogs/operator-log.md`
- `Construction/OperatorLogs/operator-notes.md`
- `Construction/OperatorLogs/kernel-32-reportback.md`
