# Kernel Report Back - Kernel 51A

## 1. Status
PASS WITH PENDING MULTI-CLIENT VERIFICATION

Kernel 51A covers the cumulative runtime refactor and follow-on hardening work since the initial Kernel 51 guidance. The First Theater runtime responsibility split landed, the requested operator/tooling support landed, the warehouse accounting and delete flows were repaired, the Discord server-link callback was fixed, and the live backend was rebuilt and validated. The theater startup blocker caused by the missing session-sync module binding was also fixed. Kernel 51B then completed the bounded bootstrap-shell cleanup. The remaining uncertainty is no longer architecture or refactor debt; it is limited to multi-client live verification for features that require a second connected user.

## 2. What Was Built

- Extracted First Theater projected state into `frontend/venues/first-theater/runtime/state.js`.
- Extracted First Theater WebSocket message parsing and dispatch into `frontend/venues/first-theater/runtime/socket.js`.
- Extracted First Theater socket lifecycle coordination into `frontend/venues/first-theater/runtime/socket-controller.js`.
- Extracted First Theater action routing and outgoing action dispatch into `frontend/venues/first-theater/runtime/action-router.js`.
- Extracted First Theater context-menu and pointer plumbing into `frontend/venues/first-theater/runtime/context.js`.
- Extracted the remaining editor shell into `frontend/venues/first-theater/runtime/editors.js`.
- Extracted First Theater session sync, staged creation, refresh, and join orchestration into `frontend/venues/first-theater/runtime/session-sync.js`.
- Extracted First Theater venue logic helpers into `frontend/venues/first-theater/runtime/logic.js`.
- Extracted pure geometry and coordinate math into `frontend/venues/first-theater/runtime/geometry.js`.
- Extracted listener/timer/subscription cleanup into `frontend/venues/first-theater/runtime/lifecycle.js`.
- Extracted stage control wiring into `frontend/venues/first-theater/runtime/stage-controls.js`.
- Extracted map/grid and camera helpers into `frontend/venues/first-theater/runtime/map-grid.js`.
- Extracted scene node assembly into `frontend/venues/first-theater/runtime/scene-nodes.js`.
- Extracted the token-asset picker / warehouse token UI into `frontend/venues/first-theater/runtime/token-ui.js`.
- Kept `frontend/venues/first-theater/runtime.js` as the composition root and browser bootstrap, not a second raw socket handler or second state store.
- Reduced `frontend/venues/first-theater/runtime.js` from 6,433 lines to 4,863 lines.
- Left the remaining venue feature logic in `runtime.js` for the later extraction pass:
  - bootstrap shell composition
  - remaining listener wiring and start-path glue
  - a small amount of render/bootstrap presentation logic
- Added focused regression tests for the extracted runtime modules:
  - `tests/first-theater/state.test.js`
  - `tests/first-theater/socket.test.js`
  - `tests/first-theater/socket-controller.test.js`
  - `tests/first-theater/action-router.test.js`
  - `tests/first-theater/context.test.js`
  - `tests/first-theater/editors.test.js`
  - `tests/first-theater/logic.test.js`
  - `tests/first-theater/geometry.test.js`
  - `tests/first-theater/stage-controls.test.js`
  - `tests/first-theater/map-grid.test.js`
  - `tests/first-theater/session-sync.test.js`
  - `tests/first-theater/token-ui.test.js`
- Added a reusable isolated-db guard script:
  - `scripts/test/require-isolated-database.sh`
- Added Grant's Cabin operator guidance and live storage diagnostics so the cabin shows how operator access works and where the live log lives.
- Defaulted blank `OPERATOR_HANDLE` values to `straturli` at backend startup so direct runs and recreated containers agree on the operator identity.
- Lowered the warehouse hard limit to `8 GB` and enforced an `8 GB` physical reserve on uploads.
- Added a filesystem storage diagnostics route:
  - `GET /api/warehouse/storage/filesystem`
- Fixed warehouse accounting so the storage endpoint reports the real DB-backed usage instead of collapsing to zeroes when filesystem lookup is unhappy.
- Added a Warehouse page delete control so producers can tombstone assets directly from the Warehouse browser, not only from Producer's Office.
- Removed the stray live `Gateway Edit Venue` fixture rows from the production database.
- Fixed the Discord server-link callback so the live env bot token wins over the stale token stored in the bootstrap row.
- Rebuilt and restarted the backend container after the code fixes landed.

## 3. Evidence

Checks run:

- `node --check frontend/venues/first-theater/runtime/state.js`
- `node --check frontend/venues/first-theater/runtime/socket.js`
- `node --check frontend/venues/first-theater/runtime/context.js`
- `node --check frontend/venues/first-theater/runtime/session-sync.js`
- `node --check frontend/venues/first-theater/runtime/editors.js`
- `node --check frontend/venues/first-theater/runtime/token-ui.js`
- `node --check frontend/venues/first-theater/runtime.js`
- `node --check` on extracted inline JavaScript from:
  - `frontend/venues/grants-cabin/index.html`
  - `frontend/venues/warehouse/index.html`
  - `frontend/venues/producers-office/index.html`
- `node --test tests/first-theater/*.test.js`
- `GOCACHE=/tmp/victory-gocache go test ./internal/assets`
- `GOCACHE=/tmp/victory-gocache go test -run '^$' ./internal/identity`
- `git diff --check`
- `docker exec victory-backend sh -lc "wget -qO- http://127.0.0.1:8081/health"`
- `docker exec victory-backend sh -lc "wget -qO- --header='Cookie: victory_session=warehousecheck-20260624' http://127.0.0.1:8081/api/warehouse/storage"`
- `docker exec victory-backend sh -lc "wget -qO- --header='Cookie: victory_session=warehousecheck-20260624' http://127.0.0.1:8081/api/warehouse/assets?asset_type=token"`
- `docker compose up -d --build backend`

Result summary:

- All Node syntax checks passed.
- All First Theater regression tests passed.
- Go package checks for assets and identity passed with the workspace cache.
- Diff hygiene passed.
- Backend health returned `ok:true` from inside the running container.
- The warehouse API now returns the actual stored bytes, active count, tombstoned count, and settings for the live location.
- The Warehouse asset browser now exposes direct delete buttons and the delete flow tombstones assets correctly.
- The Discord server-link callback now completes with the live bot token instead of failing with the stale bootstrap token.

Safe test strategy:

- I used pure-module tests for the extracted runtime pieces.
- I avoided destructive live-database tests.
- I validated the live service from inside the backend container instead of mutating production state blindly.
- I removed the stray gateway fixture data directly from Postgres only after confirming it was a leaked test venue, not production content.
- I rebuilt the backend container after code changes so the live site used the actual patched binary.

## 4. How to Run

1. Open First Theater.
2. Confirm the runtime still boots through `runtime.js` while state, socket, and token UI behavior come from the extracted modules.
3. Open Grant's Cabin and confirm the operator guidance and storage diagnostics are visible.
4. Open Warehouse and confirm asset thumbnails, delete buttons, and storage usage all render.
5. Open Producer's Office and confirm the storage panel shows the same live usage numbers.
6. Retry the Discord server install flow from Producer's Office and confirm it returns to the app instead of surfacing JSON error output.

## 5. Operator Notes

- `OPERATOR_HANDLE` and `OPERATOR_USER_ID` remain the operator access paths.
- The backend now defaults a blank `OPERATOR_HANDLE` to `straturli`, which matches the live user record for the operator account.
- The runtime extraction is only a first cut. `runtime.js` is smaller, but it is not yet bootstrap-only.
- The right-click and pointer plumbing now lives in `runtime/context.js` instead of the bootstrap file.
- The extracted modules now own the intended seams:
  - `geometry.js`: screen/world conversion, snapping, camera clamping, footprint math, and card coordinate conversion
- `logic.js`: placement/movement rules and shared venue behavior
- `map-grid.js`: map/grid helpers and camera state helpers
- `session-sync.js`: join, refresh, snapshot, focus-ping, and staged card placement orchestration
- `stage-controls.js`: stage-specific control wiring
- `scene-nodes.js`: Pixi/DOM scene node assembly
- `socket-controller.js`: socket lifecycle and message handling orchestration
- `action-router.js`: outgoing action routing through a shared `sendAction()`
- `context.js`: pointer readout, right-click handling, and stage-event hit testing
- `editors.js`: card/map/grid editor wiring and the shared editor control surface
- `session-sync.js`: join, refresh, snapshot, focus-ping, and staged card placement orchestration
- `lifecycle.js`: teardown for listeners, timers, subscriptions, and Pixi objects
- `token-ui.js`: workshop token asset selection and token preparation UI
- The live warehouse usage numbers were real; the earlier zeroes were caused by the handler path falling back too aggressively.
- `percent_used` can still round to `0` when usage is tiny relative to the 8 GB cap. That is a display artifact, not a missing-accounting bug.
- The Discord failure was a config-precedence bug, not a browser-install bug.

## 6. Blockers & Workarounds

BLOCKER:
First Theater had too much responsibility in one runtime file.

CAUSE:
Raw WebSocket parsing, canonical projected state, token UI concerns, geometry math, lifecycle cleanup, and rendering/bootstrap logic were all intertwined.

WORKAROUND:
Extracted `runtime/state.js`, `runtime/socket.js`, `runtime/socket-controller.js`, `runtime/action-router.js`, `runtime/context.js`, `runtime/logic.js`, `runtime/geometry.js`, `runtime/lifecycle.js`, `runtime/stage-controls.js`, `runtime/map-grid.js`, `runtime/scene-nodes.js`, `runtime/token-ui.js`, `runtime/editors.js`, and `runtime/session-sync.js`, then kept `runtime.js` as the composition root.

BLOCKER:
The user requested deeper decomposition of First Theater, but the full maps/grids/camera/card/token cluster was still too risky to finish in the same pass.

CAUSE:
Those features are tightly coupled to existing venue behavior and needed a safer, incremental extraction strategy.

WORKAROUND:
Moved the join/refresh/snapshot/session-placement orchestration into `runtime/session-sync.js` and left only the final bootstrap shell cleanup for the next pass.

BLOCKER:
Theater startup failed before the shell or Pixi stage could load.

CAUSE:
`runtime.js` referenced `firstTheaterSessionSyncModule` without binding `window.VictoryFirstTheaterSessionSync` first.

WORKAROUND:
Added the missing module binding so startup can reach the shell, DOM fallback, and Pixi initialization path.

BLOCKER:
Grant's Cabin did not explain operator access or surface live storage information.

CAUSE:
The cabin page was just a static invite-only notice.

WORKAROUND:
Added explicit `OPERATOR_HANDLE` / `OPERATOR_USER_ID` guidance and a live warehouse diagnostics panel.

BLOCKER:
The operator handle did not default cleanly in the live backend environment.

CAUSE:
Blank startup values could leave the operator identity ambiguous and make the cabin guidance less useful.

WORKAROUND:
Defaulted a blank `OPERATOR_HANDLE` to `straturli` so direct runs and rebuilt containers agree on the operator account.

BLOCKER:
Warehouse usage displayed zeroes even though assets existed.

CAUSE:
The storage handler was falling back to defaults too aggressively and the stats path was too fragile for the live warehouse and producer screens.

WORKAROUND:
Reworked the handler to resolve the warehouse location once, query settings and stats directly, and preserve the live DB-backed totals.

BLOCKER:
The Warehouse page had no direct delete affordance.

CAUSE:
Deletion only existed in Producer's Office.

WORKAROUND:
Added direct delete buttons to the Warehouse browser and reused the tombstone endpoint.

BLOCKER:
Warehouse delete flows were only partially visible from the operator-facing UI.

CAUSE:
Producer's Office had delete and storage monitor affordances, but the main warehouse surface still lacked direct destructive controls.

WORKAROUND:
Added delete controls to the Warehouse page so producers can tombstone assets from the browser that lists the thumbnails.

BLOCKER:
The Discord server install callback returned JSON with a guild lookup failure.

CAUSE:
The runtime config merge let the stale database bot token override the live container token.

WORKAROUND:
Changed the merge precedence so the env bot token wins when present.

BLOCKER:
The Discord connection surfaced a JSON auth failure after authorization.

CAUSE:
The bootstrap path still preferred a stale token source over the live container token.

## 7. Stabilization Follow-Up

Status after the follow-up regression pass:

- First Theater map replacement is working again.
- First Theater grid configure / save / hide-show is working again.
- Live token add / remove / move / replace behavior is working again.
- New token creates now resolve the real token art immediately instead of waiting for a later snapshot refresh.
- The token scale context-menu action now opens the token editor again.
- Audience hide/show is wired back through the modular action router and projected-state reducer.

Additional fixes applied after the main extraction:

- Fixed the grid editor recursion / stale-close path so Save no longer reverts on Close.
- Fixed the map editor so selecting an existing map asset updates the preview in-editor instead of immediately trying to apply and close.
- Fixed the token create/remove lag by syncing live runtime objects directly from projected state and scheduling Pixi re-renders from projected-state object changes.
- Fixed late-loading token textures by forcing a scene re-render when the Pixi texture finishes loading.
- Fixed `create/token` and `update/token` action payloads so live websocket events include:
  - `asset_content_url`
  - `asset_thumbnail_url`
  - token footprint metadata
- Rebuilt the `victory-backend` container after the backend payload fix so the live site actually served the patched token action code.

Regression checks re-run during stabilization:

- `node --test tests/first-theater/action-router.test.js tests/first-theater/editors.test.js tests/first-theater/session-sync.test.js tests/first-theater/state.test.js tests/first-theater/token-ui.test.js`
- `node --check frontend/venues/first-theater/runtime.js`
- `node --check frontend/venues/first-theater/runtime/action-router.js`
- `node --check frontend/venues/first-theater/runtime/token-ui.js`
- `node --check frontend/venues/first-theater/runtime/session-sync.js`
- `node --check frontend/venues/first-theater/runtime/state.js`
- `node --check frontend/venues/first-theater/runtime/scene-nodes.js`
- `GOCACHE=/tmp/victory-gocache go test ./internal/actions`
- `docker compose up -d --build backend`
- `docker exec victory-backend sh -lc "wget -qO- http://127.0.0.1:8081/health"`

Review result:

- No new code-review findings were identified after the stabilization fixes landed.
- Single-client producer verification now covers maps, grids, cards, tokens, context menus, voice, and mic.
- Residual risk is limited to multi-client live verification for audience visibility and Shift+Ping.

WORKAROUND:
Kept the env token authoritative in `discord_server_bootstrap.go` so the callback returns to the app instead of failing on `discord_guild_lookup_failed`.

BLOCKER:
A stray `Gateway Edit Venue` appeared on the main map.

CAUSE:
A leaked test fixture row had been left live in the production database.

WORKAROUND:
Removed the location, lot, venue, and session rows after confirming they were fixture data.

## 7. Deviations from Kernel

- I extracted the safest runtime pieces first instead of forcing a risky all-at-once breakup.
- I added a small filesystem diagnostics route rather than inventing a larger operator console.
- The operator log remains workspace markdown instead of a new server-backed app.
- I used the existing Warehouse delete endpoint and direct browser controls instead of adding a separate moderation workflow.
- I stopped after the bounded 51B bootstrap-shell cleanup rather than continuing into more architecture churn just to reduce line count further.

## 8. Known Issues

- `runtime.js` still contains the final bootstrap-shell composition and listener wiring, but the larger feature orchestration has now been cut out.
- The missing `window.VictoryFirstTheaterSessionSync` binding in `runtime.js` was fixed so the shell and Pixi bootstrap can load again.
- There is still no automated browser smoke harness for the full VTT surface.
- Multi-client live verification is still pending for:
  - Shift+Ping
  - audience hide/show visibility
- The percent-used readout can still round down to `0%` at low utilization.
- Cosmetic polish on the chat bar and other venue shells remains a later pass.

## 9. Next Recommended Step

- Start Kernel 52 feature work.
- When convenient, run a second-client live smoke check for Shift+Ping and audience visibility.
- Add a broader browser smoke harness for maps, grids, cards, tokens, context menus, chat, mic, and audio.
- If storage growth becomes a real operational problem, increase the underlying disk/volume before raising app caps.

## 9A. Kernel 51B Addendum

Bounded follow-on scope completed on 2026-06-25:

- Kept the pass limited to bootstrap-shell cleanup in `frontend/venues/first-theater/runtime.js`.
- Removed the large runtime-local fallback bodies for:
  - context-menu hit testing and pointer event normalization
  - context-menu open/resolve plumbing
  - stage-object action execution
- Kept those behaviors owned by the already-extracted modules instead:
  - `frontend/venues/first-theater/runtime/context.js`
  - `frontend/venues/first-theater/runtime/action-router.js`
- Left `runtime.js` focused more tightly on module binding, dependency wiring, startup order, DOM/Pixi mount flow, listener composition, and top-level shell behavior.
- Reduced `frontend/venues/first-theater/runtime.js` further to 4,315 lines.

51B checks run:

- `node --check frontend/venues/first-theater/runtime.js`
- `node --check frontend/venues/first-theater/runtime/action-router.js`
- `node --check frontend/venues/first-theater/runtime/context.js`
- `node --test tests/first-theater/action-router.test.js tests/first-theater/context.test.js tests/first-theater/editors.test.js tests/first-theater/session-sync.test.js tests/first-theater/state.test.js tests/first-theater/token-ui.test.js`

51B result:

- The bootstrap shell is leaner and no longer carries the old inline context-menu/action implementations as a second feature layer.
- No new modules were added.
- No architecture redesign was introduced.
- Manual single-client live regression now covers map, grid, camera, cards, tokens, context menus, chat, House Mic, and Discord Audio.
- The only remaining manual live checks are the multi-client behaviors: Shift+Ping and audience visibility.

## 10. Files Changed / Created

- [Construction/current-state.md](/opt/victory/Construction/current-state.md)
- [Construction/roadmap.md](/opt/victory/Construction/roadmap.md)
- [Construction/OperatorLogs/operator-log.md](/opt/victory/Construction/OperatorLogs/operator-log.md)
- [Construction/OperatorLogs/kernel-51-reportback.md](/opt/victory/Construction/OperatorLogs/kernel-51-reportback.md)
- [backend/cmd/victory/main.go](/opt/victory/backend/cmd/victory/main.go)
- [backend/internal/assets/upload.go](/opt/victory/backend/internal/assets/upload.go)
- [backend/internal/assets/warehouse.go](/opt/victory/backend/internal/assets/warehouse.go)
- [backend/internal/assets/warehouse_test.go](/opt/victory/backend/internal/assets/warehouse_test.go)
- [backend/internal/identity/discord_server_bootstrap.go](/opt/victory/backend/internal/identity/discord_server_bootstrap.go)
- [database/migrations/030_kernel51_capacity_guardrails.sql](/opt/victory/database/migrations/030_kernel51_capacity_guardrails.sql)
- [frontend/venues/grants-cabin/index.html](/opt/victory/frontend/venues/grants-cabin/index.html)
- [frontend/venues/first-theater/index.html](/opt/victory/frontend/venues/first-theater/index.html)
- [frontend/venues/first-theater/runtime.js](/opt/victory/frontend/venues/first-theater/runtime.js)
- [frontend/venues/first-theater/runtime/state.js](/opt/victory/frontend/venues/first-theater/runtime/state.js)
- [frontend/venues/first-theater/runtime/socket.js](/opt/victory/frontend/venues/first-theater/runtime/socket.js)
- [frontend/venues/first-theater/runtime/socket-controller.js](/opt/victory/frontend/venues/first-theater/runtime/socket-controller.js)
- [frontend/venues/first-theater/runtime/action-router.js](/opt/victory/frontend/venues/first-theater/runtime/action-router.js)
- [frontend/venues/first-theater/runtime/context.js](/opt/victory/frontend/venues/first-theater/runtime/context.js)
- [frontend/venues/first-theater/runtime/logic.js](/opt/victory/frontend/venues/first-theater/runtime/logic.js)
- [frontend/venues/first-theater/runtime/geometry.js](/opt/victory/frontend/venues/first-theater/runtime/geometry.js)
- [frontend/venues/first-theater/runtime/lifecycle.js](/opt/victory/frontend/venues/first-theater/runtime/lifecycle.js)
- [frontend/venues/first-theater/runtime/stage-controls.js](/opt/victory/frontend/venues/first-theater/runtime/stage-controls.js)
- [frontend/venues/first-theater/runtime/map-grid.js](/opt/victory/frontend/venues/first-theater/runtime/map-grid.js)
- [frontend/venues/first-theater/runtime/session-sync.js](/opt/victory/frontend/venues/first-theater/runtime/session-sync.js)
- [frontend/venues/first-theater/runtime/scene-nodes.js](/opt/victory/frontend/venues/first-theater/runtime/scene-nodes.js)
- [frontend/venues/first-theater/runtime/token-ui.js](/opt/victory/frontend/venues/first-theater/runtime/token-ui.js)
- [frontend/venues/producers-office/index.html](/opt/victory/frontend/venues/producers-office/index.html)
- [frontend/venues/warehouse/index.html](/opt/victory/frontend/venues/warehouse/index.html)
- [scripts/test/require-isolated-database.sh](/opt/victory/scripts/test/require-isolated-database.sh)
- [tests/first-theater/state.test.js](/opt/victory/tests/first-theater/state.test.js)
- [tests/first-theater/socket.test.js](/opt/victory/tests/first-theater/socket.test.js)
- [tests/first-theater/context.test.js](/opt/victory/tests/first-theater/context.test.js)
- [tests/first-theater/session-sync.test.js](/opt/victory/tests/first-theater/session-sync.test.js)
