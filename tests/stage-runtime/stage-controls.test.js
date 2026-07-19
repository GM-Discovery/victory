const test = require("node:test");
const assert = require("node:assert/strict");

const stageControls = require("../../frontend/venues/catharsis/runtime/stage-controls.js");

test("camera helpers report label and lock state consistently", () => {
  assert.equal(stageControls.cameraViewLabel({ zoomRelativeToFit: 1.23 }), "123%");
  assert.deepEqual(stageControls.cameraControlsState({ zoomRelativeToFit: 2 }, "fullscreen", { minZoom: 0.7, maxZoom: 4 }), {
    label: "200%",
    zoomOutDisabled: false,
    zoomInDisabled: false,
    interactionLocked: false,
  });
  assert.equal(stageControls.cameraControlsState({ zoomRelativeToFit: 1 }, "theater").interactionLocked, true);
});

test("map editor and grid editor drafts preserve the saved defaults", () => {
  assert.deepEqual(stageControls.mapEditorDraftFromState(
    { asset_id: "map-1", display_mode: "fullscreen", fit: "contain", crop_x: 0.4, crop_y: 0.6, scale: 1.5, safe_margin: 32 },
    "map-2",
    { displayMode: "theater", fit: "cover", scale: "2", cropX: "0.25", cropY: "0.75", safeMargin: "24" },
  ), {
    assetID: "map-1",
    displayMode: "theater",
    fit: "cover",
    cropX: 0.25,
    cropY: 0.75,
    scale: 2,
    safeMargin: 24,
  });

  assert.deepEqual(stageControls.defaultGridConfig(), {
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

  assert.deepEqual(stageControls.gridEditorDraftFromUI(stageControls.defaultGridConfig(), {
    gridType: "hex",
    hexOrientation: "pointy-top",
    cellSize: "80",
    offsetX: "12",
    offsetY: "-8",
    opacity: "0.6",
    lineWidth: "2",
    lineStyle: "dark",
  }), {
    grid_type: "hex",
    hex_orientation: "pointy-top",
    cell_size: 80,
    offset_x: 12,
    offset_y: -8,
    line_width: 2,
    opacity: 0.6,
    line_style: "dark",
    visible: true,
  });

  assert.equal(stageControls.gridEditorHexFieldVisible("hex"), true);
  assert.equal(stageControls.gridEditorHexFieldVisible("square"), false);
  assert.equal(stageControls.gridEditorVisibilityLabel({ visible: false }), "Show Grid");
});
