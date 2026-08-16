// Kernel 88: the Player-facing Socio play surface -- sparse, spoiler-safe,
// agency-focused (spec §2.2). Self-contained module in the same style as
// kernel85-cohort-tools.js: reads the engine only through the narrow
// read-only window.VictoryStageKernel88Bridge runtime.js exposes, and every
// mutation/read goes through the server (backend/internal/socio), which
// re-checks authority and shapes the response by viewer tier
// (socio.ProjectSocioState) -- nothing here is trusted client state.
//
// Face and mechanic data are NOT re-implemented here: Face portrait/name
// and the compiler-backed skill list both come from the existing
// GET /api/characters/venue-sheet endpoint (Kernel 59A/60's shared
// ProjectCharacterSheet projection) -- this HUD only adds Fate/Stance/
// qualitative-condition/roll controls on top of what already exists.
(function () {
  "use strict";

  const STANCE_ORDER = ["insight", "command", "convince", "sympathize", "follow"];

  // Single slate-blue hue for the whole Stance wheel (see renderStanceWheel).
  const WHEEL_HUE = 212;

  // The HUD is the Player's own optional instrument panel, not part of the
  // stage: it starts collapsed to a small launcher so nobody is forced into
  // the view, and its position/collapsed state persist per browser.
  const STORAGE_KEY = "victory.kernel88.socioHud";

  function loadChrome() {
    try {
      const raw = window.localStorage.getItem(STORAGE_KEY);
      if (!raw) return { collapsed: true, left: null, top: null };
      const parsed = JSON.parse(raw);
      return {
        collapsed: parsed.collapsed !== false,
        left: typeof parsed.left === "number" ? parsed.left : null,
        top: typeof parsed.top === "number" ? parsed.top : null,
      };
    } catch (_e) {
      return { collapsed: true, left: null, top: null };
    }
  }

  function saveChrome() {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify({
        collapsed: state.collapsed, left: state.left, top: state.top,
      }));
    } catch (_e) { /* private mode / storage disabled -- non-fatal */ }
  }

  const chrome = loadChrome();

  const state = {
    hud: null,
    body: null,
    header: null,
    collapsed: chrome.collapsed,
    left: chrome.left,
    top: chrome.top,
    lastShowID: "",
    lastCharacterID: "",
    stanceRegistry: [],
    busy: false,
    // The Face/skill sheet changes only when the played Character does, so
    // it is fetched on identity change and cached; the Socio projection is
    // re-read every tick because a Director can change it at any moment.
    sheet: null,
    mechanics: null,
    characterName: "",
    mechanicsError: "",
    lastFingerprint: "",
    forceRedraw: false,
  };

  function bridge() {
    return window.VictoryStageKernel88Bridge || null;
  }

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
      .replaceAll("\"", "&quot;").replaceAll("'", "&#39;");
  }

  async function api(path, options = {}) {
    const response = await fetch(path, { credentials: "include", headers: { "Content-Type": "application/json" }, ...options });
    let body = null;
    try { body = await response.json(); } catch (_e) { /* no body */ }
    if (!response.ok || (body && body.ok === false)) {
      const message = body?.data?.error || body?.error || `request_failed_${response.status}`;
      throw new Error(message);
    }
    return body?.data ?? body;
  }

  // Anything a mouse press is meant to do something else with. Dragging must
  // yield to these: a mousedown handler that calls preventDefault() over a
  // control suppresses the browser's own behaviour (this is what killed every
  // dropdown in the Director panels -- see kernel85-cohort-tools.js).
  const INTERACTIVE_SELECTOR =
    "button, select, input, textarea, option, optgroup, label, a, [contenteditable=''], [contenteditable='true']";

  function ensureHud() {
    if (state.hud) return state.hud;
    const el = document.createElement("div");
    el.id = "kernel88-player-hud";
    el.style.cssText = `
      position: fixed; z-index: 8500; width: 280px;
      background: rgba(20, 22, 27, 0.92); color: #f0f0f4; border: 1px solid #3a3f4b;
      border-radius: 12px; box-shadow: 0 10px 32px rgba(0,0,0,0.5);
      font: 13px/1.4 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      backdrop-filter: blur(6px); display: none;
    `;
    el.innerHTML = `
      <div data-hud-header style="display:flex; align-items:center; gap:8px; padding:8px 10px; cursor:move; user-select:none;">
        <span data-hud-title style="flex:1; font-size:12px; font-weight:600; letter-spacing:0.02em; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">Socio</span>
        <button type="button" data-hud-toggle title="Minimize"
          style="background:rgba(255,255,255,0.08); color:#f0f0f4; border:1px solid rgba(255,255,255,0.18); border-radius:6px; width:22px; height:22px; line-height:1; cursor:pointer; padding:0; font-size:13px;">–</button>
      </div>
      <div data-hud-body style="padding:0 14px 14px 14px;"></div>
    `;
    document.body.appendChild(el);
    state.hud = el;
    state.header = el.querySelector("[data-hud-header]");
    state.body = el.querySelector("[data-hud-body]");

    el.querySelector("[data-hud-toggle]").addEventListener("click", () => {
      state.collapsed = !state.collapsed;
      saveChrome();
      applyChrome();
      if (!state.collapsed) {
        // Collapsed ticks skip the network entirely, so whatever is in the
        // body is stale by definition on the way back out.
        state.forceRedraw = true;
        render();
      }
    });

    makeDraggable(el, state.header);
    applyChrome();
    return el;
  }

  function applyChrome() {
    const el = state.hud;
    if (!el) return;
    if (state.left != null && state.top != null) {
      el.style.left = state.left + "px";
      el.style.top = state.top + "px";
      el.style.bottom = "auto";
    } else {
      el.style.left = "12px";
      el.style.bottom = "12px";
      el.style.top = "auto";
    }
    const toggle = el.querySelector("[data-hud-toggle]");
    if (state.collapsed) {
      el.style.width = "auto";
      state.body.style.display = "none";
      toggle.textContent = "+";
      toggle.title = "Open Socio panel";
    } else {
      el.style.width = "280px";
      state.body.style.display = "block";
      toggle.textContent = "–";
      toggle.title = "Minimize";
    }
  }

  function makeDraggable(panel, handle) {
    let dragging = false;
    let startX = 0, startY = 0, startLeft = 0, startTop = 0;
    handle.addEventListener("mousedown", (event) => {
      if (event.target.closest(INTERACTIVE_SELECTOR)) return;
      const rect = panel.getBoundingClientRect();
      dragging = true;
      startX = event.clientX; startY = event.clientY;
      startLeft = rect.left; startTop = rect.top;
      event.preventDefault();
    });
    window.addEventListener("mousemove", (event) => {
      if (!dragging) return;
      const rect = panel.getBoundingClientRect();
      state.left = Math.min(Math.max(0, startLeft + (event.clientX - startX)), window.innerWidth - rect.width);
      state.top = Math.min(Math.max(0, startTop + (event.clientY - startY)), window.innerHeight - rect.height);
      panel.style.left = state.left + "px";
      panel.style.top = state.top + "px";
      panel.style.bottom = "auto";
    });
    window.addEventListener("mouseup", () => {
      if (!dragging) return;
      dragging = false;
      saveChrome();
    });
  }

  function stanceLabel(key) {
    const found = state.stanceRegistry.find((s) => s.key === key);
    return found ? found.label : key;
  }

  // A circular wheel of Stance slices. Active slice is marked with both a
  // border AND a filled dot -- never color alone (spec §5.1).
  function renderStanceWheel(activeKey) {
    const stances = state.stanceRegistry.length ? state.stanceRegistry : STANCE_ORDER.map((k) => ({ key: k, label: k }));
    const n = stances.length || 1;
    const sliceDeg = 360 / n;
    // One hue, stepped in lightness -- a full hue rotation gave each Stance
    // an unearned connotation (Insight came out cautionary red). Stance
    // carries no valence, so the slices must not imply one; the active slice
    // is still distinguished by fill AND border, never by colour alone.
    const gradientStops = stances.map((s, i) => {
      const start = i * sliceDeg;
      const end = start + sliceDeg;
      const lightness = 16 + Math.round((i / Math.max(n - 1, 1)) * 16); // 16%..32%
      const active = s.key === activeKey;
      return `hsl(${WHEEL_HUE},${active ? 38 : 20}%,${active ? lightness + 14 : lightness}%) ${start}deg ${end}deg`;
    }).join(", ");

    const buttons = stances.map((s, i) => {
      const mid = (i + 0.5) * sliceDeg - 90;
      const rad = (mid * Math.PI) / 180;
      const r = 46;
      const x = 50 + r * Math.cos(rad);
      const y = 50 + r * Math.sin(rad);
      const isActive = s.key === activeKey;
      return `
        <button type="button" data-set-stance="${s.key}" title="${escapeHtml(s.label)}"
          style="position:absolute; left:${x}%; top:${y}%; transform:translate(-50%,-50%);
                 width:${isActive ? 26 : 20}px; height:${isActive ? 26 : 20}px; border-radius:50%;
                 background:${isActive ? "#fff" : "rgba(255,255,255,0.15)"};
                 border:${isActive ? "2px solid #fff" : "1px solid rgba(255,255,255,0.35)"};
                 cursor:pointer; padding:0;">
          ${isActive ? "" : ""}
        </button>
      `;
    }).join("");

    return `
      <div style="position:relative; width:120px; height:120px; margin:6px auto;">
        <div style="position:absolute; inset:0; border-radius:50%; background:conic-gradient(${gradientStops}); box-shadow: inset 0 0 0 1px rgba(255,255,255,0.15);"></div>
        <div style="position:absolute; left:50%; top:50%; width:22px; height:22px; transform:translate(-50%,-50%); border-radius:50%; background:#14161b; box-shadow: 0 0 0 2px rgba(255,255,255,0.2);"></div>
        ${buttons}
      </div>
      <div style="text-align:center; font-size:12px; opacity:0.85; margin-top:2px;">
        ${activeKey ? "Stance: <strong>" + escapeHtml(stanceLabel(activeKey)) + "</strong>" : "No Stance chosen"}
      </div>
    `;
  }

  async function render() {
    const b = bridge();
    const showID = b?.getShowID?.() || "";
    const characterID = b?.getMyCharacterID?.() || "";
    const hud = ensureHud();

    if (!showID || !characterID) {
      hud.style.display = "none";
      state.lastFingerprint = "";
      return;
    }
    hud.style.display = "block";
    // Collapsed is the resting state: the launcher stays reachable but the
    // poll does no network at all until the Player opens the panel.
    if (state.collapsed) return;
    if (state.busy) return; // never let two polls overlap
    state.busy = true;
    try {
      await renderInner(b, hud, showID, characterID);
    } finally {
      state.busy = false;
    }
  }

  async function renderInner(b, hud, showID, characterID) {
    // Identity changes force a full reload; otherwise the Socio projection
    // is re-read on every tick so that Director-side changes (Fate awards,
    // applied statuses, a forced Stance) actually reach the Player. The
    // rendered DOM is only rewritten when the fetched data differs from
    // what is already on screen -- polling alone must not thrash the HUD
    // or steal focus from the Spend Fate field.
    const identityChanged = showID !== state.lastShowID || characterID !== state.lastCharacterID;
    if (identityChanged) {
      state.sheet = null;
      state.lastFingerprint = "";
    }
    state.lastShowID = showID;
    state.lastCharacterID = characterID;

    if (!state.stanceRegistry.length) {
      try { state.stanceRegistry = (await api("/api/socio/stances")).stances || []; } catch (_e) { /* best-effort */ }
    }

    let projection = null;
    try {
      projection = (await api(`/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterID)}/socio/view`)).projection;
    } catch (err) {
      state.body.innerHTML = `<div style="opacity:0.7;">Socio unavailable: ${escapeHtml(err.message)}</div>`;
      state.lastFingerprint = "";
      return;
    }
    if (identityChanged || !state.sheet) {
      try {
        // venue-sheet resolves the equipped session persona (a separate
        // mechanism from the Show-Run roster selection socio/view uses --
        // see runtime.js's getMyCharacterID comment) and 404s cleanly until
        // the Player has equipped one; the Face section is additive, so its
        // absence must never block Fate/Stance/pools above.
        state.sheet = (await api(`/api/characters/venue-sheet?session_id=${encodeURIComponent(b.getSessionID?.() || "")}`)).sheet;
      } catch (_err) {
        state.sheet = null;
      }
    }
    const sheet = state.sheet;

    // Mechanics come from the roster Character, NOT from the equipped-persona
    // venue-sheet above: the roll path authorizes against the roster
    // selection, so sourcing the buttons from anywhere else can offer a roll
    // the server will refuse -- or, as it did, offer none at all.
    if (identityChanged || state.mechanics == null) {
      try {
        const resp = await api(
          `/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterID)}/socio/mechanics`
        );
        state.mechanics = resp.mechanics || [];
        // The roster Character's name, so the HUD identifies the Character
        // the Player is actually playing even when no persona is equipped
        // and the venue-sheet above 404s.
        state.characterName = resp.character_name || "";
        state.mechanicsError = "";
      } catch (err) {
        state.mechanics = [];
        state.characterName = "";
        state.mechanicsError = err.message;
      }
    }

    const fingerprint = JSON.stringify([
      projection,
      sheet?.name || "",
      sheet?.portrait_url || "",
      (state.mechanics || []).map((s) => s.skill_id),
      state.characterName,
      state.mechanicsError,
    ]);
    if (!state.forceRedraw && fingerprint === state.lastFingerprint) return;
    state.forceRedraw = false;
    state.lastFingerprint = fingerprint;
    // Carry a half-typed Fate amount across a redraw the Player did not ask for.
    const priorSpendAmount = state.body.querySelector("[data-spend-amount]")?.value;

    const pools = projection.qualitative_pools || [];
    const flags = projection.flags || [];
    const skills = state.mechanics || [];

    state.body.innerHTML = `
      <div style="display:flex; align-items:center; gap:10px;">
        ${sheet?.portrait_url ? `<img src="${escapeHtml(sheet.portrait_url)}" alt="" style="width:44px;height:44px;border-radius:50%;object-fit:cover;border:1px solid #454b59;">` : `<div style="width:44px;height:44px;border-radius:50%;background:#2b303b;"></div>`}
        <div>
          <div style="font-weight:600; font-size:14px;">${escapeHtml(sheet?.name || state.characterName || "Your Character")}</div>
          <div style="opacity:0.7; font-size:11px;">Fate: <strong data-fate-balance>${projection.fate_balance ?? 0}</strong></div>
        </div>
      </div>

      ${renderStanceWheel(projection.stance_key)}

      <div style="margin-top:10px;">
        ${pools.map((p) => `
          <div style="display:flex; justify-content:space-between; font-size:12px; padding:2px 0; opacity:0.9;">
            <span>${escapeHtml(p.label)}</span><span>${escapeHtml(p.condition)}</span>
          </div>
        `).join("")}
      </div>

      ${flags.length ? `
        <div style="margin-top:8px; display:flex; flex-wrap:wrap; gap:4px;">
          ${flags.map((f) => `<span style="font-size:11px; background:rgba(255,255,255,0.08); border-radius:10px; padding:2px 8px;">${escapeHtml(f.label)}</span>`).join("")}
        </div>
      ` : ""}

      <div style="margin-top:10px; display:flex; gap:6px;">
        <input type="number" data-spend-amount value="1" min="1" style="width:44px; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px;">
        <button type="button" data-spend-fate style="flex:1; background:#2b303b; color:#e8e8ec; border:1px solid #454b59; border-radius:6px; padding:4px 8px; cursor:pointer; font-size:12px;">Spend Fate</button>
      </div>

      <div style="margin-top:10px; border-top:1px solid rgba(255,255,255,0.12); padding-top:8px;">
        <div style="font-size:11px; opacity:0.7; margin-bottom:4px;">Roll a mechanic</div>
        ${skills.length ? `
          <div style="display:flex; flex-direction:column; gap:4px; max-height:120px; overflow-y:auto;">
            ${skills.map((s) => `
              <button type="button" data-roll-skill="${escapeHtml(s.skill_id)}"
                style="text-align:left; background:#20242c; color:#e8e8ec; border:1px solid #333844; border-radius:6px; padding:5px 8px; cursor:pointer; font-size:12px;">
                ${escapeHtml(s.skill_name)} <span style="opacity:0.6;">(${escapeHtml(s.expression)})</span>
              </button>
            `).join("")}
          </div>
        ` : `
          <div style="font-size:11px; opacity:0.6;">
            ${state.mechanicsError
              ? "Mechanics unavailable: " + escapeHtml(state.mechanicsError)
              : "This Character has no rollable mechanics yet."}
          </div>
        `}
      </div>

      <div data-status style="margin-top:8px; font-size:11px; opacity:0.7;"></div>
    `;

    // The header is what remains visible when the panel is minimized, so it
    // carries the two facts worth glancing at without opening anything.
    const title = state.hud.querySelector("[data-hud-title]");
    if (title) {
      title.textContent = `${sheet?.name || state.characterName || "Socio"} · Fate ${projection.fate_balance ?? 0}`;
    }

    if (priorSpendAmount != null) {
      const field = state.body.querySelector("[data-spend-amount]");
      if (field) field.value = priorSpendAmount;
    }

    const status = (msg) => { const s = state.body.querySelector("[data-status]"); if (s) s.textContent = msg || ""; };

    state.body.querySelectorAll("[data-set-stance]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        try {
          await api(`/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterID)}/socio/stance`, {
            method: "POST", body: JSON.stringify({ stance_key: btn.dataset.setStance }),
          });
          state.forceRedraw = true;
          await render();
        } catch (err) { status("Stance failed: " + err.message); }
      });
    });

    state.body.querySelector("[data-spend-fate]")?.addEventListener("click", async () => {
      const amount = Number(state.body.querySelector("[data-spend-amount]")?.value || 0);
      if (!amount) return;
      try {
        const res = await api(`/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterID)}/socio/fate/spend`, {
          method: "POST", body: JSON.stringify({ amount, note: "" }),
        });
        const el = state.body.querySelector("[data-fate-balance]");
        if (el) el.textContent = String(res.fate?.balance ?? "?");
        status("Fate spent.");
      } catch (err) { status("Spend failed: " + err.message); }
    });

    state.body.querySelectorAll("[data-roll-skill]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const requestID = `k88-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`;
        const skillName = btn.textContent.trim().split(" (")[0] || "mechanic";

        // Kernel 88B: claim this request's failure before sending it. A
        // rolled mechanic that the server refuses used to report "Roll sent."
        // and then nothing at all -- the error frame existed but only reached
        // the generic stage surfaces, never the control that was pressed.
        // Registered first so the handler is in place even if the server
        // rejects faster than this function returns.
        const unregister = b?.registerActionRequest?.(requestID, (errorText) => {
          status(`${skillName} roll refused: ${errorText}`);
          return true; // claimed -- do not also announce it to the room
        });

        const ok = b?.sendAction?.("roll/dice_own_mechanic", {
          request_id: requestID,
          skill_id: btn.dataset.rollSkill,
          visibility: "",
        });
        if (!ok) {
          unregister?.();
          status("Could not send roll -- not connected.");
          return;
        }
        status(`${skillName} roll sent…`);
      });
    });
  }

  function init() {
    window.setInterval(render, 3000);
    render();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  window.VictoryKernel88PlayerHud = { render };
})();
