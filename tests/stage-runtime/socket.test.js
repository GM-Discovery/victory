const test = require("node:test");
const assert = require("node:assert/strict");

globalThis.VictoryStageVenue = { slug: "catharsis", name: "Catharsis" };
const { createProjectedState } = require("../../frontend/lib/stage-runtime/state.js");
const { createDispatcher } = require("../../frontend/lib/stage-runtime/socket.js");

test("raw socket messages are parsed once and msg.data is normalized", () => {
  let parseCount = 0;
  const state = createProjectedState({ viewerRole: "director" });
  state.replaceFromSnapshot({
    elements: [
      {
        element_id: "token-1",
        element_type: "token",
        name: "Goat",
        position: { x: 10, y: 20, order: 1 },
        visibility: { visible: true, nameplate_visible: true },
        data: {
          type: "token",
          asset_name: "Goat",
          asset_content_url: "/api/assets/asset-1/content?variant=stage",
          token_layer: "public",
        },
      },
    ],
  });
  const dispatcher = createDispatcher({
    state,
    parseMessage(raw) {
      parseCount += 1;
      return JSON.parse(raw);
    },
  });

  const result = dispatcher.dispatch(JSON.stringify({
    type: "action",
    data: {
      type: "set_nameplate_visibility",
      target: { element_id: "token-1" },
      payload: { nameplateVisible: false },
    },
  }));

  assert.equal(parseCount, 1);
  assert.equal(result.kind, "action");
  assert.equal(state.getState().objectsByKey.get("token-1").visibility.nameplate_visible, false);
  assert.equal(state.getState().objectsByKey.get("token-1").visibility.nameplateVisible, false);
});

test("snapshot and focus ping route to dedicated handlers", () => {
  const state = createProjectedState({ viewerRole: "director" });
  let focusHits = 0;
  let snapshotHits = 0;
  const dispatcher = createDispatcher({
    state,
    onSnapshot() {
      snapshotHits += 1;
    },
    onFocusPing() {
      focusHits += 1;
    },
  });

  const snapshotResult = dispatcher.dispatch({
    type: "snapshot",
    data: { elements: [] },
  });
  const focusResult = dispatcher.dispatch({
    type: "venue/focus_ping",
    data: { target: "token-1" },
  });

  assert.equal(snapshotResult.kind, "snapshot");
  assert.equal(focusResult.kind, "focus_ping");
  assert.equal(snapshotHits, 1);
  assert.equal(focusHits, 1);
});

test("malformed messages are ignored safely", () => {
  const dispatcher = createDispatcher({});
  const result = dispatcher.dispatch("{not-json");
  assert.equal(result.kind, "malformed");
});

// Kernel 86: the dispatcher must recognize all four Stage Effect message
// types the server can now send over the same venue websocket, and pass
// each payload through untouched (audience/authority filtering already
// happened server-side -- backend/internal/rollaudience -- before any of
// these are ever sent).
test("stage effect messages are dispatched with the expected kind and payload", () => {
  const dispatcher = createDispatcher({});

  const created = dispatcher.dispatch({
    type: "stage_effect",
    data: { effect_id: "fx_1", type: "dice_roll" },
  });
  assert.equal(created.kind, "stage_effect");
  assert.equal(created.effect.effect_id, "fx_1");

  const pinned = dispatcher.dispatch({
    type: "stage_effect_pinned",
    data: { effect_id: "fx_1", pinned: true },
  });
  assert.equal(pinned.kind, "stage_effect_pinned");
  assert.equal(pinned.effect.pinned, true);

  const dismissed = dispatcher.dispatch({
    type: "stage_effect_dismissed",
    effect_id: "fx_1",
  });
  assert.equal(dismissed.kind, "stage_effect_dismissed");
  assert.equal(dismissed.effectId, "fx_1");

  const hydrated = dispatcher.dispatch({
    type: "stage_effects/pinned",
    data: [{ effect_id: "fx_2" }],
  });
  assert.equal(hydrated.kind, "stage_effects_pinned");
  assert.equal(hydrated.effects.length, 1);
  assert.equal(hydrated.effects[0].effect_id, "fx_2");
});

test("bind does not duplicate message handlers on remount", () => {
  const state = createProjectedState({ viewerRole: "director" });
  const dispatcher = createDispatcher({ state });
  const listeners = new Map();
  const socket = {
    addEventListener(type, handler) {
      listeners.set(type, handler);
    },
    removeEventListener(type, handler) {
      if (listeners.get(type) === handler) {
        listeners.delete(type);
      }
    },
  };

  const cleanup1 = dispatcher.bind(socket);
  const cleanup2 = dispatcher.bind(socket);
  assert.equal(listeners.size, 1);
  cleanup1();
  assert.equal(listeners.size, 0);
  cleanup2();
  assert.equal(listeners.size, 0);
});
