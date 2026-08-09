(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryPeoplePicker = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  // Kernel 85 §7: a reusable "pick a real person" widget, built to close a
  // concrete gap -- My People (renderMyPeopleSuggestions in board.html) and
  // Third Place both show a Player who someone is by stage name, but the
  // only identity field this codebase ever asked a sharer to type back in
  // (Victory handle) was never shown to them by either source
  // (playerprofile/types.go's Face projections deliberately never include
  // handle). Selecting a person here always yields their opaque
  // player_profile_workbooks.id (profile_id) -- the same identifier
  // playerrelationships and tickets already resolve server-side -- never a
  // raw account UUID or handle (kernel §7.2).

  function escapeHtml(value) {
    return String(value ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  // fetchPeople merges two sources -- My People (private, observer-only
  // relationships) and Third Place (anyone with an active Headshot placed).
  // Either source failing (e.g. Third Place 403s with trailer_face_not_ready
  // for a caller with no Trailer Face of their own yet) is silently
  // tolerated; this is a convenience picker, not a required surface.
  async function fetchPeople() {
    const out = [];

    try {
      const res = await fetch("/api/player-relationships?state=active", { credentials: "include" });
      const payload = await res.json().catch(() => null);
      if (res.ok && payload && payload.ok) {
        for (const rel of (payload.data && payload.data.relationships) || []) {
          const subject = rel.subject || {};
          if (!subject.subject_profile_id) continue;
          out.push({
            profile_id: subject.subject_profile_id,
            display_name: subject.stage_name || "Unnamed",
            portrait_url: subject.portrait_url || "",
            source: "my_people",
          });
        }
      }
    } catch (err) {
      // Best-effort source.
    }

    try {
      const res = await fetch("/api/third-place/headshots", { credentials: "include" });
      const payload = await res.json().catch(() => null);
      if (res.ok && payload && payload.ok) {
        for (const h of (payload.data && payload.data.headshots) || []) {
          if (!h.profile_id || h.is_you) continue;
          out.push({
            profile_id: h.profile_id,
            display_name: h.stage_name || "Unnamed",
            portrait_url: h.portrait_url || "",
            source: "third_place",
          });
        }
      }
    } catch (err) {
      // Best-effort source.
    }

    // De-dupe by profile_id -- the same person can show up in both lists.
    // My People's entry wins (listed first) since it is the more
    // deliberate, private "I know this person" signal.
    const byProfileId = new Map();
    for (const person of out) {
      if (!byProfileId.has(person.profile_id)) {
        byProfileId.set(person.profile_id, person);
      }
    }
    return Array.from(byProfileId.values());
  }

  let activeOverlay = null;

  function close() {
    if (activeOverlay) {
      activeOverlay.remove();
      activeOverlay = null;
    }
  }

  function ensureStyles() {
    if (document.getElementById("victory-people-picker-styles")) return;
    const style = document.createElement("style");
    style.id = "victory-people-picker-styles";
    style.textContent = `
      .victory-people-picker-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center; z-index: 10000; }
      .victory-people-picker-modal { background: #1b1511; color: #f6ecdd; border: 1px solid rgba(255,255,255,0.14); border-radius: 14px; padding: 20px; max-width: 420px; width: 92%; max-height: 80vh; overflow-y: auto; box-sizing: border-box; font-family: inherit; }
      .victory-people-picker-modal h3 { margin: 0 0 6px; }
      .victory-people-picker-hint { margin: 0 0 14px; font-size: 0.82rem; opacity: 0.75; }
      .victory-people-picker-list { display: grid; gap: 8px; margin-bottom: 14px; }
      .victory-people-picker-row { display: flex; align-items: center; gap: 10px; padding: 8px 10px; border-radius: 10px; border: 1px solid rgba(255,255,255,0.1); background: rgba(255,255,255,0.03); color: inherit; font: inherit; cursor: pointer; text-align: left; width: 100%; box-sizing: border-box; }
      .victory-people-picker-row:hover { background: rgba(255,255,255,0.08); }
      .victory-people-picker-portrait { width: 32px; height: 32px; border-radius: 8px; object-fit: cover; flex-shrink: 0; background: rgba(255,255,255,0.08); display: inline-flex; align-items: center; justify-content: center; }
      .victory-people-picker-namewrap { display: flex; flex-direction: column; align-items: flex-start; gap: 2px; min-width: 0; }
      .victory-people-picker-id { font-size: 0.72rem; opacity: 0.55; font-family: ui-monospace, monospace; }
      .victory-people-picker-actions { display: flex; justify-content: flex-end; }
      .victory-people-picker-actions button { padding: 8px 14px; border-radius: 8px; border: 1px solid rgba(255,255,255,0.14); background: rgba(255,255,255,0.04); color: inherit; cursor: pointer; font: inherit; }
      .victory-people-picker-empty, .victory-people-picker-loading { opacity: 0.75; font-size: 0.88rem; }
    `;
    document.head.appendChild(style);
  }

  // open(options): options = { title, hint, onSelect(person) }.
  // person = { profile_id, display_name, portrait_url, source }.
  async function open(options) {
    close();
    ensureStyles();
    const opts = options || {};

    const overlay = document.createElement("div");
    overlay.className = "victory-people-picker-overlay";
    overlay.innerHTML = `
      <div class="victory-people-picker-modal" role="dialog" aria-modal="true">
        <h3>${escapeHtml(opts.title || "Choose a Person")}</h3>
        <p class="victory-people-picker-hint">${escapeHtml(opts.hint || "From My People and Third Place.")}</p>
        <div class="victory-people-picker-list" data-role="list"><p class="victory-people-picker-loading">Loading…</p></div>
        <div class="victory-people-picker-actions">
          <button type="button" data-role="close">Cancel</button>
        </div>
      </div>
    `;
    document.body.appendChild(overlay);
    activeOverlay = overlay;

    overlay.querySelector('[data-role="close"]').addEventListener("click", close);
    overlay.addEventListener("click", (event) => {
      if (event.target === overlay) close();
    });

    const listEl = overlay.querySelector('[data-role="list"]');
    const people = await fetchPeople();
    if (activeOverlay !== overlay) return; // closed while loading

    if (!people.length) {
      listEl.innerHTML = `<p class="victory-people-picker-empty">Nobody to show yet. Add someone in My People, or visit Third Place first.</p>`;
      return;
    }

    listEl.innerHTML = "";
    for (const person of people) {
      const row = document.createElement("button");
      row.type = "button";
      row.className = "victory-people-picker-row";
      const portrait = person.portrait_url
        ? `<img src="${escapeHtml(person.portrait_url)}" alt="" class="victory-people-picker-portrait" />`
        : `<span class="victory-people-picker-portrait">☺</span>`;
      // Kernel §7.2 asks for a "canonical stable user ID visible/copyable
      // where useful" alongside display name/Face. The kernel spec was
      // written before the repo audit surfaced that this codebase
      // deliberately never exposes the raw account UUID or Victory handle
      // through any people-facing projection (playerprofile/types.go).
      // Resolution: the *profile_id* already selected here IS a canonical,
      // stable, per-person identifier in this codebase's own vocabulary
      // (playerrelationships/tickets both resolve it server-side the same
      // way) -- showing it satisfies "enough identity information to know
      // I have the right person" without reopening the handle/UUID
      // privacy boundary Kernel 62/65 established on purpose.
      row.innerHTML = `${portrait}<span class="victory-people-picker-namewrap"><span class="victory-people-picker-name">${escapeHtml(person.display_name)}</span><span class="victory-people-picker-id">ID: ${escapeHtml(person.profile_id)}</span></span>`;
      row.addEventListener("click", () => {
        close();
        if (typeof opts.onSelect === "function") opts.onSelect(person);
      });
      listEl.appendChild(row);
    }
  }

  return { open, close };
});
