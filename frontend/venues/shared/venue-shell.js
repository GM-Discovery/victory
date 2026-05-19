(function () {
  const STORAGE_PREFIX = "victory.venue.ui.v1:";

  function clampNumber(value, min, max, fallback) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(min, Math.min(max, parsed));
  }

  function normalizeDrawerMode(value) {
    const mode = String(value || "").trim();
    if (mode === "always-open" || mode === "always-closed") return mode;
    return "hover";
  }

  function normalizePortraitSize(value) {
    const size = String(value || "").trim();
    if (size === "small" || size === "large") return size;
    return "medium";
  }

  function resolveStorageKey(key, prefix) {
    const safePrefix = String(prefix || STORAGE_PREFIX).trim() || STORAGE_PREFIX;
    return `${safePrefix}${key}`;
  }

  function loadPreferences(key, defaults, options = {}) {
    const base = JSON.parse(JSON.stringify(defaults || {}));
    if (!key) return base;

    try {
      const raw = localStorage.getItem(resolveStorageKey(key, options.storagePrefix));
      if (!raw) return base;
      const parsed = JSON.parse(raw);
      if (!parsed || typeof parsed !== "object") return base;

      for (const [section, value] of Object.entries(parsed)) {
        if (!value || typeof value !== "object" || Array.isArray(value)) continue;
        base[section] = {
          ...(base[section] || {}),
          ...value,
        };
      }

      if (base.left) {
        base.left.mode = normalizeDrawerMode(base.left.mode);
        base.left.opacity = clampNumber(base.left.opacity, 70, 100, base.left.opacity ?? 96);
        base.left.portraitSize = normalizePortraitSize(base.left.portraitSize);
      }

      if (base.right) {
        base.right.mode = normalizeDrawerMode(base.right.mode);
        base.right.opacity = clampNumber(base.right.opacity, 70, 100, base.right.opacity ?? 96);
      }
    } catch (error) {
      console.warn("venue shell prefs load failed", error);
    }

    return base;
  }

  function savePreferences(key, prefs, options = {}) {
    if (!key) return;
    try {
      localStorage.setItem(resolveStorageKey(key, options.storagePrefix), JSON.stringify(prefs));
    } catch (error) {
      console.warn("venue shell prefs save failed", error);
    }
  }

  function renderPresencePreview(root, users, options = {}) {
    if (!root) return "";

    const rows = Array.isArray(users) ? users.slice(0, 6) : [];
    const avatarSize = options.avatarSize || "36px";

    if (rows.length === 0) {
      root.innerHTML = `
        <span class="presence-avatar presence-avatar--fallback" style="--portrait-size:${avatarSize}" aria-hidden="true">?</span>
        <span class="presence-avatar presence-avatar--fallback" style="--portrait-size:${avatarSize}" aria-hidden="true">?</span>
        <span class="presence-avatar presence-avatar--fallback" style="--portrait-size:${avatarSize}" aria-hidden="true">?</span>
      `;
      return "No one is currently in the room.";
    }

    const displayNameForUser = (user) => String(user?.display_name || user?.handle || "Unknown Participant").trim();
    const presenceInitials = (user) => {
      const label = displayNameForUser(user);
      if (!label) return "?";
      const parts = label.split(/\s+/).filter(Boolean);
      if (parts.length === 1) {
        return parts[0].slice(0, 2).toUpperCase();
      }
      return `${parts[0][0] || ""}${parts[1][0] || ""}`.toUpperCase();
    };

    root.innerHTML = rows.map((user) => {
      const portraitURL = String(user?.persona?.portrait_url || user?.portrait_url || "").trim();
      const title = `${displayNameForUser(user)}${user?.role ? ` · ${user.role}` : ""}`;
      if (portraitURL) {
        return `
          <span class="presence-avatar" style="--portrait-size:${avatarSize}" title="${title.replaceAll('"', "&quot;")}">
            <img src="${portraitURL.replaceAll('"', "&quot;")}" alt="${displayNameForUser(user).replaceAll('"', "&quot;")}" />
          </span>
        `;
      }
      return `
        <span class="presence-avatar presence-avatar--fallback" style="--portrait-size:${avatarSize}" title="${title.replaceAll('"', "&quot;")}">
          ${presenceInitials(user)}
        </span>
      `;
    }).join("");

    const names = rows.map(displayNameForUser);
    const extraCount = Math.max(0, rows.length - 3);
    return extraCount > 0
      ? `${rows.length} connected: ${names.slice(0, 3).join(", ")} +${extraCount} more`
      : `${rows.length} connected: ${names.join(", ")}`;
  }

  function overlayMarkup(options = {}) {
    const smokeEnabled = Boolean(options.smokeEnabled);
    return `
      <div class="overlay-panel">
        <div class="overlay-row">
          <strong>Session</strong>
          <span id="session-line">Not joined</span>
        </div>
        <div class="overlay-row">
          <strong>Role</strong>
          <span id="role-line">Unknown</span>
        </div>
        <div class="overlay-row">
          <strong>Objects</strong>
          <span id="object-count-line">0</span>
        </div>
        <div class="overlay-actions">
          <button id="refresh-world" type="button" title="Reload the latest world snapshot from the server and re-render the stage.">Reload snapshot</button>
          <button id="clear-selection" type="button">Clear</button>
        </div>
        <div class="overlay-note">
          Right-click the stage to create a real index card. Right-click a live Pixi object for its menu;
          dragging a live object still uses the existing <code>act/place_element</code> path.
          Reload snapshot fetches the latest world state from the server and re-renders the stage.
          Smoke mode adds <code>g</code> for grid and <code>d</code> for drop test card.
        </div>
        <div id="smoke-panel" class="smoke-tools"${smokeEnabled ? "" : " hidden"}>
          <div class="overlay-row">
            <strong>Smoke</strong>
            <span id="smoke-line">Open this page with <code>?smoke=1</code>.</span>
          </div>
          <div class="overlay-actions">
            <button id="smoke-toggle-grid" type="button">Grid: Off</button>
            <button id="smoke-drop-card" type="button">Drop Test Card</button>
          </div>
        </div>
      </div>
      <div id="card-editor" class="card-editor" hidden>
        <div id="card-editor-header" class="card-editor-header">
          <strong>Inspect Card</strong>
          <span class="card-editor-grip">drag</span>
        </div>
        <div id="card-editor-status" class="editor-status">Select a live index card.</div>
        <label class="editor-field" for="card-editor-front">
          <span>Front Text</span>
          <textarea id="card-editor-front" rows="4"></textarea>
        </label>
        <label class="editor-field" for="card-editor-back">
          <span>Back Text</span>
          <textarea id="card-editor-back" rows="3"></textarea>
        </label>
        <label class="editor-field" for="card-editor-color">
          <span>Color</span>
          <input id="card-editor-color" type="color" />
        </label>
        <div class="overlay-actions">
          <button id="card-editor-save" type="button">Save</button>
          <button id="card-editor-cancel" type="button">Close</button>
        </div>
      </div>
      <div id="pixi-context-menu" class="context-menu" hidden></div>
    `;
  }

  function mountOverlay(root, options = {}) {
    if (!root) return;
    root.innerHTML = overlayMarkup(options);
  }

  window.VictoryVenueShell = {
    clampNumber,
    loadPreferences,
    mountOverlay,
    normalizeDrawerMode,
    normalizePortraitSize,
    resolveStorageKey,
    overlayMarkup,
    renderPresencePreview,
    savePreferences,
  };
})();
