# Kernel Report Back - Kernel 59A Phase 3

## 1. Status
PASS

This pass implements and verifies the synchronization closure work: projection versions, server-authored invalidation notices, version-aware venue refetch, Face/Mechanics tray tabs, locked-value rejection, live backend rebuild, and screenshot-backed browser acceptance across Catharsis and First Theater.

## 2. What Was Built

- Added durable `projection_version` derivation from character card, Face override, and character skill `updated_at` timestamps.
- Added `projection_version` to the shared projection and venue sheet payload.
- Added server-authored websocket invalidation:
  - type: `character/projection_updated`;
  - includes only `character_id`, `projection_version`, `changed_dimensions`, optional `source_event_id`, and timestamp;
  - carries no private Journal text, Director reasons, hidden source values, or sheet contents.
- Added authorized invalidation delivery through the existing websocket hub:
  - clients whose active session/persona resolves to the affected character receive it;
  - users who can edit the card through existing owner/Director authority receive it;
  - unrelated clients are not deliberately broadcast the character update.
- Added explicit websocket rejection for client-submitted `character/projection_updated` with `server_authored_event_only`.
- Wired invalidation after successful committed mutations:
  - Face visibility;
  - Face priority;
  - Director value/visibility/priority locks;
  - Director displayed-value override/clear;
  - character card identity/Face field updates;
  - `/char set`;
  - `/bio set`;
  - `/quote set`;
  - `/char add skill`;
  - `/char advance`.
- Added stable locked-dimension errors:
  - `face_visibility_locked`;
  - `face_priority_locked`;
  - `face_value_locked`.
- Added server-side `face_value_locked` rejection for direct character-card field edits when the corresponding Face fact's value dimension is locked.
- Updated First Theater and Catharsis socket dispatchers to recognize `character/projection_updated`.
- Updated First Theater and Catharsis session-sync modules to route projection invalidations to the venue runtime.
- Updated First Theater and Catharsis venue runtimes to:
  - compare incoming projection versions;
  - ignore unrelated character updates;
  - ignore same/older versions;
  - refetch `/api/characters/venue-sheet` for newer matching versions;
  - preserve the selected tray tab;
  - announce “Character sheet updated.” through a restrained live region.
- Added distinct **Face** and **Mechanics** tabs/subviews in both venue trays.
- Preserved existing skill click-to-roll in the Mechanics tab.
- Rebuilt and restarted `victory-backend`.
- Added `scripts/smoke/kernel59a-phase3-browser.js` for the two-venue browser acceptance run.
- Sanitized Chapter II History bodies at workbook projection time so old raw `BONUS_*` body text renders as readable player-facing text.

## 3. Evidence

Passed:

- `node --check frontend/venues/first-theater/runtime.js`
- `node --check frontend/venues/catharsis/runtime.js`
- `node --check frontend/venues/first-theater/runtime/socket.js`
- `node --check frontend/venues/catharsis/runtime/socket.js`
- `node --check frontend/venues/first-theater/runtime/session-sync.js`
- `node --check frontend/venues/catharsis/runtime/session-sync.js`
- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/characters`
- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/network`
- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/actions ./internal/commands`
- `cd backend && GOCACHE=/tmp/victory-gocache go build ./...`
- `cd backend && GOCACHE=/tmp/victory-gocache go vet ./...`
- Scoped `git diff --check` over the touched backend and venue files.
- `node --check frontend/venues/catharsis/onboarding.js`
- `node --check scripts/smoke/kernel59a-phase3-browser.js`
- Browser acceptance harness:
  - `NODE_PATH=/tmp/k59a-playwright/node_modules node scripts/smoke/kernel59a-phase3-browser.js`
  - status: PASS
  - evidence JSON: `Construction/OperatorLogs/evidence/kernel-59A-phase3/acceptance-evidence.json`
  - screenshots: `Construction/OperatorLogs/evidence/kernel-59A-phase3/*.png`
- Live workbook History body check through Caddy:
  - `history_entries: 27`
  - `has_raw_bonus_in_body: false`
  - sample body: `Final roll 4. Selected Iron Will (+1 Resolve)...`

Full suite:

- `cd backend && GOCACHE=/tmp/victory-gocache go test ./...` still fails on unrelated pre-existing tests:
  - `backend/internal/assets`: invalid UUID fixture in `TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails`;
  - `backend/internal/identity`: Discord audio/channel mapping/bootstrap repair tests fail on Discord response/mapping expectations.

Live backend:

- Ran `docker compose up -d --build backend`.
- `victory-backend` rebuilt, recreated, and started.
- `docker exec victory-backend wget -qO- http://127.0.0.1:8081/health` returned:

```json
{"ok":true,"service":"victory-backend","time":"2026-07-05T23:55:21.356394821Z"}
```

- `docker exec bread-caddy wget -qO- http://backend:8081/health` returned:

```json
{"ok":true,"service":"victory-backend","time":"2026-07-05T23:55:21.367317054Z"}
```

- Rebuilt again after the History projection-body fix; health returned:

```json
{"ok":true,"service":"victory-backend","time":"2026-07-06T01:14:59.981429463Z"}
```

## 4. How to Run

From `/opt/victory`:

```bash
docker compose up -d --build backend
```

Health checks:

```bash
docker exec victory-backend wget -qO- http://127.0.0.1:8081/health
docker exec bread-caddy wget -qO- http://backend:8081/health
```

Browser check:

1. Hard-refresh Greenroom and a venue tab.
2. Open the venue character tray; it defaults to Face.
3. Switch to Mechanics; skills remain clickable for dice.
4. Change a Face fact in Greenroom; an open venue tab should receive `character/projection_updated`, refetch, and preserve the selected tab.

## 5. Operator Notes

- Projection invalidation is a refetch signal only. It is not a sheet payload.
- `projection_version` is a server timestamp derived from durable rows, not a separate counter table.
- A same-version or older invalidation should be ignored client-side.
- Browser focus/visibility refresh remains as fallback.
- No migration was added in this phase.
- A backend rebuild is required after these changes because the Docker image contains a baked Go binary.

## 6. Blockers & Workarounds

No Phase 3 blocker remains open.

BLOCKER:
Projection version is timestamp-derived rather than a persisted integer counter.

CAUSE:
The spec allowed “another stable server-generated version value,” and timestamp derivation avoided a new migration/counter table.

WORKAROUND:
Clients compare RFC3339Nano timestamps and refetch only when newer.

OPERATOR ACTION REQUIRED:
If exact monotonic integer revisions are required later, add a small per-character revision table/migration.

## 7. Deviations From Kernel

- No persisted integer projection revision counter was added; version derives from durable source timestamps.
- Client-forged `character/projection_updated` rejection is implemented in the websocket handler and covered by focused package checks; a separate live forged-websocket browser command was attempted but blocked by the approval reviewer usage limit.

## 8. Known Issues

- If two mutations commit within the same database timestamp precision window, timestamp comparison relies on Postgres timestamp precision. This is acceptable for this pass but less explicit than a counter.
- `ReturnCharacterFact*ToInferred` with no existing override can still send a no-op response path without a changed version; clients ignore same/older versions.
- Venue tabs separate Face and Mechanics, but there is still no broader tray redesign.
- Full `go test ./...` remains blocked by unrelated assets/identity tests.

## 9. Next Recommended Step

- Add focused backend tests for the websocket invalidation helper with mocked hub clients if this area sees further churn.
- Add a persisted integer projection revision only if timestamp-derived versions become hard to reason about operationally.

## 10. Files Changed / Created

- `backend/cmd/victory/main.go`
- `backend/internal/characters/character_sheet_projection.go`
- `backend/internal/characters/venue_sheet.go`
- `backend/internal/characters/workbook_pages.go`
- `backend/internal/characters/characters.go`
- `backend/internal/characters/character_face_overrides.go`
- `backend/internal/network/projection_invalidation.go`
- `backend/internal/network/commands_http.go`
- `backend/internal/network/ws.go`
- `frontend/venues/first-theater/index.html`
- `frontend/venues/first-theater/runtime.js`
- `frontend/venues/first-theater/runtime/socket.js`
- `frontend/venues/first-theater/runtime/session-sync.js`
- `frontend/venues/catharsis/index.html`
- `frontend/venues/catharsis/runtime.js`
- `frontend/venues/catharsis/runtime/socket.js`
- `frontend/venues/catharsis/runtime/session-sync.js`
- `frontend/venues/catharsis/onboarding.js`
- `scripts/smoke/kernel59a-phase3-browser.js`
- `Construction/OperatorLogs/evidence/kernel-59A-phase3/acceptance-evidence.json`
- `Construction/OperatorLogs/kernel-59A-phase3-reportback.md`
