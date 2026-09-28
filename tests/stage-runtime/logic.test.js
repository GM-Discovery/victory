const test = require("node:test");
const assert = require("node:assert/strict");

globalThis.VictoryStageVenue = { slug: "catharsis", name: "Catharsis" };
const logic = require("../../frontend/lib/stage-runtime/logic.js");

function tokenModel(overrides = {}) {
  return {
    kind: "token",
    live: true,
    key: "token-1",
    elementId: "token-1",
    elementSlug: "goat",
    label: "Goat",
    position: { x: 10, y: 20, order: 3, frame: "center" },
    source: {
      data: {
        asset_id: "asset-1",
        asset_name: "Goat",
        asset_content_url: "/api/assets/asset-1/content?variant=stage",
        asset_shape: "circle",
        default_grid_width: 1,
        default_grid_height: 1,
        scale: 150,
        snap_mode: "grid",
        token_layer: "director",
        status_percent: 66,
        ...overrides.data,
      },
      visibility: {
        visible: true,
        nameplate_visible: true,
      },
      state: {
        locked: false,
      },
    },
    visibility: {
      visible: true,
      nameplate_visible: true,
    },
    state: {
      locked: false,
    },
    ...overrides,
  };
}

test("basic object helpers normalize model metadata", () => {
  assert.equal(logic.objectKind({ asset_id: "asset-1" }), "token");
  assert.equal(logic.objectKind({ kind: "index_card" }), "card");
  assert.deepEqual(logic.objectState({ state: { locked: 1 }, visibility: { nameplateVisible: 0 } }), {
    locked: true,
    nameplateVisible: false,
    visible: true,
  });
  assert.equal(logic.updateObjectMatches({ key: "a" }, { key: "a" }), true);
  assert.equal(logic.updateObjectMatches({ elementSlug: "goat" }, { elementSlug: "goat" }), true);
});

test("role and visibility helpers respect current permissions", () => {
  assert.equal(logic.viewerCanSeeHiddenCards("audience"), false);
  assert.equal(logic.viewerCanSeeHiddenCards("director"), true);
  assert.equal(logic.canActorRevealHideStageObjects("cast", { actors_can_reveal: "yes" }), true);
  assert.equal(logic.canActorRevealHideStageObjects("cast", { actors_can_reveal: "no" }), false);
});

test("token helpers return the expected presentation values", () => {
  const model = tokenModel();
  assert.equal(logic.isTokenObject(model), true);
  assert.equal(logic.isCardObject(model), false);
  assert.equal(logic.isLiveStageObject(model), true);
  assert.equal(logic.tokenLayerForModel(model), "director");
  assert.equal(logic.tokenScaleForModel(model), 150);
  assert.equal(logic.tokenDisplayNameForModel(model), "Goat");
  assert.equal(logic.tokenStatusPercentForModel(model), 66);
  assert.deepEqual(logic.tokenFootprintForModel(model), { width: 1, height: 1 });
  assert.deepEqual(logic.tokenDisplaySizeForModel(model, { grid_type: "square", cell_size: 80 }), { width: 120, height: 120 });
  assert.equal(logic.cardStatusBadgeText(model), "DIR");
});

test("token status helpers stay dormant until explicit metric data is present", () => {
  const dormant = {
    kind: "token",
    live: true,
    source: { data: {} },
  };
  assert.equal(logic.tokenStatusPercentForModel(dormant), null);

  const explicit = tokenModel({ data: { status_percent: 42 } });
  assert.equal(logic.tokenStatusPercentForModel(explicit), 42);
});

test("card helpers and truncation behave deterministically", () => {
  const model = {
    kind: "card",
    live: true,
    key: "card-1",
    elementId: "card-1",
    elementSlug: "note",
    label: "Note",
    source: {
      data: {
        front_text: "Front",
        back_text: "Back",
      },
      visibility: {
        visible: false,
        nameplate_visible: false,
      },
      state: {
        locked: true,
      },
    },
  };
  assert.equal(logic.isCardObject(model), true);
  assert.equal(logic.canEditLiveCard(model, true), false);
  assert.equal(logic.canMoveLiveStageObject(model, true), false);
  assert.equal(logic.canToggleNameplate(model, true), false);
  assert.equal(logic.cardStatusBadgeText(model), "");
  assert.equal(logic.truncateCardText("  A title that is definitely too long  ", 12), "A title tha…");
  assert.deepEqual(logic.cardEditorDraftFromModel(model), {
    elementId: "card-1",
    elementSlug: "note",
    label: "Note",
    frontText: "Front",
    backText: "Back",
    color: "#d9c7a6",
  });
});

test("selection menu generation covers stage and token actions", () => {
  const stageActions = logic.resolveStageObjectActions(logic.stageContextMenuModel(), {
    canManageIndexCards: true,
    canManageStageTokens: true,
    canActorRevealHideStageObjects: true,
    hasSelection: true,
  });
  assert.equal(stageActions.some((item) => item.action === "add-token"), true);
  assert.equal(stageActions.some((item) => item.action === "configure-grid"), true);
  assert.equal(stageActions.some((item) => item.action === "clear"), true);

  const tokenActions = logic.resolveStageObjectActions(tokenModel(), {
    canManageIndexCards: true,
    canManageStageTokens: true,
    canActorRevealHideStageObjects: true,
    cardFaceForModel: () => "back",
    cardDisplayMode: () => "world",
    tokenScaleForModel: () => 150,
    tokenSnapModeForModel: () => "grid",
    tokenLayerForModel: () => "director",
  });
  assert.equal(tokenActions.some((item) => item.action === "scale"), true);
  assert.equal(tokenActions.some((item) => item.action === "free-placement"), true);
  assert.equal(tokenActions.some((item) => item.action === "move-public-layer"), true);
});

// Kernel 101 (101-24) follow-up: the menu half of the Cast create path.
// "Add Token" must be offered and ENABLED for Cast, while every action that
// edits an existing token -- including the one Cast just placed -- stays
// absent. Guards the boundary that canCreateStageObjects exists to draw.
test("Cast is offered token creation but no token editing", () => {
  const castContext = {
    canManageIndexCards: false,
    canManageStageTokens: false,
    canCreateStageObjects: true,
    canActorRevealHideStageObjects: false,
    hasSelection: true,
  };

  const stageActions = logic.resolveStageObjectActions(logic.stageContextMenuModel(), castContext);
  const addToken = stageActions.find((item) => item.action === "add-token");
  assert.ok(addToken, "Cast must be offered Add Token");
  assert.notEqual(addToken.disabled, true, "Add Token must be enabled, not a dead entry");

  // Stage-level authoring Cast must not get. These entries are still listed
  // (this menu shows unavailable tools greyed rather than hiding them), so
  // what matters is that they stay DISABLED -- the exact state Add Token was
  // wrongly left in on the picker side before this fix.
  for (const denied of ["set-map", "configure-grid", "open-scene-configuration"]) {
    const entry = stageActions.find((item) => item.action === denied);
    if (entry) {
      assert.equal(entry.disabled, true, `Cast must not be able to use "${denied}"`);
    }
  }

  const tokenActions = logic.resolveStageObjectActions(tokenModel(), {
    ...castContext,
    cardFaceForModel: () => "back",
    cardDisplayMode: () => "world",
    tokenScaleForModel: () => 150,
    tokenSnapModeForModel: () => "grid",
    tokenLayerForModel: () => "director",
  });
  for (const denied of [
    "scale",
    "replace-asset",
    "free-placement",
    "snap-to-grid",
    "move-public-layer",
    "move-director-layer",
    "move-here",
    "duplicate",
    "remove",
    "lock",
    "unlock",
  ]) {
    assert.equal(
      tokenActions.some((item) => item.action === denied),
      false,
      `Cast must not be offered "${denied}" on an existing token`,
    );
  }
});
