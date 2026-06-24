const test = require("node:test");
const assert = require("node:assert/strict");

const tokenUiModule = require("../../frontend/venues/first-theater/runtime/token-ui.js");

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
