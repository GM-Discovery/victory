const test = require("node:test");
const assert = require("node:assert/strict");

const contextModule = require("../../frontend/venues/catharsis/runtime/context.js");

function makeDeps(overrides = {}) {
  const calls = [];
  const deps = {
    smokeMode: false,
    getCurrentObjects: () => [],
    getCurrentNodeMap: () => new Map(),
    getStageShell: () => ({ getBoundingClientRect: () => ({ left: 10, top: 20 }) }),
    getStageHost: () => null,
    getPixiApp: () => null,
    getLastStagePoint: () => ({ x: 0, y: 0 }),
    setLastStagePoint: (point) => calls.push(["last", point]),
    setPointerLine: (text) => calls.push(["pointer", text]),
    setSmokeLine: (text) => calls.push(["smoke", text]),
    currentSelectionSummary: () => "selection summary",
    stageContextMenuModel: () => null,
    objectKind: () => "token",
    stageScreenPointFromClient: (x, y) => ({ x: x - 10, y: y - 20 }),
    stagePointFromClient: (x, y) => ({ x, y }),
    setStagePlacementCandidate: (point, screenPoint) => calls.push(["placement", point, screenPoint]),
    setContextMenuTarget: (model) => calls.push(["target", model?.label || null]),
    closeContextMenuUI: () => calls.push(["close"]),
    ...overrides,
  };
  return { deps, calls };
}

test("context helpers identify secondary pointer events and mark handled events", () => {
  const { deps } = makeDeps();
  const helpers = contextModule.createContextInteractionHelpers(deps);
  assert.equal(helpers.isSecondaryPointerEvent({ button: 2 }), true);
  assert.equal(helpers.isSecondaryPointerEvent({ buttons: 2 }), true);
  assert.equal(helpers.isSecondaryPointerEvent({ button: 0, buttons: 1 }), false);

  const event = {};
  helpers.markContextMenuHandled(event);
  assert.equal(helpers.wasContextMenuHandled(event), true);
});

test("context helpers resolve stage points and update pointer readout", () => {
  const { deps, calls } = makeDeps();
  const helpers = contextModule.createContextInteractionHelpers(deps);
  assert.deepEqual(helpers.eventClientPoint({ clientX: 30, clientY: 40 }), { clientX: 30, clientY: 40 });
  assert.deepEqual(helpers.stageScreenPointFromClient(30, 40), { x: 20, y: 20 });
  assert.deepEqual(helpers.stagePointFromClient(30, 40), { x: 20, y: 20 });

  helpers.updatePointerReadout(30, 40, { x: 5, y: 6 }, "Overlay menu");
  assert.deepEqual(calls, [
    ["last", { x: 5, y: 6 }],
    ["pointer", "Screen 30, 40 | Stage 5, 6"],
  ]);
});

