// Kernel 85: Show Cohorts, Scene Configuration, and Game Status -- a
// self-contained Director+ tool layer, deliberately separate from the
// generic stage engine (runtime.js et al never import anything from this
// file; it reads the engine only through the narrow read-only
// window.VictoryStageKernel85Bridge runtime.js exposes). All server
// authority is re-checked on every request by the backend
// (backend/internal/cohorts, backend/internal/socio, backend/internal/
// scenes' capture actions) -- everything here is presentation only.
(function () {
  "use strict";

  const POOL_LABELS = [
    ["health", "Health"], ["psyche", "Psyche"], ["motion", "Motion"], ["will", "Will"],
    ["essence", "Essence"], ["focus", "Focus"], ["perception", "Perception"], ["heart", "Heart"],
  ];

  const state = {
    selectedCohortId: "",
    toolbar: null,
    cohortPanel: null,
    scenePanel: null,
    statusPanel: null,
    lastShowID: "",
  };

  function bridge() {
    return window.VictoryStageKernel85Bridge || null;
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

  // --- Shell: floating toolbar ---------------------------------------

  const PANEL_BASE_STYLE = `
    position: fixed; z-index: 9000; background: #1b1e24; color: #e8e8ec;
    border: 1px solid #3a3f4b; border-radius: 8px; box-shadow: 0 8px 28px rgba(0,0,0,0.45);
    font: 13px/1.4 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  `;

  function ensureToolbar() {
    if (state.toolbar) return state.toolbar;
    const el = document.createElement("div");
    el.id = "kernel85-toolbar";
    el.style.cssText = PANEL_BASE_STYLE + "top: 12px; right: 12px; padding: 6px; display: none; gap: 6px; flex-direction: row;";
    el.innerHTML = `
      <button type="button" data-k85="cohorts" style="${buttonStyle()}">Cohorts</button>
      <button type="button" data-k85="game-status" style="${buttonStyle()}">Game Status</button>
    `;
    el.querySelector('[data-k85="cohorts"]').addEventListener("click", () => openCohortPanel());
    el.querySelector('[data-k85="game-status"]').addEventListener("click", () => openGameStatusPanel());
    document.body.appendChild(el);
    state.toolbar = el;
    return el;
  }

  function buttonStyle(extra = "") {
    return `background:#2b303b; color:#e8e8ec; border:1px solid #454b59; border-radius:6px; padding:6px 10px; cursor:pointer; font-size:12px; ${extra}`;
  }

  function closeButton(onClose) {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.textContent = "✕";
    btn.style.cssText = "background:transparent; border:none; color:#aab; cursor:pointer; font-size:14px; float:right;";
    btn.addEventListener("click", onClose);
    return btn;
  }

  function makeDraggable(panel, handle) {
    let dragging = false;
    let startX = 0, startY = 0, startLeft = 0, startTop = 0;
    handle.style.cursor = "move";
    handle.addEventListener("mousedown", (event) => {
      if (event.target.closest("button")) return;
      dragging = true;
      const rect = panel.getBoundingClientRect();
      startX = event.clientX; startY = event.clientY;
      startLeft = rect.left; startTop = rect.top;
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

  // --- Cohort management panel ----------------------------------------

  function ensureCohortPanel() {
    if (state.cohortPanel) return state.cohortPanel;
    const el = document.createElement("div");
    el.id = "kernel85-cohort-panel";
    el.style.cssText = PANEL_BASE_STYLE + "top: 60px; right: 12px; width: 360px; max-height: 70vh; overflow-y: auto; padding: 12px; display: none;";
    document.body.appendChild(el);
    state.cohortPanel = el;
    return el;
  }

  async function openCohortPanel() {
    const b = bridge();
    const showID = b?.getShowID?.();
    if (!showID) return;
    const panel = ensureCohortPanel();
    panel.style.display = "block";
    panel.innerHTML = `<div>Loading cohorts&hellip;</div>`;
    await renderCohortPanel(showID);
  }

  async function renderCohortPanel(showID) {
    const panel = state.cohortPanel;
    let roster;
    try {
      roster = await api(`/api/shows/${encodeURIComponent(showID)}/cohorts`);
    } catch (err) {
      panel.innerHTML = `<div style="color:#e88;">Failed to load cohorts: ${escapeHtml(err.message)}</div>`;
      return;
    }
    const cohorts = roster.cohorts || [];
    const ungrouped = roster.ungrouped || [];

    const rows = cohorts.map((c) => `
      <div style="border:1px solid #333844; border-radius:6px; padding:8px; margin-bottom:8px;">
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <strong>${escapeHtml(c.name)}</strong>
          <button type="button" data-archive-cohort="${c.id}" style="${buttonStyle("padding:2px 6px; font-size:11px;")}">Archive</button>
        </div>
        <div style="margin-top:6px;">
          ${(c.members || []).map((m) => `
            <div style="display:flex; justify-content:space-between; align-items:center; padding:3px 0;">
              <span>${escapeHtml(m.display_name || "Participant")}${m.character_name ? " &middot; " + escapeHtml(m.character_name) : ""}</span>
              <span>
                <select data-move-user="${m.user_id}" style="background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px; font-size:11px;">
                  <option value="">Move to&hellip;</option>
                  ${cohorts.filter((oc) => oc.id !== c.id).map((oc) => `<option value="${oc.id}">${escapeHtml(oc.name)}</option>`).join("")}
                  <option value="ungrouped">Ungrouped</option>
                </select>
              </span>
            </div>
          `).join("") || `<div style="opacity:0.6;">No members yet.</div>`}
        </div>
      </div>
    `).join("");

    const ungroupedRows = ungrouped.map((p) => `
      <div style="display:flex; justify-content:space-between; align-items:center; padding:3px 0;">
        <span>${escapeHtml(p.display_name || "Participant")}${p.character_name ? " &middot; " + escapeHtml(p.character_name) : ""}</span>
        <select data-assign-user="${p.user_id}" style="background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px; font-size:11px;">
          <option value="">Assign to&hellip;</option>
          ${cohorts.map((oc) => `<option value="${oc.id}">${escapeHtml(oc.name)}</option>`).join("")}
        </select>
      </div>
    `).join("") || `<div style="opacity:0.6;">Everyone is grouped.</div>`;

    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 8px 0; font-size:14px;">Cohorts</h3>
      <button type="button" data-create-cohort style="${buttonStyle("margin-bottom:10px;")}">+ New Cohort</button>
      ${rows || `<div style="opacity:0.6; margin-bottom:10px;">No cohorts yet.</div>`}
      <h4 style="margin:10px 0 4px 0; font-size:13px;">Ungrouped</h4>
      ${ungroupedRows}
      <div data-status style="margin-top:10px; font-size:11px; opacity:0.7;"></div>
    `;
    panel.prepend(closeButton(() => { panel.style.display = "none"; }));
    makeDraggable(panel, panel);

    const status = (msg) => { const s = panel.querySelector("[data-status]"); if (s) s.textContent = msg || ""; };

    panel.querySelector("[data-create-cohort]")?.addEventListener("click", async () => {
      try {
        await api(`/api/shows/${encodeURIComponent(showID)}/cohorts`, { method: "POST" });
        await renderCohortPanel(showID);
      } catch (err) { status("Create failed: " + err.message); }
    });
    panel.querySelectorAll("[data-archive-cohort]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        try {
          await api(`/api/shows/${encodeURIComponent(showID)}/cohorts/${encodeURIComponent(btn.dataset.archiveCohort)}/archive`, { method: "POST" });
          await renderCohortPanel(showID);
        } catch (err) { status("Archive failed: " + err.message); }
      });
    });
    panel.querySelectorAll("[data-move-user]").forEach((select) => {
      select.addEventListener("change", async () => {
        const userID = select.dataset.moveUser;
        const target = select.value;
        if (!target) return;
        try {
          if (target === "ungrouped") {
            await api(`/api/shows/${encodeURIComponent(showID)}/cohorts/assignments/${encodeURIComponent(userID)}`, { method: "DELETE" });
          } else {
            await api(`/api/shows/${encodeURIComponent(showID)}/cohorts/${encodeURIComponent(target)}/assignments`, {
              method: "POST", body: JSON.stringify({ user_id: userID }),
            });
          }
          await renderCohortPanel(showID);
        } catch (err) { status("Move failed: " + err.message); }
      });
    });
    panel.querySelectorAll("[data-assign-user]").forEach((select) => {
      select.addEventListener("change", async () => {
        const userID = select.dataset.assignUser;
        const target = select.value;
        if (!target) return;
        try {
          await api(`/api/shows/${encodeURIComponent(showID)}/cohorts/${encodeURIComponent(target)}/assignments`, {
            method: "POST", body: JSON.stringify({ user_id: userID }),
          });
          await renderCohortPanel(showID);
        } catch (err) { status("Assign failed: " + err.message); }
      });
    });

    if (!state.selectedCohortId && cohorts.length) {
      state.selectedCohortId = cohorts[0].id;
    }
  }

  // --- Scene Configuration modal ---------------------------------------

  function ensureScenePanel() {
    if (state.scenePanel) return state.scenePanel;
    const el = document.createElement("div");
    el.id = "kernel85-scene-panel";
    el.style.cssText = PANEL_BASE_STYLE + "top: 80px; left: 50%; transform: translateX(-50%); width: 420px; max-height: 75vh; overflow-y: auto; padding: 14px; display: none;";
    document.body.appendChild(el);
    state.scenePanel = el;
    return el;
  }

  async function openSceneConfiguration() {
    const b = bridge();
    const showID = b?.getShowID?.();
    if (!showID) return;
    if (!b?.canManageStage?.()) return;
    const panel = ensureScenePanel();
    panel.style.display = "block";
    panel.innerHTML = `<div>Loading Scene Configuration&hellip;</div>`;
    await renderSceneConfiguration(showID);
  }

  async function renderSceneConfiguration(showID) {
    const panel = state.scenePanel;
    let roster;
    let scenesResp;
    try {
      [roster, scenesResp] = await Promise.all([
        api(`/api/shows/${encodeURIComponent(showID)}/cohorts`),
        api(`/api/shows/${encodeURIComponent(showID)}/scenes`),
      ]);
    } catch (err) {
      panel.innerHTML = `<div style="color:#e88;">Failed to load: ${escapeHtml(err.message)}</div>`;
      return;
    }
    const cohorts = roster.cohorts || [];
    const placements = scenesResp?.placements || scenesResp || [];
    if (!state.selectedCohortId && cohorts.length) state.selectedCohortId = cohorts[0].id;
    const selected = cohorts.find((c) => c.id === state.selectedCohortId) || null;

    const cohortOptions = cohorts.map((c) =>
      `<option value="${c.id}" ${c.id === state.selectedCohortId ? "selected" : ""}>${escapeHtml(c.name)}</option>`
    ).join("");

    const sceneRows = (Array.isArray(placements) ? placements : []).map((p) => {
      const title = p.scene?.title || p.title || "Untitled Scene";
      const placementID = p.placement?.id || p.id;
      const isCurrent = selected?.current_show_scene_placement_id === placementID;
      return `
        <div style="display:flex; justify-content:space-between; align-items:center; padding:4px 0; border-bottom:1px solid #2a2e37;">
          <span>${escapeHtml(title)}${isCurrent ? " <em>(current)</em>" : ""}</span>
          <button type="button" data-activate="${placementID}" style="${buttonStyle("padding:3px 8px; font-size:11px;")}" ${isCurrent ? "disabled" : ""}>Activate</button>
        </div>
      `;
    }).join("") || `<div style="opacity:0.6;">No Scenes attached to this Show yet.</div>`;

    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 10px 0; font-size:15px;">Scene Configuration</h3>
      <label style="display:block; margin-bottom:10px;">
        Cohort:
        <select data-select-cohort style="margin-left:6px; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px;">
          ${cohortOptions}
          <option value="ungrouped" ${state.selectedCohortId === "ungrouped" ? "selected" : ""}>Ungrouped (no per-cohort Scene)</option>
        </select>
      </label>
      <div style="margin-bottom:10px;">${sceneRows}</div>
      <div style="display:flex; gap:8px; margin-top:10px;">
        <button type="button" data-update-current style="${buttonStyle()}" ${selected ? "" : "disabled"}>Update Current Scene</button>
        <button type="button" data-save-as-new style="${buttonStyle()}" ${selected ? "" : "disabled"}>Save as New Scene</button>
      </div>
      <div data-status style="margin-top:10px; font-size:11px; opacity:0.7;"></div>
    `;
    panel.prepend(closeButton(() => { panel.style.display = "none"; }));
    makeDraggable(panel, panel);

    const status = (msg) => { const s = panel.querySelector("[data-status]"); if (s) s.textContent = msg || ""; };

    panel.querySelector("[data-select-cohort]")?.addEventListener("change", (event) => {
      state.selectedCohortId = event.target.value;
      renderSceneConfiguration(showID);
    });

    panel.querySelectorAll("[data-activate]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        if (!selected) { status("Select a real cohort first -- Ungrouped has no per-cohort Scene."); return; }
        try {
          await api(`/api/shows/${encodeURIComponent(showID)}/cohorts/${encodeURIComponent(selected.id)}/current-scene`, {
            method: "POST", body: JSON.stringify({ show_scene_placement_id: btn.dataset.activate }),
          });
          status("Scene activated for " + selected.name + ".");
          await renderSceneConfiguration(showID);
        } catch (err) { status("Activate failed: " + err.message); }
      });
    });

    panel.querySelector("[data-update-current]")?.addEventListener("click", async () => {
      if (!selected?.current_show_scene_placement_id) { status("This cohort has no current Scene to update."); return; }
      try {
        await api(`/api/shows/${encodeURIComponent(showID)}/scenes/${encodeURIComponent(selected.current_show_scene_placement_id)}/update-current-scene`, { method: "POST" });
        status("Current Scene updated with the arranged placements.");
      } catch (err) { status("Update failed: " + err.message); }
    });

    panel.querySelector("[data-save-as-new]")?.addEventListener("click", async () => {
      if (!selected?.current_show_scene_placement_id) { status("This cohort has no current Scene to save."); return; }
      const title = window.prompt("New Scene title:");
      if (!title) return;
      const slug = title.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "") + "-" + Date.now().toString(36);
      try {
        await api(`/api/shows/${encodeURIComponent(showID)}/scenes/${encodeURIComponent(selected.current_show_scene_placement_id)}/save-as-new-scene`, {
          method: "POST", body: JSON.stringify({ title, slug }),
        });
        status("Saved as a new Scene: " + title);
      } catch (err) { status("Save failed: " + err.message); }
    });
  }

  // --- Game Status panel -------------------------------------------------

  function ensureStatusPanel() {
    if (state.statusPanel) return state.statusPanel;
    const el = document.createElement("div");
    el.id = "kernel85-status-panel";
    el.style.cssText = PANEL_BASE_STYLE + "top: 60px; left: 12px; width: 380px; max-height: 78vh; overflow-y: auto; padding: 12px; display: none;";
    document.body.appendChild(el);
    state.statusPanel = el;
    return el;
  }

  async function openGameStatusPanel() {
    const b = bridge();
    const showID = b?.getShowID?.();
    if (!showID) return;
    const panel = ensureStatusPanel();
    panel.style.display = "block";
    panel.innerHTML = `<div>Loading Game Status&hellip;</div>`;
    await renderGameStatusPanel(showID);
  }

  async function renderGameStatusPanel(showID) {
    const panel = state.statusPanel;
    let roster;
    try {
      roster = await api(`/api/shows/${encodeURIComponent(showID)}/cohorts`);
    } catch (err) {
      panel.innerHTML = `<div style="color:#e88;">Failed to load cohorts: ${escapeHtml(err.message)}</div>`;
      return;
    }
    const cohorts = roster.cohorts || [];
    if (!state.selectedCohortId && cohorts.length) state.selectedCohortId = cohorts[0].id;
    const cohortValue = state.selectedCohortId || "ungrouped";

    let blocks = [];
    try {
      const resp = await api(`/api/shows/${encodeURIComponent(showID)}/cohorts/${encodeURIComponent(cohortValue)}/game-status`);
      blocks = resp?.characters || [];
    } catch (err) {
      panel.innerHTML = `<div></div><div style="color:#e88;">Failed to load Game Status: ${escapeHtml(err.message)}</div>`;
      panel.prepend(closeButton(() => { panel.style.display = "none"; }));
      return;
    }

    const cohortOptions = cohorts.map((c) =>
      `<option value="${c.id}" ${c.id === cohortValue ? "selected" : ""}>${escapeHtml(c.name)}</option>`
    ).join("");

    const characterBlocks = blocks.map((block) => `
      <div data-character="${block.character_card_id}" style="border:1px solid #333844; border-radius:6px; padding:8px; margin-bottom:10px;">
        <strong>${escapeHtml(block.character_name || "Character")}</strong>
        <div style="display:grid; grid-template-columns: 1fr auto auto; gap:4px 8px; margin-top:6px; align-items:center;">
          ${POOL_LABELS.map(([key, label]) => {
            const p = (block.pools || []).find((x) => x.key === key) || { current: 0, max: 0 };
            return `
              <span>${label}</span>
              <input type="number" min="0" data-pool-current="${key}" value="${p.current}" style="width:48px; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px;">
              <input type="number" min="0" data-pool-max="${key}" value="${p.max}" style="width:48px; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px;">
            `;
          }).join("")}
        </div>
        <button type="button" data-save-pools style="${buttonStyle("margin-top:6px; font-size:11px;")}">Save HP</button>
        <div style="margin-top:8px;">
          <div data-active-statuses>
            ${(block.active_statuses || []).map((s) => `
              <span data-status-chip="${s.status_key}" style="display:inline-block; background:#3a2f1f; border:1px solid #6a5230; border-radius:10px; padding:2px 8px; margin:2px; font-size:11px;">
                ${escapeHtml(s.label)}${s.intensity != null ? " " + s.intensity : ""}
                <button type="button" data-clear-status="${s.status_key}" style="background:none;border:none;color:#e8e8ec;cursor:pointer;">&times;</button>
              </span>
            `).join("") || `<span style="opacity:0.6;">No active statuses.</span>`}
          </div>
          <div style="margin-top:6px; display:flex; gap:6px;">
            <select data-apply-status style="background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px; font-size:11px;"></select>
            <button type="button" data-apply-status-btn style="${buttonStyle("padding:3px 8px; font-size:11px;")}">Apply</button>
          </div>
        </div>
      </div>
    `).join("") || `<div style="opacity:0.6;">No Characters in this selection yet.</div>`;

    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 8px 0; font-size:14px;">Game Status</h3>
      <label style="display:block; margin-bottom:10px;">
        Cohort:
        <select data-select-cohort style="margin-left:6px; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px;">
          ${cohortOptions}
          <option value="ungrouped" ${cohortValue === "ungrouped" ? "selected" : ""}>Ungrouped</option>
        </select>
      </label>
      ${characterBlocks}
      <div data-status style="margin-top:6px; font-size:11px; opacity:0.7;"></div>
    `;
    panel.prepend(closeButton(() => { panel.style.display = "none"; }));
    makeDraggable(panel, panel);

    const status = (msg) => { const s = panel.querySelector("[data-status]"); if (s) s.textContent = msg || ""; };

    panel.querySelector("[data-select-cohort]")?.addEventListener("change", (event) => {
      state.selectedCohortId = event.target.value;
      renderGameStatusPanel(showID);
    });

    let statusRegistry = [];
    try { statusRegistry = (await api("/api/socio/statuses")) || []; } catch (_e) { /* best-effort */ }

    panel.querySelectorAll("[data-character]").forEach((card) => {
      const characterCardID = card.dataset.character;
      const select = card.querySelector("[data-apply-status]");
      if (select) {
        select.innerHTML = statusRegistry.map((s) => `<option value="${s.key}">${escapeHtml(s.label)}</option>`).join("");
      }
      card.querySelector("[data-save-pools]")?.addEventListener("click", async () => {
        try {
          for (const [key] of POOL_LABELS) {
            const current = Number(card.querySelector(`[data-pool-current="${key}"]`)?.value || 0);
            const max = Number(card.querySelector(`[data-pool-max="${key}"]`)?.value || 0);
            await api(`/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterCardID)}/socio/pools/${key}`, {
              method: "POST", body: JSON.stringify({ current, max }),
            });
          }
          status("HP saved.");
        } catch (err) { status("Save failed: " + err.message); }
      });
      card.querySelector("[data-apply-status-btn]")?.addEventListener("click", async () => {
        const statusKey = select?.value;
        if (!statusKey) return;
        try {
          await api(`/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterCardID)}/socio/statuses`, {
            method: "POST", body: JSON.stringify({ status_key: statusKey }),
          });
          await renderGameStatusPanel(showID);
        } catch (err) { status("Apply failed: " + err.message); }
      });
      card.querySelectorAll("[data-clear-status]").forEach((btn) => {
        btn.addEventListener("click", async () => {
          try {
            await api(`/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterCardID)}/socio/statuses/${encodeURIComponent(btn.dataset.clearStatus)}`, { method: "DELETE" });
            await renderGameStatusPanel(showID);
          } catch (err) { status("Clear failed: " + err.message); }
        });
      });
    });
  }

  // --- Toolbar visibility polling ---------------------------------------

  function pollToolbarVisibility() {
    const b = bridge();
    const showID = b?.getShowID?.() || "";
    const canManage = Boolean(b?.canManageStage?.());
    const toolbar = ensureToolbar();
    toolbar.style.display = showID && canManage ? "flex" : "none";
    if (showID !== state.lastShowID) {
      // A different Show came into view -- drop any stale cohort selection
      // and close open panels rather than showing another Show's data.
      state.lastShowID = showID;
      state.selectedCohortId = "";
      [state.cohortPanel, state.scenePanel, state.statusPanel].forEach((p) => { if (p) p.style.display = "none"; });
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

  window.VictoryKernel85Tools = {
    openSceneConfiguration,
    openCohortPanel,
    openGameStatusPanel,
  };
})();
