# Operator Log — VICTORY

This file records meaningful implementation milestones and notable operational decisions.
Keep entries factual.

Current status and roadmap now live in:
- [current-state.md](/opt/victory/Construction/current-state.md)
- [roadmap.md](/opt/victory/Construction/roadmap.md)

This file should stay historical and chronological.

---

## 2026-07-20 — Kernel 73: Catharsis Equip Mode, Character Inventory, and Kessa Merchant Packet

Built Catharsis's first participant-local gameplay packet on top of Kernel 72's shared stage engine and Kernel 72A's venue capability flags: a Director-authored "Visit Kessa's Shop" interaction opens a private Program Panel for the triggering Player only, without moving the Show's shared current Scene. The Player picks one of five authored conversational stances (disposition fixed per stance, never flipped by a roll), attempts a server-authoritative Haggle skill check (d6 skilled / d4 unskilled, Target Value 5, narrative-only 20% discount, no currency touched), and purchases seeded starting equipment that persists as durable Character inventory.

New reusable primitives, not Kessa-specific: `equipment_items`/`character_inventory_items` (idempotent purchase via a dedicated attempt ledger mirroring Cues' `cue_executions`), `merchant_packets` (a bounded, non-dialogue-graph packet format reusable for a future second merchant), `participant_interactions` (its own table, participant-scoped rather than role-scoped like Cues, attached to a Placement through a new Director authoring panel in Stage Management), and `frontend/lib/stage-runtime/program-panel.js` (a venue-agnostic large-format overlay component).

### Real bugs found and fixed via an end-to-end integration test (not caught by reading the spec or code in isolation)
- `characters.CanEditCard` (the obvious ownership check) also silently requires a `location_memberships` role that a ticket-only Show Run Player never has -- `showruns.SelectCharacter` had already special-cased around this; three call sites in the new `merchant` package needed the same narrower ownership-only check instead.
- The only skill-lookup function in the codebase (`characters.FindCharacterSkillByName`) has the identical hidden requirement; replaced with a direct `character_skills` query scoped by an already-verified Character.
- `actions.StoreGameEvent`'s built-in authority check only allows director/producer session roles, contradicting its own doc comment ("any session participant"); needed `StoreGameEventTrusted` instead, exactly like `cues.ExecuteCue` does for the identical reason.

### Discovered precondition gap
No non-test Show Run/Show has ever been linked to Catharsis's live session (`session.show_id` was NULL) -- the only existing "Courtyard"-adjacent content was a stale Kernel 69 test fixture. Migration `057` seeds a reusable Courtyard Scene; placing it into a real Show and attaching Kessa to that placement is the real Director-authoring flow (Phase E), not something migration-seeded against a specific show_id.

### Verification
Full Go suite green; a new DB-backed integration test drives the entire chain (ticket→roster→character→Show→Courtyard placement→Kessa attached→open Equip Mode→all five stances→Haggle preview and attempt→idempotent purchase with quantity-mode clamping→inventory persistence) against the real migrated test database, plus a dedicated security-proof test file (non-rostered user, archived Character, forged interaction ID, Character-switch does not rewrite history) and a Hub-isolation test proving the targeted invalidation push never reaches another Player or Audience. New frontend tests cover the Program Panel's open/close/focus/Escape/loading/error lifecycle. Full `alpha-gate.sh` picks up the new frontend tests automatically (no gate changes needed) and passes.

---

## 2026-07-06 to 2026-07-09 — Kernel 61 / 61A: Trailer Player Workbook, Face Compiler, and Legacy Profile Migration

Replaced the fixed performer-profile form in Trailers with a server-authoritative Player Workbook: catalogue-driven pages, typed profile events, recomputed current facts, an owner-curated Trailer Face, an append-only stage-name ledger, cross-user Face viewing, targeted websocket live updates, a secure email-change flow, and closure of the legacy `/api/profiles/*` surface. Spanned several sessions; status is **PARTIAL** — see `Construction/OperatorLogs/kernel-61-reportback.md` and `kernel-61A-reportback.md` for the full acceptance-criterion ledger. Durable operator reference notes are in `operator-notes.md` under "Kernel 61 / 61A."

### Backend
- Migration `036_kernel61_player_workbook_foundation.sql`: `player_profile_workbooks`, `player_profile_events`, `player_profile_facts`, `player_stage_name_history` (partial-unique-indexed to one open row per user), `player_profile_face_overrides`. Additive only — no existing table altered or dropped.
- New package `backend/internal/playerprofile/`: versioned JSON catalogue (6 pages, ~50 fields incl. D&D class, Socio class/archetype, favorite-3-TTRPGs), event→fact derivation/recomputation, Face projection with visibility/priority overrides, stage-name ledger transaction, legacy `performer_profiles` import (idempotent, self-healing across bootstrap restarts).
- 8 HTTP routes under `/api/player-profile/*`; social route keyed by workbook ID, not raw account UUID.
- All 6 legacy `/api/profiles/*` routes now return typed `410 Gone` instead of touching `performer_profiles` (`backend/internal/profiles/profiles.go`) — Path B closure, chosen by the operator over building a compatibility adapter.
- `PATCH /api/account/email` (`backend/internal/identity/account_email.go`): owner-only, real Argon2id password reauthentication, case-insensitive uniqueness, explicit typed refusal (not weak confirmation) for password-less/provider-only accounts.
- Targeted websocket invalidation: new standalone `/ws/player-profile` endpoint (not built on the session-coupled `ServeVenueWS`), `Hub.BroadcastProfileWatchers`/`SetClientWatchProfile`, `player_profile/projection_updated` trusted event wired into every mutation via the existing `ProjectionChangeNotifier` callback pattern.

### Frontend
- `frontend/venues/trailers/face.html` — My Face (read-only preview + owner Face compiler + inline stage-name editor + Copy Trailer Link).
- `frontend/venues/trailers/workbook.html` (new) — catalogue-driven Workbook editor + History tab with client-computed deletion-impact preview.
- `frontend/venues/trailers/view.html` (new) — read-only cross-user Trailer viewer, reached via a copied link (`?id=<workbook_id>`), no edit surface at all.
- `frontend/venues/trailers/index.html` — the ~1,240-line legacy editor replaced with a small redirect stub to `face.html`.
- `frontend/account/index.html` — profile panel migrated off the deprecated legacy route; new email-change UI.
- `frontend/lib/player-profile-ws.js` (new) — shared websocket-watch client used by all three Trailer pages.

### Real bugs found and fixed live (not just in test)
- Page commits were writing a fact for every field on a page, including untouched blank ones — polluted stored facts with empty-string noise. Fixed by only submitting non-empty fields (or fields with an existing fact, so intentional clears still work).
- `RecomputePlayerFacts` sent bare Go strings to a `jsonb` column; pgx treats a plain `string` as pre-encoded JSON text rather than marshaling it, so unquoted values failed Postgres's JSON parser. Fixed with explicit `json.Marshal`.
- Legacy-import idempotency check returned early without recomputing facts, so a partial prior failure could stay broken forever across restarts. Fixed: facts always recompute on every bootstrap run.
- `scripts/smoke/fresh-install.sh`'s backend process was started via a backgrounded `go run` inside a subshell, so `$BACKEND_PID` was never the actual server process — repeated `--local` runs left orphaned backend processes holding the port and stale throwaway databases undropped. Fixed by building the binary once and running it directly, plus a belt-and-suspenders port-based cleanup fallback.
- A pre-existing, unrelated hover-reveal header CSS bug (shared across `trailers`, `directors-chair`, `the-cave`) made newly-added nav buttons unclickable — the header had no continuous hoverable region and no explicit `z-index`. Fixed using the pattern `producers-office` already had correct.

### Verification
- Live two-browser test: Copy Trailer Link → second real account opens it → sees only the compiled Face (no email/handle/UUID/workbook/history/controls) → cannot mutate the first account through any request shape → a Face/stage-name change in the first account's tab appears in the second account's open viewer tab without a reload, and in a second tab of the *same* account, via the new websocket push.
- Full stage-name proof on a disposable fixture account: ledger open/close transaction, same-name idempotency (exactly 2 rows after 2 duplicate resubmits), undeletable via the ordinary event-delete endpoint, UUID/handle/email/memberships/characters unchanged.
- Email flow proof: invalid format, duplicate (case-insensitive), missing/wrong password, valid change, and the password-less-account refusal path — all live, all via the real `/auth/signup` + `/api/account/email` flow, not synthetic data.
- `scripts/smoke/fresh-install.sh --local` run twice consecutively from empty, including new assertions proving a brand-new account can load the catalogue, get one workbook, commit a page, project a Face, and delete ordinary History — all pass, and the process/DB cleanup fix was verified by confirming a clean `ss`/`pg_database` state after each run.
- Straturli re-verified live each session: UUID/handle/operator status/Grant's Cabin access unchanged; Straturli's own stage name was changed by the actual account owner through the real UI between sessions (not by any test) — went from the legacy-migrated "The Starmaker" to "Grant A. Murray," and the Workbook shows real, deliberately-entered data (D&D class, Socio archetype, favorite TTRPGs, etc.), confirming real adoption of the feature.
- `go build`/`go vet` clean throughout; `go test ./internal/playerprofile/... ./internal/profiles/...` clean; pre-existing unrelated failures in `internal/assets` (filesystem-path test bug) and `internal/identity`/`internal/network` (live Discord API rate-limiting/state-drift tests) confirmed via `git stash` to exist independent of any Kernel 61A code.

### Known gaps (PARTIAL, not PASS)
- No Trailer-discovery mechanism beyond a manually copied link — no search, no directory, deliberately out of scope per the kernel spec's "do not build a new social/discovery system."
- Email reauthentication only covers password-holding accounts; provider-only (Discord signup) accounts are explicitly and cleanly refused, not given a weaker path.
- A pre-existing flaky Discord chat-bridge test leaks orphaned fixture users into the live `users` table when run — unrelated to this kernel, not fixed (out of scope).

---

## 2026-07-03 — Kernel 58 Greenroom Summary Panel Removal and Sidebar Contrast Pass

### Frontend
- Removed the visible Workbook Summary side panel so the Face sheet can occupy the primary workbook focus area.
- Expanded the workbook into a two-column layout with the left workbook reference panel and a wider central sheet.
- Hid the `Creation Progress` page from the visible tabs and routed any lingering selection back to `History`.
- Darkened the left-panel typography so labels and list text remain legible against the light workbook surfaces.

### Deployment
- Rebuilt and restarted the live backend container so the updated Greenroom layout is active.

### Validation
- Rechecked the Greenroom inline script with `node --check`.
- Rechecked the workspace diff with `git diff --check`.
- Verified the backend health endpoint from inside the restarted container.

---

## 2026-07-03 — Kernel 58 Greenroom Contrast Tuning

### Frontend
- Tuned the Greenroom parchment palette after the main visual refresh so contrast, label color, and surface brightness better support the new hierarchy.
- Darkened workbook text, metadata, and tab/button treatments slightly so the page stays readable without flattening the sheet into a washed-out cream panel.

### Deployment
- Rebuilt and restarted the live backend container again so the updated Greenroom color tuning is active.

### Validation
- Rechecked the Greenroom inline script with `node --check`.
- Rechecked the workspace diff with `git diff --check`.
- Verified the backend health endpoint from inside the restarted container.

---

## 2026-07-03 — Kernel 58 Greenroom Visual Refinement Pass

### Frontend
- Rebalanced the Greenroom workbook into a lighter parchment-and-ink presentation so the page reads more like a premium character dossier than a dark admin panel.
- Normalized the workbook container cards, summary panel, and face/bio surfaces to share a single visual language instead of mixing dark shell chrome with bright interior sheets.
- Improved typographic contrast inside the workbook so labels, metadata, and empty states stay readable after the surface treatment change.

### Deployment
- Rebuilt and restarted the live backend container so the updated Greenroom styling is active in the running image.

### Validation
- Rechecked the Greenroom inline script with `node --check`.
- Rechecked the workspace diff with `git diff --check`.
- Verified the backend health endpoint from inside the restarted container.

---

## 2026-07-03 — Kernel 58 Face Noise Cleanup and Bio Sheet Split

### Backend
- Removed stage and event noise from the visible workbook sheets so the Face and Mechanics pages no longer surface arbitrary internal checkpoints.
- Split the Catharsis lineage summary into a dedicated Bio page and kept the Face page focused on identity, ruleset, portrait, aura, archetype, skill, and quote.
- Preserved the parentage roll projection on Bio while dropping the combined roll and resolved starting credit from the visible sheet.

### Frontend
- Reworked the Greenroom Face page into a lighter report-style hero with a stronger hierarchy, ruleset version labeling, and less clutter.
- Added a dedicated Bio page with a scrollable centerpiece for lineage and biography notes, plus decorative side rails to keep the long-form text visually anchored.
- Shifted the workbook summary to show the ruleset key and version together instead of separate stage/event metadata.

### Deployment
- Rebuilt and restarted the live backend container again after the Face/Bio split so the running environment matches the repository state.
- Verified the backend health endpoint from inside the container after the final rebuild.

### Validation
- Re-ran `gofmt` on the updated workbook page files.
- Re-ran `node --check` on the Greenroom inline script.
- Re-ran `git diff --check`.
- Re-ran targeted Go tests from the backend module root:
  - `GOCACHE=/tmp/victory-gocache go test ./internal/characters ./cmd/victory`

---

## 2026-07-03 — Kernel 58 Character Sheet Face Projection and AAA Workbook Hero Pass

### Frontend
- Reworked the Greenroom face page into a report-style hero with a larger character-name header, identity strip, aura presentation, and supporting metric grid.
- Separated the main identity block from the rest of the workbook fields so portrait, name, pronouns, archetype, first skill, and token aura read as the primary sheet surface.
- Kept the remaining face fields in secondary sections so the workbook looks and behaves like a premium character sheet instead of a flat form grid.

### Deployment
- Rebuilt and restarted the live backend container so the shipped workbook assets picked up the updated Greenroom presentation.
- Verified the backend health endpoint from inside the container after the rebuild.

### Validation
- Rechecked the Greenroom inline script with `node --check` after the layout update.
- Rechecked the workspace diff with `git diff --check`.

---

## 2026-07-03 — Kernel 57 Catharsis Onboarding Repair, Greenroom Polish, and Custom Archetype Fixes

### Backend
- Fixed the Chapter 2 completion handoff so the last stage now advances into Chapter 3 instead of calling a missing Chapter 3 function.
- Kept the Catharsis coin-flip and parentage copy aligned with the actual roll data so the UI distinguishes parent roll totals from inherited starting credit.
- Hardened Chapter 3 custom archetype confirmation so the backend accepts a valid custom archetype payload instead of failing the flow with `unknown_archetype_key`.
- Added a server-side character account cap with a configurable default limit and enforced it at character creation.
- Corrected the effective Catharsis parentage roll to use the higher donor roll instead of summing the donor rolls.
- Updated the workbook face-page summary to derive the effective parentage roll from the parent rows and expose the combined roll separately so stale workbooks no longer present the wrong value as canonical.

### Frontend
- Tightened the Catharsis onboarding shell, onboarding cards, stage summary, and mobile stacking so the venue reads more like a finished shell and less like a prototype.
- Improved the Greenroom workbook library with clearer grouping for completed characters and characters in progress, plus cleaner mobile spacing.
- Reduced some of the densest Catharsis status labels so the live shell chrome reads more like a dashboard and less like a debug panel.

### Tests
- Re-ran targeted backend coverage for Chapter 3 confirmation from the backend module root:
  - `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/characters -run TestCommitChapter3`
- Re-ran the backend module test suite after the cap and parentage fixes:
  - `cd backend && GOCACHE=/tmp/victory-gocache go test ./...`
- Re-verified frontend syntax and diff hygiene on the touched Catharsis onboarding script and the workspace diff:
  - `node --check frontend/venues/catharsis/onboarding.js`
  - `git diff --check`

### Notes
- The Chapter 3 custom archetype fix was intentionally made backend-side so the custom payload path is authoritative instead of relying on frontend key handling alone.
- The final visual pass was limited to the Catharsis and Greenroom surfaces that were already part of Kernel 57 scope.
- The repo-wide backend test run still reports unrelated pre-existing failures in `internal/assets`, `internal/identity`, and `internal/network`.

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

## Kernel 59A Phase 3 — Projection Synchronization and Browser Acceptance Gate

Implemented and browser-verified the Phase 3 closure gate; Kernel 59A is now PASS.

- Added timestamp-derived `projection_version` to shared and venue character-sheet projections.
- Added server-authored `character/projection_updated` websocket invalidations after Face curation, Director locks/value overrides, character field saves, `/bio`, `/quote`, `/char set`, `/char add skill`, and `/char advance`.
- Added authorized websocket delivery filtering so clients receive projection invalidations only when their active session/persona or edit authority legitimately reaches the affected character.
- Added explicit websocket rejection for forged client-submitted `character/projection_updated`.
- Added First Theater and Catharsis Face/Mechanics tray tabs, version-aware matching-character refetch, same/older-version suppression, selected-tab preservation, and a restrained live-region update notice.
- Added server-side `face_value_locked` rejection for direct card-field edits when a Director value lock is active; aligned visibility/priority lock errors to `face_visibility_locked` and `face_priority_locked`.
- Rebuilt and restarted `victory-backend`; verified health from both `victory-backend` and `bread-caddy`.
- Checks passed: scoped `node --check`, `go test ./internal/characters`, `go test ./internal/network`, `go test ./internal/actions ./internal/commands`, `go build ./...`, `go vet ./...`, scoped `git diff --check`.
- Added and ran `scripts/smoke/kernel59a-phase3-browser.js` against live Caddy/backend with owner, venue, and producer-authorized Director contexts.
- Browser evidence is attached under `Construction/OperatorLogs/evidence/kernel-59A-phase3/`: 12 screenshots plus `acceptance-evidence.json`.
- Browser run proved: Catharsis and First Theater live quote hide/show without focus/reload, manual Bio priority, Director value override, Director value lock rejection (`face_value_locked`), unlock/edit projection, private Journal non-projection, 8 projection websocket frames without private reason/journal text, late join latest projection, and Mechanics tab skill rendering in both venues.
- Fixed Chapter II History body projection so old raw `BONUS_*` entry bodies render as readable bonus names; live workbook check returned `has_raw_bonus_in_body: false`.
- Final backend rebuild after the History projection fix was healthy from both `victory-backend` and `bread-caddy` at 2026-07-06T01:15Z.
- Full `go test ./...` still fails only on unrelated existing `internal/assets` UUID fixture and `internal/identity` Discord mapping/bootstrap tests.
- Remaining non-Kernel-59A backlog: `/quote add` / multi-quote collection, and optional persisted integer projection revisions if timestamp-derived versions become operationally insufficient.

### Kernel 59A Follow-up — Greenroom Make Active Repair

Fixed a post-PASS activation bug where Greenroom could snap back to the `?character_id=` URL parameter after clicking **Make Active**, making it look like activation failed. Greenroom now updates its selected card and URL to the activated card, and the completed-character library click path loads the selected workbook instead of only re-rendering stale state.

Also wired active-character activation through the existing projection websocket path with `changed_dimensions: ["active_character"]`; Catharsis and First Theater now refresh `/api/session/me` before refetching the venue sheet when that notice arrives, so open right trays switch to the newly active character.

Verification:
- Rebuilt and restarted `victory-backend`; health passed from `victory-backend` and `bread-caddy` at 2026-07-06T04:00Z.
- Browser proof: opened Catharsis on active `I`, opened Greenroom pinned to `?character_id=IV`, selected `III`, clicked **Make Active**, verified Greenroom stayed on `III` with `Active Character`, URL changed to `character_id=III`, and Catharsis tray changed to `III` without manual refresh. Then restored active back to `I` and verified Catharsis changed back.
- Final DB check: `straturli` active character is `I`.
- Checks passed: Greenroom inline `node --check`, `node --check` for Catharsis/First Theater runtime files, `go test ./internal/characters ./internal/network`, `go build ./...`, `go vet ./...`, scoped `git diff --check`.

### Kernel 59A Follow-up — Mic Command Rehearsal Gate Repair

Fixed a venue chat routing bug where Catharsis and First Theater handled `/mic` commands after the closed-chat gate. In rehearsal, this made `/mic status`, `/mic hot`, and related mic commands show `Chat is closed until the showing opens.` and never call `/api/discord/mic/control`, which could look like starting the mic had closed the session. Both venues now route `/mic` before normal chat/open-showing gating, matching The Cave's existing command path.

Verification:
- `node --check frontend/venues/catharsis/runtime.js`
- `node --check frontend/venues/first-theater/runtime.js`
- Scoped `git diff --check` for both runtime files.
- Headless browser smoke with `/api/discord/mic/control` mocked while the showing placeholder still read `Chat is closed until the showing opens.`:
  - Catharsis sent `{ "venue_slug": "catharsis", "command": "status" }` and rendered `Mock mic accepted`.
  - First Theater sent `{ "venue_slug": "first-theater", "command": "status" }` and rendered `Mock mic accepted`.
- Backend inspection confirmed `HandleDiscordMicControl` updates mic thread state only; session/showing close logic lives in session-control paths, not the mic control endpoint.

### Kernel 59A Follow-up — Catharsis Discord Channel Repair

Fixed Producer Office Discord channel repair so Catharsis receives the same venue support channels as the active stage venues. The repair spec already created a `Catharsis` category, but omitted both `Catharsis Audio` and `catharsis-chat`, so sync could appear successful while Catharsis still had no usable Discord channels for audio/mic/chat routing.

Changes:
- Added Catharsis to `venueAudioChannelSpecs()` as `Catharsis Audio`.
- Added Catharsis to `venueChatChannelSpecs()` as `catharsis-chat`.
- Updated Producer Office presence readiness to show Catharsis alongside First Theater and The Cave.
- Added Catharsis to the Producer Office venue invite dropdown.

Verification:
- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/identity`
- `cd backend && GOCACHE=/tmp/victory-gocache go build ./...`
- `cd backend && GOCACHE=/tmp/victory-gocache go vet ./...`
- Producer Office inline script syntax check via extracted `<script>` body and `node --check`.
- Scoped `git diff --check` for the touched backend and Producer Office files.
- Rebuilt/restarted `victory-backend`; health passed from both `victory-backend` and `bread-caddy`.
- Ran `/api/discord/channel-mapping/repair` through the Docker network with a temporary producer session; response reported `Catharsis Audio` and `catharsis-chat` as found.
- Live DB now has:
  - `venue_audio_channel | catharsis | Catharsis Audio | 1523545738611658792`
  - `venue_chat_channel | catharsis | catharsis-chat | 1523545739752374312`

### Kernel 59A Follow-up — Catharsis Session Route Repair

Fixed the remaining Catharsis live-session mismatch after the Discord bridge came online. The Catharsis frontend had inherited The Cave routing for world snapshots, session join, websocket connection, and stage action payloads, while the backend only exposed The Cave world/join/ws routes. This made Catharsis show `Idle` and keep chat closed even after `/session start`, because the shell was not reading the Catharsis session snapshot.

Changes:
- Added venue-aware backend helpers for world snapshots, session join, active-session lookup, session identity, and websocket serving.
- Added live backend routes:
  - `GET /api/world/catharsis`;
  - `POST /api/session/catharsis/join`;
  - `GET /ws/catharsis`.
- Updated Catharsis runtime modules to use `catharsis` instead of `the-cave` for world/session/ws/action routing.
- Wired `setCurrentSnapshot` and refreshed chat presentation after snapshot apply so the Session chip and chat gate update immediately.

Verification:
- `node --check` for Catharsis runtime files.
- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/identity ./internal/network ./internal/world`
- `cd backend && GOCACHE=/tmp/victory-gocache go build ./...`
- `cd backend && GOCACHE=/tmp/victory-gocache go vet ./...`
- Rebuilt/restarted `victory-backend`; health passed from both `victory-backend` and `bread-caddy`.
- Live `POST /api/session/catharsis/join` returned Catharsis session `359e4492-0cdd-4f3f-928f-4cf672ae5187`.
- Live `GET /api/world/catharsis` returned venue slug `catharsis` and showing status `rehearsal`.
- Browser smoke with a producer session loaded Catharsis and verified `Session Ready`, chat placeholder `Start typing here...`, and no failed requests.

## Kernel 58 — Face Projection, Token Aura, and Live Backend Refresh

Updated the character workbook Face projection to surface canonical `token_aura` data, add region/priority metadata to Face fields, and keep legacy `color` reads as a migration alias. Also updated the Greenroom Face editor to use the new field name, fixed the tagline save path, and refreshed the live backend container so the running service picked up the rebuilt code.

### Backend
- `backend/internal/characters/characters.go` and `workbook.go` now read and write `token_aura` alongside legacy `color`, with default Aura treated as unset instead of a hard default swatch.
- `backend/internal/characters/workbook_pages.go` now projects Face fields with region/priority metadata and a canonical `Token Aura` field.
- Added focused tests for Aura migration and Face projection metadata.

### Frontend
- `frontend/venues/greenroom/index.html` now renders the Face page as grouped regions instead of a single flat list, and the Face save path now posts `token_aura` plus the corrected tagline field.
- `frontend/venues/catharsis/runtime.js` and `frontend/venues/catharsis/runtime/scene-nodes.js` now carry token aura through local models and render a localized glow/halo when one is present.

### Validation
- `go test ./internal/characters ./internal/profiles ./internal/access`
- `node --check` on the Catharsis runtime files
- `node --check` on the Greenroom inline script
- `git diff --check`
- rebuilt and restarted `victory-backend`, then verified `GET /health` inside the container returned `ok`

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

## Kernel 62 — Player Relationship Matrix, Private Notes, and Relationship Journals (2026-07-09)

**Status: PASS** — see `Construction/OperatorLogs/kernel-62-reportback.md` for the full ledger. Changes uncommitted by operator decision (working tree left for personal review).

### What shipped
- Migration `037_kernel62_player_relationships.sql`: six new tables (`player_relationships` directional + unique pair + not-self CHECK, categories, facts, events, journal entries with soft delete, followups). Applied to the live DB (purely additive) and appended to the fresh-install migrations array.
- New backend package `backend/internal/playerrelationships/` mirroring the Kernel 61 `playerprofile` slice file-for-file: embedded versioned catalogue `player-relationship-v1.0.0.json` (Connection / Understanding Them / Our Relationship / Shared Work and Play — all freeform text fields), fixed §6 qualitative vocabularies + category set, events→facts fold with full-replace recompute, journal, follow-ups (open/done/dismissed + reopen; zero notification code), shared-context projection over verified `memberships`×`productions` overlap only.
- Privacy shape: observer always from session cookie (no request field exists to spoof); every non-owner access → 404 via `loadRelationshipOwned` (never 403, existence not confirmed); raw account UUIDs struct-tagged out of all JSON (unit-tested); subjects addressed only by opaque Player Workbook ID via the K61 `resolveUserIDForWorkbookID` pattern.
- Routes in `main.go`: `/api/player-relationships/catalogue`, `/api/player-relationships`, `/api/player-relationships/` (single prefix handler); startup-fatal relationship-catalogue validation beside the K61 bootstrap.
- Frontend: `frontend/venues/trailers/people.html` (My People: search/category/state filters, 3 sorts, empty states, privacy copy) and `person.html` (subject Face header with `/ws/player-profile` live refresh, nickname/categories/qualitative dropdowns, honest "Victory can currently verify" panel, 4-page workbook + history with deletion impact preview, journal with tag/category filters, follow-ups, archive/unarchive). Entry points: `Add to My People` ⇄ `Open My Notes` on `view.html`; `My People` chips on `face.html` and `/account/`.
- Archived-state rule: the state dropdown offers active/quiet/strained/rebuilding only; `archived` is entered/left exclusively through Archive/Unarchive (which also set/clear `archived_at`), so there is exactly one archive mental model.

### Verification
- `go build`/`go vet` clean; `internal/playerrelationships` 10 pure unit tests pass; `git diff --check` clean; `node --check` on every touched inline script.
- Full `go test ./...`: the two known pre-existing Discord failures (`identity` reconcile test, `network` chat-bridge fixture-leak) — re-confirmed present on the clean tree via `git stash`; no Kernel 62 file touches those packages.
- `scripts/smoke/fresh-install.sh --local`: full PASS including 9 new two-account Kernel 62 assertions (create by profile ID, nickname+qualitative save, page commit, journal, subject-404, archive filter split, self-relationship 400, page files).
- New `scripts/smoke/kernel62-browser.js` (Playwright, run with `NODE_PATH=/tmp/node_modules`): 14/14 groups PASS against the live rebuilt stack — add-from-Trailer, persistence across reload, journal CRUD + filters, follow-up done/dismiss, search, mobile viewport, subject/third-user 404s, subject-DOM-never-contains-notes scan, directionality, stage-name-change follow, archive round trip. 11 screenshots in `Construction/OperatorLogs/evidence/kernel-62/`.
- Live deploy: migration applied, `docker compose up -d --build backend`, `/health` OK. Straturli untouched (no identity/access tables modified; feature tables are all new).
- Residue: three clearly-named throwaway accounts on the live install from the browser proof (`k62_subject_1783620159501`, `k62_observer_1783620159501`, `k62_third_1783620159501`) — inert, documented in the reportback.

### Next recommended kernel
Discord fixture-leak cleanup (restores a clean `go test ./...` baseline before Third Place / Show Run roster work builds on this relationship primitive).

## Kernel 63 — Discord Test Fixture-Leak Cleanup and Back-to-Map Navigation Consistency (2026-07-10)

**Status: PASS** — see `Construction/OperatorLogs/kernel-63-reportback.md` for the full ledger. Changes uncommitted by operator decision (same as Kernel 62). Kernel 62 itself was found already committed (`3f907e9`) at the start of this session — no action needed for that prerequisite.

### Part 1: Discord fixture-leak cleanup
- Fixed all 5 test-fixture bugs diagnosed at the end of Kernel 62: missing `discord_session_threads` cleanup (the actual leak source), an incomplete mock `channelsJSON` fixture (missing catharsis category/chat, 3 voice channels, and `parent_id` on the 4 core channels — each gap surfaced only after the previous one was fixed), a stale `19` vs `21` mapping-count literal, a guild-ID-hardcoded shared mock transport, and `t.Cleanup` registered too late to survive a mid-setup failure in the chat-bridge test.
- **Found and fixed a genuine production bug** (not test drift — matches the explicit "unless the test exposes a real production bug" exception): `saveDiscordChannelMapping` cast `created_by_user_id` straight to `::uuid` with no `NULLIF` guard. `ReconcileDiscordBootstrap` — the automatic startup-reconcile goroutine that runs on every real server boot — always passes an empty `createdBy`, so **every channel-mapping save on automatic reconcile has silently failed since this code shipped**; only the manual authenticated repair endpoint ever worked. One-line `NULLIF($11, '')::uuid` fix, mirroring the adjacent parameter's existing pattern.
- Also found two more masked test bugs once the above let tests run further than ever before: a test assumed presence-list insertion order instead of the deliberate alphabetical sort `DiscordAudioPresenceStore.Snapshot()` actually uses (fixed the test assertion, not the correct production sort), and the chat-bridge test's synthetic action ID violated a real FK to `actions(id)` (added the missing fixture row).
- One-time live-DB cleanup: deleted the 1 leaked `discord_session_threads` row and 12 confirmed `bridge_operator_testmirror...` fixture users + FK-linked rows (11 pre-existing + 1 added by this kernel's own initial reproduction step). Did not touch ~25 other unrelated stale test users, real Discord config (currently empty), or Straturli.
- Validation order followed exactly: full unfiltered `go test ./internal/identity/...` and `./internal/network/...` runs (not `-run`-filtered, which is how two cascading failures were hidden originally), repeated twice consecutively to prove repeatability, plus a full `go test ./...`. Only remaining failure: `internal/assets`'s `TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails`, confirmed via `git stash` to be pre-existing and unrelated.

### Part 2: Back-to-Map navigation consistency
- New shared `frontend/lib/back-to-map.js`, modeled on `venue-account-badge.js`'s mount-detection pattern: default mode mounts into `.launchbar`/`.toolbar`/`.header-right`/`.top-actions`, floats top-left otherwise; `data-back-to-map="floating"` forces floating regardless of host.
- **Correction found during implementation**: First Theater and Catharsis's existing icon-only back-pill is hover-hidden by `.top-bar[data-open="false"] .header-right` CSS — same failure mode as The Cave/Director's Chair/Producer's Office. 5 venues needed the forced-floating treatment, not the 3 originally assumed.
- Added the script to 17 pages total: 12 in default mode (Account, Workshop, Grant's Cabin, Construction, Victory Theater, all 5 Trailers pages, Middle School Stage, Stage Template), 5 in forced-floating mode (The Cave, Director's Chair, Producer's Office, First Theater, Catharsis). Left 8 already-sufficient pages untouched (Greenroom, Warehouse, Mailbox, Login, Signup, both Legal pages, Audition Hall).
- Browser proof (`scripts/smoke/kernel63-back-to-map-browser.js`, Playwright): 10/10 assertion groups PASS on the live stack, desktop + mobile, using genuine `getComputedStyle`/`getBoundingClientRect` visibility checks (the exact check that would have caught the First Theater/Catharsis false-positive). Middle School Stage / Stage Template are operator-only venues (`ResolveVisibleVenues` excludes them for any non-operator role) — verified statically (served-HTML script-tag check) rather than granting a disposable test account real operator status via the shared `OPERATOR_HANDLE` env var, which was correctly avoided.
- Cleanup: 4 throwaway `k63_navcheck_*` accounts (each granted producer authority to reach producer-gated pages) all deleted after the proof passed, since — unlike Kernel 62's plain accounts — these held elevated privileges.

### Next recommended kernel
Either: (1) wire `TEST_DATABASE_URL` through the existing-but-unused `scripts/test/require-isolated-database.sh` for real DB-test isolation, or (2) sweep the ~25 other stale test-fixture users left untouched by this kernel's cleanup.

## Kernel 64 — DB Test Isolation and Live-DB Safety Gate (2026-07-10)

**Status: PASS** — see `Construction/OperatorLogs/kernel-64-reportback.md` for the full ledger. Changes uncommitted by operator decision (same as Kernels 62/63). This is the DB-test-isolation systemic fix both Kernel 62 and Kernel 63 flagged as their next recommended step.

- **New safety gate, enforced in two places**: `backend/internal/dbtest` (Go, used by every DB-touching test) and the rewritten `scripts/test/require-isolated-database.sh` (shell, used by the new setup/reset scripts). Both reject a missing `TEST_DATABASE_URL`, one equal to `DATABASE_URL`, or one whose database name is empty/`victory`/`postgres`/production-looking/doesn't contain `test`. DB-touching tests now `t.Fatalf` (not `t.Skip`) when the gate rejects them.
- **Fixed a real bug in the previously-unused `require-isolated-database.sh`**: it rejected `127.0.0.1`/`victory-postgres`/`localhost` hosts outright, which would have made a same-host dedicated test database (the approach this kernel's operator decisions called for) impossible to use. The discriminator is now the database name, not the host.
- **The three test-pool factories that hardcoded the live `victory` database directly in test source** (`internal/identity/discord_oauth_test.go`, `internal/network/discord_gateway_test.go`, `internal/assets/warehouse_test.go` — the exact pattern flagged in project memory as having wiped live Discord config in June) now go through `dbtest.OpenTestPool` and require `TEST_DATABASE_URL`.
- **New scripts** `scripts/test/setup-test-database.sh` (idempotent, non-destructive: creates `victory_test`, applies migrations, then boots the real `victory` binary once so its Go-side `Ensure*Surface` bootstrap runs too — several venues like `first-theater` only exist because of that, not any SQL migration) and `scripts/test/reset-test-database.sh` (the one destructive operation in this kernel, gated behind both the safety check and an explicit `CONFIRM_TEST_DB_RESET=1`).
- **Fixed the known pre-existing `internal/assets` baseline failure**, exactly as predicted in Kernels 62/63's reportbacks: the test passed a filesystem path where a location UUID belonged. Replaced with a real, self-contained fixture (own location + user + asset row).
- **Found and fixed two more fixture bugs only visible against a genuinely fresh database** (they never failed against the live DB because it has organic state — a `first-theater` venue and an `amurray-family` production — created out-of-band and never captured in any migration): `internal/network/discord_chat_bridge_test.go` now creates its own production and uses `the-cave` (migration-seeded) instead of assuming `first-theater` pre-exists.
- **Validation**: `go build`/`go vet` clean, full `go test ./...` green with `TEST_DATABASE_URL` set (zero failures, zero disclaimed-as-unrelated), proven to hard-fail without `TEST_DATABASE_URL` and to reject unsafe values (live DB name, prod-looking name, same as `DATABASE_URL`), live DB row counts (`users`/`location_memberships`/`sessions`/`player_profile_workbooks`/`player_relationships`/Discord config tables) confirmed byte-identical before and after a full DB-touching test run, Straturli re-confirmed resolvable, `fresh-install.sh --local` still fully passes (untouched by this kernel — it manages its own disposable database).
- One incidental live-DB test run happened early in the audit (before any code changes, to observe real baseline behavior) — confirmed self-reverted via `t.Cleanup`, no residue, documented in the reportback.

### Next recommended kernel
Either: (1) sweep the ~25 older stale test-fixture users left on the live DB (named in Kernels 62/63, still untouched), or (2) if a CI pipeline is ever added, wire `scripts/test/setup-test-database.sh` into it so `go test ./...` stays meaningful in CI.

## Kernel 65 — Third Place Headshot Commons MVP (2026-07-11)

**Status: PASS** — see `Construction/OperatorLogs/kernel-65-reportback.md` for the full per-criterion ledger. Changes uncommitted by operator decision (same as Kernels 62/63/64). Deployed live: migration 038 applied, backend rebuilt, `/health` OK.

- **New venue `third-place`**, seeded via migration 038 the same idempotent way `trailers`/`greenroom` were seeded in Kernel 9, and given the *exact same* map-visibility rule as `trailers` in `access.ResolveVisibleVenues` (one `IN ('trailers', 'third-place')` change, not a new gate) — per the kernel's explicit instruction to reuse the canonical Trailers visibility path rather than invent a new one.
- **New `third_place_headshots` table**: one row per placement/removal, "at most one active Headshot per account" enforced by a partial unique index (`WHERE removed_at IS NULL AND status = 'active'`) that the leave-operation's `INSERT ... ON CONFLICT` targets directly — race-safe by construction, not just idempotent by application convention. Removing closes the row rather than deleting it; re-leaving after removal opens a new one. No column anywhere stores old Trailer Face content — Headshots are always projected live.
- **New backend package `backend/internal/thirdplace/`**: `LeaveHeadshot`/`RemoveHeadshot`/`GetMyHeadshot`/`ListActiveHeadshots`/`ListMyHeadshotHistory`/`ProjectHeadshot`. `ProjectHeadshot` reuses `playerprofile.ProjectTrailerFace` (Kernel 61A) and `playerrelationships.GetRelationshipBySubjectProfile` (Kernel 62) unchanged rather than duplicating either — a Headshot card's stage name/portrait/headline facts and its Add-to-My-People/Open-My-Notes state are both always computed fresh per request, per viewer.
- **Three routes**: `GET /api/third-place/headshots` (commons list), `GET/POST/DELETE /api/third-place/headshots/me` (own lifecycle), `GET /api/third-place/headshots/me/history` (owner-only ledger). No duplicate relationship-mutation endpoint was built — the frontend calls the existing `POST /api/player-relationships` directly, per the kernel's own explicit preference.
- **New frontend `frontend/venues/third-place/index.html`**, modeled directly on `trailers/people.html`'s structure and conventions (forbidden-screen gate, Back-to-Map, client-side search/sort). Live updates reuse Kernel 61A's existing `/ws/player-profile` invalidation socket (`frontend/lib/player-profile-ws.js`, unmodified) — one watcher per visible Headshot, refetching the list on change. This was implemented, not deferred to refresh-on-reload, even though the spec would have allowed the latter for PASS.
- **Validation**: `go build`/`go vet` clean, full `go test ./...` green (10 new `internal/thirdplace` tests, zero failures anywhere), DB-touching tests proven to hard-fail without `TEST_DATABASE_URL` per Kernel 64's gate, `fresh-install.sh --local` extended with 7 new assertions and still fully passes, 17/17 browser-acceptance assertion groups PASS against the live rebuilt stack (3 disposable accounts: owner, viewer, and an unrelated third party proving relationship-privacy isolation), 4 screenshots (desktop + mobile) in `Construction/OperatorLogs/evidence/kernel-65/`.
- Cleanup: of 5 browser-proof runs during debugging, only the final successful run's 3 disposable accounts (`k65_owner_*`, `k65_viewer_*`, `k65_third_*`) were kept, per Kernel 62 precedent (plain accounts, no elevated privileges); the other 4 runs' 12 accounts were deleted. Straturli and all other live-DB counts confirmed untouched throughout.

### Next recommended kernel
Show Run primitive / Run Roster MVP is the natural next step — Third Place and My People were both explicitly built as primitives for it. The ~25 older stale test-fixture users (Kernels 62/63/64) remain a smaller, independent candidate.

## Kernel 66 — Show Run, Audience Program, and Roster MVP (2026-07-12)

**Status: PASS** — see `Construction/OperatorLogs/kernel-66-reportback.md` for the full per-criterion ledger. Changes uncommitted by operator decision (same as Kernels 62–65). Deployed live: migration 039 applied, backend rebuilt, `/health` OK.

- **Numbering collision note**: `victory-master-actual-implementation-guide-v1.md` has a stale, unrelated document-internal "Kernel 66 — Scene Transitions, Cue Groups, and Courtyard Tutorial Beat" section from earlier aspirational planning. It is not this kernel; this is the real Kernel 66, following directly from Kernel 65's own "next recommended" pointer above.
- **Corrected two load-bearing assumptions in the original draft spec before writing any code**: `productions` already existed (not a blank slate — `show_runs.production_id` follows `showings.go`'s exact `NOT NULL REFERENCES productions(id) ON DELETE RESTRICT` pattern, resolved server-side, never trusted from the client), and the draft's plan to define a future "Showing" as a pre-live scheduling concept would have collided with Kernel 22's already-shipped, differently-meaning `Showing` — resolved by documenting in the dictionary that any future scheduling capability must extend the existing `showings` table, not create a second same-named concept.
- **"Show Run" implements the dictionary's pre-existing, never-built "Production Run" entry** rather than adding a competing "Work / Game Work" hierarchy (which the original draft spec proposed) — the dictionary entry was merged (`Show Run (Production Run)`), not duplicated.
- **New 3-table data model** (`show_runs`, `show_run_roster_members`, `show_run_audience_blocks`), all with DB-level partial-unique-index enforcement for "one active row per user per run," mirroring Kernel 65's `third_place_headshots` pattern exactly. Roster cards are always live Trailer Face projections — no Face content is ever stored on a roster row.
- **New location-scoped authority primitive**: `access.CurrentLocationRoleForLocation` — the pre-existing `CurrentLocationRole` ignores which location is in play; a Producer at Location A would otherwise have passed an authority check for a Show Run at Location B. Tested with an explicit two-location negative case.
- **Audience is genuinely first-class**: a new no-role-filter `show-runs` visibility arm (unlike Third Place/Trailers, which exclude Audience) plus Audience-first ordering in both the internal roster and the curated Audience Program — verified live with a disposable audience-only account seeing the venue tile and the Audience Program correctly ordering itself with no manual role grant beyond default signup membership.
- **"Player," never "Cast,"** enforced at a single `roleDisplayLabel` chokepoint in `backend/internal/showruns/projection.go` — confirmed both in unit tests and in the live HTTP response.
- **Real bug caught by the proof process, not by unit tests**: `ShowRun`/`RosterMember`/`AudienceBlock` initially had no `json` struct tags, so responses would have silently shipped capitalized Go field names (`"ID"`, `"ShowFormat"`) to the frontend. Unit tests asserting on Go struct fields directly never touch JSON serialization and did not catch this — it only surfaced when `fresh-install.sh --local`'s new HTTP-level assertions parsed a real response. Fixed; recorded as a standing lesson in `operator-notes.md`.
- **`fresh-install.sh`'s migration list is a hardcoded array, not a glob** — migration 039 was silently skipped on the first fresh-install run until added explicitly. Fixed, and worth remembering for every future migration.
- **Validation**: `go build`/`go vet` clean, full `go test ./...` green (6 new `internal/showruns` tests plus the whole existing suite, zero failures), DB-touching tests proven to hard-fail without `TEST_DATABASE_URL`, `fresh-install.sh --local` extended with 8 new assertions and fully passes, live authenticated end-to-end proof against the real deployed server (two disposable accounts, real existing Production, full create → add-roster → enable-self-join → self-join → curated-program → visibility-arm → Third-Place-integration path, all confirmed correct) with zero residue after cleanup (verified by count).
- No screenshot-based browser evidence this kernel — `chromium-cli` was not available in this environment; substituted real authenticated-HTTP-session proof against the live server (same cookies, same requests a browser would make, no pixel rendering). Flagged as a known gap relative to Kernel 65's screenshot evidence.

### Next recommended kernel
Scene Configuration Model / Capture Scene is the natural next step now that the identity/social/scheduling-container spine (Trailer Face → My People → Third Place → Show Run) is in place — this is the displaced VTT-core work the original roadmap called for. A Showing-scheduling extension (per this kernel's dictionary note) and the older stale-test-user sweep remain smaller, independent candidates.

## Kernel 67 — Show Instance Model and Show Run Bridge (2026-07-12)

**Status: PASS** — see `Construction/OperatorLogs/kernel-67-reportback.md` for the full ledger. Changes uncommitted by operator decision (same as Kernels 62–66). Deployed live: migration 040 applied, backend rebuilt, `/health` OK.

- **Audited the live `showings`/`sessions` code before writing anything**, per the operator's own spec instruction to verify assumptions first. Found the spec's default suggestion — "prefer extending existing `showings`" — unsafe: `showings.session_id` is `NOT NULL UNIQUE`, lazily created on first action-write, and read/written from 20+ call sites across `internal/actions/*.go`, `network/session_control.go`, and Discord mic threads. Flagged to the operator before coding; confirmed decision was to build a **separate `shows` table** and leave `showings` completely untouched — not renamed, not schema-altered, not rewired.
- **New hierarchy**: `Production → Show Run → Show → Session(s)`. A Show is the concrete playable/viewable instance of a Show Run — draft/scheduled/live/paused/completed/cancelled/archived (7 statuses, deliberately distinct from Show Run's own 5). `sessions` gained a nullable `show_id` FK, set only through a new manual link/unlink action — deliberately **not** wired into the existing `/session start` command flow, so that surface stays untouched.
- **No new roster system**: a Show inherits its parent Show Run's roster and authority wholesale (thin pass-through wrappers in the new `backend/internal/shows` package), while still getting its own genuinely Show-specific Audience Program (`audience_title`/`audience_program_blurb` distinct per Show, live-verified with two Shows under one run rendering different blurbs).
- **Authority code reuse, not duplication**: exported `backend/internal/showruns/authority.go`'s `canManageShowRun`/`canViewShowRun` → `CanManageShowRun`/`CanViewShowRun` (pure rename, ~16 in-package call sites updated) so the new `shows` package calls the exact same location-scoped authority logic Kernel 66 built, rather than a second copy that could drift — a deliberate exception to this codebase's usual small-helper-duplication convention, justified because this is security-relevant code.
- **Dictionary self-correction**: Kernel 66's own dictionary note had predicted that future scheduling would extend the `showings` table. The Kernel 67 audit found that prediction wrong before it could mislead anyone — the note was rewritten to say `shows`, not `showings`, is the scheduling primitive going forward.
- **Regression discipline**: rather than assuming the new nullable `sessions.show_id` column was harmless, the full existing `internal/network` and `internal/actions` suites (the packages that actually own `/session`/`/mic` behavior) were explicitly re-run and confirmed green, not just assumed safe because the column is additive.
- **Validation**: `go build`/`go vet` clean, full `go test ./...` green (8 new `internal/shows` tests plus the whole existing suite including the `network`/`actions` regression check), DB-touching tests proven to hard-fail without `TEST_DATABASE_URL`, `fresh-install.sh --local` extended with 8 new assertions and fully passes (migration 040 added to the hardcoded array correctly on the first attempt, applying Kernel 66's own documented lesson), live authenticated end-to-end proof against the real deployed server (create Show Run → create Show → PATCH round-trip of audience/schedule fields → roster inheritance confirmed → self-join → curated Show-specific program → real session insert/link/unlink/clear → anonymous 401 → archive) with zero residue after cleanup.
- No screenshot-based browser evidence again this kernel — `chromium-cli` still not available in this environment; same substitution (real authenticated-HTTP-session proof) as Kernel 66, flagged as a known gap.

### Next recommended kernel
Scene Configuration Model / Capture Scene — the displaced VTT-core work the roadmap has pointed at for two kernels running, and now has a real container (Show) to attach to. A Showing-scheduling extension and the older stale-test-user sweep remain smaller, independent candidates.

## Kernel 68 — Venue Visibility Gates, Stage Management Surface, and Production Onboarding (2026-07-13)

**Status: PASS** — see `Construction/OperatorLogs/kernel-68-reportback.md` for the full ledger. Changes uncommitted by operator decision (same as Kernels 62–67). Deployed live: migration 041 applied, backend rebuilt, `/health` OK.

- **Found and fixed a stale role restriction, not just added a new gate**: `access.ResolveVisibleVenues` gated both `trailers` and `third-place` map tiles to `producer/director/cast/crew` location role since before this kernel — meaning a brand-new self-signup account (auto-granted plain `audience` role, zero friction) could never see either tile, even though the underlying `/api/third-place/headshots*` API has never itself required a performer role. Confirmed via `AskUserQuestion` before implementing: readiness now *replaces* the role gate for Third Place, and Trailers is now open to any authenticated user regardless of role — the old restriction was a carryover from Trailers' original rule that never got revisited when Third Place was built to be a general social space in Kernel 65.
- **`playerprofile.TrailerFaceReady`**: computed (no new durable marker) from the existing Kernel 61 projection — stage name plus at least one visible Face field. Gates the Third Place map tile (`access.SetThirdPlaceReadinessChecker`, an injected callback avoiding an `access`→`playerprofile` import cycle, same pattern as `ProjectionChangeNotifier`) and the direct "leave a Headshot" POST (clean `403 trailer_face_not_ready`); viewing the commons/history stays open to any authenticated user.
- **Stage Management** is now the user-facing label for the `show-runs` venue (slug/routes/package name unchanged, label-only rename in map/page headings). Visibility tightened from "any active location membership" to Operator/Producer/Director/active-crew-roster-row (`showruns.CanViewBackstage`, reused on both the map-visibility SQL and the backstage listing/detail API reads) — Audience/Player no longer see the tile or can load the backstage detail JSON directly, while the separate Audience Program route is untouched and still fully reachable.
- **Crew editor**: visibility-only implemented (a crew roster row unlocks the tile and backstage listing/detail); PATCH-level edit authority explicitly deferred per the spec's own permission — splitting it out safely would require a second, narrower authority check threaded through every mutating handler, which is exactly the "permission matrix" shape the spec says to avoid.
- **Minimal Create Production flow**: audited first and reconfirmed Kernel 66's documented gap (zero create route existed anywhere). Added `POST /api/productions` (Operator any location, Producer/Director their own resolved location, location_id always server-resolved) plus a small additive `productions.created_by_user_id` column and a "Create a Production" card in the Show Runs/Stage Management page that appears exactly when the picker would otherwise be empty.
- **Cloud/fog**: a base fog layer whose opacity recedes with the count of currently-visible venues, plus targeted cloud puffs at Third Place's and Stage Management's fixed map coordinates that disappear (CSS-transitioned) once each slug is present in the server's visibility response — no hidden venue name is ever rendered in the fog, and nothing hidden has a pin to click, so there's no broken click target under the cloud by construction.
- **Validation**: `go build`/`go vet` clean, full `go test ./...` green (readiness/visibility/production/backstage new tests plus the whole existing suite, one reproduced-then-passed pre-existing Discord-dispatch flake unrelated to this kernel), DB-touching tests proven to hard-fail without `TEST_DATABASE_URL`, `fresh-install.sh --local` extended with 11 new assertions and fully passes (migration 041 added to the hardcoded array on the first attempt).
- Two pre-existing `internal/thirdplace` HTTP tests had never actually set up a Trailer Face and needed their fixtures updated to supply one now that the POST endpoint they exercise is gated — not a behavior regression, a fixture catching up to the new gate.
- No screenshot-based browser evidence again — `chromium-cli` still not available in this environment; substituted a manual visual checklist (exact URLs/accounts/expected text) in the reportback.

### Next recommended kernel
Scene Configuration Model / Capture Scene — the roadmap has pointed here for three kernels running, and now has a Show container, a gated Third Place, and a labeled backstage surface all in place. A follow-up crew-edit-authority kernel and the older stale-test-user sweep remain smaller, independent candidates.

## Kernel 69 — Scene Library and Show Staging Model (2026-07-13)

**Status: PASS** — see `Construction/OperatorLogs/kernel-69-reportback.md` for the full ledger. Changes uncommitted by operator decision (same as Kernels 62–68). Deployed live: migration 042 applied, backend rebuilt, `/health` OK.

- **Confirmed via audit that no prior Scene concept exists in the real codebase** before writing anything — the only `scene` hit outside this kernel's own files is `frontend/venues/{first-theater,catharsis}/runtime/scene-nodes.js`, a PIXI.js canvas scene-*graph* helper unrelated to the Scene domain object (same word, unrelated meaning; flagged in the dictionary so it doesn't confuse future readers). The roadmap's "Scene Configuration Model" headers under document-internal Kernel 64-67 are confirmed-unbuilt aspirational plans per the roadmap's own collision note — this is genuinely additive work, not a refactor.
- **Two-layer reusable model, exactly as specified**: `scenes` (Production-scoped, `UNIQUE(production_id, slug)`) is the reusable authored object; `show_scene_placements` (`UNIQUE(show_id, scene_id)`, `scene_id ON DELETE RESTRICT`) is a Scene's use inside one specific Show. A Scene was proven live to stage into two different Shows under one Production as two fully independent placement rows — archiving one placement touches neither the Scene nor the other Show's placement, and archiving the Scene itself blocks only *new* placements while leaving every existing one untouched.
- **Authority reused, not duplicated**: `scenes.CanManageScenesForProduction`/`CanViewScenesBackstage` resolve `production_id → location_id` (the same lookup `showruns.CreateShowRun` already does) and delegate straight to Kernel 66/68's existing `showruns.CanManageShowRun`/`CanViewBackstage` — zero new authority logic, following Kernel 67's explicit precedent for security-relevant code. A new `scene_production_mismatch` check stops a Producer from staging a Scene from a different Production into one of their Shows.
- **Curated Audience data is a distinct Go type, not a filtered view of the backstage one**: `AudienceScenePlacement` has exactly three fields (title, audience_summary, sort_order) and structurally cannot carry `director_notes`/`operator_notes`/`source_ref`/`config_json`/ids. Live-proven with a `director_notes` tripwire string that never appeared in the curated response. The curated Scene Program lives at a **separate** route (`GET /api/shows/{id}/scenes/program`) from Kernel 66/67's existing roster-based Audience Program, specifically to avoid `backend/internal/shows` importing `backend/internal/scenes` (which already imports `shows`) — an import cycle. The frontend calls both and renders them together.
- **Real bug caught by writing a JOIN query, not by review**: `ListPlacementsForShow`'s SQL reused the single-table `placementColumns` constant unqualified against a two-table JOIN, producing `ERROR: column reference "id" is ambiguous`. Fixed with an explicit `p.`-qualified column list for the join specifically; the shared constant is untouched for its other (single-table) callers.
- **Live-proof workaround, not a code bug**: the disposable-account live-proof script initially got `401` on every authenticated call despite a valid `Set-Cookie` response, because `COOKIE_SECURE` defaults `true` and curl (correctly) won't re-send a `Secure` cookie over the plain-HTTP intra-Docker-network connection the proof container used. Worked around by extracting the raw session token and sending it back via an explicit `Cookie:` header instead of curl's cookie jar — the `Secure` cookie behavior itself is correct and untouched.
- **Validation**: `go build`/`go vet`/`gofmt -l` clean, `git diff --check` clean, 12 new `internal/scenes` tests plus the full existing suite green with zero regressions, confirmed the new tests hard-fail cleanly without `TEST_DATABASE_URL` (Kernel 64's gate), `fresh-install.sh --local` migration array updated (Kernel 66/67's "it's an array not a glob" lesson applied correctly on the first attempt) with 9 new assertions and a full clean-install PASS, live authenticated end-to-end proof against the real deployed server (3 disposable accounts, full create-Production → create-two-Shows → create-Scene → stage-in-both-Shows → mark-ready → curated-program → 4 distinct negative-authority checks → archive-placement → archive-Scene → new-placement-rejected path, all confirmed correct) with only the final successful run's accounts kept per Kernel 65 precedent.
- No screenshot-based browser evidence again this kernel — `chromium-cli` still not available; substituted the same real-authenticated-HTTP-session proof method Kernels 66-68 used, plus a manual visual checklist in the reportback.

### Next recommended kernel
A Fly Scene / Capture / Session-integration kernel — letting a Director make a staged Scene "live" for an active Session, the explicitly-deferred `live` status and active-scene-pointer work this kernel intentionally left out. A fuller in-page Scene editor (currently a single-field `prompt()`-based edit) is a smaller, independent candidate.

## Kernel 70 — Persistent Show Stage, Rehearsal Workspace, and Go Cue Foundation (2026-07-14)

**Status: PASS** — committed as `920aeb7` (`feat: Implement persistent Show stage and scene management`). Full backend test suite green, `fresh-install.sh --local` clean-install PASS, no regressions in any existing backend package. See `Construction/OperatorLogs/kernel-70-reportback.md` for the reconstructed formal ledger.

- **Corrected Kernel 69's Production-exclusive Scene scoping to location-scoping**, exactly as this kernel's spec required: `scenes.production_id NOT NULL` renamed to `source_production_id` (optional provenance), `scenes.location_id NOT NULL` added and backfilled, uniqueness swapped from `(production_id, slug)` to `(location_id, slug)`. Proven live: a Scene created under one Production can now be staged in a Show under a *different* Production at the same location (the direct correction proof), while cross-location placement remains rejected exactly as before.
- **Migration replay-safety required editing a prior kernel's migration file, not just adding a new one.** This repo's `setup-test-database.sh`/deploy flow replays every migration file from scratch on every run with no migrations-tracking table — a column rename in migration 043 broke migration 042's own `CREATE INDEX ... ON scenes(production_id)` line on replay (`column "production_id" does not exist`), because `CREATE INDEX IF NOT EXISTS` still fully resolves its column list even when it will end up skipping due to the name already existing (confirmed empirically — this differs from `CREATE TABLE IF NOT EXISTS`, which short-circuits before validating its body). Fixed by removing that one now-stale line from 042 (harmless — 043 creates the real replacement index) and wrapping 043's entire rename/backfill sequence in a single `DO $$ IF EXISTS(...) THEN ... END IF $$` block, since PostgreSQL only resolves a branch's column references when that branch actually executes. **Lesson for any future rename**: a migration that renames or drops a column referenced by an earlier migration's `CREATE INDEX`/`ADD CONSTRAINT` will break replay unless either the earlier file is made defensive or the whole operation is gated in PL/pgSQL.
- **The riskiest single mechanism in the kernel — the Show owning persistent stage state instead of the Session — was built and proven in isolation before any Cue code existed.** `actions` gained a nullable `show_id` column; `world.LoadVenueSnapshot`'s stage-object replay and action feed now query `WHERE session_id = $1 OR ($2 <> '' AND show_id = $2)`, reducing to the exact pre-Kernel-70 filter when `$2` (showID) is empty. Proven with three tests using hand-inserted `act/place_element` rows (not any Cue action type, since none of the three required Cue actions actually touch stage-object visuals): a Show's state persists across a session ending and a *new* session resuming it with **zero synthetic re-insertion** into the new session's own action log; an unlinked session is provably byte-for-byte unaffected; a session's own live actions still fold in normally on top.
- **A real audience-safety leak was found and fixed by a tripwire test, not by review.** `Show.VariablesJSON`/`CurrentShowScenePlacementID` were initially added as ordinary JSON-tagged fields on the `Show` struct — which `HandleShowProgram` (the Audience-viewable endpoint, gated only by `CanViewShowRun`, which Audience passes) serializes wholesale. A struct-reuse leak, not a per-endpoint filtering bug: a Show variable set via a backstage Cue would have shipped straight to Audience. Caught by the same tripwire-string technique Kernel 69 used for `director_notes`, fixed by making both fields `json:"-"` and exposing them only through `HandleShowByID`'s already-backstage-gated response map. Recorded as a standing lesson in `operator-notes.md`.
- **Go Cue foundation**: one `cues` table (validated ordered-action JSON array, not a table per action type) plus one `cue_executions` log. Three action types implemented — `go_to_scene`, `emit_game_event` (reuses `actions.StoreGameEventTrusted`, a new variant of the existing Kernel 60 game-event mirror that skips its participant-role `CanAct` gate since Cue-trigger authority already gated the whole GO press), `set_show_variable`. `reveal_object`/`hide_object`/`enable_interaction`/`disable_interaction` explicitly deferred — they would require extending the per-session visibility-layer derivation to a second show-scoped source, the "second object model" this kernel's own data-restraint section says to avoid without a fuller renderer pass.
- **Idempotent GO, proven under real concurrency**: a client-generated idempotency key per press is enforced by a DB-level `UNIQUE(cue_id, idempotency_key)` index, not just an application check. Proven with a genuine two-goroutine concurrent-press test — the loser either replays the winner's exact result or receives `cue_execution_in_progress`, never double-applies, and exactly one `cue_executions` row exists afterward either way. Each action within a Cue executes in its own transaction and fails stop (no rollback of already-committed earlier actions in the same Cue) — proven with a 3-action Cue where the middle action deliberately targets a nonexistent placement, confirming the first action's effect persisted, the third never ran, and the outcome is recorded as `partial_failure` with per-action detail.
- **Crew boundaries tested as explicit negative cases, not just omission**: Crew can create non-destructive Cues and press GO (including a GO that changes the current Scene, via the Cue path only — `shows.SetCurrentScenePlacement` called directly still rejects Crew), but cannot archive a Show, cannot edit a Base Scene, and cannot call the current-Scene-pointer endpoint outside Cue execution. `showruns.CanCrewPerformNonDestructiveEdit` is a deliberate alias of the existing `CanViewBackstage`, not a new boolean — the distinction that matters is which *write* endpoints call it, avoiding the permission-matrix shape Kernel 68 already flagged as unsafe.
- **A real gap surfaced mid-implementation and was closed, not shipped as a known hole**: player-facing stage Cue buttons (an explicitly required capability, not a deferred one) had no safe way to be listed, since the existing backstage Cue-list endpoint requires `CanViewBackstage` (which excludes Players) and the venue snapshot didn't expose which Show/placement was active. Closed with a new curated `GET /api/shows/{id}/scenes/{placement_id}/player-cues` endpoint (per-Cue `CanTriggerCue` filtering, `{id, label}` only, Audience always empty) and two new minimal snapshot fields (`session.show_id`, `session.current_show_scene_placement_id` — deliberately *not* Show variables, which stay backstage-only per the locked scope decision).
- **A parallel-looking `session-sync.js`/`socket.js`/`socket-controller.js` module tree in both venues' `runtime/` directories turned out to be genuinely live, not dead code — but the entry point was one directory up.** Grepping for `createSessionSync`/`createSocketController` inside `runtime/` found nothing referencing them, which initially looked like an orphaned in-progress refactor. The actual wiring lives in `frontend/venues/{first-theater,catharsis}/runtime.js` (a sibling file, dynamically `script.src`-loaded rather than a static `<script>` tag, which is why a plain grep of `index.html`'s script tags missed it). **Lesson: before concluding a module is dead code in this codebase, check for a same-named-but-un-suffixed sibling file at the venue root, and check for dynamic script injection, not just static `<script src>` tags.** Once confirmed live, the Rehearsal-availability banner and player Cue buttons were added as a small, self-contained floating panel (`updateStageCueControls`, wired through the real `applySnapshot`/`handleSocketMessage` dependency-injection pattern already used throughout `runtime.js`) rather than deep PIXI-canvas integration.
- **Scope honestly bounded, not silently expanded**: full in-venue Rehearsal composition (visually editing Scene content live on the PIXI stage) is NOT implemented — Scenes have no binding to the `elements`/`venue_layout_elements`/`actions` stage-object system, and building that binding was explicitly out of this kernel's time/risk budget. Base-Scene-vs-This-Show-Version editing exists today at the metadata/text-field level only (Scene Library page, Show detail page's placement rows), which is what "Rehearsal" concretely means this kernel. Recorded in the Dictionary's new "Base Scene vs. This Show's Version" entry so a future kernel doesn't assume deeper capture already exists.
- **Validation**: `go build`/`go vet` clean across the whole backend, 40+ new backend tests (`scenes`, `shows`, `world`, `cues`, `network`, `showruns`) plus the full pre-existing suite green with zero regressions, DB-touching tests proven to hard-fail without `TEST_DATABASE_URL`, existing frontend JS unit test suite (`node --test tests/first-theater/*.test.js`) confirmed at the exact same pass/fail count before and after (46 pass / 9 pre-existing-and-unrelated `dice.test.js` failures, verified via `git stash`), `fresh-install.sh --local` migration array updated (three new files, `043`/`044`/`045`) and its own pre-existing Scene assertions updated to match the new `location_id` API contract, full clean-install PASS from an empty database including every prior kernel's checks plus the new Kernel 70 Scene-scoping assertions.
- No screenshot-based browser evidence — browser automation was not available in this environment; the frontend changes were verified via JS syntax checks (`node --check`), the existing Node test suite, and manual code-path tracing through the confirmed-live `runtime.js` wiring, not visual inspection. Flagged as a known gap, consistent with every prior kernel in this log.

### Next recommended kernel
A "Scene visual capture" kernel that designs how Base-Scene-vs-This-Show composition maps onto the existing `elements`/`venue_layout_elements`/`actions` stage-object system — the explicitly-deferred piece this kernel's own Dictionary entry flags. `reveal_object`/`hide_object`/`enable_interaction`/`disable_interaction` Cue actions are a natural, smaller follow-on once that mapping exists. A dedicated browser-automation smoke pass (screenshots of the GO controls, Rehearsal banner, and player stage buttons) would close this kernel's one remaining evidence gap without requiring new backend work.

## Kernel 70A — Live Stage Closure and Alpha Path Alignment (2026-07-16)

**Status: PASS, deployed live** — committed as `2611840`. Backfilled into this log during Kernel 71's documentation pass; see `Construction/OperatorLogs/kernel-70a-reportback.md` for the full ledger and `current-state.md`'s Kernel State section for the live-deployment detail.

- Server-side `shows.StartShowSession` action so a Director never pastes a Session ID; proved Show-owned stage state (current Scene, variables) survives a full start/end/resume cycle at Catharsis with no rebuild step.
- Fixed `go_to_scene`/`set_show_variable` to work with no active session while `emit_game_event` fails cleanly and visibly; closed an Audience-facing leak where the Rehearsal banner and Cue buttons were visible regardless of viewer role.
- Added backend-computed `theater_context` empty states with exact required copy; gave First Theater its own independent, investor/demo-only venue wiring without touching the-cave; audited Character-to-Show linkage (no build, recommended `character_card_id` on `show_run_roster_members` — built in Kernel 71).
- **Live deployment surfaced a real, previously-unnoticed gap**: Kernel 70's own migrations (`043`-`045`) had never reached the live `victory` database despite being committed two days earlier — the live backend was silently erroring on every `/ws/catharsis` snapshot until this deploy replayed all 46 migration files (after a `pg_dump` backup) and restarted the backend.
- **A second genuine pre-existing bug was found during manual live verification, not fixed this kernel**: two independent, non-interchangeable membership tables (`location_memberships` vs. the older `memberships`) both claimed to answer "what is this user's role," and a real Producer with only `location_memberships` resolved as `viewerRole = "none"` at their own venue. Flagged for Kernel 71+ — closed by this kernel's canonical participation resolver (see below).
- No browser automation tooling existed in this environment; verification was a full manual Director/Player/Audience checklist driven directly over HTTP against a real compiled backend instance.

### Next recommended kernel
Resolve the `location_memberships`/`memberships` split (a live authority bug, not just tech debt) and build the missing participation bridge between the arrival/character systems and the live Show runtime — see Kernel 71 below.

## Kernel 71 — Two-Punch Show Tickets, Character Participation, and Showtime (2026-07-16)

**Status: PASS, uncommitted** — implemented and verified in this pass; not yet committed or deployed, pending review. See `Construction/OperatorLogs/kernel-71-reportback.md` for the full ledger.

- **Canonical participation resolver** (`backend/internal/participation`, new package): `ResolveParticipationContext` reconciles `location_memberships`, `show_run_roster_members`, and legacy `memberships`/`access_grants` under one precedence order (Operator → location role → show-run roster role → any location membership (audience) → legacy tables as a last-resort upgrade only, never a downgrade). `main.go`'s `lookupVenueRole` now delegates to it, closing Kernel 70A's `viewerRole="none"` bug — proven by two required bug-closure tests: a Producer with only `location_memberships` resolves as producer, not `none`; a user with only a valid ticket-derived roster row gets full venue entry with zero manual access-grant step.
- **Two-punch Show tickets** (`show_run_tickets` migration + new `backend/internal/tickets` package): the sole ordinary path to an active Player roster row. Either side may punch first; the second punch (`tickets.SecondPunch`) is one atomic, row-locked transaction that marks the ticket valid and creates/reactivates exactly one Player roster row via the existing `show_run_roster_members` partial-unique-index `ON CONFLICT` — proven under genuine concurrency (8 goroutines racing the same ticket, exactly one resulting roster row, zero errors on the losers).
- **Closed the actual ticket-bypass gap, not just the intended-primary path**: `showruns.AddRosterMember` and `UpdateRosterMemberRole` both used to allow a direct/promoted `role="player"` insert with no consent at all — a Director could call the generic roster endpoint directly, bypassing the new UI entirely. Both now reject this for non-Operator actors (`player_requires_ticket`), found and closed before this kernel's own reportback, not shipped as a known hole.
- **Third Place's "Add to Show Run" picker converted, not removed**: it now calls `tickets.InviteFromDirector` (a Director's first punch) instead of the old unilateral roster-insert endpoint — Third Place still cannot itself create participation, per spec.
- **Character selection**: new `show_run_roster_members.character_card_id` column, a new self-service endpoint, and `world.LoadVenueSnapshot`'s `TheaterContext` swapped from the old Session-scoped `current_session_personas` check to this Show-Run-scoped column — deliberately independent of `current_session_personas`/`active_user_characters`, with an archived-Character fallback proven by test.
- **Show short codes and `/showtime`**: every Show gets an auto-generated, confusable-excluding, location-unique code at creation; `/showtime <code>` resolves the Show, derives its venue from staged Scene Placements (asking only when none or multiple distinct venues are staged, never guessing), and starts/resumes the Session via the unmodified Kernel 70A `StartShowSession` — mic stays off by construction, and `/showtime end` preserves current Scene/roster/Character selections by construction (proven with a before/after snapshot test). Implemented as an in-app legacy command (`/session`'s precedent), not a real Discord-native slash command — confirmed with the user as the intended scope before this phase began.
- **A real route-registration bug was caught immediately by the fresh-install smoke script, not shipped**: `GET /api/shows/by-code/{code}` initially collided with the existing `GET /api/shows/{show_id}/program` pattern under Go 1.22 `ServeMux` (same segment shape, ambiguous wildcard-vs-literal) and panicked the backend at startup. Fixed by switching to a query parameter (`GET /api/shows/by-code?code=`) instead of a path segment.
- **A second real bug was caught the same way**: three pre-existing tests (in `cues`, `shows`, and `showruns` itself) called `AddRosterMember(..., "player", ...)` directly as a non-Operator actor for test fixture setup — silently still passing from Go's test cache in early full-suite runs, only surfacing under a fresh `go test -count=1 ./...`. All rewritten to insert the fixture roster row directly rather than through the now-gated function.
- **Validation**: `go build`/`go vet` clean, full `go test -count=1 ./...` green (new `participation`/`tickets`/`showtime` packages plus every existing package, zero regressions), `scripts/smoke/fresh-install.sh --local` extended with real HTTP-driven proof of both ticket directions (Player-request via a dedicated account, Director-invitation via another), the direct-add rejection (via a disposable non-Operator Director account, since the script's own bootstrap account is the environment's Operator), Character selection, and a full `/showtime` start/status/end cycle with automatic venue derivation — full clean-install PASS. `scripts/test/alpha-gate.sh` run end-to-end: overall PASS (the pre-existing tracked `dice.test.js` exception is the only non-blocking item).
- No browser/screenshot evidence — no browser automation tooling exists in this environment; substituted the same real-compiled-backend HTTP proof method used since Kernel 65.
- Not deployed live this pass, by scope decision — implementation and verification only.

### Next recommended kernel
Visual Scene composition/capture (the standing next-recommendation since Kernel 70). Independently, a smaller closure pass could: repair the nine pre-existing frontend `dice.test.js` failures; add browser screenshot evidence for Kernels 70/70A/71's stage and participation controls; and deduplicate the three packages (`characters`, `assets/read.go`, `showings/review.go`) that each independently UNION `location_memberships`+`memberships` instead of calling one shared resolver helper. A live deployment pass for Kernel 71 should explicitly re-verify migrations `046`-`048` reach production, per the lesson Kernel 70A recorded.

## Kernel 74 — Locked Door Intentions, Ra Guided Dialogue, and Participant-Local Tutorial Handoff (2026-07-27)

**Status: PASS, deployed live, uncommitted.** Migrations `066`–`068` applied to production with pre-apply backups (ledger at 69). See `Construction/OperatorLogs/kernel-74-reportback.md` for the full ledger.

- **The tutorial is playable end to end with no Director GO inside it.** Kessa completion → milestone-gated door hotspot → freeform Player intention → automatic Ra interruption → authored Crown Bet topics → Player-controlled Leave → participant-local handoff map. The operator walked this live.
- **Participant-local stage projection** (`participant_local_projections`, new package `backend/internal/projection`) is the one structurally new idea: it substitutes *which Scene's composition one viewer resolves*, and never writes `shows.current_show_scene_placement_id`. It deliberately does **not** fork the element/visibility model — the boundary `cues/types.go:43-48` recorded when it deferred per-participant visibility. There is still exactly one object model.
- **Three new leaf packages**: `tutorial` (five CHECK-constrained milestones), `dialogue` (bounded guided-dialogue packets with an acyclic-validated prerequisite DAG), `projection`. The flow orchestration lives in `merchant` because `ResolveEligibleContext` does — a naming debt documented in `tutorial_flow.go`'s header rather than fixed by a rename touching every call site.
- **Directors+ notes reuse the durable `messages` table** (`message_type='backstage_note'`) rather than `HandleNoteCards`, which is hardwired to `the-cave` and whose recipient roles cannot express Producer/Operator. Live push is per-recipient `BroadcastToSessionUser` — never session-wide, because the hub does no role filtering.
- **The Kernel 73A left-click defect was already fixed** in the tree before this kernel began; only the regression test was missing. Added, against a bound `scene_composition` token specifically.
- **Five real defects were found by the operator's live browser walk, not by the test suite** — each recorded in full in the reportback §9. The two worth carrying forward as durable lessons: (a) an *"omit from payload"* authority gate is never sufficient on its own — the door's reveal gate protected discovery but not invocation, so a Player who learned the interaction id could POST straight to it; and (b) *a control that cannot act must not be able to block* — an unbound hotspot still rendered a pointer-capturing box that sat over Kessa and swallowed her clicks.
- **A sixth change, on the operator's explicit approval**: `ActivateCharacterCard` now writes through to `show_run_roster_members.character_card_id`. Victory had two independent "current Character" values, and switching in the Greenroom left the Show-Run roster selection — Kernel 71's canonical one, which every Kernel 74 surface keys off — untouched.
- **Validation**: `scripts/test/alpha-gate.sh` green across multiple full runs; tracked `dice.test.js` exception unchanged at 9. New `kernel74_tutorial_dbtest_test.go` (3 tests) drives the whole PASS standard against the real migrated schema; `tests/stage-runtime/kernel74-tutorial.test.js` (13 tests) covers the frontend contract and all four live-play regressions.
- **Live side effects**: throwaway `k74-browser-*` rows fully cleaned up; 39 `k74_*` accounts remain with no elevated privileges (Kernel 62/63/65 precedent). The operator's pre-existing Catharsis Session was never touched — `scripts/smoke/kernel74-tutorial-browser.js` preflights and aborts rather than ending a Session it did not open, which is why it has never completed a full automated run.

### Next recommended kernel
**Kernel 75 — Tutorial Completion, Aftercare, and Director-Controlled Continuation**, drafted at `Construction/Kernels/kernel-75-tutorial-completion-aftercare-continuation-v0.1.md`. Mostly presentation on top of machinery that already exists: final handoff art and copy, Ra's prose rewrite, tutorial-complete presentation, progression acknowledgement, and the Director's readiness view. **Aftercare has no prior definition anywhere in Victory and needs an operator decision before anything is built.** A second open question the draft raises: authored NPC dialogue is now seed-only for the second time (Kessa in 73, Ra in 74), so every wording change is a migration — worth deciding whether a bounded packet editor is due.

## Kernel 78 — eWrite Foundation (2026-08-04)

**PASS, deployed live, uncommitted.** Migrations `084`–`085` applied to production with a pre-apply backup (ledger at 86). Full detail: `Construction/OperatorLogs/kernel-78-reportback.md`; architecture notes in `Construction/eWrite/`.

- Victory's first rules-native writing/publication/reading system: Writer's Room (Crew+ map tile, `ewrite_author_surface`) and Library (authenticated surface; its half-built seeded slot from Kernel 16 finally has pages).
- First Markdown pipeline in the repo — goldmark + bluemonday, server-side only, policy built from empty; malicious fixtures tested vector by vector. A recorded spike chose a fence-aware regex pre-pass over goldmark's `{#id}` parser, which mangles the Google-Docs anchor charset (`(`, `?`).
- Scale proof against the real `Sociov1_1.md`: 357 KB / 43,207 words / 763 headings, 137 explicit anchors preserved verbatim, render ≈136 ms, full save ≈0.9 s.
- Save conflicts are 409s that never touch the submitted text; revisions append-forward; section rows keep UUIDs across edits with anchor aliases on rename, so links survive title changes and object links degrade instead of dangling.
- Account export gained `ewrite/` (format_version 2); deletion hard-deletes sole-owned drafts and tombstones shared work.
- Repaired `scripts/smoke/fresh-install.sh`, silently unrunnable since Kernel 76 (hardcoded pre-rotation password; stale 200-expectations for routes K76 deliberately closed; a Third Place fixture predating K68's Face-readiness gate). The alpha gate now passes end to end again.
- Deferred by operator decision: the anonymous public read route (`public` visibility currently serves signed-in readers). Grant supplies the Writer's Room map icon (`frontend/assets/writers-room.png` is a placeholder copy of default.png).

## Kernel 79 / 79A — eWrite Integration, Rules Navigation, Skill Directory (2026-08-05)

Not backfilled into this log in detail — see `Construction/OperatorLogs/kernel-79-reportback.md`, `kernel-79-goal-ce-reportback.md`, `kernel-79-goal-f-reportback.md`, `kernel-79a-reportback.md` directly. Both PASS, deployed live, uncommitted.

## Kernel 80 — Storyboards Core (2026-08-06)

**Status: PASS against the written spec, deployed live, uncommitted.** Migrations `090`-`091` applied to production with a pre-apply backup. Full ledger: `Construction/OperatorLogs/kernel-80-reportback.md`; design docs in `Construction/Storyboards/`.

- Victory's first reusable grid-board primitive: ordered columns, ordered rows grouped into bands, cells holding zero-or-more ordered cards, owner + explicit per-user sharing, live server-authoritative multi-user sync, versioned JSON export.
- New backend package `backend/internal/storyboards/` (39 tests): full role-capability matrix with proven client-role-forgery rejection, optimistic-concurrency card locking proven race-free under real concurrent goroutines, hidden-from-audience filtering enforced identically over HTTP, WebSocket, and export (a per-viewer WS fan-out was built specifically so a hidden card's existence never leaks via even a content-free "something changed" event), and the spec's required occupied-structure removal resolution flow (never a silent cascade).
- **A real architecture simplification was found mid-implementation, not shipped as originally planned**: the domain-model plan called for a single global row `sort_order` with an application-enforced band-contiguity invariant. Rebuilt during implementation as a two-level order (`band.sort_order`, `row.sort_order_in_band`) instead — this makes every one of the spec's band/row invariants ("every row belongs to exactly one band," "bands never overlap or share rows") true by construction, with zero invariant-checking code, rather than trusted to be enforced correctly by every future mutation.
- **A real import-cycle constraint was discovered and worked around, not avoided by luck**: `network` already imports both `identity` and (transitively, via `actions`→`characters`) `ewrite`. Since `storyboards` needs `network` for live WS events, neither `identity` nor `ewrite` can import `storyboards` without closing a cycle — both packages instead inline the small authority/query logic they need directly, with a comment explaining why. Recorded in `operator-notes.md` for the next kernel that touches any two of these three packages together.
- **First kernel in this project's history with real, working, screenshotted browser automation** rather than a substituted HTTP-only proof (Playwright + headless Chromium installed fresh this session, authenticated via directly-seeded session tokens rather than Discord OAuth) — proved the full multi-role golden path, live cross-tab WebSocket sync, and the occupied-column-removal resolution UI flow, and caught a real layout bug (floating nav widgets overlapping page content on header-less pages) that no automated test would have found.
- **A significant post-deploy product-fit gap was surfaced by Grant's own live review, not hidden**: the shipped frontend (plain HTML-table grid, unstyled cards, no drag-and-drop, append-only columns, writers-room color palette) does not match his actual intent (card-shaped visuals matching the existing canvas index-card, drag-and-drop as the primary interaction, arbitrary column insertion, the red/rose/black palette used elsewhere, and image-attachment support with none of that in the written spec to begin with). None of this is a written-spec violation — every stated acceptance criterion has real evidence — but it is a real scope gap between the spec as written and the product as wanted. Full accounting and a scoped continuation-kernel outline are in the reportback's §8/§10, specifically so the next kernel can be budgeted without re-deriving this conversation.
- One further real conflict surfaced and deliberately left unresolved pending an operator/kernel-maker conversation: the written spec explicitly requires "cells support multiple ordered cards" as a tested pass criterion (built and proven); Grant has since said he intended exactly one card per cell. Not silently resolved either way — flagged for the kernel maker.

### Next recommended kernel
**Kernel 81 — Storyboards Presentation Rework**, once Grant has picked a rendering direction (restyle the table / CSS Grid of divs / rebuild on the `stage-runtime` canvas — the canvas engine's existing token-grid-snap math is reusable for card drag-and-visuals but not for the labeled/bounded/structured grid Storyboards actually needs, which has no precedent in either paradigm), settled the one-vs-multiple-cards-per-cell question, and decided whether image-attachment + lightbox is bundled or its own kernel. The backend, permissions, live sync, and export need no changes for any of this — it is a scoped frontend-only continuation on a proven, tested API. Independently, `current-state.md` (the Master Actual Implementation Guide) has not been updated since Kernel 78 and is now three kernels stale (79/79A/80) — a documentation-catch-up pass would be more efficient done once across all three than piecemeal.

## Kernel 81 — Storyboards Presentation Rework (2026-08-06)

**Status: PASS against the written spec (pending Grant's own live visual sign-off — see below), deployed live, uncommitted.** No new migration ledger entry beyond `092` (one small, no-FK column addition). Full ledger: `Construction/OperatorLogs/kernel-81-reportback.md`; design docs updated/added in `Construction/Storyboards/`.

- Replaced Kernel 80's plain HTML-table board with a real CSS Grid (`grid-model.js`'s `computeGridLayout` is the single source of truth for every grid line number, dual Node/browser like `stage-runtime/geometry.js`), Cave-inspired compact cards with a controlled color-token whitelist (never raw CSS injection), Pointer-Events drag-and-drop as the primary move interaction with the modal fallback retained, a deliberate Swap/Move-existing/Cancel dialog on any occupied-cell drop, column insert-left/right (frontend-only, reusing Kernel 80's existing `AddColumn`+`ReorderColumns` endpoints), the red/rose/black palette, and an entirely new one-pinned-image-per-card capability with a real lightbox and a genuine gravestone-on-deletion behavior.
- **The Kernel 80 backend was left untouched except two narrow, spec-pre-approved additions**: `SetCardImage` (Crew+-unless-locked, same authority as any other card field) and `SwapCards` (a new atomic transaction — chosen over two sequential `MoveCard` calls specifically because a cell briefly holding two cards during a swap is visible to a third concurrently-loaded watcher, and Swap is one of the kernel's own required proof scenarios).
- **A real architectural mismatch was found and solved, not sidestepped**: every existing Victory asset is stored and quota-accounted against a producer + location, but Storyboards boards are personal with no Production of their own (a Kernel 80 design decision). Requiring the *uploading* Crew member to be a Producer somewhere (matching the existing `HandleWorkshopUpload` gate) would have silently locked out most real Crew editors. Solved by scoping storage to the *board owner's* own producer membership instead (`resolveCardImageStorageScope`), and by adding a new, unexported-membership-gate-free upload helper (`assets.CreateReferencedImageAsset`) rather than reusing the Producer-only endpoint.
- **Two real bugs were found by the real-browser proof, not by inspection or the unit suite, and both fixed with regression coverage before this kernel closed:**
  1. A CSS Grid line-number collision: `board.html` originally computed the "+ row" button's line independently of `computeGridLayout`, and the two computations silently collided whenever a band had exactly one row — caught as `<span>Act Two</span> ... subtree intercepts pointer events` on a real click, something no static check would have found since both individually-computed line numbers looked valid in isolation. Fixed by making `computeGridLayout` the sole owner of that line number too, with a dedicated regression test.
  2. A genuine authorization gap: a Storyboard grant holder with zero location memberships anywhere got 403 loading a card's image, because asset-read authorization was gated entirely on location membership, unaware a card image's real authority is the Storyboard grant. Fixed in `assets/read.go` via a raw-SQL inline check (not an import of `storyboards`, which already imports `assets` and would close a cycle) mirroring `CanViewBoard`'s "any resolved tier" rule.
- Deliberate design choice worth naming: the deletion-preserving image reference (`storyboard_cards.image_asset_id`) has **no foreign key at all** — a real FK would be silently defeated by either of two existing asset-lifecycle paths (account-deletion hard-delete, or the existing soft-tombstone mechanism), both of which would erase the "an image was pinned here" state the gravestone requirement depends on. An unconstrained id column survives both paths intact; gravestone detection happens entirely by attempting to resolve the id at render time.
- Real browser proof (Playwright, same session/scratchpad as Kernel 80 — the tooling and session-seeding trick needed zero re-setup) covered all 20 required scenarios from the kernel spec plus the gravestone follow-up: 21 screenshots, three throwaway accounts with **zero location memberships at all** (deliberately, which is exactly what surfaced bug #2 above), full cleanup of all test boards/users/sessions/assets/storage afterward.
- Left deliberately unresolved, named rather than hidden: keyboard-only resolution of an occupied-cell "Move existing" drop has no non-pointer path yet (`storyboards-accessibility.md`); the Playwright/session-seeding procedure is still not captured as a reusable project skill despite two successful kernels now relying on it.

### Next recommended step
Not a new kernel — Grant's own live visual review against spec §21's 16 questions, per the spec's explicit framing that automated correctness alone is insufficient for a presentation-correction kernel. If the answer is yes, the spec's own stop-loss rule (§20) says to let Storyboards sit as a stable base rather than immediately scheduling another cosmetic continuation. `current-state.md` is now four kernels stale (79/79A/80/81).

## Fix — Missing `permission_requests` table (2026-08-07)

**Status: fixed, deployed live, uncommitted.** Not a kernel — a bug report ("made a new account on my wife's computer... could not get auto-permission to enter Catharsis") led straight to a genuine, long-standing defect, not a permissions misunderstanding.

- `permission_requests` — read and written by `backend/internal/identity/requests.go` and `permissions.go` (and read, with a silently-tolerated failure, by `access/visibility.go`'s notification-count query) since the feature was first built back in the pre-kernel "Kernel 3" era (commit `35348da`, 2026-03-30) — **had no creating migration at all.** Confirmed missing in both the live production database and the test database; every `INSERT INTO permission_requests` has been failing with a 500 since the feature shipped.
- This meant `/api/requests/create` — including Audition Hall's own documented default (venue=Catharsis, role=cast, "The default here is auto-access for Catharsis") — has never actually worked for anyone. It went unnoticed because the only account that predates this fix is the Operator account, which never needed to submit a request itself; `visibility.go`'s read of the same table is wrapped in an error-tolerant `if err == nil` (a missing table there just skips notification badge counts), so the venue map and everything else kept working normally with zero visible symptom anywhere except this one specific flow.
- Verified directly against the live database: Grant's wife's new account (`k81`-adjacent timing, created 2026-08-06) had zero `location_memberships` and zero `access_grants` rows, exactly consistent with every submission attempt failing before ever reaching the auto-approve grant.
- Fix: `backend/migrations/093_fix_missing_permission_requests_table.sql`, a purely additive `CREATE TABLE`, schema derived directly from every column the existing (and previously correct, just untestable) Go code already expected — no application code changed. Deployed with the usual pre-migrate backup.
- Added the first-ever test coverage for this handler (`backend/internal/identity/permission_requests_missing_table_test.go`): a genuinely fresh account with no pre-existing memberships submits Audition Hall's exact default request and is proven to end up with Catharsis visible on their map afterward — the actual user-facing symptom, not just a row-existence check. Full backend suite re-run clean before and after deploy.
- Manually replayed the exact auto-approve grant for Grant's wife's real account directly against the live database (same INSERT sequence the now-fixed endpoint performs) so she doesn't need to resubmit through the UI — confirmed her account now has an active `cast` membership and a live Catharsis `access_grants` row.

**Lesson worth carrying forward:** a feature can be fully built — routed, tested-by-hand once by its author, documented in its own UI copy — and still be completely non-functional for every account except the one that happened to exist before it shipped, if nothing ever exercises it with a state that account doesn't have. See `operator-notes.md` for the durable version of this lesson.

## Kernel 81A — Storyboard Structural Slugs (2026-08-07)

**Status: PASS, deployed live, uncommitted.** Small, bounded cleanup requested directly (not a formal kernel spec doc): deterministic serialized slugs for Storyboard columns, bands, and rows, so duplicate human-facing labels (two columns both titled "Scene") never produce an ambiguous serialized identifier in export or any future integration.

- `backend/migrations/094_kernel81a_storyboard_structural_slugs.sql` adds a nullable `slug TEXT` to `storyboard_columns`/`storyboard_bands`/`storyboard_rows` plus partial unique indexes (`WHERE slug IS NOT NULL`) — columns/bands unique per board, rows unique per **band** (not board-wide), matching the explicit requirement and the existing `sort_order_in_band` precedent that rows are already fundamentally band-scoped.
- New `backend/internal/storyboards/slugs.go`: `SlugifyLabel` (deterministic, idempotent: "Scene" → "scene") and `allocateUniqueSlug`, a race-safe allocator — the real correctness guarantee is the DB's own unique index, the allocator is just a liveness mechanism so a lost race produces a retry with the next candidate, never a user-facing error. Collision behavior matches spec exactly: "Scene" → `scene`, a second → `scene-2`, and if `scene-2` already exists independently (e.g. a column literally titled "Scene 2"), a third "Scene" skips it and lands on `scene-3` rather than colliding.
- Slug is assigned once, at creation, in `AddColumn`/`AddBand`/`AddRow`/`CreateBoard`'s default-column/band/row path — never touched again by `RenameColumn`/`UpdateBandLabel`/`RenameRow`/reordering, satisfying "do not silently rewrite a stable serialized key."
- Export needed **zero code changes**: `ExportDocument` already embeds `StoryboardColumn`/`StoryboardBand`/`StoryboardRow` directly, so adding `Slug` to those structs made it appear in export automatically.
- Pre-existing boards backfilled via a new `EnsureKernel81AStoryboardSlugsSurface` bootstrap (same `Ensure*Surface` convention as every other kernel's boot-time surface, registered in `main.go`), reusing the identical `allocateUniqueSlug` algorithm so a backfilled slug is produced by the same deterministic logic a freshly-created object would get — not a second, possibly-diverging implementation. Verified live: all pre-existing columns/bands/rows on the production database had their `slug IS NULL` count go to zero after deploy, no other column touched.
- **A real, separate, out-of-scope bug was found while stress-testing the new slug allocator, not fixed (correctly, per this task's explicit "keep this bounded" instruction), and narrowly worked around so it doesn't corrupt the new feature's own correctness**: `storyboard_columns`/`storyboard_bands`/`storyboard_rows` each carry a Kernel-80-era `UNIQUE(scope, sort_order)` constraint, and their `sort_order`/`sort_order_in_band` value is computed via a plain `SELECT COUNT(*)` with no locking before the `INSERT` — a pre-existing, unsynchronized race under concurrent structural creation that nothing had ever exercised before this kernel's own concurrency test tripped it. The slug allocator's retry-on-conflict logic originally treated *any* unique-violation as "the slug candidate was taken, try the next one" — which silently misdiagnosed this unrelated sort_order collision and retried forever with different slugs that could never fix it, until exhausting its retry budget. Fixed by checking the actual Postgres constraint name before retrying (only a genuine slug-constraint violation is retried); the sort_order race itself is untouched and remains a real, live hazard for a future kernel to pick up if concurrent structural creation on the same board ever becomes a realistic usage pattern.
- 9 new tests (`kernel81a_slugs_dbtest_test.go`): deterministic/idempotent slugification, duplicate-label uniqueness (including the "independently existing scene-2" skip case), duplicate labels never rejected, stability across reload/rename/reorder, per-band row scoping, concurrent-creation race-safety (12 goroutines, zero duplicate slugs, verified against the DB directly), export correctness, and full pre-migration backfill safety (simulated NULL-slug legacy rows, confirmed zero data damage and idempotent re-runs). Full backend suite clean before and after deploy (only the pre-existing, documented `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent` flake appeared on a non-reset re-run, as previously noted in Kernel 80/81's own reportbacks).

## Kernel 82 — Storyboards Timeline Mode (2026-08-07)

**Status: PASS, deployed live, uncommitted.** Full ledger: `Construction/OperatorLogs/kernel-82-reportback.md`; five new design docs in `Construction/Storyboards/`.

- Timeline ships as a built-in, code-defined (not DB-row-backed) Storyboard template: choosing it instantiates a normal owned board with three columns (Beginning/Middle/Ending — Beginning and Ending carry a new `column_role`, a genuine structural property enforced server-side in `ReorderColumns`/`RemoveColumn`, not just a UI convention), one default band/row, and a four-field Reference Panel (Premise, Beginning, Ending, and one paired-list field combining Include/Exclude as its two sides). Two Timelines are provably independent — editing one's Reference Panel, columns, or anything else never touches another instance or the template itself, proven directly by a dbtest that creates three Timelines in sequence and edits two of them.
- The Reference Panel is new, generic Storyboards infrastructure (not gated to Timeline mode at all — a Blank board can get one too), reusing Kernel 81A's slug-allocation machinery a third time without any modification to it, and reusing the exact same Crew-content/Director-structure authority split cards already use rather than inventing a third permission tier.
- Middle-mouse panning was added to the board viewport via Pointer Events (the same event model Kernel 81's card drag already uses), verified live on both axes without disturbing wheel scroll, scrollbars, or card drag-and-drop — the one real gap the browser proof surfaced (vertical panning initially showed no movement) turned out to be a test-board-had-no-vertical-overflow-yet artifact, not a code defect, confirmed by a follow-up run against a board with genuine vertical overflow.
- Explicitly did not build: Microscope terminology/mechanics, a card tone system, placement assistance, Group Leader/Current Turn state, or Save-as-Template — all named non-goals in the spec. A documented integration seam (`session-state-integration-seam.md`) exists for a future Group Leader/Current Turn platform kernel to plug into the Reference Panel's rendering layer without Storyboards needing to know its implementation.
- 18 new backend tests, all green on the first real run against the database; full Go suite clean before and after a live deploy; 20/20 required Playwright browser-proof scenarios confirmed with screenshots against the live production instance; all test data (3 throwaway accounts, 4 test boards) cleaned up afterward with zero residue verified directly in the database, including through the new Reference Panel tables.

### Next recommended step
Not a new kernel — Grant's own live walkthrough of Timeline creation, boundary/insert behavior, the Reference Panel (including the paired list), and middle-mouse panning on a real board. `current-state.md` is now six kernels stale (79/79A/80/81/81A/82) — a consolidated documentation-catch-up pass across all of them, flagged repeatedly across this cycle's reportbacks, remains the standing recommendation whenever there's a lull in active Storyboards feature work.

## Kernel 83 — Venue Leadership & Turn State (2026-08-08)

**Status: PASS, deployed live, uncommitted.** Full ledger: `Construction/OperatorLogs/kernel-83-reportback.md`; three new design docs in `Construction/Venues/` (new directory).

- Group Leader and Current Turn ship as a genuinely generic platform primitive, split into two layers on purpose: `backend/internal/venuecoordination` is a small in-memory `Registry` that knows nothing about roles, tiers, or what a "venue" even is, while `backend/internal/storyboards/coordination.go` is Storyboards' own authority-checked wrapper around it. A future venue opting in writes its own thin wrapper file rather than teaching the generic package anything venue-specific — the whole point being that Storyboards never becomes "the owner" of this feature the way a less disciplined implementation might have let it.
- **Storyboards had zero Presence Tray before this kernel** — Kernel 80's own `ws.go` comment said outright "Storyboards has no location/session/presence concept." This kernel built the first one from scratch (`#presence-tray` in `board.html`), reusing `network.Hub.BoardWatcherUserIDs` (a primitive Kernel 80 built for an unrelated purpose — per-viewer hidden-card event fan-out) as its live "who's here" roster.
- "Live venue session," which Storyboards also had no concept of, was defined as the period during which ≥1 distinct user is watching a given board — the 0→1 and 1→0 watcher-count transitions on `watch_board`/disconnect are the only session start/end triggers that exist for this venue. Full rationale, including why The Cave's unrelated Production/Show-scoped session concept was deliberately *not* reused, in `Construction/Venues/venue-session-state-lifecycle.md`.
- Right-click on a Presence Tray chip is the only mutation surface, exactly as specified: **Make Group Leader** / **Give Turn**, shown only when the acting user's own authority (Director+/owner, current Group Leader, or — for turn — current Current Turn holder) allows it, and independently re-verified server-side on every POST regardless of what the menu showed (proven directly by forging API calls from unauthorized accounts in the browser proof, not just checking the menu was hidden).
- Presence Tray ordering is sorted by handle server-side, specifically so it can never appear to reorder itself as leader/turn state changes — verified in the browser proof by capturing chip order before and after a live Give Turn action and asserting exact equality.
- Kernel 82's previously-unwired Reference Panel integration seam is now live: a small read-only "Group Leader: X / Current Turn: Y" slot renders at the top of the panel whenever a session is active, absent entirely (not disabled) otherwise, and confirmed absent from Timeline JSON export both at the Go test level and via a live fetch of the real export endpoint in the browser proof.
- **Password signup is now closed on the live site** (`password_signup_closed` — Discord-only account creation), which broke the disposable-account creation method every prior Storyboards kernel's Playwright proof relied on. Worked around using the field guide's own documented fallback (direct `users` + `auth.sessions` fixture-row insertion, raw token hashed to match `sessions.HashToken` exactly) — worth updating `kernel-maker-field-guide.md` to lead with this method now.
- 22 new backend tests (6 pure-Go registry unit tests + 16 real-WebSocket dbtests covering every spec §11 sub-requirement plus an explicit export-exclusion regression), all green on first full run; full Go suite clean on a freshly reset test database; 38/38 Playwright assertions passed against the live production instance (all 25 required spec scenarios plus 13 setup/plumbing checks), including two scenarios that specifically forge unauthorized direct API calls rather than relying only on UI-hiding. Five disposable accounts and one test board deleted afterward with zero residue verified directly in the database. No migration was needed — this is the first Storyboards kernel since 80 that added no schema, by design (the state is genuinely ephemeral and cleared on backend restart).

### Next recommended step
Not a new kernel — Grant's own live walkthrough: open a Storyboard with a second account, confirm the Presence Tray appears, right-click to assign Group Leader/Current Turn, and check the Reference Panel slot on a Timeline board. Two small, genuinely optional follow-ups flagged rather than done here: a keyboard entry point for the Presence Tray context menu (right-click-only is a named, spec-acknowledged accessibility gap), and updating the kernel-maker field guide's disposable-account technique now that password signup is closed.

## Kernel 84 — Canonical Reconciliation & Runtime Cleanup (2026-08-08)

**Status: PASS, deployed live, uncommitted.** Full ledger: `Construction/OperatorLogs/kernel-84-reportback.md`; history ledger: `Construction/OperatorLogs/kernel-history-reconciliation-through-83.md`; roadmap fully reconciled.

- Fixed the real bug Kernel 83 flagged but only worked around for its own disconnect path: `storyboards/ws.go`'s `ServeStoryboardWS` was reusing a single 5-second handshake-scoped context for a connection's entire life, so any `watch_board`/board-switch more than 5 seconds into a connection silently failed. Now: a setup-only context for the handshake, a cancel-only connection-lifetime context, and a fresh bounded context per inbound message — matching the pattern `network/ws.go`'s venue socket already used correctly. Adjacent audit found no second instance anywhere else (the Player Workbook socket does no DB work in its loop at all; Discord bridge/gateway already correct).
- Found and fixed the real root cause of `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent`'s flake, which three prior kernels' reportbacks (81A, 82, 83) had all attributed to vague "accumulated test-DB state." It was a test-isolation bug: the test hardcoded "the edit must be revision 2," which only held on a freshly-reset database — running the full suite twice in a row without a reset reproduced the failure on demand. Now asserts relative to the revision actually loaded; verified robust across repeated runs.
- Recovered the real post-Kernel-75 sequence from 12 reportbacks read in full (76, 77, 77A, 78, 79 in four separate passes, 80, 81, 81A, 82, 83) and reconciled the single Canonical Roadmap through Kernel 84: new baseline band, V3/V6/V9/A1/C4/O4 rewritten from stale "required capabilities" to actual Established/Open status, a new V11 for the Group Leader/Current Turn primitive, 9 new skip/defer entries, and a new committed horizon (Kernel 85 — Socio Sustained Play, Kernel 86 — Cartograph-Style Drawing Foundation; no invented third slot). Both old planning documents (Master Actual Implementation Guide, Parallel Track Roadmaps) marked unmistakably superseded, not deleted. `current-state.md` brought current from its Kernel-78 freeze.
- Kernel 84's own required browser regression pass (not a re-proof of any earlier kernel, a focused smoke) surfaced a real, new, narrowly-scoped bug: Storyboards' generic "+ Column (end)" action has no awareness of Timeline's protected boundary columns and will append an ordinary column after `Ending` — only the UI's per-column "Insert left/right" avoids this, via a client-side create-then-reorder dance. Investigated fully, documented in `current-state.md`/the roadmap/the cleanup ledger, deliberately not fixed (touches the same structural sort_order mechanism Kernel 81A already flagged a separate unfixed race in — out of this kernel's narrow WS/reconciliation scope).
- **A commit landed on `main` mid-session that this session did not make** (`90754da`, "Implement Kernel 83 venue coordination...") — captured Kernel 83's finished work plus whatever Kernel 84 files existed at that instant. Nothing lost; flagged transparently in the reportback rather than silently worked around or rewritten.
- 17/17 live browser regression assertions passed (Blank+Timeline Storyboards, 2-user coordination, eWrite reader/directory links, one forged negative-authority API call rejected `403`); full Go suite green on a freshly reset database and again on a second consecutive run without reset; fresh-install bootstrap proven from empty (95 migrations, 19 venues, 4 eWrite collections, 1 directory).

### Next recommended step
Per the reconciled roadmap: **Kernel 85 — Socio Sustained Play**, then **Kernel 86 — Cartograph-Style Drawing Foundation** — neither fully specified inside Kernel 84 by design; each needs its own audit of current state first. Optional, non-blocking follow-ups: the Timeline `AddColumn` boundary fix, a Presence Tray keyboard entry point, and deciding whether to amend/re-split the mid-session commit noted above.
