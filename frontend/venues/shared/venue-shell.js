(function () {
  const STORAGE_PREFIX = "victory.venue.ui.v1:";

  function normalizeVenueSlug(value) {
    const slug = String(value || "")
      .trim()
      .toLowerCase()
      .replace(/[\s_]+/g, "-")
      .replace(/[^a-z0-9-]/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "");
    return slug || "venue";
  }

  function formatVenueName(value) {
    const text = String(value || "").trim();
    if (!text) return "";
    return text
      .split("-")
      .map((part) => {
        if (!part) return "";
        return `${part[0].toUpperCase()}${part.slice(1)}`;
      })
      .join(" ")
      .replace(/\s+/g, " ")
      .trim();
  }

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

  function resolveShellNode(target) {
    if (!target) return null;
    if (typeof target === "string") {
      return document.querySelector(target);
    }
    if (target instanceof Element) {
      return target;
    }
    return null;
  }

  function resolveShellSlots(options = {}) {
    const slots = options.slots || options || {};
    return {
      header: resolveShellNode(slots.header),
      leftTray: resolveShellNode(slots.leftTray),
      rightTray: resolveShellNode(slots.rightTray),
      chatRail: resolveShellNode(slots.chatRail || slots.chat),
      audioTray: resolveShellNode(slots.audioTray || slots.audio),
      sessionChip: resolveShellNode(slots.sessionChip),
      houseMicChip: resolveShellNode(slots.houseMicChip),
      bridgeChip: resolveShellNode(slots.bridgeChip),
    };
  }

  function registerHeaderChip(target, options = {}) {
    const chip = resolveShellNode(target);
    if (!chip) return null;

    const label = String(options.label || options.text || "").trim();
    const sublabel = String(options.sublabel || options.detail || "").trim();
    const title = String(options.title || "").trim();

    chip.dataset.victoryShellChip = "true";
    if (label) {
      chip.textContent = label;
    }

    if (sublabel) {
      chip.dataset.victoryShellChipDetail = sublabel;
    } else {
      delete chip.dataset.victoryShellChipDetail;
    }

    if (title) {
      chip.title = title;
    }

    if (typeof options.onClick === "function") {
      chip.addEventListener("click", options.onClick);
    }

    return chip;
  }

  function safeRefresh(handler, fallback = null) {
    try {
      if (typeof handler === "function") {
        return Promise.resolve(handler());
      }
    } catch (error) {
      console.warn("venue shell refresh handler failed", error);
      if (typeof fallback === "function") {
        return Promise.resolve(fallback(error));
      }
      return Promise.reject(error);
    }

    if (typeof fallback === "function") {
      return Promise.resolve(fallback());
    }

    return Promise.resolve().then(() => {
      window.location.reload();
    });
  }

  function mount(options = {}) {
    const root = resolveShellNode(options.root);
    const venueSlug = normalizeVenueSlug(options.venueSlug || root?.dataset?.venueSlug);
    const venueName = String(options.venueName || root?.dataset?.venueName || formatVenueName(venueSlug)).trim();
    const slots = resolveShellSlots(options.slots || options);

    if (root) {
      root.dataset.victoryShell = "true";
      root.dataset.venueSlug = venueSlug;
      if (venueName) {
        root.dataset.venueName = venueName;
      }
    }

    if (slots.header) {
      slots.header.dataset.venueSlug = venueSlug;
      if (venueName) {
        slots.header.dataset.venueName = venueName;
      }
    }

    if (slots.leftTray) slots.leftTray.dataset.shellTray = "left";
    if (slots.rightTray) slots.rightTray.dataset.shellTray = "right";
    if (slots.chatRail) slots.chatRail.dataset.shellRail = "chat";
    if (slots.audioTray) slots.audioTray.dataset.shellRail = "audio";

    if (typeof options.onRefresh === "function") {
      const refreshTarget = resolveShellNode(options.refreshTarget);
      if (refreshTarget) {
        refreshTarget.addEventListener("click", (event) => {
          event.preventDefault();
          safeRefresh(options.onRefresh, options.onRefreshFallback);
        });
      }
    }

    return {
      root,
      venueSlug,
      venueName,
      slots,
      refresh: () => safeRefresh(options.onRefresh, options.onRefreshFallback),
    };
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
    formatVenueName,
    loadPreferences,
    mount,
    mountOverlay,
    normalizeVenueSlug,
    normalizeDrawerMode,
    normalizePortraitSize,
    resolveStorageKey,
    registerHeaderChip,
    resolveShellSlots,
    safeRefresh,
    overlayMarkup,
    renderPresencePreview,
    savePreferences,
  };
})();
