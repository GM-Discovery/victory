# Kernel 45 Reportback

## Status
Complete for the venue shell rebase + shared helper extraction slice.

Kernel 45 tightened the public Victory map shell, extracted reusable venue-shell helpers, and wired those helpers onto the main venue path without disturbing the working Discord/audio behavior from Kernel 44.

## What Was Built
- Collapsed the Victory map header into a much thinner hover-open strip so the top chrome stays out of the way until it is needed.
- Expanded the map stage so the map background can use more of the viewport instead of stopping at a 1280px ceiling.
- Added reusable shell helpers in `frontend/venues/shared/venue-shell.js` for:
  - venue slug normalization
  - venue name formatting
  - shell slot resolution
  - header chip registration
  - safe refresh/reload hooks
  - reusable shell mounting metadata
  - presence preview rendering
  - local preference storage helpers
- Wired the shared helper into:
  - `frontend/venues/first-theater/index.html`
  - `frontend/venues/middle-school-stage/index.html`
- Promoted Kernel 45 into the current canon and roadmap.
- Added a Kernel 45 operator log entry.

## Evidence
- `node --check frontend/venues/shared/venue-shell.js`
- `awk 'NR>=1842 { if ($0 ~ /<\/script>/) exit; print }' frontend/venues/first-theater/index.html > /tmp/first-theater-inline.js && node --check /tmp/first-theater-inline.js`
- `awk 'NR>=1289 { if ($0 ~ /<\/script>/) exit; print }' frontend/venues/middle-school-stage/index.html > /tmp/middle-school-stage-inline.js && node --check /tmp/middle-school-stage-inline.js`
- `git diff --check`

No backend restart was required for this pass because the changes here are frontend and canon/docs only.

## How To Run
- Open the main map page and hard refresh once so the new CSS and shared helper are definitely loaded.
- Visit:
  - `/`
  - `/venues/first-theater/`
  - `/venues/middle-school-stage/`
- Hover or focus the top edge of the map page to expand the new header chrome.

## Operator Notes
- The shared shell helper is intentionally light. It gives future venues a reusable contract without trying to replace venue-specific runtime logic.
- First Theater remains the practical source template.
- The Cave remains the proving ground.
- Middle School Stage remains the clean layout-learning shell.
- Kernel 44 audio behavior was left intact.

## Blockers & Workarounds
- The inline HTML validation had to be done block-by-block because those venue pages contain multiple `<script>` blocks.
- Browser caches may delay the visual header/map changes until a hard refresh is performed.

## Deviations From Kernel
- I used the existing shared helper location at `frontend/venues/shared/venue-shell.js` instead of introducing a second helper file under `frontend/lib/`.
- The helper API is small and practical rather than a full framework layer, which keeps the live venue code stable.

## Known Issues
- Venue pin positions may still need per-venue tuning after the broader map fill changes.
- The new header behavior is intentionally hover-open, so touch-only devices may need a later accessibility pass if we want a different mobile treatment.
- Cached assets can still make the map appear unchanged until the browser reloads cleanly.

## Next Recommended Step
- Do a live browser pass on the main map and the two helper-backed venues to confirm the new shell behavior feels right in motion.
- If the shell contract holds up, move the next venue into the shared pattern instead of cloning more inline shell code.

## Files Changed / Created
- `frontend/styles.css`
- `frontend/venues/shared/venue-shell.js`
- `frontend/venues/first-theater/index.html`
- `frontend/venues/middle-school-stage/index.html`
- `Construction/Canon/current-state.md`
- `Construction/Canon/roadmap.md`
- `Construction/OperatorLogs/operator-log.md`
- `Construction/OperatorLogs/kernel-45-reportback.md`
