# Kernel Report Back - Kernel 50+

## 1. Status
PASS

Kernel 50+ is the follow-on hardening pass for Kernel 50. It fixes the remaining refresh persistence issues exposed during live testing: token moves now survive refresh, and hidden nameplates stay hidden instead of popping back on after a reload or hover interaction.

## 2. What Was Built

- Added `x`, `y`, and `order` to `update/token` action payloads so moved tokens can be reconstructed from the action stream.
- Hardened token replay so snapshot reconstruction falls back to the persisted `venue_layout_elements.position` row when an older `update/token` record is sparse.
- Removed the old token replay gate that silently ignored `update/token` actions when `asset_id` was missing from the target payload.
- Preserved token nameplate state on replay instead of forcing it back to visible.
- Hardened First Theater token rendering so hidden labels are removed from the token container instead of just being toggled visually.
- Normalized nameplate state reads to accept both snake_case and camelCase state fields:
  - `nameplate_visible`
  - `nameplateVisible`
- Updated the First Theater runtime cache-bust to force the browser to load the new token runtime script.

## 3. Evidence

Checks run:

- `GOCACHE=/tmp/victory-gocache go test ./internal/actions ./internal/world`
- `node --check frontend/venues/first-theater/runtime.js`
- `git diff --check`
- `curl -s http://127.0.0.1:8081/health`

Result summary:

- Go checks passed.
- First Theater runtime syntax check passed.
- Diff hygiene check passed.
- Backend health check passed after the container rebuild/restart.

Safe test strategy:

- I verified the fix with compile/test checks and a live backend health probe rather than destructive DB mutation tests.
- The replay fallback is intentionally conservative so older sparse actions still resolve to the persisted layout row.

## 4. How to Run

1. Open First Theater.
2. Move a token.
3. Hide the nameplate.
4. Refresh the browser.
5. Confirm the token stays in the moved position and the nameplate stays hidden.

## 5. Operator Notes

- The root cause was split across action persistence and replay.
- The live layout row already held the correct token position; replay just was not reconstructing it reliably from older action payloads.
- The nameplate issue was a render/state mismatch, not a Warehouse asset problem.
- No new user-facing tool was required for this fix.

## 6. Blockers & Workarounds

BLOCKER:
Older `update/token` actions in the stream did not always carry coordinates.

CAUSE:
The initial token update payload only recorded asset and mode metadata.

WORKAROUND:
Added `x`, `y`, and `order` to the payload and fell back to the stored layout row during snapshot reconstruction.

BLOCKER:
Hidden nameplates could reappear after refresh or selection changes.

CAUSE:
Replay was restoring the token with a default visible nameplate instead of honoring the persisted state.

WORKAROUND:
Preserved `nameplate_visible` from the saved visibility data and removed hidden label nodes from the token container.

## 7. Deviations from Kernel

- This is a follow-on maintenance pass, not a new user-facing kernel feature.
- No new route or table was introduced for the fix.

## 8. Known Issues

- Placement permissions remain director/producer only.
- No broader operator token placement role was added.
- No new live review tooling was required beyond the browser smoke path.

## 9. Next Recommended Step

- Keep an eye on the live token flow for any remaining edge cases around duplicate tokens or stage removal.
- If there is appetite for it later, add a lightweight regression smoke script for place/move/hide/refresh.

## 10. Files Changed / Created

- [backend/internal/actions/token.go](/opt/victory/backend/internal/actions/token.go)
- [backend/internal/world/snapshot.go](/opt/victory/backend/internal/world/snapshot.go)
- [frontend/venues/first-theater/index.html](/opt/victory/frontend/venues/first-theater/index.html)
- [frontend/venues/first-theater/runtime.js](/opt/victory/frontend/venues/first-theater/runtime.js)
