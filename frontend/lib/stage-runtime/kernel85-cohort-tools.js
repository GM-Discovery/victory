// Kernel 85: Show Cohorts, Scene Configuration, and Game Status -- a
// self-contained Director+ tool layer, deliberately separate from the
// generic stage engine (runtime.js et al never import anything from this
// file; it reads the engine only through the narrow read-only
// window.VictoryStageKernel85Bridge runtime.js exposes). All server
// authority is re-checked on every request by the backend
// (backend/internal/cohorts, backend/internal/socio, backend/internal/
// scenes' capture actions) -- everything here is presentation only.
//
// Kernel 88 extends this same file (rather than forking a parallel Director
// tool file) with Fate/Stance/blank-flag controls on the existing Game
// Status card, plus two new panels -- Current Turn and the Help/Interrupt
// stack -- reusing this file's api()/escapeHtml()/buttonStyle()/
// makeDraggable() helpers instead of re-implementing them.
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
    turnPanel: null,
    interruptPanel: null,
    lastShowID: "",
    statusPollTimer: null,
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
      <button type="button" data-k88="current-turn" style="${buttonStyle()}">Current Turn</button>
      <button type="button" data-k88="help-stack" style="${buttonStyle()}">Help Stack</button>
    `;
    el.querySelector('[data-k85="cohorts"]').addEventListener("click", () => openCohortPanel());
    el.querySelector('[data-k85="game-status"]').addEventListener("click", () => openGameStatusPanel());
    el.querySelector('[data-k88="current-turn"]').addEventListener("click", () => openCurrentTurnPanel());
    el.querySelector('[data-k88="help-stack"]').addEventListener("click", () => openInterruptPanel());
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

  // Anything a mouse press is supposed to *do something else* with. A
  // mousedown handler that calls preventDefault() over one of these
  // suppresses the browser's own behaviour -- notably it stops a <select>
  // from opening its popup at all and stops an <input> taking focus from a
  // click. Dragging must therefore give way wherever a real control lives.
  const INTERACTIVE_SELECTOR =
    "button, select, input, textarea, option, optgroup, label, a, [contenteditable=''], [contenteditable='true']";

  function makeDraggable(panel, handle) {
    // Panels are created once and re-rendered many times via innerHTML,
    // which leaves the panel element's own listeners intact -- so bind the
    // drag exactly once instead of stacking a fresh pair of window
    // listeners onto every render.
    if (panel.dataset.k85Draggable === "1") return;
    panel.dataset.k85Draggable = "1";

    let dragging = false;
    let startX = 0, startY = 0, startLeft = 0, startTop = 0;
    handle.style.cursor = "move";
    handle.addEventListener("mousedown", (event) => {
      if (event.target.closest(INTERACTIVE_SELECTOR)) return;
      const rect = panel.getBoundingClientRect();
      dragging = true;
      startX = event.clientX; startY = event.clientY;
      startLeft = rect.left; startTop = rect.top;
      // Panels centred with translateX(-50%) would jump on the first drag,
      // because the measured rect already includes the transform. Bake the
      // measured position into left/top and drop the transform.
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

  // The selected cohort is held across re-renders and across panels, so it
  // can outlive the cohort it names (archiving the selected cohort used to
  // leave a dangling id: the <select> fell back to displaying the first
  // option while every fetch still asked for the archived cohort). Resolve
  // it against the live list on every render.
  function resolveCohortSelection(cohorts) {
    const known = new Set(cohorts.map((c) => c.id));
    if (state.selectedCohortId && state.selectedCohortId !== "ungrouped" && !known.has(state.selectedCohortId)) {
      state.selectedCohortId = "";
    }
    if (!state.selectedCohortId && cohorts.length) {
      state.selectedCohortId = cohorts[0].id;
    }
    return state.selectedCohortId || "ungrouped";
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

    resolveCohortSelection(cohorts);
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
    let show;
    try {
      [roster, scenesResp, show] = await Promise.all([
        api(`/api/shows/${encodeURIComponent(showID)}/cohorts`),
        api(`/api/shows/${encodeURIComponent(showID)}/scenes`),
        api(`/api/shows/${encodeURIComponent(showID)}`),
      ]);
    } catch (err) {
      panel.innerHTML = `<div style="color:#e88;">Failed to load: ${escapeHtml(err.message)}</div>`;
      return;
    }
    const cohorts = roster.cohorts || [];
    const placements = scenesResp?.placements || scenesResp || [];
    const cohortValue = resolveCohortSelection(cohorts);
    const selected = cohorts.find((c) => c.id === cohortValue) || null;
    // Scenes are cohort-agnostic: with no real cohort selected ("Ungrouped"),
    // fall back to the Show's own current_show_scene_placement_id (a plain
    // shows table column, not tied to any cohort -- see
    // backend/internal/shows/stage.go's SetCurrentScenePlacement) instead of
    // disabling scene management entirely. A real cohort's own placement
    // still takes priority when one is selected, so per-cohort overrides
    // keep working exactly as before.
    const effectivePlacementId = selected
      ? selected.current_show_scene_placement_id
      : show?.current_show_scene_placement_id;

    const cohortOptions = cohorts.map((c) =>
      `<option value="${c.id}" ${c.id === cohortValue ? "selected" : ""}>${escapeHtml(c.name)}</option>`
    ).join("");

    const bridgeRef = bridge();
    const configuratorActive = Boolean(bridgeRef?.isConfiguratorActive?.());
    const configuratorSceneID = configuratorActive ? String(bridgeRef?.getConfiguratorSceneID?.() || "") : "";

    const sceneRows = (Array.isArray(placements) ? placements : []).map((p) => {
      const title = p.scene?.title || p.title || "Untitled Scene";
      const placementID = p.placement?.id || p.id;
      const sceneID = p.scene?.id || p.scene_id || "";
      const isCurrent = effectivePlacementId === placementID;
      const isBeingBuilt = configuratorActive && sceneID === configuratorSceneID;
      return `
        <div style="display:flex; justify-content:space-between; align-items:center; padding:4px 0; border-bottom:1px solid #2a2e37;">
          <span>${escapeHtml(title)}${isCurrent ? " <em>(current)</em>" : ""}${isBeingBuilt ? " <em>(building)</em>" : ""}</span>
          <span style="display:flex; gap:6px;">
            <button type="button" data-build="${sceneID}" style="${buttonStyle("padding:3px 8px; font-size:11px;")}" ${configuratorActive ? "disabled" : ""}>Build</button>
            <button type="button" data-activate="${placementID}" style="${buttonStyle("padding:3px 8px; font-size:11px;")}" ${isCurrent ? "disabled" : ""}>Activate</button>
          </span>
        </div>
      `;
    }).join("") || `<div style="opacity:0.6;">No Scenes attached to this Show yet.</div>`;

    const configuratorBanner = configuratorActive
      ? `<div style="background:#3a2f10; border:1px solid #8a6d1f; border-radius:4px; padding:6px 8px; margin-bottom:10px; font-size:12px;">
          <div>Configurator Mode: editing a draft. Audience and other viewers do not see this. Use Activate above to publish it.</div>
          <div style="display:flex; gap:6px; margin-top:6px; flex-wrap:wrap;">
            <button type="button" data-place-hotspot style="${buttonStyle("padding:2px 8px; font-size:11px;")}">Place Hotspot</button>
            <button type="button" data-bind-interaction style="${buttonStyle("padding:2px 8px; font-size:11px;")}">Bind Selected to Interaction</button>
            <button type="button" data-exit-configurator style="${buttonStyle("padding:2px 8px; font-size:11px;")}">Exit Configurator</button>
          </div>
        </div>`
      : "";

    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 10px 0; font-size:15px;">Scene Configuration</h3>
      ${configuratorBanner}
      <label style="display:block; margin-bottom:10px;">
        Cohort:
        <select data-select-cohort style="margin-left:6px; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px;" ${configuratorActive ? "disabled" : ""}>
          ${cohortOptions}
          <option value="ungrouped" ${cohortValue === "ungrouped" ? "selected" : ""}>Ungrouped (Show default)</option>
        </select>
      </label>
      <div style="margin-bottom:10px;">${sceneRows}</div>
      <div style="display:flex; gap:8px; margin-top:10px;">
        <button type="button" data-new-scene style="${buttonStyle()}" ${configuratorActive ? "disabled" : ""}>New Scene</button>
        <button type="button" data-update-current style="${buttonStyle()}" ${effectivePlacementId && !configuratorActive ? "" : "disabled"}>Update Current Scene</button>
        <button type="button" data-save-as-new style="${buttonStyle()}" ${effectivePlacementId && !configuratorActive ? "" : "disabled"}>Save as New Scene</button>
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

    panel.querySelector("[data-exit-configurator]")?.addEventListener("click", () => {
      bridge()?.exitConfiguratorMode?.();
      status("Configurator Mode closed.");
      void renderSceneConfiguration(showID);
    });

    panel.querySelector("[data-place-hotspot]")?.addEventListener("click", () => {
      bridge()?.placeConfiguratorHotspot?.();
    });

    panel.querySelector("[data-bind-interaction]")?.addEventListener("click", () => {
      bridge()?.bindConfiguratorSelectionToInteraction?.();
    });

    panel.querySelectorAll("[data-build]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        const sceneID = btn.dataset.build;
        // Building needs a real placement to read/write a draft composition
        // through (scene_stage_elements is placement-scoped for the Show
        // layer, base layer otherwise) -- reuse whichever placement this
        // row already has rather than inventing a second lookup.
        const row = (Array.isArray(placements) ? placements : []).find((p) => (p.scene?.id || p.scene_id) === sceneID);
        const placementID = row?.placement?.id || row?.id;
        if (!sceneID || !placementID) {
          status("Couldn't find this Scene's placement to build against.");
          return;
        }
        try {
          await bridge()?.enterConfiguratorMode?.(showID, sceneID, placementID);
          status("Building this Scene -- Audience does not see these changes until you Activate it.");
          await renderSceneConfiguration(showID);
        } catch (err) {
          status("Couldn't enter Configurator Mode: " + err.message);
        }
      });
    });

    panel.querySelector("[data-new-scene]")?.addEventListener("click", async () => {
      const title = window.prompt("New Scene title:");
      if (!title) return;
      const slug = title.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "") + "-" + Date.now().toString(36);
      const locationID = show?.location_id;
      if (!locationID) {
        status("Couldn't resolve this Show's location to create a Scene under.");
        return;
      }
      try {
        // A brand-new Scene starts with an empty scene_stage_elements set --
        // Configurator Mode opens on a blank canvas, exactly like Building
        // an existing Scene, just with nothing captured yet.
        const created = await api("/api/scenes", {
          method: "POST", body: JSON.stringify({ location_id: locationID, slug, title }),
        });
        const newSceneID = created?.scene?.id;
        if (!newSceneID) { status("Scene created, but couldn't read its id."); return; }
        const placement = await api(`/api/shows/${encodeURIComponent(showID)}/scenes`, {
          method: "POST", body: JSON.stringify({ scene_id: newSceneID }),
        });
        const newPlacementID = placement?.placement?.id;
        if (!newPlacementID) { status("Scene created, but couldn't attach it to this Show."); return; }
        await bridge()?.enterConfiguratorMode?.(showID, newSceneID, newPlacementID);
        status(`Building "${title}" -- Audience does not see these changes until you Activate it.`);
        await renderSceneConfiguration(showID);
      } catch (err) {
        status("Couldn't create a new Scene: " + err.message);
      }
    });

    panel.querySelectorAll("[data-activate]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        try {
          // Ungrouped activates the Show's own default current scene
          // (POST /shows/{id}/current-scene); a real cohort activates its
          // own per-cohort override instead. Same placement concept either
          // way, just two different "whose current scene is this" scopes.
          if (selected) {
            await api(`/api/shows/${encodeURIComponent(showID)}/cohorts/${encodeURIComponent(selected.id)}/current-scene`, {
              method: "POST", body: JSON.stringify({ show_scene_placement_id: btn.dataset.activate }),
            });
            status("Scene activated for " + selected.name + ".");
          } else {
            await api(`/api/shows/${encodeURIComponent(showID)}/current-scene`, {
              method: "POST", body: JSON.stringify({ show_scene_placement_id: btn.dataset.activate }),
            });
            status("Scene activated as the Show default.");
          }
          await renderSceneConfiguration(showID);
        } catch (err) { status("Activate failed: " + err.message); }
      });
    });

    panel.querySelector("[data-update-current]")?.addEventListener("click", async () => {
      if (!effectivePlacementId) { status("No current Scene to update yet -- activate one first."); return; }
      try {
        await api(`/api/shows/${encodeURIComponent(showID)}/scenes/${encodeURIComponent(effectivePlacementId)}/update-current-scene`, { method: "POST" });
        status("Current Scene updated with the arranged placements.");
      } catch (err) { status("Update failed: " + err.message); }
    });

    panel.querySelector("[data-save-as-new]")?.addEventListener("click", async () => {
      if (!effectivePlacementId) { status("No current Scene to save yet -- activate one first."); return; }
      const title = window.prompt("New Scene title:");
      if (!title) return;
      const slug = title.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "") + "-" + Date.now().toString(36);
      try {
        // save-as-new-scene only creates a reusable Scene record -- it does
        // NOT attach it to this Show, so on its own it would never appear
        // in the list above or be reachable by Activate. Attach it as a
        // placement (POST /shows/{id}/scenes) and immediately activate that
        // placement too, since "save configuration" should mean "this is
        // now loadable," not "this now exists somewhere unreachable."
        const saved = await api(`/api/shows/${encodeURIComponent(showID)}/scenes/${encodeURIComponent(effectivePlacementId)}/save-as-new-scene`, {
          method: "POST", body: JSON.stringify({ title, slug }),
        });
        const newSceneId = saved?.scene?.id;
        if (!newSceneId) { status("Saved, but couldn't read the new Scene's id to attach it."); return; }
        const placement = await api(`/api/shows/${encodeURIComponent(showID)}/scenes`, {
          method: "POST", body: JSON.stringify({ scene_id: newSceneId }),
        });
        const newPlacementId = placement?.placement?.id;
        if (newPlacementId) {
          if (selected) {
            await api(`/api/shows/${encodeURIComponent(showID)}/cohorts/${encodeURIComponent(selected.id)}/current-scene`, {
              method: "POST", body: JSON.stringify({ show_scene_placement_id: newPlacementId }),
            });
          } else {
            await api(`/api/shows/${encodeURIComponent(showID)}/current-scene`, {
              method: "POST", body: JSON.stringify({ show_scene_placement_id: newPlacementId }),
            });
          }
        }
        status("Saved as a new Scene and activated: " + title);
        await renderSceneConfiguration(showID);
      } catch (err) { status("Save failed: " + err.message); }
    });
  }

  // --- Game Status panel -------------------------------------------------

  // Fate and Stance are shared state: a Player spending Fate from their HUD
  // changes the same row this card displays. Fetching once at render time
  // left the Director looking at a snapshot that silently drifted out of
  // agreement with the Player's HUD, so these two fields are re-read on a
  // poll. Only non-editable text is overwritten unconditionally -- a control
  // the Director is actively using is left alone rather than yanked out from
  // under them, and the HP fields are never touched by the poll at all
  // because the Director types into those.
  async function refreshCharacterLiveFields(showID, card) {
    const characterCardID = card.dataset.character;
    if (!characterCardID) return;
    let projection;
    try {
      projection = (await api(
        `/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterCardID)}/socio/view`
      )).projection;
    } catch (_e) {
      return; // best-effort; the panel keeps whatever it last showed
    }
    if (!card.isConnected) return;

    const balanceEl = card.querySelector("[data-fate-balance]");
    if (balanceEl) balanceEl.textContent = String(projection?.fate_balance ?? "?");

    const stanceSelect = card.querySelector("[data-set-stance]");
    if (stanceSelect && document.activeElement !== stanceSelect) {
      stanceSelect.value = projection?.stance_key || "";
    }
  }

  function stopStatusPolling() {
    if (state.statusPollTimer) {
      window.clearInterval(state.statusPollTimer);
      state.statusPollTimer = null;
    }
  }

  function startStatusPolling(showID) {
    stopStatusPolling();
    state.statusPollTimer = window.setInterval(() => {
      const panel = state.statusPanel;
      if (!panel || panel.style.display === "none" || !panel.isConnected) {
        stopStatusPolling();
        return;
      }
      panel.querySelectorAll("[data-character]").forEach((card) => {
        refreshCharacterLiveFields(showID, card);
      });
    }, 3000);
  }

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
    const cohortValue = resolveCohortSelection(cohorts);

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

        <div style="margin-top:10px; padding-top:8px; border-top:1px solid #2a2e37;">
          <div style="display:flex; align-items:center; gap:8px;">
            <span style="opacity:0.7;">Fate:</span>
            <strong data-fate-balance>&hellip;</strong>
            <input type="number" data-fate-delta value="1" style="width:44px; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px;">
            <button type="button" data-award-fate style="${buttonStyle("padding:3px 8px; font-size:11px;")}">Award</button>
          </div>
          <div style="display:flex; align-items:center; gap:8px; margin-top:6px;">
            <span style="opacity:0.7;">Stance:</span>
            <select data-set-stance style="background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px; font-size:11px;"></select>
          </div>
        </div>

        <div style="margin-top:8px;">
          <div data-active-statuses>
            ${(block.active_statuses || []).map((s) => (s.is_blank ? `
              <span data-flag-chip="${s.id}" style="display:inline-block; background:#26313a; border:1px solid #3d5568; border-radius:10px; padding:2px 8px; margin:2px; font-size:11px;">
                ${escapeHtml(s.label)}
                <button type="button" data-clear-flag="${s.id}" style="background:none;border:none;color:#e8e8ec;cursor:pointer;">&times;</button>
              </span>
            ` : `
              <span data-status-chip="${s.status_key}" style="display:inline-block; background:#3a2f1f; border:1px solid #6a5230; border-radius:10px; padding:2px 8px; margin:2px; font-size:11px;">
                ${escapeHtml(s.label)}${s.intensity != null ? " " + s.intensity : ""}
                <button type="button" data-clear-status="${s.status_key}" style="background:none;border:none;color:#e8e8ec;cursor:pointer;">&times;</button>
              </span>
            `)).join("") || `<span style="opacity:0.6;">No active statuses.</span>`}
          </div>
          <div style="margin-top:6px; display:flex; gap:6px;">
            <select data-apply-status style="background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px; font-size:11px;"></select>
            <button type="button" data-apply-status-btn style="${buttonStyle("padding:3px 8px; font-size:11px;")}">Apply</button>
          </div>
          <div style="margin-top:6px; display:flex; gap:6px;">
            <input type="text" data-flag-label placeholder="Blank state (e.g. Holding the gate)" maxlength="60" style="flex:1; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px; font-size:11px; padding:2px 6px;">
            <button type="button" data-add-flag-btn style="${buttonStyle("padding:3px 8px; font-size:11px;")}">Add</button>
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
    panel.prepend(closeButton(() => { panel.style.display = "none"; stopStatusPolling(); }));
    makeDraggable(panel, panel);
    startStatusPolling(showID);

    const status = (msg) => { const s = panel.querySelector("[data-status]"); if (s) s.textContent = msg || ""; };

    panel.querySelector("[data-select-cohort]")?.addEventListener("change", (event) => {
      state.selectedCohortId = event.target.value;
      renderGameStatusPanel(showID);
    });

    let statusRegistry = [];
    try { statusRegistry = (await api("/api/socio/statuses")).statuses || []; } catch (_e) { /* best-effort */ }
    let stanceRegistry = [];
    try { stanceRegistry = (await api("/api/socio/stances")).stances || []; } catch (_e) { /* best-effort */ }

    panel.querySelectorAll("[data-character]").forEach((card) => {
      const characterCardID = card.dataset.character;
      const select = card.querySelector("[data-apply-status]");
      if (select) {
        select.innerHTML = statusRegistry.map((s) => `<option value="${s.key}">${escapeHtml(s.label)}</option>`).join("");
      }

      const stanceSelect = card.querySelector("[data-set-stance]");
      if (stanceSelect) {
        stanceSelect.innerHTML = `<option value="">&hellip;</option>` +
          stanceRegistry.map((s) => `<option value="${s.key}">${escapeHtml(s.label)}</option>`).join("");
        stanceSelect.addEventListener("change", async () => {
          const key = stanceSelect.value;
          if (!key) return;
          try {
            await api(`/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterCardID)}/socio/stance`, {
              method: "POST", body: JSON.stringify({ stance_key: key }),
            });
            status("Stance set.");
          } catch (err) { status("Stance failed: " + err.message); }
        });
      }

      // Initial values come from the same refresh the poll below reuses, so
      // there is exactly one code path that decides what these fields say.
      refreshCharacterLiveFields(showID, card);

      card.querySelector("[data-award-fate]")?.addEventListener("click", async () => {
        const delta = Number(card.querySelector("[data-fate-delta]")?.value || 0);
        if (!delta) return;
        try {
          const res = await api(`/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterCardID)}/socio/fate/award`, {
            method: "POST", body: JSON.stringify({ delta, reason: "director_award" }),
          });
          const balanceEl = card.querySelector("[data-fate-balance]");
          if (balanceEl) balanceEl.textContent = String(res.fate?.balance ?? "?");
          status("Fate awarded.");
        } catch (err) { status("Fate award failed: " + err.message); }
      });

      card.querySelector("[data-add-flag-btn]")?.addEventListener("click", async () => {
        const input = card.querySelector("[data-flag-label]");
        const label = (input?.value || "").trim();
        if (!label) return;
        try {
          await api(`/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterCardID)}/socio/flags`, {
            method: "POST", body: JSON.stringify({ label }),
          });
          await renderGameStatusPanel(showID);
        } catch (err) { status("Add flag failed: " + err.message); }
      });
      card.querySelectorAll("[data-clear-flag]").forEach((btn) => {
        btn.addEventListener("click", async () => {
          try {
            await api(`/api/shows/${encodeURIComponent(showID)}/characters/${encodeURIComponent(characterCardID)}/socio/flags/${encodeURIComponent(btn.dataset.clearFlag)}`, { method: "DELETE" });
            await renderGameStatusPanel(showID);
          } catch (err) { status("Clear flag failed: " + err.message); }
        });
      });

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

  // --- Kernel 88: Current Turn panel --------------------------------------

  function ensureTurnPanel() {
    if (state.turnPanel) return state.turnPanel;
    const el = document.createElement("div");
    el.id = "kernel88-turn-panel";
    el.style.cssText = PANEL_BASE_STYLE + "top: 60px; right: 12px; width: 340px; max-height: 70vh; overflow-y: auto; padding: 12px; display: none;";
    document.body.appendChild(el);
    state.turnPanel = el;
    return el;
  }

  async function openCurrentTurnPanel() {
    const b = bridge();
    const showID = b?.getShowID?.();
    if (!showID) return;
    const panel = ensureTurnPanel();
    panel.style.display = "block";
    panel.innerHTML = `<div>Loading Current Turn&hellip;</div>`;
    await renderTurnPanel(showID);
  }

  async function renderTurnPanel(showID) {
    const panel = state.turnPanel;
    let roster;
    try {
      roster = await api(`/api/shows/${encodeURIComponent(showID)}/cohorts`);
    } catch (err) {
      panel.innerHTML = `<div style="color:#e88;">Failed to load cohorts: ${escapeHtml(err.message)}</div>`;
      return;
    }
    const cohorts = roster.cohorts || [];
    const cohortValue = resolveCohortSelection(cohorts);
    const participants = cohortValue === "ungrouped" ? (roster.ungrouped || []) : (cohorts.find((c) => c.id === cohortValue)?.members || []);

    let coordination = {};
    try {
      coordination = (await api(`/api/shows/${encodeURIComponent(showID)}/cohorts/${encodeURIComponent(cohortValue)}/socio/current-turn`))?.coordination || {};
    } catch (err) {
      panel.innerHTML = `<div></div><div style="color:#e88;">Failed to load Current Turn: ${escapeHtml(err.message)}</div>`;
      panel.prepend(closeButton(() => { panel.style.display = "none"; }));
      return;
    }

    const cohortOptions = cohorts.map((c) => `<option value="${c.id}" ${c.id === cohortValue ? "selected" : ""}>${escapeHtml(c.name)}</option>`).join("");

    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 8px 0; font-size:14px;">Current Turn</h3>
      <label style="display:block; margin-bottom:10px;">
        Cohort:
        <select data-select-cohort style="margin-left:6px; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px;">
          ${cohortOptions}
          <option value="ungrouped" ${cohortValue === "ungrouped" ? "selected" : ""}>Ungrouped</option>
        </select>
      </label>
      <div style="margin-bottom:10px;">
        Currently: <strong>${coordination.current_turn_character_name ? escapeHtml(coordination.current_turn_character_name) : (coordination.current_turn_user_id ? "someone with no Character selected" : "unset")}</strong>
      </div>
      <div>
        ${participants.map((p) => `
          <div style="display:flex; justify-content:space-between; align-items:center; padding:3px 0;">
            <span>${escapeHtml(p.display_name || "Participant")}${p.character_name ? " &middot; " + escapeHtml(p.character_name) : ""}</span>
            <button type="button" data-give-turn="${p.user_id}" style="${buttonStyle("padding:2px 8px; font-size:11px;")}" ${coordination.current_turn_user_id === p.user_id ? "disabled" : ""}>Give Turn</button>
          </div>
        `).join("") || `<div style="opacity:0.6;">No participants in this selection.</div>`}
      </div>
      <div data-status style="margin-top:10px; font-size:11px; opacity:0.7;"></div>
    `;
    panel.prepend(closeButton(() => { panel.style.display = "none"; }));
    makeDraggable(panel, panel);

    const status = (msg) => { const s = panel.querySelector("[data-status]"); if (s) s.textContent = msg || ""; };

    panel.querySelector("[data-select-cohort]")?.addEventListener("change", (event) => {
      state.selectedCohortId = event.target.value;
      renderTurnPanel(showID);
    });
    panel.querySelectorAll("[data-give-turn]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        try {
          await api(`/api/shows/${encodeURIComponent(showID)}/cohorts/${encodeURIComponent(cohortValue)}/socio/current-turn`, {
            method: "POST", body: JSON.stringify({ target_user_id: btn.dataset.giveTurn }),
          });
          await renderTurnPanel(showID);
        } catch (err) { status("Give Turn failed: " + err.message); }
      });
    });
  }

  // --- Kernel 88: Help / Interrupt stack panel ----------------------------

  function ensureInterruptPanel() {
    if (state.interruptPanel) return state.interruptPanel;
    const el = document.createElement("div");
    el.id = "kernel88-interrupt-panel";
    el.style.cssText = PANEL_BASE_STYLE + "top: 60px; left: 12px; width: 400px; max-height: 78vh; overflow-y: auto; padding: 12px; display: none;";
    document.body.appendChild(el);
    state.interruptPanel = el;
    return el;
  }

  async function openInterruptPanel() {
    const b = bridge();
    const showID = b?.getShowID?.();
    if (!showID) return;
    const panel = ensureInterruptPanel();
    panel.style.display = "block";
    panel.innerHTML = `<div>Loading Help/Interrupt stack&hellip;</div>`;
    await renderInterruptPanel(showID);
  }

  // Nesting depth is rendered as indentation -- a small stack, not a
  // workflow diagram (kernel-88 spec §13.3: "keep it visually restrained").
  function pendingActionDepth(actions, action) {
    let depth = 0;
    let current = action;
    while (current.parent_id) {
      depth += 1;
      current = actions.find((a) => a.id === current.parent_id);
      if (!current) break;
    }
    return depth;
  }

  async function renderInterruptPanel(showID) {
    const panel = state.interruptPanel;
    let actions = [];
    try {
      actions = (await api(`/api/shows/${encodeURIComponent(showID)}/socio/pending-actions`))?.pending_actions || [];
    } catch (err) {
      panel.innerHTML = `<div></div><div style="color:#e88;">Failed to load stack: ${escapeHtml(err.message)}</div>`;
      panel.prepend(closeButton(() => { panel.style.display = "none"; }));
      return;
    }

    // Kernel 89 §7: a target complexity the Director prepared and just
    // recalled prefills this field. It is a prefill and nothing more --
    // the Director still names the helper and presses Interrupt, and the
    // server still adjudicates. Recall never rolls and never acts.
    const recalledTC = window.VictoryKernel89DirectorTools?.recalledTargetComplexity?.() ?? null;
    const recalledLabel = window.VictoryKernel89DirectorTools?.recalledLabel?.() || "";

    const rows = actions.map((a) => {
      const depth = pendingActionDepth(actions, a);
      const isInterrupt = a.kind === "interrupt";
      return `
        <div data-action="${a.id}" style="margin-left:${depth * 16}px; border:1px solid #333844; border-radius:6px; padding:6px 8px; margin-top:6px;">
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <span>${isInterrupt ? "&#8618; " : ""}<strong>${escapeHtml(a.title || (isInterrupt ? "Help" : "Action"))}</strong>${isInterrupt ? ` (TC ${a.target_complexity ?? "?"})` : ""}</span>
            <button type="button" data-cancel-action="${a.id}" style="${buttonStyle("padding:2px 6px; font-size:11px;")}">Cancel</button>
          </div>
          <div style="margin-top:4px; display:flex; gap:6px;">
            <input type="text" data-helper-card placeholder="Helper Character ID" style="flex:1; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px; font-size:11px; padding:2px 6px;">
            <input type="number" data-target-complexity placeholder="TC" value="${recalledTC == null ? "" : recalledTC}" title="${recalledTC == null ? "" : "Recalled: " + escapeHtml(recalledLabel)}" style="width:50px; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px; font-size:11px;">
            <button type="button" data-open-interrupt="${a.id}" style="${buttonStyle("padding:2px 6px; font-size:11px;")}">Interrupt</button>
          </div>
          ${isInterrupt ? `
            <div style="margin-top:4px; display:flex; gap:6px;">
              <input type="number" data-roll-total placeholder="Roll total" style="width:70px; background:#20242c; color:#e8e8ec; border:1px solid #454b59; border-radius:4px; font-size:11px;">
              <button type="button" data-resolve-interrupt="${a.id}" style="${buttonStyle("padding:2px 6px; font-size:11px;")}">Resolve</button>
            </div>
          ` : ""}
        </div>
      `;
    }).join("") || `<div style="opacity:0.6;">No pending actions -- the stack is empty.</div>`;

    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 8px 0; font-size:14px;">Help / Interrupt Stack</h3>
      <p style="margin:0 0 8px 0; font-size:11px; opacity:0.7;">
        Primary actions are opened by Players on their own Current Turn. Directors adjudicate Interrupts here.
      </p>
      ${rows}
      <div data-status style="margin-top:10px; font-size:11px; opacity:0.7;"></div>
    `;
    panel.prepend(closeButton(() => { panel.style.display = "none"; }));
    makeDraggable(panel, panel);

    const status = (msg) => { const s = panel.querySelector("[data-status]"); if (s) s.textContent = msg || ""; };

    panel.querySelectorAll("[data-open-interrupt]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        const row = btn.closest("[data-action]");
        const helperCardID = row.querySelector("[data-helper-card]")?.value.trim();
        const tc = Number(row.querySelector("[data-target-complexity]")?.value || 0);
        if (!helperCardID || !tc) { status("Helper Character ID and target complexity are required."); return; }
        try {
          await api(`/api/shows/${encodeURIComponent(showID)}/socio/pending-actions/${encodeURIComponent(btn.dataset.openInterrupt)}/interrupt`, {
            method: "POST", body: JSON.stringify({ helper_character_card_id: helperCardID, target_complexity: tc }),
          });
          await renderInterruptPanel(showID);
        } catch (err) { status("Open interrupt failed: " + err.message); }
      });
    });
    panel.querySelectorAll("[data-resolve-interrupt]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        const row = btn.closest("[data-action]");
        const rollTotal = Number(row.querySelector("[data-roll-total]")?.value || 0);
        try {
          await api(`/api/shows/${encodeURIComponent(showID)}/socio/pending-actions/${encodeURIComponent(btn.dataset.resolveInterrupt)}/resolve`, {
            method: "POST", body: JSON.stringify({ roll_total: rollTotal }),
          });
          await renderInterruptPanel(showID);
        } catch (err) { status("Resolve failed: " + err.message); }
      });
    });
    panel.querySelectorAll("[data-cancel-action]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        try {
          await api(`/api/shows/${encodeURIComponent(showID)}/socio/pending-actions/${encodeURIComponent(btn.dataset.cancelAction)}/cancel`, { method: "POST" });
          await renderInterruptPanel(showID);
        } catch (err) { status("Cancel failed: " + err.message); }
      });
    });
  }

  // --- Toolbar visibility polling ---------------------------------------

  function pollToolbarVisibility() {
    const b = bridge();
    const showID = b?.getShowID?.() || "";
    const canManage = Boolean(b?.canManageStage?.());
    const toolbar = ensureToolbar();
    // Kernel 89 took over the top-right Director surface and re-presents
    // these same panels as grouped tool families (§5: one tool button, not
    // a row of them). When that module is present this file keeps every
    // panel and drops only its own flat button row -- so removing
    // kernel89-director-tools.js from a page restores the Kernel 85/88
    // toolbar rather than leaving the Director with no tools at all.
    const kernel89OwnsToolbar = window.VictoryDirectorToolbarOwner === "kernel89";
    toolbar.style.display = showID && canManage && !kernel89OwnsToolbar ? "flex" : "none";
    if (showID !== state.lastShowID) {
      // A different Show came into view -- drop any stale cohort selection
      // and close open panels rather than showing another Show's data.
      state.lastShowID = showID;
      state.selectedCohortId = "";
      stopStatusPolling();
      [state.cohortPanel, state.scenePanel, state.statusPanel, state.turnPanel, state.interruptPanel].forEach((p) => { if (p) p.style.display = "none"; });
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
    openCurrentTurnPanel,
    openInterruptPanel,
  };
})();
