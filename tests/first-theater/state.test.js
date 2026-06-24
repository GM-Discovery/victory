const test = require("node:test");
const assert = require("node:assert/strict");

const { createProjectedState } = require("../../frontend/venues/first-theater/runtime/state.js");

function tokenSnapshot(overrides = {}) {
  return {
    elements: [
      {
        element_id: "token-1",
        element_type: "token",
        name: "Goat",
        slug: "goat",
        position: { x: 10, y: 20, order: 1 },
        visibility: { visible: true, nameplate_visible: true },
        data: {
          type: "token",
          asset_id: "asset-1",
          asset_name: "Goat",
          asset_shape: "circle",
          asset_content_url: "/api/assets/asset-1/content?variant=stage",
          default_grid_width: 1,
          default_grid_height: 1,
          token_layer: "public",
          scale: 100,
          ...overrides.data,
        },
      },
    ],
  };
}

test("snapshot replacement normalizes projected objects", () => {
  const state = createProjectedState({ viewerRole: "director" });
  state.replaceFromSnapshot(tokenSnapshot());
  const current = state.getState();
  assert.equal(current.objects.length, 1);
  assert.equal(current.objects[0].label, "Goat");
  assert.equal(current.objects[0].assetContentURL.includes("/api/assets/asset-1"), true);
});

test("latest token update wins and stale updates do not snap backward", () => {
  const state = createProjectedState({ viewerRole: "director" });
  state.replaceFromSnapshot(tokenSnapshot());
  state.applyEvent({ type: "update/token", target: { element_id: "token-1" }, payload: { x: 40, y: 50 }, sequence: 2 });
  state.applyEvent({ type: "update/token", target: { element_id: "token-1" }, payload: { x: 15, y: 25 }, sequence: 1 });
  const current = state.getState().objects[0];
  assert.equal(current.position.x, 40);
  assert.equal(current.position.y, 50);
});

test("removal wins over earlier create and prevents resurrection", () => {
  const state = createProjectedState({ viewerRole: "director" });
  state.applyEvent({ type: "create/token", target: { element_id: "token-1" }, payload: { asset_name: "Goat" }, sequence: 1 });
  state.applyEvent({ type: "remove_element", target: { element_id: "token-1" }, sequence: 2 });
  state.applyEvent({ type: "update/token", target: { element_id: "token-1" }, payload: { x: 99, y: 99 }, sequence: 3 });
  assert.equal(state.getState().objects.length, 0);
  assert.equal(state.getState().objectsByKey.get("token-1").deleted, true);
});

test("director-layer tokens are hidden from audience projections", () => {
  const state = createProjectedState({ viewerRole: "audience" });
  state.replaceFromSnapshot(tokenSnapshot({
    data: { token_layer: "director" },
  }));
  assert.equal(state.getState().objects.length, 0);
});

test("missing asset metadata falls back to construction art", () => {
  const state = createProjectedState({ viewerRole: "director" });
  state.applyEvent({
    type: "create/token",
    target: { element_id: "token-2" },
    payload: { asset_name: "Fallback Goat" },
    sequence: 1,
  });
  const token = state.getState().objectsByKey.get("token-2");
  assert.equal(token.assetContentURL, "/assets/construction.png");
});

test("subscriptions fire only for relevant changes", () => {
  const state = createProjectedState({ viewerRole: "director" });
  let tokenHits = 0;
  let snapshotHits = 0;
  const stopToken = state.subscribe("token-1", () => { tokenHits += 1; });
  const stopSnapshot = state.subscribe("snapshot", () => { snapshotHits += 1; });
  state.replaceFromSnapshot(tokenSnapshot());
  state.applyEvent({ type: "update/token", target: { element_id: "token-1" }, payload: { x: 22, y: 33 }, sequence: 2 });
  state.applyEvent({ type: "update/index_card", target: { element_id: "card-1" }, payload: { front_text: "No-op" }, sequence: 1 });
  stopToken();
  stopSnapshot();
  assert.equal(snapshotHits >= 1, true);
  assert.equal(tokenHits >= 1, true);
});
