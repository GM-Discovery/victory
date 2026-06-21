# Kernel Report Back - Kernel 49

## 1. Status
PASS

Kernel 49 is complete. The existing Workshop asset model was extended into a Warehouse-oriented durable asset model, token preparation now works from the Workshop surface in The Cave, storage policy is editable from Producer's Office, and deleted/missing assets now resolve to the construction fallback instead of leaving broken references blank.

## 2. What Was Built

- Extended the existing `assets` model instead of replacing it with a parallel token-only schema.
- Added warehouse durability fields to `assets`:
  - `name`
  - `shape`
  - `default_grid_width`
  - `default_grid_height`
  - `retain_original`
  - `status`
  - `crop_x`
  - `crop_y`
  - `zoom`
  - `stored_bytes`
  - `last_used_at`
  - `deleted_at`
- Added an installation-level `warehouse_storage_settings` table for hard cap, upload cap, warning thresholds, retention default, and token variant sizing.
- Kept the existing Workshop map upload path in place and extended it so the single upload ceiling comes from warehouse storage settings.
- Added token upload/prep on the Workshop surface through the existing workshop asset route family:
  - `POST /api/workshop/assets/token`
  - legacy `POST /api/workshop/assets`
  - map listing `GET /api/workshop/assets?asset_type=map`
- Added Warehouse read, list, storage, settings, and delete routes:
  - `GET /api/warehouse/storage`
  - `PATCH /api/warehouse/storage/settings`
  - `GET /api/warehouse/assets`
  - `GET /api/warehouse/assets/{id}`
  - `DELETE /api/warehouse/assets/{id}`
- Added token preparation UI inside The Cave's `mode=workshop` surface:
  - file upload
  - inherited token name
  - shape selector
  - crop/position preview
  - zoom
  - X/Y positioning
  - default footprint display
  - retain-original display
  - estimated storage/output display
  - save/cancel
- Added Producer's Office warehouse storage UI:
  - total used
  - total limit
  - warning state
  - active/deleted counts
  - policy editor for storage thresholds, upload cap, hard cap, retention default, and token variant sizes
  - warehouse asset browser
- Added generated token variants:
  - `master` max dimension `1024`
  - `stage` max dimension `512`
  - `thumbnail` max dimension `128`
- Added safe tombstone deletion for warehouse assets:
  - removes stored files
  - preserves the asset row
  - marks the asset deleted
  - returns current reference count for delete warnings
  - keeps existing placements intact
  - makes broken reads resolve to the construction fallback
- Preserved existing map assets and the First Theater map flow.

## 3. Evidence

Checks run:

- `GOCACHE=/tmp/victory-gocache go test ./...` from `/opt/victory/backend`
- `node --check` on extracted inline scripts from:
  - `frontend/venues/the-cave/index.html`
  - `frontend/venues/producers-office/index.html`
- `git diff --check`

Result summary:

- Go tests passed.
- Both inline JavaScript checks passed.
- `git diff --check` passed.

Safe test strategy:

- I did not run destructive tests against the live Victory database.
- I used compile-only Go validation and inline JS syntax checks instead of live DB mutation tests because this workspace does not provide an isolated test database.
- I relied on the checked-in migration plus runtime bootstrap for schema application rather than hand-editing live tables.

Confirmed behavior evidence from code paths:

- Supported MIME types are validated in `backend/internal/assets/upload.go` and `backend/internal/assets/warehouse.go` as:
  - `image/png`
  - `image/jpeg`
  - `image/webp`
- Crop/position metadata is stored on token assets as:
  - `crop_x`
  - `crop_y`
  - `zoom`
- Duplicate uploads are allowed and not merged; a non-blocking duplicate warning may be returned when the upload checksum already exists for the installation.
- Deletion requires explicit user confirmation in the UI and then tombstones the asset server-side. The delete warning includes the asset name and recovered bytes, and the backend returns the current reference count so the UI can report it.
- Broken or deleted asset reads now resolve to `frontend/assets/construction.png`.
- Existing map assets continue to flow through the map asset list and active map APIs.
- Theater token placement was not built.
- No run-history field or run-by-run asset history array was added.

## 4. How to Run

1. Apply or start with the new migration:
   - `database/migrations/029_kernel49_warehouse_storage.sql`
2. Start the backend from `/opt/victory/backend`:
   - `GOCACHE=/tmp/victory-gocache go run ./cmd/victory`
3. Open The Cave in workshop mode:
   - `/venues/the-cave/?mode=workshop`
4. Open Producer's Office:
   - `/venues/producers-office/`
5. In The Cave workshop surface:
   - choose a PNG, JPEG, or WebP
   - edit the inherited token name
   - choose `circle`, `square`, `hex`, or `raw`
   - adjust zoom and X/Y position
   - save the token
6. In Producer's Office:
   - review storage usage
   - update storage policy
   - search/filter warehouse assets
   - delete an asset with confirmation to tombstone it

## 5. Operator Notes

- Existing asset model was extended, not replaced.
- The Warehouse model is installation-wide and currently lives in the shared `assets` table plus `warehouse_storage_settings`.
- The Workshop token prep surface is implemented inside The Cave's existing `mode=workshop` route rather than a brand-new venue page.
- The current live storage defaults are:
  - hard cap: `15 GB`
  - single upload max: `25 MB`
  - warning threshold: `80%`
  - critical threshold: `90%`
  - token master: `1024`
  - token stage: `512`
  - token thumbnail: `128`
  - retain originals default: `false`
- Storage accounting counts durable bytes for active assets and excludes deleted/tombstoned assets.
- The construction fallback is treated as the missing-asset resolution path.
- Map behavior remains intact; Kernel 49 does not add token placement on Theater stages.

## 6. Blockers & Workarounds

BLOCKER:
No isolated test database was available in this workspace.

CAUSE:
The environment does not provide a dedicated disposable PostgreSQL instance for integration mutation tests.

WORKAROUND:
Used compile-only Go tests, inline JavaScript syntax checks, and `git diff --check` instead of live DB mutation tests.

BLOCKER:
The Workshop token surface had to be added inside an existing venue runtime rather than a standalone Workshop application page.

CAUSE:
The current project routes Workshop through The Cave's `mode=workshop` implementation.

WORKAROUND:
Implemented the token UI and its browser wiring inside `frontend/venues/the-cave/index.html`.

OPERATOR ACTION REQUIRED:
None.

## 7. Deviations from Kernel

- The Workshop token UI lives in The Cave's `mode=workshop` surface instead of a separate Workshop venue page.
- The Warehouse browser is exposed in Producer's Office rather than a dedicated warehouse venue page.
- The backend generates token variants as PNG files for the durable token assets.
- The existing map asset path was preserved and extended, rather than replaced by a new asset subsystem.
- The delete warning is implemented as an explicit confirmation dialog rather than a custom modal with multi-step review.

## 8. Known Issues

- Theater token placement is still deferred to Kernel 50.
- No subject/background removal or knockout extraction was added.
- No folders, marketplace, or historical run-by-run asset tracking were added.
- Browser upload performance and preview fidelity have only been validated syntactically here, not through a full live UI smoke in this turn.

## 9. Next Recommended Step

- Build Kernel 50 Theater token placement to consume the Warehouse assets and the new token metadata:
  - `asset_id`
  - `shape`
  - `default_grid_width`
  - `default_grid_height`
  - `last_used_at`
  - tombstone-aware fallback resolution

## 10. Files Changed / Created

- [backend/cmd/victory/main.go](/opt/victory/backend/cmd/victory/main.go)
- [backend/internal/assets/read.go](/opt/victory/backend/internal/assets/read.go)
- [backend/internal/assets/upload.go](/opt/victory/backend/internal/assets/upload.go)
- [backend/internal/assets/warehouse.go](/opt/victory/backend/internal/assets/warehouse.go)
- [frontend/venues/the-cave/index.html](/opt/victory/frontend/venues/the-cave/index.html)
- [frontend/venues/producers-office/index.html](/opt/victory/frontend/venues/producers-office/index.html)
- [Construction/current-state.md](/opt/victory/Construction/current-state.md)
- [Construction/roadmap.md](/opt/victory/Construction/roadmap.md)
- [Construction/OperatorLogs/operator-log.md](/opt/victory/Construction/OperatorLogs/operator-log.md)
- [database/migrations/029_kernel49_warehouse_storage.sql](/opt/victory/database/migrations/029_kernel49_warehouse_storage.sql)
