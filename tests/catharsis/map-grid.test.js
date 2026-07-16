const test = require("node:test");
const assert = require("node:assert/strict");

const mapGrid = require("../../frontend/venues/catharsis/runtime/map-grid.js");

test("panel clamping keeps editor windows inside the shell", () => {
  assert.deepEqual(mapGrid.clampPanelPosition(10, 20, { width: 300, height: 200 }, { width: 100, height: 80 }, 100, 80), {
    left: 10,
    top: 20,
  });
  assert.deepEqual(mapGrid.clampPanelPosition(999, 999, { width: 300, height: 200 }, { width: 100, height: 80 }, 100, 80), {
    left: 192,
    top: 112,
  });
});

test("camera controls and playable bounds derive from the current stage", () => {
  assert.deepEqual(mapGrid.cameraControlsState({ zoomRelativeToFit: 0.2 }, "theater", { minZoom: 0.25, maxZoom: 4 }), {
    label: "25%",
    zoomOutDisabled: true,
    zoomInDisabled: false,
    interactionLocked: false,
  });

  assert.deepEqual(mapGrid.getPlayableBounds({ width: 1000, height: 700 }), {
    x: 40,
    y: 112,
    width: 920,
    height: 588,
  });

  const calls = [];
  const stageCamera = {
    setActiveMapId(id, options) {
      calls.push(["active", id, options]);
    },
    fit(id) {
      calls.push(["fit", id]);
    },
    getView() {
      return { zoomRelativeToFit: 1 };
    },
    setInteractionLocked(value) {
      calls.push(["lock", value]);
    },
  };
  const state = mapGrid.resetCameraToFit(stageCamera, "map-1", null, { display_mode: "fullscreen" }, "");
  assert.equal(state.interactionLocked, true);
  assert.deepEqual(calls, [
    ["active", "map-1", { reset: true }],
    ["fit", "map-1"],
    ["lock", true],
  ]);
});

test("map and grid drafts preserve the live selections and saved defaults", () => {
  assert.deepEqual(mapGrid.mapEditorDraftFromState(
    { asset_id: "map-1", display_mode: "fullscreen", fit: "contain", crop_x: 0.2, crop_y: 0.8, scale: 1.5, safe_margin: 32 },
    "map-2",
    { displayMode: "theater", fit: "cover", cropX: "0.4", cropY: "0.6", scale: "2", safeMargin: "24" },
  ), {
    assetID: "map-1",
    displayMode: "theater",
    fit: "cover",
    cropX: 0.4,
    cropY: 0.6,
    scale: 2,
    safeMargin: 24,
  });

  assert.deepEqual(mapGrid.mapEditorPayloadFromUI(
    { asset_id: "map-1", display_mode: "theater", fit: "cover", crop_x: 0.5, crop_y: 0.5, scale: 1, safe_margin: 24 },
    "map-2",
    { assetID: "map-3", fit: "contain", cropX: "0.25", cropY: "0.75", scale: "1.75", safeMargin: "18" },
  ), {
    asset_id: "map-3",
    display_mode: "theater",
    fit: "contain",
    crop_x: 0.25,
    crop_y: 0.75,
    scale: 1.75,
    safe_margin: 18,
  });

  assert.deepEqual(mapGrid.defaultGridConfig(), {
    grid_type: "none",
    hex_orientation: "flat-top",
    cell_size: 50,
    offset_x: 0,
    offset_y: 0,
    line_width: 1,
    opacity: 0.45,
    line_style: "neutral",
    visible: true,
  });

  assert.deepEqual(mapGrid.gridEditorDraftFromUI(mapGrid.defaultGridConfig(), {
    gridType: "hex",
    hexOrientation: "pointy-top",
    cellSize: "90",
    offsetX: "8",
    offsetY: "-8",
    opacity: "0.55",
    lineWidth: "2",
    lineStyle: "dark",
  }), {
    grid_type: "hex",
    hex_orientation: "pointy-top",
    cell_size: 90,
    offset_x: 8,
    offset_y: -8,
    line_width: 2,
    opacity: 0.55,
    line_style: "dark",
    visible: true,
  });

  assert.equal(mapGrid.gridEditorHexFieldVisible("hex"), true);
  assert.equal(mapGrid.gridEditorHexFieldVisible("square"), false);
  assert.equal(mapGrid.gridEditorVisibilityLabel({ visible: false }), "Show Grid");
});
