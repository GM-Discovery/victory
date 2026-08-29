(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStageDrawing = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  "use strict";

  // Kernel 87: Cartograph shared drawing. This module owns everything
  // client-side about the drawing toolbar, tool interaction, and PIXI
  // rendering of canonical drawing objects. It is intentionally a *thin*
  // renderer over server-authoritative state: every create/edit/delete/
  // lock/z-order/settings/coordination call goes straight to the Kernel 87
  // backend (backend/internal/drawing) and nothing is optimistic beyond
  // "draw what the mouse is doing right now, then ask the server" for the
  // in-progress stroke preview -- on any create response the object is
  // re-rendered from what the server actually stored.
  //
  // Geometry is always stored/rendered in the same "world" coordinate
  // space tokens/index cards already use (frontend/lib/stage-runtime/
  // geometry.js's stagePointFromClient), so pan/zoom/fullscreen never
  // needs to rewrite a stored point (kernel-87 §2).

  const TOOLS = ["select", "freehand", "line", "polyline", "rectangle", "ellipse", "polygon", "text", "stamp", "measure"];

  const STAMP_GLYPHS = {
    settlement: "⌂",
    ruin: "⛰",
    mountain: "⛰",
    forest: "♣",
    "river-mark": "≈",
    camp: "▲",
    danger: "☠",
    treasure: "♦",
    waypoint: "⚑",
    skull: "☠",
  };

  function el(tag, attrs, children) {
    const node = document.createElement(tag);
    Object.entries(attrs || {}).forEach(([k, v]) => {
      if (k === "style" && typeof v === "object") {
        Object.assign(node.style, v);
      } else if (k === "text") {
        node.textContent = v;
      } else if (k.startsWith("on") && typeof v === "function") {
        node.addEventListener(k.slice(2), v);
      } else if (v !== undefined && v !== null) {
        node.setAttribute(k, v);
      }
    });
    (children || []).forEach((c) => c && node.appendChild(c));
    return node;
  }

  async function apiGet(url) {
    const res = await fetch(url, { credentials: "include" });
    const body = await res.json().catch(() => ({}));
    if (!res.ok || body.ok === false) {
      throw new Error(body?.data?.error || `request_failed_${res.status}`);
    }
    return body.data;
  }
  async function apiSend(method, url, payload) {
    const res = await fetch(url, {
      method,
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload || {}),
    });
    const body = await res.json().catch(() => ({}));
    if (!res.ok || body.ok === false) {
      throw new Error(body?.data?.error || `request_failed_${res.status}`);
    }
    return body.data;
  }

  function hexToPixi(hex, fallback) {
    if (typeof hex !== "string" || !/^#[0-9a-fA-F]{3,8}$/.test(hex)) return fallback;
    let h = hex.slice(1);
    if (h.length === 3) h = h.split("").map((c) => c + c).join("");
    return parseInt(h.slice(0, 6), 16);
  }

  function lineDash(g, points, dash) {
    // Minimal dashed/dotted line drawing for PIXI.Graphics (no native dash
    // API in the PIXI v7 line used elsewhere in this codebase -- kept
    // deliberately simple per kernel-87 §4 "simple solid/dashed/dotted
    // line if inexpensive," not a general stroke-style engine).
    if (points.length < 4) return;
    const segLen = dash === "dotted" ? 3 : 10;
    const gapLen = dash === "dotted" ? 6 : 8;
    for (let i = 0; i < points.length - 2; i += 2) {
      const x1 = points[i], y1 = points[i + 1], x2 = points[i + 2], y2 = points[i + 3];
      const dx = x2 - x1, dy = y2 - y1;
      const len = Math.hypot(dx, dy) || 1;
      const ux = dx / len, uy = dy / len;
      let travelled = 0;
      let draw = true;
      while (travelled < len) {
        const step = Math.min(draw ? segLen : gapLen, len - travelled);
        const sx = x1 + ux * travelled, sy = y1 + uy * travelled;
        const ex = x1 + ux * (travelled + step), ey = y1 + uy * (travelled + step);
        if (draw) {
          g.moveTo(sx, sy).lineTo(ex, ey);
        }
        travelled += step;
        draw = !draw;
      }
    }
  }

  function drawObjectGraphic(g, obj) {
    g.clear();
    const stroke = hexToPixi(obj.stroke_color, 0x2b2622);
    const fill = obj.fill_color ? hexToPixi(obj.fill_color, null) : null;
    const width = Math.max(0.5, Number(obj.stroke_width) || 2);
    const alpha = Math.min(1, Math.max(0, Number(obj.opacity ?? 1)));
    const dashed = obj.line_style === "dashed" || obj.line_style === "dotted";

    g.alpha = alpha;
    g.rotation = ((Number(obj.rotation) || 0) * Math.PI) / 180;

    const pts = Array.isArray(obj.geometry?.points) ? obj.geometry.points : [];
    switch (obj.object_type) {
      case "freehand":
      case "polyline": {
        if (dashed) {
          g.lineStyle(0);
          g.lineStyle(width, stroke, 1);
          const flat = [];
          pts.forEach((p) => flat.push(p.x, p.y));
          lineDash(g, flat, obj.line_style);
        } else {
          g.lineStyle(width, stroke, 1);
          pts.forEach((p, i) => (i === 0 ? g.moveTo(p.x, p.y) : g.lineTo(p.x, p.y)));
        }
        break;
      }
      case "line": {
        g.lineStyle(width, stroke, 1);
        if (dashed) {
          lineDash(g, [pts[0]?.x || 0, pts[0]?.y || 0, pts[1]?.x || 0, pts[1]?.y || 0], obj.line_style);
        } else if (pts[0] && pts[1]) {
          g.moveTo(pts[0].x, pts[0].y).lineTo(pts[1].x, pts[1].y);
        }
        break;
      }
      case "polygon": {
        g.lineStyle(width, stroke, 1);
        if (fill !== null) g.beginFill(fill, 1);
        pts.forEach((p, i) => (i === 0 ? g.moveTo(p.x, p.y) : g.lineTo(p.x, p.y)));
        g.closePath();
        if (fill !== null) g.endFill();
        break;
      }
      case "rectangle": {
        const { x = 0, y = 0, width: w = 1, height: h = 1 } = obj.geometry || {};
        g.lineStyle(width, stroke, 1);
        if (fill !== null) g.beginFill(fill, 1);
        g.drawRect(x, y, w, h);
        if (fill !== null) g.endFill();
        break;
      }
      case "ellipse": {
        const { x = 0, y = 0, width: w = 1, height: h = 1 } = obj.geometry || {};
        g.lineStyle(width, stroke, 1);
        if (fill !== null) g.beginFill(fill, 1);
        g.drawEllipse(x + w / 2, y + h / 2, Math.abs(w) / 2, Math.abs(h) / 2);
        if (fill !== null) g.endFill();
        break;
      }
      case "stamp": {
        const { x = 0, y = 0, scale = 1 } = obj.geometry || {};
        g.lineStyle(Math.max(1, width / 2), stroke, 1);
        if (fill !== null) g.beginFill(fill, 1); else g.beginFill(0xffffff, 0.85);
        g.drawCircle(x, y, 18 * scale);
        g.endFill();
        break;
      }
      default:
        break;
    }
  }

  function mount(options) {
    const {
      hostElement,
      worldLayer,
      pixiApp,
      getSessionId,
      getShowId,
      getViewerRole,
      getWorldBounds,
      stagePointFromClient,
      getUserId,
      getCamera,
      // Kernel 87 toolbar-placement fix: the toolbar panel is docked into
      // the same DOM overlay layer the Map/Grid/Card/Token editor panels
      // already use (#overlay-root, position:fixed inset:0), not injected
      // directly into the stage/canvas host -- so it floats as a
      // draggable, collapsible card alongside those editors instead of
      // sitting fixed on top of the map (kernel-87 §19 "map remains
      // visually dominant"). panelHost/dragBoundsElement are optional so
      // callers/tests that don't pass them still get the old
      // stage-host-anchored behavior rather than a null-ref crash.
      panelHost,
      dragBoundsElement,
    } = options;

    if (!hostElement || !worldLayer || !pixiApp) return null;

    const state = {
      tool: "select",
      objects: [],
      selectedId: null,
      strokeColor: "#2b2622",
      fillColor: "",
      useFill: false,
      strokeWidth: 3,
      opacity: 1,
      lineStyle: "solid",
      scope: "cohort",
      settings: null,
      coordination: null,
      canDraw: false,
      pendingPoints: [],
      dragState: null,
      detailView: null, // { previousView, rect }
      measurePoints: [],
      stampKey: "settlement",
    };

    const layerNode = new PIXI.Container();
    layerNode.sortableChildren = true;
    worldLayer.addChild(layerNode);
    const graphicById = new Map();
    const selectionOverlay = new PIXI.Graphics();
    selectionOverlay.zIndex = 999;
    layerNode.addChild(selectionOverlay);

    const measureLayer = new PIXI.Graphics();
    measureLayer.zIndex = 998;
    layerNode.addChild(measureLayer);

    // --- Toolbar DOM -------------------------------------------------
    // Docked panel styling matches the established .map-editor/.card-editor
    // convention (see frontend/venues/catharsis/index.html): a translucent
    // dark rounded card, positioned absolute within the fixed overlay
    // layer (not the stage/canvas host), with a drag-by-header grip. CSS
    // class does the heavy styling (see .cartograph-toolbar in the venue
    // stylesheet); inline style here only sets initial position/z-index/
    // display so the module still degrades gracefully with no stylesheet.
    // Position/collapse memory (live-testing feedback, 2026-08-28): the
    // panel used to always reopen at the same top:16px/right:16px slot,
    // uncollapsed, regardless of anything the user did last session.
    // localStorage, matching this app's existing convention for this class
    // of cosmetic per-browser preference (reactions.js's own prefs).
    const CARTOGRAPH_PREFS_KEY = "cartographToolbarPrefs";

    function parsePx(value) {
      const n = Number.parseFloat(value);
      return Number.isFinite(n) ? n : null;
    }

    function loadCartographPrefs() {
      try {
        const raw = window.localStorage?.getItem(CARTOGRAPH_PREFS_KEY);
        if (!raw) return { collapsed: true, left: null, top: null };
        const parsed = JSON.parse(raw);
        return {
          // Only ever defaults to collapsed=true when nothing was saved yet
          // (first-ever use, spec: "start compacted as its initial default
          // state") -- once saved, an explicit false is honored same as true.
          collapsed: typeof parsed.collapsed === "boolean" ? parsed.collapsed : true,
          left: Number.isFinite(parsed.left) ? parsed.left : null,
          top: Number.isFinite(parsed.top) ? parsed.top : null,
        };
      } catch {
        return { collapsed: true, left: null, top: null };
      }
    }

    function saveCartographPrefs(prefs) {
      try {
        window.localStorage?.setItem(CARTOGRAPH_PREFS_KEY, JSON.stringify(prefs));
      } catch {
        // Private-browsing / storage-disabled: position/collapse just don't
        // persist across reloads -- not worth surfacing as an error.
      }
    }

    const cartographPrefs = loadCartographPrefs();

    const mountHost = panelHost || hostElement;
    const panel = el("div", {
      class: "cartograph-toolbar",
      style: {
        position: "absolute", top: "16px", right: "16px", zIndex: "18",
        display: "none",
      },
    });
    if (mountHost === hostElement) {
      hostElement.style.position = hostElement.style.position || "relative";
    }
    mountHost.appendChild(panel);

    if (cartographPrefs.left != null && cartographPrefs.top != null) {
      panel.style.left = `${cartographPrefs.left}px`;
      panel.style.top = `${cartographPrefs.top}px`;
      panel.style.right = "auto";
    }

    const header = el("div", { class: "cartograph-toolbar-header" }, [
      el("strong", { text: "Cartography" }),
      el("span", { class: "cartograph-toolbar-grip", text: "drag" }),
    ]);
    const collapseBtn = el("button", {
      type: "button", class: "cartograph-toolbar-collapse", title: "Collapse/expand the drawing toolbar",
      text: "−",
    });
    header.appendChild(collapseBtn);
    panel.appendChild(header);

    const body = el("div", { class: "cartograph-toolbar-body" });
    panel.appendChild(body);

    let collapsed = cartographPrefs.collapsed;
    body.style.display = collapsed ? "none" : "";
    collapseBtn.textContent = collapsed ? "□" : "−";
    collapseBtn.addEventListener("click", () => {
      collapsed = !collapsed;
      body.style.display = collapsed ? "none" : "";
      collapseBtn.textContent = collapsed ? "□" : "−";
      saveCartographPrefs({ collapsed, left: parsePx(panel.style.left), top: parsePx(panel.style.top) });
    });

    // Default position (only when nothing was ever dragged/saved before):
    // sit immediately left of the Director Tools/Aftercare row
    // (#kernel89-director-toolbar, itself fixed at top:12px/right:12px)
    // instead of the old top:16px/right:16px slot, which sat almost
    // exactly on top of it. Computed lazily on first real show (from
    // renderAll below), not at module-init, so the Director toolbar's own
    // role-gated visibility has had a chance to resolve. Falls back to
    // that same top:12px/right:12px slot itself if Director Tools isn't
    // visible for this viewer at all (e.g. Cast, who can use Cartography
    // without Director Tools existing on their screen).
    let defaultPositionResolved = cartographPrefs.left != null;
    function applyDefaultPositionIfNeeded() {
      if (defaultPositionResolved) return;
      defaultPositionResolved = true;
      const directorToolbar = document.getElementById("kernel89-director-toolbar");
      const visible = directorToolbar && directorToolbar.offsetParent !== null;
      panel.style.left = "auto";
      if (visible) {
        const rect = directorToolbar.getBoundingClientRect();
        panel.style.top = `${Math.round(rect.top)}px`;
        panel.style.right = `${Math.round(window.innerWidth - rect.left + 8)}px`;
      } else {
        panel.style.top = "12px";
        panel.style.right = "12px";
      }
    }

    // Drag-by-header, matching the offsetX/offsetY pointer-capture pattern
    // runtime.js already uses for the Map/Grid/Card/Token editor panels
    // (beginMapEditorDrag et al.), kept self-contained here rather than
    // adding another copy of that per-panel wiring to runtime.js.
    let panelDrag = null;
    header.addEventListener("pointerdown", (evt) => {
      if (evt.button !== 0 || evt.target === collapseBtn) return;
      evt.preventDefault();
      const rect = panel.getBoundingClientRect();
      panelDrag = { offsetX: evt.clientX - rect.left, offsetY: evt.clientY - rect.top };
      header.setPointerCapture?.(evt.pointerId);
      header.style.cursor = "grabbing";
    });
    header.addEventListener("pointermove", (evt) => {
      if (!panelDrag) return;
      const boundsEl = dragBoundsElement || mountHost;
      const boundsRect = boundsEl.getBoundingClientRect?.() || { left: 0, top: 0 };
      panel.style.left = `${evt.clientX - boundsRect.left - panelDrag.offsetX}px`;
      panel.style.top = `${evt.clientY - boundsRect.top - panelDrag.offsetY}px`;
      panel.style.right = "auto";
    });
    ["pointerup", "pointercancel"].forEach((type) => header.addEventListener(type, () => {
      if (panelDrag) {
        saveCartographPrefs({ collapsed, left: parsePx(panel.style.left), top: parsePx(panel.style.top) });
      }
      panelDrag = null;
      header.style.cursor = "move";
    }));

    const toolRow = el("div", { style: { display: "flex", flexWrap: "wrap", gap: "4px", marginBottom: "6px" } });
    const toolButtons = {};
    TOOLS.forEach((tool) => {
      const btn = el("button", {
        text: tool[0].toUpperCase() + tool.slice(1),
        style: { padding: "3px 6px", fontSize: "11px", cursor: "pointer", borderRadius: "5px", border: "1px solid #6b5c4c", background: "#2e2721", color: "#f1e9dc" },
        onclick: () => setTool(tool),
      });
      toolButtons[tool] = btn;
      toolRow.appendChild(btn);
    });
    body.appendChild(toolRow);

    const styleRow = el("div", { style: { display: "flex", flexWrap: "wrap", gap: "6px", alignItems: "center", marginBottom: "6px" } });
    const strokeInput = el("input", { type: "color", value: state.strokeColor, title: "Stroke color", onchange: (e) => { state.strokeColor = e.target.value; } });
    const fillToggle = el("input", { type: "checkbox", title: "Fill", onchange: (e) => { state.useFill = e.target.checked; } });
    const fillInput = el("input", { type: "color", value: "#8899aa", title: "Fill color", onchange: (e) => { state.fillColor = e.target.value; } });
    const widthInput = el("input", { type: "range", min: "1", max: "40", value: String(state.strokeWidth), title: "Width", onchange: (e) => { state.strokeWidth = Number(e.target.value); } });
    const opacityInput = el("input", { type: "range", min: "0.1", max: "1", step: "0.05", value: String(state.opacity), title: "Opacity", onchange: (e) => { state.opacity = Number(e.target.value); } });
    const lineStyleSelect = el("select", { onchange: (e) => { state.lineStyle = e.target.value; } }, [
      el("option", { value: "solid", text: "Solid" }),
      el("option", { value: "dashed", text: "Dashed" }),
      el("option", { value: "dotted", text: "Dotted" }),
    ]);
    styleRow.appendChild(el("label", { text: "Stroke " }, [strokeInput]));
    styleRow.appendChild(el("label", { text: "Fill " }, [fillToggle, fillInput]));
    styleRow.appendChild(el("label", { text: "Width " }, [widthInput]));
    styleRow.appendChild(el("label", { text: "Opacity " }, [opacityInput]));
    styleRow.appendChild(lineStyleSelect);
    body.appendChild(styleRow);

    const stampRow = el("div", { style: { display: "none", flexWrap: "wrap", gap: "4px", marginBottom: "6px" } });
    Object.keys(STAMP_GLYPHS).forEach((key) => {
      stampRow.appendChild(el("button", {
        text: STAMP_GLYPHS[key] + " " + key,
        style: { fontSize: "10px", padding: "2px 4px", cursor: "pointer" },
        onclick: () => { state.stampKey = key; },
      }));
    });
    body.appendChild(stampRow);

    const scopeRow = el("div", { style: { display: "flex", gap: "6px", marginBottom: "6px", alignItems: "center" } });
    const scopeSelect = el("select", { onchange: (e) => { state.scope = e.target.value; } }, [
      el("option", { value: "cohort", text: "Scope: Cohort" }),
      el("option", { value: "show", text: "Scope: Show" }),
    ]);
    scopeRow.appendChild(scopeSelect);
    body.appendChild(scopeRow);

    const actionRow = el("div", { style: { display: "flex", flexWrap: "wrap", gap: "4px", marginBottom: "6px" } });
    const btnDuplicate = el("button", { text: "Duplicate", onclick: () => duplicateSelected() });
    const btnDelete = el("button", { text: "Delete", onclick: () => deleteSelected() });
    const btnLock = el("button", { text: "Lock/Unlock", onclick: () => toggleLockSelected() });
    const btnFront = el("button", { text: "↑ Front", onclick: () => reorderSelected("front") });
    const btnForward = el("button", { text: "↑ Fwd", onclick: () => reorderSelected("forward") });
    const btnBackward = el("button", { text: "↓ Back", onclick: () => reorderSelected("backward") });
    const btnBack = el("button", { text: "↓ Bottom", onclick: () => reorderSelected("back") });
    [btnDuplicate, btnDelete, btnLock, btnFront, btnForward, btnBackward, btnBack].forEach((b) => {
      Object.assign(b.style, { padding: "3px 6px", fontSize: "11px", cursor: "pointer", borderRadius: "5px", border: "1px solid #6b5c4c", background: "#2e2721", color: "#f1e9dc" });
      actionRow.appendChild(b);
    });
    body.appendChild(actionRow);

    const detailRow = el("div", { style: { display: "flex", gap: "4px", marginBottom: "6px" } });
    const btnDetailOpen = el("button", { text: "Open Detail View (select region first)", style: { fontSize: "11px" }, onclick: () => openDetailViewFromSelection() });
    const btnDetailClose = el("button", { text: "Close Detail View", style: { fontSize: "11px", display: "none" }, onclick: () => closeDetailView() });
    detailRow.appendChild(btnDetailOpen);
    detailRow.appendChild(btnDetailClose);
    body.appendChild(detailRow);

    const exportRow = el("div", { style: { display: "flex", gap: "4px", marginBottom: "4px" } });
    const btnExport = el("button", { text: "Export Map PNG", onclick: () => exportPNG() });
    exportRow.appendChild(btnExport);
    body.appendChild(exportRow);

    const statusLine = el("div", { style: { fontSize: "10px", opacity: "0.8", marginTop: "4px" } });
    body.appendChild(statusLine);

    function setStatus(text) { statusLine.textContent = text || ""; }

    function highlightTool() {
      TOOLS.forEach((t) => {
        toolButtons[t].style.outline = t === state.tool ? "2px solid #d9c7a6" : "none";
      });
      stampRow.style.display = state.tool === "stamp" ? "flex" : "none";
    }

    function setTool(tool) {
      state.tool = tool;
      state.pendingPoints = [];
      state.measurePoints = [];
      highlightTool();
      redrawMeasure();
    }

    // --- Data sync -----------------------------------------------------
    async function refresh() {
      const sessionId = getSessionId();
      if (!sessionId) return;
      try {
        const [objData, settingsData, coordData] = await Promise.all([
          apiGet(`/api/sessions/${encodeURIComponent(sessionId)}/drawing-objects`),
          getShowId() ? apiGet(`/api/shows/${encodeURIComponent(getShowId())}/drawing-settings`) : Promise.resolve(null),
          apiGet(`/api/sessions/${encodeURIComponent(sessionId)}/drawing-coordination/current-turn`).catch(() => null),
        ]);
        state.objects = objData?.objects || [];
        state.settings = settingsData?.settings || null;
        state.coordination = coordData?.coordination || null;
        computeCanDraw();
        renderAll();
      } catch (err) {
        setStatus("Cartograph: " + err.message);
      }
    }

    function computeCanDraw() {
      const role = String(getViewerRole() || "").toLowerCase();
      if (role === "director" || role === "producer") {
        state.canDraw = true;
        return;
      }
      const mode = state.settings?.drawing_mode || "director_only";
      if (mode === "director_only") {
        state.canDraw = false;
      } else if (mode === "freeform") {
        state.canDraw = role !== "audience";
      } else if (mode === "turn_leader") {
        // Client-side hint only -- the server re-checks authority on every
        // write regardless (kernel-87 §8), so a stale coordination read
        // here only affects whether buttons *look* enabled, never whether
        // a write actually succeeds.
        const myID = String(getUserId?.() || "");
        const c = state.coordination || {};
        state.canDraw = !!myID && (c.group_leader_user_id === myID || c.current_turn_user_id === myID);
      } else {
        state.canDraw = false;
      }
      panel.style.opacity = state.canDraw || role === "director" || role === "producer" ? "1" : "0.55";
    }

    function renderAll() {
      panel.style.display = "block";
      applyDefaultPositionIfNeeded();
      const seen = new Set();
      state.objects
        .slice()
        .sort((a, b) => a.z_order - b.z_order)
        .forEach((obj, index) => {
          seen.add(obj.id);
          let g = graphicById.get(obj.id);
          if (!g) {
            g = new PIXI.Graphics();
            g.eventMode = "static";
            g.cursor = "pointer";
            g.on("pointerdown", (evt) => {
              evt.stopPropagation();
              if (state.tool === "select") selectObject(obj.id);
            });
            graphicById.set(obj.id, g);
            layerNode.addChild(g);
          }
          g.zIndex = index;
          g.__obj = obj;
          drawObjectGraphic(g, obj);
          // Rotate around the object's own bounding-box center, not the
          // world origin: geometry is drawn in absolute world coordinates
          // inside each per-object Graphics, so an un-pivoted g.rotation
          // would orbit the object around (0,0) instead of spinning it in
          // place. Setting pivot == position to the bounds center cancels
          // the translation while making that point the rotation origin.
          const b = objectBounds(obj);
          if (b) {
            const cx = b.x + b.width / 2, cy = b.y + b.height / 2;
            g.pivot.set(cx, cy);
            g.position.set(cx, cy);
            // Explicit hit area, not PIXI's default fill-based hit-test:
            // an unfilled shape (no fill_color -- the common case for a
            // stroke-only rectangle/ellipse/polygon) otherwise only
            // accepts pointer events exactly on its outline pixels, so
            // clicking anywhere in the shape's visible interior silently
            // fails to select it. Padded slightly so thin freehand/line
            // strokes and small stamps/text stay comfortably clickable.
            const pad = 6;
            g.hitArea = new PIXI.Rectangle(b.x - pad, b.y - pad, Math.max(1, b.width + pad * 2), Math.max(1, b.height + pad * 2));
          }
        });
      // Remove graphics for deleted objects.
      Array.from(graphicById.keys()).forEach((id) => {
        if (!seen.has(id)) {
          const g = graphicById.get(id);
          layerNode.removeChild(g);
          g.destroy();
          graphicById.delete(id);
        }
      });
      layerNode.sortChildren();
      drawSelectionOverlay();
    }

    function selectObject(id) {
      state.selectedId = id;
      const obj = state.objects.find((o) => o.id === id);
      if (obj) {
        strokeInput.value = obj.stroke_color || "#2b2622";
        if (obj.fill_color) { fillInput.value = obj.fill_color; fillToggle.checked = true; state.useFill = true; }
      }
      drawSelectionOverlay();
    }

    function drawSelectionOverlay(previewBounds) {
      selectionOverlay.clear();
      const obj = state.objects.find((o) => o.id === state.selectedId);
      if (!obj) return;
      const bounds = previewBounds || objectBounds(obj);
      if (!bounds) return;
      selectionOverlay.lineStyle(1.5, 0xffcc66, 0.9);
      selectionOverlay.drawRect(bounds.x - 4, bounds.y - 4, bounds.width + 8, bounds.height + 8);
      // Move/resize/rotate handles: two small handles (drag semantics
      // wired via handleHitTest + onPointerDown/Move/Up above). Kept
      // intentionally simple (single resize handle, single rotate handle)
      // per kernel-87 §3 "No node-by-node Bezier editing" and the effort
      // spent instead on correctness of server authority.
      selectionOverlay.beginFill(0xffcc66, 1);
      selectionOverlay.drawRect(bounds.x + bounds.width - 2, bounds.y + bounds.height - 2, 8, 8); // resize
      selectionOverlay.drawCircle(bounds.x + bounds.width / 2, bounds.y - 16, 5); // rotate
      selectionOverlay.endFill();
    }

    function objectBounds(obj) {
      const pts = Array.isArray(obj.geometry?.points) ? obj.geometry.points : null;
      if (pts && pts.length) {
        const xs = pts.map((p) => p.x), ys = pts.map((p) => p.y);
        return { x: Math.min(...xs), y: Math.min(...ys), width: Math.max(...xs) - Math.min(...xs), height: Math.max(...ys) - Math.min(...ys) };
      }
      const g = obj.geometry || {};
      if (obj.object_type === "stamp") {
        return { x: (g.x || 0) - 18, y: (g.y || 0) - 18, width: 36, height: 36 };
      }
      if (obj.object_type === "text") {
        return { x: g.x || 0, y: (g.y || 0) - 10, width: 80, height: 20 };
      }
      return { x: g.x || 0, y: g.y || 0, width: g.width || 1, height: g.height || 1 };
    }

    // --- Pointer interaction --------------------------------------------
    function worldPointFromEvent(evt) {
      const p = stagePointFromClient(evt.clientX, evt.clientY);
      return { x: p.x, y: p.y };
    }

    function handleHitTest(p) {
      // Hit-test the resize (bottom-right square) and rotate (circle above
      // top-center) handles drawn by drawSelectionOverlay for the current
      // selection. Kept as a simple radius/box check against the same
      // bounds math used to draw them, rather than a PIXI interactive
      // child, so it works uniformly across all object types.
      const obj = state.objects.find((o) => o.id === state.selectedId);
      if (!obj || obj.locked) return null;
      const bounds = objectBounds(obj);
      if (!bounds) return null;
      const resizeX = bounds.x + bounds.width + 2, resizeY = bounds.y + bounds.height + 2;
      if (Math.hypot(p.x - resizeX, p.y - resizeY) <= 10) return { mode: "resize", obj, bounds };
      const rotateX = bounds.x + bounds.width / 2, rotateY = bounds.y - 16;
      if (Math.hypot(p.x - rotateX, p.y - rotateY) <= 10) return { mode: "rotate", obj, bounds };
      return null;
    }

    function onPointerDown(evt) {
      if (state.tool === "select") {
        if (state.selectedId && !state.canDraw) return;
        const p = worldPointFromEvent(evt);
        const hit = handleHitTest(p);
        if (hit && hit.mode === "resize") {
          state.dragState = {
            mode: "resize", obj: hit.obj, startBounds: hit.bounds,
            startGeometry: JSON.parse(JSON.stringify(hit.obj.geometry || {})),
            startScale: hit.obj.geometry?.scale, start: p,
          };
        } else if (hit && hit.mode === "rotate") {
          const cx = hit.bounds.x + hit.bounds.width / 2, cy = hit.bounds.y + hit.bounds.height / 2;
          const startAngle = Math.atan2(p.y - cy, p.x - cx);
          state.dragState = {
            mode: "rotate", obj: hit.obj, center: { x: cx, y: cy },
            startAngle, startRotation: Number(hit.obj.rotation) || 0,
          };
        }
        return; // plain selection is handled per-object via each Graphics' own pointerdown
      }
      if (!state.canDraw && state.tool !== "measure") {
        setStatus("You are not authorized to draw right now.");
        return;
      }
      const p = worldPointFromEvent(evt);
      if (state.tool === "measure") {
        state.measurePoints.push(p);
        redrawMeasure();
        return;
      }
      if (state.tool === "freehand") {
        state.pendingPoints = [p];
        state.dragState = { drawing: true };
        return;
      }
      if (state.tool === "line") {
        state.pendingPoints.push(p);
        if (state.pendingPoints.length === 2) finishShape("line", { points: state.pendingPoints });
        return;
      }
      if (state.tool === "polyline" || state.tool === "polygon") {
        state.pendingPoints.push(p);
        redrawPending();
        return;
      }
      if (state.tool === "rectangle" || state.tool === "ellipse") {
        state.pendingPoints = [p];
        state.dragState = { drawing: true };
        return;
      }
      if (state.tool === "text") {
        const text = window.prompt("Label text (max 500 chars):", "");
        if (text && text.trim()) {
          finishShape("text", { x: p.x, y: p.y }, { text_content: text.trim() });
        }
        return;
      }
      if (state.tool === "stamp") {
        finishShape("stamp", { x: p.x, y: p.y, scale: 1 }, { stamp_key: state.stampKey });
        return;
      }
    }

    function onPointerMove(evt) {
      if (!state.dragState) return;
      const p = worldPointFromEvent(evt);
      if (state.dragState.mode === "resize" || state.dragState.mode === "rotate") {
        state.dragState.current = p;
        drawSelectionOverlay(previewBoundsFor(state.dragState, p));
        return;
      }
      if (!state.dragState.drawing) return;
      if (state.tool === "freehand") {
        const last = state.pendingPoints[state.pendingPoints.length - 1];
        if (!last || Math.hypot(p.x - last.x, p.y - last.y) > 2) {
          state.pendingPoints.push(p);
          if (state.pendingPoints.length > 2000) state.pendingPoints.shift(); // client-side mirror of MaxPointsPerStroke
          redrawPending();
        }
      } else if (state.tool === "rectangle" || state.tool === "ellipse") {
        state.dragState.current = p;
        redrawPending(true);
      }
    }

    function previewBoundsFor(dragState, p) {
      if (dragState.mode === "resize") {
        const b = dragState.startBounds;
        return { x: b.x, y: b.y, width: Math.max(4, p.x - b.x), height: Math.max(4, p.y - b.y) };
      }
      return null; // rotate preview keeps the same bounds box; only the handle position implies angle
    }

    async function onPointerUp() {
      if (state.dragState?.mode === "resize") {
        const { obj, startBounds, startGeometry, current } = state.dragState;
        state.dragState = null;
        if (!current) { drawSelectionOverlay(); return; }
        const newWidth = Math.max(4, current.x - startBounds.x);
        const newHeight = Math.max(4, current.y - startBounds.y);
        const scaleX = startBounds.width > 0 ? newWidth / startBounds.width : 1;
        const scaleY = startBounds.height > 0 ? newHeight / startBounds.height : 1;
        const geometry = JSON.parse(JSON.stringify(startGeometry));
        if (Array.isArray(geometry.points)) {
          geometry.points = geometry.points.map((pt) => ({
            x: startBounds.x + (pt.x - startBounds.x) * scaleX,
            y: startBounds.y + (pt.y - startBounds.y) * scaleY,
          }));
        } else if (obj.object_type === "stamp") {
          geometry.scale = Math.max(0.1, (geometry.scale || 1) * ((scaleX + scaleY) / 2));
        } else if ("width" in geometry || "height" in geometry) {
          geometry.width = newWidth;
          geometry.height = newHeight;
        } else {
          drawSelectionOverlay();
          return; // e.g. text: synthetic bounds only, nothing real to resize
        }
        await persistGeometry(obj, geometry);
        return;
      }
      if (state.dragState?.mode === "rotate") {
        const { obj, center, startAngle, startRotation, current } = state.dragState;
        state.dragState = null;
        if (!current) { drawSelectionOverlay(); return; }
        const angle = Math.atan2(current.y - center.y, current.x - center.x);
        const deltaDeg = ((angle - startAngle) * 180) / Math.PI;
        const rotation = ((startRotation + deltaDeg) % 360 + 360) % 360;
        await persistRotation(obj, rotation);
        return;
      }
      if (state.tool === "freehand" && state.dragState?.drawing) {
        state.dragState = null;
        if (state.pendingPoints.length >= 2) {
          finishShape("freehand", { points: state.pendingPoints });
        } else {
          state.pendingPoints = [];
          redrawPending();
        }
      } else if ((state.tool === "rectangle" || state.tool === "ellipse") && state.dragState?.drawing) {
        const start = state.pendingPoints[0];
        const end = state.dragState.current || start;
        state.dragState = null;
        state.pendingPoints = [];
        if (start && end) {
          const x = Math.min(start.x, end.x), y = Math.min(start.y, end.y);
          const width = Math.abs(end.x - start.x) || 1, height = Math.abs(end.y - start.y) || 1;
          finishShape(state.tool, { x, y, width, height });
        }
        redrawPending();
      }
    }

    async function persistGeometry(obj, geometry) {
      const sessionId = getSessionId();
      if (!sessionId) return;
      try {
        await apiSend("PATCH", `/api/sessions/${encodeURIComponent(sessionId)}/drawing-objects/${encodeURIComponent(obj.id)}`, { session_id: sessionId, geometry });
        await refresh();
        setStatus("Resized.");
      } catch (err) {
        setStatus("Resize failed: " + err.message);
      }
    }

    async function persistRotation(obj, rotation) {
      const sessionId = getSessionId();
      if (!sessionId) return;
      try {
        await apiSend("PATCH", `/api/sessions/${encodeURIComponent(sessionId)}/drawing-objects/${encodeURIComponent(obj.id)}`, { session_id: sessionId, rotation });
        await refresh();
        setStatus("Rotated.");
      } catch (err) {
        setStatus("Rotate failed: " + err.message);
      }
    }

    const pendingGraphic = new PIXI.Graphics();
    pendingGraphic.zIndex = 997;
    layerNode.addChild(pendingGraphic);

    function redrawPending(rectDrag) {
      pendingGraphic.clear();
      pendingGraphic.lineStyle(Math.max(1, state.strokeWidth), hexToPixi(state.strokeColor, 0x2b2622), 0.8);
      if (rectDrag && state.dragState?.current) {
        const start = state.pendingPoints[0];
        const end = state.dragState.current;
        const x = Math.min(start.x, end.x), y = Math.min(start.y, end.y);
        const w = Math.abs(end.x - start.x), h = Math.abs(end.y - start.y);
        if (state.tool === "ellipse") pendingGraphic.drawEllipse(x + w / 2, y + h / 2, w / 2, h / 2);
        else pendingGraphic.drawRect(x, y, w, h);
        return;
      }
      const pts = state.pendingPoints;
      pts.forEach((p, i) => (i === 0 ? pendingGraphic.moveTo(p.x, p.y) : pendingGraphic.lineTo(p.x, p.y)));
    }

    function redrawMeasure() {
      measureLayer.clear();
      const pts = state.measurePoints;
      if (pts.length < 1) return;
      measureLayer.lineStyle(2, 0x38b6ff, 0.9);
      pts.forEach((p, i) => (i === 0 ? measureLayer.moveTo(p.x, p.y) : measureLayer.lineTo(p.x, p.y)));
      pts.forEach((p) => {
        measureLayer.beginFill(0x38b6ff, 1);
        measureLayer.drawCircle(p.x, p.y, 3);
        measureLayer.endFill();
      });
      if (pts.length >= 2 && state.settings) {
        const dist = pathDistance(pts, state.settings);
        setStatus(`Measured: ${dist.toFixed(1)} ${state.settings.scale_unit_label} (${pts.length - 1} segment${pts.length > 2 ? "s" : ""})`);
      }
    }

    function pathDistance(pts, settings) {
      // Mirrors backend/internal/drawing/measurement.go's square-grid
      // diagonal policy math client-side for instant feedback; the
      // server value (if ever surfaced through a future pinned-
      // measurement endpoint) is the canonical one -- this is display-
      // only, ephemeral by default (kernel-87 §7).
      let total = 0;
      for (let i = 1; i < pts.length; i++) {
        const dx = pts[i].x - pts[i - 1].x, dy = pts[i].y - pts[i - 1].y;
        const cellsX = Math.abs(dx) / 50, cellsY = Math.abs(dy) / 50; // 50px/cell fallback grid assumption
        let cells;
        if (settings.diagonal_policy === "euclidean") cells = Math.hypot(cellsX, cellsY);
        else if (settings.diagonal_policy === "every_diagonal_1") cells = Math.max(cellsX, cellsY);
        else {
          const straight = Math.abs(cellsX - cellsY);
          const diagonal = Math.min(cellsX, cellsY);
          cells = straight + diagonal + Math.floor(diagonal / 2);
        }
        total += (cells / (settings.scale_grid_units || 1)) * (settings.scale_real_units || 1);
      }
      return total;
    }

    async function finishShape(objectType, geometry, extra) {
      const sessionId = getSessionId();
      if (!sessionId) return;
      const payload = Object.assign({
        session_id: sessionId,
        scope: state.scope,
        object_type: objectType,
        geometry,
        stroke_color: state.strokeColor,
        fill_color: state.useFill ? fillInput.value : "",
        stroke_width: state.strokeWidth,
        opacity: state.opacity,
        line_style: state.lineStyle,
        rotation: 0,
      }, extra || {});
      state.pendingPoints = [];
      redrawPending();
      try {
        await apiSend("POST", `/api/sessions/${encodeURIComponent(sessionId)}/drawing-objects`, payload);
        await refresh();
      } catch (err) {
        setStatus("Create failed: " + err.message);
      }
    }

    function finishPolyShape() {
      if (state.pendingPoints.length < (state.tool === "polygon" ? 3 : 2)) {
        setStatus(`Need at least ${state.tool === "polygon" ? 3 : 2} points.`);
        return;
      }
      finishShape(state.tool, { points: state.pendingPoints });
    }

    async function duplicateSelected() {
      const obj = state.objects.find((o) => o.id === state.selectedId);
      if (!obj) return;
      const geometry = JSON.parse(JSON.stringify(obj.geometry || {}));
      if (Array.isArray(geometry.points)) geometry.points = geometry.points.map((p) => ({ x: p.x + 20, y: p.y + 20 }));
      else { geometry.x = (geometry.x || 0) + 20; geometry.y = (geometry.y || 0) + 20; }
      await finishShape(obj.object_type, geometry, {
        text_content: obj.text_content, stamp_key: obj.stamp_key,
      });
    }

    async function deleteSelected() {
      const sessionId = getSessionId();
      if (!state.selectedId || !sessionId) return;
      try {
        await apiSend("DELETE", `/api/sessions/${encodeURIComponent(sessionId)}/drawing-objects/${encodeURIComponent(state.selectedId)}`, {});
        state.selectedId = null;
        await refresh();
      } catch (err) {
        setStatus("Delete failed: " + err.message);
      }
    }

    async function toggleLockSelected() {
      const sessionId = getSessionId();
      const obj = state.objects.find((o) => o.id === state.selectedId);
      if (!obj || !sessionId) return;
      try {
        await apiSend("POST", `/api/sessions/${encodeURIComponent(sessionId)}/drawing-objects/${encodeURIComponent(obj.id)}/lock`, { locked: !obj.locked });
        await refresh();
      } catch (err) {
        setStatus("Lock failed: " + err.message);
      }
    }

    async function reorderSelected(direction) {
      const sessionId = getSessionId();
      if (!state.selectedId || !sessionId) return;
      try {
        await apiSend("POST", `/api/sessions/${encodeURIComponent(sessionId)}/drawing-objects/${encodeURIComponent(state.selectedId)}/z-order`, { direction });
        await refresh();
      } catch (err) {
        setStatus("Reorder failed: " + err.message);
      }
    }

    async function moveSelected(dx, dy) {
      const sessionId = getSessionId();
      const obj = state.objects.find((o) => o.id === state.selectedId);
      if (!obj || !sessionId || obj.locked) return;
      const geometry = JSON.parse(JSON.stringify(obj.geometry || {}));
      if (Array.isArray(geometry.points)) geometry.points = geometry.points.map((p) => ({ x: p.x + dx, y: p.y + dy }));
      else { geometry.x = (geometry.x || 0) + dx; geometry.y = (geometry.y || 0) + dy; }
      try {
        await apiSend("PATCH", `/api/sessions/${encodeURIComponent(sessionId)}/drawing-objects/${encodeURIComponent(obj.id)}`, { session_id: sessionId, geometry });
        await refresh();
      } catch (err) {
        setStatus("Move failed: " + err.message);
      }
    }

    // --- Detail View (camera zoom-to-region over the SAME objects) -----
    // A real camera zoom, not just a UI-state flag: uses the exact same
    // stageCamera.setView the shared runtime already uses for pan/zoom
    // controls (frontend/lib/victory-stage-camera.js), so this is
    // provably a camera transform over the one shared world container
    // (kernel-87 §6), not a second document -- there is no separate
    // Detail View render target anywhere in this module.
    function openDetailViewFromSelection() {
      const obj = state.objects.find((o) => o.id === state.selectedId);
      const bounds = obj ? objectBounds(obj) : null;
      const region = bounds || getWorldBounds();
      const camera = getCamera?.();
      const playable = getWorldBounds();
      if (camera?.getView && camera?.setView && region.width > 0 && region.height > 0) {
        state.detailView = { previousView: camera.getView() };
        const margin = 1.15; // small breathing room around the region
        const zoom = Math.max(0.1, Math.min(
          playable.width / (region.width * margin),
          playable.height / (region.height * margin),
        ));
        const regionCenterX = region.x + region.width / 2;
        const regionCenterY = region.y + region.height / 2;
        const playableCenterX = playable.x + playable.width / 2;
        const playableCenterY = playable.y + playable.height / 2;
        camera.setView({
          zoomRelativeToFit: zoom,
          panX: playableCenterX - regionCenterX * zoom,
          panY: playableCenterY - regionCenterY * zoom,
        });
        // The camera clamps zoomRelativeToFit to its own [minZoom,maxZoom]
        // (victory-stage-camera.js), which a small single-cell region can
        // easily exceed (a 28x28 region wants ~40x zoom; the camera's
        // default ceiling is far lower). If the applied zoom differs from
        // what was requested above, the panX/panY already sent were
        // computed against the WRONG zoom and no longer center the
        // region -- re-read the camera's actually-applied zoom and, if it
        // was clamped, recompute+resend pan for that real value so the
        // region stays correctly framed instead of drifting off to an
        // unrelated part of the map.
        const applied = camera.getView?.();
        if (applied && Math.abs(Number(applied.zoomRelativeToFit) - zoom) > 0.001) {
          const appliedZoom = Number(applied.zoomRelativeToFit) || zoom;
          camera.setView({
            zoomRelativeToFit: appliedZoom,
            panX: playableCenterX - regionCenterX * appliedZoom,
            panY: playableCenterY - regionCenterY * appliedZoom,
          });
        }
      } else {
        state.detailView = { previousView: null };
      }
      btnDetailOpen.style.display = "none";
      btnDetailClose.style.display = "inline-block";
      setStatus("Detail View: same canonical objects, camera zoomed to region.");
      pixiApp.stage.emit?.("cartograph:detail-view-open", region);
    }
    function closeDetailView() {
      const camera = getCamera?.();
      if (camera?.setView && state.detailView?.previousView) {
        camera.setView(state.detailView.previousView);
      } else if (camera?.fit) {
        camera.fit();
      }
      state.detailView = null;
      btnDetailOpen.style.display = "inline-block";
      btnDetailClose.style.display = "none";
      pixiApp.stage.emit?.("cartograph:detail-view-close");
      setStatus("Detail View closed. Objects remain in their map-relative positions.");
    }

    // --- Export ----------------------------------------------------------
    function exportPNG() {
      try {
        // Kernel 87 §12: base map + drawings, no UI chrome, no transient
        // measurement. worldLayer already excludes toolbar/HUD (all plain
        // DOM), and we hide the ephemeral overlays for one frame.
        selectionOverlay.visible = false;
        measureLayer.visible = false;
        pendingGraphic.visible = false;
        const canvas = pixiApp.renderer.extract.canvas(pixiApp.stage);
        selectionOverlay.visible = true;
        measureLayer.visible = true;
        pendingGraphic.visible = true;
        const link = document.createElement("a");
        link.download = `victory-map-${Date.now()}.png`;
        link.href = canvas.toDataURL ? canvas.toDataURL("image/png") : canvas;
        link.click();
        setStatus("Exported PNG.");
      } catch (err) {
        setStatus("Export failed: " + err.message);
      }
    }

    // --- Wire pointer events on the shared stage host --------------------
    const downHandler = (evt) => onPointerDown(evt);
    const moveHandler = (evt) => onPointerMove(evt);
    const upHandler = () => onPointerUp();
    const dblClickHandler = () => {
      if (state.tool === "polyline" || state.tool === "polygon") finishPolyShape();
    };
    const keyHandler = (evt) => {
      if (evt.key === "Enter" && (state.tool === "polyline" || state.tool === "polygon")) finishPolyShape();
      if (evt.key === "Escape") { state.pendingPoints = []; redrawPending(); }
      if (evt.key === "Delete" || evt.key === "Backspace") {
        if (state.selectedId && document.activeElement === document.body) deleteSelected();
      }
      if (state.selectedId && ["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"].includes(evt.key)) {
        const step = evt.shiftKey ? 10 : 1;
        const deltas = { ArrowUp: [0, -step], ArrowDown: [0, step], ArrowLeft: [-step, 0], ArrowRight: [step, 0] };
        const [dx, dy] = deltas[evt.key];
        moveSelected(dx, dy);
      }
    };
    hostElement.addEventListener("pointerdown", downHandler);
    hostElement.addEventListener("pointermove", moveHandler);
    hostElement.addEventListener("pointerup", upHandler);
    hostElement.addEventListener("dblclick", dblClickHandler);
    window.addEventListener("keydown", keyHandler);

    highlightTool();
    refresh();

    return {
      refresh,
      setTool,
      getState: () => state,
      destroy() {
        hostElement.removeEventListener("pointerdown", downHandler);
        hostElement.removeEventListener("pointermove", moveHandler);
        hostElement.removeEventListener("pointerup", upHandler);
        hostElement.removeEventListener("dblclick", dblClickHandler);
        window.removeEventListener("keydown", keyHandler);
        panel.remove();
        layerNode.destroy({ children: true });
      },
    };
  }

  return { mount, drawObjectGraphic, TOOLS, STAMP_GLYPHS };
});
