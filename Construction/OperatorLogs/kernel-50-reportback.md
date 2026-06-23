# Kernel Report Back - Kernel 50

## 1. Status
PASS

Kernel 50 is complete. First Theater now consumes reusable Warehouse token assets, lets authorized users place multiple persistent token instances on stage, supports grid-aware sizing and snapping, and keeps those placements in the canonical Victory state instead of treating them as browser-only edits.

## 2. What Was Built

- Added a First Theater `Add Token` stage context-menu action.
- Added a Warehouse token picker panel in First Theater with:
  - search
  - shape filter
  - active-asset list
  - thumbnail
  - name
  - shape
  - default footprint
  - select/cancel
- Wired the picker to the Warehouse token list endpoint:
  - `GET /api/warehouse/assets?asset_type=token&status=active`
- Kept token placement on the existing live action stream rather than adding a separate placement API:
  - websocket action `create/token`
  - websocket action `update/token`
- Reused the existing `elements` plus `venue_layout_elements` stage model as the canonical placement record.
- Stored token asset metadata on the placed element so one Warehouse asset can back many stage instances:
  - `asset_id`
  - `asset_name`
  - `asset_shape`
  - `asset_content_url`
  - `asset_thumbnail_url`
  - `default_grid_width`
  - `default_grid_height`
  - `retain_original`
  - `snap_mode`
  - `grid_relative`
  - `token_layer`
  - `scale`
- Added grid-aware placement behavior:
  - active grid uses the token footprint and grid snapping
  - no grid uses a `64px` baseline at `100%`
  - default token footprint is `1 x 1`
- Added token stage context-menu actions for:
  - `Snap to Grid` / `Free Placement`
  - `Hide Nameplate` / `Show Nameplate`
  - `Remove from Stage`
- Preserved the live map and card behavior already in First Theater while adding token placement on the same surface.

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
- Backend health check passed after rebuild/restart.

Safe test strategy:

- I used compile/test validation and syntax checks instead of live destructive database tests.
- I verified the backend service restarted cleanly after the token placement changes.
- I relied on the persisted action stream plus canonical layout rows rather than adding ad hoc test-only storage.

Confirmed behavior from code paths:

- Token placement is authorized for `director` and `producer` in the current implementation.
- Audience users do not get the `Add Token` action.
- No new dedicated token-placement table was added.
- No new HTTP placement route was added; placement rides the websocket action stream.

## 4. How to Run

1. Start or restart the backend.
2. Open First Theater.
3. Right-click the playable stage and choose `Add Token`.
4. Select a reusable Warehouse token asset.
5. Place the token at the original clicked world point.
6. Use `Snap to Grid` or `Free Placement` from the token context menu if needed.
7. Refresh the page to confirm the token remains placed.

## 5. Operator Notes

- The Warehouse remains the reusable asset store.
- First Theater is the proving venue for token instances, not the asset store itself.
- Placement permission is currently `director` and `producer`; the UI does not expose token placement to audience users.
- The stage token uses the Warehouse asset image variants; it does not copy the file per placement.
- The placement state is persisted in Victory canonical data, not only in Pixi memory.

## 6. Blockers & Workarounds

BLOCKER:
The environment does not provide an isolated disposable PostgreSQL test database.

CAUSE:
Integration mutation tests would have been destructive against the shared workspace database.

WORKAROUND:
Used Go compile/test checks, frontend syntax checks, and diff hygiene checks instead of destructive database tests.

BLOCKER:
Kernel 50 needed a stable source of reusable token assets.

CAUSE:
The theater placement work depends on the Warehouse asset model from Kernel 49.

WORKAROUND:
Used the Warehouse token picker and the existing `assets` / `venue_layout_elements` model rather than inventing a parallel placement store.

## 7. Deviations from Kernel

- The user-facing picker is embedded in the existing First Theater shell rather than in a separate tool page.
- The current implementation gates placement to `director` and `producer`; the kernel brief suggested a broader `Director and above` range, but the code only authorizes those two roles.
- No separate token-placement database table was added.
- No new placement HTTP endpoint was introduced.

## 8. Known Issues

- No operator-level token placement permission is exposed in the current implementation.
- No folders, marketplace, initiative system, token vision, or fog-of-war were added.
- This kernel does not add a dedicated asset-management page for Theater-only placements.

## 9. Next Recommended Step

- Do a live browser smoke of:
  - place token
  - move token
  - toggle nameplate
  - refresh
  - remove from stage
- If placement stays stable, the next useful step is broader First Theater token workflow polish rather than more storage plumbing.

## 10. Files Changed / Created

- [backend/internal/actions/token.go](/opt/victory/backend/internal/actions/token.go)
- [backend/internal/network/ws.go](/opt/victory/backend/internal/network/ws.go)
- [backend/internal/world/snapshot.go](/opt/victory/backend/internal/world/snapshot.go)
- [frontend/venues/first-theater/index.html](/opt/victory/frontend/venues/first-theater/index.html)
- [frontend/venues/first-theater/runtime.js](/opt/victory/frontend/venues/first-theater/runtime.js)
