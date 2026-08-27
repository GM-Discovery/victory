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
  };

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
        position: fixed; left: 50%; bottom: 14px; transform: translateX(-50%);
        z-index: 8500; display: flex; gap: 4px; align-items: flex-end;
        padding: 6px 8px; border-radius: 999px;
        background: rgba(16, 17, 20, 0.72); border: 1px solid rgba(255,255,255,0.14);
        backdrop-filter: blur(6px);
        max-width: min(94vw, 640px); overflow-x: auto;
      }
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

      #${HOST_ID} {
        position: fixed; left: 0; right: 0; bottom: 64px; height: 40vh;
        z-index: 8400; pointer-events: none; overflow: hidden;
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
    const bar = document.createElement("div");
    bar.id = BAR_ID;
    bar.setAttribute("role", "group");
    bar.setAttribute("aria-label", "Send a reaction");
    bar.innerHTML = REACTIONS.map((r) => `
      <button type="button" class="k1-react-btn" data-kind="${r.kind}" data-accent="${r.accent ? "1" : "0"}" title="${escapeHtml(r.label)}">
        <span class="k1-react-glyph" aria-hidden="true">${r.glyph}</span>
        <span class="k1-react-label">${escapeHtml(r.label)}</span>
      </button>
    `).join("");
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
    document.body.appendChild(bar);
    state.bar = bar;
    return bar;
  }

  function unmount() {
    state.bar?.remove();
    state.bar = null;
  }

  window.VictoryReactions = { present, mount, unmount };
})();
