# Measured Tabletop Contract — Kernel 87

Canon for physical scale, unit configuration, and distance measurement over the shared map. Backend: `backend/internal/drawing/measurement.go`, `stage_drawing_settings` (migration `098_kernel87_cartograph_drawing.sql`). Frontend: `frontend/lib/stage-runtime/drawing.js`'s `pathDistance`.

## 1. Chain

```text
grid/map (existing venue_grid_configs: grid_type, cell_size, hex_orientation)
→ physical scale (stage_drawing_settings: scale_grid_units, scale_real_units, scale_unit_label)
→ unit (scale_unit_label, e.g. "ft", "mi", free text — never hardcoded to feet)
→ measurement (straight / path, computed against the configured diagonal policy or hex geometry)
```

## 2. Physical scale

Stored per Show in `stage_drawing_settings`:

- `scale_grid_units` (DOUBLE, default 1) — how many grid cells/hexes...
- `scale_real_units` (DOUBLE, default 5) — ...equal this many real-world units...
- `scale_unit_label` (TEXT, default `"ft"`) — ...in this unit.

Example: `1 square = 5 ft` is `{scale_grid_units: 1, scale_real_units: 5, scale_unit_label: "ft"}`. `1 hex = 2 miles` is `{scale_grid_units: 1, scale_real_units: 2, scale_unit_label: "mi"}`. Neither the schema nor `drawing.CellsToRealUnits` special-cases feet or any other unit — the label is opaque free text (validated non-empty, ≤24 chars), and the ratio is just `cells / scale_grid_units * scale_real_units`.

Only Director+ may change settings (`drawing.SaveSettings`, session-scoped `rollaudience.IsDirectorPlus` check).

## 3. Diagonal policy (kernel §7 "do not silently assume one tabletop convention")

`stage_drawing_settings.diagonal_policy`, one of three, CHECK-constrained, **default `alternating_1_2`**:

- **`alternating_1_2`** (default) — classic 5e "5/10/5": diagonal cost is `diagonal + floor(diagonal/2)`, i.e. every other diagonal cell costs double. A straight 3-cell diagonal move costs 4 cells, not 3.
- **`every_diagonal_1`** — Chebyshev distance (`max(cellsX, cellsY)`) — every diagonal costs the same as orthogonal (D&D 4e-style / many simplified systems).
- **`euclidean`** — true geometric distance (`hypot(cellsX, cellsY)`), ignoring grid step-counting entirely.

Implemented in `drawing.SquareGridDistance`, unit-tested for all three policies (`TestSquareGridDistanceDiagonalPolicies`, `TestSquareGridDistanceOrthogonalUnaffectedByPolicy` — orthogonal moves are policy-invariant by construction, only diagonal moves differ).

**This is documented, not silent**, per kernel §7's explicit requirement: the default is stated here, is a real configured column (not a code constant nobody can see), and is settable per-Show through the same `drawing-settings` endpoint as physical scale.

## 4. Hex

`drawing.HexGridDistance` respects the existing venue's `hex_orientation` (`flat-top` or `pointy-top` from `venue_grid_configs`) using the standard axial/cube-coordinate conversion, with `size = cellSize / 2` matching `frontend/lib/stage-runtime/geometry.js`'s existing hex circumradius convention exactly (so a distance in "cells" always corresponds to the same visual hex spacing the grid itself is drawn with — no second, drifting definition of hex size). Unit-tested (`TestHexGridDistanceAdjacentCell` — one hex step measures to 1.0 cell for both orientations' respective axis).

## 5. Gridless calibration

`stage_drawing_settings.gridless_calibration` (nullable JSONB) is reserved for the simplest workable gridless model: two reference world-points plus the real-world distance between them, from which a scalar world-distance-per-real-unit ratio can be derived. `drawing.PathDistanceCells` already branches on `GridGeometry.GridType == "none"` to fall back to raw Euclidean world-distance (tested, `TestPathDistanceCellsGridless`), which the calibration ratio then converts via the same `CellsToRealUnits` function used for gridded maps. The calibration-capture UI (clicking two points and entering a known distance) is **not** wired into `drawing.js` in this kernel — the server-side model and math exist and are tested, but there is no "Calibrate gridless map" button yet. Recorded as a known gap in the reportback, not a silent omission.

## 6. Measure tool (kernel §7)

Client-side (`drawing.js`'s Measure tool): click to place points, path distance sums consecutive segments through the same diagonal-policy math as the backend (`pathDistance`, a client mirror of `drawing.PathDistanceCells`/`SquareGridDistance` — kept in sync by hand, not by sharing code across the Go/JS boundary, since this repo has no shared-codegen tooling). Straight measurement is simply a 2-point path. Distance display updates live as points are added.

**Ephemeral by default** (kernel §7): measurement points are local UI state (`state.measurePoints`) in `drawing.js`, never persisted as a drawing object and never sent to the backend. **Pin Measurement was not implemented** this kernel — deferred as optional/easy-if-cheap polish per the kernel's own permission ("Optional Pin Measurement is acceptable if easy, but pinned measurement is reference/presentation state, not drawing truth"). No PNG export or canonical drawing state is ever affected by an in-progress measurement.

## 7. Explicit exclusions (kernel §7, confirmed absent)

No movement cost, no pathfinding, no movement enforcement, no initiative, no attack-range automation. The measure tool computes and displays a number; it does not gate, cost, or validate any other action.
