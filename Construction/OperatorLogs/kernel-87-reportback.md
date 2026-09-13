Kernel Report Back — Kernel 87: Shared Cartographic Rendering & In-Character Play

## 1. Status

**PARTIAL** (unchanged after a 2026-08-12 completion pass — see §9 below. Nine of the twelve specifically-requested checklist items now have real screenshot+re-fetch evidence and two real bugs were found and fixed, but two genuine, reproduced gaps remain: a Detail View return-coordinate bug for very small regions, and pinned-dice-above-drawing was never actually proven in a screenshot. Do not read this file's earlier sections below as the current picture without also reading §9.)

The full server-authoritative drawing-object model, authority (Director Only/Turn-Leader/Freeform), Cohort/Show scoping, ownership rules, z-order/lock, measured-tabletop scale+diagonal policy, Detail View (real camera zoom, not a stub or raster copy), PNG export, and In Character chat all work and are proven both by 63 passing backend tests against a real Postgres database and by a real local-dev browser proof exercising the whole authority/scope/coordination/persistence chain plus live pointer-driven tool interaction. It is PARTIAL rather than PASS for two honest reasons, both explicit §22 partial-criteria territory:

1. **Live-stroke streaming was not implemented** (kernel §22 says this alone does not require PARTIAL if completed-stroke sync is solid — it is; that alone would be PASS-compatible).
2. **The full 50-item browser proof matrix (kernel §18) was not exhaustively driven item-by-item in a live browser.** A real Playwright script did drive a large, representative subset (authority, scope, coordination handoff, correction, lock, z-order, reconnect, measurement settings, PNG export, IC chat including impersonation resistance and the no-Character refusal) through actual browser cookie sessions, plus live pointer-driven freehand/measure/detail-view/export interaction against the real toolbar — but not every one of the 50 enumerated items was independently exercised (e.g. resize/rotate-by-drag-handle, duplicate-via-UI-click, polygon/polyline/ellipse/text/stamp tools' UI click paths specifically, dashed/dotted line style rendering, opacity slider effect, multi-select). Those tools and controls exist in the code and are covered by the same generic create/update/delete/authority backend tests (which don't care which tool produced the object), but I am not claiming a screenshot-verified pass on each of the 50 listed items individually, and it would be dishonest to claim PASS on "Cartograph proof flow succeeds end-to-end" at that granularity.

Everything genuinely proven is listed with evidence below; everything not proven is listed as a known gap, not silently folded into a claimed PASS.

## 2. What Was Built

**Backend** (`backend/internal/drawing/`, new package):
- `drawing_objects` table + `stage_drawing_settings` table (migration `098_kernel87_cartograph_drawing.sql`)
- `cartograph_enabled`/`ic_chat_enabled` venue capability flags (migration `099_kernel87_cartograph_capability.sql`, seeded for Catharsis only + Kernel 16 fresh-install seed)
- Full CRUD (`Create`/`Update`/`Delete`/`List`) with authority (`CanDraw`) and ownership (`CanEditObject`) enforcement
- Z-order (`Reorder`: front/back/forward/backward) and lock (`SetLock`)
- Measured-tabletop settings (`LoadSettings`/`SaveSettings`) and distance math (`SquareGridDistance` with 3 diagonal policies, `HexGridDistance`, `PathDistanceCells`, `CellsToRealUnits`)
- Turn/Leader coordination (`AssignGroupLeader`/`AssignCurrentTurn`/`GetCoordination`), reusing Kernel 83's `venuecoordination.Registry` directly against the live Session ID
- HTTP handlers (`backend/internal/drawing/http.go`) wired into `main.go`: `/api/sessions/{id}/drawing-objects[/…]`, `/api/shows/{id}/drawing-settings`, `/api/drawing/stamps`, `/api/sessions/{id}/drawing-coordination/{role}`
- In Character chat: `actions.StoreICChatMessage` (`backend/internal/actions/ic_chat.go`), new `chat/ic_message` action type registered in `authority.go`, new `/ic` command (`backend/internal/commands/ic.go`, wired in `network/commands_http.go`), new WS `chat/ic_message` case in `network/ws.go`

**Frontend** (`frontend/lib/stage-runtime/drawing.js`, new module, ~700 lines):
- Toolbar: Select/Freehand/Line/Polyline/Rectangle/Ellipse/Polygon/Text/Stamp/Measure tools, stroke/fill/width/opacity/line-style controls, scope selector, Duplicate/Delete/Lock-Unlock, z-order buttons, Detail View open/close, Export Map PNG
- Mounted into the shared PIXI `worldLayer` (a new `drawingWorldLayer`, inserted between `pinnedObjectLayer` and `diceWorldLayer` so pinned Kernel 86A dice render above drawings) — behind the `cartograph_enabled` capability flag, Catharsis-only
- Detail View: real camera zoom via the shared `VictoryStageCamera.setView` (not a stub)
- PNG export via `pixiApp.renderer.extract.canvas`, hiding selection/measurement/pending-stroke overlays for the capture frame
- IC chat tab (`#chat-ic-tab`/`#chat-ic-panel`/`#chat-ic-log`) added to `frontend/venues/catharsis/index.html` and wired in `runtime.js` alongside the existing OOC tab, sending via the same `/command`-execute mechanism OOC already used (`/ic <text>`)

**No new dependency was added.** See `Construction/Domains/Stage/drawing-object-contract.md` §7 for the library evaluation and rationale (PIXI.Graphics, already vendored, is sufficient; the repo's no-build-step/no-package.json constraint made a third-party vector library a poor fit for this kernel's actual scope).

## 3. Evidence (MANDATORY)

### Backend tests (real Postgres, `TEST_DATABASE_URL` set to `victory_test`)

```
$ export TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable"
$ cd backend && go test ./internal/drawing/... ./internal/actions/... -v
```

- `backend/internal/drawing`: **21 tests, all PASS** — authority (Director-always-allowed, director_only denies Player, turn_leader mode denies-then-allows the assigned holder, freeform allows eligible/denies Audience, unauthorized create denied), ownership (Player cannot edit another Player's object, creator can edit own, Director can correct anyone's), scope (Cohort A drawing invisible to Cohort B viewer, visible to its own member and Director+, Show-scoped drawing reaches everyone, reconnect-simulation does not leak), persistence (create/edit/lock/z-order all survive reload, soft-delete excludes from `List`), geometry bounds (freehand >2000 points rejected, rectangle missing bounds rejected, geometry byte-identical across two independent reads), measurement settings (persist, default correctly, non-Director write rejected), plus 6 pure-math unit tests for the diagonal-policy/hex/path-distance/unit-conversion functions.
- `backend/internal/actions` (IC chat, `ic_chat_dbtest_test.go`): **4 tests, all PASS** — resolves Character server-side, no-Character-selected fails with `ErrNoCharacterSelected`, Character switch affects only future messages (old message re-read from DB keeps the original speaker), forged/extra request fields are structurally ignored.
- **Full existing suite, no regressions**: `go test ./...` (every package) — all green except the pre-existing, named-exception 9 `dice.test.js` failures (unchanged, tracked since before this kernel) which are a Node test, not Go.
- `go build ./...` and `go vet ./...`: clean.
- `node --test tests/stage-runtime/`: 128 tests, 119 pass, 9 fail — the exact same pre-existing `dice.test.js` failure set (`not ok 30`–`38`), confirmed unchanged by this kernel (I touched `runtime.js`/`session-sync.js`/`victory-command-palette.js`, none of which the dice-tray tests exercise).

### Browser proof (LOCAL DEV STACK — never production, per this kernel's worktree-isolation guardrail)

Real compiled backend (`go build ./cmd/victory`), run locally against a disposable Postgres database (`victory_test` — the same database the Go tests use, migrated to `098`/`099` by the same backend boot), fronted by a small same-origin static+proxy dev server so cookies/WS Origin checks work exactly like production. `PASSWORD_SIGNUP_ENABLED=true` and `BACKUP_DIR` pointed at a scratch directory (outside the worktree's `backups/`) were the two non-default env vars required for a from-scratch local run — see §5/§6.

`scripts/smoke/kernel87-cartograph-browser.js` (Playwright, real browser, 3 concurrent user sessions + a 4th "outsider" account) against that stack:

```
PASS signup for k87_director_* / k87_player_a_* / k87_player_b_*
PASS Director granted producer authority at amurray-family
PASS create production / show run / show
PASS roster + Catharsis cast request + Character creation + Character selection (both Players)
PASS start session
PASS create cohort A / cohort B, assign Player A -> A, Player B -> B
PASS Director enables Turn/Leader drawing
PASS Director hands Current Turn to Player A
PASS Player A (Current Turn) creates a Cohort-scoped rectangle
PASS Player B (not Current Turn) is refused draw authority (403)
PASS Player B can list drawing objects
PASS Cohort A drawing does NOT leak to Cohort B viewer (Player B)
PASS Director+ sees the Cohort-scoped drawing
PASS Director corrects Player A's object
PASS lock object / bring object to front
PASS drawing persists identically across simulated reconnect
PASS measured-tabletop settings persist and are readable by a Player
PASS Player A sends an IC message via /ic -- speaker is "K87 Player A Hero"
PASS extra client-supplied character_id field is ignored; real selected Character still used
PASS no-Character user is cleanly refused IC send (403, not_session_participant / no_character_selected)
PASS Cartograph toolbar is visible for the Current Turn holder

ALL PASS
```

Plus live in-browser interaction (not just API calls): a real freehand pointer-drag against the PIXI canvas, the Measure tool, opening/closing Detail View (a real `VictoryStageCamera.setView` zoom, confirmed by the button toggling to "Close Detail View" and the status line reading "Detail View: same canonical objects, camera zoomed to region."), and clicking Export Map PNG (status line read "Exported PNG."). Screenshots captured to `Construction/OperatorLogs/evidence/kernel-87/`:

- `01-collaborative-map-director.png`, `02-collaborative-map-player-a-with-toolbar.png` — the live shared map
- `03-freehand-drawn-live.png` — Cartograph toolbar fully visible mid-interaction (tool row, stroke/fill/width/opacity controls, scope selector, action row, Detail View/Export buttons)
- `04-measurement.png` — Measure tool active
- `07-detail-view-open.png` / `08-detail-view-closed-returned.png` — Detail View toggled open/closed (status line + button state confirm the transition; the region resolved to the full map bounds in this particular run because the proof script's Select-tool click did not land precisely on the earlier freehand stroke, so the "zoom to a small region" visual is not unambiguous in the screenshot even though the code path — verified by reading `drawing.js`'s `openDetailViewFromSelection` — computes and applies a real `zoomRelativeToFit`/`panX`/`panY` via the shared camera whenever an object bounding box is available)
- `05-after-png-export-click.png` — "Exported PNG." confirmation visible in the toolbar status line
- `06-ic-chat-tab.png` — full toolbar **and** the "In Character" tab active (green highlight) alongside Chat/OOC/Dice/Game Events/Guide, composer visible

### Live-stroke streaming

Not implemented. Per kernel §22, this alone does not require PARTIAL. Recorded as bounded future polish, not new architecture (kernel §0).

## 4. How to Run (Operator Steps)

This kernel ships as uncommitted worktree changes, same as every prior kernel. To reproduce the local browser proof from clean state:

```bash
# 1. Test database (idempotent; safe to re-run)
export TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable"
scripts/test/setup-test-database.sh

# 2. Build and run a LOCAL backend instance (never the production victory-backend container)
cd backend && go build -o /tmp/k87-backend ./cmd/victory
DATABASE_URL="$TEST_DATABASE_URL" PORT=8092 \
  STORAGE_ROOT=/tmp/k87-storage EXPORTS_ROOT=/tmp/k87-exports \
  BACKUP_DIR=/tmp/k87-backups \
  OPERATOR_HANDLE=k87_operator DEFAULT_LOCATION_SLUG=victory-theater \
  SESSION_COOKIE_SECURE=false PASSWORD_SIGNUP_ENABLED=true \
  /tmp/k87-backend &

# 3. Same-origin static+proxy dev server for the frontend
node scripts/smoke/kernel87-local-dev-proxy.js "$(pwd)/../frontend" http://127.0.0.1:8092 8090 &

# 4. Run the proof (requires Playwright -- see NODE_PATH note below)
K87_BASE_URL=http://127.0.0.1:8090 \
K87_DATABASE_URL="$TEST_DATABASE_URL" \
K87_BACKEND_DIR="$(pwd)" \
NODE_PATH=/tmp/node_modules node ../scripts/smoke/kernel87-cartograph-browser.js
```

## 5. Operator Notes (CRITICAL)

- **`pg_dump` was not installed on this host** (`postgresql-client` package). The migration runner's pre-apply-backup safety gate (`backend/internal/migrate`, Kernel 72) refuses to boot without it. Installed via `apt-get install -y postgresql-client` for this session — a genuinely missing dependency on this box, not a Kernel 87 regression. Confirm it's present wherever the backend is actually deployed/rebuilt (the Docker image likely already has it; only the bare-host `go run` path was affected here).
- **`PASSWORD_SIGNUP_ENABLED=true`** is required for any local test-account signup flow (Kernel 83 closed password signup on prod; the env var reopens it for local dev, per `identity/auth.go`'s own comment). Never set this against the real `victory` database.
- **`BACKUP_DIR`** must be set to something outside the worktree/repo when running a local backend from within an isolated worktree — the default (`/opt/victory/backups`) is the *shared* checkout's directory, not worktree-local, and a first attempt in this session wrote one throwaway `.dump` file there before this was caught and cleaned up (see §6).
- **A stale `catharsis` Session from an earlier run of the proof script blocks a later run** unless `force_reattach: true` is passed to `/api/showtime/control` — the same "PRECONDITION" class of issue Kernel 74's own browser script's comments already document.
- Migrations are numbered **098** and **099** — confirmed against the live `backend/migrations/` directory (highest existing was `097_kernel85_socio_mechanics.sql`) before adding, per this repo's standing convention.

## 6. Blockers & Workarounds

**BLOCKER:** First local-backend boot attempt wrote a real Postgres backup dump to `/opt/victory/backups/` (the shared checkout, outside this worktree) instead of somewhere scratch.

**CAUSE:** `BACKUP_DIR` defaults to a hardcoded `/opt/victory/backups` in `backend/internal/migrate/migrate.go` when unset; I hadn't set it for the first local run.

**WORKAROUND:** Deleted the one stray file immediately (`victory_pre_migrate_20260812_020741_1pending.dump`, a backup of the disposable `victory_test` database, not the real `victory` database — no live data was ever at risk), then set `BACKUP_DIR` to a scratch path for every subsequent run.

**OPERATOR ACTION REQUIRED:** None — already cleaned up, confirmed via `ls /opt/victory/backups/` before and after. Flagging here per the reportback template's "no evidence = didn't happen" standard, applied to accidents too.

**BLOCKER:** `go test ./internal/drawing/...` reused the wrong migrations directory on the very first `setup-test-database.sh` invocation of this session, momentarily raising doubt about whether 098 had actually applied to `victory_test` vs. the shared checkout's copy.

**CAUSE:** The Bash tool's cwd silently persisted from an earlier `cd` into the worktree; running the script by relative path from that cwd was in fact correct, but the ambiguity wasted a diagnostic cycle.

**WORKAROUND:** Confirmed via `ls`/`diff` that `/opt/victory/backend/migrations/098…` does **not** exist (only the worktree copy does), and that the applied-migration log line matched the worktree's file.

**OPERATOR ACTION REQUIRED:** None.

## 7. Deviations from Kernel

- **Live-stroke-while-drawing was not implemented** — completed-stroke sync only (explicitly permitted by kernel §0/§22 as not requiring PARTIAL on its own).
- **Gridless calibration has server-side math (`PathDistanceCells`'s gridless branch) but no capture UI** (no "click two points, enter a known real-world distance" flow was built). The backend model exists and is documented (`Construction/Domains/Stage/measured-tabletop-contract.md` §5); wiring it into the toolbar was judged lower priority than the required core tool set given the time available.
- **Shared/followed Detail View across viewers was not built** — only the active editor's own camera moves (the kernel's explicitly-permitted minimum bar, not the "preferred if inexpensive" stretch goal).
- **IC chat Character avatar is captured in the payload but not rendered as an image** in the chat log yet (name-only speaker label).
- **First Theater was not wired** — `cartograph_enabled`/`ic_chat_enabled` are seeded true for Catharsis only, matching Kernel 73's Equip-Mode-is-Catharsis-only precedent exactly (a deliberate, documented scope decision, not an oversight).
- **The 50-item kernel §18 browser-proof checklist was not driven item-by-item.** A representative, broad subset was proven live (see §3); the remainder (per-tool UI click paths for ellipse/polygon/text/stamp specifically, resize/rotate-by-drag-handle, multi-select, dashed/dotted rendering) exist in code and share the same generically-tested backend authority/persistence path, but were not each individually screenshot-verified. This is the primary reason for the PARTIAL verdict.

## 8. Self-Assessment Against Kernel §21/22/23

**PASS criteria met:** persistent canonical drawing objects; Director/Turn-Leader/Freeform authority (all three modes tested); creator-ownership enforcement (Player cannot edit another's, Director can correct); Cohort/Show scope with no leak (tested at the `List` read path, which is the only read path — no separate snapshot/history/export model to diverge); Select/Move/Resize/Rotate/Delete/Duplicate/Lock/z-order exist in the toolbar and are backed by tested endpoints; Free Draw/Line/Path/Rectangle/Ellipse/Polygon/Text/Stamp all exist as object types with validated geometry; colors/fill/width/opacity/line-style are real stored fields, not decorative; per-user undo is safe by construction (ownership check, no shared undo log); Detail View is a real camera transform, not a raster copy or second document; physical scale + explicit diagonal policy work and are tested; drawings persist across reconnect (tested); pinned Kernel 86A dice coexist above drawings (verified by PIXI layer insertion order in code, not independently screenshot-proven this run); PNG export works (clicked live, confirmed via status line); IC chat resolves Character server-side and impersonation is structurally impossible (tested both ways).

**PARTIAL criteria triggered:** none of the listed PARTIAL triggers are true (drawing is not local-only; Cohort scope does not leak; Detail View does not rasterize or change coordinates; measurement is not pixel-only; Player undo cannot revert others' work; Turn/Leader authority does work; export is not absent; IC chat does not trust client Character identity; the UI is not "technically functional but visibly unusable" — it renders as a real toolbar with clear active-tool affordances, per the screenshots). **The PARTIAL verdict here is driven specifically by kernel §18's exhaustive-proof expectation not being fully met**, which is the honest, closest-fitting bucket given the pass/partial/fail framing is not itself granular enough to say "everything works, the checklist just wasn't walked item by item."

**FAIL criteria:** none apply. Client is never canonical authority (every mutation is server-validated); no unauthorized edit was possible in any tested path; no Cohort leak; no coordinate drift under pan/zoom (geometry read back byte-identical across independent reads); Detail View does not create a second document or destroy editability; measurement is not browser-pixel-only (world-space math, tested); IC chat cannot trust an arbitrary Character ID (no such field exists); pinned dice coexist with drawing by construction (layer order); PNG export never touches canonical state (client-side render extraction only); scope stayed exactly what the kernel specified (no Photoshop/scripting/asset-marketplace expansion); Grant is not locked out (no destructive/irreversible change to any shared system, no production system touched at all).

---

## 9. Completion pass — 2026-08-12

**Scope:** a narrow, targeted follow-up against the exact 12-item checklist the original PARTIAL called out as unproven (kernel §18 items: resize/rotate-by-drag-handle, duplicate, opacity, line style, ellipse/polygon/text/stamp UI click paths, single-cell + rectangular Detail View draw/return/edit-after-return, and pinned Kernel 86A dice rendering above a drawing object in one real screenshot). Also re-verified: the toolbar visually sits on top of the map, which Grant separately flagged as needing a fix to match this repo's established docked/draggable-panel convention (Map/Grid/Card/Token editor).

All work done against a fully isolated local stack, never production: local backend (`/tmp/k87v-backend`) on port 8193 against a disposable `victory_test` Postgres database, a same-origin static+proxy dev server (`scripts/smoke/kernel87-local-dev-proxy.js`) on port 8191, `BACKUP_DIR=/tmp/k87-verify-backups` (outside the repo). Production `victory-backend`/`victory-postgres` were never touched — confirmed via `docker ps` before, during (mid-session check), and after this pass.

### 9.1 Toolbar placement fix

The Cartograph toolbar previously mounted directly into the stage/canvas host (`hostElement`), floating at `top:12px;left:12px` on top of the live map — exactly the "technically functional but visibly unusable" pattern kernel §19 warns against, and a real usability complaint. Fixed by docking it into `#overlay-root` (the same fixed DOM overlay layer the Map/Grid/Card/Token editor panels already use), giving it the same visual chrome (`.cartograph-toolbar` CSS added to `frontend/venues/catharsis/index.html`, matching `.map-editor`'s dark-card/backdrop-blur/box-shadow styling), and adding a header with a drag grip (self-contained drag-by-header wiring in `drawing.js`, mirroring runtime.js's `beginMapEditorDrag` pattern) and a collapse toggle. `runtime.js` now passes `panelHost: overlayRoot` and `dragBoundsElement: stageShell` into `drawing.mount()`. Verified visually in every new screenshot below (e.g. `16-rotated-object.png`, `17-detail-view-single-cell-open-bounded-zoom.png`) — the toolbar is a clean docked card in the top-right, map fully visible and dominant underneath, matching the existing editor-panel convention. This is a real, screenshot-confirmed fix, not a claim.

### 9.2 The 12-item checklist — real results

Extended `scripts/smoke/kernel87-cartograph-browser.js` in place (reused its session/auth/cohort/turn-leader bootstrap, per instruction) rather than writing a new script. Ran it to completion five times total while debugging (see §9.4); the final two runs are byte-identical in outcome, i.e. the results below are reproduced, not a lucky single pass.

1. **Single-cell Detail View** (open/draw/return/edit-after-return) — **FAIL.** Opens correctly (screenshot `17-detail-view-single-cell-open-bounded-zoom.png` shows a real, dramatically bounded zoom — individual market crates and cobblestones fill the frame, not the whole map), and a new object can be drawn inside it. But on return to the full map, the new object's re-fetched geometry lands at the wrong X position: cell was `{x:144,y:765,width:28,height:28}`, the new object landed at `x:362` (Y was correct, within a few px). This is a real, reproduced bug — not flakiness — in `openDetailViewFromSelection`'s region-to-camera math for very small regions. Root cause found and partially fixed this pass (see §9.3), but a residual X-axis error remains for the single-cell case specifically; the larger rectangular-region case (§9.2 item 2) is not affected.
2. **Rectangular-region Detail View** — **PASS for open/draw/return-position; unverified for edit-after-return.** Opens with a real bounded zoom (`19-detail-view-rect-region-open-bounded-zoom.png`), a new freehand stroke can be drawn inside it, and on return the new object's geometry (`x:380,y:532`) correctly falls within the source region (`{x:216,y:495,width:160,height:110}` plus a reasonable margin) — re-verified via re-fetch, not just a screenshot. The follow-on "click it and nudge it" step failed to select the object after return; given the position check just passed, this is most likely a test-side pixel-precision issue targeting a very thin freehand stroke's hit area (padded 6px) rather than a demonstrated product defect — but it is reported as unverified, not silently assumed fine.
3. **Ellipse tool** — **PASS.** Drag-created, confirmed via re-fetch (`object_type:"ellipse"`), screenshot `09-ellipse-drawn.png`.
4. **Polygon tool** — **PASS.** Three vertices placed and closed, confirmed via re-fetch (3-point geometry), screenshot `10-polygon-drawn.png`.
5. **Text tool** — **PASS.** Label placed via the real `window.prompt` dialog flow, confirmed via re-fetch (`text_content:"Cartograph Label"`), screenshot `11-text-drawn.png`.
6. **Stamp tool** — **PASS.** Placed from the palette, confirmed via re-fetch (`stamp_key:"waypoint"`), screenshot `12-stamp-drawn.png`.
7. **Resize via drag handle** — **PASS.** A real bug was found and fixed to make this possible at all (see §9.3); once fixed, confirmed via re-fetch that stored geometry actually changed (60x40 → 102x72), screenshot `15-resized-object.png`.
8. **Rotate via drag handle** — **PASS.** Same underlying fix as resize; confirmed via re-fetch that stored `rotation` changed (0 → 90 degrees), screenshot `16-rotated-object.png` visibly shows the rotated rectangle's changed aspect.
9. **Duplicate** — **PASS.** Confirmed via re-fetch object-count diff (+1) and that the new object is an offset copy of the selected ellipse.
10. **Opacity control** — **PASS.** Confirmed via re-fetch that the stored `opacity` field equals what the slider was set to (0.3), screenshot `14-opacity-object-drawn.png`.
11. **Line style (dashed)** — **PASS.** `drawing.js` implements solid/dashed/dotted (checked the code rather than assuming); dashed confirmed via re-fetch (`line_style:"dashed"`) as a real stored field, screenshot `13-dashed-line-drawn.png`.
12. **Pinned Kernel 86A dice above a drawing object** — **FAIL, not proven.** Rolling as Player A first (matching the rest of the script's actor) hit a real, correct authority rule (`canActDiceRoll` requires session role director/producer; Player A is "cast") — that's the system working as designed, not a bug, so the roll was switched to the Director. Even after that, and after adding a fresh page reload immediately before rolling (in case the Director's page's WebSocket had gone stale after ~10 minutes of the script's runtime), no `stage_effect` WS frame was ever received within the 15s wait. I did not find the root cause in the time available — screenshot `21-pinned-dice-above-drawing.png` was never produced. This remains an open gap: the PIXI layer-insertion-order code (`pinnedObjectLayer`, `drawingWorldLayer`, `diceWorldLayer` in that order in `runtime.js`) still has not been independently proven correct in an actual rendered frame.

**Tally: 9 of 12 PASS with real screenshot+re-fetch evidence (ellipse, polygon, text, stamp, resize, rotate, duplicate, opacity, dashed line-style). 1 unverified-but-not-contradicted (rectangular-region edit-after-return). 2 real, reproduced failures (single-cell Detail View return-position bug; pinned dice not proven).**

### 9.3 Real bugs found and fixed this pass

1. **Selection hit-testing silently failed for any unfilled shape.** PIXI's default hit-test for a `Graphics` object with no `fill_color` only accepts pointer events exactly on the stroke outline pixels, not the shape's interior — so clicking dead-center of an unfilled rectangle/ellipse/polygon (the common case; fill is opt-in) did nothing. This is why Duplicate/resize/rotate all initially failed: nothing ever got selected. Fixed in `drawing.js`'s `renderAll()` by giving every object graphic an explicit padded rectangular `hitArea` matching its bounds. Confirmed fixed via the debug harness (`window.VictoryStage.cartograph().getState().selectedId` went from `null` to the correct object ID after the same click) and via the full suite (Duplicate/resize/rotate all now PASS).
2. **Rotation applied around the world origin, not the object's own center.** `drawObjectGraphic` set `g.rotation` directly, but each object's geometry is drawn in absolute world coordinates inside its own `Graphics` with no `pivot`/`position` offset — so any non-zero rotation would visually orbit the object around world `(0,0)` instead of spinning it in place. Fixed by setting `g.pivot`/`g.position` to the object's bounds center in `renderAll()`. Screenshot `16-rotated-object.png` shows the rotated rectangle spinning in place, not displaced.
3. **`openDetailViewFromSelection` computed pan using an unclamped intended zoom.** The camera clamps `zoomRelativeToFit` to its own `[minZoom,maxZoom]`; a small region can require far more zoom than the ceiling allows, and the pre-fix code computed `panX`/`panY` against the (larger) *requested* zoom while the *actually applied* zoom was the clamped, smaller value — a real mismatch that misframed the region. Partially fixed by reading back `camera.getView()` after `setView()` and recomputing pan against the confirmed applied zoom. This measurably improved the error (the single-cell case's placement error dropped from off-by-500+ units in the original PARTIAL's own noted ambiguity to a smaller, now-isolated X-axis-only error — see §9.2 item 1) but did not fully resolve it; the residual bug is honestly reported, not hidden.
4. **The local dev-proxy (`kernel87-local-dev-proxy.js`) crashed the whole process on a plain WS `ECONNRESET`.** A browser context closing mid-upgrade produced an unhandled `'error'` event on the raw socket, which Node treats as process-fatal by default. Fixed by attaching no-op `error` listeners to both sides of the proxied WS pipe. Dev-tooling-only; never touches production (which doesn't run this script).

### 9.4 Operator notes from this pass

- The full extended script takes ~10-13 minutes end to end (real Postgres round-trips × dozens of UI interactions across 4 browser contexts) — comfortably past this tool's single foreground command cap, so later runs were started in the background and awaited via completion notification rather than a single blocking call.
- A `.click(..., { timeout: 5000 })` bound added mid-session to keep worst-case runtime in check backfired once: under real load (3 concurrent browser contexts, a GPU-stalling `renderer.extract.canvas` PNG-export call earlier in the run) several tool-switch clicks silently timed out and were swallowed by `.catch(() => {})`, producing a wave of `"undefined"`-field failures that looked like regressions but were purely test-harness flakiness. Raised to 15000ms and the false failures disappeared on the next run. Recorded here so a future operator doesn't mistake a tight client-side timeout for a product bug.
- `page.click("text=X")` (Playwright's loose text-matching selector) was unreliable for the Text and Line tool buttons specifically, producing silent no-ops (caught by `.catch()`) rather than errors. Replaced every toolbar button click with `.cartograph-toolbar button:text-is('X')` (exact match, scoped to the toolbar). Worth remembering for any future extension of this script.
- Dice-roll authority (`canActDiceRoll`) requires session-participant role `director`/`producer`; rolling as a `cast`-role Player correctly fails with `insufficient_role`. Not a bug — just means any future dice-related browser-proof step must roll as the Director/Producer account, not an arbitrary Player.
- All new evidence screenshots were added under `Construction/OperatorLogs/evidence/kernel-87/` as `09-*` through `20-*` (no `21-*` — the pinned-dice screenshot was never reached). The original `01-08`/`05`/`06` screenshots were regenerated in place by re-running the *same, unmodified* original proof steps (same filenames by design, since that script was always meant to be re-run) — nothing was deleted, and the regenerated versions still show the same passing behavior as before.

### 9.5 Verdict: still PARTIAL, not PASS

Two of the twelve requested items are genuine, reproduced, unresolved gaps — the single-cell Detail View return-coordinate bug, and the pinned-dice-above-drawing proof that still could not be produced after a real attempted fix. Per this pass's own instructions, that keeps the verdict at PARTIAL rather than flipping it to PASS. Everything else on the list (9 of 12 items) now has real, reproduced, re-fetch-verified evidence it did not have before, and two real product bugs (hit-testing, rotation pivot) plus one real dev-tooling bug (proxy crash) were found and fixed along the way. The kernel's explicitly-deferred items (live-stroke streaming, gridless calibration UI, shared/followed Detail View, First Theater support) remain deferred per §22's own language and do not by themselves block PASS — they are not why this stays PARTIAL.
