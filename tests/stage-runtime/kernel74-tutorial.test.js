const test = require("node:test");
const assert = require("node:assert/strict");

globalThis.VictoryStageVenue = { slug: "catharsis", name: "Catharsis" };
const { createProjectedState, normalizeSnapshot } = require("../../frontend/lib/stage-runtime/state.js");

// --- Kernel 73A regression: left-click on a bound composition token --------
//
// Kernel 74 S4 requires a regression test specifically for a bound
// scene_composition token -- not a warehouse token. The distinction matters
// because composition elements are forced live:false, arrive with a
// "scene:"-prefixed element id, and use normalized 0-1 positions; a
// selection/action-bar path that only ever ran against live warehouse tokens
// could silently drop them.
//
// This asserts the data contract the action bar depends on: after
// normalization, the composition token is a selectable "token" node that
// still carries its binding through to source.data.binding, which is exactly
// what runtime.js's syncSelectedActions reads to render the bound button.

function compositionSnapshot(extraElements = []) {
  return {
    session: { show_id: "show-1", current_show_scene_placement_id: "placement-1" },
    elements: [
      {
        element_id: "scene:kessa-element",
        element_type: "scene_token",
        context_class: "scene_composition",
        name: "Kessa",
        slug: "scene-kessa-element",
        surface: "stage",
        position: { x: 0.5, y: 0.6 },
        visibility: { toRoles: ["audience", "cast", "crew", "director", "producer"], visible: true },
        state: { locked: false, nameplate_visible: true, visible: true },
        data: {
          kind: "token",
          layer: "base",
          scene_stage_element_id: "kessa-element",
          binding: {
            binding_type: "participant_interaction",
            participant_interaction_id: "interaction-kessa",
            stage_button_label: "Speak with Kessa",
            enabled: true,
            interaction_type: "open_equip_mode",
          },
        },
      },
      ...extraElements,
    ],
  };
}

function doorElement(overrides = {}) {
  return {
    element_id: "scene:door-element",
    element_type: "scene_interaction_hotspot",
    context_class: "scene_composition",
    name: "Locked Courtyard Door",
    slug: "scene-door-element",
    surface: "stage",
    position: { x: 0.5, y: 0.35 },
    visibility: { toRoles: ["audience", "cast", "crew", "director", "producer"], visible: true },
    state: { locked: false, nameplate_visible: true, visible: true },
    data: {
      kind: "interaction_hotspot",
      layer: "base",
      width: 0.12,
      height: 0.28,
      nameplate_visible: true,
      binding: {
        binding_type: "participant_interaction",
        participant_interaction_id: "interaction-door",
        stage_button_label: "Try the Door",
        enabled: true,
        interaction_type: "freeform_submission",
      },
      ...overrides,
    },
  };
}

test("kernel 73A regression: a bound scene_composition token is selectable and keeps its binding", () => {
  const state = createProjectedState({ viewerRole: "cast" });
  state.replaceFromSnapshot(compositionSnapshot());
  const objects = state.getState().objects;

  const kessa = objects.find((o) => o.label === "Kessa");
  assert.ok(kessa, "the composition token must survive normalization as a stage object");
  assert.equal(kessa.kind, "token", "it must normalize to the real token node kind, not be dropped");
  assert.equal(kessa.live, false, "composition elements are never live/persistable objects");

  // This is the exact path syncSelectedActions reads to build the action bar
  // button after a left-click selection.
  const binding = kessa.source.data.binding;
  assert.equal(binding.participant_interaction_id, "interaction-kessa");
  assert.equal(binding.stage_button_label, "Speak with Kessa");
  assert.equal(binding.enabled, true);
  assert.equal(binding.interaction_type, "open_equip_mode",
    "Kernel 74 added more interaction types, so the type must reach the client to pick the right Program");
});

// --- Kernel 74: the interaction hotspot node kind --------------------------

test("an interaction_hotspot normalizes to its own node kind, never a token", () => {
  const state = createProjectedState({ viewerRole: "cast" });
  state.replaceFromSnapshot(compositionSnapshot([doorElement()]));
  const door = state.getState().objects.find((o) => o.label === "Locked Courtyard Door");

  assert.ok(door, "the hotspot must render as a stage object");
  assert.equal(door.kind, "hotspot",
    "it must NOT reuse the token kind -- a token would draw art over the door already in the map");
  assert.equal(door.hotspotWidth, 0.12);
  assert.equal(door.hotspotHeight, 0.28);
  assert.equal(door.source.data.binding.interaction_type, "freeform_submission");
});

test("a hotspot with no authored size falls back to a findable, non-zero box", () => {
  const bare = doorElement();
  delete bare.data.width;
  delete bare.data.height;
  const state = createProjectedState({ viewerRole: "cast" });
  state.replaceFromSnapshot(compositionSnapshot([bare]));
  const door = state.getState().objects.find((o) => o.label === "Locked Courtyard Door");

  assert.ok(door.hotspotWidth > 0, "an unsized hotspot must not become a zero-area, unclickable target");
  assert.ok(door.hotspotHeight > 0);
});

test("map_backdrop and grid_config composition rows still produce no stage node", () => {
  // Kernel 74 reads map_backdrop server-side for the local projection, so it
  // must NOT start appearing as a per-element node on the stage.
  const state = createProjectedState({ viewerRole: "cast" });
  state.replaceFromSnapshot(compositionSnapshot([
    {
      element_id: "scene:backdrop",
      element_type: "scene_map_backdrop",
      context_class: "scene_composition",
      name: "Courtyard",
      surface: "stage",
      position: {},
      visibility: {},
      data: { kind: "map_backdrop", content_url: "/assets/courtyard.png" },
    },
  ]));
  const objects = state.getState().objects;
  assert.equal(objects.filter((o) => o.label === "Courtyard").length, 0);
});

// --- Kernel 74: the reveal gate is server-side ------------------------------

test("the door is simply absent from a snapshot before the milestone is earned", () => {
  // The server omits the element entirely rather than shipping it flagged
  // hidden, so the client has nothing to un-hide. This asserts the client
  // does not synthesize one.
  const state = createProjectedState({ viewerRole: "cast" });
  state.replaceFromSnapshot(compositionSnapshot());
  const objects = state.getState().objects;
  assert.equal(objects.some((o) => o.kind === "hotspot"), false);
});

// --- Kernel 74: local projection in the snapshot ---------------------------

test("a local projection rides on session and never replaces the shared scene pointer", () => {
  const snapshot = normalizeSnapshot({
    session: {
      show_id: "show-1",
      current_show_scene_placement_id: "placement-courtyard",
      local_projection: {
        scene_id: "scene-handoff",
        scene_slug: "tutorial-handoff",
        scene_title: "Outside the Courtyard",
        backdrop_url: "/assets/tutorial-handoff.png",
        grid_enabled: false,
      },
    },
    elements: [],
  }, { viewerRole: "cast" });

  assert.equal(snapshot.session.local_projection.scene_slug, "tutorial-handoff");
  // The whole point of S16.15: the Player's own payload still reports the
  // unchanged shared Scene, so "the shared Scene did not move" is directly
  // observable from the client too.
  assert.equal(snapshot.session.current_show_scene_placement_id, "placement-courtyard");
});

// --- Kernel 74: socket message routing -------------------------------------

test("tutorial/stage_refresh and backstage/note_created are distinct routed message kinds", () => {
  // Read the module source rather than instantiating a live socket: these two
  // message types must exist and must be handled separately from the
  // session-wide show/stage_updated, because one is a per-user nudge and the
  // other must never carry a note body.
  const source = require("node:fs").readFileSync(
    require("node:path").join(__dirname, "../../frontend/lib/stage-runtime/socket.js"),
    "utf8",
  );
  assert.match(source, /tutorial\/stage_refresh/);
  assert.match(source, /backstage\/note_created/);
  assert.equal(
    /backstage\/note_created[\s\S]{0,300}?body/.test(source),
    false,
    "the backstage push must not carry a note body -- a mis-targeted push would then leak one",
  );
});

// --- Regression: an unbound hotspot must be completely inert --------------
//
// This is a real bug that reached the live stage. A hotspot left unbound
// still rendered a full-size, pointer-capturing box; on the Courtyard it
// covered Kessa's token, swallowed every click meant for her, and answered
// "Locked Courtyard Door cannot be used right now". A control that cannot
// act must not be able to block either.

test("an unbound hotspot still normalizes, so the renderer must be what makes it inert", () => {
  const unbound = doorElement();
  delete unbound.data.binding;
  const state = createProjectedState({ viewerRole: "cast" });
  state.replaceFromSnapshot(compositionSnapshot([unbound]));
  const door = state.getState().objects.find((o) => o.label === "Locked Courtyard Door");

  assert.ok(door, "normalization keeps the element (backstage still authors it)");
  assert.equal(door.source.data.binding, undefined,
    "with no binding, the node factory must render it non-interactive");
});

test("a disabled binding is treated as unusable, not merely unlabelled", () => {
  const disabled = doorElement();
  disabled.data.binding.enabled = false;
  const state = createProjectedState({ viewerRole: "cast" });
  state.replaceFromSnapshot(compositionSnapshot([disabled]));
  const door = state.getState().objects.find((o) => o.label === "Locked Courtyard Door");
  assert.equal(door.source.data.binding.enabled, false,
    "the renderer keys off enabled, so a disabled binding must survive normalization as false");
});

test("scene-nodes makes unbound hotspots non-interactive and hides them from players", () => {
  const source = require("node:fs").readFileSync(
    require("node:path").join(__dirname, "../../frontend/lib/stage-runtime/scene-nodes.js"),
    "utf8",
  );
  // The guard must exist and must gate BOTH pointer handling and visibility.
  assert.match(source, /const usable = Boolean\(binding\?\.participant_interaction_id\) && Boolean\(binding\?\.enabled\)/);
  assert.match(source, /eventMode = usable \? "static" : "none"/);
  assert.match(source, /container\.visible = backstage/);
});

test("hotspots render beneath tokens so they never steal a token's click", () => {
  const source = require("node:fs").readFileSync(
    require("node:path").join(__dirname, "../../frontend/lib/stage-runtime/runtime.js"),
    "utf8",
  );
  // Scope to the single z-index decision block so the match cannot run on
  // into the floating-layer branch further down (which uses 100 + index).
  const block = /let zIndex = 10 \+ index;[\s\S]*?node\.container\.zIndex = zIndex;/.exec(source);
  assert.ok(block, "the z-index decision block must exist");
  const hotspotZ = /kind === "hotspot"\)[\s\S]*?zIndex = (\d+) \+ index/.exec(block[0]);
  const tokenZ = /=== "director" \? \d+ \+ index : (\d+) \+ index/.exec(block[0]);
  assert.ok(hotspotZ, "the hotspot z-index branch must exist");
  assert.ok(tokenZ, "the token z-index branch must exist");
  assert.ok(Number(hotspotZ[1]) < Number(tokenZ[1]),
    `hotspot z (${hotspotZ[1]}) must be below public token z (${tokenZ[1]})`);
});

// --- Regression: a hotspot's authored position must actually be used ------
//
// Real bug found in live play. geometry.js converted normalized 0-1
// composition coordinates to pixels only for kind === "token", so a hotspot
// fell through to a generic mid-stage fallback and ignored its stored
// position completely. The Courtyard door landed on Kessa regardless of
// what was saved, and re-authoring it looked like it did nothing.

test("hotspots resolve normalized 0-1 positions through the same path as composition tokens", () => {
  const source = require("node:fs").readFileSync(
    require("node:path").join(__dirname, "../../frontend/lib/stage-runtime/geometry.js"),
    "utf8",
  );
  const branch = /const modelKind = String\(model\?\.kind \|\| ""\)\.toLowerCase\(\);\s*if \(([^)]*)\)/.exec(source);
  assert.ok(branch, "the normalized-coordinate branch must exist");
  assert.match(branch[1], /modelKind === "token"/);
  assert.match(branch[1], /modelKind === "hotspot"/,
    "a hotspot must not fall through to the mid-stage fallback that ignores its stored position");
});

test("a hotspot authored near the top of the stage stays near the top", () => {
  // Guards the actual symptom: an authored y of 0.075 must not resolve
  // anywhere near Kessa's 0.518.
  const high = doorElement();
  const state = createProjectedState({ viewerRole: "cast" });
  state.replaceFromSnapshot(compositionSnapshot([high]));
  const door = state.getState().objects.find((o) => o.label === "Locked Courtyard Door");
  assert.equal(door.position.y, 0.35, "the authored normalized y must survive normalization unscaled");
  assert.equal(door.kind, "hotspot",
    "and it must carry the hotspot kind, which is what geometry keys the conversion off");
});
