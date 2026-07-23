const test = require("node:test");
const assert = require("node:assert/strict");

globalThis.VictoryStageVenue = { slug: "catharsis", name: "Catharsis" };
const { createProjectedState } = require("../../frontend/lib/stage-runtime/state.js");

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

test("hide and reveal element actions update projected visibility", () => {
  const state = createProjectedState({ viewerRole: "director" });
  state.replaceFromSnapshot(tokenSnapshot());
  state.applyEvent({ type: "act/hide_element", target: { element_id: "token-1" }, sequence: 2 });
  assert.equal(state.getState().objects[0].visibility.visible, false);
  state.applyEvent({ type: "act/reveal_element", target: { element_id: "token-1" }, sequence: 3 });
  assert.equal(state.getState().objects[0].visibility.visible, true);
});

// Kernel 73A: a scene_composition-context_class element (synthesized by
// backend/internal/world/snapshot.go's loadSceneCompositionAsPlacedElements
// from scene_stage_elements, never a warehouse-asset element) must map onto
// the existing "token"/"card" node kinds this pipeline already renders,
// carry its bound participant interaction through to model.source.data.binding
// untouched, and never be treated as a persistable live stage object (it has
// no elements-table row for update/token or act/place_element to target).
test("scene composition elements map onto token/card kinds and stay non-live", () => {
  const state = createProjectedState({ viewerRole: "director" });
  state.replaceFromSnapshot({
    elements: [
      {
        element_id: "scene:kessa-token-id",
        element_type: "scene_token",
        context_class: "scene_composition",
        name: "Kessa",
        slug: "scene-kessa-token-id",
        position: { x: 0.4, y: 0.6 },
        visibility: { visible: true, nameplate_visible: true },
        data: {
          scene_stage_element_id: "kessa-token-id",
          kind: "token",
          layer: "base",
          binding: {
            binding_type: "participant_interaction",
            participant_interaction_id: "interaction-1",
            stage_button_label: "Talk to Kessa",
            enabled: true,
          },
        },
      },
      {
        element_id: "scene:note-card-id",
        element_type: "scene_index_card",
        context_class: "scene_composition",
        name: "A Note",
        slug: "scene-note-card-id",
        position: { x: 0.2, y: 0.2 },
        visibility: { visible: true, nameplate_visible: true },
        data: { scene_stage_element_id: "note-card-id", kind: "index_card", layer: "base" },
      },
      {
        element_id: "scene:backdrop-id",
        element_type: "scene_map_backdrop",
        context_class: "scene_composition",
        name: "Backdrop",
        slug: "scene-backdrop-id",
        position: {},
        visibility: { visible: true, nameplate_visible: true },
        data: { scene_stage_element_id: "backdrop-id", kind: "map_backdrop", layer: "base" },
      },
    ],
  });

  const objects = state.getState().objects;
  // map_backdrop has no generic per-element node yet -- must be dropped,
  // not mis-rendered as a token or card.
  assert.equal(objects.length, 2);

  const kessa = objects.find((o) => o.label === "Kessa");
  assert.ok(kessa, "expected the Kessa token to be present");
  assert.equal(kessa.kind, "token");
  assert.equal(kessa.live, false);
  assert.equal(kessa.source.data.binding.participant_interaction_id, "interaction-1");
  assert.equal(kessa.source.data.binding.stage_button_label, "Talk to Kessa");
  assert.equal(kessa.source.data.binding.enabled, true);

  const note = objects.find((o) => o.label === "A Note");
  assert.ok(note, "expected the index card to be present");
  assert.equal(note.kind, "card");
  assert.equal(note.live, false);
  assert.equal(note.source.data.binding, undefined);
});
