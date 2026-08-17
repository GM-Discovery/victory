// Kernel 89 unit tests for the two pure/DOM-light pieces of the Director
// tool surface: the context-menu shape logic.js produces, and the
// announcement renderer's routing/queue behaviour.
//
// The panels themselves are not unit-tested here on purpose -- they are
// fetch-driven DOM, and the honest proof for those is the browser run
// (scripts/smoke/kernel89-director-tools-browser.js), not a mock of every
// endpoint they call.
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

function stageModel() {
  return { kind: "stage", key: "stage", label: "Stage", live: false };
}

function tokenModel() {
  return {
    kind: "token",
    live: true,
    key: "token-1",
    elementId: "token-1",
    elementSlug: "russel",
    label: "Russel",
    position: { x: 10, y: 20, order: 1, frame: "center" },
    source: { data: { asset_id: "a1", snap_mode: "grid", token_layer: "public" }, visibility: {} },
    state: { visible: true, locked: false, nameplateVisible: true },
  };
}

test("the stage menu offers Director tools as ONE nested family, not a row of buttons", () => {
  const actions = logic.resolveStageObjectActions(stageModel(), DIRECTOR_CONTEXT);
  const family = actions.find((a) => a.action === "k89-director-tools");
  assert.ok(family, "stage menu should carry a Director Tools family");
  assert.ok(Array.isArray(family.submenu) && family.submenu.length >= 4,
    "the family should hold its operations as a submenu");

  // The whole point of §5/§25: the family's children must NOT also appear
  // as top-level entries, or nesting has bought nothing.
  const topLevel = new Set(actions.map((a) => a.action));
  for (const child of family.submenu) {
    if (child.action === "open-scene-configuration") continue; // deliberately both, see logic.js
    assert.ok(!topLevel.has(child.action),
      `${child.action} should live only inside the family, not also at top level`);
  }
});

test("a token menu reaches the same Director tool family", () => {
  const actions = logic.resolveStageObjectActions(tokenModel(), DIRECTOR_CONTEXT);
  const family = actions.find((a) => a.action === "k89-director-tools");
  assert.ok(family, "token menu should carry a Director Tools family (§12)");
  const labels = family.submenu.map((c) => c.action);
  assert.ok(labels.includes("k89-merchant"), "merchant should be reachable from a token");
  assert.ok(labels.includes("k89-aftercare"), "aftercare should be reachable from a token");
});

test("a Player's token menu carries no Director tool family", () => {
  const actions = logic.resolveStageObjectActions(tokenModel(), {
    role: "cast",
    canManageIndexCards: false,
    canManageStageTokens: false,
    canActorRevealHideStageObjects: false,
    hasSelection: false,
  });
  assert.equal(actions.find((a) => a.action === "k89-director-tools"), undefined);
});

test("the stage menu stays short enough to read", () => {
  // §40 marks "context menu becomes an unmanageable flat list" as a
  // PARTIAL. Nesting is what keeps this number from growing with every
  // Director operation a future kernel adds.
  const actions = logic.resolveStageObjectActions(stageModel(), DIRECTOR_CONTEXT);
  assert.ok(actions.length <= 8, `stage menu has ${actions.length} top-level entries`);
});

// --- Announcement renderer -------------------------------------------------

function loadAnnouncements() {
  // The module is a plain IIFE that assigns to window, so it needs a DOM
  // shaped just enough to install itself. Rather than pull in jsdom (this
  // repo has no test dependencies at all -- plain node --test), stub the
  // handful of DOM calls the module makes at load and during present().
  const nodes = [];
  function makeEl(tag) {
    const el = {
      tagName: tag, id: "", className: "", textContent: "", innerHTML: "",
      dataset: {}, style: { setProperty() {} }, children: [],
      appendChild(child) { this.children.push(child); return child; },
      remove() { nodes.splice(nodes.indexOf(this), 1); },
      setAttribute() {}, addEventListener() {},
      querySelector() { return null; },
      querySelectorAll() { return []; },
      classList: { add() {}, remove() {} },
    };
    nodes.push(el);
    return el;
  }
  const doc = {
    readyState: "complete",
    head: makeEl("head"),
    body: makeEl("body"),
    createElement: makeEl,
    getElementById: () => null,
    addEventListener() {},
  };
  doc.body.contains = () => true;

  const previousWindow = globalThis.window;
  const previousDocument = globalThis.document;
  const previousRAF = globalThis.requestAnimationFrame;

  globalThis.window = {
    setTimeout: () => 0,
    clearTimeout: () => {},
    requestAnimationFrame: (fn) => fn(),
  };
  globalThis.document = doc;
  globalThis.requestAnimationFrame = (fn) => fn();

  delete require.cache[require.resolve("../../frontend/lib/stage-runtime/kernel89-announcements.js")];
  require("../../frontend/lib/stage-runtime/kernel89-announcements.js");
  const controller = globalThis.window.VictoryKernel89Announcements;

  return {
    controller,
    restore() {
      globalThis.window = previousWindow;
      globalThis.document = previousDocument;
      globalThis.requestAnimationFrame = previousRAF;
    },
  };
}

function announcementEffect(overrides = {}) {
  return {
    effect_id: "fx_1",
    type: "announcement",
    duration_ms: 5000,
    payload: {
      text: "The training wall gives way!",
      style: {
        key: "consequences", label: "Consequences", accent: "#e3585f",
        background: "#2c0f13", ink: "#ffe8ea", glyph: "▲",
        motion: "shake", shape: "jagged", emphasis: "loud",
      },
    },
    ...overrides,
  };
}

test("the announcement renderer claims announcements and declines everything else", () => {
  const { controller, restore } = loadAnnouncements();
  try {
    assert.equal(controller.present(announcementEffect()), true, "should claim an announcement");
    assert.equal(controller.present({ effect_id: "fx_2", type: "dice_roll", payload: {} }), false,
      "a dice roll must fall through to the Kernel 86 projection");
    assert.equal(controller.present(null), false);

    const shown = controller.inspect();
    assert.equal(shown.showing, "consequences");
    assert.equal(shown.text, "The training wall gives way!");
  } finally {
    restore();
  }
});

test("the same announcement arriving twice does not stack", () => {
  const { controller, restore } = loadAnnouncements();
  try {
    controller.present(announcementEffect());
    controller.present(announcementEffect());
    assert.equal(controller.inspect().queued, 0, "a duplicate effect id must not queue a second banner");
  } finally {
    restore();
  }
});

test("an announcement with no text is claimed but never rendered", () => {
  const { controller, restore } = loadAnnouncements();
  try {
    // Claimed, so it is never handed to the dice projection (which would
    // try to read dice off it); dropped, so nothing blank appears on stage.
    assert.equal(controller.present(announcementEffect({ payload: { text: "  ", style: {} } })), true);
    assert.equal(controller.inspect().showing, "");
  } finally {
    restore();
  }
});

test("pin claims only announcements, and dismiss clears the pin", () => {
  const { controller, restore } = loadAnnouncements();
  try {
    controller.present(announcementEffect());
    assert.equal(controller.pin(announcementEffect()), true);
    assert.deepEqual(controller.inspect().pinned, ["fx_1"]);

    assert.equal(controller.pin({ effect_id: "fx_9", type: "dice_roll" }), false,
      "a pinned dice roll must still reach the dice projection");

    controller.dismiss("fx_1");
    assert.deepEqual(controller.inspect().pinned, []);
  } finally {
    restore();
  }
});
