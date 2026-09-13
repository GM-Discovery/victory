# Kernel 46 Reportback

## Status
Complete for the Pixi map layer + Workshop map upload slice.

Kernel 46 added a real First Theater map workflow: a producer or director can right-click the stage, upload or select a map asset through a Workshop-style callout, save crop/fit/scale state, and have Pixi render that map on its own dedicated layer without replacing the older First Theater stage façade.

## What Was Built
- Added a dedicated First Theater map API:
  - `GET /api/venues/{slug}/map`
  - `POST /api/venues/{slug}/map`
- Extended the workshop upload pipeline so uploaded assets can be tagged as `map` and listed with `GET /api/workshop/assets?asset_type=map`.
- Added an authenticated asset content route at `GET /api/assets/{id}/content` so the browser can load the saved map image directly from the server.
- Added durable schema for:
  - `assets.asset_type`
  - `assets.tags`
  - `venue_active_maps`
- Kept the old First Theater stage façade intact while adding a separate Pixi map layer above the façade and below the live object layer.
- Added a right-click `Add / Replace Map` stage action in First Theater.
- Added a Workshop-style map editor panel with:
  - file upload
  - existing asset selection
  - fit mode
  - crop X/Y controls
  - scale control
  - safe-margin control
- Updated the canon trail so Kernel 46 is now the map-layer kernel instead of the earlier backdrop experiment.

## Evidence
- `node --check frontend/lib/victory-pixi-stage.js`
- `awk 'NR>=2026 { if ($0 ~ /^  <\\/script>/) exit; print }' frontend/venues/first-theater/index.html > /tmp/first-theater-inline.js && node --check /tmp/first-theater-inline.js`
- `GOCACHE=/tmp/victory-gocache go test ./internal/assets ./internal/venues ./cmd/victory`
- `git diff --check`

No backend restart was required for verification in this workspace, but the running backend binary would need to be restarted in a live deployment so the new routes and schema-aware code are active.

## How To Run
- Open `/venues/first-theater/`.
- Hard refresh once so the new HTML, Pixi helper, and inline script are definitely loaded.
- Right-click the stage and choose `Add / Replace Map`.
- Upload a PNG, JPEG, or WebP map image, or choose an already uploaded map asset.
- Save the placement to publish the active First Theater map.

## Operator Notes
- The older First Theater stage façade remains the live presentation around the map layer.
- The map is persisted server-side, not in local Pixi state.
- The new map workflow is First Theater only.
- The Cave was left untouched, per the kernel scope.
- The workshop asset list is intentionally narrow and only shows map assets for the map editor path.

## Blockers & Workarounds
- The First Theater HTML contains multiple `<script>` blocks, so validation had to target the inline block directly.
- The asset content route had to be opened up for venue viewers, not just asset owners, so the active map image can actually render for people looking at the venue.
- Public access to the map image still depends on venue visibility, which is the safest place to keep the browser fetch rules.

## Deviations From Kernel
- I did not force Pixi into The Cave.
- I did not turn the map into the venue background.
- I did not replace the First Theater façade with the uploaded art.
- I used a dedicated venue-map table instead of pushing the map into the existing Cave action stream, because the directive wanted a single active First Theater map with its own crop/fit state.

## Known Issues
- The map editor is intentionally narrow in scope and does not yet provide full pan/crop editing.
- The upload/list path is limited to map assets for this kernel.
- The First Theater map layer still relies on same-origin browser loading for the asset image route.

## Next Recommended Step
- Do a live browser pass on First Theater to confirm the map layer feels right in motion and the safe margin keeps the menus usable.
- If the map flow holds up, the next venue-specific improvement should be a refinement pass on the editor UI rather than broadening the feature into other venues.

## Files Changed / Created
- `backend/cmd/victory/main.go`
- `backend/internal/assets/read.go`
- `backend/internal/assets/upload.go`
- `backend/internal/venues/map.go`
- `database/migrations/026_kernel46_first_theater_map.sql`
- `frontend/lib/victory-pixi-stage.js`
- `frontend/venues/first-theater/index.html`
- `scripts/smoke/fresh-install.sh`
- `Construction/Canon/current-state.md`
- `Construction/Canon/roadmap.md`
- `Construction/OperatorLogs/operator-log.md`
- `Construction/OperatorLogs/kernel-46-reportback.md`
