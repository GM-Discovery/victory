// Kernel 92: the Director's Showtime scheduler/launcher popup.
//
// Lives on the Director's Chair page (not the in-stage Kernel 89 toolbar)
// because its whole point is to work *before* any live Session/Show
// context exists -- the Kernel 89 toolbar only appears once
// window.VictoryStageKernel88Bridge already has a show_id from a live
// snapshot (backend/lib/stage-runtime/runtime.js), which is exactly the
// state Showtime is meant to create, not require.
//
// GUI and the /showtime command call the identical server-side domain
// operations (backend/internal/showtime.Start/Status/End/Preflight) --
// this file is only presentation over POST /api/showtime/control and
// GET/POST /api/showtime/showings.
(function () {
  "use strict";

  const state = {
    panel: null,
    selectedShowID: "",
    lastPreflight: null,
  };

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

  function isoToLocalInputValue(iso) {
    if (!iso) return "";
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return "";
    const pad = (n) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
  }

  function localInputValueToIso(value) {
    if (!value) return "";
    const d = new Date(value);
    if (Number.isNaN(d.getTime())) return "";
    return d.toISOString();
  }

  function formatLocal(iso) {
    if (!iso) return "—";
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return "—";
    return d.toLocaleString();
  }

  const BAND_LABELS = {
    lt_7d: "Less than 1 week ago",
    d7_30: "1 week – 1 month ago",
    d31_90: "1 – 3 months ago",
    d91_180: "3 – 6 months ago",
    d181_365: "6 – 12 months ago",
    d366_36mo: "1 – 3 years ago",
    older: "Older",
  };
  const BAND_ORDER = ["lt_7d", "d7_30", "d31_90", "d91_180", "d181_365", "d366_36mo", "older"];

  function ensurePanel() {
    if (state.panel && document.body.contains(state.panel)) return state.panel;
    const el = document.createElement("div");
    el.id = "kernel92-showtime-panel";
    el.style.cssText = `
      position: fixed; z-index: 9500; top: 60px; right: 12px; width: 420px; max-height: 82vh;
      overflow-y: auto; padding: 14px; display: none;
      background: #1b1e24; color: #e8e8ec; border: 1px solid #3a3f4b; border-radius: 8px;
      box-shadow: 0 8px 28px rgba(0,0,0,0.45);
      font: 13px/1.4 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    `;
    document.body.appendChild(el);
    state.panel = el;
    return el;
  }

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
  function statusSetter(panel) {
    return (msg, isError) => {
      const s = panel.querySelector("[data-status]");
      if (!s) return;
      s.textContent = msg || "";
      s.style.color = isError ? "#e88" : "#8bd18b";
    };
  }

  async function openPanel() {
    const panel = ensurePanel();
    panel.style.display = "block";
    await renderRoot();
  }

  function closePanel() {
    if (state.panel) state.panel.style.display = "none";
  }

  async function renderRoot(message) {
    const panel = state.panel;
    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 10px 0; font-size:15px;">Showtime</h3>
      <div data-message style="margin-bottom:10px; font-size:12px; white-space:pre-line;"></div>

      <details open style="margin-bottom:12px;">
        <summary style="cursor:pointer; font-weight:600;">New Showing</summary>
        <div style="margin-top:8px;">
          <label style="display:block; font-size:11px; margin-bottom:6px;">
            Show Run
            <select data-new-show-run style="${inputStyle("width:100%; margin-top:3px;")}"></select>
          </label>
          <label style="display:block; font-size:11px; margin-bottom:6px;">
            Date &amp; Time
            <input type="datetime-local" data-new-when style="${inputStyle("width:100%; margin-top:3px;")}" />
          </label>
          <label style="display:block; font-size:11px; margin-bottom:6px;">
            Short Nickname of Showing — 50 characters maximum
            <input type="text" data-new-nickname maxlength="50" style="${inputStyle("width:100%; margin-top:3px;")}" placeholder="e.g. Friday Night Special" />
          </label>
          <button type="button" data-schedule style="${buttonStyle("width:100%; border-color:#6a7cff;")}">Schedule Showing</button>
          <div data-schedule-status style="margin-top:6px; font-size:11px;"></div>
        </div>
      </details>

      <div data-lists></div>
      <div data-status style="margin-top:10px; font-size:11px;"></div>
    `;
    panel.prepend(closeButton(closePanel));
    const status = statusSetter(panel);
    if (message) panel.querySelector("[data-message]").textContent = message;

    await populateShowRunPicker(panel);
    await renderLists(panel);

    panel.querySelector("[data-schedule]").addEventListener("click", async () => {
      const scheduleStatus = (msg, isError) => {
        const s = panel.querySelector("[data-schedule-status]");
        s.textContent = msg || "";
        s.style.color = isError ? "#e88" : "#8bd18b";
      };
      const showRunID = panel.querySelector("[data-new-show-run]").value;
      const whenLocal = panel.querySelector("[data-new-when]").value;
      const nickname = panel.querySelector("[data-new-nickname]").value.trim();
      if (!showRunID) { scheduleStatus("Choose a Show Run.", true); return; }
      if (!whenLocal) { scheduleStatus("Choose a date and time.", true); return; }
      if (!nickname) { scheduleStatus("Nickname is required.", true); return; }
      if (nickname.length > 50) { scheduleStatus("Nickname must be 50 characters or fewer.", true); return; }
      scheduleStatus("Scheduling…");
      try {
        await api("/api/showtime/showings", {
          method: "POST",
          body: JSON.stringify({
            show_run_id: showRunID,
            nickname,
            scheduled_start_at: localInputValueToIso(whenLocal),
          }),
        });
        panel.querySelector("[data-new-nickname]").value = "";
        scheduleStatus("Scheduled.");
        await renderLists(panel);
      } catch (err) {
        scheduleStatus("Could not schedule: " + err.message, true);
      }
    });

    void status;
  }

  async function populateShowRunPicker(panel) {
    const select = panel.querySelector("[data-new-show-run]");
    try {
      const data = await api("/api/show-runs");
      const runs = (data.show_runs || []).filter((r) => r.can_manage);
      select.innerHTML = runs.map((r) => `<option value="${escapeHtml(r.id)}">${escapeHtml(r.title)}</option>`).join("")
        || `<option value="">No manageable Show Runs</option>`;
    } catch (_err) {
      select.innerHTML = `<option value="">Could not load Show Runs</option>`;
    }
  }

  async function renderLists(panel) {
    const container = panel.querySelector("[data-lists]");
    container.innerHTML = `<div style="opacity:0.7;">Loading Showings…</div>`;
    let data;
    try {
      data = await api("/api/showtime/showings");
    } catch (err) {
      container.innerHTML = `<div style="color:#e88;">Could not load Showings: ${escapeHtml(err.message)}</div>`;
      return;
    }

    const rowHtml = (item) => `
      <button type="button" data-showing="${escapeHtml(item.show_id)}"
        style="display:block; width:100%; text-align:left; background:transparent; border:none; border-bottom:1px solid #2a2e37; padding:8px 4px; cursor:pointer; color:inherit;">
        <span style="display:flex; justify-content:space-between;">
          <span>${escapeHtml(item.nickname || item.show_run_title)}${item.is_live ? " <span style=\"color:#8bd18b;\">● LIVE</span>" : ""}</span>
          <span style="opacity:0.6; font-size:11px;">${escapeHtml(item.short_code || "")}</span>
        </span>
        <span style="display:block; font-size:11px; opacity:0.7;">
          ${formatLocal(item.scheduled_start_at)} · ${escapeHtml(item.show_run_title)}
        </span>
      </button>
    `;

    const section = (title, items) => items && items.length
      ? `<section style="margin-bottom:10px;"><h4 style="margin:0 0 4px 0; font-size:12px; opacity:0.8;">${title}</h4>${items.map(rowHtml).join("")}</section>`
      : "";

    const historicalHtml = BAND_ORDER
      .filter((band) => (data.historical?.[band] || []).length)
      .map((band) => `
        <details style="margin-bottom:6px;">
          <summary style="cursor:pointer; font-size:12px; opacity:0.85;">${BAND_LABELS[band]} (${data.historical[band].length})</summary>
          ${data.historical[band].map(rowHtml).join("")}
        </details>
      `).join("");

    container.innerHTML =
      section("TODAY", data.today) +
      section("UPCOMING", data.upcoming) +
      (historicalHtml ? `<section><h4 style="margin:0 0 4px 0; font-size:12px; opacity:0.8;">HISTORICAL</h4>${historicalHtml}</section>` : "");

    if (!data.today?.length && !data.upcoming?.length && !historicalHtml) {
      container.innerHTML = `<div style="opacity:0.6; font-size:12px;">No Showings yet. Schedule one above.</div>`;
    }

    container.querySelectorAll("[data-showing]").forEach((btn) => {
      btn.addEventListener("click", () => selectShowing(panel, btn.dataset.showing));
    });
  }

  async function selectShowing(panel, showID) {
    state.selectedShowID = showID;
    const status = statusSetter(panel);
    status("Loading preflight…");
    let preflight;
    try {
      preflight = await api("/api/showtime/control", {
        method: "POST",
        body: JSON.stringify({ action: "preflight", show_id: showID }),
      });
    } catch (err) {
      status("Preflight failed: " + err.message, true);
      return;
    }
    state.lastPreflight = preflight;
    status("");
    renderPreflight(panel, preflight);
  }

  function renderPreflight(panel, p) {
    let host = panel.querySelector("[data-preflight]");
    if (!host) {
      host = document.createElement("div");
      host.dataset.preflight = "1";
      host.style.cssText = "margin-top:12px; border-top:1px solid #2a2e37; padding-top:10px;";
      panel.appendChild(host);
    }

    const blockers = p.blockers || [];
    const warnings = p.warnings || [];
    const canGo = blockers.length === 0;

    host.innerHTML = `
      <h4 style="margin:0 0 6px 0; font-size:13px;">Ready for Showtime — ${escapeHtml(p.nickname || p.show_title)}</h4>
      <div style="font-size:12px; opacity:0.85; margin-bottom:6px;">
        Venue: ${escapeHtml(p.venue_name || p.venue_slug || "—")}<br/>
        Current Scene: ${escapeHtml(p.current_scene_name || "—")}<br/>
        Cast: ${p.ready_character_count || 0} ready, ${p.incomplete_character_count || 0} still preparing<br/>
        Tickets: ${p.ticket_count || 0}<br/>
        Chat Bridge: ${p.chat_bridge_ready ? "ready" : "not ready"}
      </div>
      ${blockers.length ? `<div style="color:#e88; font-size:11px; margin-bottom:6px;">Blocked: ${blockers.map(escapeHtml).join(", ")}</div>` : ""}
      ${warnings.length ? `<div style="color:#e2c250; font-size:11px; margin-bottom:6px;">Note: ${warnings.map(escapeHtml).join(", ")}</div>` : ""}
      <button type="button" data-go-showtime ${canGo ? "" : "disabled"} style="${buttonStyle("width:100%; border-color:#8bd18b;" + (canGo ? "" : "opacity:0.5; cursor:not-allowed;"))}">
        ${p.is_live ? "Already Live — Refresh" : "GO TO SHOWTIME"}
      </button>
      <button type="button" data-end-showtime style="${buttonStyle("width:100%; margin-top:6px; border-color:#e88;")}">End Showtime</button>
      <div data-preflight-status style="margin-top:6px; font-size:11px;"></div>
    `;

    const pfStatus = (msg, isError) => {
      const s = host.querySelector("[data-preflight-status]");
      s.textContent = msg || "";
      s.style.color = isError ? "#e88" : "#8bd18b";
    };

    host.querySelector("[data-go-showtime]").addEventListener("click", async () => {
      pfStatus("Starting…");
      try {
        const result = await api("/api/showtime/control", {
          method: "POST",
          body: JSON.stringify({ action: "start", show_id: p.show_id }),
        });
        await renderRoot(result.message || "Showtime.");
      } catch (err) {
        pfStatus("Could not start: " + err.message, true);
      }
    });

    host.querySelector("[data-end-showtime]").addEventListener("click", async () => {
      pfStatus("Ending…");
      try {
        const result = await api("/api/showtime/control", {
          method: "POST",
          body: JSON.stringify({ action: "end", show_id: p.show_id }),
        });
        if (result.aftercare_eligible_count > 0) {
          await renderAftercareReminder(result);
        } else {
          await renderRoot(result.message || "Showtime ended.");
        }
      } catch (err) {
        pfStatus("Could not end: " + err.message, true);
      }
    });
  }

  // Kernel 89's own "Send Aftercare" panel (frontend/lib/stage-runtime/
  // kernel89-director-tools.js) is only loaded on live-stage venue pages,
  // not here on the Director's Chair -- so the reminder calls the same
  // Kernel 89 send endpoint directly rather than depending on that module.
  async function renderAftercareReminder(endResult) {
    const panel = state.panel;
    panel.innerHTML = `
      <div></div>
      <h3 style="margin:0 0 10px 0; font-size:15px;">${escapeHtml(endResult.message || "Showtime ended.")}</h3>
      <p style="margin:0 0 10px 0; font-size:12px;">Send Aftercare?</p>
      <button type="button" data-send-aftercare style="${buttonStyle("width:100%; border-color:#6a7cff; margin-bottom:6px;")}">Send Aftercare</button>
      <button type="button" data-not-yet style="${buttonStyle("width:100%;")}">Not Yet</button>
      <div data-status style="margin-top:10px; font-size:11px;"></div>
    `;
    panel.prepend(closeButton(closePanel));
    const status = statusSetter(panel);

    panel.querySelector("[data-not-yet]").addEventListener("click", () => renderRoot());
    panel.querySelector("[data-send-aftercare]").addEventListener("click", async () => {
      status("Sending…");
      try {
        await api(`/api/shows/${encodeURIComponent(endResult.show_id)}/aftercare/send`, {
          method: "POST",
          body: JSON.stringify({}),
        });
        status("Aftercare sent.");
      } catch (err) {
        status("Send failed: " + err.message, true);
      }
    });
  }

  function init() {
    const button = document.getElementById("showtime-open-button");
    if (button) button.addEventListener("click", openPanel);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  window.VictoryKernel92Showtime = { openPanel };
})();
