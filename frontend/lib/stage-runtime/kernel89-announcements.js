// Kernel 89 §10: theatrical Director announcements.
//
// A DOM overlay rather than a PIXI node, deliberately. Kernel 86's dice
// projection is in PIXI because dice land at map-relative coordinates and
// must ride the camera transform; an announcement is a caption ABOUT the
// stage, fixed in screen space, and it wants real typography, real text
// wrapping, and a real accessible reading order -- none of which a canvas
// gives for free. It renders above the stage and below the Director panels.
//
// The palette is NOT defined here. Styles arrive on the effect payload,
// server-side from backend/internal/announcements, so a style can never mean
// one thing to the sender and another to the viewer. This module knows how
// to draw a Style; it does not know which ones exist.
//
// Nothing here reads a roll (§10.4). The module has no access to dice state
// and never asks for any -- an announcement carries only the words and the
// style the Director chose.
(function () {
  "use strict";

  const HOST_ID = "kernel89-announcement-host";
  const STYLE_ID = "kernel89-announcement-css";
  const FADE_MS = 420;

  const state = {
    host: null,
    queue: [],
    current: null,
    timers: [],
    pinned: new Set(),
  };

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
      .replaceAll("\"", "&quot;").replaceAll("'", "&#39;");
  }

  // Motion is a closed vocabulary on the server, so the keyframes are a
  // closed set here too -- there is no path from authored text to CSS.
  function ensureStylesheet() {
    if (document.getElementById(STYLE_ID)) return;
    const el = document.createElement("style");
    el.id = STYLE_ID;
    el.textContent = `
      #${HOST_ID} {
        position: fixed; top: 14%; left: 50%; transform: translateX(-50%);
        z-index: 8600; pointer-events: none;
        display: flex; flex-direction: column; align-items: center; gap: 10px;
        width: min(760px, 88vw);
      }

      /* A Director with a tool panel open has ~400px of the right edge
         spoken for. The banner is the same object every Player sees, so it
         is not moved for them -- it just gives way on the one screen that
         has something else on it. kernel89-director-tools.js sets the
         class while any of its panels is open. */
      body.k89-director-panel-open #${HOST_ID} {
        left: calc(50% - 210px);
        width: min(660px, calc(100vw - 460px));
      }
      .k89-ann {
        pointer-events: auto;
        width: 100%; box-sizing: border-box;
        padding: 18px 26px;
        border: 2px solid var(--k89-accent);
        border-radius: 10px;
        background: var(--k89-bg);
        color: var(--k89-ink);
        box-shadow: 0 14px 44px rgba(0,0,0,0.55), 0 0 0 1px rgba(0,0,0,0.4);
        font-family: "Iowan Old Style", Georgia, "Times New Roman", serif;
        text-align: center;
        opacity: 0;
        transition: opacity ${FADE_MS}ms ease;
      }
      .k89-ann.is-in { opacity: 1; }
      .k89-ann.is-out { opacity: 0; }

      /* Shape carries the same distinction the accent does, so the palette
         never depends on colour alone (§10.2). */
      .k89-ann[data-shape="burst"] { border-radius: 4px; border-style: double; border-width: 5px; }
      .k89-ann[data-shape="jagged"] { border-style: dashed; border-width: 3px; }
      .k89-ann[data-shape="halo"] { border-radius: 999px; padding-left: 44px; padding-right: 44px; }
      .k89-ann[data-shape="crown"] { border-top-width: 7px; border-radius: 10px 10px 4px 4px; }
      .k89-ann[data-shape="slab"] { border-left-width: 8px; border-radius: 3px; }

      .k89-ann__tag {
        display: inline-flex; align-items: center; gap: 7px;
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        font-size: 11px; letter-spacing: 0.16em; text-transform: uppercase;
        color: var(--k89-accent); margin-bottom: 8px;
      }
      .k89-ann__glyph { font-size: 15px; line-height: 1; }
      .k89-ann__text { margin: 0; line-height: 1.15; overflow-wrap: anywhere; }
      .k89-ann[data-emphasis="loud"] .k89-ann__text { font-size: clamp(30px, 5.4vw, 58px); font-weight: 700; letter-spacing: 0.02em; }
      .k89-ann[data-emphasis="firm"] .k89-ann__text { font-size: clamp(24px, 4vw, 42px); font-weight: 600; }
      .k89-ann[data-emphasis="soft"] .k89-ann__text { font-size: clamp(20px, 3vw, 32px); font-weight: 500; font-style: italic; }
      .k89-ann__by {
        margin: 10px 0 0 0;
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        font-size: 11px; opacity: 0.6; letter-spacing: 0.04em;
      }
      .k89-ann__dismiss {
        margin-top: 12px; background: transparent; color: var(--k89-ink);
        border: 1px solid var(--k89-accent); border-radius: 5px;
        padding: 4px 12px; font-size: 11px; cursor: pointer; opacity: 0.8;
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      }
      .k89-ann__dismiss:hover { opacity: 1; }

      .k89-ann[data-motion="slam"].is-in   { animation: k89Slam 340ms cubic-bezier(.2,1.5,.4,1) both; }
      .k89-ann[data-motion="rise"].is-in   { animation: k89Rise 520ms ease-out both; }
      .k89-ann[data-motion="shake"].is-in  { animation: k89Shake 620ms ease-in-out both; }
      .k89-ann[data-motion="unveil"].is-in { animation: k89Unveil 760ms ease-out both; }
      .k89-ann[data-motion="flicker"].is-in{ animation: k89Flicker 560ms steps(1, end) both; }

      @keyframes k89Slam { 0% { transform: scale(1.5); opacity: 0; } 60% { transform: scale(0.96); opacity: 1; } 100% { transform: scale(1); opacity: 1; } }
      @keyframes k89Rise { 0% { transform: translateY(26px); opacity: 0; } 100% { transform: translateY(0); opacity: 1; } }
      @keyframes k89Shake {
        0% { transform: translate(0,0) scale(1.16); opacity: 0; }
        25% { transform: translate(-7px,2px) scale(1); opacity: 1; }
        45% { transform: translate(6px,-2px); } 65% { transform: translate(-4px,1px); }
        85% { transform: translate(2px,0); } 100% { transform: translate(0,0); opacity: 1; }
      }
      @keyframes k89Unveil { 0% { letter-spacing: 0.5em; opacity: 0; } 100% { letter-spacing: 0.02em; opacity: 1; } }
      @keyframes k89Flicker { 0% { opacity: 0; } 20% { opacity: 1; } 35% { opacity: 0.15; } 50% { opacity: 1; } 65% { opacity: 0.3; } 100% { opacity: 1; } }

      /* Motion is decoration, never the message: with reduced motion the
         announcement simply appears, keeping glyph/shape/label distinctions. */
      @media (prefers-reduced-motion: reduce) {
        .k89-ann.is-in { animation: none !important; }
      }
    `;
    document.head.appendChild(el);
  }

  function ensureHost() {
    if (state.host && document.body.contains(state.host)) return state.host;
    ensureStylesheet();
    const el = document.createElement("div");
    el.id = HOST_ID;
    el.setAttribute("aria-live", "polite");
    document.body.appendChild(el);
    state.host = el;
    return el;
  }

  function clearTimers() {
    state.timers.forEach((t) => window.clearTimeout(t));
    state.timers = [];
  }

  function after(ms, fn) {
    state.timers.push(window.setTimeout(fn, ms));
  }

  function normalize(rawEffect) {
    const effect = rawEffect || {};
    const payload = effect.payload || {};
    const style = payload.style || {};
    const text = String(payload.text || "").trim();
    if (!text) return null;
    return {
      id: String(effect.effect_id || effect.id || ""),
      text,
      actorLabel: String(payload.actor_label || "").trim(),
      durationMs: Math.max(1200, Number(effect.duration_ms || 5000)),
      canDismiss: Boolean(effect.__canDismiss),
      style: {
        key: String(style.key || "custom"),
        label: String(style.label || "Announcement"),
        accent: String(style.accent || "#c8ccd6"),
        background: String(style.background || "#15181e"),
        ink: String(style.ink || "#f2f4f8"),
        glyph: String(style.glyph || "▪"),
        motion: String(style.motion || "rise"),
        shape: String(style.shape || "slab"),
        emphasis: String(style.emphasis || "soft"),
      },
    };
  }

  function build(effect) {
    const s = effect.style;
    const node = document.createElement("div");
    node.className = "k89-ann";
    node.dataset.effectId = effect.id;
    node.dataset.styleKey = s.key;
    node.dataset.shape = s.shape;
    node.dataset.motion = s.motion;
    node.dataset.emphasis = s.emphasis;
    node.style.setProperty("--k89-accent", s.accent);
    node.style.setProperty("--k89-bg", s.background);
    node.style.setProperty("--k89-ink", s.ink);
    node.setAttribute("role", "status");

    // The style's own name is printed, not just implied by colour: a viewer
    // who cannot see the accent still reads "EXPLOSION".
    node.innerHTML = `
      <div class="k89-ann__tag"><span class="k89-ann__glyph" aria-hidden="true">${escapeHtml(s.glyph)}</span>${escapeHtml(s.label)}</div>
      <p class="k89-ann__text">${escapeHtml(effect.text)}</p>
      ${effect.actorLabel ? `<p class="k89-ann__by">${escapeHtml(effect.actorLabel)}</p>` : ""}
      ${effect.canDismiss ? `<button type="button" class="k89-ann__dismiss" data-k89-dismiss="1">Dismiss</button>` : ""}
    `;
    const dismiss = node.querySelector("[data-k89-dismiss]");
    if (dismiss) {
      dismiss.addEventListener("click", () => {
        state.pinned.delete(effect.id);
        window.VictoryStageKernel88Bridge?.sendAction?.("stage_effect/dismiss", { effect_id: effect.id });
        retire();
      });
    }
    return node;
  }

  function retire() {
    const entry = state.current;
    if (!entry) return;
    clearTimers();
    state.current = null;
    entry.node.classList.remove("is-in");
    entry.node.classList.add("is-out");
    window.setTimeout(() => {
      entry.node.remove();
      startNext();
    }, FADE_MS);
  }

  function startNext() {
    if (state.current) return;
    const effect = state.queue.shift();
    if (!effect) return;

    const node = build(effect);
    ensureHost().appendChild(node);
    state.current = { effect, node };
    // One frame before adding is-in so the entry animation actually runs
    // rather than being collapsed into the initial paint.
    window.requestAnimationFrame(() => node.classList.add("is-in"));

    if (!state.pinned.has(effect.id)) {
      after(effect.durationMs, () => {
        // Re-checked at fire time: a pin that arrived while the banner was
        // on screen must keep it there.
        if (state.pinned.has(effect.id)) return;
        retire();
      });
    }
  }

  const controller = {
    // present returns true when it took ownership of the effect, so the
    // caller (runtime.js) can fall through to the dice projection for
    // everything else without this module knowing what dice are.
    present(rawEffect, options = {}) {
      if (!rawEffect || String(rawEffect.type || "") !== "announcement") return false;
      const effect = normalize({ ...rawEffect, __canDismiss: Boolean(options.canDismiss) });
      if (!effect) return true; // ours, but unrenderable -- do not hand it to dice
      if (state.current && state.current.effect.id === effect.id) return true;
      if (state.queue.some((e) => e.id === effect.id)) return true;
      if (state.queue.length >= 4) state.queue.shift();
      state.queue.push(effect);
      startNext();
      return true;
    },

    pin(rawEffect) {
      const id = String(rawEffect?.effect_id || rawEffect?.id || "");
      if (!id) return false;
      if (String(rawEffect?.type || "") !== "announcement") return false;
      state.pinned.add(id);
      clearTimers();
      return true;
    },

    dismiss(effectId) {
      const id = String(effectId || "");
      if (!id) return false;
      state.pinned.delete(id);
      state.queue = state.queue.filter((e) => e.id !== id);
      if (state.current && state.current.effect.id === id) {
        retire();
        return true;
      }
      return false;
    },

    // Test/diagnostic surface. Deliberately read-only.
    inspect() {
      return {
        showing: state.current ? state.current.effect.style.key : "",
        text: state.current ? state.current.effect.text : "",
        queued: state.queue.length,
        pinned: Array.from(state.pinned),
      };
    },
  };

  window.VictoryKernel89Announcements = controller;
})();
