// Kernel 1 §8: "Immediate reactions" — the audience-to-production half of
// the exchange the very first production slice spec called for ("outbound:
// performer tells story; inbound: audience reacts... that makes it theater,
// not slideshow"). The server side (backend/internal/actions/react.go,
// network/ws.go's "react/emote" case) has existed since before this kernel
// and was fully wired, broadcasting to every connection -- there was simply
// no frontend caller and no renderer anywhere in the repo (Kernel 93 ledger
// A18). This module is both.
//
// Deliberately NOT gated to Audience only: the backend stores react/emote
// visible to every role (`toRoles: [audience, cast, crew, director,
// producer]`), and Kernel 1 §4/§9 describes the Director as wanting to
// *see* audience reactions, not send them exclusively -- but nothing in
// the spec forbids Cast/Crew/Director from reacting too, and a shared bar
// everyone can use is the simpler v1 than inventing a role split nobody
// asked for. Revisit if that turns out to be the wrong call live.
//
// note_card (Kernel 1's "Routed reactions", needing delivery/addressing
// logic) is explicitly out of scope here -- see kernel-11-note-card-
// delivery.md and Kernel 93 ledger A19 for that separate feature.
(function () {
  "use strict";

  const BAR_ID = "k1-reactions-bar";
  const HOST_ID = "k1-reactions-host";
  const STYLE_ID = "k1-reactions-css";
  const FLIGHT_MS = 2200;
  const MAX_CONCURRENT = 24;
  const PREFS_KEY = "k1ReactionsPrefs";
  // Grant, 2026-08-27 live testing: the bar's original fixed bottom:14px
  // sat inside the chat panel's collapsed rail (catharsis/index.html's
  // --chat-rail-height: 64px), hiding its handle. COLLAPSED_BOTTOM is the
  // bar's own resting spot when collapsed (kept low and out of the way);
  // DEFAULT_LIFT (24px = 1/4in @96dpi) is how far it rises above that when
  // expanded, per Grant's literal instruction -- both the collapse and the
  // lift are also independently adjustable below (data-collapsed persists
  // like the opacity/lift sliders' values).
  const COLLAPSED_BOTTOM = 8;
  const DEFAULT_LIFT = 24;
  const DEFAULT_OPACITY = 0.45;

  // Glyph AND label together, always -- never emoji alone. clap/cheer/
  // standing_clap and thumbs_up/thumbs_down/boo are each other's obvious
  // confusion pairs at a glance, and a viewer relying on a screen reader or
  // who simply can't parse a small emoji gets nothing at all otherwise.
  const REACTIONS = [
    { kind: "clap", glyph: "👏", label: "Clap" },
    { kind: "cheer", glyph: "🙌", label: "Cheer" },
    { kind: "standing_clap", glyph: "👏", label: "Standing Ovation", accent: true },
    { kind: "laugh", glyph: "😂", label: "Laugh" },
    { kind: "smile", glyph: "🙂", label: "Smile" },
    { kind: "heart", glyph: "❤️", label: "Heart" },
    { kind: "cry", glyph: "😢", label: "Cry" },
    { kind: "startle", glyph: "😱", label: "Startle" },
    { kind: "thumbs_up", glyph: "👍", label: "Thumbs Up" },
    { kind: "thumbs_down", glyph: "👎", label: "Thumbs Down" },
    { kind: "boo", glyph: "🍅", label: "Boo" },
  ];
  const REACTIONS_BY_KIND = Object.fromEntries(REACTIONS.map((r) => [r.kind, r]));

  const state = {
    bar: null,
    host: null,
    live: 0,
    lastSendAt: 0,
    prefs: { collapsed: false, opacity: DEFAULT_OPACITY, lift: DEFAULT_LIFT },
  };

  function clampNum(value, min, max, fallback) {
    const n = Number(value);
    return Number.isFinite(n) ? Math.min(max, Math.max(min, n)) : fallback;
  }

  function loadPrefs() {
    try {
      const raw = window.localStorage?.getItem(PREFS_KEY);
      const parsed = raw ? JSON.parse(raw) : {};
      return {
        collapsed: Boolean(parsed.collapsed),
        opacity: clampNum(parsed.opacity, 0.15, 0.95, DEFAULT_OPACITY),
        lift: clampNum(parsed.lift, 12, 140, DEFAULT_LIFT),
      };
    } catch {
      return { collapsed: false, opacity: DEFAULT_OPACITY, lift: DEFAULT_LIFT };
    }
  }

  function savePrefs() {
    try {
      window.localStorage?.setItem(PREFS_KEY, JSON.stringify(state.prefs));
    } catch {
      // Private-browsing / storage-disabled: settings just don't persist
      // across reloads. Not worth surfacing as an error for a cosmetic pref.
    }
  }

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
      .replaceAll("\"", "&quot;").replaceAll("'", "&#39;");
  }

  function ensureStylesheet() {
    if (document.getElementById(STYLE_ID)) return;
    const el = document.createElement("style");
    el.id = STYLE_ID;
    el.textContent = `
      #${BAR_ID} {
        position: fixed; z-index: 8500; display: flex; gap: 4px; align-items: center;
        padding: 6px 8px; border-radius: 999px;
        background: rgba(16, 17, 20, ${DEFAULT_OPACITY});
        background: rgba(16 17 20 / var(--k1-react-opacity, ${DEFAULT_OPACITY}));
        border: 1px solid rgba(255,255,255,0.14);
        backdrop-filter: blur(6px);
        max-width: min(94vw, 680px);
        transition: bottom 180ms ease, left 180ms ease, right 180ms ease, transform 180ms ease, background-color 180ms ease;
      }
      /* Collapsed: tucked into the bottom-right corner, out of the way of
         the centered chat rail -- same corner for every role, Audience
         included; this bar was never role-gated (see the top of this
         file). Expanded: centered above the chat rail, raised by the
         adjustable lift amount. */
      #${BAR_ID}[data-collapsed="true"] {
        left: auto; right: 14px; bottom: ${COLLAPSED_BOTTOM}px; transform: none;
      }
      #${BAR_ID}[data-collapsed="false"] {
        left: 50%; right: auto; transform: translateX(-50%);
        bottom: calc(${COLLAPSED_BOTTOM}px + var(--k1-react-lift, ${DEFAULT_LIFT}px));
      }
      #${BAR_ID}[data-collapsed="true"] .k1-react-buttons,
      #${BAR_ID}[data-collapsed="true"] .k1-react-gear,
      #${BAR_ID}[data-collapsed="true"] .k1-react-settings {
        display: none;
      }
      .k1-react-buttons {
        display: flex; gap: 4px; align-items: flex-end;
        max-width: min(88vw, 600px); overflow-x: auto;
      }
      .k1-react-toggle, .k1-react-gear {
        flex: 0 0 auto; display: flex; align-items: center; justify-content: center;
        width: 26px; height: 26px; border-radius: 50%;
        background: rgba(255,255,255,0.06); border: 1px solid rgba(255,255,255,0.16);
        color: #e8e8ec; cursor: pointer; font-size: 12px; line-height: 1;
      }
      .k1-react-toggle:hover, .k1-react-gear:hover { background: rgba(255,255,255,0.14); }
      #${BAR_ID}[data-collapsed="true"] .k1-react-toggle::before { content: "🙂"; font-size: 14px; }
      #${BAR_ID}[data-collapsed="false"] .k1-react-toggle::before { content: "▾"; }
      .k1-react-btn {
        flex: 0 0 auto; display: flex; flex-direction: column; align-items: center;
        gap: 1px; width: 44px; padding: 5px 2px 4px;
        background: transparent; border: 1px solid transparent; border-radius: 8px;
        cursor: pointer; color: #e8e8ec; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      }
      .k1-react-btn:hover, .k1-react-btn:focus-visible { background: rgba(255,255,255,0.1); border-color: rgba(255,255,255,0.22); }
      .k1-react-btn:active { transform: scale(0.92); }
      .k1-react-btn[data-accent="1"] .k1-react-glyph { filter: drop-shadow(0 0 4px #f0d49b); }
      .k1-react-glyph { font-size: 18px; line-height: 1; }
      .k1-react-label { font-size: 8px; letter-spacing: 0.01em; opacity: 0.72; white-space: nowrap; }
      .k1-react-btn.is-sending { opacity: 0.5; pointer-events: none; }

      .k1-react-settings {
        position: absolute; bottom: calc(100% + 8px); left: 50%; transform: translateX(-50%);
        display: flex; flex-direction: column; gap: 8px;
        padding: 10px 14px; border-radius: 12px; width: 200px;
        background: rgba(16, 17, 20, 0.94); border: 1px solid rgba(255,255,255,0.16);
        box-shadow: 0 10px 30px rgba(0,0,0,0.4);
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        color: #e8e8ec; font-size: 11px;
      }
      .k1-react-settings[hidden] { display: none; }
      .k1-react-settings label { display: flex; flex-direction: column; gap: 3px; }
      .k1-react-settings input[type="range"] { width: 100%; accent-color: #f0d49b; }

      #${HOST_ID} {
        position: fixed; left: 0; right: 0; bottom: 64px; height: 40vh;
        z-index: 8400; pointer-events: none; overflow: hidden;
        transition: bottom 180ms ease;
      }
      .k1-react-fly {
        position: absolute; bottom: 0; display: flex; flex-direction: column; align-items: center;
        gap: 2px; opacity: 0; animation: k1ReactRise ${FLIGHT_MS}ms ease-out forwards;
      }
      .k1-react-fly__glyph { font-size: 26px; line-height: 1; }
      .k1-react-fly__by {
        font-size: 9px; color: #f2f4f8; background: rgba(0,0,0,0.55);
        padding: 1px 6px; border-radius: 999px; white-space: nowrap;
      }
      @keyframes k1ReactRise {
        0% { opacity: 0; transform: translateY(0) scale(0.7); }
        12% { opacity: 1; transform: translateY(-10px) scale(1); }
        78% { opacity: 1; }
        100% { opacity: 0; transform: translateY(-220px) scale(1.05); }
      }
      @media (prefers-reduced-motion: reduce) {
        .k1-react-fly { animation: none !important; opacity: 1; bottom: 40%; }
      }
    `;
    document.head.appendChild(el);
  }

  function ensureHost() {
    if (state.host && document.body.contains(state.host)) return state.host;
    ensureStylesheet();
    const el = document.createElement("div");
    el.id = HOST_ID;
    el.setAttribute("aria-hidden", "true");
    document.body.appendChild(el);
    state.host = el;
    return el;
  }

  // Keeps the floating-reaction layer's origin just above wherever the bar
  // currently sits, so an expanded (raised) bar never has reactions rising
  // up through/behind it, and a collapsed bar doesn't leave a large empty
  // gap underneath.
  function syncHostPosition() {
    if (!state.bar) return;
    const host = ensureHost();
    const barBottom = state.prefs.collapsed ? COLLAPSED_BOTTOM : COLLAPSED_BOTTOM + state.prefs.lift;
    const barHeight = state.bar.offsetHeight || (state.prefs.collapsed ? 38 : 64);
    host.style.bottom = `${barBottom + barHeight + 8}px`;
  }

  function applyPrefs() {
    if (!state.bar) return;
    state.bar.dataset.collapsed = state.prefs.collapsed ? "true" : "false";
    state.bar.style.setProperty("--k1-react-opacity", String(state.prefs.opacity));
    state.bar.style.setProperty("--k1-react-lift", `${state.prefs.lift}px`);
    syncHostPosition();
  }

  // present() renders every incoming react/emote action, including the
  // sender's own -- their click already gave them a click, not confirmation
  // it reached anyone else. This is that confirmation.
  function present(action) {
    const kind = String(action?.payload?.kind || "").trim();
    const reaction = REACTIONS_BY_KIND[kind];
    if (!reaction) return false;
    ensureHost();
    if (state.live >= MAX_CONCURRENT) return true; // ours, just dropped -- a wall of reactions is still a successful reaction, not a queue to backlog
    const actorLabel = String(action?.actor_display_name || action?.actor?.display_name || "").trim();

    const node = document.createElement("div");
    node.className = "k1-react-fly";
    // Spread horizontally so a burst of the same reaction doesn't stack
    // into one unreadable column.
    const spreadPct = 38 + Math.random() * 24;
    node.style.left = `${spreadPct}%`;
    node.style.transform = `translateX(-50%) translateX(${(Math.random() - 0.5) * 40}px)`;
    node.innerHTML = `
      <span class="k1-react-fly__glyph" aria-hidden="true">${reaction.glyph}</span>
      ${actorLabel ? `<span class="k1-react-fly__by">${escapeHtml(actorLabel)}</span>` : ""}
    `;
    ensureHost().appendChild(node);
    state.live += 1;
    window.setTimeout(() => {
      node.remove();
      state.live = Math.max(0, state.live - 1);
    }, FLIGHT_MS + 100);
    return true;
  }

  // mount() builds the always-available send bar. sendFn is
  // window.VictoryStageKernel88Bridge.sendAction, injected rather than read
  // directly so this module has zero dependency on runtime.js's globals
  // (matching kernel93-audience-overlay.js's own independence).
  function mount(sendFn) {
    if (state.bar && document.body.contains(state.bar)) return state.bar;
    ensureStylesheet();
    state.prefs = loadPrefs();

    const bar = document.createElement("div");
    bar.id = BAR_ID;
    bar.setAttribute("role", "group");
    bar.setAttribute("aria-label", "Reactions");
    bar.innerHTML = `
      <button type="button" class="k1-react-toggle" aria-label="Collapse or expand reactions"></button>
      <div class="k1-react-buttons">
        ${REACTIONS.map((r) => `
          <button type="button" class="k1-react-btn" data-kind="${r.kind}" data-accent="${r.accent ? "1" : "0"}" title="${escapeHtml(r.label)}">
            <span class="k1-react-glyph" aria-hidden="true">${r.glyph}</span>
            <span class="k1-react-label">${escapeHtml(r.label)}</span>
          </button>
        `).join("")}
      </div>
      <button type="button" class="k1-react-gear" aria-label="Reaction bar display settings">⚙</button>
      <div class="k1-react-settings" hidden>
        <label>Transparency
          <input type="range" class="k1-react-opacity-slider" min="0.15" max="0.95" step="0.05">
        </label>
        <label>Height above chat bar
          <input type="range" class="k1-react-lift-slider" min="12" max="140" step="4">
        </label>
      </div>
    `;

    bar.querySelectorAll("[data-kind]").forEach((btn) => {
      btn.addEventListener("click", () => {
        if (typeof sendFn !== "function") return;
        // A cheap client-side debounce (250ms) against accidental double-
        // fire / key-repeat -- the server has no rate limit of its own, and
        // one honest reaction is the intent, not a burst from one click.
        const now = Date.now();
        if (now - state.lastSendAt < 250) return;
        state.lastSendAt = now;
        btn.classList.add("is-sending");
        window.setTimeout(() => btn.classList.remove("is-sending"), 250);
        sendFn("react/emote", { kind: btn.dataset.kind });
      });
    });

    bar.querySelector(".k1-react-toggle").addEventListener("click", () => {
      state.prefs.collapsed = !state.prefs.collapsed;
      applyPrefs();
      savePrefs();
      if (state.prefs.collapsed) bar.querySelector(".k1-react-settings").hidden = true;
    });

    const settingsPanel = bar.querySelector(".k1-react-settings");
    bar.querySelector(".k1-react-gear").addEventListener("click", (event) => {
      event.stopPropagation();
      settingsPanel.hidden = !settingsPanel.hidden;
    });
    document.addEventListener("click", (event) => {
      if (!settingsPanel.hidden && !bar.contains(event.target)) settingsPanel.hidden = true;
    });

    const opacitySlider = bar.querySelector(".k1-react-opacity-slider");
    opacitySlider.value = String(state.prefs.opacity);
    opacitySlider.addEventListener("input", () => {
      state.prefs.opacity = clampNum(opacitySlider.value, 0.15, 0.95, DEFAULT_OPACITY);
      applyPrefs();
    });
    opacitySlider.addEventListener("change", savePrefs);

    const liftSlider = bar.querySelector(".k1-react-lift-slider");
    liftSlider.value = String(state.prefs.lift);
    liftSlider.addEventListener("input", () => {
      state.prefs.lift = clampNum(liftSlider.value, 12, 140, DEFAULT_LIFT);
      applyPrefs();
    });
    liftSlider.addEventListener("change", savePrefs);

    document.body.appendChild(bar);
    state.bar = bar;
    applyPrefs();
    return bar;
  }

  function unmount() {
    state.bar?.remove();
    state.bar = null;
  }

  window.VictoryReactions = { present, mount, unmount };
})();
