// eWrite client-side outline parser (Kernel 78). Display-only convenience
// for the editor's docked rail and import preview: parses headings and
// explicit {#anchor} ids from Markdown source, tracking fenced code blocks
// so fenced pseudo-headings are ignored. The SERVER is the sole authority
// on anchors and sanitized rendering (backend/internal/ewrite/markdown.go);
// this module mirrors its recognition rules closely enough for a live rail
// but nothing security- or identity-bearing may ever depend on it.
//
// UMD-with-global (stage-runtime/editors.js pattern) so plain node --test
// can require it directly.
(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  } else {
    root.VictoryEwriteOutline = factory();
  }
})(typeof self !== "undefined" ? self : this, function () {
  "use strict";

  var HEADING_RE = /^(#{1,6})[ \t]+(.*?)[ \t]*$/;
  var EXPLICIT_ANCHOR_RE = /^(.*?)[ \t]+\{#([^}]+)\}$/;
  var FENCE_RE = /^(```|~~~)/;

  // Mirrors the server's slugifyAnchor: lowercase, spaces to hyphens, drop
  // everything outside [a-z0-9-], collapse runs.
  function slugify(title) {
    var s = String(title || "").trim().toLowerCase();
    s = s.replace(/[^a-z0-9\- _]/g, "");
    s = s.replace(/[ _]+/g, "-");
    s = s.replace(/-{2,}/g, "-");
    s = s.replace(/^-+|-+$/g, "");
    return s;
  }

  // parseOutline(source) -> { headings: [{level, title, anchor, explicit,
  // line, spacer}], spacerCount }
  function parseOutline(source) {
    var lines = String(source || "").split("\n");
    var headings = [];
    var spacerCount = 0;
    var used = Object.create(null);
    var inFence = false;
    var fenceMark = "";

    for (var i = 0; i < lines.length; i++) {
      var line = lines[i];
      var trimmedLead = line.replace(/^[ \t]+/, "");
      var fence = FENCE_RE.exec(trimmedLead);
      if (fence) {
        if (!inFence) {
          inFence = true;
          fenceMark = fence[1];
        } else if (trimmedLead.indexOf(fenceMark) === 0) {
          inFence = false;
        }
        continue;
      }
      if (inFence) continue;

      // Spacer heading: markers with no title (Google Docs export idiom).
      if (/^#{1,6}[ \t]*$/.test(line)) {
        spacerCount += 1;
        continue;
      }

      var m = HEADING_RE.exec(line);
      if (!m) continue;
      var level = m[1].length;
      var title = m[2];
      var explicit = false;
      var anchor = "";

      var em = EXPLICIT_ANCHOR_RE.exec(title);
      if (em && em[1].trim() !== "") {
        title = em[1].trim();
        anchor = em[2];
        explicit = true;
      } else {
        anchor = slugify(title);
      }
      if (!anchor) continue;

      // Deterministic -2/-3 dedupe, matching the server.
      var candidate = anchor;
      var n = 2;
      while (used[candidate]) {
        candidate = anchor + "-" + n;
        n += 1;
      }
      used[candidate] = true;

      headings.push({
        level: level,
        title: title,
        anchor: candidate,
        explicit: explicit,
        line: i + 1,
      });
    }

    return { headings: headings, spacerCount: spacerCount };
  }

  return {
    parseOutline: parseOutline,
    slugify: slugify,
  };
});
