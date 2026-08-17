// Kernel 90: the Director's canonical stage-object visibility surface.
//
// This module is the UI half of ONE reconciliation. Before it, "who can see
// this?" had three unrelated answers (a replayed reveal/hide action log, a
// set of deferred Cue actions, and a global participant-interaction boolean).
// Every control here writes the same canonical state a Cue writes, through
// the same endpoint, so a Director and a Cue can never disagree (§24).
//
// Nothing here is authoritative. The server re-resolves the Show, the object,
// the actor's Director+ authority, and every scope target on each call; this
// file decides only what a Director is shown. In particular it never decides
// what a PLAYER sees -- hidden objects are omitted from a Player's snapshot
// server-side (§36), so there is no client-side concealment to get wrong.
//
// Deliberately not built here: fog-of-war of any kind (§46), per-Cohort Scene
// copies (§47), and any form of condition or trigger authoring (§48). The
// scope panel edits a set of named audiences and nothing else.
(function () {
  "use strict";

  const state = {
    scopePanel: null,
    scopeTargets: null,
    scopeTargetsShowID: "",
  };

  function bridge() {
    return window.VictoryStageKernel88Bridge || window.VictoryStageKernel85Bridge || null;
  }

  function showID() {
    return bridge()?.getShowID?.() || "";
  }

  function canManage() {
    return Boolean(bridge()?.canManageStage?.());
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

  const PANEL_STYLE = `
    position: fixed; z-index: 9000; background: #1b1e24; color: #e8e8ec;
    border: 1px solid #3a3f4b; border-radius: 8px; box-shadow: 0 8px 28px rgba(0,0,0,0.45);
    font: 13px/1.4 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    padding: 14px; width: 320px; top: 90px; right: 24px;
  `;

  function buttonStyle(extra = "") {
    return `background:#2b303b; color:#e8e8ec; border:1px solid #454b59; border-radius:6px; padding:6px 10px; cursor:pointer; font-size:12px; ${extra}`;
  }

  // The object's canonical reference, read from the snapshot the server sent.
  // Mirrors logic.js's canonicalStageObjectRef rather than importing it,
  // because this module is loaded as a plain script alongside the engine (the
  // same arrangement every other kernel tool module uses) and the shape is
  // two string fields.
  function refFor(objectModel) {
    const st = objectModel?.state || objectModel?.source?.state || {};
    const kind = String(st.stage_object_kind || "").trim();
    const id = String(st.stage_object_id || "").trim();
    if (!kind || !id) return null;
    return { kind, id };
  }

  function interactionRefFor(objectModel) {
    const id = String(objectModel?.source?.data?.binding?.participant_interaction_id || "").trim();
    if (!id) return null;
    return { kind: "participant_interaction", id };
  }

  function currentScopes(objectModel) {
    const st = objectModel?.state || objectModel?.source?.state || {};
    const raw = Array.isArray(st.visibility_scopes) ? st.visibility_scopes : [];
    return raw
      .map((s) => ({
        scope_kind: String(s?.scope_kind || "").trim(),
        scope_id: String(s?.scope_id || "").trim(),
      }))
      .filter((s) => s.scope_kind);
  }

  function status(message, isError) {
    // Reuse the engine's own stage status line rather than inventing a second
    // place for a Director to look for feedback.
    const setter = bridge()?.setStageStatus;
    if (typeof setter === "function") {
      setter(message);
    }
    if (isError) console.warn("kernel90:", message);
  }

  async function postMutation(body) {
    const id = showID();
    if (!id) throw new Error("no_show");
    return api(`/api/shows/${encodeURIComponent(id)}/stage-object-states`, {
      method: "POST",
      body: JSON.stringify(body),
    });
  }

  // applyOperation is the single write this module performs for the four
  // canonical operations. The interaction operations target the bound
  // participant interaction, not the element -- §8 keeps the two dimensions
  // separate, and an interaction's own identity is what the Cue actions
  // target too, so manual and Cue paths address the same row.
  async function applyOperation(objectModel, operation) {
    if (!canManage()) return;
    const isInteractionOp = operation === "enable_interaction" || operation === "disable_interaction";
    const ref = isInteractionOp ? interactionRefFor(objectModel) : refFor(objectModel);
    if (!ref) {
      status("That object has no canonical identity to target.", true);
      return;
    }
    try {
      await postMutation({ object_kind: ref.kind, object_id: ref.id, operation });
      // No optimistic local mutation: the server broadcasts a stage
      // invalidation and the resulting snapshot is the truth. Guessing here
      // would be wrong in the one case that matters -- revealing an object a
      // viewer does not yet have.
      status(`${operation.replaceAll("_", " ")} applied.`);
    } catch (err) {
      status(`Could not apply: ${err.message}`, true);
    }
  }

  async function loadScopeTargets() {
    const id = showID();
    if (!id) throw new Error("no_show");
    if (state.scopeTargets && state.scopeTargetsShowID === id) {
      return state.scopeTargets;
    }
    const data = await api(`/api/shows/${encodeURIComponent(id)}/stage-object-scope-targets`);
    state.scopeTargets = data;
    state.scopeTargetsShowID = id;
    return data;
  }

  function closeScopePanel() {
    if (state.scopePanel) {
      state.scopePanel.remove();
      state.scopePanel = null;
    }
  }

  // openScopePanel edits the grant set for one object: the list of audiences
  // that may perceive it while it is hidden.
  //
  // The panel says so explicitly, because the single most confusing thing
  // about scoped visibility is that grants do nothing to a VISIBLE object.
  // Rather than hide that rule, the panel states it and offers to hide the
  // object at the same time.
  async function openScopePanel(objectModel) {
    if (!canManage()) return;
    const ref = refFor(objectModel);
    if (!ref) {
      status("That object has no canonical identity to scope.", true);
      return;
    }
    closeScopePanel();

    let targets;
    try {
      targets = await loadScopeTargets();
    } catch (err) {
      status(`Could not load scope targets: ${err.message}`, true);
      return;
    }

    const selected = new Set(
      currentScopes(objectModel).map((s) => `${s.scope_kind}:${s.scope_id}`)
    );
    const label = String(objectModel?.label || "this object");

    const panel = document.createElement("div");
    panel.setAttribute("style", PANEL_STYLE);
    panel.setAttribute("role", "dialog");
    panel.setAttribute("aria-label", `Visibility scope for ${label}`);

    const rows = [];
    (targets.tiers || []).forEach((tier) => {
      rows.push({ kind: tier.scope_kind, id: "", label: tier.label });
    });
    (targets.cohorts || []).forEach((c) => {
      rows.push({ kind: "cohort", id: c.id, label: `Cohort: ${c.label}` });
    });
    (targets.characters || []).forEach((c) => {
      rows.push({ kind: "character", id: c.id, label: `Character: ${c.label}` });
    });

    panel.innerHTML = `
      <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:8px;">
        <strong style="font-size:13px;">Visibility scope</strong>
        <button type="button" data-close aria-label="Close" style="${buttonStyle("padding:2px 7px;")}">✕</button>
      </div>
      <div style="font-size:12px; color:#a9b0bd; margin-bottom:10px;">
        ${escapeHtml(label)} — who may see it <em>while it is hidden</em>.
        Scopes have no effect on a visible object.
      </div>
      <div data-rows style="max-height:220px; overflow:auto; display:flex; flex-direction:column; gap:5px;">
        ${rows.map((r, i) => `
          <label style="display:flex; gap:7px; align-items:center; font-size:12px;">
            <input type="checkbox" data-scope-index="${i}" ${selected.has(`${r.kind}:${r.id}`) ? "checked" : ""} />
            <span>${escapeHtml(r.label)}</span>
          </label>
        `).join("")}
        ${rows.length ? "" : `<div style="font-size:12px; color:#a9b0bd;">No Cohorts or roster Characters yet.</div>`}
      </div>
      <div style="font-size:12px; color:#a9b0bd; margin:10px 0 8px;">
        With nothing selected, a hidden object is Director-only.
      </div>
      <div style="display:flex; gap:6px; flex-wrap:wrap;">
        <button type="button" data-save style="${buttonStyle()}">Save scope</button>
        <button type="button" data-save-hide style="${buttonStyle()}">Save &amp; hide</button>
      </div>
      <div data-status style="margin-top:8px; font-size:12px; min-height:1em;"></div>
    `;

    const setStatus = (msg, isError) => {
      const el = panel.querySelector("[data-status]");
      if (!el) return;
      el.textContent = msg || "";
      el.style.color = isError ? "#e88" : "#8bd18b";
    };

    const collect = () =>
      rows
        .filter((_r, i) => panel.querySelector(`[data-scope-index="${i}"]`)?.checked)
        .map((r) => (r.id ? { scope_kind: r.kind, scope_id: r.id } : { scope_kind: r.kind }));

    const save = async (alsoHide) => {
      try {
        // set_scopes replaces the whole grant set, so an unchecked box is a
        // removal. The two-request form for "save and hide" is deliberate:
        // it keeps the scope write and the visibility write as the same two
        // canonical operations a Cue would perform, rather than inventing a
        // combined server operation that only this panel could produce.
        await postMutation({
          object_kind: ref.kind,
          object_id: ref.id,
          operation: "set_scopes",
          scopes: collect(),
        });
        if (alsoHide) {
          await postMutation({ object_kind: ref.kind, object_id: ref.id, operation: "hide_object" });
        }
        setStatus("Saved.");
        closeScopePanel();
      } catch (err) {
        setStatus(`Could not save: ${err.message}`, true);
      }
    };

    panel.querySelector("[data-close]")?.addEventListener("click", closeScopePanel);
    panel.querySelector("[data-save]")?.addEventListener("click", () => save(false));
    panel.querySelector("[data-save-hide]")?.addEventListener("click", () => save(true));
    // §44: the panel is reachable and dismissible from the keyboard even
    // though the context menu that opens it is pointer-driven.
    panel.addEventListener("keydown", (event) => {
      if (event.key === "Escape") closeScopePanel();
    });

    document.body.appendChild(panel);
    state.scopePanel = panel;
    panel.querySelector("input,button")?.focus();
  }

  window.VictoryKernel90VisibilityTools = {
    applyOperation,
    openScopePanel,
    closeScopePanel,
  };
})();
