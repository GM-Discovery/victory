const test = require("node:test");
const assert = require("node:assert/strict");

const editorsModule = require("../../frontend/venues/first-theater/runtime/editors.js");

function makeElement(tagName = "div") {
  const listeners = new Map();
  return {
    tagName: tagName.toUpperCase(),
    hidden: false,
    value: "",
    textContent: "",
    src: "",
    style: {
      setProperty() {},
    },
    classList: {
      _set: new Set(),
      add(...values) { values.forEach((value) => this._set.add(value)); },
      remove(...values) { values.forEach((value) => this._set.delete(value)); },
      contains(value) { return this._set.has(value); },
    },
    children: [],
    innerHTML: "",
    appendChild(node) { this.children.push(node); return node; },
    addEventListener(type, handler) {
      listeners.set(type, handler);
    },
    dispatchEvent(event) {
      const handler = listeners.get(event.type);
      if (handler) handler.call(this, event);
    },
    click() {
      this.dispatchEvent({ type: "click", target: this, stopPropagation() {}, preventDefault() {} });
    },
    focus() {},
  };
}

test("editor controllers expose the expected wiring surface", () => {
  const panel = { hidden: true };
  const deps = {
    cardEditorPanel: panel,
    cardEditorFront: { value: "", focus() {} },
    cardEditorBack: { value: "" },
    cardEditorColor: { value: "" },
    cardEditorStatus: { textContent: "" },
    gridEditorPanel: panel,
    defaultGridConfig: () => ({ grid_type: "none", visible: true }),
    gridEditorDraftFromUI: () => ({ grid_type: "hex", visible: true }),
    syncGridEditorVisibilityButton: () => {},
    syncGridEditorHexFieldVisibility: () => {},
    syncGridEditorWithState: () => {},
    syncMapEditorPreview: () => {},
    syncMapAssetList: () => {},
    syncMapEditorWithState: () => {},
    livePreviewMapOnStage: () => {},
    mapEditorPanel: panel,
    mapEditorFile: { focus() {}, value: "" },
    mapEditorStatus: { textContent: "" },
    gridEditorStatus: { textContent: "" },
    setCardEditorTargetKey: () => {},
    setCardEditorDirty: () => {},
    setGridEditorDirty: () => {},
    setGridEditorOriginalState: () => {},
    setMapEditorDirty: () => {},
    setMapEditorOriginalState: () => {},
    setMapEditorPosition: () => {},
    setGridEditorPosition: () => {},
    clampMapEditorPosition: (left, top) => ({ left, top }),
    clampGridEditorPosition: (left, top) => ({ left, top }),
    clampCardEditorPosition: (left, top) => ({ left, top }),
    renderVenueGridLayer: () => {},
    renderPixiScene: () => {},
    setStageStatus: () => {},
    setMovementLine: () => {},
    canManageIndexCards: () => true,
    canManageStageTokens: () => true,
    isCardObject: () => true,
    isTokenObject: () => true,
    objectState: () => ({ locked: false }),
    selectObject: () => {},
    sendAction: () => true,
    updateLocalObjectModel: () => {},
    updateLocalCardPinModel: () => {},
    updateTokenLocalModel: () => {},
    setLocalPositionOverrideForModel: () => null,
    currentDisplayedPointForModel: () => ({ x: 0, y: 0 }),
    tokenScaleForModel: () => 100,
    tokenSnapModeForModel: () => "free",
    tokenLayerForModel: () => "public",
    tokenPlacementPointForCreate: (point) => point,
    tokenDuplicatePlacementForModel: () => null,
    cardDuplicatePlacementForModel: () => null,
    tokenMoveTargetForModel: () => null,
    cardMoveTargetForModel: () => null,
    refreshVenueMapAssets: async () => {},
    refreshVenueMapState: async () => {},
    refreshVenueGridConfig: async () => {},
    escapeHtml: (value) => String(value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;"),
    uploadMapAsset: async () => null,
    saveVenueMap: async () => {},
    removeVenueMap: async () => {},
    saveGridConfig: async () => {},
    getCurrentRole: () => "producer",
    getCurrentSelection: () => null,
    getCurrentObjects: () => [],
    getCurrentVenueMapState: () => null,
    getCurrentVenueMapAssetID: () => "",
    getCurrentVenueMapAssets: () => [],
    setCurrentVenueMapState: () => {},
    setCurrentVenueMapAssetID: () => {},
    setCurrentVenueMapAssets: () => {},
    getCurrentVenueGridConfig: () => null,
    setCurrentVenueGridConfig: () => {},
    getCurrentSnapshot: () => null,
    getStageShell: () => null,
    getCardEditorTarget: () => null,
    hideMapEditor: () => {},
    hideGridEditor: () => {},
    hideCardEditor: () => {},
    hideTokenEditor: () => {},
    closeTokenPicker: () => {},
    openTokenPicker: () => {},
    canActorRevealHideStageObjects: () => true,
    setCurrentSelection: () => {},
    syncSelectedActions: () => {},
    syncTokenEditorWithSelection: () => {},
    getMapEditorOriginalState: () => null,
    setMapEditorOriginalState: () => {},
    setMapEditorDirty: () => {},
    getGridEditorOriginalState: () => null,
    setGridEditorOriginalState: () => {},
    setGridEditorDirty: () => {},
    mapEditorDraftFromState: () => ({}),
    mapEditorPayloadFromUI: () => ({}),
    createIndexCardFromMenu: () => {},
    placeCreatedIndexCard: () => {},
  };

  const controllers = editorsModule.createEditorControllers(deps);
  assert.equal(typeof controllers.openCardEditor, "function");
  assert.equal(typeof controllers.openMapEditor, "function");
  assert.equal(typeof controllers.openGridEditor, "function");
  assert.equal(typeof controllers.saveGridConfig, "function");
  assert.equal(typeof controllers.refreshVenueGridConfig, "function");
});

test("map editor sync renders assets and state without recursive wrappers", () => {
  const assetsContainer = makeElement("div");
  const preview = makeElement("img");
  const previewMode = makeElement("span");
  const previewFocus = makeElement("div");
  const previewSafe = makeElement("div");
  const displayMode = makeElement("select");
  const fit = makeElement("select");
  const scale = makeElement("input");
  const cropX = makeElement("input");
  const cropY = makeElement("input");
  const safeMargin = makeElement("input");

  let currentVenueMapState = {
    asset: { content_url: "/api/assets/map-1/content" },
    asset_id: "map-1",
    display_mode: "theater",
    fit: "contain",
    crop_x: 0.25,
    crop_y: 0.75,
    scale: 1.5,
    safe_margin: 32,
  };
  let currentVenueMapAssetID = "map-1";
  const currentVenueMapAssets = [
    { id: "map-1", name: "Map One", asset_type: "map", default_grid_width: 1, default_grid_height: 1, content_url: "/api/assets/map-1/content", thumbnail_url: "/api/assets/map-1/content?variant=thumbnail" },
  ];

  const originalDocument = global.document;
  global.document = {
    createElement: makeElement,
  };

  try {
    const controllers = editorsModule.createEditorControllers({
      getMapEditorElements: () => ({
        panel: makeElement("div"),
        preview,
        previewMode,
        previewFocus,
        previewSafe,
        displayMode,
        fit,
        scale,
        cropX,
        cropY,
        safeMargin,
        assets: assetsContainer,
      }),
      getCurrentVenueMapState: () => currentVenueMapState,
      getCurrentVenueMapAssetID: () => currentVenueMapAssetID,
      getCurrentVenueMapAssets: () => currentVenueMapAssets,
      setCurrentVenueMapState: (value) => { currentVenueMapState = value; },
      setCurrentVenueMapAssetID: (value) => { currentVenueMapAssetID = value; },
      setCurrentVenueMapAssets: () => {},
      getCurrentVenueGridConfig: () => ({ grid_type: "square", visible: true }),
      setCurrentVenueGridConfig: () => {},
      defaultGridConfig: () => ({ grid_type: "none", visible: true }),
      getPlayableBounds: () => ({ x: 0, y: 0, width: 100, height: 100 }),
      renderPixiScene: () => {},
      renderVenueGridLayer: () => {},
      setMapEditorStatus: () => {},
      setGridEditorStatus: () => {},
      setMapEditorPosition: () => {},
      setGridEditorPosition: () => {},
      clampMapEditorPosition: (left, top) => ({ left, top }),
      clampGridEditorPosition: (left, top) => ({ left, top }),
      clampCardEditorPosition: (left, top) => ({ left, top }),
      syncGridEditorHexFieldVisibility: () => {},
      syncGridEditorVisibilityButton: () => {},
      syncGridEditorWithState: () => {},
      syncMapEditorPreview: () => {},
      syncMapAssetList: () => {},
      livePreviewMapOnStage: () => {},
      canManageIndexCards: () => true,
      canManageStageTokens: () => true,
      isCardObject: () => true,
      isTokenObject: () => true,
      objectState: () => ({ locked: false }),
      selectObject: () => {},
      setStageStatus: () => {},
      setMovementLine: () => {},
      sendAction: () => true,
      updateLocalObjectModel: () => {},
      updateLocalCardPinModel: () => {},
      updateTokenLocalModel: () => {},
      setLocalPositionOverrideForModel: () => null,
      currentDisplayedPointForModel: () => ({ x: 0, y: 0 }),
      tokenScaleForModel: () => 100,
      tokenSnapModeForModel: () => "free",
      tokenLayerForModel: () => "public",
      tokenPlacementPointForCreate: (point) => point,
      tokenDuplicatePlacementForModel: () => null,
      cardDuplicatePlacementForModel: () => null,
      tokenMoveTargetForModel: () => null,
      cardMoveTargetForModel: () => null,
      refreshVenueMapAssets: async () => {},
      refreshVenueMapState: async () => {},
      refreshVenueGridConfig: async () => {},
      uploadMapAsset: async () => null,
      saveVenueMap: async () => {},
      removeVenueMap: async () => {},
      saveGridConfig: async () => {},
      getCurrentRole: () => "producer",
      getCurrentSelection: () => null,
      getCurrentObjects: () => [],
      getCurrentSnapshot: () => null,
      getStageShell: () => null,
      getCardEditorTarget: () => null,
      hideMapEditor: () => {},
      hideGridEditor: () => {},
      hideCardEditor: () => {},
      hideTokenEditor: () => {},
      closeTokenPicker: () => {},
      openTokenPicker: () => {},
      canActorRevealHideStageObjects: () => true,
      setCurrentSelection: () => {},
      syncSelectedActions: () => {},
      syncTokenEditorWithSelection: () => {},
      getMapEditorOriginalState: () => null,
      setMapEditorOriginalState: () => {},
      setMapEditorDirty: () => {},
      getGridEditorOriginalState: () => null,
      setGridEditorOriginalState: () => {},
      setGridEditorDirty: () => {},
      mapEditorDraftFromState: () => ({ fit: "contain", cropX: 0.25, cropY: 0.75, scale: 1.5, safeMargin: 32 }),
      mapEditorPayloadFromUI: () => ({}),
      createIndexCardFromMenu: () => {},
      placeCreatedIndexCard: () => {},
    });

    controllers.syncMapEditorWithState();
    assert.equal(displayMode.value, "theater");
    assert.equal(fit.value, "contain");
    assert.equal(preview.src, "/api/assets/map-1/content");
    assert.ok(assetsContainer.children.length >= 1);
  } finally {
    global.document = originalDocument;
  }
});

test("grid editor live sync previews draft state without mutating saved config", () => {
  const renders = [];
  let currentVenueGridConfig = {
    grid_type: "hex",
    hex_orientation: "flat-top",
    cell_size: 60,
    offset_x: 4,
    offset_y: 8,
    line_width: 1,
    opacity: 0.45,
    line_style: "neutral",
    visible: true,
  };

  const controllers = editorsModule.createEditorControllers({
    getGridEditorElements: () => ({
      type: { value: "square" },
      hexOrientation: { value: "pointy-top" },
      cellSize: { value: "80" },
      offsetX: { value: "12" },
      offsetY: { value: "-8" },
      opacity: { value: "0.6" },
      lineWidth: { value: "2" },
      lineStyle: { value: "dark" },
      visibility: { textContent: "Show Grid" },
    }),
    getCurrentVenueGridConfig: () => currentVenueGridConfig,
    setCurrentVenueGridConfig: (value) => { currentVenueGridConfig = value; },
    gridEditorDraftFromUI: () => ({
      grid_type: "square",
      hex_orientation: "pointy-top",
      cell_size: 80,
      offset_x: 12,
      offset_y: -8,
      line_width: 2,
      opacity: 0.6,
      line_style: "dark",
      visible: true,
    }),
    defaultGridConfig: () => ({
      grid_type: "none",
      hex_orientation: "flat-top",
      cell_size: 50,
      offset_x: 0,
      offset_y: 0,
      line_width: 1,
      opacity: 0.45,
      line_style: "neutral",
      visible: true,
    }),
    getCurrentVenueMapBounds: () => ({ x: 0, y: 0, width: 100, height: 100 }),
    getPlayableBounds: () => ({ x: 0, y: 0, width: 100, height: 100 }),
    renderVenueGridLayer: (...args) => renders.push(args),
    syncGridEditorHexFieldVisibility: () => {},
    syncGridEditorVisibilityButton: () => {},
    syncGridEditorWithState: () => {},
    setGridEditorDirty: () => {},
    setGridEditorStatus: () => {},
  });

  controllers.liveSyncGrid();

  assert.equal(currentVenueGridConfig.grid_type, "hex");
  assert.equal(renders.length, 1);
  assert.equal(renders[0][1].grid_type, "square");
  assert.equal(renders[0][1].hex_orientation, "pointy-top");
  assert.equal(renders[0][1].cell_size, 80);
});

test("grid editor sync updates controls without recursive wrappers", () => {
  const visibility = makeElement("button");
  const hexOrientationField = makeElement("label");
  const type = makeElement("select");
  const hexOrientation = makeElement("select");
  const cellSize = makeElement("input");
  const offsetX = makeElement("input");
  const offsetY = makeElement("input");
  const opacity = makeElement("input");
  const lineWidth = makeElement("input");
  const lineStyle = makeElement("select");
  let currentVenueGridConfig = {
    grid_type: "hex",
    hex_orientation: "pointy-top",
    cell_size: 72,
    offset_x: 12,
    offset_y: -8,
    line_width: 2,
    opacity: 0.6,
    line_style: "dark",
    visible: false,
  };

  const controllers = editorsModule.createEditorControllers({
    getGridEditorElements: () => ({
      panel: makeElement("div"),
      type,
      hexOrientation,
      hexOrientationField,
      cellSize,
      offsetX,
      offsetY,
      opacity,
      lineWidth,
      lineStyle,
      visibility,
    }),
    getCurrentVenueGridConfig: () => currentVenueGridConfig,
    setCurrentVenueGridConfig: (value) => { currentVenueGridConfig = value; },
    defaultGridConfig: () => ({
      grid_type: "none",
      hex_orientation: "flat-top",
      cell_size: 50,
      offset_x: 0,
      offset_y: 0,
      line_width: 1,
      opacity: 0.45,
      line_style: "neutral",
      visible: true,
    }),
    getCurrentVenueMapBounds: () => ({ x: 0, y: 0, width: 100, height: 100 }),
    getPlayableBounds: () => ({ x: 0, y: 0, width: 100, height: 100 }),
    renderVenueGridLayer: () => {},
    syncGridEditorHexFieldVisibility: () => {},
    syncGridEditorVisibilityButton: () => {},
    syncGridEditorWithState: () => {},
    setGridEditorDirty: () => {},
    setGridEditorStatus: () => {},
  });

  controllers.syncGridEditorWithState();
  assert.equal(type.value, "hex");
  assert.equal(hexOrientation.value, "pointy-top");
  assert.equal(hexOrientationField.hidden, false);
  assert.equal(visibility.textContent, "Show Grid");
});

test("saving the grid keeps the saved visibility when the editor closes", async () => {
  const visibility = makeElement("button");
  const type = makeElement("select");
  const hexOrientation = makeElement("select");
  const cellSize = makeElement("input");
  const offsetX = makeElement("input");
  const offsetY = makeElement("input");
  const opacity = makeElement("input");
  const lineWidth = makeElement("input");
  const lineStyle = makeElement("select");
  const renders = [];
  let currentVenueGridConfig = {
    grid_type: "hex",
    hex_orientation: "flat-top",
    cell_size: 72,
    offset_x: 12,
    offset_y: -8,
    line_width: 2,
    opacity: 0.6,
    line_style: "dark",
    visible: true,
  };
  let originalGridState = null;
  const fetchedBodies = [];

  const controllers = editorsModule.createEditorControllers({
    fetch: async (url, options = {}) => {
      if (url !== "/api/venues/first-theater/grid" || options.method !== "PUT") {
        throw new Error(`unexpected fetch ${url}`);
      }
      fetchedBodies.push(JSON.parse(String(options.body || "{}")));
      return {
        ok: true,
        json: async () => ({ ok: true, data: { ...currentVenueGridConfig, visible: false } }),
      };
    },
    getGridEditorElements: () => ({
      panel: makeElement("div"),
      type,
      hexOrientation,
      hexOrientationField: makeElement("label"),
      cellSize,
      offsetX,
      offsetY,
      opacity,
      lineWidth,
      lineStyle,
      visibility,
    }),
    getCurrentVenueGridConfig: () => currentVenueGridConfig,
    setCurrentVenueGridConfig: (value) => { currentVenueGridConfig = value; },
    getGridEditorOriginalState: () => originalGridState,
    setGridEditorOriginalState: (value) => { originalGridState = value; },
    setGridEditorDirty: () => {},
    setGridEditorStatus: () => {},
    gridEditorDraftFromUI: () => ({
      grid_type: "hex",
      hex_orientation: "flat-top",
      cell_size: 72,
      offset_x: 12,
      offset_y: -8,
      line_width: 2,
      opacity: 0.6,
      line_style: "dark",
      visible: false,
    }),
    defaultGridConfig: () => ({
      grid_type: "none",
      hex_orientation: "flat-top",
      cell_size: 50,
      offset_x: 0,
      offset_y: 0,
      line_width: 1,
      opacity: 0.45,
      line_style: "neutral",
      visible: true,
    }),
    getCurrentVenueMapBounds: () => ({ x: 0, y: 0, width: 100, height: 100 }),
    getPlayableBounds: () => ({ x: 0, y: 0, width: 100, height: 100 }),
    renderVenueGridLayer: (...args) => renders.push(args),
    syncGridEditorHexFieldVisibility: () => {},
    syncGridEditorVisibilityButton: () => {},
    syncGridEditorWithState: () => {},
  });

  await controllers.saveGridConfig();
  assert.equal(currentVenueGridConfig.visible, false);
  assert.equal(originalGridState.visible, false);

  controllers.hideGridEditor();
  assert.equal(currentVenueGridConfig.visible, false);
  assert.equal(fetchedBodies.length, 1);
  assert.equal(fetchedBodies[0].visible, false);
  assert.ok(renders.length >= 1);
});

test("map asset selection updates the draft preview before save", () => {
  const assetsContainer = makeElement("div");
  const preview = makeElement("img");
  const previewMode = makeElement("span");
  const previewFocus = makeElement("div");
  const previewSafe = makeElement("div");
  const displayMode = makeElement("select");
  const fit = makeElement("select");
  const scale = makeElement("input");
  const cropX = makeElement("input");
  const cropY = makeElement("input");
  const safeMargin = makeElement("input");
  let currentVenueMapState = {
    asset: { content_url: "/api/assets/map-1/content" },
    asset_id: "map-1",
    display_mode: "theater",
    fit: "contain",
    crop_x: 0.25,
    crop_y: 0.75,
    scale: 1.5,
    safe_margin: 32,
  };
  let currentVenueMapAssetID = "map-1";
  const currentVenueMapAssets = [
    { id: "map-1", name: "Map One", asset_type: "map", default_grid_width: 1, default_grid_height: 1, content_url: "/api/assets/map-1/content", thumbnail_url: "/api/assets/map-1/content?variant=thumbnail" },
    { id: "map-2", name: "Map Two", asset_type: "map", default_grid_width: 1, default_grid_height: 1, content_url: "/api/assets/map-2/content", thumbnail_url: "/api/assets/map-2/content?variant=thumbnail" },
  ];

  const originalDocument = global.document;
  global.document = { createElement: makeElement };

  try {
    const controllers = editorsModule.createEditorControllers({
      getMapEditorElements: () => ({
        panel: makeElement("div"),
        preview,
        previewMode,
        previewFocus,
        previewSafe,
        displayMode,
        fit,
        scale,
        cropX,
        cropY,
        safeMargin,
        assets: assetsContainer,
      }),
      getCurrentVenueMapState: () => currentVenueMapState,
      getCurrentVenueMapAssetID: () => currentVenueMapAssetID,
      getCurrentVenueMapAssets: () => currentVenueMapAssets,
      setCurrentVenueMapState: (value) => { currentVenueMapState = value; },
      setCurrentVenueMapAssetID: (value) => { currentVenueMapAssetID = value; },
      setCurrentVenueMapAssets: () => {},
      getCurrentVenueGridConfig: () => ({ grid_type: "square", visible: true }),
      setCurrentVenueGridConfig: () => {},
      defaultGridConfig: () => ({ grid_type: "none", visible: true }),
      getPlayableBounds: () => ({ x: 0, y: 0, width: 100, height: 100 }),
      renderPixiScene: () => {},
      renderVenueGridLayer: () => {},
      setMapEditorStatus: () => {},
      setGridEditorStatus: () => {},
      setMapEditorPosition: () => {},
      setGridEditorPosition: () => {},
      clampMapEditorPosition: (left, top) => ({ left, top }),
      clampGridEditorPosition: (left, top) => ({ left, top }),
      clampCardEditorPosition: (left, top) => ({ left, top }),
      syncGridEditorHexFieldVisibility: () => {},
      syncGridEditorVisibilityButton: () => {},
      syncGridEditorWithState: () => {},
      syncMapEditorPreview: () => {},
      syncMapAssetList: () => {},
      livePreviewMapOnStage: () => {},
      canManageIndexCards: () => true,
      canManageStageTokens: () => true,
      isCardObject: () => true,
      isTokenObject: () => true,
      objectState: () => ({ locked: false }),
      selectObject: () => {},
      setStageStatus: () => {},
      setMovementLine: () => {},
      sendAction: () => true,
      updateLocalObjectModel: () => {},
      updateLocalCardPinModel: () => {},
      updateTokenLocalModel: () => {},
      setLocalPositionOverrideForModel: () => null,
      currentDisplayedPointForModel: () => ({ x: 0, y: 0 }),
      tokenScaleForModel: () => 100,
      tokenSnapModeForModel: () => "free",
      tokenLayerForModel: () => "public",
      tokenPlacementPointForCreate: (point) => point,
      tokenDuplicatePlacementForModel: () => null,
      cardDuplicatePlacementForModel: () => null,
      tokenMoveTargetForModel: () => null,
      cardMoveTargetForModel: () => null,
      refreshVenueMapAssets: async () => {},
      refreshVenueMapState: async () => {},
      refreshVenueGridConfig: async () => {},
      uploadMapAsset: async () => null,
      saveVenueMap: async () => {},
      removeVenueMap: async () => {},
      saveGridConfig: async () => {},
      getCurrentRole: () => "producer",
      getCurrentSelection: () => null,
      getCurrentObjects: () => [],
      getCurrentSnapshot: () => null,
      getStageShell: () => null,
      getCardEditorTarget: () => null,
      hideMapEditor: () => {},
      hideGridEditor: () => {},
      hideCardEditor: () => {},
      hideTokenEditor: () => {},
      closeTokenPicker: () => {},
      openTokenPicker: () => {},
      canActorRevealHideStageObjects: () => true,
      setCurrentSelection: () => {},
      syncSelectedActions: () => {},
      syncTokenEditorWithSelection: () => {},
      getMapEditorOriginalState: () => null,
      setMapEditorOriginalState: () => {},
      setMapEditorDirty: () => {},
      getGridEditorOriginalState: () => null,
      setGridEditorOriginalState: () => {},
      setGridEditorDirty: () => {},
      mapEditorDraftFromState: () => ({}),
      mapEditorPayloadFromUI: () => ({}),
      createIndexCardFromMenu: () => {},
      placeCreatedIndexCard: () => {},
    });

    controllers.syncMapAssetList();
    assert.equal(assetsContainer.children.length, 2);
    const nextButton = assetsContainer.children[1];
    nextButton.click();
    assert.equal(currentVenueMapAssetID, "map-1");
    assert.equal(controllers.mapEditorDraftFromState().assetID, "map-2");
  } finally {
    global.document = originalDocument;
  }
});
