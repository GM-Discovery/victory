const test = require("node:test");
const assert = require("node:assert/strict");

// Catharsis and First Theater each ship their own frontend/venues/<venue>/
// runtime/scene-nodes.js -- confirmed genuine drift, not a bug: Catharsis
// renders an extra token-aura Graphics layer First Theater doesn't have
// (see Construction planning notes). This contract test runs the SAME
// assertions against both modules' shared behavior only -- it deliberately
// never asserts on the aura block or on exact PIXI child counts for
// makeTokenNode, since those numbers legitimately differ between venues.
const catharsisModule = require("../../frontend/venues/catharsis/runtime/scene-nodes.js");
const firstTheaterModule = require("../../frontend/venues/first-theater/runtime/scene-nodes.js");

class FakeDisplayObject {
  constructor() {
    this.children = [];
    this.parent = null;
    this.position = { x: 0, y: 0, set(x, y) { this.x = x; this.y = y === undefined ? x : y; } };
    this.anchor = { x: 0, y: 0, set(x, y) { this.x = x; this.y = y === undefined ? x : y; } };
    this.scale = { x: 1, y: 1, set(x, y) { this.x = x; this.y = y === undefined ? x : y; } };
    this.eventMode = null;
    this.cursor = null;
    this.interactive = false;
    this.visible = true;
    this.alpha = 1;
    this.zIndex = 0;
    this.sortableChildren = false;
    this.x = 0;
    this.y = 0;
    this.width = 0;
    this.height = 0;
    this._listeners = {};
  }
  addChild(...kids) {
    for (const kid of kids) {
      kid.parent = this;
      this.children.push(kid);
    }
    return kids[0];
  }
  removeChild(kid) {
    const idx = this.children.indexOf(kid);
    if (idx !== -1) {
      this.children.splice(idx, 1);
      kid.parent = null;
    }
    return kid;
  }
  removeChildren() {
    for (const kid of this.children) kid.parent = null;
    this.children = [];
  }
  on(event, handler) {
    this._listeners[event] = handler;
    return this;
  }
}

class FakeGraphics extends FakeDisplayObject {
  beginFill() { return this; }
  endFill() { return this; }
  lineStyle() { return this; }
  drawRoundedRect() { return this; }
  drawRect() { return this; }
  drawCircle() { return this; }
  drawEllipse() { return this; }
  clear() { return this; }
}

class FakeText extends FakeDisplayObject {
  constructor(text, style) {
    super();
    this.text = text;
    this.style = style;
    this.width = String(text || "").length * 6;
  }
}

class FakeSprite extends FakeDisplayObject {
  constructor(texture) {
    super();
    this.texture = texture;
  }
  static from(url) {
    return new FakeSprite({ url });
  }
}

class FakeContainer extends FakeDisplayObject {}

function makeFakePIXI() {
  return {
    Container: FakeContainer,
    Graphics: FakeGraphics,
    Text: FakeText,
    TextStyle: function (opts) { return opts || {}; },
    Sprite: FakeSprite,
    Texture: { from: (url) => ({ url, baseTexture: { valid: true } }) },
    utils: { string2hex: (hex) => parseInt(String(hex || "").replace("#", ""), 16) || 0 },
  };
}

function makeDeps(overrides = {}) {
  const stageState = {
    currentNodeMap: new Map(),
    currentSelection: null,
    pinnedObjectLayer: null,
    overlayObjectLayer: null,
    floatingObjectLayer: null,
    facadeLayer: null,
  };
  return {
    PIXI: makeFakePIXI(),
    objectState: (model) => model.state || {},
    canEditLiveCard: () => true,
    canToggleLock: () => true,
    canManageIndexCards: () => true,
    currentRole: () => "producer",
    cardFaceForModel: () => "front",
    cardStatusBadgeText: () => "",
    viewerCanSeeHiddenCards: () => false,
    tokenDisplayNameForModel: (model) => model.label || "Token",
    tokenStatusPercentForModel: () => NaN,
    tokenLayerForModel: () => "public",
    tokenDisplaySizeForModel: () => ({ width: 64, height: 64 }),
    truncateCardText: (text) => text,
    isTokenObject: (model) => model.kind === "token",
    selectObject: () => {},
    setStageStatus: () => {},
    setMovementReport: () => {},
    openCardEditor: () => {},
    wasContextMenuHandled: () => false,
    cancelContextMenuEvent: () => {},
    openResolvedContextMenu: () => {},
    eventClientPoint: () => ({ clientX: 0, clientY: 0 }),
    stageScreenPointFromClient: () => ({ x: 0, y: 0 }),
    cardDisplayMode: () => "overlay",
    tokenSnapModeForModel: () => "free",
    renderPixiScene: () => {},
    setDragState: () => {},
    cardFaceState: new Map(),
    stageState,
    ...overrides,
  };
}

const modules = [
  ["catharsis", catharsisModule],
  ["first-theater", firstTheaterModule],
];

for (const [venueName, mod] of modules) {
  test(`${venueName} scene-nodes: exposes the same shared factory contract`, () => {
    const deps = makeDeps();
    const factory = mod.createSceneNodeFactory(deps);
    assert.equal(typeof factory.clearSceneNodes, "function");
    assert.equal(typeof factory.refreshNodeSelection, "function");
    assert.equal(typeof factory.makeCardNode, "function");
    assert.equal(typeof factory.makeFireNode, "function");
    assert.equal(typeof factory.makeTokenNode, "function");
  });

  test(`${venueName} scene-nodes: makeCardNode builds a selectable, visibility-gated node`, () => {
    const deps = makeDeps();
    const factory = mod.createSceneNodeFactory(deps);
    const model = { elementId: "card-1", label: "Card", frontText: "Front", backText: "Back", color: "#d9c7a6", state: { visible: true } };
    const node = factory.makeCardNode(model);
    assert.equal(node.model, model);
    assert.equal(typeof node.updateSelected, "function");
    assert.equal(node.container.visible, true);
    assert.ok(node.container.children.length > 0);

    // updateSelected must not throw and must reflect selection via zIndex,
    // shared behavior for every card node regardless of venue.
    node.updateSelected(true);
    assert.equal(node.container.zIndex, 50);
    node.updateSelected(false);
    assert.equal(node.container.zIndex, 10);
  });

  test(`${venueName} scene-nodes: makeCardNode hides an invisible card unless the viewer may see hidden cards`, () => {
    const hiddenModel = { elementId: "card-2", label: "Hidden Card", state: { visible: false } };

    const restrictedDeps = makeDeps({ viewerCanSeeHiddenCards: () => false });
    const restrictedNode = mod.createSceneNodeFactory(restrictedDeps).makeCardNode(hiddenModel);
    assert.equal(restrictedNode.container.visible, false);

    const privilegedDeps = makeDeps({ viewerCanSeeHiddenCards: () => true });
    const privilegedNode = mod.createSceneNodeFactory(privilegedDeps).makeCardNode(hiddenModel);
    assert.equal(privilegedNode.container.visible, true);
  });

  test(`${venueName} scene-nodes: makeFireNode builds a selectable node with a label`, () => {
    const deps = makeDeps();
    const factory = mod.createSceneNodeFactory(deps);
    const node = factory.makeFireNode({ label: "First Fire" });
    assert.equal(typeof node.updateSelected, "function");
    assert.ok(node.container.children.length > 0);
    node.updateSelected(true);
    node.updateSelected(false);
  });

  // makeTokenNode: intentionally does NOT assert on container.children
  // length or contents -- Catharsis includes an extra aura Graphics child
  // First Theater doesn't. What both venues must share is the nameplate
  // show/hide behavior and selection z-ordering, neither of which touches
  // the aura block.
  test(`${venueName} scene-nodes: makeTokenNode shares nameplate visibility and selection behavior`, () => {
    const model = {
      elementId: "token-1",
      label: "Goat",
      state: { visible: true, nameplateVisible: true },
      source: { data: {} },
    };

    const deps = makeDeps();
    const factory = mod.createSceneNodeFactory(deps);
    const node = factory.makeTokenNode(model);
    assert.equal(node.model, model);
    assert.equal(typeof node.updateSelected, "function");

    // Nameplate visible: label is attached as a child of the container.
    const labelChild = node.container.children.find((child) => child.text === "Goat");
    assert.ok(labelChild, "expected the nameplate label to be attached when nameplateVisible is true");

    // Selection moves the token to the front.
    node.updateSelected(true);
    assert.equal(node.container.zIndex, 60);
    node.updateSelected(false);
    assert.equal(node.container.zIndex, 25);

    // Hiding the nameplate and re-running the same selection-driven
    // refresh must detach the label -- this is the shared interactive
    // contract, independent of the aura drift.
    model.state.nameplateVisible = false;
    node.updateSelected(false);
    const labelAfterHide = node.container.children.find((child) => child.text === "Goat");
    assert.equal(labelAfterHide, undefined, "expected the nameplate label to be detached when nameplateVisible is false");
  });
}
