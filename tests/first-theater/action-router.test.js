const test = require("node:test");
const assert = require("node:assert/strict");

const actionRouterModule = require("../../frontend/venues/first-theater/runtime/action-router.js");

function makeDeps(overrides = {}) {
  const calls = [];
  const contextMenu = { hidden: true, innerHTML: "", dataset: {} };
  const deps = {
    objectState: (model) => model.state || {},
    objectKind: (model) => model.kind,
    isTokenObject: (model) => model.kind === "token",
    isLiveStageObject: (model) => Boolean(model.live),
    setStageStatus: (text) => calls.push(["stage", text]),
    setMovementReport: (text) => calls.push(["move", text]),
    setMovementLine: (text) => calls.push(["line", text]),
    selectObject: (model, reason) => calls.push(["select", model?.label || null, reason]),
    closeContextMenu: () => calls.push(["close"]),
    openTokenPicker: (...args) => calls.push(["token-picker", ...args]),
    openTokenEditor: (...args) => calls.push(["token-editor", ...args]),
    openMapEditor: () => calls.push(["map-editor"]),
    openGridEditor: () => calls.push(["grid-editor"]),
    updateLocalObjectModel: (model, updater) => {
      updater(model);
      calls.push(["update-object"]);
    },
    updateTokenLocalModel: (model, updater) => {
      updater(model);
      calls.push(["update-token"]);
    },
    updateLocalCardPinModel: (model, updater) => {
      updater(model);
      calls.push(["update-card"]);
    },
    setLocalPositionOverrideForModel: (model, point) => {
      model.positionOverride = point;
      calls.push(["override", point]);
      return point;
    },
    tokenMoveTargetForModel: () => ({ snapMode: "grid" }),
    cardMoveTargetForModel: () => ({ pinMode: "world", world_x: 11, world_y: 22, screen_x: 33, screen_y: 44 }),
    tokenLayerForModel: () => "public",
    tokenScaleForModel: () => 125,
    tokenPlacementPointForCreate: (point) => point,
    currentDisplayedPointForModel: () => ({ x: 8, y: 9 }),
    tokenDuplicatePlacementForModel: () => ({ x: 14, y: 15, world_x: 0, world_y: 0, screen_x: 0, screen_y: 0, pinMode: "overlay" }),
    cardDuplicatePlacementForModel: () => ({ x: 21, y: 22, world_x: 31, world_y: 32, screen_x: 41, screen_y: 42, pinMode: "world" }),
    sendAction: (type, payload) => {
      calls.push(["send", type, payload]);
      return true;
    },
    syncCurrentObjectsFromProjectedState: () => {
      calls.push(["sync-current"]);
      return true;
    },
    removeLocalObject: (model) => calls.push(["remove-local", model?.label || null]),
    setPlacementState: () => {},
    getStagePoint: () => ({ x: 100, y: 200 }),
    stageScreenPointFromClient: (x, y) => ({ x, y }),
    stagePointFromClient: (x, y) => ({ x, y }),
    stagePlacementFromEvent: (event) => {
      calls.push(["stage-placement", Number(event?.clientX || 0), Number(event?.clientY || 0)]);
      return { x: Number(event?.clientX || 0), y: Number(event?.clientY || 0) };
    },
    hitTestContextMenuTarget: () => null,
    stageContextMenuModel: () => null,
    eventClientPoint: () => ({ clientX: 10, clientY: 20 }),
    resolveStageObjectActions: () => [],
    resolveStageObjectActionsContext: () => ({}),
    contextMenu: () => contextMenu,
    pixiApp: () => ({ view: {} }),
    cameraControlsContains: () => false,
    isStageEvent: () => true,
    isSecondaryPointerEvent: () => false,
    markContextMenuHandled: () => calls.push(["handled"]),
    renderContextMenu: (items) => calls.push(["render", items.length]),
    positionContextMenu: (x, y) => calls.push(["position", x, y]),
    setContextMenuTarget: (model) => calls.push(["target", model?.label || null]),
    closeContextMenuUI: () => calls.push(["close-ui"]),
    setStagePlacementCandidate: (point, screenPoint) => calls.push(["placement", point, screenPoint]),
    ...overrides,
  };
  return { deps, calls, contextMenu };
}

test("performStageObjectAction routes token move-here and nameplate visibility", () => {
  const { deps, calls } = makeDeps();
  const router = actionRouterModule.createActionRouter(deps);
  const token = {
    kind: "token",
    label: "Goat",
    elementId: "asset-1",
    elementSlug: "goat",
    position: { order: 4 },
    state: {},
    source: { data: {} },
  };

  router.performStageObjectAction("move-here", token);
  assert.deepEqual(calls.find((entry) => entry[0] === "send"), [
    "send",
    "update/token",
    {
      element_id: "asset-1",
      element_slug: "goat",
      venue_slug: "the-cave",
      layer: "stage",
      x: 100,
      y: 200,
      order: 4,
      snap_mode: "grid",
      token_layer: "public",
      scale: 125,
    },
  ]);
  assert.ok(calls.some((entry) => entry[0] === "update-token"));
  assert.ok(calls.some((entry) => entry[0] === "close"));

  calls.length = 0;
  router.performStageObjectAction("hide-nameplate", token);
  assert.deepEqual(calls.find((entry) => entry[0] === "send"), [
    "send",
    "act/set_nameplate_visibility",
    {
      element_id: "asset-1",
      element_slug: "goat",
      layer: "audience",
      visible: false,
    },
  ]);
  assert.equal(token.state.nameplate_visible, false);
  assert.equal(token.state.nameplateVisible, false);

  calls.length = 0;
  router.performStageObjectAction("hide", token);
  assert.deepEqual(calls.find((entry) => entry[0] === "send"), [
    "send",
    "act/hide_element",
    {
      element_id: "asset-1",
      element_slug: "goat",
      layer: "audience",
    },
  ]);
  assert.equal(token.visibility.visible, false);
});

test("performStageObjectAction routes token scale to the token editor", () => {
  const { deps, calls } = makeDeps();
  const router = actionRouterModule.createActionRouter(deps);
  const token = {
    kind: "token",
    live: true,
    label: "Goat",
    elementId: "asset-1",
    elementSlug: "goat",
    state: {},
    source: { data: {} },
  };

  router.performStageObjectAction("scale", token);
  assert.ok(calls.some((entry) => entry[0] === "token-editor" && entry[1] === token));
  assert.ok(calls.some((entry) => entry[0] === "close"));
});

test("performStageObjectAction routes card move-here and remove", () => {
  const { deps, calls } = makeDeps();
  const router = actionRouterModule.createActionRouter(deps);
  const card = {
    kind: "card",
    label: "Card",
    elementId: "card-1",
    elementSlug: "card-one",
    frontText: "Front",
    backText: "Back",
    color: "#d9c7a6",
    position: { order: 2 },
    state: {},
    source: { data: {} },
  };

  router.performStageObjectAction("move-here", card);
  assert.deepEqual(calls.filter((entry) => entry[0] === "send").map((entry) => entry[1]), [
    "update/index_card",
    "act/place_element",
  ]);
  assert.ok(calls.some((entry) => entry[0] === "update-card"));
  assert.ok(calls.some((entry) => entry[0] === "close"));

  calls.length = 0;
  router.performStageObjectAction("remove", card);
  assert.deepEqual(calls.find((entry) => entry[0] === "send"), [
    "send",
    "act/remove_element",
    {
      element_id: "card-1",
      element_slug: "card-one",
      venue_slug: "the-cave",
      layer: "stage",
    },
  ]);
  assert.ok(calls.some((entry) => entry[0] === "remove-local"));
  assert.ok(calls.some((entry) => entry[0] === "select" && entry[1] === null));
  assert.ok(calls.some((entry) => entry[0] === "sync-current"));
});

test("resolveContextMenuTargetFromClient and openContextMenu use injected stage helpers", () => {
  const { deps, calls, contextMenu } = makeDeps({
    hitTestContextMenuTarget: () => ({ label: "Hit", kind: "token" }),
    resolveStageObjectActions: () => [{ action: "select", label: "Select", group: "base" }],
  });
  const router = actionRouterModule.createActionRouter(deps);

  const resolved = router.resolveContextMenuTargetFromClient(11, 12);
  assert.equal(resolved.objectModel.label, "Hit");
  assert.deepEqual(resolved.screenPoint, { x: 11, y: 12 });
  assert.deepEqual(resolved.stagePoint, { x: 11, y: 12 });

  router.openContextMenu({ clientX: 30, clientY: 40 }, { label: "Menu object", kind: "stage" });
  assert.ok(calls.some((entry) => entry[0] === "target"));
  assert.ok(calls.some((entry) => entry[0] === "stage-placement"));
  assert.ok(calls.some((entry) => entry[0] === "render"));
  assert.ok(calls.some((entry) => entry[0] === "position"));
});

test("handleNativeStageContextMenu ignores already-handled events", () => {
  const { deps, calls } = makeDeps({
    wasContextMenuHandled: () => true,
  });
  const router = actionRouterModule.createActionRouter(deps);
  router.handleNativeStageContextMenu({ target: {}, type: "contextmenu" });
  assert.equal(calls.some((entry) => entry[0] === "render"), false);
});
