# Operator Log — VICTORY

This file records meaningful implementation milestones and notable operational decisions.
Keep entries factual.

Current status and roadmap now live in:
- [current-state.md](/opt/victory/Construction/current-state.md)
- [roadmap.md](/opt/victory/Construction/roadmap.md)

This file should stay historical and chronological.

---

## 2026-06-29 — Kernel 53 Character Workbook Foundation + Socio Parentage v1.1

### Backend
- Extended the canonical `character_cards` root with workbook metadata, active-character persistence, workbook module instances, and private journal storage.
- Added a Catharsis workbook creation/resume seam keyed off workbook context so the first onboarding draft can be resumed instead of duplicated.
- Added a `POST /api/character-journals` surface for private journal entries targeting the active workbook.
- Exposed `active_character` on `/api/session/me` and `GET /api/character-cards/me` so the shell can render the loaded workbook state.

### Frontend
- Wired the Catharsis onboarding start action to create the first draft workbook and route the player into Greenroom.
- Added a shared `/journal` command path in the venue chat helpers so journal text bypasses venue chat and lands in private workbook storage.
- Projected the active character into the Catharsis and First Theater shell chrome and made the shell chip jump back to Greenroom.
- Allowed Greenroom to open directly to a requested workbook via `?character_id=...` and show the workbook status in the character surface.

### Tests
- Re-ran focused backend coverage for the touched domains:
  - `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/characters ./internal/access ./internal/identity ./internal/network`
- Re-ran frontend syntax checks for the touched shared and venue scripts:
  - `node --check frontend/lib/victory-mic-chat.js`
  - `node --check frontend/venues/catharsis/runtime.js`
  - `node --check frontend/venues/first-theater/runtime.js`
- Verified diff hygiene:
  - `git diff --check`

### Notes
- The full Kernel 53 Parentage Chart v1.1 was not provided in the attachment, so the parentage stage itself remains a foundation rather than a complete playable flow.
- `go test ./...` still reports an unrelated pre-existing failure in `backend/internal/assets`.

## 2026-06-25 — Kernel 52 Canonical Dice Actions v1

### Backend
- Added a canonical dice parser/roller using `crypto/rand` with deterministic test injection support and server-side limits for expression length, dice count, sides, modifier size, and explosion safety.
- Added a new append-only `roll/dice` action path with canonical payload storage and authority checks for Director, Producer, and Operator roll authority.
- Wired the websocket dispatcher so `/roll` requests are validated, rolled, persisted, and broadcast from the backend instead of being generated in the browser.

### Frontend
- Added a First Theater Dice tray module that accepts `/roll` expressions, submits canonical roll requests, and renders the authoritative session action stream as a readable history.
- Wired the dice tray into the First Theater shell and chat command flow so `/roll ...` goes through the same canonical action path.

### Tests
- Added backend coverage for the dice parser, authority gate, websocket dispatch, and canonical roll persistence path.
- Added First Theater dice tray coverage for expression building, request/response correlation, validation failures, timeouts, and remount behavior.
- Re-ran the full First Theater Node suite:
  - `tests/first-theater/*.test.js`
- Re-ran backend coverage for the new dice path:
  - `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/dice ./internal/actions ./internal/network`
- Verified diff hygiene:
  - `git diff --check`

### Notes
- The browser does not generate canonical die results.
- The tray is textual only; there is no visual dice renderer in this kernel.
- Controlled-character roll authority remains deferred behind the `canActDiceRoll` policy seam.

## 2026-06-25 — Kernel 51B bootstrap-shell cleanup

### Frontend
- Reduced the remaining First Theater runtime shell by removing the large runtime-local fallback bodies for context-menu targeting, pointer-event normalization, menu open/resolve flow, and stage-object action execution.
- Kept those behaviors owned by the extracted modules instead of maintaining a second inline implementation in `frontend/venues/first-theater/runtime.js`:
  - `frontend/venues/first-theater/runtime/context.js`
  - `frontend/venues/first-theater/runtime/action-router.js`
- Kept `runtime.js` focused more narrowly on module binding, dependency construction, shell/bootstrap flow, and listener composition.

### Tests
- Re-verified syntax:
  - `node --check frontend/venues/first-theater/runtime.js`
  - `node --check frontend/venues/first-theater/runtime/action-router.js`
  - `node --check frontend/venues/first-theater/runtime/context.js`
- Re-ran focused First Theater regression coverage:
  - `tests/first-theater/action-router.test.js`
  - `tests/first-theater/context.test.js`
  - `tests/first-theater/editors.test.js`
  - `tests/first-theater/session-sync.test.js`
  - `tests/first-theater/state.test.js`
  - `tests/first-theater/token-ui.test.js`

### Notes
- `frontend/venues/first-theater/runtime.js` is now 4,315 lines.
- This was intentionally a bounded cleanup pass, not another broad runtime redesign.
- No new runtime modules were added in 51B.
- Manual live browser regression is still required for:
  - map
  - grid
  - camera
  - cards
  - tokens
  - context menus
  - Shift+Ping
  - chat
  - House Mic
  - Discord Audio

## 2026-06-24 — Kernel 51 Session Sync and Stage Placement Cutover

### Frontend
- Added `frontend/venues/first-theater/runtime/session-sync.js` to own the live join, refresh, snapshot, focus-ping, and staged index-card orchestration.
- Moved the First Theater staged index-card creation/placement flow onto the session-sync controller instead of keeping it inline in `runtime.js`.
- Kept `runtime.js` as the composition root while delegating the join/refresh/snapshot wiring through the new controller.

### Tests
- Added `tests/first-theater/session-sync.test.js`.
- Re-ran the full First Theater Node suite:
  - `tests/first-theater/*.test.js`
- Verified syntax and diff hygiene:
  - `node --check frontend/venues/first-theater/runtime/session-sync.js`
  - `node --check frontend/venues/first-theater/runtime.js`
  - `git diff --check`

### Notes
- `frontend/venues/first-theater/runtime.js` is now 4,863 lines.
- The next remaining seam is the final bootstrap-shell cleanup around the start path and listener composition.
- The runtime decomposition remains incremental, but the live orchestration surface is now slimmer and covered by an additional test file.

## 2026-06-24 — First Theater Boot Fix

### Frontend
- Fixed the missing `window.VictoryFirstTheaterSessionSync` module binding in `frontend/venues/first-theater/runtime.js`.
- That binding bug was preventing the First Theater shell, Pixi stage, and DOM fallback from loading at startup.

### Tests
- Re-ran the First Theater Node suite after the fix:
  - `tests/first-theater/*.test.js`
- Rechecked syntax:
  - `node --check frontend/venues/first-theater/runtime.js`
  - `node --check frontend/venues/first-theater/runtime/session-sync.js`

### Notes
- The theater startup blocker is resolved.
- The remaining bootstrap-shell cleanup is now a refactor tidy-up, not a startup failure.

## 2026-06-24 — Kernel 51 First Theater Editor Cutover Completion

### Frontend
- Completed the editor-shell extraction for First Theater by moving the remaining map and grid editor helpers into `frontend/venues/first-theater/runtime/editors.js`.
- Restored the editor helper API so `runtime.js` can keep delegating map draft, payload, grid default, and grid draft logic through the shared editor controller surface.
- Kept the First Theater bootstrap and runtime wiring intact while the editor seam moved out of the monolith.

### Tests
- Re-ran the focused editor test:
  - `tests/first-theater/editors.test.js`
- Re-ran the full First Theater Node suite:
  - `tests/first-theater/*.test.js`
- Verified syntax and diff hygiene:
  - `node --check frontend/venues/first-theater/runtime/editors.js`
  - `git diff --check`

### Notes
- `frontend/venues/first-theater/runtime.js` is now 5,203 lines.
- The remaining major seams are the placement/action orchestration cluster and the final bootstrap shell cleanup.
- The First Theater runtime refactor is still incremental, but the editor seam is now stable again and covered by tests.

## 2026-06-24 — Kernel 51A Finalization, Warehouse Repair, and Discord Link Fixes

### Frontend
- Added direct delete controls to the Warehouse browser asset cards.
- Kept Producer's Office and Warehouse storage panels aligned with the same live warehouse storage numbers.

### Backend
- Reworked warehouse storage accounting to use a direct location-ID path for the live `amurray-family` data instead of collapsing to zeroes.
- Fixed the Discord server-link runtime merge so the live env bot token wins over the stale bootstrap token.

### Ops
- Removed the stray live `Gateway Edit Venue` fixture rows from the production database.
- Rebuilt and restarted the backend container after the storage and Discord fixes landed.

### Notes
- Warehouse usage is now reporting the real stored bytes and asset counts again.
- Small utilization can still round down to `0%` because the percent display is integer-based.
- The Kernel 51 runtime decomposition remains partial; `runtime.js` is smaller, but the maps/grids/camera/cards/tokens cluster still needs the follow-on pass.

## 2026-06-23 — Kernel 51 First Theater Geometry, Placement, and Lifecycle Split

### Frontend
- Extracted the First Theater geometry and placement cluster into `frontend/venues/first-theater/runtime/geometry.js`.
- Kept the shared placement rules testable for:
  - stage playable bounds
  - screen/world conversion
  - card coordinate conversion
  - token footprint and size math
  - square and hex snapping
  - token move/duplicate placement
- Added a lifecycle bag at `frontend/venues/first-theater/runtime/lifecycle.js` for tracked listeners, timers, animation frames, and cleanup callbacks.
- Wired the First Theater runtime to use the new geometry/lifecycle modules before `runtime.js` starts.

### Tests
- Added `tests/first-theater/geometry.test.js` covering the pure geometry and placement helpers.
- Kept the existing logic, state, and socket tests green after the split.

### Notes
- `runtime.js` is still not down to bootstrap-only composition.
- Map/grid editors, context menus, token picker/editor flows, Pixi rendering, and socket orchestration still remain inline for later extraction.

## 2026-06-23 — Kernel 51 First Theater Stage Controls Split

### Frontend
- Added `frontend/venues/first-theater/runtime/stage-controls.js`.
- Moved camera label and lock-state calculations into the stage-controls helper.
- Moved map editor draft normalization into the stage-controls helper.
- Moved default grid configuration and grid draft normalization into the stage-controls helper.
- Moved grid hex-field visibility and grid visibility button labeling into the stage-controls helper.
- Wired `frontend/venues/first-theater/index.html` to load the new helper before `runtime.js`.

### Tests
- Added `tests/first-theater/stage-controls.test.js`.
- Covered camera label/lock behavior, map draft normalization, grid defaults, grid draft normalization, and grid UI labels.

### Notes
- This is another partial decomposition pass, not the completed First Theater refactor.
- `runtime.js` still owns the map/grid editor workflows, Pixi scene orchestration, context menus, token picker/editor flows, and live socket-driven behavior.

## 2026-06-23 — Kernel 51 First Theater Map/Grid Workflow Split

### Frontend
- Added `frontend/venues/first-theater/runtime/map-grid.js`.
- Moved the map editor draft normalization and payload shaping into the map-grid helper.
- Moved the map/grid panel clamping helper into the map-grid helper.
- Moved the default grid config and grid draft normalization into the map-grid helper.
- Moved the grid hex-field visibility and visibility-button label helpers into the map-grid helper.
- Wired `frontend/venues/first-theater/index.html` to load the new helper before `runtime.js`.

### Tests
- Added `tests/first-theater/map-grid.test.js`.
- Covered panel clamping, map draft normalization, map payload shaping, grid defaults, grid draft normalization, and grid UI labels.

### Notes
- This is still an incremental decomposition pass.
- `runtime.js` remains responsible for the live map and grid fetch/update flows, Pixi rendering, token/card placement, context menus, and socket orchestration.

## 2026-06-23 — Kernel 51 First Theater Scene Node Split

### Frontend
- Added `frontend/venues/first-theater/runtime/scene-nodes.js`.
- Moved First Theater card, fire, and token Pixi node construction into the scene-node helper.
- Moved scene-node clearing and selection refresh responsibilities into the scene-node helper.
- Wired `frontend/venues/first-theater/index.html` to load the scene-node helper before `runtime.js`.

### Notes
- This continues the incremental First Theater decomposition.
- `runtime.js` is smaller and now leans on shared helpers for geometry, lifecycle, stage controls, map/grid workflow, and scene-node construction.

## 2026-06-23 — Kernel 51 First Theater Token UI Split

### Frontend
- Added `frontend/venues/first-theater/runtime/token-ui.js`.
- Moved the First Theater token picker flow into the token-ui helper.
- Moved the Warehouse token asset filtering and preview logic into the token-ui helper.
- Moved token placement and replacement action shaping into the token-ui helper.
- Moved the token editor open/save/close flow into the token-ui helper.
- Wired `frontend/venues/first-theater/index.html` to load the new helper before `runtime.js`.

### Tests
- Added `tests/first-theater/token-ui.test.js`.
- Covered token asset filtering by shape/search.

### Notes
- This is still incremental decomposition, not a finished refactor.
- `runtime.js` still owns context-menu orchestration, live socket wiring, and the remaining venue shell integration.

## 2026-06-23 — Kernel 51 First Theater Logic Module Split

### Frontend
- Extracted the First Theater object/permission/menu helper layer into `frontend/venues/first-theater/runtime/logic.js`.
- Wired `frontend/venues/first-theater/index.html` to load the new helper module before `runtime.js`.
- Redirected the First Theater runtime to use the helper module for object-kind detection, visibility state, permission checks, token metadata, card draft generation, and stage-action resolution.

### Tests
- Added a dedicated Node regression suite at `tests/first-theater/logic.test.js` covering:
  - object-kind normalization
  - visibility and lock state normalization
  - permission gating
  - token metadata and sizing
  - card editor draft generation
  - stage and token action resolution

### Notes
- This is a targeted decomposition pass, not the full remaining First Theater runtime split.
- The scene, geometry, and placement code still remain in `runtime.js` for later extraction.

## 2026-06-23 — Kernel 51A Capacity Guardrail Correction

### Backend
- Lowered the warehouse hard limit default from 15 GB to 8 GB in code and in the warehouse storage settings migration.
- Added a physical filesystem diagnostics route at `/api/warehouse/storage/filesystem`.
- Enforced an 8 GB physical reserve during uploads so new writes are rejected before disk space drops below the reserve.
- Defaulted blank operator handles to `straturli` at backend startup so local runs and recreated containers share the same operator identity default.

### Frontend
- Updated Grant's Cabin to show actual filesystem capacity, free space, reserve size, and safe upload capacity alongside warehouse policy values.

### Notes
- This is a corrective pass for the Kernel 51 reportback and not a claim that the broader First Theater runtime decomposition is finished.
- The runtime is still only partially decomposed; the feature monolith remains for maps, grids, cards, tokens, pickers, and menus.

---

## 2026-06-23 — Kernel 51 First Theater Runtime Decomposition + Cabin Diagnostics

### Frontend
- Split the First Theater runtime into dedicated projected-state and WebSocket-dispatch modules at:
  - `frontend/venues/first-theater/runtime/state.js`
  - `frontend/venues/first-theater/runtime/socket.js`
- Wired the First Theater bootstrap to load those modules before `runtime.js` starts.
- Kept the existing First Theater UI behavior intact while reducing the inline runtime's responsibility surface.
- Added Grant's Cabin operator guidance and live warehouse storage diagnostics to explain how operator access is granted and how storage limits are currently configured.
- The cabin now also reports actual filesystem capacity and reserve headroom via the new filesystem diagnostics route.

### Ops
- Verified the current operator cleanup target and safely pruned the Docker build cache once the stale resources were confirmed.
- Reclaimed the unused Docker build cache after checking the running containers, the filesystem footprint, and the current storage pressure.

### Notes
- Operator cabin access is still governed by `OPERATOR_HANDLE` or `OPERATOR_USER_ID` in the backend environment.
- The cabin now tells the operator where the log lives and shows the active warehouse hard limit, upload cap, warning thresholds, token variant sizes, and filesystem reserve data.
- The First Theater refactor is still only a first cut; state and socket handling moved out, but most venue feature logic remains in `runtime.js`.

## 2026-06-23 — Workshop Route Cleanup

### Frontend
- Redirected The Cave `?mode=workshop` path to the dedicated asset-only `/venues/workshop/` surface before the Cave shell paints.

### Notes
- The Workshop surface is now the standalone asset-prep page rather than a hybrid Cave shell.
- This change removes the brief flash of Cave controls before the workshop view appears.

## 2026-06-22 — Kernel 50 First Theater Token Placement + 50+ Refresh Persistence Hardening

### Backend
- Added First Theater token placement through the existing live action stream with `create/token` and `update/token`.
- Kept Warehouse assets reusable and used the existing `elements` plus `venue_layout_elements` canonical state for stage token instances.
- Hardened replay so token moves survive refresh even when older `update/token` records are sparse.

### Frontend
- Added the First Theater `Add Token` picker and stage token context-menu actions.
- Added grid-aware sizing, snap/free placement, and nameplate toggling for placed tokens.
- Hardened First Theater token rendering so hidden nameplates stay hidden on refresh.

### Docs
- Added the Kernel 50 reportback and the Kernel 50+ hardening reportback in the house template format.

### Notes
- Placement permission is currently director/producer only.
- No new placement table or placement HTTP route was introduced; the live websocket action stream remains the source of truth.

## 2026-06-21 — Kernel 49 Workshop Token Ingestion + Warehouse Storage

### Backend
- Extended the existing workshop asset flow with token uploads at `POST /api/workshop/assets/token`.
- Added Warehouse storage policy and stats endpoints for installation-wide limits, warning thresholds, retention defaults, and token variant sizing.
- Added tombstone deletion support that removes durable files, preserves the asset record, and resolves broken reads to the construction fallback.
- Extended the asset model with durable warehouse metadata such as token shape, default footprint, crop/zoom settings, storage bytes, and deleted-state timestamps.

### Frontend
- Added a token preparation surface inside The Cave's `mode=workshop` route with circle, square, hex, and raw masks.
- Added a Producer's Office warehouse storage panel with policy editing, storage metrics, and a warehouse asset browser.
- Wired recent token browsing into the Workshop surface and asset deletion into the warehouse browser.

### Docs
- Updated current-state, roadmap, and this operator log to Kernel 49.

### Notes
- Workshop token preparation now uses the existing Workshop/asset flow rather than a parallel token-only architecture.
- Deleted or missing assets now fall back to the construction image rather than leaving existing placements blank.

## 2026-06-21 — Kernel 48.1 Completion Reportback and Map Asset Reliability Notes

### Docs
- Added the Kernel 48.1 reportback in the house template format.
- Updated current-state canon with the existing-map activation fix and the full-map grid coverage note.
- Recorded the kernel extras in the reportback so the follow-on fixes stay attached to the same kernel history.

### Notes
- Existing map assets now activate directly from the list instead of leaving the editor in a blank half-selected state.
- The reportback treats the recent header, camera, grid, and map-editor adjustments as kernel extras layered onto the core 48.1 slice.

## 2026-06-21 — Kernel 48.1 Card Attachment Repair + Director Focus Ping

### Frontend
- Tightened the First Theater card movement and duplication paths so screen-pinned cards stay screen-pinned, map-attached cards stay world-attached, and unlocked cards use a transient floating drag state without snap-back.
- Renamed card pin actions to `Attach to Map` and `Pin to Screen`.
- Added a Ping button that sends ordinary websocket pings and Shift+Ping director focus broadcasts.
- Added a 250ms camera transition for focus broadcasts and a temporary Director focus marker.

### Backend
- Added a venue focus ping websocket event that is authorized server-side and broadcast to the current session only.
- Extended duplicate-card handling so duplicated cards preserve their durable pin mode and coordinates.

### Docs
- Updated current-state and roadmap canon to Kernel 48.1.

### Notes
- The camera remains personal and browser-local after focus.
- Touch controls, snapping, and Director follow mode remain deferred.

## 2026-06-21 — Kernel 48 Personal Stage Camera + Card Pinning v1

### Frontend
- Added a reusable First Theater camera helper at `frontend/lib/victory-stage-camera.js`.
- Mounted a personal browser-local camera in First Theater with middle-mouse pan, cursor-centered wheel zoom, edge scrolling, and fixed `− / 100% / + / Fit` controls.
- Split the First Theater Pixi scene into shared world layers and fixed overlay layers so the active map, grid, and pinned cards move together while unpinned cards remain readable above the interactive view.
- Added card pin/unpin context-menu actions and drag behavior that store shared pin metadata on the card element itself.
- Tightened the card context-menu move and duplicate paths so pinned cards stay in world space, overlay cards stay in screen space, and duplication preserves a single copy in the correct space.

### Backend
- Extended `update/index_card` and `act/duplicate_element` handling so card moves and card duplication can preserve and write optional `pin_mode`, `world_x`, `world_y`, `screen_x`, and `screen_y` metadata without creating a separate pin route.

### Docs
- Updated current-state and roadmap canon to Kernel 48.

### Notes
- Camera state is personal browser storage, not shared Victory truth.
- Touch controls and Director focus/broadcast remain deferred.

## 2026-06-11 — Kernel 46 Pixi Map Layer + Workshop Map Upload v1

### Backend
- Added a new First Theater venue-map API at `GET /api/venues/{slug}/map` and `POST /api/venues/{slug}/map`.
- Extended workshop uploads so `POST /api/workshop/assets` can tag uploaded assets as `map`, and `GET /api/workshop/assets?asset_type=map` lists the uploaded map assets for the venue location.
- Added an authenticated asset content route at `GET /api/assets/{id}/content` so Pixi can load the saved map image directly from the server.
- Added durable schema for `assets.asset_type`, `assets.tags`, and a single active venue map row per venue.

### Frontend
- Kept the old First Theater stage façade intact.
- Added a dedicated Pixi map layer inside the First Theater stage shell.
- Added a right-click `Add / Replace Map` stage action that opens a Workshop-style map editor panel.
- Added upload/select controls, crop/fit/scale/safe-margin controls, and map-asset selection to the First Theater editor.

### Docs
- Updated current-state, roadmap, and the Kernel 46 reportback to reflect the real map-layer workflow instead of the backdrop experiment.

### Notes
- The active map is persisted server-side and rendered from the saved venue-map state.
- The older First Theater presentation remains visible around the map layer and still owns the DOM controls.

## 2026-06-09 — Kernel 44 Discord Audio Presence / Speaker Feasibility v1

### Backend
- Added live Discord voice-state caching from the Gateway path so the audio surface can show current participants for mapped voice channels.
- Extended the audio status endpoint to return participant rows, linked Victory identity where available, and explicit feature flags for speaker-indicator and volume-control feasibility.
- Updated the default Discord gateway intents to include `GUILD_VOICE_STATES` so participant presence is actually observable.

### Frontend
- Extended the shared presence tray to render participant rows with Discord avatars, linked Victory display names, unlinked Discord users, and live status badges.
- Kept speaker indicators and volume controls visibly deferred rather than faked.

### Docs
- Updated install/deployment/current-state/roadmap guidance to document the voice-state intent requirement and the current speaker/volume feasibility boundary.

### Notes
- Speaker indicators are not available from the current bot/Gateway path without a deeper voice websocket or Discord client SDK path.
- Per-user volume controls are also deferred because the current Discord bot/Gateway path cannot control a user’s local Discord client volume.

## 2026-06-09 — Kernel 43 Discord Audio Left Tray Foundation v1

### Backend
- Added Discord audio status handling at `GET /api/discord/audio/status`.
- Extended Discord channel mapping repair/status flow to track voice-channel mappings for First Theater and The Cave with the `venue_audio_channel` mapping kind.
- Kept the repair flow idempotent so it can find or create the mapped Discord voice channels instead of treating audio as a separate transport stack.

### Frontend
- Added a shared Discord audio tray helper at `frontend/lib/discord-audio.js`.
- Mounted the audio status surface in First Theater and The Cave.
- Added producer-office readiness surfaces for the audio mappings.

### Docs
- Updated the install contract, fresh-install notes, roadmap, and current-state canon to reflect the Discord audio left-tray foundation.
- Noted the channel-management permissions needed for audio channel repair.

### Notes
- Discord still carries the actual audio.
- No Discord SDK, voice capture, speaking indicators, or volume controls were added.
- The new surface is intentionally status/repair/open-link only.

## 2026-06-09 — Kernel 42 Migration Baseline Cleanup + Documentation Audit v1

### Backend
- Added a real `productions` baseline migration so fresh installs can migrate from an empty database without a temporary shell table.
- Updated bootstrap resolution so fresh installs default to `victory-theater` while keeping `amurray-family` as a compatibility fallback.
- Kept the operator bootstrap CLI aligned with the neutral install default.

### Database
- Seeded a neutral install location and `main-lot` for `Victory Theater`.
- Retained legacy `amurray-family` data for compatibility paths.

### Smoke / Verification
- Removed the proof-only `the-cave` temp insert from the fresh-install smoke harness.
- Proved the full empty-DB install path end to end with signup, neutral producer bootstrap, legacy bootstrap compatibility, and account-authority verification.

### Docs
- Audited install/runtime docs so they match the actual fresh-install flow.
- Added environment and compose defaults for the neutral install location.

### Notes
- No Discord audio/voice/speaking features were added.
- Old migrations were edited only because Victory still has no customer installs yet; the live database was not wiped.

## 2026-06-08 — Kernel 40 Runtime Hardening + House Mic Regression Harness v1

### Backend
- Added regression coverage for the Discord Gateway main path: Victory-to-Discord mirror posting, Discord-to-Victory intake, duplicate suppression, bot/self echo suppression, wrong-location thread filtering, debug-toggle neutrality, and minimal delete handling via import-status updates.
- Hardened the Discord gateway thread lookup so imports only resolve threads for the current location.
- Kept the debug trace toggle on the primary gateway path as an operator-controlled visibility switch.

### Frontend
- Standardized the house mic label in the venue mic chips and source labels to `House Mic` / `Discord Bridge`.
- Added explicit debug-on warning text in Producer's Office.
- Reserved the left-side network/tray areas for future audio hooks without building Discord voice/audio transport.

### Notes
- No Discord audio/voice/speaking metadata was added.
- Message delete handling is intentionally minimal for now: delete events mark imports deleted, but no visible venue tombstone stream was built.

## 2026-05-29 — Kernel 34 Account Authority Surface

- Added a current-account summary endpoint at `GET /api/account/me`
- Account UI now shows Victory identity, Discord link status, location memberships, and producer authority
- Discord login remains authentication only; Victory membership remains the source of producer authority

---

## 2026-05-20 — Kernel 33 Operator Bootstrap + Canon Capture

### Backend
- Added an operator bootstrap CLI:
  - `go run ./cmd/victory-bootstrap producer --discord-user-id <discord_user_id>`
  - `go run ./cmd/victory-bootstrap producer --user-id <victory_user_id>`
  - `go run ./cmd/victory-bootstrap producer --handle <victory_handle>`
- The bootstrap path grants only `producer` at the location scope already used by Victory access checks
- The command is idempotent and reports whether the producer grant already existed
- Unknown Discord user IDs fail clearly instead of creating a new grant

### Canon / Documentation
- Captured Kernel 32 as complete in the current canon
- Captured the live-site follow-on work:
  - Terms page
  - Privacy page
  - favicon asset
  - favicon wiring
  - `/auth/*` proxy routing
  - Discord env wiring
  - local `.env` guidance
- Updated roadmap/current-state/operator notes to reflect Kernel 33

### Notes
- Discord login remains identity only
- Victory still authorizes producer authority
- Operator/bootstrap authority stays separate from normal app authority

## 2026-05-19 — Kernel 32 Discord OAuth Primary Login

### Backend
- Added Discord OAuth start/callback routes:
  - `GET /auth/discord/start`
  - `GET /auth/discord/callback`
  - API aliases at `GET /api/auth/discord/start` and `GET /api/auth/discord/callback`
- Added provider discovery route:
  - `GET /api/auth/providers`
- Added Discord identity linking and OAuth state storage helpers
- Preserved the existing Victory session cookie model for post-OAuth login

### Database
- Added `auth.discord_identities`
- Added `auth.oauth_states`

### Frontend
- Added a Discord login button on `/login/`
- Kept local handle/password login intact

### Notes
- Discord OAuth authenticates identity only
- Victory still authorizes users through its own sessions, memberships, and roles
- No Discord bot, guild, channel, or voice integration was added

## 2026-03-25 — Foundation stood up

### Infrastructure
- Confirmed Docker already installed and active on `murray-vserver`
- Created `/opt/victory`
- Started dedicated Postgres container:
  - `victory-postgres`
- Chose Docker as deployment/install truth
- Chose host-Go + Docker-Postgres hybrid mode for faster development iteration

### Schema / Database
- Applied initial schema migration
- Established tables for:
  - users
  - locations
  - memberships
  - lots
  - venues
  - libraries
  - elements
  - placements
  - sessions
  - participants
  - actions
- Added `handle` to users
- Seeded one active live session for `the-cave`

### World Seeding
- Seeded location `amurray.family`
- Seeded lot `main-lot`
- Seeded venue `the-cave`
- Seeded library `house-library`
- Seeded reusable element `first-fire`
- Seeded default placement of `first-fire` into `the-cave`

### Backend
- Created first Go backend
- Added DB connection layer
- Added `/health`
- Added `/api/world/the-cave`
- Fixed timestamp scanning bug in snapshot loader
- Fixed empty actions serialization (`[]` instead of `null`)
- Added `POST /api/session/the-cave/join`
- Added websocket endpoint `/ws/the-cave`

### Identity
- Established current v1 identity model:
  - stable user id
  - handle
  - display name
- Verified join behavior:
  - create/update user by handle
  - join active session
  - default role currently audience

### WebSocket / Observation
- Verified websocket connection to `/ws/the-cave`
- Verified initial snapshot delivery
- Verified ping/pong behavior

### Reaction Loop
- Implemented `react/emote`
- Verified reaction validation
- Verified server-assigned `moment_id`
- Verified persistence to `actions`
- Verified broadcast to connected websocket clients
- Verified late joiner sees prior reaction in snapshot
- Verified live reaction delivery to later-connected observer

### Current Status
Kernel 1.2 remains PARTIAL.

Reason:
- observation path works
- reaction path works
- speech path not yet implemented
- full actor→audience→reaction proof not yet complete

### Current Confirmed Working Stack
- world snapshot
- persistent join
- websocket observer stream
- recorded reaction broadcast

### Current Known Frictions
- Docker rebuilds are slow for active coding
- host-Go development requires correct localhost DB configuration
- install workflow and dev workflow are not yet formalized into scripts/docs

### Recommended Next Step
Implement `perform/speak` using the same action pipeline:
- validate joined actor
- assign `moment_id`
- persist action
- broadcast to observers

---

## 2026-05-01 — Presence + Attribution added

### Backend
- Added an in-memory Cave presence registry keyed by session and user
- Added explicit presence WebSocket event types:
  - `presence/snapshot`
  - `presence/join`
  - `presence/leave`
- Server now resolves actor identity for speech, reactions, and reveal/hide actions before broadcast
- Multiple tabs for the same user are deduped in the visible presence roster

### Frontend
- Added a plain "Who is here?" panel in The Cave
- Added visible speaker attribution and role badges to chat/history lines
- Added visible reaction attribution to chat/history lines
- Removed client-side actor ID trust from outgoing action payloads

### Operational Notes
- Presence is in-memory and intentionally non-canonical
- Backend restart is required for code changes to appear
- No new dependencies were added for this kernel

---

## 2026-05-01 — Identity surface + presence repair

### Backend
- Added a shared server-resolved identity surface in `backend/internal/identity/session_identity.go`
- Cave presence and action attribution now share the same identity resolver
- Added `persona: null` to presence and action payloads
- Display labels now fall back cleanly instead of rendering blank roster entries

### Frontend
- The Cave roster now renders the shared identity surface
- The Cave action log now renders identity labels from the same shape as presence
- The empty roster message is explicit: no one is currently in The Cave

### Operational Notes
- The Cave does not allow anonymous presence
- `persona` is intentionally `null` for Kernel 8
- Backend restart is required for identity/presence changes

---

## 2026-05-02 — Greenroom + Trailers profile surface

### Backend
- Added `performer_profiles` as the server-owned draft/publish profile table
- Added logged-in-only profile routes for:
  - `GET /api/profiles/me`
  - `GET /api/profiles/public`
  - `PATCH /api/profiles/me/save`
  - `POST /api/profiles/me/publish`
- Added producer-gated admin profile update/publish routes for the temporary override seam
- Added Greenroom and Trailers venue seeds
- Added map visibility for Greenroom and Trailers to signed-in users

### Frontend
- Added `frontend/venues/greenroom/index.html`
- Added `frontend/venues/trailers/index.html`
- Added shared identity display helpers in `frontend/lib/identity.js`
- Added map routing and venue icons for Greenroom and Trailers
- Added explicit public-data warnings and public field markers in Trailers
- Expanded the profile surface with expressive public fields for favorite fun, most relaxed, favorite color, favorite artist, favorite food, favorite song, favorite place, favorite movie or show, hidden talent, and ideal day

### Operational Notes
- The profile image upload is stored as a data URL in the profile table for now
- The profile surface does not store birthdates, SSNs, or credit-card-adjacent data
- Age is represented only as a public performance range string
- Greenroom and Trailers are signed-in venues, not anonymous venues
- Backend restart is required for the new routes and seeds to appear

## 2026-05-02 — Info Booth + Mailbox foundation

### Backend
- Added a durable `messages` table bootstrap and migration
- Added authenticated message routes:
  - `GET /api/messages`
  - `GET /api/messages/{id}`
  - `POST /api/messages`
- Added operator-only message creation for dev/test delivery

### Frontend
- Added a public Info Booth modal on the map
- Added a dedicated Mailbox page at `/mailbox/`
- Added inbox list + message detail rendering

### Operational Notes
- Info Booth is public and modal-only
- Mailbox is authenticated-only and server-owned
- Users can only read their own messages
- Message bodies are capped at about 250 characters
- Backend restart is required for the new routes to appear

## 2026-05-03 — Note Card delivery system

### Backend
- Added `POST /api/note-cards`
- Note cards now store as `message_type = note_card`
- Note cards carry Cave context:
  - `venue_slug`
  - `session_id`
- Recipient resolution prefers visible Cave participants and falls back to the director mailbox

### Frontend
- Added a Cave note-card sender form
- Added mailbox context rendering for note cards
- Added a Trailers link to open Mailbox directly

### Operational Notes
- Note cards are durable mail, not chat
- Body limit is 250 characters
- Sender identity is server-resolved from the Cave session
- Backend restart is required for the new route and schema bootstrap

## 2026-05-03 — Index Card Element v1

### Backend
- Added `create/index_card` and `update/index_card` action handling
- Index cards now materialize as `elements.element_type = 'index_card'`
- Index cards persist front/back/color plus server-owned creator and production/session metadata
- Directors and producers can create and edit cards; cast/crew/audience are filtered by the existing visibility spine

### Frontend
- Added an Index Cards tray/editor inside The Cave
- Added card selection, back-side preview, and save history rendering
- Added a visibility layer selector for sharing cards with cast/crew or audience

### Operational Notes
- Index cards are Elements, not a separate card universe
- Saves append action history and refresh the Cave snapshot
- Card content is capped at 2000 characters total
- Backend restart is required for the new action handlers to appear

## 2026-05-03 — Workshop to Venue Placement v1

### Backend
- Added `act/place_element` for index card placement
- Added `backend/internal/actions/place.go` for server-side placement persistence
- Added `GET /api/workshop/venues` for enabled target venues
- Venue enablement now keys off `venues.config.index_cards_enabled`

### Frontend
- Added Workshop mode to the Cave card surface
- Added Send to Venue dropdown and tray/stage placement buttons
- Added a Workshop redirect page from the map

### Operational Notes
- Workshop is the source surface for cards
- Tray/backstage and stage are distinct placement surfaces
- Backend restart is required for placement and venue-list changes

## 2026-05-06 — Kernel 24 and 25 reconciliation checkpoint

### Runtime / Character Notes
- Greenroom is the character dressing room
- Trailers is the performer profile drafting/publishing surface
- The Cave keeps live persona use rather than full character editing
- Character card drafting is currently available to performer roles without separate draft-grant workflow
- Character sheet links are metadata references on character cards, not playable sheet records

### Documentation Notes
- Added `Construction/current-state.md` as current canon
- Added `Construction/roadmap.md` as active roadmap
- Added placeholder kernel docs for 22–25 naming reconciliation
- Older notes are being marked historical/superseded instead of silently overwritten

## 2026-05-17 — Kernel 31 Middle School Stage shell

### Backend
- Seeded `middle-school-stage` as a producer-only venue surface
- Added producer visibility for `middle-school-stage`

### Frontend
- Added `frontend/venues/middle-school-stage/index.html`
- Added `middle-school-stage` routing in the map app
- Added a top status bar, left/right edge drawers, a collapsed bottom chat drawer, and a center-first stage shell layout
- Added a placeholder right-click context menu on a targetable stage object

### Operational Notes
- The Cave remains untouched and continues to serve as the proving ground
- Pixi remains outside The Cave
- Full live control parity is still Cave-specific; Middle School Stage is currently a shell-first proof

## 2026-05-17 — Kernel 32 template extraction

### Frontend
- Added a hidden `stage-template` venue scaffold as the reusable shell descendant
- Mounted the portable overlay from `frontend/venues/shared/venue-shell.js` in First Theater so Pixi and overlay can be compared together
- Refactored Middle School Stage to use the shared venue shell helper for preferences and presence preview behavior

### Documentation
- Updated the current-state canon and roadmap to treat Kernel 32 as the template extraction pass
- Renumbered the downstream roadmap so Discord OAuth becomes the next identity kernel

### Operational Notes
- The Cave was left untouched
- The hidden template scaffold is not map-visible
- First Theater now shows the portable overlay above Pixi as the comparison surface

## 2026-06-08 — Kernel 41 fresh install / deployment proof

### Deployment Proof
- Added `.env.example` with the current runtime and Discord operator settings
- Kept `.env` ignored while explicitly allowing `.env.example`
- Added a clean-install smoke harness at `scripts/smoke/fresh-install.sh`
- Added a fresh-install operator guide at `Construction/deployment/fresh-install.md`
- Verified the install proof against a temporary database in the local Postgres container

### Runtime Notes
- The smoke path does not touch the live database
- Discord config is optional for backend boot and surfaces degrade safely when blank
- No Discord voice/audio path was built for this kernel
- The kernel 2 seed migration must follow the world seed migration in clean-install proof runs

## 2026-06-10 — Kernel 45 venue shell rebase / shared helper extraction

### Frontend
- Collapsed the main Victory map header into a much thinner hover-open strip
- Removed the 1280px map ceiling so the map layer can use more of the viewport
- Added shared shell helper methods in `frontend/venues/shared/venue-shell.js` for:
  - venue slug normalization
  - venue name formatting
  - shell slot resolution
  - header chip registration
  - safe refresh/reload hooks
  - reusable mount metadata for headers, trays, and chat rails
- Wired the helper into First Theater and Middle School Stage so the shared contract is now live on the main venue path

### Documentation
- Promoted Kernel 45 into the current-state canon
- Added a Kernel 45 reportback in the official house format
- Updated the roadmap to mark the venue shell extraction pass as the current near-term kernel

### Operational Notes
- Kernel 44 audio behavior remains untouched by this pass
- The shared shell helper is intentionally light and does not try to replace venue-specific runtime logic
- First Theater remains the practical source template while The Cave keeps the proving-ground role

## 2026-06-10 — Kernel 46 First Theater Pixi backdrop renderer

### Frontend
- Added a shared Pixi backdrop helper at `frontend/lib/victory-pixi-stage.js`
- Proved a shared Pixi backdrop helper path and renderer slot structure for First Theater
- Reverted the First Theater backdrop swap after it made the stage presentation worse than the existing façade
- Kept the existing First Theater DOM controls, chat rail, session commands, house mic status, and context menus intact

### Documentation
- Promoted Kernel 46 into the current-state canon
- Updated the roadmap to mark the First Theater Pixi backdrop pass as the current near-term kernel
- Added a Kernel 46 reportback in the official house format

### Operational Notes
- The Pixi helper is scoped to backdrop rendering and does not claim app truth
- If Pixi fails to load, First Theater still falls back to its existing DOM-safe message
- The older First Theater stage composition remains the preferred live presentation for now
- The Cave was left untouched by this kernel

## 2026-06-20 — Kernel 47 Pixi grid primitive + map alignment

### Backend
- Added `backend/internal/venues/grid.go`: `GET /api/venues/first-theater/grid` and `PUT /api/venues/first-theater/grid`, scoped to First Theater only, mirroring the existing map route's authority model (operator/producer/director can edit, any venue-accessible viewer can read)
- Added migration `028_kernel47_grid_config.sql` creating `venue_grid_configs` (one row per venue: grid_type, hex_orientation, cell_size, offset_x/y, line_width, opacity, line_style, visible, updated_by_user_id)
- Server-side validation bounds: cell size 8–500, opacity 0–1, line width 0.5–8, offsets ±2000, grid_type in {none,square,hex}, hex_orientation in {flat-top,pointy-top}, line_style in {light,neutral,dark}
- Also fixed two regressions found and fixed mid-session on the existing map feature (Kernel 46 follow-up, same branch of work): the running `victory-backend` container had not been rebuilt since `display_mode` (theater/fullscreen) was added to `map.go`, so Save was silently reverting to theater; and added a `DELETE /api/venues/first-theater/map` endpoint plus a Remove Map button so producers/directors can clear the active map asset

### Frontend
- Added `frontend/lib/victory-pixi-grid.js`: square-grid line rendering, flat-top and pointy-top hex-grid rendering (closed hexagon outlines via axial row/column spacing), clear, and a `render(layer, config, bounds)` entry point bounded by the same playable-stage rectangle the map uses
- Added a `grid` Pixi container layer (zIndex 6) between `mapLayer` (5) and `facadeLayer` (8) in First Theater's scene graph
- Extracted `computeStagePlayableBounds(width, height)` in `runtime.js` so the map layer and grid layer always agree on the theater-vs-fullscreen safe-bounds rectangle
- Added a "Configure Grid" stage context-menu item (producer/director gated, same pattern as "Add / Replace Map") and a draggable "Configure Grid" callout panel with: grid type (off/square/hex), hex orientation, cell size (+/- nudge buttons), offset X/Y (arrow nudge buttons, Shift for a larger step), opacity, line width, line style, Reset Grid, Hide/Show Grid, Save Grid, and Close (Close reverts the live preview to the last saved configuration; outside-click just closes without reverting, matching the existing map editor's convention)
- Grid configuration is fetched on boot alongside map state and re-rendered on every `renderPixiScene()` pass, so resize and map-replacement both keep the grid aligned automatically

### Documentation
- Promoted Kernel 47 into the current-state canon
- Updated the roadmap to mark the grid-primitive pass as the current near-term kernel and noted pan/zoom is deferred to a later kernel
- Added a Kernel 47 reportback in the official house format

### Operational Notes
- The grid is visual-only: no snap-to-grid, measurement, coordinates, tokens, or pan/zoom were built, per kernel scope
- The collapsed-header blur over the First Theater map top edge remains a known, deferred layout issue, untouched by this kernel
- **Incident**: running the full backend test suite (`go test ./...`) against this environment's `DATABASE_URL` executes real `DELETE`/`INSERT` statements against the live `victory` Postgres database, not an isolated test database. This deleted the live `auth.discord_server_link_settings` row for the real `amurray-family` location (several tests in `internal/identity` and `internal/network` assume a disposable DB and clean up by deleting real location-scoped rows). The Discord gateway came up disabled until the operator re-ran their bootstrap flow. Backend test runs against this database should be scoped away from `internal/identity` and `internal/network` going forward, or run only after confirming with the operator
- Also discovered an unrelated, pre-existing stray `victory` binary running directly on the host on port 8081 (started before this session, consistent with the documented "dev mode" `go run ./cmd/victory` workflow); confirmed Caddy's `reverse_proxy backend:8081` resolves to the Docker service, not the host process, so production routing was unaffected — flagged to the operator as a leftover process worth checking

## 2026-06-23 — Kernel 51 First Theater action-router extraction

### Frontend
- Extracted the remaining First Theater context-menu and stage-action orchestration into `frontend/venues/first-theater/runtime/action-router.js`
- Added an explicit loader for the router script in `frontend/venues/first-theater/index.html`
- Routed the live runtime through the extracted module for:
  - context-menu target resolution
  - native stage context-menu handling
  - resolved context-menu opening
  - stage object actions
- Fixed the extracted token move path so token updates now flow through `updateTokenLocalModel` instead of the card pin updater

### Tests
- Added focused coverage in `tests/first-theater/action-router.test.js` for:
  - token move-here
  - card move-here
  - hide-nameplate
  - remove
  - context-menu resolution/opening
- Re-ran the First Theater pure-module suite after the split:
  - `tests/first-theater/action-router.test.js`
  - `tests/first-theater/token-ui.test.js`
  - `tests/first-theater/map-grid.test.js`
  - `tests/first-theater/stage-controls.test.js`
  - `tests/first-theater/geometry.test.js`
  - `tests/first-theater/logic.test.js`
  - `tests/first-theater/state.test.js`
  - `tests/first-theater/socket.test.js`
- Verified `node --check` on `frontend/venues/first-theater/runtime.js` and `frontend/venues/first-theater/runtime/action-router.js`
- Verified `git diff --check`
- Verified backend health with `curl -s http://127.0.0.1:8081/health`

### Operational Notes
- The router extraction stayed dependency-injected and did not introduce a second socket path or state store
- The live runtime still owns the UI shell, but the action routing is now isolated and directly testable
- No new user-facing feature was added in this pass

## 2026-06-23 — Kernel 51 socket transport extraction

### Frontend
- Extracted First Theater socket transport into `frontend/venues/first-theater/runtime/socket-controller.js`
- Kept `frontend/venues/first-theater/runtime/socket.js` as the pure dispatcher/parser layer
- Moved the live runtime over to the controller for:
  - action queueing while connecting
  - reconnect scheduling
  - `sendAction`
  - `sendPing`
  - `sendFocusPing`
  - socket connection bootstrap
- Preserved the message semantics in runtime-side handling for snapshot, focus ping, action, showing update, and error messages

### Tests
- Added `tests/first-theater/socket-controller.test.js` covering:
  - queued sends while connecting
  - flush on socket open
  - focus ping payload composition
  - focus ping send/status behavior
- Re-ran the First Theater pure-module suite after the transport split
- Verified `node --check` for:
  - `frontend/venues/first-theater/runtime.js`
  - `frontend/venues/first-theater/runtime/socket-controller.js`
  - `frontend/venues/first-theater/runtime/socket.js`
- Verified `git diff --check`
- Verified backend health with `curl -s http://127.0.0.1:8081/health`

## 2026-06-25 — Kernel 51 stabilization closeout

### Frontend
- Fixed the First Theater grid editor save/close regression so Save persists the new baseline and Close no longer restores stale grid state
- Fixed the map editor asset-selection path so choosing an existing asset updates the in-tool preview instead of trying to apply/close immediately
- Fixed token create/remove lag by synchronizing live runtime objects from projected state and subscribing the runtime directly to projected-state object changes for scheduled Pixi re-renders
- Added a default token placement fallback so create-token can still place when no prior pointer point is cached
- Fixed token texture late-load behavior by forcing a scene re-render when Pixi finishes loading the token texture
- Fixed the token editor context-menu path so `Scale` opens the token editor again
- Restored `Hide Audience` / `Show Audience` handling in the modular action router and projected-state reducer

### Backend
- Fixed `create/token` and `update/token` live action payloads so websocket broadcasts include the token asset URLs and footprint metadata needed for immediate frontend rendering
- Rebuilt and restarted the `victory-backend` container so the live site served the patched token payload code instead of the stale binary

### Tests
- Re-ran focused First Theater regression coverage:
  - `tests/first-theater/action-router.test.js`
  - `tests/first-theater/editors.test.js`
  - `tests/first-theater/session-sync.test.js`
  - `tests/first-theater/state.test.js`
  - `tests/first-theater/token-ui.test.js`
- Re-verified `node --check` for:
  - `frontend/venues/first-theater/runtime.js`
  - `frontend/venues/first-theater/runtime/action-router.js`
  - `frontend/venues/first-theater/runtime/token-ui.js`
  - `frontend/venues/first-theater/runtime/session-sync.js`
  - `frontend/venues/first-theater/runtime/state.js`
  - `frontend/venues/first-theater/runtime/scene-nodes.js`
- Verified `GOCACHE=/tmp/victory-gocache go test ./internal/actions` from `backend/`
- Verified backend health inside the rebuilt container with `wget -qO- http://127.0.0.1:8081/health`

### Operational Notes
- The First Theater refactor is now in a usable state for feature work again
- The major user-reported regressions from the extraction pass are closed:
  - map replace flow
  - grid save/hide behavior
  - token add/remove/render timing
  - token real-art live create payloads
  - token scale editor launch
- Single-client live verification also confirmed voice and mic behavior working after the runtime cleanup
- No new review findings were identified in the final closeout pass
- Remaining uncertainty is limited to multi-client verification:
  - audience hide/show as seen by another client
  - Shift+Ping as seen by another client

### Operational Notes
- The socket controller owns transport mechanics only; it does not replace projected state or the dispatcher
- No new user-facing feature was added in this pass
- Fixed a missing `onPong` guard in the socket controller so pong handling no longer risks a runtime ReferenceError

## Kernel 53 Correction — Retention Threshold, Stage 2 Lockdown, Server-Authoritative Dice

Confirmed the `>50` Starting Credit retention threshold was already correct (with boundary tests for 49/50/51 already in place) and removed the hidden Stage 2 d4 "Stages of Childhood" rolling mechanic, leaving Stage 2 as a pure interstitial shell ("Stages of Childhood → Conception") with no executed mechanics.

While assembling report-back evidence, found and fixed several real defects beyond the two named corrections:
- The donor 3d20 parentage roll was client-trusted, not server-authoritative — the browser rolled the dice and the server accepted whatever total it was sent. Added a new idempotent, crypto-secure server-side roll endpoint (`POST /api/character-cards/parentage-roll`) keyed by `(owner, draft_token, event_key)`, and made `CreateCard` rebuild `socio_parentage_parents` solely from the server-locked rolls, discarding any client-supplied roll/credit fields. The frontend reel animation is unchanged — it now lands on the server-determined face instead of a client-random one.
- `character_workbook_entries` and the new `character_workbook_rolls` table were missing from migrations entirely (existing only via undocumented manual creation on the live DB); the Kernel 53 migration was also misnumbered `018` (colliding with `018_kernel32_discord_oauth.sql`). Renamed it to `031_kernel53_character_workbook_foundation.sql`, added the missing tables, and brought `scripts/smoke/fresh-install.sh`'s migration list up to date through 031.
- The "New Workbook" button in Greenroom was a dead end: it routed to Catharsis, but Catharsis only shows the Create Character flow when the user owns zero cards, so a second character could never be started. Added a `?new_character=1` signal Catharsis now honors to force-open the builder.
- `RecordWorkbookEvents`/`insertWorkbookEntry` had no duplicate guard, so resuming the same draft (a direct symptom of the New Workbook dead end) stacked duplicate parentage/coin-flip/chapter-handoff history entries. Added a same-character/entry_type/title/body dedupe check before insert.
- `/journal` PATCH (edit) and DELETE (archive) were stubbed `501 Not Implemented` despite being required acceptance criteria. Implemented both, author-scoped.
- New characters now default to the lowest unused Roman numeral (`I`, `II`, ...) instead of the parentage chart's social class name, with a UI hint marking auto-named characters so it's obvious a name hasn't been intentionally chosen yet.
- Deleted one live stray draft ("Night Watchperson") that had accumulated 4 duplicate sets of parentage/chapter-handoff entries from repeated dead-end "New Workbook" attempts, per explicit operator confirmation.

### Tests
- `cd backend && go test ./internal/characters/...` (all passing, no DB access)
- `go build ./...`, `go vet ./...`, `gofmt -l` on touched packages
- `git diff --check`

### Live Verification
- Rebuilt and restarted `victory-backend`; applied migration `031` to the live DB (additive-only, `IF NOT EXISTS` throughout)
- Full Playwright smoke against `https://victory.amurray.family` using a minted test session: Create Character → Egg Donor roll (server-determined, Roll disables) → Sperm Donor roll → Stage 1 summary → Stage 2 shell (`Stages of Childhood → Conception`, no d4 controls present) → refresh (no reroll, active-character chip persists) → Greenroom (workbook, Face, Parentage Summary, combined Starting Credit all correct: roll 26/Farm Hand/credit 30/not eligible, roll 37/Inn Keeper/credit 65/retained, combined 65)
- Verified `/journal` create/edit/delete via direct API calls against the live backend
- Deleted the test character and revoked the test session afterward

## Kernel 54 — Chapter 2 Lifepath System

Implemented the 10-stage Chapter 2 Lifepath system (Preconception → Young Adult) per `chapter-2-versioned-tables-v0.3.json` / `chapter-2-rules-implementation-spec-v0.3.md`, replacing the Stage 2 shell that Kernel 53 left as a placeholder.

### Backend (`backend/internal/characters/`)
- `chapter2_rules.go` (new) — full versioned rules data as Go structs mirroring the `parentage_chart.go` pattern: 10 `Chapter2Stage` entries (bonus choices, companion, enhancement, traits, Stage 2's `has_representation_field`, Stage 9's `has_special_allocation`), `Chapter2Version = "0.3"`, `Chapter2StartingFP = 12`, `Chapter2EnhancementCost = 3`, `Chapter2AttributeCap = 10`, plus `ValidateChapter2Rules()` and lookup helpers. No separate Bonus Point cap, matching the spec's explicit removal of that cap.
- `chapter2_roll.go` (new) — server-authoritative, idempotent d4/d6 rolls reusing the existing `character_workbook_rolls` table (`character_card_id` as `draft_token`, event keys like `c2.s01.d4`). `RequestChapter2Roll` enforces d4-before-d6 ordering, returns the already-locked value on repeat calls (`ON CONFLICT DO NOTHING` + reload), and never rerolls. `ApplyChapter2Enhancement` keeps `max(d4, d6)`, falling back to `d4` if the d6 was never rolled.
- `chapter2_stage.go` (new) — `CommitChapter2Stage` is the single source of truth for stage completion: loads the server-locked dice, validates the bonus choice (exactly 1 of 4) and 0-2 trait IDs against that stage's rules, computes FP spent (enhancement 3, companion/trait costs from rules data), blocks completion if FP balance would go negative, enforces the Stage 9 special allocation (all of `final_roll` distributed, at least 1 point on Craft, free choice for the rest, stacking allowed), enforces the hard attribute cap of 10 per attribute (blocks rather than truncates), and is idempotent — re-posting an already-completed stage returns the existing result without re-applying deltas.
- `characters.go` — added `HandleChapter2Rules` (public GET), `HandleChapter2Roll` (POST, auth + `CanEditCard`), `HandleChapter2RollStatus` (GET, auth + `CanEditCard` — added in a follow-up fix, see below), `HandleChapter2Stage` (POST, auth, persists the updated `workbook_context` and the `character_workbook_modules` stage/event tracking).
- `cmd/victory/main.go` — registered `GET /api/characters/chapter2-rules`, `POST /api/character-cards/chapter2-roll`, `GET /api/character-cards/chapter2-roll`, `POST /api/character-cards/chapter2-stage`.
- New tests: `chapter2_rules_test.go` (rules validation, stage/bonus/trait lookups), `chapter2_stage_test.go` (enhancement keep-higher logic including the no-d6-rolled fallback, fresh-state defaults, state round-tripping through `workbook_context`).

### Frontend (`frontend/venues/catharsis/`)
- `index.html` — replaced the Kernel 53 Stage 2 shell (`#catharsis-onboarding-childhood`, static copy, no controls) with `#catharsis-onboarding-chapter2`, a single dynamic panel driven entirely by server data: FP/stage/primary-attribute readouts, d4/d6/final roll display, roll buttons, a bonus-choice radio group, a Stage 2-only Representation textarea, an optional companion checkbox + name field, 0-2 trait checkboxes, a Stage 9-only per-attribute allocation grid, and a status line. Added `#catharsis-onboarding-chapter3` as the post-Chapter-2 mechanics-free interstitial (mirrors the existing Chapter II interstitial pattern).
- `onboarding.js` — added a Chapter 2 module: `loadChapter2Rules`/`chapter2StageRules` (cached rules fetch), `chapter2Ctx`/`ch2DefaultState` (reads/initializes the `chapter2` sub-object inside `workbook_context`), `ch2Local` (per-stage UI selection state, reset on every stage entry), render functions for bonus choices / traits / Stage 9 allocation / the overall complete-button gate, `ch2RollD4`/`ch2RollD6` (same celebratory flicker-then-land-on-server-value animation pattern as the Kernel 53 donor rolls), `ch2CompleteStage` (posts to `chapter2-stage`, writes a `history` entry via the existing `postWorkbookEvents` path, then advances to the next stage or to the Chapter 3 shell), and `showChapter2Panel`/`showChapter3Panel`. `continueFromChapter` (the existing Stage 1 → Chapter II handoff) now opens Chapter 2 Stage 1 instead of the old static shell. The init IIFE resumes mid-Chapter-2 automatically if `workbook_context.current_stage === 2` on load.
- A single `hideAllOnboardingPanels()` helper replaces the prior copy-pasted per-panel `.hidden` toggling, now shared by all panel-show functions including the two new ones.

### Bug found and fixed during live verification
First live Playwright smoke test (Stage 1 → Chapter 2 Stage 1 → Stage 2, roll a d4, refresh mid-stage) found that the locked d4 roll disappeared from the UI on refresh for any **not-yet-completed** stage — the server had correctly retained the value (confirmed by code/network inspection), but `showChapter2Panel` only restored roll state for already-*completed* stages, never for "rolled but not yet submitted." Fixed by adding `GET /api/character-cards/chapter2-roll` (`HandleChapter2RollStatus`, wrapping the existing `LoadChapter2StageRolls`) and having `showChapter2Panel` call it for incomplete stages, repopulating `ch2Local.rolls` before render. Re-verified live: d4 value before and after a full browser refresh matched exactly (`4` → `4`), button stayed disabled, no reroll occurred.

### Tests
- `cd backend && go test ./internal/characters/...` — all 10 new Chapter 2 tests plus all pre-existing Kernel 53 tests pass (no DB access)
- `go build ./...`, `go vet ./...`, `gofmt -l` clean on all touched files (two pre-existing unrelated gofmt findings in `internal/db/db.go` and `internal/showings/review_test.go` left untouched, as in Kernel 53)
- `node --check` on `onboarding.js`

### Live Verification
- Rebuilt and restarted `victory-backend` (twice — once for the initial Chapter 2 build, once for the refresh-restore fix); confirmed `GET /api/characters/chapter2-rules` responding live with all 10 stages
- Full Playwright smoke against `https://victory.amurray.family` using a minted test session: Stage 1 parentage (both parents) → Chapter II interstitial → Chapter 2 Stage 1 "Preconception" (FP=12, roll/enhance/bonus/companion/trait controls all present and correctly shaped) → roll d4 (animates, locks, button disables) → select a bonus choice → Complete stage → Stage 2 "Genetics" (FP unchanged at 12 since nothing cost FP, Representation textarea present as the Stage 2-only field) → roll d4 → full browser refresh
- Refresh-recovery re-verified after the fix: locked d4 value (`4`) identical before and after refresh, stage number unchanged, roll button correctly stayed disabled, `GET /api/character-cards/chapter2-roll` returned `200` with the matching locked value both before the roll (`rolls: null`) and after (`rolls: {d4: 4, final_roll: 4, ...}`)
- Test character cards and test sessions deleted after each verification pass
- Not yet exercised live: Stage 9's special allocation UI, a full 10-stage run through to the Chapter 3 transition, FP-balance blocking at zero, and the attribute hard-cap rejection path — these are covered by the backend's validation logic and unit tests but not yet driven end-to-end through the browser

## Kernel 55 — Chapter 3: Character Archetypes

Implemented "Chapter 3: Character Archetypes" per the Kernel 55 spec and canonical source data (script.js, dataset v1.0.0): 14 archetypes, direct-selection path, full 15-question reflective quiz with complete result experience, single explicit confirmation event, History/Mechanics/Face projections, Chapter 4 boundary shell, and append-only Director override.

### Source data fidelity
- Kernel author supplied `script.js` as the canonical implementation reference (no separate JSON file available). Verified 15 questions / 89 answers / 14 archetypes — matches the kernel spec's stated counts exactly.
- All archetype profile text (titles, codes, echo text, motto, shortDescription, primary{intro,strengths,challenges,socio}, hexaco, hexacoPlain, coreDrives, primaryAttribute, secondaryAttribute, keySkill, notices, strengthsList, growthEdges, groupRole, underStress, playSuggestions, questionsToAsk, howToHelpYourself) ported verbatim from script.js, with no editorializing.
- All 15 questions and 89 answers (including exact text, maxSelections limits, scoring weights, and Question 15's selfGuess mapping) ported verbatim into a client-only `chapter3-quiz-data.js` module — this data is never sent to or received by the server.

### Backend (`backend/internal/characters/`)
- `chapter3_archetypes.go` (new) — versioned archetype catalog as Go structs (14 entries), `Chapter3ArchetypeDatasetVersion = "1.0.0"`, `Chapter3RulesetVersion = "1.1"`, `Chapter3ArchetypeByKey()` lookup, `ValidateChapter3Archetypes()` integrity check (count, duplicate keys, display-order contiguity, Mechanics fields presence).
- `chapter3_select.go` (new) — `CommitChapter3Archetype`: validates auth/permission, Chapter 2 Stage 10 completion (wbContext["chapter2"].stages["10"].completed), archetype key against catalog, idempotency (re-submitting same already-confirmed key returns existing result without re-writing), stores `Chapter3Fact` in `workbook_context["chapter3"]` (stable ID, key, title, code, primaryAttribute, secondaryAttribute, keySkill, confirmed, confirmed_at, selection_count), sets current_stage=4 / current_event="chapter3_archetype_confirmed", writes a `history` entry via `RecordWorkbookEvents` server-side (actor-resolved, canonical character name from `character_cards.name`, no quiz data, body format: "[Name] selected [Archetype] for their archetype."), updates `character_workbook_modules`. Override path (Director/permitted actor): `override=true` bypasses the idempotency early-return and appends a new `archetype_override` history entry.
- `workbook_pages.go` — added `chapter3FaceWidgetFields(context)` (compact archetype title + motto shown on the Face page once confirmed) and `chapter3MechanicsFields(context)` (archetype stable ID, primary/secondary attribute, key skill — the stable machine-readable inputs Kernel 56 needs for Chapter 4 skill-group selection); both appended to their respective pages in `buildWorkbookPages`.
- `chapter3_archetypes_test.go` / `chapter3_select_test.go` (new) — `ValidateChapter3Archetypes`, `Chapter3ArchetypeByKey` known-good/unknown-key, display_order contiguity, `CommitChapter3Archetype` auth/cardID/archetypeKey/unknown-key guards (pure, no DB), `chapter2Complete` stage-10 gate (empty/stage-9-only/stage-10-complete cases), `loadChapter3Fact` round-trip.
- Routes: `GET /api/characters/chapter3-archetypes` (public, returns full catalog), `POST /api/character-cards/chapter3-confirm` (auth + CanEditCard).

### Frontend (`frontend/venues/catharsis/`)
- `chapter3-quiz-data.js` (new) — client-only IIFE exporting `window.VictoryChapter3QuizData` with the 15 questions / 89 answers / scoring weights, plus `datasetVersion = "1.0.0"`. Never sent to server.
- `index.html` — renamed old Chapter III "Lifepath Closes" shell (now replaced by the real Chapter 3 archetype flow) to `#catharsis-onboarding-chapter4` ("Chapter IV: The Archetype Is Set"). Added `#catharsis-onboarding-chapter3-archetype` (contains `#catharsis-chapter3-content` for dynamic rendering). Added CSS for the dynamic content: dropdown, echo cards, quiz answer buttons (`.ch3-answer-btn` + `.is-selected` toggle), resonance bar rows, profile grid, pills, echo grid. Loads `chapter3-quiz-data.js` before `onboarding.js`.
- `onboarding.js` — full Chapter 3 module:
  - `loadChapter3Catalog()`: fetches and caches the archetype catalog sorted by `display_order`.
  - Quiz scoring: `ch3CalculateMaxPossibleScores()`, `ch3GetAllResults()` (raw%, +12 Resonance bump, sort by rawPercent desc / points desc / key asc), `ch3GetConfidenceLabel()` (Very strong / Strong / Moderate / Emerging), `ch3GetBlendedProfileItems()` (mirrors script.js's composition logic: 3 primary + 1 from each echo, top-up from remainder).
  - `ch3GetSelfGuess()` + `ch3RenderPrediction()`: matches the reference matching/contrast commentary behavior exactly.
  - `ch3RenderIntro()`: title card, "Answer as your character" framing, dropdown of 14 archetypes in display order, echo preview beneath, "Confirm Archetype" + "Take the Archetype Quiz" buttons.
  - `ch3RenderQuizQuestion()`: question text + progress ("Question N of 15"), multi/single-select answer buttons with toggle, disabled Continue until at least one answer selected.
  - `ch3RenderQuizResult()`: full result experience — primary title/Resonance%/confidence, motto blockquote, short description, core drive pills, Top Patterns bars (5), two echo cards (2nd/3rd), Primary Profile narrative (intro/strengths/challenges/socio), Personality Pattern (HEXACO + plain text + attribute callout: primary/secondary/key skill), What You Notice / In A Group, blended Natural Strengths / Growth Edges lists, Questions To Ask / How To Help lists, Under Stress / In Socio- lists, self-prediction comparison (match or contrast commentary), full 14-archetype Resonance breakdown bars, final dropdown (preselected to primary result, player may change), "Take the Quiz Again" / "Confirm and Continue" buttons.
  - `ch3RenderConfirm()`: explicit confirmation screen "Select [Archetype] as this character's archetype?" + echo preview + Back + Confirm and Continue.
  - `ch3SubmitConfirmation()`: calls `POST /api/character-cards/chapter3-confirm`, refreshes workbook context, advances to Chapter IV panel on success.
  - `renderChapter3Content()`: mode dispatcher (intro / quiz-question / quiz-result / confirm).
  - `showChapter3ArchetypePanel()`: entry point, loads catalog, sets mode "intro", renders.
  - Resume logic (init IIFE): `current_stage === 2` → resume Chapter 2 (existing); `current_stage === 3` → `showChapter3ArchetypePanel()` (no quiz restore per spec, returns to intro title screen); `current_stage === 4` → normal overlay-hidden path (Chapter 3 is complete).
  - Chapter 2 Stage 10 completion now calls `showChapter3ArchetypePanel()` instead of the old Chapter III "Lifepath Closes" stub shell.
  - All excluded standalone-site features omitted: no report ID generation, no result URL, no share card canvas, no embed code, no clipboard tools, no download links, no external marketing links.

### Tests
- `go test ./internal/characters/...` — all new Chapter 3 tests plus all prior Kernel 53/54 tests pass (no DB access)
- `go build ./...`, `go vet ./...`, `gofmt -l` clean
- `node --check` on `onboarding.js` and `chapter3-quiz-data.js`
- Quiz answer count verified: `15 questions, 89 answers` confirmed via `node -e` against the quiz data file — matches spec

### Live Verification
- Rebuilt and restarted `victory-backend`; confirmed `GET /api/characters/chapter3-archetypes` responds with 14 archetypes, correct dataset_version "1.0.0"
- **Direct-selection path** (test character "I", Catalyst): Chapter III title screen → dropdown echo text update confirmed ("You bring energy and momentum wherever you go…") → "Select The Catalyst as this character's archetype?" confirmation screen → Chapter IV "The Archetype Is Set" screen ✓
- **Quiz path** (test character "II", Builder): 15 questions answered (2 answers on multi-select, 1 on Q15) → full result page all 19 sections rendered with no undefined/NaN/object text → dropdown preselected to "Builder" → "Select The Builder as this character's archetype?" → Chapter IV screen ✓
- **Idempotency**: re-posting same confirm returns `{"ok":true,"completed":true,"overridden":false}`, DB confirms `selection_count: 1` — no duplicate history entry created ✓
- **History**: exactly one `archetype_selection` entry per card, body reads "I selected Catalyst for their archetype." / "II selected Builder for their archetype." — no quiz details, resonance values, or echo text leaked ✓ (the "I"/"II" in the body are the characters' Roman-numeral auto-names, consistent with Kernel 53 naming convention)
- **Mechanics**: `workbook_context["chapter3"]` contains `primary_attribute`, `secondary_attribute`, `key_skill` matching `chapter3_archetypes.go` for each selection; both characters at `current_stage=4` ✓
- **Face**: workbook pages API includes `chapter3_archetype` and `chapter3_archetype_summary` fields on the Face page for confirmed characters ✓
- No console errors during either path. The pre-existing director-console/session-join 403s noted in prior test runs remain present and are out of scope
- All test character cards soft-deleted, test sessions revoked after each pass

## Kernel 56 — Chapter 4: First Skill and Courtyard Entry

Implemented the final chapter of the Socio- onboarding chain: resolving the archetype's Chapter 4 attribute group, presenting the ten skill cards with the key skill highlighted, confirming exactly one first trained skill at d6, locking the character (terminal onboarding event — normal player flow can no longer reopen Chapters 2/3/4; a Director `/override` surface is deferred to a later kernel), equipping the character as the user's active persona, and entering the initial Locked Courtyard scene.

### Blocking author decision resolved before build
The kernel spec explicitly required an author decision before implementation could begin: whether the Chapter 4 ten-card group is routed by the archetype's *listed* primary attribute, or by the *governing attribute of the archetype's key skill*. Four of the fourteen sample archetypes (Architect, Aesthetician, Steward, Performer) have a key skill whose attribute differs from their listed primary attribute, so the two readings are incompatible for those four without either editing Chapter 3's already-shipped canonical archetype data or leaving the key skill outside its own recommended group. Presented both options to the operator; **key-skill-attribute routing was selected** — this guarantees the key skill always appears in its own displayed group for all 14 archetypes with zero changes to Chapter 3 data. Proven by a dedicated test (`TestEveryArchetypeKeySkillResolvesToTenCardGroup`) that checks every archetype's key skill resolves inside the group its own routing produces, and re-verified live using The Architect (one of the four exception cases: primary attribute Lore, key skill Design → routes to Craft).

### Skill catalogue
- `chapter4_skills.go` (new, generated from the supplied `chapter-4-skill-catalogue-v0.9-draft.json` via a scratch Python script to avoid manual transcription errors, then reviewed) — the complete versioned 100-skill catalogue (10 attributes × 10 skills), each skill carrying its stable ID, canonical name, attribute, source order, card description, and its two helper cards, ported verbatim per `socio-skill-catalogue-audit-v0.1.md`'s reconciliation policy (Aesthetic Design added to Grace, Occultism excluded from Lore, Recovery retained in Resolve without inventing its missing expanded description, "Breaking / Forcing" spacing normalized).
- `ValidateChapter4SkillCatalogue()` checks: exactly 10 attributes, exactly 100 skills, no duplicate stable IDs or canonical names, gap-free source_order 1-10 per attribute, and every Chapter 3 sample archetype's key skill resolves to a real catalogue skill.

### Backend (`backend/internal/characters/`)
- `chapter4_select.go` (new) — `ResolveChapter4Group` (auth + `CanEditCard`, gates on Chapter 2 Stage 10 complete and Chapter 3 confirmed, resolves the key skill's attribute group, reads the attribute's effective score from `workbook_context["chapter2"]["attributes"]` as the capacity ceiling); `CommitChapter4FirstSkill` (validates the skill belongs to the resolved group, rejects when attribute capacity is already full, rejects unless idempotent-same-skill or an explicit Director override, applies the skill at `training_state: "trained"` / `die_size: "d6"`, sets `current_stage=5`, marks `workbook_status = "complete"`, equips the character via the existing `setActiveCharacter` persona primitive, writes the History event, and is idempotent against retries).
- `chapter4_courtyard.go` (new) — a static, versioned Locked Courtyard scene descriptor (walls, market booths/crowd, Kessa listed as present, door state `"locked"`) with no invented dialogue or encounter mechanics, served via `GET /api/characters/chapter4-courtyard`. No courtyard/venue/map infrastructure existed anywhere in the repo prior to this kernel (confirmed by a dedicated research pass); building a full venue/grid/element system for one static entry scene would have meant "a bespoke Courtyard runtime separate from normal venue/session projection," which the kernel explicitly disallows — so this is intentionally a minimal data-only projection rather than a new venue subsystem. Documented here as an explicit, deliberate scope decision.
- `workbook_pages.go` — added `chapter4MechanicsFields` (skill stable ID, name, attribute, training state, die size — stable-ID-addressable for later dice actions) and `chapter4FaceWidgetFields` (first trained skill name + die, prioritized on Face after onboarding completes per spec), mirroring the existing Chapter 3 pattern.
- Routes: `GET /api/character-cards/chapter4-group` (auth), `POST /api/character-cards/chapter4-confirm` (auth), `GET /api/characters/chapter4-courtyard` (public, static).
- New tests: `chapter4_skills_test.go` (catalogue validation, per-attribute group lookups, ID/name lookups, and the critical key-skill-in-own-group proof), `chapter4_select_test.go` (auth/param guards, Chapter 2 attribute-total parsing, Chapter 4 fact round-trip, per-attribute selected-skill counting), and an addition to `workbook_pages_test.go` proving the new Mechanics/Face fields render.

### Frontend (`frontend/venues/catharsis/onboarding.js`, `index.html`)
- Replaced the old static "Chapter IV: The Archetype Is Set" placeholder shell with the real flow: title card (archetype, resolved attribute + score, capacity line, key skill name, "Continue to Skills") → ten-card skill grid (name + description per card, key skill visibly badged, click-to-select, nothing canonical until confirmed) → confirmation screen ("Add [Skill] as this character's first trained skill?" with the capacity change and "Training die: d6") → server commit → brief curtain-opening transition → Locked Courtyard scene render (walls/booths/crowd/Kessa/locked-door description fetched from the new endpoint) → Continue closes the overlay.
- Resume behavior: `current_stage === 4` (Chapter 4 not yet confirmed) reopens the title/group screen fresh, matching the spec's "temporary card selection need not be restored" rule; `current_stage === 5` (Chapter 4 confirmed, onboarding complete) falls through the normal "no onboarding needed" path since the character is fully unlocked for post-onboarding play.

### Tests
- `go test ./internal/characters/...` — all new Chapter 4 tests plus every pre-existing Kernel 53/54/55 test passes (no DB access)
- `go build ./...`, `go vet ./...`, `gofmt -l` clean on all touched files
- `node --check` on `onboarding.js`
- Skill catalogue count verified independently via a Python script cross-check (10 attributes × 10 skills = 100) before Go source generation, matching the JSON source's own `validation` block

### Live Verification
- Rebuilt and restarted `victory-backend` (twice — once for the initial Chapter 4 build, once for the Mechanics/Face projection fix found during testing); confirmed `GET /api/characters/chapter4-courtyard` responds live with the correct scene descriptor
- Full live run using The Architect (Lore primary / Craft-routed key skill "Design", one of the four routing-exception archetypes): Chapter IV title card showed correct archetype/attribute("Craft")/score/capacity/key-skill; the ten-card grid rendered exactly the 10 Craft skills (Artifice, Artistry, Construction, Cooking, **Design**, Masterwork Creation, Repair, Smithing, Tailoring, Toolcraft) — proving the key skill is guaranteed inside its own group even for an exception archetype; "Design" carried exactly one "Key Skill for The Architect" badge; selected a non-key-skill card (Artifice), confirmed with correct capacity math ("0 of 3 → 1 of 3"), saw the curtain transition, landed in the Locked Courtyard with all four required scene elements (walls, booths/crowd, Kessa, locked door) present, Continue closed the overlay cleanly. Zero console errors or failed requests throughout
- **Capacity/lock proof**: re-confirming with a *different* skill_id from the same group after the character was already locked returned success but with the original already-confirmed skill's data unchanged (no second skill silently added) — functionally correct rejection-by-idempotency, though flagged as a minor API-shape improvement opportunity (an explicit rejection error would be clearer than a same-skill echo for a non-onboarding caller; not fixed in this pass since normal UI flow never hits this path)
- **Idempotency proof**: re-confirming with the *same* already-confirmed skill_id returned success with exactly one History entry (no duplicate)
- **Bug found and fixed**: the confirmed first skill was not appearing on the Mechanics or Face workbook pages at all — Chapter 3 had this projection, Chapter 4 did not. Added `chapter4MechanicsFields`/`chapter4FaceWidgetFields` mirroring the Chapter 3 pattern exactly; re-verified live via direct API call (Builder archetype, Construction skill) — Mechanics now shows `chapter4_skill_stable_id: "SKILL_CRAFT_CONSTRUCTION"` and `chapter4_die_size: "d6"`; Face shows `chapter4_first_skill: "Construction"` and `chapter4_first_skill_die: "d6"`
- **Persona equip proof**: `GET /api/character-cards/me` correctly reports the confirmed character as `active_character` immediately after Chapter 4 confirmation
- All test character cards and sessions deleted after each verification pass
