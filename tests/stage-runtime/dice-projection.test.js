const test = require("node:test");
const assert = require("node:assert/strict");

const diceProjectionModule = require("../../frontend/lib/stage-runtime/dice-projection.js");

// Minimal fake PIXI: only the surface dice-projection.js actually touches
// (Container/Graphics/Text with chainable no-op drawing calls, settable
// position/scale/alpha/rotation, and a bare pointertap listener registry
// for the Pin/Dismiss buttons). Mirrors dice.test.js's FakeElement/
// FakeDocument convention of a small hand-rolled double rather than a real
// rendering dependency. NOTE: position.set writes to the display object's
// own .x/.y (not .position.x/.position.y) -- dice-projection.js never
// reads position back, only ever calls .position.set(...), so tests read
// landed coordinates via node.x / node.y.
class FakeDisplayObject {
  constructor() {
    this.children = [];
    this.alpha = 1;
    this.rotation = 0;
    this.eventMode = "auto";
    this.cursor = "";
    this.width = 0;
    this._handlers = {};
    const self = this;
    this.position = { x: 0, y: 0, set(x, y) { self.x = x; self.y = y; } };
    this.scale = { x: 1, y: 1, set(v) { self.scale.x = v; self.scale.y = v; } };
  }
  addChild(...items) {
    this.children.push(...items);
    return items[0];
  }
  destroy() {
    this.destroyed = true;
  }
  on(event, cb) {
    this._handlers[event] = cb;
    return this;
  }
  removeAllListeners(event) {
    delete this._handlers[event];
    return this;
  }
  emit(event) {
    this._handlers[event]?.();
  }
}

class FakeContainer extends FakeDisplayObject {
  constructor() {
    super();
    this.sortableChildren = false;
  }
}

// PIXI v7 Graphics API (beginFill/lineStyle/drawRoundedRect/endFill) --
// matches the real pinned frontend/lib/pixi.min.js, not the v8 fill()/
// stroke()/roundRect() chain.
class FakeGraphics extends FakeDisplayObject {
  beginFill() { return this; }
  lineStyle() { return this; }
  drawRoundedRect() { return this; }
  endFill() { return this; }
}

// PIXI v7 Text takes positional (text, style) args, not a v8-style
// { text, style } options object.
class FakeText extends FakeDisplayObject {
  constructor(text = "", style = {}) {
    super();
    this.text = String(text ?? "");
    this.style = style;
    this.anchor = { set() {} };
  }
}

class FakeTextStyle {
  constructor(style = {}) {
    return style;
  }
}

const FakePIXI = { Container: FakeContainer, Graphics: FakeGraphics, Text: FakeText, TextStyle: FakeTextStyle };

// A deterministic fake timer queue: setTimeoutFn records {id, fn} and
// returns an incrementing id (ignores the ms argument entirely -- never
// actually waiting); drain() runs every currently-scheduled callback in
// FIFO order, including any new ones a callback itself schedules
// (dice-projection's roll "tumble" phase reschedules itself every tick),
// up to a safety cap or an explicit step count.
function makeFakeTimers() {
  let nextId = 1;
  let scheduled = [];
  return {
    setTimeoutFn(fn) {
      const id = nextId++;
      scheduled.push({ id, fn });
      return id;
    },
    clearTimeoutFn(id) {
      scheduled = scheduled.filter((entry) => entry.id !== id);
    },
    drain(maxSteps = 500) {
      let steps = 0;
      while (scheduled.length > 0 && steps < maxSteps) {
        const [next, ...rest] = scheduled;
        scheduled = rest;
        next.fn();
        steps += 1;
      }
    },
  };
}

function makeEffect(overrides = {}) {
  return {
    effect_id: overrides.effect_id || "fx_1",
    type: "dice_roll",
    source_action_id: "action-1",
    session_id: "session-1",
    audience: overrides.audience || "show",
    actor_id: overrides.actor_id || "user-1",
    label: overrides.label || "",
    duration_ms: overrides.duration_ms ?? 50,
    payload: {
      actor: { display_name: "Producer Mira" },
      expression: "2d6+3",
      dice: [{ index: 0, chain: [4], subtotal: 4 }, { index: 1, chain: [6], subtotal: 6 }],
      modifier: 3,
      total: 13,
      explosion_count: 0,
      ...overrides.payload,
    },
    ...overrides.raw,
  };
}

const DEFAULT_MAP_BOUNDS = { x: 0, y: 0, width: 1200, height: 900 };

function makeController(overrides = {}) {
  const timers = makeFakeTimers();
  const events = [];
  const sentActions = [];
  const hudLayer = new FakeContainer();
  const worldLayer = new FakeContainer();
  const controller = diceProjectionModule.createDiceProjectionController({
    PIXI: FakePIXI,
    setTimeout: timers.setTimeoutFn,
    clearTimeout: timers.clearTimeoutFn,
    getStageSize: () => ({ width: 960, height: 640 }),
    getDefaultTokenSize: () => 64,
    getMapWorldBounds: () => DEFAULT_MAP_BOUNDS,
    sendAction: (type, extra) => {
      sentActions.push({ type, extra });
      return true;
    },
    onEffectChange: (event) => events.push(event),
    ...overrides,
  });
  const mounted = controller.mount(hudLayer, worldLayer);
  return { controller, timers, events, sentActions, hudLayer, worldLayer, mounted };
}

// --- lifecycle / sequencing -------------------------------------------

test("dice-projection enqueues, animates, holds, and expires a single roll", () => {
  const { controller, timers, events } = makeController();
  controller.enqueue(makeEffect());

  assert.deepEqual(events.map((e) => e.kind).slice(0, 1), ["entered"]);

  timers.drain();

  const kinds = events.map((e) => e.kind);
  assert.deepEqual(kinds, ["entered", "settled", "expired"]);
});

test("announcement never appears until every die in the roll has landed", () => {
  const { controller, timers, events, mounted } = makeController();
  controller.enqueue(makeEffect({
    payload: {
      dice: [
        { index: 0, chain: [3], subtotal: 3 },
        { index: 1, chain: [5], subtotal: 5 },
        { index: 2, chain: [1], subtotal: 1 },
      ],
      total: 9,
    },
  }));

  // Synchronous with "entered" -- no tick has run yet, so nothing has
  // settled and the announcement must not exist.
  assert.equal(mounted.hud.children.length, 0, "announcement must not exist before any die has landed");
  assert.equal(mounted.world.children[0].__dieNodes.length, 3, "all three dice must be in the world layer, not the hud");

  timers.drain();

  const settledEvent = events.find((e) => e.kind === "settled");
  assert.ok(settledEvent, "settled event must fire once all dice land");
  assert.equal(mounted.hud.children.length, 1, "exactly one announcement node appears once every die has landed");
});

test("dice land at individually distinguishable world coordinates and never in the hud layer", () => {
  const { controller, timers, mounted } = makeController();
  controller.enqueue(makeEffect({
    payload: {
      dice: [
        { index: 0, chain: [3], subtotal: 3 },
        { index: 1, chain: [5], subtotal: 5 },
        { index: 2, chain: [1], subtotal: 1 },
      ],
      total: 9,
    },
  }));
  timers.drain();

  const cluster = mounted.world.children[0];
  const dieNodes = cluster.__dieNodes;
  assert.equal(dieNodes.length, 3);

  const seen = new Set();
  for (const node of dieNodes) {
    const key = `${node.x},${node.y}`;
    assert.equal(seen.has(key), false, "no two dice may land on the exact same coordinate");
    seen.add(key);
    assert.ok(node.x >= DEFAULT_MAP_BOUNDS.x && node.x <= DEFAULT_MAP_BOUNDS.x + DEFAULT_MAP_BOUNDS.width, "die must land inside the visible map region (x)");
    assert.ok(node.y >= DEFAULT_MAP_BOUNDS.y && node.y <= DEFAULT_MAP_BOUNDS.y + DEFAULT_MAP_BOUNDS.height, "die must land inside the visible map region (y)");
  }
});

test("dice-projection settles on the exact server-provided final die values, never invents its own", () => {
  const { controller, timers, mounted } = makeController();
  controller.enqueue(makeEffect());
  timers.drain();

  const cluster = mounted.world.children[0];
  const dieNodes = cluster.__dieNodes;
  assert.equal(dieNodes[0].__text.text, "4");
  assert.equal(dieNodes[1].__text.text, "6");

  const announcement = mounted.hud.children[0];
  assert.equal(announcement.children.some((c) => String(c.text || "").includes("13")), true);
});

test("final displayed die value is exactly the server value regardless of what the decorative tumble RNG rolls", () => {
  const originalRandom = Math.random;
  Math.random = () => 0.999; // bias every decorative tumble frame toward the highest visible face
  try {
    const { controller, timers, mounted } = makeController();
    controller.enqueue(makeEffect({
      payload: { dice: [{ index: 0, chain: [2], subtotal: 2 }], total: 2, modifier: 0 },
    }));
    timers.drain();
    const dieNodes = mounted.world.children[0].__dieNodes;
    assert.equal(dieNodes[0].__text.text, "2", "landed face must equal the canonical server value regardless of tumble RNG bias");
  } finally {
    Math.random = originalRandom;
  }
});

// --- queueing ------------------------------------------------------------

test("dice-projection queues a second roll and plays it only after the first expires", () => {
  const { controller, events, timers } = makeController();
  controller.enqueue(makeEffect({ effect_id: "fx_a" }));
  assert.equal(controller.getQueueLength(), 0);

  controller.enqueue(makeEffect({ effect_id: "fx_b" }));
  assert.equal(controller.getQueueLength(), 1, "second roll must queue, not play concurrently");

  timers.drain();

  const enteredOrder = events.filter((e) => e.kind === "entered").map((e) => e.effect.id);
  assert.deepEqual(enteredOrder, ["fx_a", "fx_b"]);
  const expiredOrder = events.filter((e) => e.kind === "expired").map((e) => e.effect.id);
  assert.deepEqual(expiredOrder, ["fx_a", "fx_b"]);
});

test("dice-projection never drops a queued roll -- caps the queue by dropping the OLDEST still-waiting one", () => {
  const { controller } = makeController();
  controller.enqueue(makeEffect({ effect_id: "fx_current" }));
  for (let i = 0; i < 13; i += 1) {
    controller.enqueue(makeEffect({ effect_id: `fx_${i}` }));
  }
  assert.equal(controller.getQueueLength(), 12, "queue must stay bounded, not grow past its cap");
});

// --- pin / dismiss authority & persistence -------------------------------

test("pinning mid-flight keeps the dice in place, fades only the announcement, and the queue continues", () => {
  const { controller, events, timers, mounted } = makeController();
  controller.enqueue(makeEffect({ effect_id: "fx_pin_me" }));
  controller.enqueue(makeEffect({ effect_id: "fx_next" }));

  // Simulate the server's stage_effect_pinned echo arriving while fx_pin_me
  // is still the one actively animating/holding.
  controller.applyPinned(makeEffect({ effect_id: "fx_pin_me" }));

  timers.drain();

  assert.deepEqual(controller.getPinnedIds(), ["fx_pin_me"]);
  const expiredIds = events.filter((e) => e.kind === "expired").map((e) => e.effect.id);
  assert.equal(expiredIds.includes("fx_pin_me"), false, "a pinned roll must not also fade out as expired");
  const enteredIds = events.filter((e) => e.kind === "entered").map((e) => e.effect.id);
  assert.deepEqual(enteredIds, ["fx_pin_me", "fx_next"], "a later transient roll must still project while one is pinned");

  // Dice remain (world layer still has a cluster); nothing forces the
  // announcement to persist alongside pinned dice (spec 1.10).
  const stillHasWorldDice = mounted.world.children.some((c) => Array.isArray(c.__dieNodes) && c.__dieNodes.length > 0);
  assert.equal(stillHasWorldDice, true, "pinned dice must remain in the world layer");
});

test("applyPinned on a reconnect-hydrated effect renders directly into the pinned set without a transient announcement", () => {
  const { controller, mounted } = makeController();
  controller.hydratePinned([makeEffect({ effect_id: "fx_old" }), makeEffect({ effect_id: "fx_old_2" })]);
  assert.deepEqual(controller.getPinnedIds().sort(), ["fx_old", "fx_old_2"]);
  assert.equal(mounted.hud.children.length, 0, "hydrating pinned dice must not resurrect their announcement banner");
});

test("reconnect recomputes identical landed coordinates -- a fresh controller with the same map bounds lands dice in the exact same spot", () => {
  const a = makeController();
  const b = makeController();
  const effect = makeEffect({
    effect_id: "fx_stable",
    payload: {
      dice: [
        { index: 0, chain: [3], subtotal: 3 },
        { index: 1, chain: [5], subtotal: 5 },
      ],
      total: 8,
    },
  });

  a.controller.applyPinned(effect);
  b.controller.applyPinned(effect);

  const posA = a.mounted.world.children[0].__dieNodes.map((n) => ({ x: n.x, y: n.y }));
  const posB = b.mounted.world.children[0].__dieNodes.map((n) => ({ x: n.x, y: n.y }));
  assert.deepEqual(posA, posB, "a reconnecting client must not relocate pinned dice");
});

test("dismiss removes a pinned effect's world cluster and the Dismiss control calls sendAction with the effect id", () => {
  const { controller, sentActions, mounted } = makeController();
  controller.applyPinned(makeEffect({ effect_id: "fx_static" }));
  assert.deepEqual(controller.getPinnedIds(), ["fx_static"]);

  const cluster = mounted.world.children[0];
  cluster.__dismissButton.emit("pointertap");
  assert.deepEqual(sentActions, [{ type: "stage_effect/dismiss", extra: { effect_id: "fx_static" } }]);

  controller.removePinned("fx_static");
  assert.deepEqual(controller.getPinnedIds(), []);
  assert.equal(cluster.destroyed, true, "dismissing must destroy the world-space dice cluster");
});

test("the Pin button on a landed roll calls sendAction with stage_effect/pin and the right effect id", () => {
  const { controller, timers, sentActions, mounted } = makeController();
  controller.enqueue(makeEffect({ effect_id: "fx_click_pin" }));
  timers.drain();

  const announcement = mounted.hud.children[0];
  announcement.__pinButton.emit("pointertap");

  assert.deepEqual(sentActions, [{ type: "stage_effect/pin", extra: { effect_id: "fx_click_pin" } }]);
});

test("hydratePinned clears prior pinned state before applying the fresh reconnect list", () => {
  const { controller } = makeController();
  controller.applyPinned(makeEffect({ effect_id: "fx_stale" }));
  assert.deepEqual(controller.getPinnedIds(), ["fx_stale"]);

  controller.hydratePinned([makeEffect({ effect_id: "fx_fresh" })]);
  assert.deepEqual(controller.getPinnedIds(), ["fx_fresh"]);
});

// --- no-active-map fallback ------------------------------------------------

test("falls back to the original grouped HUD presentation when no active map is loaded", () => {
  const { controller, timers, mounted } = makeController({ getMapWorldBounds: () => null });
  controller.enqueue(makeEffect());
  timers.drain();

  assert.equal(mounted.world.children.length, 0, "no map means no world-space surface to land dice on");
  assert.equal(mounted.hud.children.length, 1);
  const node = mounted.hud.children[0];
  assert.equal(node.__dieNodes.length, 2, "legacy grouped node still carries dice + caption together");
  assert.equal(node.__dieNodes[0].__text.text, "4");
});

test("pinning with no active map falls back to a static grouped HUD node, not degenerate (0,0) dice", () => {
  const { controller, mounted } = makeController({ getMapWorldBounds: () => null });
  controller.applyPinned(makeEffect({ effect_id: "fx_no_map" }));
  assert.deepEqual(controller.getPinnedIds(), ["fx_no_map"]);
  assert.equal(mounted.world.children.length, 0);
  assert.equal(mounted.hud.children.length, 1);
});

// --- explosion safety (kernel 86A spec 13) ---------------------------------

test("the renderer never adds to an explosion chain -- a single-value chain always renders exactly that value, never more", () => {
  const { controller, timers, mounted } = makeController();
  // explosion_count: 0, single chain entry -- an ordinary non-exploding die.
  controller.enqueue(makeEffect({
    payload: { dice: [{ index: 0, chain: [6], subtotal: 6 }], total: 6, modifier: 0, explosion_count: 0 },
  }));
  timers.drain();
  const dieNodes = mounted.world.children[0].__dieNodes;
  assert.equal(dieNodes[0].__text.text, "6", "renderer must never turn a single-entry chain into an explosion");
});

test("an explicitly-exploding canonical chain renders its supplied subtotal faithfully", () => {
  const { controller, timers, mounted } = makeController();
  // A genuine explosion chain the server already resolved (e.g. d12! landing
  // 12 then 7): the renderer must show it faithfully, not suppress it.
  controller.enqueue(makeEffect({
    payload: { dice: [{ index: 0, chain: [12, 7], subtotal: 19 }], total: 19, modifier: 0, explosion_count: 1 },
  }));
  timers.drain();
  const dieNodes = mounted.world.children[0].__dieNodes;
  assert.equal(dieNodes[0].__text.text, "19", "an explicitly requested explosion chain's canonical result must render faithfully");
});

// --- deterministic placement (unit-level, spec 4.2 / 4.3 / 9) --------------

test("computeLandingPositions is a pure deterministic function of (effectId, count, bounds, tokenSize)", () => {
  const bounds = { x: 0, y: 0, width: 1000, height: 800 };
  const a = diceProjectionModule.computeLandingPositions("fx_seed", 4, bounds, 64);
  const b = diceProjectionModule.computeLandingPositions("fx_seed", 4, bounds, 64);
  assert.deepEqual(a, b, "identical inputs must always produce identical positions");

  const differentEffect = diceProjectionModule.computeLandingPositions("fx_other", 4, bounds, 64);
  assert.notDeepEqual(a, differentEffect, "different effect ids should (almost always) land differently");
});

test("computeLandingPositions keeps every die inside the inset-safe map rectangle", () => {
  const bounds = { x: 100, y: 50, width: 600, height: 400 };
  const tokenSize = 64;
  const positions = diceProjectionModule.computeLandingPositions("fx_bounds", 6, bounds, tokenSize);
  for (const p of positions) {
    assert.ok(p.x >= bounds.x && p.x <= bounds.x + bounds.width);
    assert.ok(p.y >= bounds.y && p.y <= bounds.y + bounds.height);
  }
});

test("computeLandingPositions spaces dice apart when the map has room for it", () => {
  const bounds = { x: 0, y: 0, width: 2000, height: 2000 };
  const positions = diceProjectionModule.computeLandingPositions("fx_spacing", 5, bounds, 64);
  const minSeparation = 64 * 1.15;
  for (let i = 0; i < positions.length; i++) {
    for (let j = i + 1; j < positions.length; j++) {
      const dist = Math.hypot(positions[i].x - positions[j].x, positions[i].y - positions[j].y);
      assert.ok(dist >= minSeparation - 1, `dice ${i} and ${j} landed too close together on a spacious map (${dist}px)`);
    }
  }
});

test("normalizeMapBounds rejects missing or non-positive map dimensions", () => {
  assert.equal(diceProjectionModule.normalizeMapBounds(null), null);
  assert.equal(diceProjectionModule.normalizeMapBounds({}), null);
  assert.equal(diceProjectionModule.normalizeMapBounds({ width: 0, height: 500 }), null);
  assert.equal(diceProjectionModule.normalizeMapBounds({ width: 500, height: -1 }), null);
  assert.deepEqual(diceProjectionModule.normalizeMapBounds({ x: 1, y: 2, width: 500, height: 400 }), { x: 1, y: 2, width: 500, height: 400 });
});
