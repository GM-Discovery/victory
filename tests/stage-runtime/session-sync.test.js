const test = require("node:test");
const assert = require("node:assert/strict");

globalThis.VictoryStageVenue = { slug: "catharsis", name: "Catharsis" };
const sessionSyncModule = require("../../frontend/lib/stage-runtime/session-sync.js");

test("session sync handles join, refresh, and staged index-card placement", async () => {
  const calls = [];
  let currentIdentity = null;
  let currentRole = "audience";
  let currentSessionId = "";
  let currentActorId = "";
  let currentObjects = [];
  let currentSelection = null;
  let currentSnapshot = null;
  let stagePlacementCandidate = { x: 31, y: 47 };
  let stagePlacementScreenCandidate = { x: 31, y: 47 };
  let pendingStageCardPlacement = null;
  let renderCount = 0;

  const deps = {
    fetch: async (url, options = {}) => {
      calls.push(["fetch", url, options.method || "GET"]);
      if (url === "/api/session/me") {
        return {
          ok: true,
          json: async () => ({ data: { handle: "straturli", display_name: "Straturli" } }),
        };
      }
      if (url === "/api/session/catharsis/join") {
        return {
          ok: true,
          json: async () => ({ ok: true, data: { role: "producer", session_id: "session-1", actor_id: "actor-1" } }),
        };
      }
      if (url === "/api/world/catharsis") {
        return {
          ok: true,
          json: async () => ({
            ok: true,
            data: {
              elements: [
                { slug: "first-fire", element_id: "fire-1", name: "Fire" },
                {
                  element_type: "index_card",
                  element_id: "card-1",
                  slug: "card-one",
                  name: "Card One",
                  data: { front_text: "Card One", back_text: "Back", color: "#d9c7a6" },
                  position: { order: 3 },
                  surface: "stage",
                },
              ],
              actions: [
                {
                  id: "roll-1",
                  moment_id: 7,
                  type: "roll/dice",
                  actor_id: "actor-1",
                  payload: {
                    request_id: "req-1",
                    expression: "d20",
                    dice: [{ index: 0, chain: [20], subtotal: 20 }],
                    total: 20,
                    modifier: 0,
                    explosion_count: 0,
                    roll_version: 1,
                    visibility_mode: "public",
                  },
                },
              ],
              overlay: null,
              showing: { status: "open" },
            },
          }),
        };
      }
      throw new Error(`unexpected fetch ${url}`);
    },
    getCurrentIdentity: () => currentIdentity,
    setCurrentIdentity: (value) => { currentIdentity = value; },
    getCurrentRole: () => currentRole,
    setCurrentRole: (value) => { currentRole = value; },
    getCurrentSessionId: () => currentSessionId,
    setCurrentSessionId: (value) => { currentSessionId = value; },
    getCurrentActorId: () => currentActorId,
    setCurrentActorId: (value) => { currentActorId = value; },
    getCurrentSnapshot: () => currentSnapshot,
    getCurrentObjects: () => currentObjects,
    setCurrentObjects: (value) => { currentObjects = value; },
    getCurrentSelection: () => currentSelection,
    setCurrentSelection: (value) => { currentSelection = value; },
    replaceProjectedState: (snapshot) => { currentSnapshot = snapshot; },
    buildObjects: (snapshot) => (snapshot?.elements || []).map((element, index) => ({
      key: `obj-${index}`,
      live: true,
      kind: element.element_type === "index_card" ? "card" : "fire",
      elementId: element.element_id,
      elementSlug: element.slug,
      label: element.name,
      position: element.position || {},
      source: element,
    })),
    applyVenueFocusPing: () => {},
    updateStatusSummary: () => {},
    setStageStatus: (text) => calls.push(["stage", text]),
    setMovementLine: (text) => calls.push(["move", text]),
    setSnapshotSummary: (text) => calls.push(["snapshot", text]),
    setLiveFeedLine: (text) => calls.push(["live", text]),
    setSelectionLine: (text) => calls.push(["selection", text]),
    updateChatPresentation: () => {},
    renderPixiScene: () => { renderCount += 1; calls.push(["render"]); },
    canManageIndexCards: () => true,
    canManageStageTokens: () => true,
    canActorRevealHideStageObjects: () => true,
    cardFaceForModel: () => "front",
    cardDisplayMode: () => "overlay",
    tokenScaleForModel: () => 100,
    tokenSnapModeForModel: () => "free",
    tokenLayerForModel: () => "public",
    closeContextMenu: () => calls.push(["close"]),
    selectObject: (model, reason) => calls.push(["select", model?.label || null, reason]),
    syncSelectedActions: () => calls.push(["sync-actions"]),
    syncCardEditorWithSelection: () => calls.push(["sync-card"]),
    syncTokenEditorWithSelection: () => calls.push(["sync-token"]),
    refreshVenueGridConfig: async () => {},
    refreshVenueMapState: async () => {},
    refreshVenueMapAssets: async () => {},
    loadPixiLibrary: async () => {},
    initializePixi: async () => {},
    initializeShellChrome: () => {},
    updateShellMetaPresentation: () => {},
    updateShellTargetPresentation: () => {},
    updateHeaderPresentation: () => {},
    setRendererFallback: () => {},
    appendSystemChatNotice: () => {},
    chatClosedMessage: () => "closed",
    getCurrentVenueMapState: () => null,
    getCurrentVenueMapAssetID: () => "",
    getCurrentVenueGridConfig: () => null,
    roleLabel: (role) => role,
    getStageStatus: () => "",
    getStagePlacementCandidate: () => stagePlacementCandidate,
    setStagePlacementCandidate: (value) => { stagePlacementCandidate = value; },
    getStagePlacementScreenCandidate: () => stagePlacementScreenCandidate,
    setStagePlacementScreenCandidate: (value) => { stagePlacementScreenCandidate = value; },
    getPendingStageCardPlacement: () => pendingStageCardPlacement,
    setPendingStageCardPlacement: (value) => { pendingStageCardPlacement = value; },
    setSmokeLine: (text) => calls.push(["smoke", text]),
    normalizeOverlayPoint: (point) => ({ screen_x: point.x / 100, screen_y: point.y / 100 }),
    setRecentPlacementMarker: (point, label) => calls.push(["marker", point, label]),
    smokeMode: true,
    setLocalPositionOverrideForModel: () => null,
    syncDiceTrayFromSnapshot: (snapshot) => calls.push(["dice-snapshot", snapshot?.actions?.length || 0]),
    handleDiceTrayAction: (action) => calls.push(["dice-action", action?.type || ""]),
    handleDiceTrayError: (errorText) => calls.push(["dice-error", errorText]),
    rejectPendingDiceTrayRolls: (reason) => calls.push(["dice-reject", reason]),
    sendAction: (type, payload) => {
      calls.push(["send", type, payload]);
      return true;
    },
  };

  const sessionSync = sessionSyncModule.createSessionSync(deps);

  sessionSync.createIndexCardFromMenu();
  assert.deepEqual(calls.find((entry) => entry[0] === "send"), ["send", "create/index_card", {
    front_text: "New Index Card",
    back_text: "",
    color: "#d9c7a6",
  }]);
  assert.equal(stagePlacementCandidate, null);
  assert.equal(stagePlacementScreenCandidate, null);
  assert.deepEqual(pendingStageCardPlacement, { x: 31, y: 47 });

  await sessionSync.joinCaveOnce();
  assert.equal(currentRole, "producer");
  assert.equal(currentSessionId, "session-1");
  assert.equal(currentActorId, "actor-1");

  await sessionSync.refreshWorld();
  assert.equal(currentObjects.length, 2);
  assert.equal(currentSnapshot.elements.length, 2);
  assert.ok(calls.some((entry) => entry[0] === "dice-snapshot" && entry[1] === 1));
  assert.ok(renderCount > 0);

  const rendersAfterRefresh = renderCount;
  await sessionSync.handleSocketMessage({
    kind: "action",
    action: {
      id: "roll-2",
      type: "roll/dice",
      payload: { request_id: "req-2", expression: "d20", total: 7 },
    },
  });
  assert.ok(calls.some((entry) => entry[0] === "dice-action" && entry[1] === "roll/dice"));

  await sessionSync.handleSocketMessage({
    kind: "action",
    action: {
      type: "create/token",
      target: { element_id: "token-1", element_slug: "token-one" },
    },
  });
  assert.ok(renderCount > rendersAfterRefresh);

  sessionSync.placeCreatedIndexCard({ target: { element_id: "card-1" } });
  assert.ok(calls.some((entry) => entry[0] === "select" && entry[1] === "Card One"));
  assert.ok(calls.some((entry) => entry[0] === "send" && entry[1] === "update/index_card"));
  assert.ok(calls.some((entry) => entry[0] === "send" && entry[1] === "act/place_element"));
  assert.equal(pendingStageCardPlacement, null);
});

test("operator identity is promoted to producer in the Cave join flow", async () => {
  let currentIdentity = null;
  let currentRole = "audience";
  let currentSessionId = "";
  let currentActorId = "";

  const deps = {
    fetch: async (url, options = {}) => {
      if (url === "/api/session/me") {
        return {
          ok: true,
          json: async () => ({ data: { handle: "straturli", display_name: "Straturli", is_operator: true } }),
        };
      }
      if (url === "/api/session/catharsis/join") {
        return {
          ok: true,
          json: async () => ({ ok: true, data: { role: "audience", session_id: "session-2", actor_id: "actor-2" } }),
        };
      }
      throw new Error(`unexpected fetch ${url}`);
    },
    getCurrentIdentity: () => currentIdentity,
    setCurrentIdentity: (value) => { currentIdentity = value; },
    getCurrentRole: () => currentRole,
    setCurrentRole: (value) => { currentRole = value; },
    getCurrentSessionId: () => currentSessionId,
    setCurrentSessionId: (value) => { currentSessionId = value; },
    getCurrentActorId: () => currentActorId,
    setCurrentActorId: (value) => { currentActorId = value; },
    getCurrentSnapshot: () => null,
    getCurrentObjects: () => [],
    setCurrentObjects: () => {},
    getCurrentSelection: () => null,
    setCurrentSelection: () => {},
    replaceProjectedState: () => {},
    buildObjects: () => [],
    applyVenueFocusPing: () => {},
    updateStatusSummary: () => {},
    setStageStatus: () => {},
    setMovementLine: () => {},
    setSnapshotSummary: () => {},
    setLiveFeedLine: () => {},
    setSelectionLine: () => {},
    updateChatPresentation: () => {},
    renderPixiScene: () => {},
    canManageIndexCards: () => true,
    canManageStageTokens: () => true,
    canActorRevealHideStageObjects: () => true,
    cardFaceForModel: () => "front",
    cardDisplayMode: () => "overlay",
    tokenScaleForModel: () => 100,
    tokenSnapModeForModel: () => "free",
    tokenLayerForModel: () => "public",
    closeContextMenu: () => {},
    selectObject: () => {},
    syncSelectedActions: () => {},
    syncCardEditorWithSelection: () => {},
    syncTokenEditorWithSelection: () => {},
    refreshVenueGridConfig: async () => {},
    refreshVenueMapState: async () => {},
    refreshVenueMapAssets: async () => {},
    loadPixiLibrary: async () => {},
    initializePixi: async () => {},
    initializeShellChrome: () => {},
    updateShellMetaPresentation: () => {},
    updateShellTargetPresentation: () => {},
    updateHeaderPresentation: () => {},
    setRendererFallback: () => {},
    appendSystemChatNotice: () => {},
    chatClosedMessage: () => "closed",
    getCurrentVenueMapState: () => null,
    getCurrentVenueMapAssetID: () => "",
    getCurrentVenueGridConfig: () => null,
    roleLabel: (role) => role,
    getStageStatus: () => "",
    getStagePlacementCandidate: () => null,
    setStagePlacementCandidate: () => {},
    getStagePlacementScreenCandidate: () => null,
    setStagePlacementScreenCandidate: () => {},
    getPendingStageCardPlacement: () => null,
    setPendingStageCardPlacement: () => {},
    setSmokeLine: () => {},
    normalizeOverlayPoint: (point) => point,
    setRecentPlacementMarker: () => {},
    smokeMode: false,
    setLocalPositionOverrideForModel: () => null,
    sendAction: () => true,
  };

  const sessionSync = sessionSyncModule.createSessionSync(deps);
  await sessionSync.joinCaveOnce();

  assert.equal(currentRole, "producer");
  assert.equal(currentSessionId, "session-2");
  assert.equal(currentActorId, "actor-2");
});
