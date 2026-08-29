// Kernel 89: the Director's grouped live-play tool surface.
//
// §5's doctrine, stated as code: ONE tool button opens a selector of tool
// FAMILIES; a family opens one panel. Kernel 85/88 had grown to four
// equal-weight buttons pinned to the top-right of a live stage, and this
// kernel adds six more operations -- ten flat buttons would be exactly the
// button wall §34 forbids. So this module takes ownership of the toolbar
// (kernel85-cohort-tools.js stands down when it sees
// window.VictoryDirectorToolbarOwner) and re-presents the existing Kernel
// 85/88 panels as families alongside the new ones, calling straight into
// window.VictoryKernel85Tools rather than reimplementing any of them.
//
// Send Aftercare is the deliberate exception (§17): it stays a top-level
// button, because "can I find Aftercare quickly" is one of §42's operator
// questions and burying it one click deeper answers that question wrong.
//
// Nothing here is authoritative. Every operation is re-authorized server
// side; this file decides only what a Director is shown.
(function () {
  "use strict";

  window.VictoryDirectorToolbarOwner = "kernel89";

  const state = {
    toolbar: null,
    menu: null,
    announcePanel: null,
    rollPrepPanel: null,
    merchantPanel: null,
    aftercarePanel: null,
    styles: null,
    lastShowID: "",
    recalledTargetComplexity: null,
    recalledLabel: "",
    audienceMode: "show",
  };

  function bridge() {
    return window.VictoryStageKernel88Bridge || window.VictoryStageKernel85Bridge || null;
  }

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
      .replaceAll("\"", "&quot;").replaceAll("'", "&#39;");
  }

  async function api(path, options = {}) {
    const response = await fetch(path, {
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      ...options,
    });
    let body = null;
    try { body = await response.json(); } catch (_e) { /* no body */ }
    if (!response.ok || (body && body.ok === false)) {
      throw new Error(body?.data?.error || body?.error || `request_failed_${response.status}`);
    }
    return body?.data ?? body;
  }

  const PANEL_BASE_STYLE = `
    position: fixed; z-index: 9000; background: #1b1e24; color: #e8e8ec;
    border: 1px solid #3a3f4b; border-radius: 8px; box-shadow: 0 8px 28px rgba(0,0,0,0.45);
    font: 13px/1.4 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  `;

  function buttonStyle(extra = "") {
    return `background:#2b303b; color:#e8e8ec; border:1px solid #454b59; border-radius:6px; padding:6px 10px; cursor:pointer; font-size:12px; ${extra}`;
  }

  function inputStyle(extra = "") {
    return `background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px; font-size:12px; padding:4px 6px; ${extra}`;
  }

  function closeButton(onClose) {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.textContent = "✕";
    btn.setAttribute("aria-label", "Close");
    btn.style.cssText = "background:transparent; border:none; color:#aab; cursor:pointer; font-size:14px; float:right;";
    btn.addEventListener("click", onClose);
    return btn;
  }

  // Kernel 88A's B1 defect in one constant: a drag handler that
  // preventDefault()s over a <select> kills the browser's own popup. Any
  // panel that drags must give way to real controls.
  const INTERACTIVE_SELECTOR =
    "button, select, input, textarea, option, optgroup, label, a, [contenteditable=''], [contenteditable='true']";

  function makeDraggable(panel) {
    if (panel.dataset.k89Draggable === "1") return;
    panel.dataset.k89Draggable = "1";
    let dragging = false;
    let startX = 0, startY = 0, startLeft = 0, startTop = 0;
    panel.addEventListener("mousedown", (event) => {
      if (event.target.closest(INTERACTIVE_SELECTOR)) return;
      const rect = panel.getBoundingClientRect();
      dragging = true;
      startX = event.clientX; startY = event.clientY;
      startLeft = rect.left; startTop = rect.top;
      if (panel.style.transform) {
        panel.style.transform = "none";
        panel.style.left = startLeft + "px";
        panel.style.top = startTop + "px";
      }
      event.preventDefault();
    });
    window.addEventListener("mousemove", (event) => {
      if (!dragging) return;
      panel.style.left = Math.max(0, startLeft + (event.clientX - startX)) + "px";
      panel.style.top = Math.max(0, startTop + (event.clientY - startY)) + "px";
      panel.style.right = "auto";
    });
    window.addEventListener("mouseup", () => { dragging = false; });
  }

  // Announcements render centred over the stage; a Director tool panel
  // occupies the right edge. This flag lets the announcement stylesheet
  // give way for the Director without moving the banner for anyone else.
  const PANEL_KEYS = ["announcePanel", "rollPrepPanel", "merchantPanel", "aftercarePanel"];

  function syncPanelOpenClass() {
    const anyOpen = PANEL_KEYS.some((key) => state[key] && state[key].style.display === "block");
    document.body.classList.toggle("k89-director-panel-open", anyOpen);
  }

  function ensurePanel(key, id, extraStyle) {
    if (state[key] && document.body.contains(state[key])) return state[key];
    const el = document.createElement("div");
    el.id = id;
    el.style.cssText = PANEL_BASE_STYLE + extraStyle;
    document.body.appendChild(el);
    state[key] = el;
    return el;
  }

  function showID() {
    return bridge()?.getShowID?.() || "";
  }

  function statusSetter(panel) {
    return (msg, isError) => {
      const s = panel.querySelector("[data-status]");
      if (!s) return;
      s.textContent = msg || "";
      s.style.color = isError ? "#e88" : "#8bd18b";
    };
  }

  // --- Toolbar and the family selector ------------------------------------

  // The tool families. Order is the order a Director reaches for them
  // during a scene, not alphabetical: say something, set a number, hand out
  // a thing, change someone's state, move the stage, manage the group.
  // `available` keeps a family out of the menu when the module that
  // implements it is not loaded on this page. First Theater loads the
  // announcement renderer but not Kernel 85's Socio tools (§10 asks for
  // announcements in both venues; it does not ask for Cohorts in First
  // Theater), and a menu entry that silently does nothing is worse than an
  // absent one.
  const socioToolsPresent = () => Boolean(window.VictoryKernel85Tools);

  const ALL_FAMILIES = [
    { key: "announce", label: "Announce…", hint: "Theatrical banners", open: () => openAnnouncePanel() },
    { key: "roll-prep", label: "Roll Prep…", hint: "Prepared target complexities", open: () => openRollPrepPanel() },
    { key: "merchant", label: "Merchant…", hint: "Author and expose a merchant", open: () => openMerchantPanel() },
    { key: "state", label: "State…", hint: "Fate, statuses, pools", available: socioToolsPresent, open: () => window.VictoryKernel85Tools?.openGameStatusPanel?.() },
    { key: "stage", label: "Stage…", hint: "Scene configuration", available: socioToolsPresent, open: () => window.VictoryKernel85Tools?.openSceneConfiguration?.() },
    { key: "cohorts", label: "Cohorts…", hint: "Who plays together", available: socioToolsPresent, open: () => window.VictoryKernel85Tools?.openCohortPanel?.() },
    { key: "turn", label: "Turn & Help…", hint: "Current Turn and the interrupt stack", available: socioToolsPresent, open: () => {
      window.VictoryKernel85Tools?.openCurrentTurnPanel?.();
      window.VictoryKernel85Tools?.openInterruptPanel?.();
    } },
  ];

  function families() {
    return ALL_FAMILIES.filter((f) => (f.available ? f.available() : true));
  }

  function ensureToolbar() {
    if (state.toolbar && document.body.contains(state.toolbar)) return state.toolbar;
    const el = document.createElement("div");
    el.id = "kernel89-director-toolbar";
    el.style.cssText = PANEL_BASE_STYLE + "top: 12px; right: 12px; padding: 6px; display: none; gap: 6px; flex-direction: row; align-items: center;";
    el.innerHTML = `
      <button type="button" data-k89="tools" aria-haspopup="true" aria-expanded="false" style="${buttonStyle()}">Director Tools ▾</button>
      <button type="button" data-k89="aftercare" style="${buttonStyle("border-color:#6a7cff;")}">Send Aftercare</button>
      <span data-k89-recall style="display:none; font-size:11px; opacity:0.85; padding-left:4px; border-left:1px solid #3a3f4b;"></span>
    `;
    el.querySelector('[data-k89="tools"]').addEventListener("click", toggleMenu);
    el.querySelector('[data-k89="aftercare"]').addEventListener("click", () => openAftercarePanel());
    document.body.appendChild(el);
    state.toolbar = el;
    return el;
  }

  function ensureMenu() {
    if (state.menu && document.body.contains(state.menu)) return state.menu;
    const el = document.createElement("div");
    el.id = "kernel89-director-menu";
    el.setAttribute("role", "menu");
    el.style.cssText = PANEL_BASE_STYLE + "top: 52px; right: 12px; padding: 6px; display: none; min-width: 240px;";
    el.innerHTML = families().map((f) => `
      <button type="button" role="menuitem" data-family="${f.key}"
        style="${buttonStyle("display:block; width:100%; text-align:left; border:none; background:transparent; padding:8px 10px;")}">
        <span style="display:block;">${escapeHtml(f.label)}</span>
        <span style="display:block; font-size:11px; opacity:0.6;">${escapeHtml(f.hint)}</span>
      </button>
    `).join("");
    el.querySelectorAll("[data-family]").forEach((btn) => {
      btn.addEventListener("mouseenter", () => { btn.style.background = "#2b303b"; });
      btn.addEventListener("mouseleave", () => { btn.style.background = "transparent"; });
      btn.addEventListener("click", () => {
        const family = ALL_FAMILIES.find((f) => f.key === btn.dataset.family);
        closeMenu();
        family?.open?.();
      });
    });
    document.body.appendChild(el);
    state.menu = el;
    return el;
  }

  function closeMenu() {
    const menu = state.menu;
    if (menu) menu.style.display = "none";
    state.toolbar?.querySelector('[data-k89="tools"]')?.setAttribute("aria-expanded", "false");
  }

  function toggleMenu() {
    const menu = ensureMenu();
    const open = menu.style.display !== "block";
    menu.style.display = open ? "block" : "none";
    state.toolbar?.querySelector('[data-k89="tools"]')?.setAttribute("aria-expanded", open ? "true" : "false");
  }

  document.addEventListener("click", (event) => {
    if (!state.menu || state.menu.style.display !== "block") return;
    if (state.menu.contains(event.target)) return;
    if (state.toolbar?.contains(event.target)) return;
    closeMenu();
  });

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") closeMenu();
  });

  // --- Announce ------------------------------------------------------------

  async function loadStyles() {
    if (state.styles) return state.styles;
    const data = await api("/api/announcement-styles");
    state.styles = data.styles || [];
    return state.styles;
  }

  function audienceOptions(selected) {
    // Exactly Kernel 86's roll-audience vocabulary, reused rather than a
    // parallel one. §9.3's "no arbitrary whole-Show Director targeting" is
    // about merchant exposure; an announcement to the room IS the room's
    // Show audience, which is what "cohort" widens to when the Director is
    // ungrouped.
    return [
      ["cohort", "This Cohort (or the room)"],
      ["show", "Everyone in the Show"],
      ["director", "Directors only"],
    ].map(([value, label]) =>
      `<option value="${value}" ${value === selected ? "selected" : ""}>${escapeHtml(label)}</option>`
    ).join("");
  }

  async function openAnnouncePanel() {
    if (!showID()) return;
    const panel = ensurePanel("announcePanel", "kernel89-announce-panel",
      "top: 60px; right: 12px; width: 380px; max-height: 76vh; overflow-y: auto; padding: 12px; display: none;");
    panel.style.display = "block";
    syncPanelOpenClass();
    panel.innerHTML = "<div>Loading announcement palette…</div>";
    await renderAnnouncePanel();
  }

  async function renderAnnouncePanel() {
    const panel = state.announcePanel;
    const id = showID();
    let styles;
    let saved = [];
    try {
      styles = await loadStyles();
      saved = (await api(`/api/shows/${encodeURIComponent(id)}/director-preparations?kind=announcement`)).preparations || [];
    } catch (err) {
      panel.innerHTML = `<div style="color:#e88;">Failed to load: ${escapeHtml(err.message)}</div>`;
      return;
    }

    const presets = styles.filter((s) => s.key !== "custom");
    const presetGrid = presets.map((s) => `
      <button type="button" data-send-style="${escapeHtml(s.key)}" title="${escapeHtml(s.default_text)}"
        style="${buttonStyle(`text-align:left; border-color:${s.accent}; background:${s.background}; color:${s.ink}; padding:8px 10px;`)}">
        <span aria-hidden="true" style="color:${s.accent};">${escapeHtml(s.glyph)}</span>
        <strong style="margin-left:6px;">${escapeHtml(s.label)}</strong>
        <span style="display:block; font-size:10px; opacity:0.75; margin-top:2px;">${escapeHtml(s.default_text)}</span>
      </button>
    `).join("");

    const styleOptions = styles.map((s) =>
      `<option value="${escapeHtml(s.key)}">${escapeHtml(s.label)}</option>`
    ).join("");

    const savedRows = saved.map((p) => `
      <div style="display:flex; justify-content:space-between; align-items:center; gap:6px; padding:4px 0; border-bottom:1px solid #2a2e37;">
        <span style="flex:1; min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">
          ${escapeHtml(p.label)}
          <span style="opacity:0.55; font-size:11px;"> · ${escapeHtml(String(p.payload?.style || ""))}</span>
        </span>
        <button type="button" data-send-saved="${p.id}" style="${buttonStyle("padding:2px 8px; font-size:11px;")}">Send</button>
        <button type="button" data-delete-prep="${p.id}" style="${buttonStyle("padding:2px 6px; font-size:11px;")}">✕</button>
      </div>
    `).join("") || `<div style="opacity:0.6; font-size:11px;">No saved announcements yet.</div>`;

    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 4px 0; font-size:14px;">Announce</h3>
      <p style="margin:0 0 10px 0; font-size:11px; opacity:0.7;">
        You decide what the moment means. Victory never reads a roll and picks for you.
      </p>
      <label style="display:block; margin-bottom:10px; font-size:11px;">
        Who sees it
        <select data-audience style="${inputStyle("width:100%; margin-top:3px;")}">${audienceOptions(state.audienceMode)}</select>
      </label>
      <div style="display:grid; grid-template-columns:1fr 1fr; gap:6px; margin-bottom:12px;">${presetGrid}</div>
      <h4 style="margin:0 0 6px 0; font-size:12px;">Your own words</h4>
      <div style="display:flex; gap:6px; margin-bottom:6px;">
        <select data-custom-style style="${inputStyle("flex:0 0 130px;")}">${styleOptions}</select>
        <input type="text" data-custom-text placeholder="What happened?" maxlength="160" style="${inputStyle("flex:1;")}">
      </div>
      <div style="display:flex; gap:6px; margin-bottom:12px;">
        <button type="button" data-send-custom style="${buttonStyle("flex:1;")}">Send</button>
        <button type="button" data-save-custom style="${buttonStyle()}">Save as preset</button>
      </div>
      <h4 style="margin:0 0 6px 0; font-size:12px;">Saved</h4>
      ${savedRows}
      <div data-status style="margin-top:10px; font-size:11px;"></div>
    `;
    panel.prepend(closeButton(() => { panel.style.display = "none"; syncPanelOpenClass(); }));
    makeDraggable(panel);
    const status = statusSetter(panel);

    panel.querySelector("[data-audience]")?.addEventListener("change", (event) => {
      state.audienceMode = event.target.value;
    });

    const send = (style, text) => {
      const b = bridge();
      if (!b?.sendAction) { status("Stage socket unavailable.", true); return; }
      const requestId = "k89ann-" + Math.random().toString(36).slice(2, 10);
      // Kernel 88B: claim the refusal so it lands here rather than being
      // announced to the whole room as a generic chat notice.
      b.registerActionRequest?.(requestId, (errorText) => {
        status("Announcement refused: " + errorText, true);
        return true;
      });
      const ok = b.sendAction("announce/push", {
        request_id: requestId,
        style,
        text: text || "",
        visibility: state.audienceMode,
      });
      status(ok ? "Announced." : "Stage socket unavailable.", !ok);
    };

    panel.querySelectorAll("[data-send-style]").forEach((btn) => {
      btn.addEventListener("click", () => send(btn.dataset.sendStyle, ""));
    });

    panel.querySelector("[data-send-custom]")?.addEventListener("click", () => {
      const style = panel.querySelector("[data-custom-style]").value;
      const text = panel.querySelector("[data-custom-text]").value.trim();
      send(style, text);
    });

    panel.querySelector("[data-save-custom]")?.addEventListener("click", async () => {
      const style = panel.querySelector("[data-custom-style]").value;
      const text = panel.querySelector("[data-custom-text]").value.trim();
      const label = window.prompt("Name this announcement:", text.slice(0, 60));
      if (!label) return;
      try {
        await api(`/api/shows/${encodeURIComponent(id)}/director-preparations`, {
          method: "POST",
          body: JSON.stringify({ kind: "announcement", label, payload: { style, text } }),
        });
        await renderAnnouncePanel();
      } catch (err) { status("Save failed: " + err.message, true); }
    });

    panel.querySelectorAll("[data-send-saved]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const prep = saved.find((p) => p.id === btn.dataset.sendSaved);
        if (!prep) return;
        send(String(prep.payload?.style || "custom"), String(prep.payload?.text || ""));
      });
    });

    panel.querySelectorAll("[data-delete-prep]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        try {
          await api(`/api/director-preparations/${encodeURIComponent(btn.dataset.deletePrep)}`, { method: "DELETE" });
          await renderAnnouncePanel();
        } catch (err) { status("Delete failed: " + err.message, true); }
      });
    });
  }

  // --- Roll Prep (prepared target complexities) ----------------------------

  function updateRecallChip() {
    const chip = state.toolbar?.querySelector("[data-k89-recall]");
    if (!chip) return;
    if (state.recalledTargetComplexity == null) {
      chip.style.display = "none";
      chip.textContent = "";
      return;
    }
    chip.style.display = "inline";
    chip.textContent = `TC ${state.recalledTargetComplexity} · ${state.recalledLabel}`;
  }

  async function openRollPrepPanel() {
    if (!showID()) return;
    const panel = ensurePanel("rollPrepPanel", "kernel89-roll-prep-panel",
      "top: 60px; right: 12px; width: 340px; max-height: 70vh; overflow-y: auto; padding: 12px; display: none;");
    panel.style.display = "block";
    syncPanelOpenClass();
    panel.innerHTML = "<div>Loading prepared complexities…</div>";
    await renderRollPrepPanel();
  }

  async function renderRollPrepPanel() {
    const panel = state.rollPrepPanel;
    const id = showID();
    let items = [];
    try {
      items = (await api(`/api/shows/${encodeURIComponent(id)}/director-preparations?kind=target_complexity`)).preparations || [];
    } catch (err) {
      panel.innerHTML = `<div style="color:#e88;">Failed to load: ${escapeHtml(err.message)}</div>`;
      return;
    }

    const rows = items.map((p) => `
      <div style="border-bottom:1px solid #2a2e37; padding:6px 0;">
        <div style="display:flex; align-items:center; gap:6px;">
          <strong style="flex:1; min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">${escapeHtml(p.label)}</strong>
          <input type="number" data-value-for="${p.id}" value="${Number(p.payload?.value || 0)}" min="1" max="99"
            style="${inputStyle("width:56px;")}">
          <button type="button" data-recall="${p.id}" style="${buttonStyle("padding:2px 8px; font-size:11px;")}">Recall</button>
          <button type="button" data-delete-prep="${p.id}" style="${buttonStyle("padding:2px 6px; font-size:11px;")}">✕</button>
        </div>
        ${p.payload?.note ? `<div style="font-size:11px; opacity:0.65; margin-top:3px;">${escapeHtml(p.payload.note)}</div>` : ""}
      </div>
    `).join("") || `<div style="opacity:0.6; font-size:11px;">Nothing prepared yet.</div>`;

    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 4px 0; font-size:14px;">Roll Prep</h3>
      <p style="margin:0 0 10px 0; font-size:11px; opacity:0.7;">
        Prepared target complexities. Recalling one shows it to you — it never rolls,
        never picks a skill, and never decides whether the attempt makes sense.
      </p>
      ${rows}
      <div style="margin-top:10px; display:flex; gap:6px;">
        <input type="text" data-new-label placeholder="Label, e.g. Climb Training Wall" style="${inputStyle("flex:1;")}">
        <input type="number" data-new-value placeholder="TC" min="1" max="99" style="${inputStyle("width:56px;")}">
        <button type="button" data-create style="${buttonStyle()}">Save</button>
      </div>
      <div data-status style="margin-top:10px; font-size:11px;"></div>
    `;
    panel.prepend(closeButton(() => { panel.style.display = "none"; syncPanelOpenClass(); }));
    makeDraggable(panel);
    const status = statusSetter(panel);

    panel.querySelectorAll("[data-recall]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const prep = items.find((p) => p.id === btn.dataset.recall);
        if (!prep) return;
        const live = Number(panel.querySelector(`[data-value-for="${prep.id}"]`)?.value || prep.payload?.value || 0);
        state.recalledTargetComplexity = live;
        state.recalledLabel = prep.label;
        updateRecallChip();
        status(`Recalled ${prep.label} at TC ${live}. It will prefill the next Interrupt.`);
      });
    });

    // Changing the number saves it (§7: "Director can change the value
    // live"). On "change" rather than per keystroke, so a half-typed "1" on
    // the way to "14" is never persisted.
    panel.querySelectorAll("[data-value-for]").forEach((input) => {
      input.addEventListener("change", async () => {
        const prepID = input.getAttribute("data-value-for");
        try {
          await api(`/api/director-preparations/${encodeURIComponent(prepID)}`, {
            method: "PATCH",
            body: JSON.stringify({ payload: { value: Number(input.value || 0) } }),
          });
          status("Updated.");
        } catch (err) { status("Update failed: " + err.message, true); }
      });
    });

    panel.querySelector("[data-create]")?.addEventListener("click", async () => {
      const label = panel.querySelector("[data-new-label]").value.trim();
      const value = Number(panel.querySelector("[data-new-value]").value || 0);
      if (!label || !value) { status("A label and a number are both required.", true); return; }
      try {
        await api(`/api/shows/${encodeURIComponent(id)}/director-preparations`, {
          method: "POST",
          body: JSON.stringify({ kind: "target_complexity", label, payload: { value } }),
        });
        await renderRollPrepPanel();
      } catch (err) { status("Save failed: " + err.message, true); }
    });

    panel.querySelectorAll("[data-delete-prep]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        try {
          await api(`/api/director-preparations/${encodeURIComponent(btn.dataset.deletePrep)}`, { method: "DELETE" });
          await renderRollPrepPanel();
        } catch (err) { status("Delete failed: " + err.message, true); }
      });
    });
  }

  // --- Merchant ------------------------------------------------------------

  async function openMerchantPanel() {
    if (!showID()) return;
    const panel = ensurePanel("merchantPanel", "kernel89-merchant-panel",
      "top: 60px; right: 12px; width: 420px; max-height: 78vh; overflow-y: auto; padding: 12px; display: none;");
    panel.style.display = "block";
    syncPanelOpenClass();
    panel.innerHTML = "<div>Loading merchants…</div>";
    await renderMerchantPanel();
  }

  async function renderMerchantPanel(selectedPacketID = "") {
    const panel = state.merchantPanel;
    const id = showID();
    let data;
    let cohorts = [];
    try {
      data = await api(`/api/shows/${encodeURIComponent(id)}/merchant-packets`);
      cohorts = (await api(`/api/shows/${encodeURIComponent(id)}/cohorts`)).cohorts || [];
    } catch (err) {
      panel.innerHTML = `<div style="color:#e88;">Failed to load: ${escapeHtml(err.message)}</div>`;
      return;
    }
    const packets = data.packets || [];
    const catalog = (data.catalog || []).filter((item) => item.active);
    const selected = packets.find((p) => p.id === selectedPacketID) || packets[0] || null;
    const stockIDs = new Set((selected?.stock || []).map((s) => s.id));

    const packetOptions = packets.map((p) =>
      `<option value="${p.id}" ${selected && p.id === selected.id ? "selected" : ""}>${escapeHtml(p.display_name)}</option>`
    ).join("");

    const catalogRows = catalog.map((item) => `
      <label style="display:flex; align-items:center; gap:6px; padding:2px 0; font-size:12px;">
        <input type="checkbox" data-stock-item="${item.id}" ${stockIDs.has(item.id) ? "checked" : ""}>
        <span style="flex:1; min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">${escapeHtml(item.name)}</span>
        <span style="opacity:0.5; font-size:10px;">${escapeHtml(item.category || "")}</span>
      </label>
    `).join("");

    const cohortOptions = `<option value="">Everyone on this Scene</option>` + cohorts.map((c) =>
      `<option value="${c.id}">${escapeHtml(c.name)}</option>`
    ).join("");

    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 4px 0; font-size:14px;">Merchant</h3>
      <p style="margin:0 0 10px 0; font-size:11px; opacity:0.7;">
        Stock comes from the one canonical equipment catalog. Conversation stays roleplay.
      </p>
      <div style="display:flex; gap:6px; margin-bottom:10px;">
        <select data-packet-select style="${inputStyle("flex:1;")}">${packetOptions || `<option value="">No merchants yet</option>`}</select>
        <button type="button" data-new-packet style="${buttonStyle()}">+ New</button>
      </div>
      ${selected ? `
        <label style="display:block; font-size:11px; margin-bottom:6px;">Name
          <input type="text" data-packet-name value="${escapeHtml(selected.display_name)}" style="${inputStyle("width:100%; margin-top:3px;")}">
        </label>
        <label style="display:block; font-size:11px; margin-bottom:6px;">Opening line
          <textarea data-packet-intro rows="2" style="${inputStyle("width:100%; margin-top:3px; resize:vertical;")}">${escapeHtml(selected.intro_text)}</textarea>
        </label>
        <div style="font-size:11px; margin-bottom:4px;">Stock (${stockIDs.size} selected of ${catalog.length})</div>
        <div style="max-height:180px; overflow-y:auto; border:1px solid #333844; border-radius:6px; padding:6px; margin-bottom:8px;">
          ${catalogRows || `<div style="opacity:0.6;">No equipment in the catalog.</div>`}
        </div>
        <button type="button" data-save-packet style="${buttonStyle("width:100%; margin-bottom:12px;")}">Save merchant</button>

        <h4 style="margin:0 0 6px 0; font-size:12px;">Expose on the current Scene</h4>
        <div style="display:flex; gap:6px; margin-bottom:6px;">
          <input type="text" data-expose-label placeholder="Stage button label" value="Speak with ${escapeHtml(selected.display_name)}" style="${inputStyle("flex:1;")}">
        </div>
        <div style="display:flex; gap:6px;">
          <select data-expose-cohort style="${inputStyle("flex:1;")}">${cohortOptions}</select>
          <button type="button" data-expose style="${buttonStyle()}">Expose</button>
        </div>
      ` : `<div style="opacity:0.6; font-size:12px;">Create a merchant to begin.</div>`}
      <div data-status style="margin-top:10px; font-size:11px;"></div>
    `;
    panel.prepend(closeButton(() => { panel.style.display = "none"; syncPanelOpenClass(); }));
    makeDraggable(panel);
    const status = statusSetter(panel);

    panel.querySelector("[data-packet-select]")?.addEventListener("change", (event) => {
      renderMerchantPanel(event.target.value);
    });

    panel.querySelector("[data-new-packet]")?.addEventListener("click", async () => {
      const name = window.prompt("New merchant's name:");
      if (!name) return;
      try {
        const created = await api(`/api/shows/${encodeURIComponent(id)}/merchant-packets`, {
          method: "POST",
          body: JSON.stringify({ display_name: name, intro_text: "", equipment_item_ids: [] }),
        });
        await renderMerchantPanel(created.packet?.id || "");
      } catch (err) { status("Create failed: " + err.message, true); }
    });

    panel.querySelector("[data-save-packet]")?.addEventListener("click", async () => {
      if (!selected) return;
      const ids = Array.from(panel.querySelectorAll("[data-stock-item]"))
        .filter((cb) => cb.checked)
        .map((cb) => cb.getAttribute("data-stock-item"));
      try {
        await api(`/api/merchant-packets/${encodeURIComponent(selected.id)}`, {
          method: "PATCH",
          body: JSON.stringify({
            display_name: panel.querySelector("[data-packet-name]").value,
            intro_text: panel.querySelector("[data-packet-intro]").value,
            stance_dispositions: selected.stance_dispositions || {},
            haggle_success_text: selected.haggle_success_text || "",
            haggle_failure_text: selected.haggle_failure_text || "",
            return_label: selected.return_label || "",
            close_label: selected.close_label || "",
            equipment_item_ids: ids,
          }),
        });
        status("Merchant saved.");
        await renderMerchantPanel(selected.id);
      } catch (err) { status("Save failed: " + err.message, true); }
    });

    panel.querySelector("[data-expose]")?.addEventListener("click", async () => {
      if (!selected) return;
      const placementID = await currentPlacementID(id);
      if (!placementID) { status("This Show has no current Scene to attach to.", true); return; }
      const label = panel.querySelector("[data-expose-label]").value.trim() || `Speak with ${selected.display_name}`;
      const cohortID = panel.querySelector("[data-expose-cohort]").value;
      try {
        await api(`/api/shows/${encodeURIComponent(id)}/scenes/${encodeURIComponent(placementID)}/participant-interactions`, {
          method: "POST",
          body: JSON.stringify({
            internal_name: selected.display_name,
            stage_button_label: label,
            interaction_type: "open_equip_mode",
            configuration_json: { packet_slug: selected.slug, target_cohort_id: cohortID },
          }),
        });
        status(cohortID ? "Exposed to that Cohort." : "Exposed on this Scene.");
      } catch (err) { status("Expose failed: " + err.message, true); }
    });
  }

  async function currentPlacementID(id) {
    try {
      const data = await api(`/api/shows/${encodeURIComponent(id)}`);
      return String(data?.show?.current_show_scene_placement_id || data?.current_show_scene_placement_id || "");
    } catch (_err) {
      return "";
    }
  }

  // --- Aftercare -----------------------------------------------------------

  async function openAftercarePanel() {
    if (!showID()) return;
    const panel = ensurePanel("aftercarePanel", "kernel89-aftercare-panel",
      "top: 60px; right: 12px; width: 340px; max-height: 70vh; overflow-y: auto; padding: 12px; display: none;");
    panel.style.display = "block";
    syncPanelOpenClass();
    closeMenu();
    renderAftercarePanel();
  }

  async function renderAftercarePanel(result) {
    const panel = state.aftercarePanel;
    const id = showID();
    let cohorts = [];
    try {
      cohorts = (await api(`/api/shows/${encodeURIComponent(id)}/cohorts`)).cohorts || [];
    } catch (_err) { /* cohorts are optional context, not a blocker */ }

    const cohortOptions = `<option value="">Everyone playing</option>` + cohorts.map((c) =>
      `<option value="${c.id}">${escapeHtml(c.name)}</option>`
    ).join("");

    const receipt = result ? `
      <div style="margin-top:10px; border-top:1px solid #2a2e37; padding-top:8px;">
        <div style="font-size:12px; color:#8bd18b;">
          Sent to ${result.delivered} of ${result.targeted} ${result.targeted === 1 ? "Player" : "Players"}.
        </div>
        ${result.targeted && result.delivered < result.targeted ? `
          <div style="font-size:11px; opacity:0.75; margin-top:4px;">
            The rest are not connected right now. Their Aftercare is still waiting for them —
            it belongs to the Show, not to this Session.
          </div>` : ""}
        <ul style="margin:6px 0 0 0; padding-left:18px; font-size:11px; opacity:0.85;">
          ${(result.recipients || []).map((r) => `<li>${escapeHtml(r.display_name)}${r.character_name ? " · " + escapeHtml(r.character_name) : ""}${r.delivered ? "" : " (offline)"}</li>`).join("")}
        </ul>
      </div>` : "";

    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 4px 0; font-size:14px;">Send Aftercare</h3>
      <p style="margin:0 0 10px 0; font-size:11px; opacity:0.7;">
        Opens each Player's own Aftercare form. Nothing sends itself — ending the
        Session does not do this, and it never will unless you press this.
      </p>
      <label style="display:block; font-size:11px; margin-bottom:8px;">
        Who
        <select data-aftercare-cohort style="${inputStyle("width:100%; margin-top:3px;")}">${cohortOptions}</select>
      </label>
      <button type="button" data-send-aftercare style="${buttonStyle("width:100%; border-color:#6a7cff;")}">Send Aftercare now</button>
      ${receipt}
      <div data-status style="margin-top:10px; font-size:11px;"></div>
    `;
    panel.prepend(closeButton(() => { panel.style.display = "none"; syncPanelOpenClass(); }));
    makeDraggable(panel);
    const status = statusSetter(panel);

    panel.querySelector("[data-send-aftercare]")?.addEventListener("click", async () => {
      const cohortID = panel.querySelector("[data-aftercare-cohort]").value;
      status("Sending…");
      try {
        const data = await api(`/api/shows/${encodeURIComponent(id)}/aftercare/send`, {
          method: "POST",
          body: JSON.stringify({ cohort_id: cohortID }),
        });
        await renderAftercarePanel(data);
      } catch (err) { status("Send failed: " + err.message, true); }
    });
  }

  // --- Visibility ----------------------------------------------------------

  function pollToolbarVisibility() {
    const b = bridge();
    const id = b?.getShowID?.() || "";
    const canManage = Boolean(b?.canManageStage?.());
    const toolbar = ensureToolbar();
    toolbar.style.display = id && canManage ? "flex" : "none";
    if (id !== state.lastShowID) {
      state.lastShowID = id;
      state.recalledTargetComplexity = null;
      state.recalledLabel = "";
      state.styles = null;
      updateRecallChip();
      closeMenu();
      [state.announcePanel, state.rollPrepPanel, state.merchantPanel, state.aftercarePanel]
        .forEach((p) => { if (p) p.style.display = "none"; });
      syncPanelOpenClass();
    }
  }

  function init() {
    window.setInterval(pollToolbarVisibility, 1000);
    pollToolbarVisibility();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  window.VictoryKernel89DirectorTools = {
    openAnnouncePanel,
    openRollPrepPanel,
    openMerchantPanel,
    openAftercarePanel,
    openToolMenu: toggleMenu,
    families: () => families().map((f) => ({ key: f.key, label: f.label })),
    // Read by kernel85-cohort-tools.js to prefill an Interrupt's target
    // complexity with whatever the Director just recalled -- the one place
    // Roll Prep touches live mechanics, and it only ever prefills a field a
    // human still has to press a button next to.
    recalledTargetComplexity: () => state.recalledTargetComplexity,
    recalledLabel: () => state.recalledLabel,
  };
})();
