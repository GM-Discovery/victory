const test = require("node:test");
const assert = require("node:assert/strict");

const geometry = require("../../frontend/venues/first-theater/runtime/geometry.js");

function tokenModel(overrides = {}) {
  return {
    kind: "token",
    live: true,
    position: { x: 12, y: 18, order: 4 },
    source: {
      data: {
        default_grid_width: 2,
        default_grid_height: 3,
        scale: 150,
        snap_mode: "grid",
        token_layer: "director",
        ...overrides.data,
      },
    },
    ...overrides,
  };
}

function cardModel(overrides = {}) {
  return {
    kind: "card",
    live: true,
    position: { x: 0.25, y: 0.5, frame: "screen" },
    source: {
      data: {
        pin_mode: "overlay",
        screen_x: 0.25,
        screen_y: 0.5,
        ...overrides.data,
      },
    },
    ...overrides,
  };
}

test("screen/world conversion and camera clamping stay deterministic", () => {
  assert.deepEqual(geometry.screenToWorld({ x: 120, y: 90 }, { panX: 20, panY: 10, zoomRelativeToFit: 2 }), { x: 50, y: 40 });
  assert.deepEqual(geometry.worldToScreen({ x: 50, y: 40 }, { panX: 20, panY: 10, zoomRelativeToFit: 2 }), { x: 120, y: 90 });

  const clamped = geometry.cameraStateFromView(
    { panX: 999, panY: -999, zoomRelativeToFit: 1 },
    { x: 0, y: 0, width: 400, height: 300 },
    { x: 0, y: 0, width: 200, height: 100 },
  );
  assert.deepEqual(clamped, { panX: 100, panY: 100, zoomRelativeToFit: 1 });
});

test("square and hex snapping use the expected center points", () => {
  const square = geometry.snapPoint(
    { gridType: "square", cellSize: 80, offsetX: 0, offsetY: 0, visible: true },
    { x: 0, y: 0, width: 400, height: 400 },
    { x: 12, y: 13 },
  );
  assert.deepEqual(square, { x: 40, y: 40 });

  const flatHex = geometry.snapPoint(
    { gridType: "hex", hexOrientation: "flat-top", cellSize: 80, offsetX: 0, offsetY: 0, visible: true },
    { x: 0, y: 0, width: 400, height: 400 },
    { x: 58, y: 102 },
  );
  assert.deepEqual(flatHex, { x: 60, y: 104 });

  const pointyHex = geometry.snapPoint(
    { gridType: "hex", hexOrientation: "pointy-top", cellSize: 80, offsetX: 0, offsetY: 0, visible: true },
    { x: 0, y: 0, width: 400, height: 400 },
    { x: 102, y: 58 },
  );
  assert.deepEqual(pointyHex, { x: 104, y: 60 });
});

test("token size, scale, and movement helpers stay consistent", () => {
  const model = tokenModel();
  assert.equal(geometry.tokenSnapModeForModel(model, { grid_type: "square", cell_size: 80 }), "grid");
  assert.equal(geometry.tokenLayerForModel(model), "director");
  assert.equal(geometry.tokenScaleForModel(model), 150);
  assert.deepEqual(geometry.tokenFootprintForModel(model), { width: 2, height: 3 });
  assert.equal(geometry.tokenPlacementBaseSize(model, { grid_type: "square", cell_size: 80 }), 80);
  assert.deepEqual(geometry.tokenDisplaySizeForModel(model, { grid_type: "square", cell_size: 80 }), { width: 240, height: 360 });
  assert.deepEqual(geometry.tokenPlacementPointForCreate({ x: 12, y: 13 }, model, {
    forcedSnapMode: "grid",
    gridConfig: { grid_type: "square", cell_size: 80, offset_x: 0, offset_y: 0, visible: true },
    currentVenueMapBounds: { x: 0, y: 0, width: 400, height: 400 },
    snapPoint: geometry.snapPoint,
  }), { x: 40, y: 40 });
  assert.deepEqual(geometry.tokenMoveTargetForModel(model, { x: 12, y: 13 }, {
    gridConfig: { grid_type: "square", cell_size: 80, offset_x: 0, offset_y: 0, visible: true },
    currentVenueMapBounds: { x: 0, y: 0, width: 400, height: 400 },
    snapPoint: geometry.snapPoint,
  }), { x: 40, y: 40, snapMode: "grid" });
});

test("card coordinate-mode conversion preserves world and overlay placement", () => {
  const worldCard = {
    kind: "card",
    live: true,
    position: { x: 10, y: 20, frame: "world" },
    source: {
      data: {
        pin_mode: "world",
        world_x: 10,
        world_y: 20,
      },
    },
  };
  assert.equal(geometry.cardDisplayMode(worldCard), "world");
  assert.deepEqual(geometry.currentDisplayedPointForModel(worldCard, {
    camera: { worldToScreen: (x, y) => ({ x: x + 5, y: y + 7 }) },
    getPlayableBounds: () => ({ x: 0, y: 0, width: 400, height: 300 }),
    toStagePoint: () => ({ x: 0, y: 0 }),
    stagePointForWorldPoint: (point) => ({ x: point.x + 5, y: point.y + 7 }),
    worldPointForStagePoint: (point) => point,
    size: { width: 960, height: 640 },
  }), { x: 15, y: 27 });

  const overlayCard = cardModel();
  assert.equal(geometry.cardDisplayMode(overlayCard), "overlay");
  assert.deepEqual(geometry.currentDisplayedPointForModel(overlayCard, {
    camera: null,
    getPlayableBounds: () => ({ x: 10, y: 20, width: 100, height: 200 }),
    toStagePoint: () => ({ x: 0, y: 0 }),
    stagePointForWorldPoint: (point) => point,
    worldPointForStagePoint: (point) => point,
    size: { width: 960, height: 640 },
  }), { x: 35, y: 120 });

  assert.deepEqual(geometry.cardMoveTargetForModel(worldCard, { x: 8, y: 9 }, {
    bounds: { x: 0, y: 0, width: 100, height: 100 },
  }), {
    pinMode: "world",
    world_x: 8,
    world_y: 9,
    screen_x: null,
    screen_y: null,
  });

  assert.deepEqual(geometry.cardDuplicatePlacementForModel(overlayCard, {
    getPlayableBounds: () => ({ x: 10, y: 20, width: 100, height: 200 }),
    toStagePoint: () => ({ x: 0, y: 0 }),
    stagePointForWorldPoint: (point) => point,
    worldPointForStagePoint: (point) => point,
    size: { width: 960, height: 640 },
    bounds: { x: 10, y: 20, width: 100, height: 200 },
  }), {
    pinMode: "overlay",
    x: 59,
    y: 136,
    world_x: null,
    world_y: null,
    screen_x: 0.49,
    screen_y: 0.58,
  });
});

