// eWrite editor pure logic (Kernel 78): selection-wrapping formatting ops,
// word counting, localStorage autosave keys, and the save/conflict state
// reducer. No DOM access -- every function takes plain values and returns
// plain values, so the editor page stays a thin binding layer and plain
// node --test covers the behavior.
//
// UMD-with-global (stage-runtime/editors.js pattern).
(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  } else {
    root.VictoryEwriteEditorCore = factory();
  }
})(typeof self !== "undefined" ? self : this, function () {
  "use strict";

  // applyWrap(text, selStart, selEnd, marker) -> {text, selStart, selEnd}
  // Wraps the selection in symmetric markers (e.g. ** or _). If the
  // selection is already exactly wrapped, unwraps instead (toggle).
  function applyWrap(text, selStart, selEnd, marker) {
    text = String(text);
    var before = text.slice(0, selStart);
    var sel = text.slice(selStart, selEnd);
    var after = text.slice(selEnd);
    var m = marker.length;

    var wrapped = sel.length >= 2 * m && sel.slice(0, m) === marker && sel.slice(-m) === marker;
    if (wrapped) {
      var inner = sel.slice(m, sel.length - m);
      return { text: before + inner + after, selStart: selStart, selEnd: selStart + inner.length };
    }
    var surrounding = before.slice(-m) === marker && after.slice(0, m) === marker;
    if (surrounding) {
      return {
        text: before.slice(0, before.length - m) + sel + after.slice(m),
        selStart: selStart - m,
        selEnd: selEnd - m,
      };
    }
    return {
      text: before + marker + sel + marker + after,
      selStart: selStart + m,
      selEnd: selEnd + m,
    };
  }

  // applyHeading(text, selStart, level) -> {text, selStart, selEnd}
  // Sets the line containing the selection start to the given heading
  // level; same level again removes the heading (toggle).
  function applyHeading(text, selStart, level) {
    text = String(text);
    var lineStart = text.lastIndexOf("\n", selStart - 1) + 1;
    var lineEnd = text.indexOf("\n", lineStart);
    if (lineEnd === -1) lineEnd = text.length;
    var line = text.slice(lineStart, lineEnd);
    var existing = /^(#{1,6})[ \t]+/.exec(line);
    var body = existing ? line.slice(existing[0].length) : line;
    var newLine;
    if (existing && existing[1].length === level) {
      newLine = body;
    } else {
      newLine = new Array(level + 1).join("#") + " " + body;
    }
    var newText = text.slice(0, lineStart) + newLine + text.slice(lineEnd);
    var caret = lineStart + newLine.length;
    return { text: newText, selStart: caret, selEnd: caret };
  }

  // applyLink(text, selStart, selEnd, url) -> {text, selStart, selEnd}
  // [selection](url); empty selection inserts a template.
  function applyLink(text, selStart, selEnd, url) {
    text = String(text);
    var sel = text.slice(selStart, selEnd) || "link text";
    var insert = "[" + sel + "](" + (url || "#") + ")";
    return {
      text: text.slice(0, selStart) + insert + text.slice(selEnd),
      selStart: selStart + 1,
      selEnd: selStart + 1 + sel.length,
    };
  }

  function countWords(text) {
    var s = String(text || "").trim();
    if (!s) return 0;
    return s.split(/\s+/).length;
  }

  // Autosave keys are scoped per publication AND per base revision: a
  // snapshot taken against revision 4 must not be offered as a restore on
  // top of revision 6 without the user seeing the conflict flow.
  function autosaveKey(publicationId, baseRevisionId) {
    return "ewrite-draft-" + String(publicationId) + "-" + String(baseRevisionId || "r0");
  }

  // Save/conflict state reducer. States:
  //   idle -> dirty -> saving -> idle        (success)
  //                 -> conflict              (409; submitted text preserved)
  //   conflict -> saving (retry as new base after reload-theirs)
  //   conflict -> dirty  (keep-mine: base swapped to server's, still dirty)
  function reduceEditorState(state, event) {
    var s = state || { status: "idle", baseRevisionId: "", conflict: null };
    switch (event.type) {
      case "loaded":
        return { status: "idle", baseRevisionId: event.baseRevisionId || "", conflict: null };
      case "edited":
        if (s.status === "conflict") return s;
        return { status: "dirty", baseRevisionId: s.baseRevisionId, conflict: null };
      case "save_started":
        return { status: "saving", baseRevisionId: s.baseRevisionId, conflict: null };
      case "save_succeeded":
        return { status: "idle", baseRevisionId: event.newRevisionId, conflict: null };
      case "save_conflicted":
        return { status: "conflict", baseRevisionId: s.baseRevisionId, conflict: event.conflict || {} };
      case "conflict_reload_theirs":
        // User discards local text; editor reloads server content.
        return { status: "idle", baseRevisionId: (s.conflict && s.conflict.current_revision_id) || "", conflict: null };
      case "conflict_keep_mine":
        // User keeps local text and will re-save on top of the server
        // revision -- deliberate overwrite, base advanced, still dirty.
        return { status: "dirty", baseRevisionId: (s.conflict && s.conflict.current_revision_id) || "", conflict: null };
      default:
        return s;
    }
  }

  return {
    applyWrap: applyWrap,
    applyHeading: applyHeading,
    applyLink: applyLink,
    countWords: countWords,
    autosaveKey: autosaveKey,
    reduceEditorState: reduceEditorState,
  };
});
