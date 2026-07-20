const test = require("node:test");
const assert = require("node:assert/strict");

const programPanelModule = require("../../frontend/lib/stage-runtime/program-panel.js");

// Minimal DOM shim, following the established convention in dice.test.js
// (this codebase has no jsdom dependency -- program-panel.js accepts an
// injected document/window exactly like dice.js's createDiceTrayController
// does, for the same testability reason).
class FakeElement {
  constructor(tagName) {
    this.tagName = String(tagName || "div").toUpperCase();
    this.children = [];
    this.parentNode = null;
    this.className = "";
    this.textContent = "";
    this.dataset = {};
    this.style = {};
    this.hidden = false;
    this.attributes = {};
    this.listeners = new Map();
    this._focused = false;
    Object.defineProperty(this, "innerHTML", {
      get: () => this._innerHTML || "",
      set: (value) => {
        this._innerHTML = value;
        for (const child of this.children) {
          if (child) child.parentNode = null;
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

  setAttribute(name, value) { this.attributes[name] = value; }
  removeAttribute(name) { delete this.attributes[name]; }

  addEventListener(type, handler) {
    if (!this.listeners.has(type)) this.listeners.set(type, new Set());
    this.listeners.get(type).add(handler);
  }

  dispatchEvent(event) {
    const set = this.listeners.get(event.type);
    if (!set) return true;
    for (const handler of set) handler.call(this, event);
    return true;
  }

  focus() { this._focused = true; }
  querySelectorAll() { return []; }
}

class FakeDocument {
  constructor() {
    this.body = new FakeElement("body");
    this.head = new FakeElement("head");
    this.activeElement = null;
    this._styleIds = new Set();
    // Track style injection so ensureStyles' own de-dup check
    // (doc.getElementById("victory-program-panel-style")) works.
    this.head.appendChild = (child) => {
      FakeElement.prototype.appendChild.call(this.head, child);
      if (child && child.id) this._styleIds.add(child.id);
      return child;
    };
  }
  createElement(tagName) { return new FakeElement(tagName); }
  getElementById(id) { return this._styleIds.has(id) ? { id } : null; }
}

function makeDoc() {
  return new FakeDocument();
}

test("createProgramPanel: escapeHtml escapes the five reserved characters", () => {
  assert.equal(programPanelModule.escapeHtml(`<b>"it's" & fine</b>`), "&lt;b&gt;&quot;it&#39;s&quot; &amp; fine&lt;/b&gt;");
});

test("createProgramPanel: open builds the backdrop, sets header content, and reports isOpen", () => {
  const doc = makeDoc();
  const panel = programPanelModule.createProgramPanel({ document: doc });

  assert.equal(panel.isOpen(), false);
  panel.open({ title: "Kessa", subtitle: "Visit Kessa's Shop" });
  assert.equal(panel.isOpen(), true);

  // The backdrop was appended to document.body exactly once.
  assert.equal(doc.body.children.length, 1);
  const backdrop = doc.body.children[0];
  assert.equal(backdrop.hidden, false);
});

test("createProgramPanel: close hides the backdrop and restores focus", () => {
  const doc = makeDoc();
  const trigger = new FakeElement("button");
  doc.activeElement = trigger;

  const panel = programPanelModule.createProgramPanel({ document: doc });
  let closedCalled = false;
  panel.open({ title: "Kessa", onClose: () => { closedCalled = true; } });
  assert.equal(panel.isOpen(), true);

  panel.close();
  assert.equal(panel.isOpen(), false);
  assert.equal(closedCalled, true);
  assert.equal(trigger._focused, true, "focus should return to the element that was active before open()");
});

test("createProgramPanel: Escape key closes the panel", () => {
  const doc = makeDoc();
  const panel = programPanelModule.createProgramPanel({ document: doc });
  panel.open({ title: "Kessa" });
  assert.equal(panel.isOpen(), true);

  const backdrop = doc.body.children[0];
  backdrop.dispatchEvent({ type: "keydown", key: "Escape", preventDefault: () => {} });
  assert.equal(panel.isOpen(), false);
});

test("createProgramPanel: setBody/setLoading/setError update the body content region", () => {
  const doc = makeDoc();
  const panel = programPanelModule.createProgramPanel({ document: doc });
  panel.open({ title: "Kessa" });

  panel.setLoading(true);
  const backdrop = doc.body.children[0];
  const bodyEl = backdrop.children[0].children[1]; // panel > [header, body]
  assert.match(bodyEl.innerHTML, /Loading/);

  panel.setBody("<p>six options</p>");
  assert.equal(bodyEl.innerHTML, "<p>six options</p>");

  panel.setError("Something went wrong");
  assert.match(bodyEl.innerHTML, /Something went wrong/);
});

test("createProgramPanel: updateHeader changes title/subtitle without clearing the body", () => {
  const doc = makeDoc();
  const panel = programPanelModule.createProgramPanel({ document: doc });
  panel.open({ title: "Loading..." });
  panel.setBody("<p>stance options</p>");

  panel.updateHeader({ title: "Kessa", subtitle: "Visit Kessa's Shop" });

  const backdrop = doc.body.children[0];
  const panelEl = backdrop.children[0];
  const bodyEl = panelEl.children[1];
  assert.equal(bodyEl.innerHTML, "<p>stance options</p>", "updateHeader must not clear the body");
});
