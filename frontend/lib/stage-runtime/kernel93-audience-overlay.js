// Kernel 93: the Audience seat's own left overlay.
//
// Deliberately independent of the rest of the stage-runtime engine (no
// dependency on runtime.js's closures/bridges) -- it does its own read-only
// fetch of the same GET /api/world/{venue} snapshot endpoint
// session-sync.js's refreshWorld() already calls, and only ever touches DOM
// it owns (#audience-drawer and friends) plus hiding #left-drawer/
// #right-drawer for this one viewer. No Director/Cast-only controls are
// ever rendered here (spec §6): if this module has a bug, the failure mode
// is "the Audience overlay doesn't work," never "Audience sees backstage
// tools."
//
// No-ops entirely on any page that doesn't have #audience-drawer in the DOM
// -- safe to load on every venue via the shared /lib/stage-runtime/ path,
// effective only where a venue's own HTML opts in (currently Catharsis
// only, per spec §1/§22).
(function () {
  "use strict";

  function venueSlug() {
    return String((window.VictoryStageVenue || {}).slug || "").trim();
  }

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;");
  }

  function applySnapshot(snapshot) {
    const drawer = document.getElementById("audience-drawer");
    if (!drawer) return;

    const kind = String(snapshot?.theater_context?.kind || "").trim().toLowerCase();
    const leftDrawer = document.getElementById("left-drawer");
    const rightDrawer = document.getElementById("right-drawer");
    const leftEdge = document.getElementById("left-drawer-edge-trigger");
    const rightEdge = document.getElementById("right-drawer-edge-trigger");
    const chatPanel = document.getElementById("chat-panel");
    const topBarEl = document.getElementById("top-bar");
    if (kind !== "audience") {
      // Not (or no longer) resolved as Audience for this venue. This module
      // polls independently every 10s (see boot() below), so a viewer whose
      // role/theater_context genuinely changes mid-session (or who was ever
      // transiently misresolved as "audience" -- the exact Kernel 101 101-20
      // bug) must have the Cast/Director drawers explicitly RESTORED here,
      // not just left alone: this same function is what hid them via
      // inline style in the first place, and nothing else in this file ever
      // clears that style. Without this, one bad poll permanently hides a
      // Cast/Director's own tools for the rest of the browser tab's life,
      // survivable only by a full page reload.
      if (leftDrawer) leftDrawer.style.removeProperty("display");
      if (rightDrawer) rightDrawer.style.removeProperty("display");
      if (leftEdge) leftEdge.style.removeProperty("display");
      if (rightEdge) rightEdge.style.removeProperty("display");
      if (chatPanel) chatPanel.style.removeProperty("display");
      if (topBarEl) topBarEl.style.removeProperty("display");
      drawer.hidden = true;
      return;
    }

    // Kernel 93 §6: Audience gets this simpler overlay INSTEAD of the
    // Cast/Director drawers, never alongside them -- those carry Stage
    // Controls and Character-authoring tools that are not Audience's to see.
    // 2026-08-28 live testing: chat also goes away for Audience -- the
    // Note to the Director box + reactions bar are the intended
    // replacement (ledger A19), not a third channel alongside them. Same
    // pass: #top-bar (brand/status chips, camera/target controls, header-
    // right buttons -- all stage tooling Audience has no use for) is
    // hidden outright, not trimmed down -- catharsis/index.html separately
    // loads /lib/back-to-map.js with data-back-to-map="floating", which
    // mounts its own independent, always-visible "Back to Map" pill fixed
    // to the page (not inside #top-bar), so nothing here needs to survive
    // for that. The theater-context message ("You are part of the
    // Audience") is a separate floating banner (runtime.js's
    // #kernel70a-theater-context-banner), untouched by any of this.
    if (leftDrawer) leftDrawer.style.display = "none";
    if (rightDrawer) rightDrawer.style.display = "none";
    if (leftEdge) leftEdge.style.display = "none";
    if (rightEdge) rightEdge.style.display = "none";
    if (chatPanel) chatPanel.style.display = "none";
    if (topBarEl) topBarEl.style.display = "none";
    drawer.hidden = false;
    bindDrawerToggle();
    bindNoteSend();

    const cfg = snapshot?.audience_config || {};

    const presenceItem = document.getElementById("audience-presence-item");
    const presenceDetails = document.getElementById("audience-presence-details");
    if (presenceItem) {
      presenceItem.hidden = !cfg.show_presence;
      if (cfg.show_presence && presenceDetails) {
        // Spec §5/§17: starts collapsed, may be opened locally -- <details>
        // gives us this for free without tracking any state ourselves.
        const preview = document.getElementById("audience-presence-preview");
        if (preview && !preview.dataset.filled) {
          preview.innerHTML = '<span class="presence-preview-note">Presence sharing is on for this Showing.</span>';
          preview.dataset.filled = "1";
        }
      }
    }

    const healthItem = document.getElementById("audience-health-item");
    if (healthItem) {
      healthItem.hidden = !cfg.show_health_statuses;
      if (cfg.show_health_statuses) {
        const list = document.getElementById("audience-health-list");
        if (list && !list.dataset.filled) {
          list.innerHTML = '<span class="presence-preview-note">Qualitative health status is on for this Showing.</span>';
          list.dataset.filled = "1";
        }
      }
    }
  }

  let drawerToggleBound = false;
  function bindDrawerToggle() {
    if (drawerToggleBound) return;
    const head = document.getElementById("audience-drawer-head");
    const drawer = document.getElementById("audience-drawer");
    if (!head || !drawer) return;
    drawerToggleBound = true;
    const toggle = () => {
      const open = drawer.dataset.open === "true";
      drawer.dataset.open = open ? "false" : "true";
      head.setAttribute("aria-expanded", open ? "false" : "true");
    };
    head.addEventListener("click", toggle);
    head.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        toggle();
      }
    });
  }

  // A19: sending never depends on the drawer having a fresh snapshot --
  // the backend (audiencenotes.Submit) resolves the live session itself,
  // so this is a plain, independent fetch, same posture as the rest of
  // this module.
  let noteSendBound = false;
  function bindNoteSend() {
    if (noteSendBound) return;
    const button = document.getElementById("audience-note-send");
    const textarea = document.getElementById("audience-note-body");
    const status = document.getElementById("audience-note-status");
    if (!button || !textarea) return;
    noteSendBound = true;
    button.addEventListener("click", async () => {
      const body = textarea.value.trim();
      if (!body) return;
      button.disabled = true;
      if (status) status.textContent = "Sending...";
      try {
        const response = await fetch("/api/session/catharsis/notes", {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ body }),
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok) {
          throw new Error(payload?.data?.error || `HTTP ${response.status}`);
        }
        textarea.value = "";
        if (status) status.textContent = "Sent to the Director.";
      } catch (error) {
        if (status) status.textContent = `Couldn't send: ${error.message || error}`;
      } finally {
        button.disabled = false;
      }
    });
  }

  async function refresh() {
    const slug = venueSlug();
    if (!slug || !document.getElementById("audience-drawer")) return;
    try {
      const response = await fetch(`/api/world/${encodeURIComponent(slug)}`, {
        method: "GET",
        credentials: "include",
        cache: "no-store",
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok || !payload?.data) return;
      applySnapshot(payload.data);
    } catch (error) {
      console.warn("audience overlay snapshot refresh failed", error);
    }
  }

  function boot() {
    if (!document.getElementById("audience-drawer")) return;
    refresh();
    // Kernel 93 setup pass: a light poll rather than a live WS subscription
    // (this module is deliberately independent of the socket pipeline) --
    // enough for a Director's toggle change to reach an already-connected
    // Audience viewer within a few seconds. Pass B may replace this with a
    // real-time hook once live-tested to be worth the added coupling.
    window.setInterval(refresh, 10000);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot, { once: true });
  } else {
    boot();
  }
})();
