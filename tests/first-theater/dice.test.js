const test = require("node:test");
const assert = require("node:assert/strict");

const diceModule = require("../../frontend/venues/first-theater/runtime/dice.js");

class FakeElement {
  constructor(tagName) {
    this.tagName = String(tagName || "div").toUpperCase();
    this.children = [];
    this.parentNode = null;
    this.className = "";
    this.textContent = "";
    this.dataset = {};
    this.style = {};
    this.value = "";
    this.checked = false;
    this.type = "";
    this.placeholder = "";
    this.maxLength = 0;
    this.min = "";
    this.max = "";
    this.step = "";
    this.autocomplete = "";
    this.spellcheck = true;
    this.hidden = false;
    this.isConnected = true;
    this.listeners = new Map();
    Object.defineProperty(this, "innerHTML", {
      get: () => this.children.map((child) => child?.textContent || "").join(""),
      set: () => {
        for (const child of this.children) {
          if (child) {
            child.parentNode = null;
          }
        }
        this.children = [];
      },
      configurable: true,
    });
  }

  appendChild(child) {
    if (!child) return child;
    child.parentNode = this;
    this.children.push(child);
    return child;
  }

  append(...nodes) {
    for (const node of nodes) {
      if (typeof node === "string") {
        const text = new FakeElement("#text");
        text.textContent = node;
        this.appendChild(text);
      } else {
        this.appendChild(node);
      }
    }
  }

  addEventListener(type, handler) {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, new Set());
    }
    this.listeners.get(type).add(handler);
  }

  removeEventListener(type, handler) {
    const set = this.listeners.get(type);
    if (set) {
      set.delete(handler);
    }
  }

  dispatchEvent(event) {
    const type = String(event?.type || "");
    const set = this.listeners.get(type);
    if (!set) return true;
    for (const handler of set) {
      handler.call(this, event);
    }
    return true;
  }

  click() {
    this.dispatchEvent({ type: "click" });
  }

  querySelector(selector) {
    if (!selector.startsWith(".")) return null;
    const className = selector.slice(1);
    return this.find((node) => String(node.className || "").split(/\s+/).includes(className));
  }

  find(predicate) {
    if (predicate(this)) return this;
    for (const child of this.children) {
      if (typeof child?.find === "function") {
        const found = child.find(predicate);
        if (found) return found;
      }
    }
    return null;
  }
}

class FakeDocument {
  createElement(tagName) {
    return new FakeElement(tagName);
  }
}

function findButtons(root) {
  const out = [];
  root.find((node) => {
    if (String(node.tagName).toLowerCase() === "button") {
      out.push(node);
    }
    return false;
  });
  return out;
}

test("dice tray helpers build expressions and format explosion chains", () => {
  assert.equal(diceModule.buildExpression({ count: 1, sides: 20, explode: false, modifier: 0 }), "d20");
  assert.equal(diceModule.buildExpression({ count: 2, sides: 13, explode: true, modifier: 4 }), "2d13!+4");
  assert.deepEqual(diceModule.rollSequenceProgressions([10, 10, 4]), ["10", "10 ↻ 10", "10 ↻ 10 ↻ 4"]);
});

test("dice tray quick buttons and manual controls update expressions", () => {
  const doc = new FakeDocument();
  const root = new FakeElement("div");
  const controller = diceModule.createDiceTrayController({
    document: doc,
    window: { setTimeout, clearTimeout, crypto: globalThis.crypto, matchMedia: () => ({ matches: true }) },
    mountRoot: root,
    sendAction: () => true,
    getCurrentSnapshot: () => ({ actions: [] }),
    setStageStatus: () => {},
    setMovementLine: () => {},
    appendSystemChatNotice: () => {},
  });

  const buttons = findButtons(root);
  const d100Button = buttons.find((button) => button.textContent === "d100");
  assert.ok(d100Button);
  d100Button.click();

  controller.setCount(2);
  controller.setExplode(true);
  controller.setModifier(3);

  assert.equal(controller.getExpression(), "2d100!+3");
  controller.setExpression("3d7");
  assert.equal(controller.getExpression(), "3d7");
});

test("dice tray roll button submits the current expression", async () => {
  const doc = new FakeDocument();
  const root = new FakeElement("div");
  const calls = [];
  const controller = diceModule.createDiceTrayController({
    document: doc,
    window: { setTimeout, clearTimeout, crypto: globalThis.crypto, matchMedia: () => ({ matches: true }) },
    mountRoot: root,
    sendAction: (type, payload) => {
      calls.push([type, payload]);
      return true;
    },
    getCurrentSnapshot: () => ({ actions: [] }),
    setStageStatus: () => {},
    setMovementLine: () => {},
    appendSystemChatNotice: () => {},
  });

  const rollButton = findButtons(root).find((button) => button.textContent === "Roll");
  assert.ok(rollButton);
  controller.setExpression("2d6+1");
  rollButton.click();

  const payload = calls.find((entry) => entry[0] === "roll/dice")?.[1];
  assert.ok(payload);
  assert.equal(payload.expression, "2d6+1");
  assert.equal(payload.visibility, "public");

  controller.handleAction({
    id: "roll-1",
    type: "roll/dice",
    moment_id: 1,
    ts: "2026-06-26T00:00:00Z",
    actor_id: "user-1",
    actor: { user_id: "user-1", display_name: "Grant", handle: "grant", role: "producer" },
    payload: {
      request_id: payload.request_id,
      expression: "2d6+1",
      spec: { count: 2, sides: 6, explode_on_max: false, modifier: 1 },
      dice: [{ index: 0, chain: [3], subtotal: 3 }, { index: 1, chain: [4], subtotal: 4 }],
      explosion_count: 0,
      modifier: 1,
      total: 8,
      roll_version: 1,
      visibility_mode: "public",
      label: "",
    },
  });
  await new Promise((resolve) => setTimeout(resolve, 0));
});

test("dice tray roll requests resolve against the canonical action stream", async () => {
  const doc = new FakeDocument();
  const root = new FakeElement("div");
  const calls = [];
  const controller = diceModule.createDiceTrayController({
    document: doc,
    window: { setTimeout, clearTimeout, crypto: globalThis.crypto, matchMedia: () => ({ matches: true }) },
    mountRoot: root,
    sendAction: (type, payload) => {
      calls.push([type, payload]);
      return true;
    },
    getCurrentSnapshot: () => ({ actions: [] }),
    setStageStatus: () => {},
    setMovementLine: () => {},
    appendSystemChatNotice: () => {},
  });

  const promise = controller.roll({ expression: "2d6+3", label: "Parent One" });
  const payload = calls.find((entry) => entry[0] === "roll/dice")[1];
  controller.handleAction({
    id: "roll-1",
    type: "roll/dice",
    moment_id: 44,
    ts: "2026-06-25T00:00:00Z",
    actor_id: "user-1",
    actor: { user_id: "user-1", display_name: "Grant", handle: "grant", role: "producer" },
    payload: {
      request_id: payload.request_id,
      expression: "2d6+3",
      spec: { count: 2, sides: 6, explode_on_max: false, modifier: 3 },
      dice: [
        { index: 0, chain: [4], subtotal: 4 },
        { index: 1, chain: [6], subtotal: 6 },
      ],
      explosion_count: 0,
      modifier: 3,
      total: 13,
      roll_version: 1,
      visibility_mode: "public",
      label: "Parent One",
    },
  });

  const result = await promise;
  assert.equal(result.total, 13);
  assert.equal(controller.getHistory().length, 1);
  assert.equal(controller.getHistory()[0].total, 13);
});

test("dice tray resolves a single pending roll even if the action omits request id", async () => {
  const doc = new FakeDocument();
  const root = new FakeElement("div");
  const calls = [];
  const controller = diceModule.createDiceTrayController({
    document: doc,
    window: { setTimeout, clearTimeout, crypto: globalThis.crypto, matchMedia: () => ({ matches: true }) },
    mountRoot: root,
    sendAction: (type, payload) => {
      calls.push([type, payload]);
      return true;
    },
    getCurrentSnapshot: () => ({ actions: [] }),
    setStageStatus: () => {},
    setMovementLine: () => {},
    appendSystemChatNotice: () => {},
  });

  const promise = controller.roll({ expression: "d20" });
  const payload = calls.find((entry) => entry[0] === "roll/dice")[1];
  controller.handleAction({
    id: "roll-1",
    type: "roll/dice",
    moment_id: 99,
    ts: "2026-06-26T00:00:00Z",
    actor_id: "user-1",
    actor: { user_id: "user-1", display_name: "Grant", handle: "grant", role: "producer" },
    payload: {
      expression: "d20",
      spec: { count: 1, sides: 20, explode_on_max: false, modifier: 0 },
      dice: [{ index: 0, chain: [17], subtotal: 17 }],
      explosion_count: 0,
      modifier: 0,
      total: 17,
      roll_version: 1,
      visibility_mode: "public",
      label: "",
    },
  });

  const result = await promise;
  assert.equal(result.total, 17);
  assert.equal(controller.getPendingCount(), 0);
});

test("dice tray rejects timed out rolls and cleans pending state", async () => {
  const doc = new FakeDocument();
  const root = new FakeElement("div");
  let timerCallback = null;
  const controller = diceModule.createDiceTrayController({
    document: doc,
    window: { setTimeout, clearTimeout, crypto: globalThis.crypto, matchMedia: () => ({ matches: true }) },
    mountRoot: root,
    sendAction: () => true,
    getCurrentSnapshot: () => ({ actions: [] }),
    setStageStatus: () => {},
    setMovementLine: () => {},
    appendSystemChatNotice: () => {},
    timeoutMs: 1,
    setTimeoutFn: (callback) => {
      timerCallback = callback;
      return 1;
    },
    clearTimeoutFn: () => {},
  });

  const promise = controller.roll({ expression: "d20" });
  assert.equal(controller.getPendingCount(), 1);
  timerCallback();
  await assert.rejects(promise, /roll_timeout/);
  assert.equal(controller.getPendingCount(), 0);
});

test("dice tray refreshes world state after a timeout and surfaces the canonical roll", async () => {
  const doc = new FakeDocument();
  const root = new FakeElement("div");
  let timerCallback = null;
  let refreshCount = 0;
  const controller = diceModule.createDiceTrayController({
    document: doc,
    window: { setTimeout, clearTimeout, crypto: globalThis.crypto, matchMedia: () => ({ matches: true }) },
    mountRoot: root,
    sendAction: () => true,
    getCurrentSnapshot: () => ({ actions: [] }),
    refreshWorld: async () => {
      refreshCount += 1;
      controller.handleSnapshot({
        actions: [
          {
            id: "roll-9",
            moment_id: 9,
            type: "roll/dice",
            actor_id: "user-1",
            payload: {
              request_id: "req-9",
              expression: "d20",
              dice: [{ index: 0, chain: [11], subtotal: 11 }],
              total: 11,
              modifier: 0,
              explosion_count: 0,
              roll_version: 1,
              visibility_mode: "public",
            },
          },
        ],
      });
    },
    setStageStatus: () => {},
    setMovementLine: () => {},
    appendSystemChatNotice: () => {},
    timeoutMs: 1,
    setTimeoutFn: (callback) => {
      timerCallback = callback;
      return 1;
    },
    clearTimeoutFn: () => {},
  });

  const promise = controller.submitRoll();
  timerCallback();
  const result = await promise;
  assert.equal(refreshCount, 1);
  assert.equal(result.total, 11);
  assert.equal(controller.getHistory()[0].total, 11);
});

test("dice tray rejects validation errors using the request id", async () => {
  const doc = new FakeDocument();
  const root = new FakeElement("div");
  const calls = [];
  const controller = diceModule.createDiceTrayController({
    document: doc,
    window: { setTimeout, clearTimeout, crypto: globalThis.crypto, matchMedia: () => ({ matches: true }) },
    mountRoot: root,
    sendAction: (type, payload) => {
      calls.push([type, payload]);
      return true;
    },
    getCurrentSnapshot: () => ({ actions: [] }),
    setStageStatus: () => {},
    setMovementLine: () => {},
    appendSystemChatNotice: () => {},
  });

  await assert.rejects(async () => {
    const promise = controller.roll({ expression: "d20" });
    assert.equal(controller.getPendingCount(), 1);
    const payload = calls.find((entry) => entry[0] === "roll/dice")[1];
    assert.equal(controller.handleError("invalid_expression", { request_id: payload.request_id }), true);
    await promise;
  }, /invalid_expression/);
  assert.equal(controller.getPendingCount(), 0);
});

test("dice tray keeps only the last two rolls in history", () => {
  const doc = new FakeDocument();
  const root = new FakeElement("div");
  const controller = diceModule.createDiceTrayController({
    document: doc,
    window: { setTimeout, clearTimeout, crypto: globalThis.crypto, matchMedia: () => ({ matches: true }) },
    mountRoot: root,
    sendAction: () => true,
    getCurrentSnapshot: () => ({ actions: [] }),
    setStageStatus: () => {},
    setMovementLine: () => {},
    appendSystemChatNotice: () => {},
  });

  controller.handleSnapshot({
    actions: [
      {
        id: "roll-1",
        moment_id: 1,
        type: "roll/dice",
        ts: "2026-06-26T00:00:01Z",
        actor: { display_name: "Grant", handle: "grant", role: "producer" },
        payload: { expression: "d20", total: 3, dice: [{ index: 0, chain: [3], subtotal: 3 }], modifier: 0 },
      },
      {
        id: "roll-2",
        moment_id: 2,
        type: "roll/dice",
        ts: "2026-06-26T00:00:02Z",
        actor: { display_name: "Grant", handle: "grant", role: "producer" },
        payload: { expression: "d20", total: 7, dice: [{ index: 0, chain: [7], subtotal: 7 }], modifier: 0 },
      },
      {
        id: "roll-3",
        moment_id: 3,
        type: "roll/dice",
        ts: "2026-06-26T00:00:03Z",
        actor: { display_name: "Grant", handle: "grant", role: "producer" },
        payload: { expression: "d20", total: 11, dice: [{ index: 0, chain: [11], subtotal: 11 }], modifier: 0 },
      },
    ],
  });

  const history = controller.getHistory();
  assert.equal(history.length, 2);
  assert.deepEqual(history.map((roll) => roll.id), ["roll-3", "roll-2"]);
});

test("dice tray mount and destroy do not duplicate listeners", () => {
  const doc = new FakeDocument();
  const root = new FakeElement("div");
  const controller = diceModule.createDiceTrayController({
    document: doc,
    window: { setTimeout, clearTimeout, crypto: globalThis.crypto, matchMedia: () => ({ matches: true }) },
    mountRoot: root,
    sendAction: () => true,
    getCurrentSnapshot: () => ({ actions: [] }),
    setStageStatus: () => {},
    setMovementLine: () => {},
    appendSystemChatNotice: () => {},
  });

  controller.destroy();
  controller.mount(root);
  const buttons = findButtons(root);
  const d100Button = buttons.find((button) => button.textContent === "d100");
  assert.equal(d100Button.listeners.get("click").size, 1);
  d100Button.click();
  assert.equal(controller.getExpression(), "d100");
});
