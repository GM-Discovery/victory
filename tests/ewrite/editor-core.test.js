"use strict";

const test = require("node:test");
const assert = require("node:assert");

const core = require("../../frontend/lib/ewrite/editor-core.js");

test("applyWrap wraps a selection in bold markers", () => {
  const r = core.applyWrap("hello world", 6, 11, "**");
  assert.strictEqual(r.text, "hello **world**");
  assert.strictEqual(r.text.slice(r.selStart, r.selEnd), "world");
});

test("applyWrap toggles off an already-wrapped selection", () => {
  const r = core.applyWrap("hello **world**", 6, 15, "**");
  assert.strictEqual(r.text, "hello world");
});

test("applyWrap unwraps when markers surround the selection", () => {
  const r = core.applyWrap("hello **world**", 8, 13, "**");
  assert.strictEqual(r.text, "hello world");
  assert.strictEqual(r.text.slice(r.selStart, r.selEnd), "world");
});

test("applyHeading sets and toggles a heading level", () => {
  let r = core.applyHeading("my line", 3, 2);
  assert.strictEqual(r.text, "## my line");
  r = core.applyHeading(r.text, 4, 2);
  assert.strictEqual(r.text, "my line");
  r = core.applyHeading("### old", 2, 2);
  assert.strictEqual(r.text, "## old");
});

test("applyLink wraps the selection as a markdown link", () => {
  const r = core.applyLink("see docs here", 4, 8, "#docs");
  assert.strictEqual(r.text, "see [docs](#docs) here");
  assert.strictEqual(r.text.slice(r.selStart, r.selEnd), "docs");
});

test("applyLink inserts a template on empty selection", () => {
  const r = core.applyLink("", 0, 0, "");
  assert.strictEqual(r.text, "[link text](#)");
});

test("countWords", () => {
  assert.strictEqual(core.countWords(""), 0);
  assert.strictEqual(core.countWords("  \n "), 0);
  assert.strictEqual(core.countWords("one two\nthree"), 3);
});

test("autosaveKey scopes by publication and base revision", () => {
  assert.strictEqual(core.autosaveKey("p1", "r4"), "ewrite-draft-p1-r4");
  assert.strictEqual(core.autosaveKey("p1", ""), "ewrite-draft-p1-r0");
  assert.notStrictEqual(core.autosaveKey("p1", "r4"), core.autosaveKey("p1", "r5"));
});

test("conflict flow preserves base until user chooses, then advances", () => {
  let s = core.reduceEditorState(null, { type: "loaded", baseRevisionId: "r4" });
  s = core.reduceEditorState(s, { type: "edited" });
  assert.strictEqual(s.status, "dirty");
  s = core.reduceEditorState(s, { type: "save_started" });
  s = core.reduceEditorState(s, {
    type: "save_conflicted",
    conflict: { current_revision_id: "r5", current_revision_number: 5 },
  });
  assert.strictEqual(s.status, "conflict");
  assert.strictEqual(s.baseRevisionId, "r4", "base must not silently advance on conflict");

  // Keep-mine: base advances to the server's revision, state is dirty so
  // the user re-saves deliberately.
  const keep = core.reduceEditorState(s, { type: "conflict_keep_mine" });
  assert.strictEqual(keep.status, "dirty");
  assert.strictEqual(keep.baseRevisionId, "r5");

  // Reload-theirs: local text discarded, clean at the server revision.
  const reload = core.reduceEditorState(s, { type: "conflict_reload_theirs" });
  assert.strictEqual(reload.status, "idle");
  assert.strictEqual(reload.baseRevisionId, "r5");
});

test("successful save advances the base revision", () => {
  let s = core.reduceEditorState(null, { type: "loaded", baseRevisionId: "r4" });
  s = core.reduceEditorState(s, { type: "edited" });
  s = core.reduceEditorState(s, { type: "save_started" });
  s = core.reduceEditorState(s, { type: "save_succeeded", newRevisionId: "r5" });
  assert.strictEqual(s.status, "idle");
  assert.strictEqual(s.baseRevisionId, "r5");
});
