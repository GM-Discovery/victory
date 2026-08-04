"use strict";

const test = require("node:test");
const assert = require("node:assert");

const outline = require("../../frontend/lib/ewrite/outline.js");

test("parses headings with levels and generated anchors", () => {
  const r = outline.parseOutline("# Title\n\n## A Section Name\n\n### Sub-Section: One!");
  assert.strictEqual(r.headings.length, 3);
  assert.deepStrictEqual(
    r.headings.map((h) => h.anchor),
    ["title", "a-section-name", "sub-section-one"]
  );
  assert.deepStrictEqual(
    r.headings.map((h) => h.level),
    [1, 2, 3]
  );
});

test("preserves explicit {#anchor} ids verbatim including hostile charset", () => {
  const src = [
    "# Socio- {#socio-}",
    "## Why This Game? {#why-this-game?}",
    "## (Open System) {#(an-open-source-collaborative-tabletop-role-playing-system)}",
  ].join("\n");
  const r = outline.parseOutline(src);
  assert.deepStrictEqual(
    r.headings.map((h) => h.anchor),
    ["socio-", "why-this-game?", "(an-open-source-collaborative-tabletop-role-playing-system)"]
  );
  assert.ok(r.headings.every((h) => h.explicit));
});

test("counts spacer headings without giving them anchors", () => {
  const r = outline.parseOutline("# Title\n## \n## Real\n### ");
  assert.strictEqual(r.spacerCount, 2);
  assert.deepStrictEqual(
    r.headings.map((h) => h.title),
    ["Title", "Real"]
  );
});

test("ignores pseudo-headings inside code fences", () => {
  const r = outline.parseOutline("```\n## Not A Heading {#nope}\n```\n\n## Real {#real}");
  assert.strictEqual(r.headings.length, 1);
  assert.strictEqual(r.headings[0].anchor, "real");
});

test("deduplicates repeated headings deterministically", () => {
  const r = outline.parseOutline("## Overview\n## Overview\n## Overview");
  assert.deepStrictEqual(
    r.headings.map((h) => h.anchor),
    ["overview", "overview-2", "overview-3"]
  );
});

test("records source line numbers for editor rail jumps", () => {
  const r = outline.parseOutline("intro\n\n# One\n\ntext\n\n## Two");
  assert.deepStrictEqual(
    r.headings.map((h) => h.line),
    [3, 7]
  );
});
