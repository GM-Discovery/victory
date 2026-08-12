Kernel Report Back — Kernel 87: Shared Cartographic Rendering & In-Character Play

## 1. Status

**PARTIAL.**

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

**No new dependency was added.** See `Construction/Stage/drawing-object-contract.md` §7 for the library evaluation and rationale (PIXI.Graphics, already vendored, is sufficient; the repo's no-build-step/no-package.json constraint made a third-party vector library a poor fit for this kernel's actual scope).

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
- **Gridless calibration has server-side math (`PathDistanceCells`'s gridless branch) but no capture UI** (no "click two points, enter a known real-world distance" flow was built). The backend model exists and is documented (`Construction/Stage/measured-tabletop-contract.md` §5); wiring it into the toolbar was judged lower priority than the required core tool set given the time available.
- **Shared/followed Detail View across viewers was not built** — only the active editor's own camera moves (the kernel's explicitly-permitted minimum bar, not the "preferred if inexpensive" stretch goal).
- **IC chat Character avatar is captured in the payload but not rendered as an image** in the chat log yet (name-only speaker label).
- **First Theater was not wired** — `cartograph_enabled`/`ic_chat_enabled` are seeded true for Catharsis only, matching Kernel 73's Equip-Mode-is-Catharsis-only precedent exactly (a deliberate, documented scope decision, not an oversight).
- **The 50-item kernel §18 browser-proof checklist was not driven item-by-item.** A representative, broad subset was proven live (see §3); the remainder (per-tool UI click paths for ellipse/polygon/text/stamp specifically, resize/rotate-by-drag-handle, multi-select, dashed/dotted rendering) exist in code and share the same generically-tested backend authority/persistence path, but were not each individually screenshot-verified. This is the primary reason for the PARTIAL verdict.

## 8. Self-Assessment Against Kernel §21/22/23

**PASS criteria met:** persistent canonical drawing objects; Director/Turn-Leader/Freeform authority (all three modes tested); creator-ownership enforcement (Player cannot edit another's, Director can correct); Cohort/Show scope with no leak (tested at the `List` read path, which is the only read path — no separate snapshot/history/export model to diverge); Select/Move/Resize/Rotate/Delete/Duplicate/Lock/z-order exist in the toolbar and are backed by tested endpoints; Free Draw/Line/Path/Rectangle/Ellipse/Polygon/Text/Stamp all exist as object types with validated geometry; colors/fill/width/opacity/line-style are real stored fields, not decorative; per-user undo is safe by construction (ownership check, no shared undo log); Detail View is a real camera transform, not a raster copy or second document; physical scale + explicit diagonal policy work and are tested; drawings persist across reconnect (tested); pinned Kernel 86A dice coexist above drawings (verified by PIXI layer insertion order in code, not independently screenshot-proven this run); PNG export works (clicked live, confirmed via status line); IC chat resolves Character server-side and impersonation is structurally impossible (tested both ways).

**PARTIAL criteria triggered:** none of the listed PARTIAL triggers are true (drawing is not local-only; Cohort scope does not leak; Detail View does not rasterize or change coordinates; measurement is not pixel-only; Player undo cannot revert others' work; Turn/Leader authority does work; export is not absent; IC chat does not trust client Character identity; the UI is not "technically functional but visibly unusable" — it renders as a real toolbar with clear active-tool affordances, per the screenshots). **The PARTIAL verdict here is driven specifically by kernel §18's exhaustive-proof expectation not being fully met**, which is the honest, closest-fitting bucket given the pass/partial/fail framing is not itself granular enough to say "everything works, the checklist just wasn't walked item by item."

**FAIL criteria:** none apply. Client is never canonical authority (every mutation is server-validated); no unauthorized edit was possible in any tested path; no Cohort leak; no coordinate drift under pan/zoom (geometry read back byte-identical across independent reads); Detail View does not create a second document or destroy editability; measurement is not browser-pixel-only (world-space math, tested); IC chat cannot trust an arbitrary Character ID (no such field exists); pinned dice coexist with drawing by construction (layer order); PNG export never touches canonical state (client-side render extraction only); scope stayed exactly what the kernel specified (no Photoshop/scripting/asset-marketplace expansion); Grant is not locked out (no destructive/irreversible change to any shared system, no production system touched at all).
