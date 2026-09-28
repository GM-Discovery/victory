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

// Kernel 101 (101-24) follow-up regression: the menu enabled "Add Token" for
// Cast and canActCreateToken permitted it, but openTokenPicker and
// refreshWarehouseTokenAssets both still demanded canManageStageTokens --
// so Cast saw an enabled action that could never be completed. These three
// tests pin the whole create path open for Cast and every edit path shut.
// renderTokenPickerList builds real DOM nodes; these tests only care about
// authority and which endpoint is reached, so a minimal stub is enough.
function withDocumentStub(run) {
  const originalDocument = global.document;
  global.document = {
    createElement: () => ({
      className: "",
      textContent: "",
      innerHTML: "",
      type: "",
      classList: { add() {}, contains() { return false; } },
      addEventListener() {},
      appendChild() {},
    }),
  };
  // Always returns a promise and restores only after it settles, so an
  // async refresh doesn't lose the stub mid-flight.
  return Promise.resolve()
    .then(run)
    .finally(() => { global.document = originalDocument; });
}

function castPickerDeps(overrides = {}) {
  const state = { search: "", filterShape: "all", mode: "create" };
  return {
    state,
    // Exactly what runtime.js passes for a Cast viewer.
    canManageStageTokens: () => false,
    canCreateStageObjects: () => true,
    tokenPlacementPointForCreate: (point) => point,
    tokenScaleForModel: () => 100,
    tokenSnapModeForModel: () => "grid",
    tokenLayerForModel: () => "public",
    renderPixiScene: () => {},
    setStageStatus: () => {},
    setMovementLine: () => {},
    formatByteSize: (n) => `${n} B`,
    escapeHtml: (v) => String(v),
    sendAction: () => true,
    refreshVenueMapAssets: () => Promise.resolve(),
    currentGridConfig: () => ({ grid_type: "square" }),
    lastStagePoint: () => ({ x: 20, y: 20 }),
    stagePlacementCandidate: () => ({ x: 30, y: 30 }),
    stagePlacementScreenCandidate: () => ({ x: 30, y: 30 }),
    defaultPlacementPoint: () => ({ x: 0, y: 0 }),
    setPlacementState: () => {},
    openEditor: () => {},
    closeEditor: () => {},
    setEditorStatus: () => {},
    getEditorState: () => ({}),
    getTokenAssets: () => [],
    setTokenAssets: () => {},
    getTokenPickerElements: () => ({
      list: { innerHTML: "", appendChild() {} },
      preview: { src: "" },
      previewBadge: { textContent: "" },
      search: { value: "" },
      shape: { value: "all" },
      status: { textContent: "" },
      apply: { textContent: "" },
      setPosition: () => {},
      panel: { hidden: true },
    }),
    refreshWarehouseTokenAssets: () => Promise.resolve(),
    clampNumber: (value) => Number(value),
    objectState: () => ({}),
    isTokenObject: () => true,
    currentObjects: () => [],
    updateTokenLocalModel: () => {},
    ...overrides,
  };
}

test("Cast can open the create-mode token picker", () => {
  const elements = {
    list: { innerHTML: "", appendChild() {} },
    preview: { src: "" },
    previewBadge: { textContent: "" },
    search: { value: "" },
    shape: { value: "all" },
    status: { textContent: "" },
    apply: { textContent: "" },
    setPosition: () => {},
    panel: { hidden: true },
  };
  const statuses = [];
  const deps = castPickerDeps({
    getTokenPickerElements: () => elements,
    setStageStatus: (message) => statuses.push(message),
  });
  const picker = tokenUiModule.createTokenUi(deps);

  picker.openTokenPicker("create", null);

  assert.equal(elements.panel.hidden, false, "create-mode picker must open for Cast");
  assert.equal(deps.state.mode, "create");
  assert.equal(elements.apply.textContent, "Add Token");
  assert.equal(statuses.length, 0, "Cast must not be told they cannot place tokens");
});

test("Cast create-mode picker loads assets from the Cast-readable stage endpoint", async () => {
  const requested = [];
  const originalFetch = global.fetch;
  global.fetch = async (url) => {
    requested.push(String(url));
    return {
      ok: true,
      status: 200,
      json: async () => ({ ok: true, data: [{ id: "asset-1", name: "Goat", shape: "circle" }] }),
    };
  };
  let stored = null;
  const deps = castPickerDeps({
    setTokenAssets: (assets) => { stored = assets; },
    getTokenAssets: () => stored || [],
  });

  try {
    const picker = tokenUiModule.createTokenUi(deps);
    await withDocumentStub(() => picker.refreshWarehouseTokenAssets());
  } finally {
    global.fetch = originalFetch;
  }

  assert.equal(requested.length, 1, "Cast must actually reach an asset endpoint");
  assert.ok(
    requested[0].startsWith("/api/stage/token-assets"),
    `expected the Cast-readable stage endpoint, got ${requested[0]}`,
  );
  assert.equal(stored.length, 1, "the picker must end up with a usable asset list");
  assert.equal(stored[0].id, "asset-1");
});

test("Cast is denied replace mode, which edits an existing token's asset", () => {
  const elements = {
    list: { innerHTML: "", appendChild() {} },
    preview: { src: "" },
    previewBadge: { textContent: "" },
    search: { value: "" },
    shape: { value: "all" },
    status: { textContent: "" },
    apply: { textContent: "" },
    setPosition: () => {},
    panel: { hidden: true },
  };
  const statuses = [];
  const requested = [];
  const originalFetch = global.fetch;
  global.fetch = async (url) => {
    requested.push(String(url));
    return { ok: true, status: 200, json: async () => ({ ok: true, data: [] }) };
  };
  const deps = castPickerDeps({
    getTokenPickerElements: () => elements,
    setStageStatus: (message) => statuses.push(message),
  });

  try {
    const picker = tokenUiModule.createTokenUi(deps);
    picker.openTokenPicker("replace", { key: "token-1", elementId: "element-1" });

    assert.equal(elements.panel.hidden, true, "replace-mode picker must stay shut for Cast");
    assert.equal(statuses.length, 1);
    assert.match(statuses[0], /producers and directors/);

    // And the asset list must stay empty in replace mode even if something
    // calls the refresh directly -- the two gates agree by construction.
    deps.state.mode = "replace";
    return withDocumentStub(() => picker.refreshWarehouseTokenAssets()).then(() => {
      assert.equal(requested.length, 0, "replace mode must not fetch assets for Cast");
    });
  } finally {
    global.fetch = originalFetch;
  }
});
