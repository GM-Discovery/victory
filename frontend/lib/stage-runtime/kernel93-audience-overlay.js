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
    if (kind !== "audience") {
      // Not (or no longer) resolved as Audience for this venue -- leave the
      // ordinary Cast/Director drawers alone and keep this overlay hidden.
      drawer.hidden = true;
      return;
    }

    const leftDrawer = document.getElementById("left-drawer");
    const rightDrawer = document.getElementById("right-drawer");
    const leftEdge = document.getElementById("left-drawer-edge-trigger");
    const rightEdge = document.getElementById("right-drawer-edge-trigger");
    // Kernel 93 §6: Audience gets this simpler overlay INSTEAD of the
    // Cast/Director drawers, never alongside them -- those carry Stage
    // Controls and Character-authoring tools that are not Audience's to see.
    if (leftDrawer) leftDrawer.style.display = "none";
    if (rightDrawer) rightDrawer.style.display = "none";
    if (leftEdge) leftEdge.style.display = "none";
    if (rightEdge) rightEdge.style.display = "none";
    drawer.hidden = false;

    const diagnosticLine = document.getElementById("audience-diagnostic-line");
    if (diagnosticLine) {
      const showingStatus = String(snapshot?.showing?.status || "").trim();
      const venueName = String(snapshot?.venue?.name || venueSlug()).trim();
      diagnosticLine.textContent = showingStatus
        ? `Watching ${venueName} -- Showing is ${showingStatus}.`
        : `Watching ${venueName}.`;
    }

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
