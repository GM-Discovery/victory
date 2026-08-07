const test = require("node:test");
const assert = require("node:assert/strict");

const grid = require("../../frontend/venues/storyboards/grid-model.js");

test("roleAffordances matches backend's tierAtLeastCrew/Director/owner boundaries", () => {
  const audience = grid.roleAffordances("audience");
  assert.equal(audience.canEditCards, false);
  assert.equal(audience.canDragCards, false);
  assert.equal(audience.canEditStructure, false);
  assert.equal(audience.canManageSharing, false);

  const crew = grid.roleAffordances("crew");
  assert.equal(crew.canEditCards, true);
  assert.equal(crew.canDragCards, true);
  assert.equal(crew.canAttachImage, true);
  assert.equal(crew.canEditStructure, false);
  assert.equal(crew.canInsertColumns, false);
  assert.equal(crew.canManageSharing, false);
  assert.equal(crew.canExport, false);

  const director = grid.roleAffordances("director");
  assert.equal(director.canEditStructure, true);
  assert.equal(director.canInsertColumns, true);
  assert.equal(director.canExport, true);
  assert.equal(director.canManageSharing, false);

  const producer = grid.roleAffordances("producer");
  assert.equal(producer.canEditStructure, true);
  assert.equal(producer.canManageSharing, false);

  const owner = grid.roleAffordances("owner");
  assert.equal(owner.canManageSharing, true);
  assert.equal(owner.canArchive, true);

  const operator = grid.roleAffordances("operator");
  assert.equal(operator.canManageSharing, true);
  assert.equal(operator.canEditStructure, true);
});

test("visibleCardForCell picks the lowest sort_order_in_cell, matching backend display order", () => {
  const cards = [
    { id: "c2", row_id: "r1", column_id: "col1", sort_order_in_cell: 2 },
    { id: "c1", row_id: "r1", column_id: "col1", sort_order_in_cell: 0 },
    { id: "c3", row_id: "r1", column_id: "col1", sort_order_in_cell: 1 },
  ];
  const byCell = grid.cardsByCell(cards);
  const visible = grid.visibleCardForCell(byCell["r1|col1"]);
  assert.equal(visible.id, "c1");
});

test("visibleCardForCell returns null for an empty cell", () => {
  assert.equal(grid.visibleCardForCell(undefined), null);
  assert.equal(grid.visibleCardForCell([]), null);
});

test("computeGridLayout places header at row 1, then band-bar + member rows in band sort_order", () => {
  const snapshot = {
    columns: [{ id: "colA" }, { id: "colB" }],
    bands: [
      { id: "bandB", sort_order: 1, is_collapsed: false },
      { id: "bandA", sort_order: 0, is_collapsed: false },
    ],
    rows: [
      { id: "r1", band_id: "bandA", sort_order_in_band: 0 },
      { id: "r2", band_id: "bandA", sort_order_in_band: 1 },
      { id: "r3", band_id: "bandB", sort_order_in_band: 0 },
    ],
  };
  const layout = grid.computeGridLayout(snapshot);

  assert.equal(layout.columnLine.colA, 2);
  assert.equal(layout.columnLine.colB, 3);
  assert.equal(layout.lastColumnLine, 3);

  // bandA sorts first (sort_order 0): band-bar at row 2, its two rows at
  // 3/4, then a reserved add-row spacer at 5.
  assert.equal(layout.bandHeaderLine.bandA, 2);
  assert.equal(layout.rowLine.r1, 3);
  assert.equal(layout.rowLine.r2, 4);
  assert.equal(layout.addRowLine.bandA, 5);
  // bandB sorts second: band-bar at row 6, its one row at 7, spacer at 8.
  assert.equal(layout.bandHeaderLine.bandB, 6);
  assert.equal(layout.rowLine.r3, 7);
  assert.equal(layout.addRowLine.bandB, 8);
  assert.equal(layout.lastRowLine, 8);
  assert.deepEqual(layout.bandOrder, ["bandA", "bandB"]);
});

test("computeGridLayout skips row/card layout for a collapsed band without dropping its rows from bandRowIds", () => {
  const snapshot = {
    columns: [{ id: "colA" }],
    bands: [{ id: "bandA", sort_order: 0, is_collapsed: true }],
    rows: [{ id: "r1", band_id: "bandA", sort_order_in_band: 0 }],
  };
  const layout = grid.computeGridLayout(snapshot);
  assert.equal(layout.rowLine.r1, undefined);
  assert.deepEqual(layout.bandRowIds.bandA, []);
  // A collapsed band reserves no add-row spacer either.
  assert.equal(layout.addRowLine.bandA, undefined);
  // Only the header row (1) and the one band-bar row (2) exist.
  assert.equal(layout.lastRowLine, 2);
});

// Regression: board.html used to compute the "+ row" spacer line itself
// (last member row's line + 1, or the band-bar line + 1 if empty) instead
// of reading it from computeGridLayout. That independent computation
// silently collided with the next band's bandHeaderLine whenever a band
// had exactly one row, which real Playwright drag/click testing caught as
// "next band's label intercepts pointer events" on the add-row button.
// addRowLine must never equal any other band's bandHeaderLine/rowLine.
test("addRowLine never collides with another band's header or row lines", () => {
  const snapshot = {
    columns: [{ id: "colA" }],
    bands: [
      { id: "bandA", sort_order: 0, is_collapsed: false },
      { id: "bandB", sort_order: 1, is_collapsed: false },
    ],
    rows: [{ id: "r1", band_id: "bandA", sort_order_in_band: 0 }],
  };
  const layout = grid.computeGridLayout(snapshot);
  const usedLines = new Set([1, ...Object.values(layout.bandHeaderLine), ...Object.values(layout.rowLine)]);
  for (const line of Object.values(layout.addRowLine)) {
    assert.ok(!usedLines.has(line), "addRowLine " + line + " collides with a header/row line");
  }
});

test("colorTokenClass only ever returns a whitelisted class name, never the raw token", () => {
  assert.equal(grid.colorTokenClass("amber"), "sb-color-amber");
  assert.equal(grid.colorTokenClass("AMBER"), "sb-color-amber");
  assert.equal(grid.colorTokenClass(""), "sb-color-default");
  assert.equal(grid.colorTokenClass(undefined), "sb-color-default");
  assert.equal(
    grid.colorTokenClass('"><script>alert(1)</script>'),
    "sb-color-default"
  );
  for (const name of grid.colorTokenNames()) {
    assert.ok(grid.COLOR_TOKENS[name].startsWith("sb-color-"));
  }
});

test("isImageGravestone", () => {
  assert.equal(grid.isImageGravestone(false, null), false, "no reference at all is never a gravestone");
  assert.equal(grid.isImageGravestone(true, null), true, "404 (asset hard-deleted) is a gravestone");
  assert.equal(grid.isImageGravestone(true, { missing_asset: true }), true, "tombstoned asset is a gravestone");
  assert.equal(grid.isImageGravestone(true, { missing_asset: false }), false, "a live asset is not a gravestone");
});

test("dropResolutionKind", () => {
  assert.equal(grid.dropResolutionKind(null, "dragged"), "move");
  assert.equal(grid.dropResolutionKind({ id: "dragged" }, "dragged"), "noop");
  assert.equal(grid.dropResolutionKind({ id: "other" }, "dragged"), "occupied");
});
