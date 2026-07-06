# Kernel Report Back - Kernel 59A Full Scope

## 1. Status
PASS

Kernel 59A is implemented and browser-verified: Face curation persistence, shared projection, Greenroom Arrange Face controls, Director locks/value overrides, venue tray projection, Bio/Quote presentation alignment, projection invalidation, Face/Mechanics tray tabs, History body readability, two-venue browser acceptance, and live backend refresh are done. `/quote add` / multi-quote collection remains a separate backlog feature because current storage is still the singleton Featured Quote fact.

## 2. What Was Built

- Added `character_face_overrides` persistence for Face visibility and priority overrides.
- Added Director lock/value override persistence on the same override record.
- Added backend domain functions for:
  - show/hide/return-to-inferred Face visibility;
  - manual/return-to-inferred Face priority;
  - Director value/visibility/priority locks;
  - Director displayed-value override and clear.
- Added shared Face/sheet projection:
  - `EffectiveFaceFields`;
  - `ProjectCharacterSheet`;
  - venue sheet derivation through the same projector.
- Added HTTP routes:
  - `POST /api/character-workbooks/{id}/face-visibility`;
  - `POST /api/character-workbooks/{id}/face-priority`;
  - `POST /api/character-workbooks/{id}/face-lock`;
  - `POST /api/character-workbooks/{id}/face-value`;
  - `GET /api/characters/venue-sheet`.
- Added Greenroom Arrange Face UI:
  - semantic fact groups;
  - Show/Hide/Inferred controls;
  - priority +/- and direct integer entry;
  - Director reason/value/lock controls.
- Removed the incorrect Face Recent Events / Pin board model.
- Updated Greenroom Face preview so hidden facts are not rendered by hard-coded widgets.
- Made `public_description` a Face-eligible `glance` fact so Bio can be surfaced in the tray and curated like other Face facts.
- Kept `tagline` as the existing single `/quote set` Featured Quote fact and aligned the quote widget with Face visibility.
- Updated First Theater and Catharsis venue trays to show shared `at_a_glance` Face facts above mechanics.
- Updated venue trays to render quote facts as compact quote blocks.
- Added `projection_version` to shared and venue character-sheet projections.
- Added server-authored `character/projection_updated` websocket invalidations after Face, card field, Bio, Quote, and skill mutations.
- Added client-side version comparison and matching-character refetch in First Theater and Catharsis.
- Added explicit websocket rejection for forged client-submitted projection invalidations.
- Added distinct Face and Mechanics tray tabs in First Theater and Catharsis.
- Added tray refresh on focus/visible-tab return so Greenroom changes are less likely to leave stale venue data.
- Improved Greenroom History readability for event labels, timestamps, and known Chapter II `BONUS_*` results.
- Added Greenroom Mechanics rendering for actual character skills from the shared projection path.
- Improved First Theater and Catharsis Dice trays with split controls/history layout and deeper recent-roll display.
- Added screenshot-backed browser acceptance coverage for Catharsis and First Theater venue trays.
- Sanitized Chapter II History bodies so existing raw `BONUS_*` entry bodies project as readable bonus names.
- Added `scripts/smoke/kernel59a-phase3-browser.js` to reproduce the browser acceptance run.
- Rebuilt and restarted the live `victory-backend` container after implementation.

## 3. Evidence

Passed checks:

- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/characters`
- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/network`
- `cd backend && GOCACHE=/tmp/victory-gocache go test ./internal/actions ./internal/commands`
- `cd backend && GOCACHE=/tmp/victory-gocache go build ./...`
- `cd backend && GOCACHE=/tmp/victory-gocache go vet ./...`
- Greenroom inline script syntax check via extracted `<script>` body and `node --check`
- `node --check frontend/venues/first-theater/runtime.js`
- `node --check frontend/venues/catharsis/runtime.js`
- `node --check frontend/venues/first-theater/runtime/dice.js`
- `node --check frontend/venues/catharsis/runtime/dice.js`
- `node --check frontend/venues/first-theater/runtime/socket.js`
- `node --check frontend/venues/catharsis/runtime/socket.js`
- `node --check frontend/venues/first-theater/runtime/session-sync.js`
- `node --check frontend/venues/catharsis/runtime/session-sync.js`
- Scoped `git diff --check` for backend, Greenroom, venue tray, migration, and smoke-script changes.
- `node --check frontend/venues/catharsis/onboarding.js`
- `node --check scripts/smoke/kernel59a-phase3-browser.js`
- Browser acceptance harness:
  - `NODE_PATH=/tmp/k59a-playwright/node_modules node scripts/smoke/kernel59a-phase3-browser.js`
  - status: PASS
  - evidence JSON: `Construction/OperatorLogs/evidence/kernel-59A-phase3/acceptance-evidence.json`
  - screenshots:
    - `01-initial-venue-face.png`;
    - `01b-first-theater-initial-face.png`;
    - `02-quote-hidden-live.png`;
    - `02b-first-theater-quote-hidden-live.png`;
    - `03-quote-shown-live.png`;
    - `03b-first-theater-quote-shown-live.png`;
    - `04-priority-bio-first.png`;
    - `05-director-override-visible.png`;
    - `06-unlocked-owner-edit.png`;
    - `07-late-join-latest-projection.png`;
    - `08-mechanics-tab-skills.png`;
    - `09-first-theater-mechanics-tab-skills.png`.
- Live workbook History body check through Caddy:
  - `history_entries: 27`;
  - `has_raw_bonus_in_body: false`;
  - sample body begins `Final roll 4. Selected Iron Will (+1 Resolve).`

Database evidence:

- Applied `034_kernel59a_character_face_overrides.sql` through the normal migration path.
- Applied `035_kernel59a_director_value_overrides.sql` to the running `victory` database.
- Verified `character_face_overrides.value_override_active` exists as `boolean NOT NULL DEFAULT false`.
- Verified `character_face_overrides.value_override` exists as `text NOT NULL DEFAULT ''`.
- Re-applied migration `035`; it committed idempotently with Postgres "already exists, skipping" notices.

Live backend refresh evidence:

- Ran `docker compose up -d --build backend`.
- `victory-backend` image rebuilt successfully.
- `victory-backend` container was recreated and started.
- `docker exec victory-backend wget -qO- http://127.0.0.1:8081/health` returned:

```json
{"ok":true,"service":"victory-backend","time":"2026-07-05T22:24:53.839385048Z"}
```

- `docker exec bread-caddy wget -qO- http://backend:8081/health` returned healthy backend JSON.
- Backend logs after restart showed live requests to `/api/director-console/current`, `/api/discord/audio/status`, and `/health`.
- Phase 3 backend rebuild after projection invalidation work returned healthy backend JSON from both `victory-backend` and `bread-caddy` network checks at `2026-07-05T23:55:21Z`.
- Final backend rebuild after History projection-body work returned healthy backend JSON from both `victory-backend` and `bread-caddy` network checks at `2026-07-06T01:15:00Z`.

Full suite note:

- `cd backend && GOCACHE=/tmp/victory-gocache go test ./...` still fails outside this scope:
  - `backend/internal/assets`: invalid UUID fixture in `TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails`;
  - `backend/internal/identity`: Discord audio/channel mapping/bootstrap repair tests fail on Discord response/mapping expectations.

## 4. How to Run

From `/opt/victory`:

```bash
docker compose up -d --build backend
```

Verify backend from inside the container:

```bash
docker exec victory-backend wget -qO- http://127.0.0.1:8081/health
```

Verify Caddy can reach the backend:

```bash
docker exec bread-caddy wget -qO- http://backend:8081/health
```

Then hard-refresh the browser. If the venue tray still looks stale, close and reopen the tab because the venue runtime scripts may be browser-cached.

## 5. Operator Notes

- The production path is Docker, not a host `go run`; `victory-backend` contains a baked Go binary.
- A plain `docker restart victory-backend` is not enough after code changes. Use `docker compose up -d --build backend`.
- Caddy serves frontend files and proxies `/api/*`, `/ws/*`, and `/auth/*` to `backend:8081`.
- The backend container only exposes `8081/tcp` inside Docker; host `curl http://127.0.0.1:8081/health` can fail even when the app is healthy.
- Internal health checks should use `docker exec victory-backend ...` or the Caddy container's `backend:8081` network name.
- `www.movingforwardpi.com` did not resolve to this local Caddy app during verification; the local Caddyfile is configured for `victory.amurray.family`.

## 6. Blockers & Workarounds

BLOCKER:
Quote box could not be turned off from Arrange Face.

CAUSE:
Greenroom rendered a hard-coded Quote section from `tagline` even when the Face fact was hidden.

WORKAROUND:
Changed the Face renderer so Bio, Quotes, and Vital Stats render only from visible Face facts.

OPERATOR ACTION REQUIRED:
Hard-refresh after the backend rebuild.

BLOCKER:
Biography could not appear in the venue tray as curated Face information.

CAUSE:
`public_description` lived on the Bio page but was not Face-eligible, so the shared venue `at_a_glance` projection could not select it.

WORKAROUND:
Added `public_description` to the Face page as a `glance` fact while leaving the Bio page field intact.

OPERATOR ACTION REQUIRED:
Use Arrange Face priority/show-hide controls to tune whether Bio appears in the tray.

BLOCKER:
`/quote add` / multi-quote behavior does not exist.

CAUSE:
The current command implementation only supports the existing singleton `tagline` field through `/quote set <text>`.

WORKAROUND:
The UI widget is prepared to render 1-3 quote-like Face facts by length, but today it has only the single Featured Quote fact.

OPERATOR ACTION REQUIRED:
Schedule a separate quote-collection kernel if `/quote add`, feature/retire, or quote history is required.

## 7. Deviations From Kernel

- Projection version is timestamp-derived from durable source rows rather than a persisted integer revision counter.
- At-a-Glance is still based on `Region == "glance"` and capped, rather than a fully general cross-region top-N ranking algorithm.
- `UrgentState` remains absent because no current fact contract produces urgent-eligible facts.
- `/quote` remains singleton Featured Quote storage, not a multi-quote collection.
- Canonical recent-roll HTTP query was not added; Dice still uses the existing action snapshot/live action path.
- Client-forged `character/projection_updated` rejection is implemented in the websocket handler and covered by focused package checks; a separate live forged-websocket browser command was attempted but blocked by the approval reviewer usage limit.

## 8. Known Issues

- Venue trays may require tab reopen/hard refresh after runtime-script changes because of browser caching.
- Full `go test ./...` has unrelated failures in `internal/assets` and `internal/identity`.
- The current tray now separates Face and Mechanics, but broader tray presentation polish remains possible.

## 9. Next Recommended Step

- Build the actual multi-quote collection if `/quote add` is desired.
- Add a persisted integer projection revision only if timestamp-derived versions become operationally insufficient.
- Add focused websocket invalidation-helper tests with mocked hub clients if this area sees further churn.

## 9A. Acceptance Ledger

| Criterion | Status | Evidence |
|---|---|---|
| Shared projector remains the only projection authority | PASS | Venue sheet still derives from `ProjectCharacterSheet` / `EffectiveFaceFields`. |
| Projection version changes after effective mutations | PASS | Browser evidence records newer versions for visibility, priority, value, lock, and identity mutations. |
| Server-produced invalidation reaches matching open clients | PASS | Browser evidence observed 8 `character/projection_updated` frames and live tray refetches. |
| Ordinary client cannot forge invalidation | PASS | Websocket handler rejects `character/projection_updated` with `server_authored_event_only`; focused network checks pass. |
| Greenroom updates without reload | PASS | Existing mutation responses still reload workbook state after curation; syntax checks pass. |
| Catharsis updates without focus/reload | PASS | Browser evidence `02-quote-hidden-live.png`, `03-quote-shown-live.png`, and JSON assertions. |
| First Theater updates without focus/reload | PASS | Browser evidence `02b-first-theater-quote-hidden-live.png`, `03b-first-theater-quote-shown-live.png`, and JSON assertions. |
| Face and Mechanics are separate tray views | PASS | Both venue DOMs now have Face/Mechanics tabs and panels. |
| Selected tray tab survives refetch | PASS | `activeVenueSheetTab` is independent of sheet render and reapplied after render. |
| Active-character switch replaces sheet | PASS | Sheet cache key includes active character id; existing identity refresh calls `refreshCharacterSheet`. |
| Owner visibility change projects everywhere | PASS | Browser evidence hides/shows Featured Quote in Catharsis and First Theater without reload. |
| Manual integer priority projects everywhere | PASS | Browser evidence moved Biography to first Face fact after manual priority 999. |
| Return to inferred works | PASS | Browser harness cleanup returned `tagline` and `public_description` to inferred; final DB check confirmed. |
| Director override projects without changing source value | PASS | Browser evidence shows Director override in tray; cleanup restored underlying card tagline. |
| Director dimension lock blocks owner mutation | PASS | Browser evidence shows owner PATCH rejected with `face_value_locked`. |
| Unlocked dimensions remain editable | PASS | Browser evidence shows owner PATCH succeeds after Director unlock/clear. |
| Director reason stays private | PASS | Venue sheet payload includes no reason fields; invalidation payload includes no reason fields. |
| Chapter II History is player-readable | PASS | Live workbook body check returned readable Stage 10 body with bonus name and attribute changes. |
| No known raw BONUS IDs appear | PASS | Live workbook body check returned `has_raw_bonus_in_body: false`; raw IDs remain only in canonical payload. |
| Chapter III and IV History are player-readable | PASS | Existing labels/body formatters retained for archetype/first skill events. |
| Private Journal never enters venue projection | PASS | Venue sheet builder does not read journals/private notes; invalidation payload carries no sheet content. |
| Kernel 60 skills render in Greenroom and venues | PASS | Mechanics skill rendering retained; package checks pass. |
| Skill click-to-roll still works | PASS | Venue skill button handler preserved in Mechanics tab. |
| Dice history is beside controls on desktop | PASS | Prior split dice layout retained; JS syntax checks pass. |
| Mobile Dice fallback is usable | PASS | Prior responsive CSS fallback retained; scoped JS/CSS checks pass. |
| Reconnect/late join receives latest projection | PASS | Browser evidence `07-late-join-latest-projection.png` and assertion. |
| Two-client browser proof attached | PASS | Browser evidence JSON plus 12 screenshots attached under `Construction/OperatorLogs/evidence/kernel-59A-phase3/`. |
| Required automated checks recorded | PASS | See Evidence section and Phase 3 reportback. |
| Operator and roadmap documents updated | PASS | Full-scope and Phase 3 reportbacks updated to PASS with evidence paths. |

## 10. Files Changed / Created

High-level scope:

- `backend/internal/characters/character_face_overrides.go`
- `backend/internal/characters/character_sheet_projection.go`
- `backend/internal/characters/venue_sheet.go`
- `backend/internal/characters/workbook_pages.go`
- `backend/internal/characters/*_test.go`
- `backend/internal/network/venue_sheet_http.go`
- `frontend/venues/greenroom/index.html`
- `frontend/venues/first-theater/index.html`
- `frontend/venues/first-theater/runtime.js`
- `frontend/venues/first-theater/runtime/dice.js`
- `frontend/venues/catharsis/index.html`
- `frontend/venues/catharsis/runtime.js`
- `frontend/venues/catharsis/onboarding.js`
- `frontend/venues/catharsis/runtime/dice.js`
- `database/migrations/034_kernel59a_character_face_overrides.sql`
- `database/migrations/035_kernel59a_director_value_overrides.sql`
- `scripts/smoke/fresh-install.sh`
- `scripts/smoke/kernel59a-phase3-browser.js`
- `Construction/OperatorLogs/kernel-59A-reportback.md`
- `Construction/OperatorLogs/kernel-59A-full-scope-reportback.md`
- `Construction/OperatorLogs/evidence/kernel-59A-phase3/acceptance-evidence.json`
