const test = require("node:test");
const assert = require("node:assert/strict");

globalThis.VictoryStageVenue = { slug: "catharsis", name: "Catharsis" };
const tokenUiModule = require("../../frontend/lib/stage-runtime/token-ui.js");

test("token picker filtering respects shape and search", () => {
  const state = { search: "goat", filterShape: "circle" };
  const assets = [
    { id: "1", name: "Goat", shape: "circle", asset_type: "token", status: "active" },
    { id: "2", name: "Wolf", shape: "square", asset_type: "token", status: "active" },
  ];
  const picker = tokenUiModule.createTokenUi({
    state,
    canManageStageTokens: () => true,
    tokenPlacementPointForCreate: (point) => point,
    tokenScaleForModel: () => 100,
    tokenSnapModeForModel: () => "free",
    tokenLayerForModel: () => "public",
    renderPixiScene: () => {},
    setStageStatus: () => {},
    setMovementLine: () => {},
    formatByteSize: (n) => `${n} B`,
    escapeHtml: (v) => String(v),
    sendAction: () => true,
    refreshVenueMapAssets: () => Promise.resolve(),
    currentGridConfig: () => ({ grid_type: "none" }),
    lastStagePoint: () => null,
    stagePlacementCandidate: () => null,
    stagePlacementScreenCandidate: () => null,
    setPlacementState: () => {},
    openEditor: () => {},
    closeEditor: () => {},
    setEditorStatus: () => {},
    getEditorState: () => ({}),
    getTokenAssets: () => assets,
    setTokenAssets: () => {},
    getTokenPickerElements: () => ({
      list: { innerHTML: "", appendChild() {} },
      preview: { src: "" },
      previewBadge: { textContent: "" },
      search: { value: "goat" },
      shape: { value: "circle" },
      status: { textContent: "" },
      setPosition: () => {},
    }),
    refreshWarehouseTokenAssets: () => Promise.resolve(),
    clampNumber: (value) => Number(value),
    objectState: () => ({}),
    isTokenObject: () => true,
    currentObjects: () => [],
    updateTokenLocalModel: () => {},
  });

  const filtered = picker.tokenPickerFilteredAssets();
  assert.equal(filtered.length, 1);
  assert.equal(filtered[0].name, "Goat");
});

test("token picker placement uses the latest live stage point on create click", () => {
  const state = { search: "", filterShape: "all", placementPoint: { x: 10, y: 10 }, placementScreenPoint: { x: 10, y: 10 } };
  const sent = [];
  const listeners = new Map();
  const assetButton = {
    type: "button",
    classList: { add() {}, contains() { return false; } },
    innerHTML: "",
    addEventListener(type, handler) { listeners.set(type, handler); },
    click() {
      const handler = listeners.get("click");
      if (handler) handler({ preventDefault() {}, stopPropagation() {} });
    },
  };
  const originalDocument = global.document;
  global.document = {
    createElement: () => assetButton,
  };

  try {
    const picker = tokenUiModule.createTokenUi({
      state,
      canManageStageTokens: () => true,
      tokenPlacementPointForCreate: (point) => point,
      tokenScaleForModel: () => 100,
      tokenSnapModeForModel: () => "grid",
      tokenLayerForModel: () => "public",
      renderPixiScene: () => {},
      setStageStatus: () => {},
      setMovementLine: () => {},
      formatByteSize: (n) => `${n} B`,
      escapeHtml: (v) => String(v),
      sendAction: (type, payload) => { sent.push([type, payload]); return true; },
      refreshVenueMapAssets: () => Promise.resolve(),
      currentGridConfig: () => ({ grid_type: "square" }),
      lastStagePoint: () => ({ x: 20, y: 20 }),
      stagePlacementCandidate: () => ({ x: 30, y: 30 }),
      stagePlacementScreenCandidate: () => ({ x: 30, y: 30 }),
      setPlacementState: () => {},
      openEditor: () => {},
      closeEditor: () => {},
      setEditorStatus: () => {},
      getEditorState: () => ({}),
      getTokenAssets: () => [{ id: "1", name: "Goat", shape: "circle", asset_type: "token", status: "active", content_url: "/api/assets/1/content" }],
      setTokenAssets: () => {},
      getTokenPickerElements: () => ({
        list: { innerHTML: "", appendChild() {} },
        preview: { src: "" },
        previewBadge: { textContent: "" },
        search: { value: "" },
        shape: { value: "all" },
        status: { textContent: "" },
        setPosition: () => {},
        panel: { hidden: false },
      }),
      refreshWarehouseTokenAssets: () => Promise.resolve(),
      clampNumber: (value) => Number(value),
      objectState: () => ({}),
      isTokenObject: () => true,
      currentObjects: () => [],
      updateTokenLocalModel: () => {},
    });

    picker.renderTokenPickerList();
    assetButton.click();

    assert.equal(sent.length, 1);
    assert.equal(sent[0][0], "create/token");
    assert.deepEqual(sent[0][1].x, 30);
    assert.deepEqual(sent[0][1].y, 30);
  } finally {
    global.document = originalDocument;
  }
});
