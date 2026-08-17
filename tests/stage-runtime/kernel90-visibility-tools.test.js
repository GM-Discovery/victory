// Kernel 90 unit tests for the pure part of the canonical visibility surface:
// the context-menu shape logic.js produces, and the identity helpers the menu
// and the tool module both key off.
//
// The scope panel is not unit-tested here on purpose -- it is fetch-driven DOM,
// and the honest proof for it is the browser run
// (scripts/smoke/kernel90-visibility-browser.js), not a mock of every endpoint
// it calls. Same reasoning as Kernel 89's test file.
//
// What IS worth pinning here: that the menu never offers a control the object
// cannot support, never exposes identity a Player should not have, and never
// degenerates into the flat wall §53 marks as a PARTIAL.
const test = require("node:test");
const assert = require("node:assert/strict");

globalThis.VictoryStageVenue = { slug: "catharsis", name: "Catharsis" };
const logic = require("../../frontend/lib/stage-runtime/logic.js");

const DIRECTOR_CONTEXT = {
  role: "director",
  canManageIndexCards: true,
  canManageStageTokens: true,
  canActorRevealHideStageObjects: true,
  hasSelection: false,
};

const PLAYER_CONTEXT = {
  role: "cast",
  canManageIndexCards: false,
  canManageStageTokens: false,
  canActorRevealHideStageObjects: false,
  hasSelection: false,
};

// A live warehouse token, as the server sends it to a BACKSTAGE viewer: the
// canonical identity keys are present only for those viewers.
function liveTokenModel(overrides = {}) {
  return {
    kind: "token",
    live: true,
    key: "token-1",
    elementId: "token-1",
    elementSlug: "russel",
    label: "Russel",
    position: { x: 10, y: 20, order: 1, frame: "center" },
    source: { data: { asset_id: "a1", snap_mode: "grid", token_layer: "public" }, visibility: {} },
    state: {
      visible: true,
      locked: false,
      nameplateVisible: true,
      stage_object_kind: "venue_layout_element",
      stage_object_id: "token-1",
      hidden_backstage_only: false,
      ...overrides,
    },
  };
}

// A Kernel 73A Scene-authored composition element. `live` is false, which is
// the branch that had NO visibility control at all before Kernel 90.
function sceneElementModel(overrides = {}) {
  return {
    kind: "token",
    live: false,
    key: "scene:abc",
    elementId: "scene:abc",
    label: "Training Wall",
    source: { data: {}, state: {} },
    state: {
      visible: true,
      locked: false,
      stage_object_kind: "scene_stage_element",
      stage_object_id: "abc",
      hidden_backstage_only: false,
      ...overrides,
    },
  };
}

function withBinding(model, binding) {
  model.source.data = { ...(model.source.data || {}), binding };
  return model;
}

function familyOf(actions, name) {
  return actions.find((a) => a.action === name);
}

function submenuActions(family) {
  return (family?.submenu || []).map((c) => c.action);
}

test("canonical identity is read from the snapshot, never derived from labels or keys", () => {
  // §22/§54: identity must be kind+id from the server. A model with no
  // canonical keys yields NO ref, even though it has a key, an elementId and a
  // label that a lazier implementation could have used.
  assert.deepEqual(
    logic.canonicalStageObjectRef(liveTokenModel()),
    { kind: "venue_layout_element", id: "token-1" }
  );
  assert.deepEqual(
    logic.canonicalStageObjectRef(sceneElementModel()),
    { kind: "scene_stage_element", id: "abc" }
  );

  const noIdentity = liveTokenModel();
  delete noIdentity.state.stage_object_kind;
  delete noIdentity.state.stage_object_id;
  assert.equal(logic.canonicalStageObjectRef(noIdentity), null,
    "a model without server-sent identity must not get a synthesised one");

  // Half an identity is no identity.
  const halfIdentity = liveTokenModel({ stage_object_id: "" });
  assert.equal(logic.canonicalStageObjectRef(halfIdentity), null);
});

test("a Player is offered no visibility or interaction controls at all", () => {
  // §35: a Player cannot reveal, hide, or change scope. The server refuses
  // independently (that is the boundary), but the menu must not offer it
  // either -- and a Player's snapshot carries no canonical identity anyway,
  // so both guards are exercised here.
  const playerModel = liveTokenModel();
  delete playerModel.state.stage_object_kind;
  delete playerModel.state.stage_object_id;

  const actions = logic.resolveStageObjectActions(playerModel, PLAYER_CONTEXT);
  assert.equal(familyOf(actions, "k90-visibility"), undefined);
  assert.equal(familyOf(actions, "k90-interaction"), undefined);

  // Even if a forged client somehow held identity, the authority gate refuses.
  const forged = logic.resolveStageObjectActions(liveTokenModel(), PLAYER_CONTEXT);
  assert.equal(familyOf(forged, "k90-visibility"), undefined,
    "a Player holding canonical identity must still be offered no visibility control");
});

test("the Director gets Visibility as ONE nested family, not a row of buttons", () => {
  // §11/§12, and the §53 PARTIAL trigger about the menu becoming a flat list.
  const actions = logic.resolveStageObjectActions(liveTokenModel(), DIRECTOR_CONTEXT);
  const family = familyOf(actions, "k90-visibility");
  assert.ok(family, "the Director should get a Visibility family");
  assert.deepEqual(submenuActions(family), [
    "k90-visibility-visible",
    "k90-visibility-hidden",
    "k90-visibility-scope",
  ]);

  // The children must not ALSO appear top-level, or nesting bought nothing.
  const topLevel = new Set(actions.map((a) => a.action));
  for (const child of submenuActions(family)) {
    assert.ok(!topLevel.has(child), `${child} should live only inside the family`);
  }
});

test("the flat pre-Kernel-90 Hide Audience toggle is gone", () => {
  // The old control could only say hidden-or-not against a fixed audience
  // layer. Leaving it beside the new family would be the fourth mechanism §0
  // forbids -- two controls writing visibility with different vocabularies.
  const actions = logic.resolveStageObjectActions(liveTokenModel(), DIRECTOR_CONTEXT);
  const labels = actions.map((a) => a.label);
  assert.ok(!labels.includes("Hide Audience"), "the legacy Hide Audience entry should be gone");
  assert.ok(!labels.includes("Show Audience"), "the legacy Show Audience entry should be gone");
  assert.ok(!actions.some((a) => a.action === "hide" || a.action === "show"),
    "the legacy hide/show menu actions should no longer be produced");
});

test("Scene-authored composition elements get visibility controls too", () => {
  // The gap Kernel 90 actually closes: these objects previously had NO
  // visibility control in any menu, so hiding Kessa's stall from one Cohort
  // meant editing the Scene -- exactly the Scene-as-visibility-container
  // confusion §2 forbids.
  const actions = logic.resolveStageObjectActions(sceneElementModel(), DIRECTOR_CONTEXT);
  const family = familyOf(actions, "k90-visibility");
  assert.ok(family, "a Scene composition element should offer Visibility");
  assert.deepEqual(submenuActions(family), [
    "k90-visibility-visible",
    "k90-visibility-hidden",
    "k90-visibility-scope",
  ]);
});

test("the current state is marked, so the Director reads it rather than inferring it", () => {
  // §12's wording follows current vocabulary; the substance is that a Director
  // should see WHICH state is active, not deduce it from which verb the menu
  // offers.
  const visible = logic.resolveStageObjectActions(liveTokenModel(), DIRECTOR_CONTEXT);
  const visibleFamily = familyOf(visible, "k90-visibility");
  const visibleLabels = visibleFamily.submenu.map((c) => c.label);
  assert.ok(visibleLabels[0].includes("✓"), "Visible should be marked current");
  assert.ok(!visibleLabels[1].includes("✓"), "Hidden should not be marked current");

  const hidden = logic.resolveStageObjectActions(
    liveTokenModel({ visible: false, hidden_backstage_only: true }),
    DIRECTOR_CONTEXT
  );
  const hiddenLabels = familyOf(hidden, "k90-visibility").submenu.map((c) => c.label);
  assert.ok(!hiddenLabels[0].includes("✓"), "Visible should not be marked current");
  assert.ok(hiddenLabels[1].includes("✓"), "Hidden should be marked current");
});

test("Interaction controls appear only for objects that can be acted upon", () => {
  // §12 explicitly forbids showing Interaction controls for objects with no
  // interaction capability. An Enabled/Disabled pair on an inert prop would be
  // a lie about what the object can do.
  const inert = logic.resolveStageObjectActions(liveTokenModel(), DIRECTOR_CONTEXT);
  assert.equal(familyOf(inert, "k90-interaction"), undefined,
    "a token with no bound interaction should get no Interaction family");

  const bound = withBinding(sceneElementModel(), {
    participant_interaction_id: "int-1",
    stage_button_label: "Speak with Kessa",
    enabled: true,
    show_interaction_enabled: true,
  });
  const actions = logic.resolveStageObjectActions(bound, DIRECTOR_CONTEXT);
  const family = familyOf(actions, "k90-interaction");
  assert.ok(family, "a bound interaction should get an Interaction family");
  assert.deepEqual(submenuActions(family), [
    "k90-interaction-enabled",
    "k90-interaction-disabled",
  ]);
});

test("interaction state reflects the Show-scoped flag, not the global authoring flag", () => {
  // §20's bridge, at the UI layer. A Director who switched an interaction off
  // for this Show must see "Disabled ✓" -- and must NOT see it merely because
  // an author disabled the interaction globally, which is a different fact
  // they cannot fix from here.
  const showDisabled = withBinding(sceneElementModel(), {
    participant_interaction_id: "int-1",
    enabled: false,
    show_interaction_enabled: false,
  });
  const labels = familyOf(
    logic.resolveStageObjectActions(showDisabled, DIRECTOR_CONTEXT),
    "k90-interaction"
  ).submenu.map((c) => c.label);
  assert.ok(labels[1].includes("✓"), "Disabled should be marked current");
  assert.ok(!labels[0].includes("✓"));

  // Globally disabled but NOT Show-disabled: the Director's own switch is
  // still "Enabled", because that is the only dimension this menu controls.
  const globallyOff = withBinding(sceneElementModel(), {
    participant_interaction_id: "int-1",
    enabled: false,
    show_interaction_enabled: true,
  });
  const globalLabels = familyOf(
    logic.resolveStageObjectActions(globallyOff, DIRECTOR_CONTEXT),
    "k90-interaction"
  ).submenu.map((c) => c.label);
  assert.ok(globalLabels[0].includes("✓"),
    "a globally-disabled interaction should still show the Director's own switch as Enabled");
});

test("a locked object offers no visibility controls", () => {
  // Consistent with every other edit in this menu: locking an object is how a
  // Director protects it from accidental change, and visibility is a change.
  const locked = logic.resolveStageObjectActions(
    liveTokenModel({ locked: true }),
    DIRECTOR_CONTEXT
  );
  assert.equal(familyOf(locked, "k90-visibility"), undefined);
});

test("an object with no canonical identity offers no controls", () => {
  // Fail closed. A stage object this engine cannot name is one whose hidden
  // state it cannot check, so offering a control would produce a request the
  // server must refuse.
  const anonymous = liveTokenModel();
  delete anonymous.state.stage_object_kind;
  delete anonymous.state.stage_object_id;
  const actions = logic.resolveStageObjectActions(anonymous, DIRECTOR_CONTEXT);
  assert.equal(familyOf(actions, "k90-visibility"), undefined);
});

test("stageObjectHidden prefers the backstage marker over the legacy visible flag", () => {
  assert.equal(logic.stageObjectHidden(liveTokenModel()), false);
  assert.equal(logic.stageObjectHidden(liveTokenModel({ hidden_backstage_only: true })), true);

  // With no marker (an ordinary viewer's payload), fall back to the "visible"
  // key the canonical projector now writes.
  const noMarker = liveTokenModel({ visible: false });
  delete noMarker.state.hidden_backstage_only;
  assert.equal(logic.stageObjectHidden(noMarker), true);
});

test("boundInteractionRef reads the binding the server sent and nothing else", () => {
  assert.equal(logic.boundInteractionRef(sceneElementModel()), null);

  const bound = withBinding(sceneElementModel(), {
    participant_interaction_id: "int-9",
    stage_button_label: "Train",
    enabled: true,
    show_interaction_enabled: false,
  });
  assert.deepEqual(logic.boundInteractionRef(bound), {
    kind: "participant_interaction",
    id: "int-9",
    enabled: true,
    showEnabled: false,
    label: "Train",
  });
});
