# Kernel 75 Follow-up Reportback — Golden Journey, Presentation, and Profile UI

## 1. Status

**PARTIAL / operational.** The follow-up work is implemented and the rebuilt
backend is healthy. The original Kernel 75 completion report remains the
authoritative report for tutorial completion, Story So Far, and Aftercare; this
document records the additional journey fixes and presentation work completed
after that report.

## 2. Concrete deliverables

### Golden Journey and tutorial flow

- Character selection now automatically creates/updates an open-enrollment
  Player roster row for active Catharsis Show Runs.
- Courtyard door hotspots use playable-stage coordinates across aspect ratios.
- Kessa and Ra receive large participant-local speaker portraits; Kessa is
  mirrored on the left and Ra appears on the right.
- Ra opens with the focused question “Why were you looking for me?” and no
  longer presents the Crown Bet conversation.
- Ra’s closing gate action is compacted to one reveal beat before handoff.
- The tutorial handoff completion card is backstage-only; Players use the
  completion panel instead of a floating index card.
- Handoff grid settings now honor `grid_enabled: false` without changing the
  shared courtyard grid.

### Map presentation

- Fog base opacity, venue puffs, ambient puffs, and woods puffs were reduced so
  nearby venues do not sit in visibly different fog shades.
- Info Booth emphasis remains separate from the general venue treatment.
- Map/runtime cache-busters were advanced so the browser receives the updated
  fog and handoff renderer.

### Profile and Trailers UI

- Workbook and all Trailers pages now use the crimson/black/gold palette.
- Workbook spacing, typography, fields, cards, navigation, and action controls
  were modernized to reduce the old form-heavy appearance.
- TTRPG Identity now has three favorite-system selectors, with Socio first and
  D&D second.
- The supplied categorized TTRPG list is included in the profile catalogue.
- TTRPG categories are collapsed until opened; each game has an experience
  status selector covering the requested eight statuses.
- Backend validation persists the matrix as a sanitized per-system status map.
- Explicit `/assets/favicon.png` and Apple touch-icon links were added to all
  HTML pages so venue tabs use the Victory icon instead of generic browser
  icons.

## 3. Evidence

- `node --check frontend/app.js` — passed after fog changes.
- `node --check frontend/lib/stage-runtime/runtime.js` — passed after handoff
  and grid changes.
- `node --check frontend/lib/stage-runtime/logic.js` — passed after hidden-card
  visibility changes.
- `jq empty backend/internal/playerprofile/catalogues/player-profile-v1.0.0.json`
  — passed after the expanded TTRPG catalogue was generated.
- `git diff --check` — passed.
- Backend rebuild completed with `docker compose up -d --build backend`.
- Live health check returned:

  `{ "ok": true, "service": "victory-backend" }`

- Migration ledger verified `080_kernel75_hide_handoff_card_for_players.sql`.

## 4. Operator steps

1. Open the desired venue or `/venues/trailers/workbook.html`.
2. Use a normal browser refresh after frontend changes.
3. If a favicon remains stale, close/reopen the tab or perform one hard refresh;
   browser favicon caches are unusually persistent.

## 5. Restart and cache notes

- Backend was rebuilt and restarted for the catalogue and migration changes.
- No further container restart is needed for the current frontend-only edits.
- Frontend cache-busters were advanced on the map and stage runtime scripts.

## 6. Known issues / follow-up

- Some older venue files contain legacy inline presentation rules and should be
  visually consolidated in a later design-system pass.
- The TTRPG matrix is intentionally collapsed by default because the supplied
  list is large.
- The original Kernel 75 report’s art and dice-test limitations remain open.

## 7. Files touched at a high level

- `backend/migrations/077_kernel75_ra_opening_copy.sql`
- `backend/migrations/078_kernel75_ra_supervisor_handoff.sql`
- `backend/migrations/079_kernel75_ra_gate_reveal_compact.sql`
- `backend/migrations/080_kernel75_hide_handoff_card_for_players.sql`
- `backend/internal/characters/chapter4_select.go`
- `backend/internal/playerprofile/catalogue.go`
- `backend/internal/playerprofile/validation.go`
- `backend/internal/playerprofile/catalogues/player-profile-v1.0.0.json`
- `frontend/app.js`, `frontend/styles.css`, `frontend/index.html`
- `frontend/lib/stage-runtime/{geometry,logic,runtime,program-panel,participant-interactions}.js`
- `frontend/venues/catharsis/index.html`
- `frontend/venues/first-theater/index.html`
- `frontend/venues/trailers/*.html`
- `frontend/styles/trailers-theme.css`
