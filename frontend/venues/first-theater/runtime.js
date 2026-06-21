    const stageShell = document.getElementById("stage-shell");
    const stageHost = document.getElementById("pixi-stage-host");
    const stageStatus = document.getElementById("stage-status");
    const liveFeedLine = document.getElementById("live-feed-line");
    const selectionLine = document.getElementById("selection-line");
    const pointerLine = document.getElementById("pointer-line");
    const movementLine = document.getElementById("movement-line");
    const sessionLine = document.getElementById("session-line");
    const roleLine = document.getElementById("role-line");
    const objectCountLine = document.getElementById("object-count-line");
    const stageEmptyState = document.getElementById("stage-empty-state");
    const snapshotSummary = document.getElementById("snapshot-summary");
    const rendererFallback = document.getElementById("renderer-fallback");
    const rendererFallbackLine = document.getElementById("renderer-fallback-line");
    const overlayRoot = document.getElementById("overlay-root");
    const overlayAnchor = document.getElementById("overlay-anchor");
    const overlayBadge = document.getElementById("overlay-badge");
    const refreshWorldButton = document.getElementById("refresh-world");
    const clearSelectionButton = document.getElementById("clear-selection");
    const selectedActions = document.getElementById("selected-actions");
    const contextMenu = document.getElementById("pixi-context-menu");
    const smokePanel = document.getElementById("smoke-panel");
    const smokeLine = document.getElementById("smoke-line");
    const smokeToggleGridButton = document.getElementById("smoke-toggle-grid");
    const smokeDropCardButton = document.getElementById("smoke-drop-card");
    const cardEditorPanel = document.getElementById("card-editor");
    const cardEditorHeader = document.getElementById("card-editor-header");
    const cardEditorStatus = document.getElementById("card-editor-status");
    const cardEditorFront = document.getElementById("card-editor-front");
    const cardEditorBack = document.getElementById("card-editor-back");
    const cardEditorColor = document.getElementById("card-editor-color");
    const cardEditorSave = document.getElementById("card-editor-save");
    const cardEditorCancel = document.getElementById("card-editor-cancel");
    const mapEditorPanel = document.getElementById("map-editor");
    const mapEditorHeader = document.getElementById("map-editor-header");
    const mapEditorStatus = document.getElementById("map-editor-status");
    const mapEditorFile = document.getElementById("map-editor-file");
    const mapEditorPreview = document.getElementById("map-editor-preview");
    const mapEditorPreviewMode = document.getElementById("map-editor-preview-mode");
    const mapEditorPreviewFocus = document.getElementById("map-editor-preview-focus");
    const mapEditorPreviewSafe = document.getElementById("map-editor-preview-safe");
    const mapEditorDisplayMode = document.getElementById("map-editor-display-mode");
    const mapEditorFit = document.getElementById("map-editor-fit");
    const mapEditorScale = document.getElementById("map-editor-scale");
    const mapEditorCropX = document.getElementById("map-editor-crop-x");
    const mapEditorCropY = document.getElementById("map-editor-crop-y");
    const mapEditorSafeMargin = document.getElementById("map-editor-safe-margin");
    const mapEditorAssets = document.getElementById("map-editor-assets");
    const mapEditorSave = document.getElementById("map-editor-save");
    const mapEditorRemove = document.getElementById("map-editor-remove");
    const mapEditorCancel = document.getElementById("map-editor-cancel");
    const gridEditorPanel = document.getElementById("grid-editor");
    const gridEditorHeader = document.getElementById("grid-editor-header");
    const gridEditorStatus = document.getElementById("grid-editor-status");
    const gridEditorType = document.getElementById("grid-editor-type");
    const gridEditorHexOrientationField = document.getElementById("grid-editor-hex-orientation-field");
    const gridEditorHexOrientation = document.getElementById("grid-editor-hex-orientation");
    const gridEditorCellSize = document.getElementById("grid-editor-cell-size");
    const gridEditorCellSizeUp = document.getElementById("grid-editor-cell-size-up");
    const gridEditorCellSizeDown = document.getElementById("grid-editor-cell-size-down");
    const gridEditorOffsetX = document.getElementById("grid-editor-offset-x");
    const gridEditorOffsetXUp = document.getElementById("grid-editor-offset-x-up");
    const gridEditorOffsetXDown = document.getElementById("grid-editor-offset-x-down");
    const gridEditorOffsetY = document.getElementById("grid-editor-offset-y");
    const gridEditorOffsetYUp = document.getElementById("grid-editor-offset-y-up");
    const gridEditorOffsetYDown = document.getElementById("grid-editor-offset-y-down");
    const gridEditorOpacity = document.getElementById("grid-editor-opacity");
    const gridEditorLineWidth = document.getElementById("grid-editor-line-width");
    const gridEditorLineStyle = document.getElementById("grid-editor-line-style");
    const gridEditorReset = document.getElementById("grid-editor-reset");
    const gridEditorVisibility = document.getElementById("grid-editor-visibility");
    const gridEditorSave = document.getElementById("grid-editor-save");
    const gridEditorCancel = document.getElementById("grid-editor-cancel");
    let pixiLoadPromise = null;
    let firstTheaterRuntimeStarted = false;
    window.VictoryVenueShell?.mount?.({
      root: stageShell,
      venueSlug: "first-theater",
      venueName: "First Theater",
      slots: {
        header: stageStatus,
        leftTray: document.getElementById("overlay-root"),
        rightTray: document.getElementById("snapshot-summary"),
        chatRail: document.getElementById("renderer-fallback"),
        audioTray: document.getElementById("first-theater-audio-tray"),
      },
    });

    let ws = null;
    let currentSessionId = "";
    let currentActorId = "";
    let currentRole = "audience";
    let joinPromise = null;
    let currentSnapshot = null;
    let currentObjects = [];
    let currentSelection = null;
    let currentNodeMap = new Map();
    let dragState = null;
    let lastPointerReport = "Screen n/a | Stage n/a";
    let lastStagePoint = null;
    let contextMenuTarget = null;
    let suppressStageContextMenu = false;
    let stagePlacementCandidate = null;
    let pendingStageCardPlacement = null;
    let recentPlacementMarker = null;
    let cardEditorTargetKey = "";
    let cardEditorDirty = false;
    let cardEditorDragState = null;
    let cardFaceState = new Map();
    let localPositionOverrides = new Map();
    let currentVenueMapState = null;
    let currentVenueMapAssets = [];
    let currentVenueMapAssetID = "";
    let mapEditorDirty = false;
    let mapEditorDragState = null;
    let mapEditorPreviewURL = "";
    let venueMapTexture = null;
    let venueMapTextureURL = "";
    let venueMapTexturePromise = null;
    let venueMapTextureFailedURL = "";
    let pendingSocketActions = [];
    let socketReconnectTimer = null;
    let socketReconnectDelay = 1000;
    let worldRefreshSerial = 0;
    const smokeMode = new URLSearchParams(location.search).has("smoke");
    let smokeGridEnabled = smokeMode;
    let pixiApp = null;
    let sceneRoot = null;
    let backgroundLayer = null;
    let mapLayer = null;
    let gridLayer = null;
    let facadeLayer = null;
    let objectLayer = null;
    let uiLayer = null;
    let mapEditorOriginalState = null;
    let currentVenueGridConfig = null;
    let gridEditorOriginalState = null;
    let gridEditorDirty = false;
    let gridEditorDragState = null;
    let resizeObserver = null;
    let currentIdentity = null;
    let uiPreferenceKey = "";
    let uiPreferences = null;
    let presenceRefreshTimer = null;
    let unreadChatCount = 0;
    let lastPingMs = null;
    let currentPresenceUsers = [];
    let chatHoverOpen = false;
    let chatPinnedOpen = false;
    let chatDismissedUntilPointerLeavesRail = false;
    let lastChatPointerY = Number.NaN;
    let headerHoverOpen = false;
    let lastHeaderPointerY = Number.NaN;
    let drawerHoverOpen = { left: false, right: false };
    let drawerHoverCloseTimers = { left: null, right: null };
    const shellDefaults = {
      header: { pinned: false, opacity: 100 },
      chat: { opacity: 96 },
    };
    const drawerDefaults = {
      left: { mode: "hover", opacity: 96, portraitSize: "medium" },
      right: { mode: "hover", opacity: 96 },
    };
    const shellRoot = document.getElementById("shell-root");
    const topBar = document.getElementById("top-bar");
    const identityValue = document.getElementById("identity-value");
    const roleValue = document.getElementById("role-value");
    const characterValue = document.getElementById("character-value");
    const viewValue = document.getElementById("view-value");
    const pingValue = document.getElementById("ping-value");
    const targetValue = document.getElementById("target-value");
    const targetClassValue = document.getElementById("target-class-value");
    const pointerValue = document.getElementById("pointer-value");
    const targetInfoButton = document.getElementById("target-info");
    const headerPinButton = document.getElementById("header-pin-button");
    const headerSettingsButton = document.getElementById("header-settings-button");
    const headerSettingsPanel = document.getElementById("header-settings-panel");
    const headerOpacity = document.getElementById("header-opacity");
    const headerOpacityValue = document.getElementById("header-opacity-value");
    const connectionValue = document.getElementById("connection-value");
    const resetShellPrefsButton = document.getElementById("reset-shell-prefs");
    const chatOpacity = document.getElementById("chat-opacity");
    const chatOpacityValue = document.getElementById("chat-opacity-value");
    const leftDrawer = document.getElementById("left-drawer");
    const rightDrawer = document.getElementById("right-drawer");
    const leftSettingsButton = document.getElementById("left-settings-button");
    const rightSettingsButton = document.getElementById("right-settings-button");
    const leftSettingsPanel = document.getElementById("left-settings-panel");
    const rightSettingsPanel = document.getElementById("right-settings-panel");
    const leftOpenMode = document.getElementById("left-open-mode");
    const leftOpacity = document.getElementById("left-opacity");
    const leftOpacityValue = document.getElementById("left-opacity-value");
    const leftPortraitSize = document.getElementById("left-portrait-size");
    const rightOpenMode = document.getElementById("right-open-mode");
    const rightOpacity = document.getElementById("right-opacity");
    const rightOpacityValue = document.getElementById("right-opacity-value");
    const networkState = document.getElementById("network-state");
    const presencePreview = document.getElementById("presence-preview");
    const presenceState = document.getElementById("presence-state");
    const rightCharacterValue = document.getElementById("right-character-value");
    const rightCardEditorButton = document.getElementById("right-card-editor-button");
    const chatPanel = document.getElementById("chat-panel");
    const chatHead = document.getElementById("chat-head");
    const chatBadge = document.getElementById("chat-badge");
    const chatLog = document.getElementById("chat-log");
    const chatInput = document.getElementById("chat-input");
    const chatSend = document.getElementById("chat-send");
    const sessionStatus = document.getElementById("session-status");
    const shellRefreshWorldButton = document.getElementById("shell-refresh-world");
    const shellClearSelectionButton = document.getElementById("shell-clear-selection");
    const accountMenu = document.getElementById("account-menu");
    const accountMenuToggle = document.getElementById("account-menu-toggle");
    const accountLabel = document.getElementById("account-label");
    const accountLogoutButton = document.getElementById("account-logout-button");
    const clampNumber = window.VictoryVenueShell?.clampNumber || ((value, min, max, fallback) => {
      const num = Number(value);
      if (!Number.isFinite(num)) return fallback;
      return Math.min(max, Math.max(min, num));
    });

    function escapeHtml(value) {
      return String(value)
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll("\"", "&quot;")
        .replaceAll("'", "&#39;");
    }

    function normalizeRole(role) {
      const value = String(role || "audience").trim().toLowerCase();
      if (["producer", "director", "cast", "crew", "audience", "actor"].includes(value)) {
        return value;
      }
      return "audience";
    }

    function roleLabel(role) {
      switch (normalizeRole(role)) {
        case "producer":
          return "Producer";
        case "director":
          return "Director";
        case "cast":
        case "actor":
          return "Cast";
        case "crew":
          return "Crew";
        default:
          return "Audience";
      }
    }

    function canManageIndexCards(role) {
      const normalized = normalizeRole(role);
      return normalized === "producer" || normalized === "director";
    }

    function loadPixiLibrary() {
      if (window.PIXI) {
        return Promise.resolve(window.PIXI);
      }
      if (pixiLoadPromise) {
        return pixiLoadPromise;
      }

      pixiLoadPromise = new Promise((resolve) => {
        const existing = document.querySelector('script[data-victory-pixi-loader="first-theater"]');
        if (existing) {
          if (window.PIXI) {
            resolve(window.PIXI);
            return;
          }
          existing.addEventListener("load", () => resolve(window.PIXI || null), { once: true });
          existing.addEventListener("error", () => resolve(null), { once: true });
          window.setTimeout(() => resolve(window.PIXI || null), 8000);
          return;
        }

        const script = document.createElement("script");
        script.dataset.victoryPixiLoader = "first-theater";
        script.src = "/lib/pixi.min.js";
        script.async = true;
        script.onload = () => resolve(window.PIXI || null);
        script.onerror = () => resolve(null);
        document.head.appendChild(script);
        window.setTimeout(() => resolve(window.PIXI || null), 8000);
      });

      return pixiLoadPromise;
    }

    function formatShortId(value) {
      const text = String(value || "").trim();
      if (!text) return "n/a";
      return text.length > 10 ? `${text.slice(0, 10)}…` : text;
    }

    function formatStagePosition(position) {
      const x = Number(position?.x ?? 0);
      const y = Number(position?.y ?? 0);
      const order = Number(position?.order ?? 0);
      const frame = String(position?.frame || position?.anchor_frame || position?.coordinate_frame || "").trim().toLowerCase() || "legacy-centered";
      return `x ${Number.isFinite(x) ? x : 0}, y ${Number.isFinite(y) ? y : 0}, order ${Number.isFinite(order) ? order : 0}, frame ${frame}`;
    }

    function setStageStatus(text) {
      if (stageStatus) {
        stageStatus.hidden = false;
        stageStatus.textContent = text || "";
      }
    }

    function setMovementLine(text) {
      if (movementLine) movementLine.textContent = text || "";
    }

    function setPointerLine(text) {
      lastPointerReport = text || "Screen n/a | Stage n/a";
      if (pointerLine) pointerLine.textContent = lastPointerReport;
      updateShellTargetPresentation();
    }

    function setSmokeLine(text) {
      if (smokeLine) smokeLine.textContent = text || "";
    }

    function setLastStagePoint(point) {
      if (!point) return;
      lastStagePoint = {
        x: Math.round(Number(point.x || 0)),
        y: Math.round(Number(point.y || 0)),
      };
    }

    function currentStagePointLabel() {
      if (!lastStagePoint) return "n/a";
      return `${lastStagePoint.x}, ${lastStagePoint.y}`;
    }

    function currentSelectionSummary() {
      if (!currentSelection) return "none";
      const position = currentSelection.position || {};
      const frame = String(position.frame || position.anchor_frame || position.coordinate_frame || "").trim().toLowerCase() || "legacy-centered";
      const state = objectState(currentSelection);
      return `${currentSelection.label || "selection"} · ${currentSelection.kind || "object"} · ${frame} · ${Number(position.x ?? 0)}, ${Number(position.y ?? 0)} · locked ${state.locked ? "yes" : "no"} · nameplate ${state.nameplateVisible ? "visible" : "hidden"} · ${state.visible ? "visible" : "hidden"}`;
    }

    function updateSmokeUI() {
      if (smokePanel) {
        smokePanel.hidden = !smokeMode;
      }
      if (overlayAnchor) {
        overlayAnchor.hidden = !smokeMode;
      }
      if (overlayBadge) {
        overlayBadge.textContent = smokeMode ? "Smoke overlay" : "Smoke overlay";
      }
      if (smokeToggleGridButton) {
        smokeToggleGridButton.textContent = smokeGridEnabled ? "Grid: On" : "Grid: Off";
      }
      if (smokeMode && smokeLine && !smokeLine.textContent) {
        setSmokeLine("Smoke mode ready. Use buttons or g/d.");
      }
    }

    function updateConnectionPresentation() {
      if (!connectionValue) return;
      const socketOpen = ws && ws.readyState === WebSocket.OPEN;
      const socketConnecting = ws && ws.readyState === WebSocket.CONNECTING;
      connectionValue.textContent = socketOpen ? "Live" : (socketConnecting || currentSessionId ? "Connecting" : "Waiting");
    }

    function updateStageEmptyState() {
      if (!stageEmptyState) return;
      stageEmptyState.hidden = !smokeMode || currentObjects.length > 0;
    }

    function setSelectionLine(text) {
      if (selectionLine) selectionLine.textContent = text || "";
    }

    function setLiveFeedLine(text) {
      if (liveFeedLine) liveFeedLine.textContent = text || "";
    }

    function setSnapshotSummary(text) {
      if (snapshotSummary) snapshotSummary.textContent = text || "";
    }

    function setRendererFallback(visible, text = "") {
      if (rendererFallback) {
        rendererFallback.hidden = !visible;
      }
      if (rendererFallbackLine && text) {
        rendererFallbackLine.textContent = text;
      }
    }

    function setSessionLine(text) {
      if (sessionLine) sessionLine.textContent = text || "";
    }

    function setRoleLine(text) {
      if (roleLine) roleLine.textContent = text || "";
    }

    function setObjectCountLine(text) {
      if (objectCountLine) objectCountLine.textContent = text || "";
    }

    function updateStatusSummary() {
      setSessionLine(currentSessionId ? formatShortId(currentSessionId) : "Not joined");
      setRoleLine(roleLabel(currentRole));
      setObjectCountLine(String(currentObjects.length));
      setPointerLine(lastPointerReport);
      updateConnectionPresentation();
      if (networkState) {
        networkState.textContent = currentSessionId ? "Online" : "Idle";
      }
      updateSmokeUI();
      updateStageEmptyState();
    }

    function loadUiPreferencesForKey(key) {
      return window.VictoryVenueShell?.loadPreferences?.(key, {
        header: { ...shellDefaults.header },
        chat: { ...shellDefaults.chat },
        left: { ...drawerDefaults.left },
        right: { ...drawerDefaults.right },
      }) || {
        header: { ...shellDefaults.header },
        chat: { ...shellDefaults.chat },
        left: { ...drawerDefaults.left },
        right: { ...drawerDefaults.right },
      };
    }

    function saveUiPreferences() {
      if (!uiPreferenceKey) return;
      window.VictoryVenueShell?.savePreferences?.(uiPreferenceKey, uiPreferences);
    }

    function resetShellPreferences() {
      if (!uiPreferenceKey) return;
      const storageKey = window.VictoryVenueShell?.resolveStorageKey?.(uiPreferenceKey) || `victory.venue.ui.v1:${uiPreferenceKey}`;
      try {
        localStorage.removeItem(storageKey);
      } catch (error) {
        console.warn("failed to clear first theater shell prefs", error);
      }
      initializeShellChrome(currentIdentity);
    }

    function capitalizeLabel(value) {
      const text = String(value || "").trim();
      if (!text) return "Unknown Class";
      return `${text.charAt(0).toUpperCase()}${text.slice(1)}`;
    }

    function updateShellMetaPresentation() {
      const displayName = String(currentIdentity?.display_name || "").trim();
      const handle = String(currentIdentity?.handle || "").trim();
      const genericDisplayNames = new Set(["web user", "webuser", "browser user", "browser", "account", "user"]);
      const identityLabel = displayName && !genericDisplayNames.has(displayName.toLowerCase())
        ? displayName
        : (handle || "Unknown User");
      if (identityValue) {
        identityValue.textContent = identityLabel;
      }
      if (roleValue) {
        roleValue.textContent = roleLabel(currentRole);
      }
      if (characterValue) {
        characterValue.textContent = "No Character";
      }
      if (viewValue) {
        viewValue.textContent = "Self";
      }
      if (pingValue) {
        pingValue.textContent = Number.isFinite(lastPingMs) ? `${lastPingMs}ms` : "—";
      }
      if (rightCharacterValue) {
        rightCharacterValue.textContent = "No Character";
      }
      if (networkState) {
        networkState.textContent = currentSessionId ? "Online" : "Idle";
      }
      if (accountLabel) {
        accountLabel.textContent = identityLabel === "Unknown User" ? "Account" : identityLabel;
      }
      updateConnectionPresentation();
    }

    function updateShellTargetPresentation() {
      const targetLabel = currentSelection?.label || "No Target";
      const targetClass = currentSelection?.kind ? capitalizeLabel(currentSelection.kind) : "Unknown Class";
      if (targetValue) targetValue.textContent = targetLabel;
      if (targetClassValue) targetClassValue.textContent = targetClass;
      if (pointerValue) pointerValue.textContent = lastPointerReport || "Screen n/a | Stage n/a";
      if (targetInfoButton) {
        targetInfoButton.title = currentSelection ? currentSelectionSummary() : "No target selected";
      }
    }

    function isHeaderPinned() {
      return Boolean(uiPreferences?.header?.pinned);
    }

    function isHeaderDetailsVisible() {
      return Boolean((headerSettingsPanel && !headerSettingsPanel.hidden) || (accountMenu && !accountMenu.hidden));
    }

    function isDrawerDetailsVisible(side) {
      return side === "left"
        ? Boolean(leftSettingsPanel && !leftSettingsPanel.hidden)
        : Boolean(rightSettingsPanel && !rightSettingsPanel.hidden);
    }

    function syncHeaderHoverState(clientY = Number.NaN) {
      if (!topBar) return;
      if (Number.isFinite(clientY)) {
        lastHeaderPointerY = clientY;
      }

      const prefs = uiPreferences?.header || shellDefaults.header;
      if (prefs.pinned || isHeaderDetailsVisible() || topBar.contains(document.activeElement)) {
        if (!headerHoverOpen) {
          headerHoverOpen = true;
          updateHeaderPresentation();
        }
        return;
      }

      const pointerY = Number.isFinite(lastHeaderPointerY) ? lastHeaderPointerY : clientY;
      if (!Number.isFinite(pointerY)) return;

      const headerRect = topBar.getBoundingClientRect();
      const triggerLine = 24;
      const releaseLine = headerRect.bottom + 10;
      const nextOpen = headerHoverOpen ? pointerY <= releaseLine : pointerY <= triggerLine;

      if (nextOpen !== headerHoverOpen) {
        headerHoverOpen = nextOpen;
        updateHeaderPresentation();
      }
    }

    function setHeaderPinned(pinned) {
      uiPreferences.header = {
        ...(uiPreferences.header || shellDefaults.header),
        pinned: Boolean(pinned),
      };
      saveUiPreferences();
      updateHeaderPresentation();
    }

    function closeHeaderSettings() {
      if (!headerSettingsPanel || !headerSettingsButton) return;
      headerSettingsPanel.hidden = true;
      headerSettingsButton.setAttribute("aria-expanded", "false");
    }

    function toggleHeaderSettings() {
      if (!headerSettingsPanel || !headerSettingsButton) return;
      const open = headerSettingsPanel.hidden;
      headerSettingsPanel.hidden = !open;
      headerSettingsButton.setAttribute("aria-expanded", String(open));
    }

    function closeDrawerSettings() {
      if (!leftSettingsPanel || !rightSettingsPanel || !leftSettingsButton || !rightSettingsButton) return;
      leftSettingsPanel.hidden = true;
      rightSettingsPanel.hidden = true;
      leftSettingsButton.setAttribute("aria-expanded", "false");
      rightSettingsButton.setAttribute("aria-expanded", "false");
    }

    function toggleDrawerSettings(side) {
      const panel = side === "left" ? leftSettingsPanel : rightSettingsPanel;
      const button = side === "left" ? leftSettingsButton : rightSettingsButton;
      if (!panel || !button) return;
      const open = panel.hidden;
      panel.hidden = !open;
      button.setAttribute("aria-expanded", String(open));
      if (side === "left" && rightSettingsPanel) rightSettingsPanel.hidden = true;
      if (side === "right" && leftSettingsPanel) leftSettingsPanel.hidden = true;
    }

    function setDrawerSetting(side, key, value) {
      if (!uiPreferences) return;
      uiPreferences[side] = {
        ...(uiPreferences[side] || drawerDefaults[side]),
        [key]: value,
      };
      saveUiPreferences();
      applyDrawerState(side);
    }

    function setDrawerHoverState(side, open) {
      if (!uiPreferences) return;
      const prefs = uiPreferences?.[side] || drawerDefaults[side];
      if (prefs.mode === "always-open") {
        drawerHoverOpen[side] = true;
        if (drawerHoverCloseTimers[side]) {
          window.clearTimeout(drawerHoverCloseTimers[side]);
          drawerHoverCloseTimers[side] = null;
        }
        applyDrawerState(side);
        return;
      }
      if (prefs.mode === "always-closed") {
        drawerHoverOpen[side] = false;
        if (drawerHoverCloseTimers[side]) {
          window.clearTimeout(drawerHoverCloseTimers[side]);
          drawerHoverCloseTimers[side] = null;
        }
        applyDrawerState(side);
        return;
      }
      if (drawerHoverCloseTimers[side]) {
        window.clearTimeout(drawerHoverCloseTimers[side]);
        drawerHoverCloseTimers[side] = null;
      }
      if (open) {
        drawerHoverOpen[side] = true;
        applyDrawerState(side, true);
        return;
      }
      drawerHoverOpen[side] = false;
      applyDrawerState(side, false);
    }

    function applyDrawerState(side, hoverOverride = null) {
      const state = side === "left" ? leftDrawer : rightDrawer;
      if (!state) return;
      const prefs = uiPreferences?.[side] || drawerDefaults[side];
      const open = prefs.mode === "always-open"
        ? true
        : prefs.mode === "always-closed"
          ? false
          : hoverOverride === null
            ? (drawerHoverOpen[side] || state.contains(document.activeElement) || isDrawerDetailsVisible(side))
            : Boolean(hoverOverride);

      state.dataset.openMode = prefs.mode;
      state.dataset.open = open ? "true" : "false";
      state.classList.toggle("drawer--expanded", open);
      state.classList.toggle("drawer--collapsed", !open);
      state.style.setProperty("--drawer-surface-opacity", `${clampNumber(prefs.opacity, 0, 100, 96) / 100}`);
      const width = open
        ? (window.innerWidth <= 720
          ? "min(320px, calc(100vw - 24px))"
          : "min(24vw, 420px)")
        : "20px";
      if (side === "left") {
        state.style.setProperty("--left-drawer-width", width);
        if (leftOpenMode) leftOpenMode.value = prefs.mode;
        if (leftOpacity) leftOpacity.value = String(clampNumber(prefs.opacity, 0, 100, 96));
        if (leftOpacityValue) leftOpacityValue.textContent = `${clampNumber(prefs.opacity, 0, 100, 96)}%`;
        if (leftPortraitSize) leftPortraitSize.value = window.VictoryVenueShell?.normalizePortraitSize?.(prefs.portraitSize) || "medium";
        if (presencePreview) {
          const avatarSize = prefs.portraitSize === "large" ? "44px" : prefs.portraitSize === "small" ? "30px" : "36px";
          if (presenceState) {
            presenceState.textContent = window.VictoryVenueShell?.renderPresencePreview?.(presencePreview, currentPresenceUsers, { avatarSize }) || presenceState.textContent;
          }
        }
      } else {
        state.style.setProperty("--right-drawer-width", width);
        if (rightOpenMode) rightOpenMode.value = prefs.mode;
        if (rightOpacity) rightOpacity.value = String(clampNumber(prefs.opacity, 0, 100, 96));
        if (rightOpacityValue) rightOpacityValue.textContent = `${clampNumber(prefs.opacity, 0, 100, 96)}%`;
      }
    }

    function updateHeaderPresentation() {
      if (!topBar) return;
      const prefs = uiPreferences?.header || shellDefaults.header;
      const open = prefs.pinned
        || headerHoverOpen
        || topBar.matches(":hover")
        || topBar.contains(document.activeElement)
        || isHeaderDetailsVisible();
      topBar.dataset.open = open ? "true" : "false";
      topBar.dataset.pinned = prefs.pinned ? "true" : "false";
      topBar.style.setProperty("--header-surface-opacity", `${clampNumber(prefs.opacity, 0, 100, 100) / 100}`);
      if (headerPinButton) {
        headerPinButton.classList.toggle("is-active", Boolean(prefs.pinned));
        headerPinButton.setAttribute("aria-pressed", prefs.pinned ? "true" : "false");
        headerPinButton.title = prefs.pinned ? "Unpin header" : "Pin header open";
      }
      if (headerOpacity) headerOpacity.value = String(clampNumber(prefs.opacity, 0, 100, 100));
      if (headerOpacityValue) headerOpacityValue.textContent = `${clampNumber(prefs.opacity, 0, 100, 100)}%`;
      updateShellTargetPresentation();
      updateConnectionPresentation();
    }

    function isChatActive() {
      const active = document.activeElement;
      return Boolean(chatPanel && active && chatPanel.contains(active) && active !== chatHead);
    }

    function currentShowingIsOpen() {
      const showing = currentSnapshot?.showing || null;
      if (!showing) return false;
      return String(showing.status || "").trim().toLowerCase() !== "closed";
    }

    function currentSessionStatusLabel() {
      const showing = currentSnapshot?.showing || null;
      if (!showing || !currentSessionId) {
        return "Idle";
      }

      switch (String(showing.status || "").trim().toLowerCase()) {
        case "live":
          return "Active";
        case "rehearsal":
          return "Ready";
        case "closed":
          return "Closed";
        default:
          return "Idle";
      }
    }

    function chatClosedMessage() {
      return "Chat is closed until the showing opens.";
    }

    function chatOpenMessage() {
      return "Showing open. Chat is live.";
    }

    function appendSystemChatNotice(text) {
      appendChatEntry("System", text);
      unreadChatCount = 0;
      updateChatPresentation();
    }

    function updateChatPresentation() {
      if (!chatPanel) return;
      const prefs = uiPreferences?.chat || shellDefaults.chat;
      chatPanel.style.setProperty("--chat-surface-opacity", `${clampNumber(prefs.opacity, 0, 100, 96) / 100}`);
      if (chatOpacity) chatOpacity.value = String(clampNumber(prefs.opacity, 0, 100, 96));
      if (chatOpacityValue) chatOpacityValue.textContent = `${clampNumber(prefs.opacity, 0, 100, 96)}%`;
      const active = isChatActive();
      const open = chatPinnedOpen || chatHoverOpen || active;
      const chatOpen = currentShowingIsOpen();
      chatPanel.dataset.open = open ? "true" : "false";
      chatPanel.classList.toggle("is-open", open);
      chatPanel.classList.toggle("is-chat-open", chatOpen);
      chatPanel.classList.toggle("is-closed", !chatOpen);
      chatPanel.style.setProperty("--chat-panel-width", open ? "min(50vw, 720px)" : "clamp(140px, 18vw, 220px)");
      if (chatInput) {
        chatInput.placeholder = chatOpen ? "Start typing here..." : chatClosedMessage();
      }
      if (chatSend) {
        chatSend.disabled = false;
      }
      if (chatBadge) {
        chatBadge.hidden = open || unreadChatCount === 0;
        chatBadge.textContent = String(unreadChatCount);
      }
      if (sessionStatus) {
        sessionStatus.textContent = currentSessionStatusLabel();
      }
    }

    function setChatHoverOpen(open) {
      chatHoverOpen = Boolean(open);
      if (!chatHoverOpen && !chatPinnedOpen && !isChatActive()) {
        lastChatPointerY = Number.NaN;
      }
      updateChatPresentation();
    }

    function setChatPinnedOpen(open) {
      chatPinnedOpen = Boolean(open);
      if (!chatPinnedOpen && !isChatActive()) {
        chatHoverOpen = false;
        lastChatPointerY = Number.NaN;
      }
      updateChatPresentation();
    }

    function closeChatPanel() {
      if (chatPanel?.contains(document.activeElement)) {
        document.activeElement?.blur?.();
      }
      chatHead?.blur?.();
      chatPinnedOpen = false;
      chatHoverOpen = false;
      chatDismissedUntilPointerLeavesRail = true;
      lastChatPointerY = Number.NaN;
      updateChatPresentation();
    }

    function appendChatEntry(author, text) {
      if (!chatLog) return;
      const entry = document.createElement("div");
      entry.className = "chat-entry";
      const title = document.createElement("strong");
      title.textContent = author;
      const body = document.createElement("span");
      body.textContent = text;
      entry.append(title, body);
      chatLog.appendChild(entry);
      chatLog.scrollTop = chatLog.scrollHeight;
      unreadChatCount += 1;
      updateChatPresentation();
    }

    function appendChatActionLine(action) {
      const text = String(action?.payload?.text || action?.text || "").trim();
      if (!text) return;
      const author = String(action?.actor_display_name || action?.actor?.display_name || action?.actor_handle || "Unknown").trim() || "Unknown";
      const source = String(action?.payload?.source || action?.source || "").trim().toLowerCase();
      const edited = Boolean(action?.payload?.edited || action?.edited);
      const label = source === "discord" ? `${author} via Discord Bridge${edited ? " (edited)" : ""}` : author;
      appendChatEntry(label, text);
    }

    async function sendChatDraft() {
      if (!chatInput) return;
      const text = chatInput.value.trim();
      if (!text) return;

      try {
        const sessionResult = await window.VictoryMicChat?.sendSessionCommand?.("first-theater", text);
        if (sessionResult?.handled) {
          const message = String(sessionResult.message || "").replace(/\n+/g, " · ").trim();
          if (message) {
            appendSystemChatNotice(message);
          }
          chatInput.value = "";
          return;
        }
      } catch (error) {
        appendSystemChatNotice(String(error?.message || error || "Session command failed"));
        chatInput.value = "";
        return;
      }

      if (!currentShowingIsOpen()) {
        chatInput.value = "";
        appendSystemChatNotice(chatClosedMessage());
        return;
      }

      try {
        const micResult = await window.VictoryMicChat?.sendMicCommand?.("first-theater", text);
        if (micResult?.handled) {
          const message = String(micResult.message || "").replace(/\n+/g, " · ").trim();
          if (message) {
            appendSystemChatNotice(message);
          }
          chatInput.value = "";
          return;
        }
      } catch (error) {
        appendSystemChatNotice(String(error?.message || error || "Mic command failed"));
        chatInput.value = "";
        return;
      }

      const sent = sendAction("chat/message", { text });
      if (!sent) {
        appendSystemChatNotice("Chat could not be sent right now.");
        chatInput.value = "";
        return;
      }
      chatInput.value = "";
    }

    function syncChatOpenState() {
      updateChatPresentation();
    }

    function syncChatHoverState(clientX = Number.NaN, clientY = Number.NaN) {
      if (!chatPanel) return;
      if (Number.isFinite(clientY)) {
        lastChatPointerY = clientY;
      }

      if (chatPinnedOpen || isChatActive()) {
        chatDismissedUntilPointerLeavesRail = false;
        if (!chatHoverOpen) {
          chatHoverOpen = true;
          updateChatPresentation();
        }
        return;
      }

      const chatRect = chatPanel.getBoundingClientRect();
      const pointerY = Number.isFinite(lastChatPointerY) ? lastChatPointerY : clientY;
      const pointerX = Number.isFinite(clientX) ? clientX : Number.NaN;
      if (!Number.isFinite(pointerX) || !Number.isFinite(pointerY)) return;
      const insideRail = pointerX >= chatRect.left - 16 && pointerX <= chatRect.right + 16 && pointerY >= chatRect.top - 10 && pointerY <= chatRect.bottom + 10;
      const nearRail = pointerY >= window.innerHeight - 92;

      const nextOpen = insideRail || nearRail;

      if (chatDismissedUntilPointerLeavesRail) {
        if (nextOpen) {
          if (chatHoverOpen) {
            chatHoverOpen = false;
            updateChatPresentation();
          }
          return;
        }
        chatDismissedUntilPointerLeavesRail = false;
      }

      if (nextOpen !== chatHoverOpen) {
        chatHoverOpen = nextOpen;
        updateChatPresentation();
      }
    }

    async function refreshPresencePreview() {
      try {
        const response = await fetch("/api/director-console/current", { credentials: "include" });
        if (!response.ok) return;
        const payload = await response.json().catch(() => null);
        const presence = Array.isArray(payload?.data?.presence) ? payload.data.presence : [];
        const avatarSize = uiPreferences?.left?.portraitSize === "large"
          ? "44px"
          : uiPreferences?.left?.portraitSize === "small"
            ? "30px"
            : "36px";
        const stateText = window.VictoryVenueShell?.renderPresencePreview?.(presencePreview, presence, { avatarSize });
        if (presenceState) {
          presenceState.textContent = stateText || "Roster";
        }
        currentPresenceUsers = presence;
      } catch (error) {
        console.warn("failed to refresh first theater presence", error);
      }
    }

    function initializeShellChrome(identity) {
      currentIdentity = identity || null;
      uiPreferenceKey = String(identity?.user_id || identity?.handle || identity?.display_name || "unknown-user");
      uiPreferences = loadUiPreferencesForKey(uiPreferenceKey);
      uiPreferences.header = {
        ...shellDefaults.header,
        ...(uiPreferences.header || {}),
      };
      uiPreferences.chat = {
        ...shellDefaults.chat,
        ...(uiPreferences.chat || {}),
      };
      uiPreferences.left = {
        ...drawerDefaults.left,
        ...(uiPreferences.left || {}),
      };
      uiPreferences.right = {
        ...drawerDefaults.right,
        ...(uiPreferences.right || {}),
      };
      chatHoverOpen = true;
      chatPinnedOpen = false;
      chatDismissedUntilPointerLeavesRail = false;
      lastChatPointerY = Number.NaN;
      drawerHoverOpen = { left: true, right: true };
      if (drawerHoverCloseTimers.left) {
        window.clearTimeout(drawerHoverCloseTimers.left);
      }
      if (drawerHoverCloseTimers.right) {
        window.clearTimeout(drawerHoverCloseTimers.right);
      }
      drawerHoverCloseTimers = { left: null, right: null };
      updateShellMetaPresentation();
      updateShellTargetPresentation();
      updateHeaderPresentation();
      updateStatusSummary();
      applyDrawerState("left");
      applyDrawerState("right");
      updateChatPresentation();
      if (presenceRefreshTimer) {
        window.clearInterval(presenceRefreshTimer);
      }
      presenceRefreshTimer = window.setInterval(refreshPresencePreview, 15000);
      refreshPresencePreview();
    }

    function closeContextMenu() {
      if (contextMenu) {
        contextMenu.hidden = true;
        contextMenu.innerHTML = "";
      }
      contextMenuTarget = null;
      stagePlacementCandidate = null;
    }

    function setRecentPlacementMarker(point, label = "") {
      if (!point) {
        recentPlacementMarker = null;
        return;
      }
      recentPlacementMarker = {
        x: Math.round(Number(point.x || 0)),
        y: Math.round(Number(point.y || 0)),
        label: String(label || "").trim(),
        at: Date.now(),
      };
    }

    function venueConfigFlag(key, fallback = false) {
      const config = currentSnapshot?.venue?.config || {};
      const value = config[key];
      if (typeof value === "boolean") return value;
      if (typeof value === "string") {
        const lowered = value.trim().toLowerCase();
        if (["true", "1", "yes", "on"].includes(lowered)) return true;
        if (["false", "0", "no", "off"].includes(lowered)) return false;
      }
      return fallback;
    }

    function objectKind(model) {
      return String(model?.kind || model?.contextClass || model?.elementType || "").trim().toLowerCase();
    }

    function objectState(model) {
      const source = model?.source || {};
      const state = model?.state || source?.state || {};
      const visibility = model?.visibility || source?.visibility || {};
      return {
        locked: Boolean(state.locked ?? visibility.locked ?? false),
        nameplateVisible: Boolean(state.nameplate_visible ?? visibility.nameplate_visible ?? true),
        visible: Boolean(state.visible ?? visibility.visible ?? true),
      };
    }

    function updateObjectMatches(a, b) {
      if (!a || !b) return false;
      const aKey = String(a.key || "");
      const bKey = String(b.key || "");
      const aElementId = String(a.elementId || "");
      const bElementId = String(b.elementId || "");
      const aElementSlug = String(a.elementSlug || "");
      const bElementSlug = String(b.elementSlug || "");
      return (aKey && aKey === bKey) || (aElementId && aElementId === bElementId) || (aElementSlug && aElementSlug === bElementSlug);
    }

    function viewerCanSeeHiddenCards() {
      return normalizeRole(currentRole) !== "audience";
    }

    function cardStatusBadgeText(model) {
      const state = objectState(model);
      if (!state.nameplateVisible) {
        return "";
      }
      if (!state.visible) {
        return "HIDDEN";
      }
      if (state.locked) {
        return "LOCKED";
      }
      return "";
    }

    function updateLocalObjectModel(matchModel, mutator) {
      if (!matchModel || typeof mutator !== "function") return null;
      let updated = null;
      currentObjects = currentObjects.map((item) => {
        if (!updateObjectMatches(item, matchModel)) {
          return item;
        }

        mutator(item);
        if (item.source && item.source.data) {
          item.source.data.front_text = item.frontText ?? item.source.data.front_text;
          item.source.data.back_text = item.backText ?? item.source.data.back_text;
          item.source.data.color = item.color ?? item.source.data.color;
        }
        if (item.source) {
          item.source.name = item.label;
        }
        if (item.source && item.state) {
          item.source.state = { ...item.state };
        }
        if (item.source && item.visibility) {
          item.source.visibility = { ...item.visibility };
        }
        updated = item;
        return item;
      });

      if (!updated) return null;

      if (currentSelection && updateObjectMatches(currentSelection, matchModel)) {
        currentSelection = updated;
      }
      if (cardEditorTargetKey && updateObjectMatches({ key: cardEditorTargetKey }, matchModel)) {
        cardEditorTargetKey = updated.key;
      }

      updateStatusSummary();
      updateShellTargetPresentation();
      renderPixiScene();
      syncSelectedActions();
      syncCardEditorWithSelection();
      return updated;
    }

    function canActorRevealHideStageObjects() {
      const role = normalizeRole(currentRole);
      if (role === "producer" || role === "director") {
        return true;
      }
      return (role === "cast" || role === "actor") && venueConfigFlag("actors_can_reveal", false);
    }

    function isCardObject(model) {
      return objectKind(model) === "card";
    }

    function isLiveStageObject(model) {
      return !!model?.live && ["card", "prop", "fire"].includes(objectKind(model));
    }

    function canEditLiveCard(model) {
      return isCardObject(model) && model?.live && !objectState(model).locked && canManageIndexCards(currentRole);
    }

    function canMoveLiveStageObject(model) {
      return isLiveStageObject(model) && !objectState(model).locked && canManageIndexCards(currentRole);
    }

    function canDuplicateLiveStageObject(model) {
      return isLiveStageObject(model) && !objectState(model).locked && canManageIndexCards(currentRole);
    }

    function canRemoveLiveStageObject(model) {
      return isLiveStageObject(model) && !objectState(model).locked && canManageIndexCards(currentRole);
    }

    function canDeleteLiveCard(model) {
      return isCardObject(model) && model?.live && !objectState(model).locked && canManageIndexCards(currentRole);
    }

    function cardFaceForModel(model) {
      const cardKey = String(model?.elementId || model?.elementSlug || model?.key || "");
      return String(cardFaceState.get(cardKey) || "front").toLowerCase() === "back" ? "back" : "front";
    }

    function truncateCardText(text, maxChars = 24) {
      const value = String(text || "").replace(/\s+/g, " ").trim();
      if (!value) return "";
      if (value.length <= maxChars) return value;
      return `${value.slice(0, Math.max(1, maxChars - 1)).trimEnd()}…`;
    }

    function canToggleLiveVisibility(model) {
      return isLiveStageObject(model) && !objectState(model).locked && canActorRevealHideStageObjects();
    }

    function canToggleNameplate(model) {
      return isLiveStageObject(model) && !objectState(model).locked && canManageIndexCards(currentRole);
    }

    function canToggleLock(model) {
      return isLiveStageObject(model) && canManageIndexCards(currentRole);
    }

    function cardEditorDraftFromModel(model) {
      const source = model?.source || {};
      const data = source?.data || {};
      return {
        elementId: String(model?.elementId || source?.element_id || ""),
        elementSlug: String(model?.elementSlug || source?.slug || ""),
        label: String(model?.label || source?.name || data.front_text || "Index card"),
        frontText: String(model?.frontText || data.front_text || source?.name || "").trim(),
        backText: String(model?.backText || data.back_text || "").trim(),
        color: String(model?.color || data.color || "#d9c7a6").trim() || "#d9c7a6",
      };
    }

    function stageContextMenuModel() {
      return {
        key: "stage:canvas",
        kind: "stage",
        live: false,
        elementId: "",
        elementSlug: "",
        label: "Stage",
        position: { x: 0, y: 0, order: 0 },
      };
    }

    function resolveStageObjectActions(objectModel) {
      const kind = objectKind(objectModel);
      const state = objectState(objectModel);
      const actions = [];

      const push = (action, label, group, options = {}) => {
        actions.push({
          action,
          label,
          group,
          requiresPoint: Boolean(options.requiresPoint),
          disabled: Boolean(options.disabled),
        });
      };

      if (kind === "stage") {
        push("create-card", "Create Index Card", "create", { disabled: !canManageIndexCards(currentRole) });
        push("set-map", "Add / Replace Map", "create", { disabled: !canManageIndexCards(currentRole) });
        push("configure-grid", "Configure Grid", "create", { disabled: !canManageIndexCards(currentRole) });
        push("inspect", "Inspect Stage", "info");
        if (currentSelection) {
          push("clear", "Clear selection", "clear");
        }
        return actions;
      }

      if (kind === "fire") {
        push("select", "Select", "info");
        push("info", "Info", "info");
        push("create-card", "Create Index Card", "create", { disabled: !canManageIndexCards(currentRole) });
        if (canRemoveLiveStageObject(objectModel)) {
          push("remove", "Remove from Stage", "remove");
        }
        if (currentSelection) {
          push("clear", "Clear selection", "clear");
        }
        return actions;
      }

      if (!objectModel?.live) {
        push("select", "Select", "info");
        push("info", "Info", "info");
        if (currentSelection) {
          push("clear", "Clear selection", "clear");
        }
        return actions;
      }

      if (state.locked) {
        if (canToggleLock(objectModel)) {
          push("unlock", "Unlock", "lock");
        }
        push("info", "Info", "info");
        return actions;
      }

      push("info", "Info", "info");

      if (kind === "card" && canEditLiveCard(objectModel)) {
        push("inspect", "Inspect", "edit");
        push("edit", "Edit", "edit");
        push("flip", cardFaceForModel(objectModel) === "back" ? "Show Front" : "Flip", "face");
      }

      if (canMoveLiveStageObject(objectModel)) {
        push("move-here", "Move here", "move", { requiresPoint: true });
      }

      if (canToggleLiveVisibility(objectModel)) {
        push(state.visible ? "hide" : "show", state.visible ? "Hide Audience" : "Show Audience", "visibility");
      }

      if (kind === "card" && canToggleNameplate(objectModel)) {
        push(state.nameplateVisible ? "hide-nameplate" : "show-nameplate", state.nameplateVisible ? "Hide Nameplate" : "Show Nameplate", "visibility");
      }

      if (kind === "card" && canDuplicateLiveStageObject(objectModel)) {
        push("duplicate", "Duplicate on stage", "duplicate", { requiresPoint: true });
      }

      if ((kind === "card" || kind === "prop" || kind === "fire") && canRemoveLiveStageObject(objectModel)) {
        push("remove", "Remove from Stage", "remove");
      }

      if (kind === "card" && canDeleteLiveCard(objectModel)) {
        push("delete-card", "Delete Card", "remove");
      }

      if ((kind === "card" || kind === "fire") && canToggleLock(objectModel)) {
        push("lock", "Lock", "lock");
      }

      return actions;
    }

    function syncSelectedActions() {
      if (!selectedActions) return;
      selectedActions.innerHTML = "";
      if (!currentSelection) return;

      const actions = resolveStageObjectActions(currentSelection);
      const hasPoint = !!(stagePlacementCandidate || lastStagePoint);

      actions.forEach((item, index) => {
        if (index > 0 && actions[index - 1].group !== item.group) {
          const separator = document.createElement("div");
          separator.className = "action-separator";
          selectedActions.appendChild(separator);
        }

        const button = document.createElement("button");
        button.type = "button";
        button.textContent = item.label;

        const shouldDisable = item.requiresPoint && !hasPoint;
        button.disabled = shouldDisable || item.disabled;

        button.addEventListener("click", () => {
          if (shouldDisable) {
            setStageStatus("Move and duplicate actions need a pointer point on the stage.");
            return;
          }
          performStageObjectAction(item.action, currentSelection);
        });

        selectedActions.appendChild(button);
      });
    }

    function showCardEditorFor(model) {
      if (!cardEditorPanel || !cardEditorFront || !cardEditorBack || !cardEditorColor) return;
      if (!model || !model.live || !isCardObject(model) || objectState(model).locked) {
        cardEditorTargetKey = "";
        cardEditorDirty = false;
        cardEditorPanel.hidden = true;
        if (cardEditorStatus) {
          cardEditorStatus.textContent = "Select a live index card.";
        }
        return;
      }

      const draft = cardEditorDraftFromModel(model);
      cardEditorTargetKey = model.key;
      cardEditorDirty = false;
      if (cardEditorPanel.hidden) {
        setCardEditorPosition(24, 24);
      }
      cardEditorPanel.hidden = false;
      cardEditorFront.value = draft.frontText;
      cardEditorBack.value = draft.backText;
      cardEditorColor.value = draft.color || "#d9c7a6";
      if (cardEditorStatus) {
        cardEditorStatus.textContent = `${draft.label} ready to inspect.`;
      }
    }

    function hideCardEditor() {
      cardEditorTargetKey = "";
      cardEditorDirty = false;
      if (cardEditorPanel) {
        cardEditorPanel.hidden = true;
      }
      if (cardEditorStatus) {
        cardEditorStatus.textContent = "Select a live index card.";
      }
    }

    function syncCardEditorWithSelection() {
      if (!cardEditorPanel || cardEditorPanel.hidden) {
        return;
      }

      const target = getCardEditorTarget();
      if (!target || !target.live || !isCardObject(target) || objectState(target).locked) {
        hideCardEditor();
        return;
      }

      if (currentSelection && currentSelection.key !== target.key) {
        hideCardEditor();
        return;
      }

      if (cardEditorDirty) {
        return;
      }

      showCardEditorFor(target);
    }

    function mapEditorDraftFromState() {
      const state = currentVenueMapState || {};
      const liveDisplayMode = String(mapEditorDisplayMode?.value || "").trim();
      const liveFit = String(mapEditorFit?.value || "").trim();
      const liveScale = mapEditorScale?.value;
      const liveCropX = mapEditorCropX?.value;
      const liveCropY = mapEditorCropY?.value;
      const liveSafeMargin = mapEditorSafeMargin?.value;
      const rawDisplayMode = liveDisplayMode || String(state.display_mode || "theater");
      return {
        assetID: String(state.asset_id || currentVenueMapAssetID || ""),
        displayMode: rawDisplayMode === "fullscreen" ? "fullscreen" : "theater",
        fit: liveFit || String(state.fit || "cover"),
        cropX: clampNumber(liveCropX ?? state.crop_x ?? 0.5, 0, 1, 0.5),
        cropY: clampNumber(liveCropY ?? state.crop_y ?? 0.5, 0, 1, 0.5),
        scale: clampNumber(liveScale ?? state.scale ?? 1, 0.25, 4, 1),
        safeMargin: clampNumber(liveSafeMargin ?? state.safe_margin ?? 24, 0, 128, 24),
      };
    }

    function setMapEditorStatus(text) {
      if (mapEditorStatus) {
        mapEditorStatus.textContent = text || "";
      }
    }

    function clampMapEditorPosition(left, top) {
      const shellRect = stageShell?.getBoundingClientRect?.();
      const panelRect = mapEditorPanel?.getBoundingClientRect?.();
      const panelWidth = Number(panelRect?.width || 560);
      const panelHeight = Number(panelRect?.height || 520);
      const shellWidth = Number(shellRect?.width || 0);
      const shellHeight = Number(shellRect?.height || 0);
      const maxLeft = Math.max(8, shellWidth - panelWidth - 8);
      const maxTop = Math.max(8, shellHeight - panelHeight - 8);
      return {
        left: Math.max(8, Math.min(Math.round(left), maxLeft)),
        top: Math.max(8, Math.min(Math.round(top), maxTop)),
      };
    }

    function setMapEditorPosition(left, top) {
      if (!mapEditorPanel) return;
      const position = clampMapEditorPosition(left, top);
      mapEditorPanel.style.left = `${position.left}px`;
      mapEditorPanel.style.top = `${position.top}px`;
      mapEditorPanel.style.right = "auto";
      mapEditorPanel.style.bottom = "auto";
    }

    function clampGridEditorPosition(left, top) {
      const shellRect = stageShell?.getBoundingClientRect?.();
      const panelRect = gridEditorPanel?.getBoundingClientRect?.();
      const panelWidth = Number(panelRect?.width || 420);
      const panelHeight = Number(panelRect?.height || 480);
      const shellWidth = Number(shellRect?.width || 0);
      const shellHeight = Number(shellRect?.height || 0);
      const maxLeft = Math.max(8, shellWidth - panelWidth - 8);
      const maxTop = Math.max(8, shellHeight - panelHeight - 8);
      return {
        left: Math.max(8, Math.min(Math.round(left), maxLeft)),
        top: Math.max(8, Math.min(Math.round(top), maxTop)),
      };
    }

    function setGridEditorPosition(left, top) {
      if (!gridEditorPanel) return;
      const position = clampGridEditorPosition(left, top);
      gridEditorPanel.style.left = `${position.left}px`;
      gridEditorPanel.style.top = `${position.top}px`;
      gridEditorPanel.style.right = "auto";
      gridEditorPanel.style.bottom = "auto";
    }

    function syncMapEditorPreview(url) {
      if (!mapEditorPreview) return;
      if (mapEditorPreviewURL && mapEditorPreviewURL !== url && mapEditorPreviewURL.startsWith("blob:")) {
        try {
          URL.revokeObjectURL(mapEditorPreviewURL);
        } catch (error) {
          console.warn("map preview revoke failed", error);
        }
      }
      mapEditorPreviewURL = url || "";
      mapEditorPreview.src = mapEditorPreviewURL || currentVenueMapState?.asset?.content_url || "";
      const draft = mapEditorDraftFromState();
      const fitMode = draft.fit === "contain" ? "contain" : "cover";
      mapEditorPreview.style.objectFit = fitMode;
      mapEditorPreview.style.objectPosition = `${Math.round(draft.cropX * 100)}% ${Math.round(draft.cropY * 100)}%`;
      mapEditorPreview.style.transform = `scale(${draft.scale})`;
      if (mapEditorPreviewMode) {
        mapEditorPreviewMode.textContent = fitMode;
      }
      if (mapEditorPreviewFocus) {
        mapEditorPreviewFocus.style.setProperty("--map-crop-x", `${Math.round(draft.cropX * 100)}%`);
        mapEditorPreviewFocus.style.setProperty("--map-crop-y", `${Math.round(draft.cropY * 100)}%`);
      }
      if (mapEditorPreviewSafe) {
        mapEditorPreviewSafe.style.setProperty("--map-safe-margin", `${draft.safeMargin}px`);
      }
    }

    function syncMapAssetList() {
      if (!mapEditorAssets || !mapEditorPanel || mapEditorPanel.hidden) return;
      mapEditorAssets.innerHTML = "";
      const draft = mapEditorDraftFromState();
      const selectedAssetID = String(currentVenueMapAssetID || draft.assetID || "");
      if (currentVenueMapAssets.length === 0) {
        const empty = document.createElement("div");
        empty.className = "editor-status";
        empty.textContent = "No uploaded map assets yet.";
        mapEditorAssets.appendChild(empty);
        return;
      }

      currentVenueMapAssets.forEach((asset) => {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "map-asset-button";
        if (asset.asset_id === selectedAssetID) {
          button.classList.add("is-selected");
        }
        button.innerHTML = `
          <strong>${escapeHtml(asset.original_filename || asset.asset_id || "Map asset")}</strong>
          <span>${escapeHtml(asset.asset_type || "map")} · ${escapeHtml(String(asset.width || 0))}x${escapeHtml(String(asset.height || 0))}</span>
        `;
        button.addEventListener("click", () => {
          if (mapEditorFile) {
            mapEditorFile.value = "";
          }
          currentVenueMapAssetID = asset.asset_id;
          mapEditorDirty = true;
          currentVenueMapState = {
            ...(currentVenueMapState || {}),
            asset_id: asset.asset_id,
            asset: asset,
          };
          syncMapEditorPreview(asset.content_url);
          syncMapAssetList();
          if (mapEditorStatus) {
            mapEditorStatus.textContent = `Selected ${asset.original_filename || asset.asset_id}.`;
          }
        });
        mapEditorAssets.appendChild(button);
      });
    }

    function syncMapEditorWithState() {
      if (!mapEditorPanel || mapEditorPanel.hidden) {
        return;
      }
      const state = currentVenueMapState || {};
      if (mapEditorFile) {
        mapEditorFile.value = "";
      }
      // Read selects directly from state — mapEditorDraftFromState() reads the
      // live select value first, which is always the HTML default ("theater", "cover")
      // on a fresh page load, masking the saved state value.
      if (mapEditorDisplayMode) mapEditorDisplayMode.value = String(state.display_mode || "theater");
      if (mapEditorFit) mapEditorFit.value = String(state.fit || "cover");
      if (mapEditorScale) mapEditorScale.value = String(state.scale ?? 1);
      if (mapEditorCropX) mapEditorCropX.value = String(state.crop_x ?? 0.5);
      if (mapEditorCropY) mapEditorCropY.value = String(state.crop_y ?? 0.5);
      if (mapEditorSafeMargin) mapEditorSafeMargin.value = String(state.safe_margin ?? 24);
      currentVenueMapAssetID = String(state.asset_id || "");
      syncMapAssetList();
      syncMapEditorPreview(state.asset?.content_url || "");
      if (mapEditorRemove) mapEditorRemove.disabled = !currentVenueMapAssetID;
    }

    function livePreviewMapOnStage() {
      if (!venueMapTexture || !pixiApp) return;
      const draft = mapEditorDraftFromState();
      currentVenueMapState = {
        ...(currentVenueMapState || {}),
        display_mode: draft.displayMode,
        fit: draft.fit,
        scale: draft.scale,
        crop_x: draft.cropX,
        crop_y: draft.cropY,
        safe_margin: draft.safeMargin,
      };
      const size = getStageSize();
      renderVenueMapLayer(Math.max(320, size.width), Math.max(320, size.height));
    }

    function openMapEditor() {
      if (!mapEditorPanel) return;
      if (!canManageIndexCards(currentRole)) {
        setStageStatus("Only producers and directors can add or replace the First Theater map.");
        return;
      }
      hideCardEditor();
      hideGridEditor();
      mapEditorOriginalState = currentVenueMapState ? { ...currentVenueMapState } : null;
      if (mapEditorPanel.hidden) {
        setMapEditorPosition(24, 24);
      }
      mapEditorPanel.hidden = false;
      mapEditorDirty = false;
      setMapEditorStatus("Choose a map asset for the First Theater stage.");
      void refreshVenueMapAssets();
      syncMapEditorWithState();
      mapEditorFile?.focus?.({ preventScroll: true });
    }

    function hideMapEditor() {
      mapEditorDirty = false;
      mapEditorOriginalState = null;
      if (mapEditorPanel) {
        mapEditorPanel.hidden = true;
      }
      if (mapEditorPreviewURL) {
        if (mapEditorPreviewURL.startsWith("blob:")) {
          try {
            URL.revokeObjectURL(mapEditorPreviewURL);
          } catch (error) {
            console.warn("map preview revoke failed", error);
          }
        }
        mapEditorPreviewURL = "";
      }
      setMapEditorStatus("Choose a map asset for the First Theater stage.");
    }

    function cancelMapEditor() {
      if (mapEditorOriginalState) {
        currentVenueMapState = mapEditorOriginalState;
        if (mapEditorDisplayMode) mapEditorDisplayMode.value = String(currentVenueMapState.display_mode || "theater");
        if (mapEditorFit) mapEditorFit.value = String(currentVenueMapState.fit || "cover");
        if (mapEditorScale) mapEditorScale.value = String(currentVenueMapState.scale ?? 1);
        if (mapEditorCropX) mapEditorCropX.value = String(currentVenueMapState.crop_x ?? 0.5);
        if (mapEditorCropY) mapEditorCropY.value = String(currentVenueMapState.crop_y ?? 0.5);
        if (mapEditorSafeMargin) mapEditorSafeMargin.value = String(currentVenueMapState.safe_margin ?? 24);
        renderPixiScene();
      }
      hideMapEditor();
    }

    function mapEditorPayloadFromUI(assetID) {
      const draft = mapEditorDraftFromState();
      return {
        asset_id: String(assetID || currentVenueMapAssetID || currentVenueMapState?.asset_id || "").trim(),
        display_mode: draft.displayMode,
        fit: String(mapEditorFit?.value || currentVenueMapState?.fit || "cover"),
        crop_x: clampNumber(mapEditorCropX?.value ?? currentVenueMapState?.crop_x ?? 0.5, 0, 1, 0.5),
        crop_y: clampNumber(mapEditorCropY?.value ?? currentVenueMapState?.crop_y ?? 0.5, 0, 1, 0.5),
        scale: clampNumber(mapEditorScale?.value ?? currentVenueMapState?.scale ?? 1, 0.25, 4, 1),
        safe_margin: Math.round(clampNumber(mapEditorSafeMargin?.value ?? currentVenueMapState?.safe_margin ?? 24, 0, 128, 24)),
      };
    }

    async function refreshVenueMapAssets() {
      try {
        const response = await fetch("/api/workshop/assets?asset_type=map", {
          credentials: "include",
          cache: "no-store",
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok) {
          throw new Error(payload?.error || `HTTP ${response.status}`);
        }
        currentVenueMapAssets = Array.isArray(payload.data) ? payload.data : [];
        syncMapAssetList();
      } catch (error) {
        console.warn("refreshVenueMapAssets failed", error);
        currentVenueMapAssets = [];
        syncMapAssetList();
      }
    }

    async function refreshVenueMapState() {
      try {
        const response = await fetch("/api/venues/first-theater/map", {
          credentials: "include",
          cache: "no-store",
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok) {
          throw new Error(payload?.error || `HTTP ${response.status}`);
        }
        currentVenueMapState = payload.data || null;
        currentVenueMapAssetID = String(currentVenueMapState?.asset_id || "");
        if (currentVenueMapState?.asset?.content_url) {
          syncMapEditorPreview(currentVenueMapState.asset.content_url);
        }
        syncMapEditorWithState();
        renderPixiScene();
        void ensureVenueMapTexture().then(() => {
          renderPixiScene();
        });
      } catch (error) {
        console.warn("refreshVenueMapState failed", error);
        currentVenueMapState = null;
        currentVenueMapAssetID = "";
        renderPixiScene();
      }
    }

    async function uploadMapAsset(file) {
      if (!file) return null;
      const formData = new FormData();
      formData.append("file", file);
      formData.append("asset_type", "map");
      formData.append("tags", "map");
      const response = await fetch("/api/workshop/assets", {
        method: "POST",
        credentials: "include",
        body: formData,
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok) {
        throw new Error(payload?.error || `HTTP ${response.status}`);
      }
      return payload.data || null;
    }

    async function saveVenueMap() {
      if (!canManageIndexCards(currentRole)) {
        setStageStatus("Only producers and directors can add or replace the First Theater map.");
        return;
      }

      const selectedFile = mapEditorFile?.files?.[0] || null;
      let assetID = String(currentVenueMapAssetID || currentVenueMapState?.asset_id || "").trim();
      if (selectedFile) {
        setMapEditorStatus(`Uploading ${selectedFile.name}...`);
        const uploadResult = await uploadMapAsset(selectedFile);
        assetID = String(uploadResult?.asset_id || "").trim();
        if (!assetID) {
          throw new Error("upload_failed");
        }
        currentVenueMapAssetID = assetID;
        await refreshVenueMapAssets();
      }

      if (!assetID) {
        throw new Error("asset_id_required");
      }

      const payload = mapEditorPayloadFromUI(assetID);
      setMapEditorStatus("Saving map placement...");
      const response = await fetch("/api/venues/first-theater/map", {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });
      const result = await response.json().catch(() => null);
      if (!response.ok || !result?.ok) {
        throw new Error(result?.error || `HTTP ${response.status}`);
      }
      currentVenueMapState = result.data || null;
      currentVenueMapAssetID = String(currentVenueMapState?.asset_id || assetID);
      mapEditorOriginalState = currentVenueMapState ? { ...currentVenueMapState } : null;
      renderPixiScene();
      setStageStatus("First Theater map updated.");
      hideMapEditor();
      void ensureVenueMapTexture().then(() => {
        renderPixiScene();
      });
    }

    async function removeVenueMap() {
      if (!canManageIndexCards(currentRole)) {
        setStageStatus("Only producers and directors can add or replace the First Theater map.");
        return;
      }

      setMapEditorStatus("Removing map...");
      const response = await fetch("/api/venues/first-theater/map", {
        method: "DELETE",
        credentials: "include",
      });
      const result = await response.json().catch(() => null);
      if (!response.ok || !result?.ok) {
        throw new Error(result?.error || `HTTP ${response.status}`);
      }
      currentVenueMapState = null;
      currentVenueMapAssetID = "";
      mapEditorOriginalState = null;
      void ensureVenueMapTexture();
      renderPixiScene();
      setStageStatus("First Theater map removed.");
      hideMapEditor();
    }

    function defaultGridConfig() {
      return {
        grid_type: "none",
        hex_orientation: "flat-top",
        cell_size: 50,
        offset_x: 0,
        offset_y: 0,
        line_width: 1,
        opacity: 0.45,
        line_style: "neutral",
        visible: true,
      };
    }

    function gridEditorDraftFromUI() {
      const state = currentVenueGridConfig || defaultGridConfig();
      const liveGridType = String(gridEditorType?.value || "");
      const liveHexOrientation = String(gridEditorHexOrientation?.value || "");
      const liveLineStyle = String(gridEditorLineStyle?.value || "");
      return {
        grid_type: liveGridType === "square" || liveGridType === "hex" ? liveGridType : "none",
        hex_orientation: liveHexOrientation === "pointy-top" ? "pointy-top" : "flat-top",
        cell_size: clampNumber(gridEditorCellSize?.value, 8, 500, 50),
        offset_x: clampNumber(gridEditorOffsetX?.value, -2000, 2000, 0),
        offset_y: clampNumber(gridEditorOffsetY?.value, -2000, 2000, 0),
        line_width: clampNumber(gridEditorLineWidth?.value, 0.5, 8, 1),
        opacity: clampNumber(gridEditorOpacity?.value, 0, 1, 0.45),
        line_style: liveLineStyle === "light" || liveLineStyle === "dark" ? liveLineStyle : "neutral",
        visible: state.visible !== false,
      };
    }

    function setGridEditorStatus(text) {
      if (gridEditorStatus) {
        gridEditorStatus.textContent = text || "";
      }
    }

    function syncGridEditorHexFieldVisibility() {
      if (!gridEditorHexOrientationField) return;
      gridEditorHexOrientationField.style.display = String(gridEditorType?.value || "") === "hex" ? "" : "none";
    }

    function syncGridEditorVisibilityButton() {
      if (!gridEditorVisibility) return;
      const visible = currentVenueGridConfig ? currentVenueGridConfig.visible !== false : true;
      gridEditorVisibility.textContent = visible ? "Hide Grid" : "Show Grid";
    }

    function syncGridEditorWithState() {
      if (!gridEditorPanel || gridEditorPanel.hidden) return;
      const state = currentVenueGridConfig || defaultGridConfig();
      if (gridEditorType) gridEditorType.value = String(state.grid_type || "none");
      if (gridEditorHexOrientation) gridEditorHexOrientation.value = String(state.hex_orientation || "flat-top");
      if (gridEditorCellSize) gridEditorCellSize.value = String(state.cell_size ?? 50);
      if (gridEditorOffsetX) gridEditorOffsetX.value = String(state.offset_x ?? 0);
      if (gridEditorOffsetY) gridEditorOffsetY.value = String(state.offset_y ?? 0);
      if (gridEditorOpacity) gridEditorOpacity.value = String(state.opacity ?? 0.45);
      if (gridEditorLineWidth) gridEditorLineWidth.value = String(state.line_width ?? 1);
      if (gridEditorLineStyle) gridEditorLineStyle.value = String(state.line_style || "neutral");
      syncGridEditorHexFieldVisibility();
      syncGridEditorVisibilityButton();
    }

    function liveSyncGrid() {
      currentVenueGridConfig = gridEditorDraftFromUI();
      syncGridEditorHexFieldVisibility();
      const size = getStageSize();
      renderVenueGridLayer(Math.max(320, size.width), Math.max(320, size.height));
    }

    function nudgeGridNumberField(inputEl, delta, min, max) {
      if (!inputEl) return;
      const current = Number(inputEl.value) || 0;
      const next = Math.max(min, Math.min(max, current + delta));
      inputEl.value = String(next);
      gridEditorDirty = true;
      liveSyncGrid();
    }

    function openGridEditor() {
      if (!gridEditorPanel) return;
      if (!canManageIndexCards(currentRole)) {
        setStageStatus("Only producers and directors can configure the First Theater grid.");
        return;
      }
      hideCardEditor();
      hideMapEditor();
      gridEditorOriginalState = currentVenueGridConfig ? { ...currentVenueGridConfig } : defaultGridConfig();
      if (gridEditorPanel.hidden) {
        setGridEditorPosition(24, 24);
      }
      gridEditorPanel.hidden = false;
      gridEditorDirty = false;
      setGridEditorStatus("Choose a grid style for the First Theater stage.");
      syncGridEditorWithState();
    }

    function hideGridEditor() {
      gridEditorDirty = false;
      gridEditorOriginalState = null;
      if (gridEditorPanel) {
        gridEditorPanel.hidden = true;
      }
      setGridEditorStatus("Choose a grid style for the First Theater stage.");
    }

    function cancelGridEditor() {
      if (gridEditorOriginalState) {
        currentVenueGridConfig = gridEditorOriginalState;
        const size = getStageSize();
        renderVenueGridLayer(Math.max(320, size.width), Math.max(320, size.height));
      }
      hideGridEditor();
    }

    function resetGridEditorDraft() {
      if (!window.confirm("Reset grid fields to default values? Click Save Grid to persist.")) return;
      currentVenueGridConfig = defaultGridConfig();
      gridEditorDirty = true;
      syncGridEditorWithState();
      const size = getStageSize();
      renderVenueGridLayer(Math.max(320, size.width), Math.max(320, size.height));
      setGridEditorStatus("Grid reset to defaults. Save to persist.");
    }

    function toggleGridVisibilityDraft() {
      const draft = gridEditorDraftFromUI();
      draft.visible = !(currentVenueGridConfig ? currentVenueGridConfig.visible !== false : true);
      currentVenueGridConfig = draft;
      gridEditorDirty = true;
      syncGridEditorVisibilityButton();
      const size = getStageSize();
      renderVenueGridLayer(Math.max(320, size.width), Math.max(320, size.height));
      setGridEditorStatus(draft.visible ? "Grid will be shown after save." : "Grid will be hidden after save.");
    }

    async function saveGridConfig() {
      if (!canManageIndexCards(currentRole)) {
        setStageStatus("Only producers and directors can configure the First Theater grid.");
        return;
      }
      const payload = gridEditorDraftFromUI();
      setGridEditorStatus("Saving grid configuration...");
      const response = await fetch("/api/venues/first-theater/grid", {
        method: "PUT",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });
      const result = await response.json().catch(() => null);
      if (!response.ok || !result?.ok) {
        throw new Error(result?.error || `HTTP ${response.status}`);
      }
      currentVenueGridConfig = result.data || defaultGridConfig();
      gridEditorOriginalState = { ...currentVenueGridConfig };
      renderPixiScene();
      setStageStatus("First Theater grid updated.");
      hideGridEditor();
    }

    async function refreshVenueGridConfig() {
      try {
        const response = await fetch("/api/venues/first-theater/grid", {
          credentials: "include",
          cache: "no-store",
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok) {
          throw new Error(payload?.error || `HTTP ${response.status}`);
        }
        currentVenueGridConfig = payload.data || defaultGridConfig();
        renderPixiScene();
      } catch (error) {
        console.warn("refreshVenueGridConfig failed", error);
        currentVenueGridConfig = defaultGridConfig();
      }
    }

    async function ensureVenueMapTexture() {
      const assetURL = String(currentVenueMapState?.asset?.content_url || "").trim();
      if (!assetURL || !window.PIXI) {
        venueMapTexture = null;
        venueMapTextureURL = "";
        venueMapTexturePromise = null;
        return null;
      }
      if (assetURL === venueMapTextureURL && venueMapTexture) {
        return venueMapTexture;
      }
      if (venueMapTexturePromise && venueMapTexturePromise.assetURL === assetURL) {
        return venueMapTexturePromise;
      }

      venueMapTexturePromise = window.PIXI.Assets.load({ src: assetURL, loadParser: "loadTextures" })
        .then((texture) => {
          const resolvedTexture = texture?.texture || texture || null;
          venueMapTexturePromise = null;
          if (!resolvedTexture) {
            venueMapTextureFailedURL = assetURL;
            return null;
          }
          if (String(currentVenueMapState?.asset?.content_url || "").trim() === assetURL) {
            venueMapTexture = resolvedTexture;
            venueMapTextureURL = assetURL;
            venueMapTextureFailedURL = "";
          }
          return resolvedTexture;
        })
        .catch((error) => {
          console.warn("venue map texture load failed", error);
          venueMapTexturePromise = null;
          venueMapTextureFailedURL = assetURL;
          return null;
        });
      venueMapTexturePromise.assetURL = assetURL;
      return venueMapTexturePromise;
    }

    function computeStagePlayableBounds(width, height) {
      const displayMode = String(currentVenueMapState?.display_mode || "theater").trim() === "fullscreen" ? "fullscreen" : "theater";
      if (displayMode === "fullscreen") {
        return { x: 0, y: 0, width, height };
      }
      const prosceniumInset = Math.max(18, Math.round(width * 0.04));
      const upperDrapeHeight = Math.max(80, Math.round(height * 0.16));
      const floorBandTop = Math.max(Math.round(height * 0.72), height - 200);
      return {
        x: prosceniumInset,
        y: upperDrapeHeight,
        width: Math.max(1, width - prosceniumInset * 2),
        height: Math.max(1, floorBandTop - upperDrapeHeight),
      };
    }

    function renderVenueGridLayer(width, height) {
      if (!gridLayer || !window.PIXI) return;
      const bounds = computeStagePlayableBounds(width, height);
      window.VictoryPixiGrid?.render?.(gridLayer, currentVenueGridConfig, bounds);
    }

    function renderVenueMapLayer(width, height) {
      if (!mapLayer || !window.PIXI) return;
      mapLayer.removeChildren();
      mapLayer.mask = null;
      const state = currentVenueMapState;
      if (!state || !state.asset || !String(state.asset.content_url || "").trim()) {
        return;
      }

      if (!venueMapTexture || venueMapTextureURL !== state.asset.content_url) {
        if (venueMapTextureFailedURL === state.asset.content_url) {
          return;
        }
        ensureVenueMapTexture().then((texture) => {
          if (!texture) return;
          if (currentVenueMapState?.asset?.content_url === state.asset.content_url) {
            renderPixiScene();
          }
        });
        return;
      }

      const displayMode = String(state.display_mode || "theater").trim() === "fullscreen" ? "fullscreen" : "theater";
      const bounds = computeStagePlayableBounds(width, height);
      const boundsX = bounds.x;
      const boundsY = bounds.y;
      const boundsWidth = bounds.width;
      const boundsHeight = bounds.height;

      const sprite = new PIXI.Sprite(venueMapTexture);
      sprite.anchor.set(0.5);
      sprite.alpha = 0.96;
      sprite.eventMode = "none";
      sprite.interactive = false;
      mapLayer.addChild(sprite);

      const scale = Math.max(0.25, Math.min(4, Number(state.scale ?? 1) || 1));
      window.VictoryPixiStage?.fitSpriteToBounds?.(sprite, boundsWidth, boundsHeight, {
        fit: state.fit || "cover",
        x: boundsX,
        y: boundsY,
        cropX: Number(state.crop_x ?? 0.5),
        cropY: Number(state.crop_y ?? 0.5),
        scale,
      });

      if (displayMode === "theater") {
        const mapMask = new PIXI.Graphics();
        mapMask.beginFill(0xffffff);
        mapMask.drawRoundedRect(boundsX, boundsY, boundsWidth, boundsHeight, 12);
        mapMask.endFill();
        mapMask.renderable = false;
        mapLayer.addChild(mapMask);
        mapLayer.mask = mapMask;

        const border = new PIXI.Graphics();
        border.lineStyle(1, 0xf0d49b, 0.18);
        border.drawRoundedRect(boundsX, boundsY, boundsWidth, boundsHeight, 12);
        mapLayer.addChild(border);
      }
    }

    function currentLiveSelection() {
      if (!currentSelection) return null;
      return currentObjects.find((item) => item.key === currentSelection.key) || currentSelection;
    }

    function getCardEditorTarget() {
      if (!cardEditorTargetKey) return null;
      return currentObjects.find((item) => item.key === cardEditorTargetKey) || null;
    }

    function clampCardEditorPosition(left, top) {
      const shellRect = stageShell?.getBoundingClientRect?.();
      const panelRect = cardEditorPanel?.getBoundingClientRect?.();
      const panelWidth = Number(panelRect?.width || 420);
      const panelHeight = Number(panelRect?.height || 420);
      const shellWidth = Number(shellRect?.width || 0);
      const shellHeight = Number(shellRect?.height || 0);
      const maxLeft = Math.max(8, shellWidth - panelWidth - 8);
      const maxTop = Math.max(8, shellHeight - panelHeight - 8);
      return {
        left: Math.max(8, Math.min(Math.round(left), maxLeft)),
        top: Math.max(8, Math.min(Math.round(top), maxTop)),
      };
    }

    function setCardEditorPosition(left, top) {
      if (!cardEditorPanel) return;
      const position = clampCardEditorPosition(left, top);
      cardEditorPanel.style.left = `${position.left}px`;
      cardEditorPanel.style.top = `${position.top}px`;
      cardEditorPanel.style.right = "auto";
      cardEditorPanel.style.bottom = "auto";
    }

    function openCardEditor(model = null) {
      const target = model || currentLiveSelection();
      if (!target || !isCardObject(target) || !target.live) {
        setStageStatus("Select a live index card to edit it.");
        return;
      }
      if (objectState(target).locked) {
        setStageStatus("This card is locked.");
        return;
      }
      hideMapEditor();
      hideGridEditor();
      showCardEditorFor(target);
      selectObject(target, `${target.label} selected.`);
      if (cardEditorFront) {
        cardEditorFront.focus({ preventScroll: true });
      }
    }

    function saveCardEditor() {
      const target = getCardEditorTarget() || currentLiveSelection();
      if (!target || !isCardObject(target) || !target.live) {
        setStageStatus("Select a live index card to save edits.");
        return;
      }
      if (objectState(target).locked) {
        setStageStatus("This card is locked.");
        return;
      }
      const frontText = String(cardEditorFront?.value || "").trim();
      const backText = String(cardEditorBack?.value || "").trim();
      const color = String(cardEditorColor?.value || "").trim() || target.color || "#d9c7a6";
      const sent = sendAction("update/index_card", {
        element_id: target.elementId || "",
        element_slug: target.elementSlug || "",
        front_text: frontText,
        back_text: backText,
        color,
      });
      if (!sent) {
        setStageStatus("Socket unavailable.");
        if (cardEditorStatus) {
          cardEditorStatus.textContent = "Save failed: socket unavailable.";
        }
        return;
      }
      updateLocalObjectModel(target, (model) => {
        model.frontText = frontText;
        model.backText = backText;
        model.color = color;
        model.label = frontText || model.label || "Index card";
        model.state = { ...(model.state || {}) };
        model.visibility = { ...(model.visibility || {}) };
      });
      cardEditorDirty = false;
      setStageStatus(`Saved ${target.label}.`);
      if (cardEditorStatus) {
        cardEditorStatus.textContent = `${target.label} save sent.`;
      }
      setMovementLine(`update/index_card sent for ${target.label}.`);
    }

    function toggleCardEditorFromSelection() {
      if (!currentSelection || !isCardObject(currentSelection) || !currentSelection.live) {
        return;
      }
      if (cardEditorPanel && cardEditorPanel.hidden) {
        openCardEditor(currentSelection);
      } else {
        hideCardEditor();
      }
    }

    function isSecondaryPointerEvent(event) {
      const button = Number(event?.button ?? event?.data?.button ?? -1);
      const buttons = Number(event?.buttons ?? event?.data?.buttons ?? 0);
      return button === 2 || (buttons & 2) === 2;
    }

    function markContextMenuHandled(event) {
      if (!event) return;
      event.__firstTheaterContextMenuHandled = true;
      if (event.data?.originalEvent) {
        event.data.originalEvent.__firstTheaterContextMenuHandled = true;
      }
    }

    function wasContextMenuHandled(event) {
      return !!(event?.__firstTheaterContextMenuHandled || event?.data?.originalEvent?.__firstTheaterContextMenuHandled);
    }

    function stagePlacementFromEvent(event) {
      const clientPoint = eventClientPoint(event);
      return stagePointFromClient(clientPoint.clientX, clientPoint.clientY);
    }

    function stagePointFromClient(clientX, clientY) {
      const rect = stageHost?.getBoundingClientRect();
      const point = {
        x: rect ? Number(clientX || 0) - rect.left : Number(clientX || 0),
        y: rect ? Number(clientY || 0) - rect.top : Number(clientY || 0),
      };
      const size = getStageSize();

      return {
        x: Math.max(0, Math.min(Math.round(size.width), Math.round(point.x))),
        y: Math.max(0, Math.min(Math.round(size.height), Math.round(point.y))),
      };
    }

    function hitTestContextMenuTarget(stagePoint) {
      for (let index = currentObjects.length - 1; index >= 0; index -= 1) {
        const model = currentObjects[index];
        const node = currentNodeMap.get(model.key);
        const bounds = node?.container?.getBounds?.();
        if (bounds?.contains?.(stagePoint.x, stagePoint.y)) {
          return model;
        }

        const localPoint = node?.container?.toLocal && window.PIXI
          ? node.container.toLocal(new PIXI.Point(stagePoint.x, stagePoint.y))
          : null;
        if (!localPoint) {
          continue;
        }

        const kind = objectKind(model);
        const width = kind === "fire" ? 124 : 200;
        const height = kind === "fire" ? 150 : 140;
        const hitSlop = 8;
        if (
          localPoint.x >= -hitSlop &&
          localPoint.x <= width + hitSlop &&
          localPoint.y >= -hitSlop &&
          localPoint.y <= height + hitSlop
        ) {
          return model;
        }
      }
      return null;
    }

    function resolveContextMenuTargetFromClient(clientX, clientY) {
      const stagePoint = stagePointFromClient(clientX, clientY);
      const objectModel = hitTestContextMenuTarget(stagePoint) || stageContextMenuModel();
      return { objectModel, stagePoint };
    }

    function eventClientPoint(event) {
      if (!event) {
        return { clientX: 0, clientY: 0 };
      }

      const originalClientX = Number(event.data?.originalEvent?.clientX ?? event.originalEvent?.clientX ?? Number.NaN);
      const originalClientY = Number(event.data?.originalEvent?.clientY ?? event.originalEvent?.clientY ?? Number.NaN);
      if (Number.isFinite(originalClientX) && Number.isFinite(originalClientY)) {
        return { clientX: originalClientX, clientY: originalClientY };
      }

      const clientX = Number(event.clientX ?? event.x ?? 0);
      const clientY = Number(event.clientY ?? event.y ?? 0);
      if (Number.isFinite(clientX) && Number.isFinite(clientY) && (clientX !== 0 || clientY !== 0)) {
        return { clientX, clientY };
      }

      const stagePoint = event.data?.global || event.global || null;
      if (stagePoint && pixiApp?.renderer) {
        const rect = stageHost?.getBoundingClientRect();
        const screenWidth = Number(pixiApp.renderer.screen?.width || rect?.width || 1);
        const screenHeight = Number(pixiApp.renderer.screen?.height || rect?.height || 1);
        const domX = rect ? rect.left + (Number(stagePoint.x || 0) * (rect.width / screenWidth)) : Number(stagePoint.x || 0);
        const domY = rect ? rect.top + (Number(stagePoint.y || 0) * (rect.height / screenHeight)) : Number(stagePoint.y || 0);
        return { clientX: domX, clientY: domY };
      }

      return { clientX: 0, clientY: 0 };
    }

    function updatePointerReadout(screenX, screenY, stagePoint, note = "") {
      if (stagePoint) {
        setLastStagePoint(stagePoint);
      }
      const resolved = stagePoint || lastStagePoint || { x: 0, y: 0 };
      const screenLabel = `Screen ${Math.round(screenX || 0)}, ${Math.round(screenY || 0)}`;
      const stageLabel = `Stage ${resolved.x}, ${resolved.y}`;
      const text = `${screenLabel} | ${stageLabel}`;
      setPointerLine(text);
      if (smokeMode) {
        const prefix = note ? `${note} · ` : "";
        setSmokeLine(`${prefix}${stageLabel} · selection ${currentSelectionSummary()}`);
      }
    }

    function cancelContextMenuEvent(event) {
      event?.stopPropagation?.();
      event?.preventDefault?.();
      event?.stopImmediatePropagation?.();
      event?.data?.originalEvent?.preventDefault?.();
      event?.data?.originalEvent?.stopPropagation?.();
      event?.data?.originalEvent?.stopImmediatePropagation?.();
    }

    function handleNativeStageContextMenu(event) {
      if (!event) return;
      const target = event.target;
      const composedPath = typeof event.composedPath === "function" ? event.composedPath() : [];
      const isStageEvent = target === stageHost || target === stageShell || stageShell?.contains?.(target) || stageHost?.contains?.(target) || composedPath.includes(stageHost) || composedPath.includes(stageShell) || composedPath.includes(pixiApp?.view);
      if (!isStageEvent) return;
      if (event.type !== "contextmenu" && !isSecondaryPointerEvent(event)) {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      event.stopImmediatePropagation?.();
      const clientPoint = eventClientPoint(event);
      const resolved = resolveContextMenuTargetFromClient(clientPoint.clientX, clientPoint.clientY);
      const nativeEvent = event?.originalEvent || event?.data?.originalEvent || event;
      markContextMenuHandled(nativeEvent);
      openContextMenu(nativeEvent, resolved.objectModel);
    }

    function openResolvedContextMenu(event) {
      if (!event || !contextMenu || !pixiApp) return;
      const clientPoint = eventClientPoint(event);
      const resolved = resolveContextMenuTargetFromClient(clientPoint.clientX, clientPoint.clientY);
      const nativeEvent = event?.data?.originalEvent || event?.originalEvent || event;
      markContextMenuHandled(nativeEvent);
      openContextMenu(nativeEvent, resolved.objectModel);
    }

    function performStageObjectAction(action, objectModel) {
      if (!objectModel || !action) return;

      const kind = objectKind(objectModel);
      const state = objectState(objectModel);
      const targetPoint = stagePlacementCandidate || lastStagePoint || null;
      const point = targetPoint ? { x: Math.round(targetPoint.x), y: Math.round(targetPoint.y) } : null;

      if (action === "select") {
        selectObject(objectModel, `${objectModel.label} selected.`);
        closeContextMenu();
        return;
      }

      if (action === "info" || action === "inspect") {
        selectObject(objectModel, describeObject(objectModel));
        if (action === "inspect" && kind === "card" && objectModel.live) {
          showCardEditorFor(objectModel);
        }
        closeContextMenu();
        return;
      }

      if (action === "edit") {
        openCardEditor(objectModel);
        closeContextMenu();
        return;
      }

      if (action === "flip") {
        if (state.locked) {
          setStageStatus(`${objectModel.label} is locked.`);
          closeContextMenu();
          return;
        }
        const key = String(objectModel.elementId || objectModel.elementSlug || objectModel.key || "");
        const nextFace = cardFaceForModel(objectModel) === "back" ? "front" : "back";
        if (key) {
          cardFaceState.set(key, nextFace);
          renderPixiScene();
        }
        setStageStatus(`${objectModel.label} flipped to ${nextFace}.`);
        closeContextMenu();
        return;
      }

      if (action === "create-card") {
        createIndexCardFromMenu();
        return;
      }

      if (action === "set-map") {
        openMapEditor();
        closeContextMenu();
        return;
      }

      if (action === "configure-grid") {
        openGridEditor();
        closeContextMenu();
        return;
      }

      if (action === "clear") {
        selectObject(null, "Selection cleared.");
        hideCardEditor();
        hideMapEditor();
        hideGridEditor();
        closeContextMenu();
        return;
      }

      if (action === "unlock" || action === "lock") {
        const locked = action === "lock";
        const sent = sendAction("act/set_element_lock", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          locked,
        });
        if (sent) {
          updateLocalObjectModel(objectModel, (model) => {
            model.state = { ...(model.state || {}), locked };
            model.visibility = { ...(model.visibility || {}), locked };
          });
        }
        setStageStatus(sent ? `${locked ? "Locking" : "Unlocking"} ${objectModel.label}...` : "Socket unavailable.");
        if (!sent && cardEditorStatus) {
          cardEditorStatus.textContent = "Socket unavailable.";
        }
        closeContextMenu();
        return;
      }

      if (action === "hide-nameplate" || action === "show-nameplate") {
        const visible = action === "show-nameplate";
        const sent = sendAction("act/set_nameplate_visibility", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          layer: "audience",
          visible,
        });
        if (sent) {
          updateLocalObjectModel(objectModel, (model) => {
            model.state = { ...(model.state || {}), nameplateVisible: visible };
            model.visibility = { ...(model.visibility || {}), nameplate_visible: visible };
          });
        }
        setStageStatus(sent ? `${visible ? "Showing" : "Hiding"} nameplate for ${objectModel.label}...` : "Socket unavailable.");
        closeContextMenu();
        return;
      }

      if (action === "hide" || action === "show") {
        const type = action === "show" ? "act/reveal_element" : "act/hide_element";
        const sent = sendAction(type, {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          layer: "audience",
        });
        if (sent) {
          updateLocalObjectModel(objectModel, (model) => {
            model.state = { ...(model.state || {}), visible: action === "show" };
            model.visibility = { ...(model.visibility || {}), visible: action === "show" };
          });
        }
        setStageStatus(sent ? `${action === "show" ? "Showing" : "Hiding"} ${objectModel.label} for the audience...` : "Socket unavailable.");
        closeContextMenu();
        return;
      }

      if (action === "move-here") {
        if (!point) {
          setStageStatus("Move here needs a stage point.");
          return;
        }
        const sent = sendAction("act/place_element", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
          x: point.x,
          y: point.y,
          order: Number(objectModel.position?.order ?? 0),
        });
        if (sent) {
          updateLocalObjectModel(objectModel, (model) => {
            model.position = {
              ...(model.position || {}),
              anchor: "stage",
              x: point.x,
              y: point.y,
              z: 0,
              order: Number(objectModel.position?.order ?? 0),
              frame: "top-left",
            };
          });
        }
        setMovementLine(sent ? `Move sent to x ${point.x}, y ${point.y}.` : "Socket unavailable.");
        setStageStatus(sent ? `Moving ${objectModel.label} to x ${point.x}, y ${point.y}.` : "Socket unavailable.");
        closeContextMenu();
        return;
      }

      if (action === "duplicate") {
        if (!point) {
          setStageStatus("Duplicate on stage needs a stage point.");
          return;
        }
        const sent = sendAction("act/duplicate_element", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
          x: point.x,
          y: point.y,
          order: Number(objectModel.position?.order ?? 0),
        });
        setMovementLine(sent ? `Duplicate sent to x ${point.x}, y ${point.y}.` : "Socket unavailable.");
        setStageStatus(sent ? `Duplicating ${objectModel.label} to x ${point.x}, y ${point.y}.` : "Socket unavailable.");
        closeContextMenu();
        return;
      }

      if (action === "remove") {
        const sent = sendAction("act/remove_element", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
        });
        if (sent) {
          updateLocalObjectModel(objectModel, (model) => {
            model.state = { ...(model.state || {}), visible: false };
            model.visibility = { ...(model.visibility || {}), visible: false };
          });
        }
        setMovementReport(sent ? `Remove sent for ${objectModel.label}.` : "Socket unavailable.");
        setStageStatus(sent ? `Removing ${objectModel.label} from the stage...` : "Socket unavailable.");
        closeContextMenu();
        return;
      }

      if (action === "delete-card") {
        const sent = sendAction("delete/index_card", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
        });
        if (sent) {
          currentObjects = currentObjects.filter((item) => !updateObjectMatches(item, objectModel));
          if (currentSelection && updateObjectMatches(currentSelection, objectModel)) {
            currentSelection = null;
          }
          if (cardEditorTargetKey && updateObjectMatches({ key: cardEditorTargetKey }, objectModel)) {
            cardEditorTargetKey = "";
          }
          updateStatusSummary();
          updateShellTargetPresentation();
          renderPixiScene();
          syncSelectedActions();
          syncCardEditorWithSelection();
        }
        setMovementReport(sent ? `Delete sent for ${objectModel.label}.` : "Socket unavailable.");
        setStageStatus(sent ? `Deleting ${objectModel.label}...` : "Socket unavailable.");
        closeContextMenu();
        return;
      }

      if (action === "place") {
        if (kind === "card" && objectModel.live && point) {
          const sent = sendAction("act/place_element", {
            element_id: objectModel.elementId || "",
            element_slug: objectModel.elementSlug || "",
            venue_slug: "the-cave",
            layer: "stage",
            x: point.x,
            y: point.y,
            order: Number(objectModel.position?.order ?? 0),
          });
          setMovementLine(sent ? `Placement sent to x ${point.x}, y ${point.y}.` : "Socket unavailable.");
        }
        closeContextMenu();
      }
    }

    function openContextMenu(event, objectModel) {
      if (!contextMenu || !objectModel) return;
      contextMenuTarget = objectModel;
      const clientPoint = eventClientPoint(event);
      const stagePoint = stagePlacementFromEvent(event);
      if (objectModel.kind === "stage" || objectModel.kind === "fire" || (objectModel.live && isLiveStageObject(objectModel))) {
        stagePlacementCandidate = stagePoint;
        if (stagePlacementCandidate) {
          updatePointerReadout(clientPoint.clientX || 0, clientPoint.clientY || 0, stagePlacementCandidate, `${objectModel.kind === "fire" ? "Fire" : "Stage"} menu`);
        }
      }

      const items = resolveStageObjectActions(objectModel);
      contextMenu.innerHTML = items
        .map((item, index) => {
          let separator = "";
          if (index > 0 && items[index - 1].group !== item.group) {
            separator = '<div class="menu-separator"></div>';
          }
          const disabled = item.disabled ? " disabled" : "";
          return `${separator}<button type="button" data-menu-action="${escapeHtml(item.action)}"${disabled}>${escapeHtml(item.label)}</button>`;
        })
        .join("") +
        `<div class="menu-separator"></div>
        <button type="button" data-menu-action="copy" disabled>Copy</button>
        <button type="button" data-menu-action="paste" disabled>Paste</button>`;

      contextMenu.hidden = false;
      contextMenu.style.left = "0px";
      contextMenu.style.top = "0px";
      const rect = contextMenu.getBoundingClientRect();
      const maxX = Math.max(8, window.innerWidth - rect.width - 8);
      const maxY = Math.max(8, window.innerHeight - rect.height - 8);
      contextMenu.style.left = `${Math.max(8, Math.min(clientPoint.clientX || 0, maxX))}px`;
      contextMenu.style.top = `${Math.max(8, Math.min(clientPoint.clientY || 0, maxY))}px`;
      contextMenu.dataset.objectKey = objectModel.key;
    }

    function getStageSize() {
      const rect = stageShell?.getBoundingClientRect();
      if (!rect || rect.width <= 0 || rect.height <= 0) {
        return { width: 960, height: 640 };
      }
      return { width: rect.width, height: rect.height };
    }

    function toStagePoint(model, size, options = {}) {
      const position = model?.position || {};
      const frame = String(position.frame || position.anchor_frame || position.coordinate_frame || "").trim().toLowerCase();
      const normX = Number(position.x ?? 0);
      const normY = Number(position.y ?? 0);
      const order = Number(position.order ?? 0);
      const width = Number(size?.width || 960);
      const height = Number(size?.height || 640);
      if (frame === "top-left" || frame === "topleft") {
        return {
          x: Math.max(0, Math.min(width, Math.round(normX))),
          y: Math.max(0, Math.min(height, Math.round(normY))),
        };
      }
      const baseY = options.baseY ?? (model?.kind === "fire" ? 0.62 : 0.52);
      const xScale = width * 0.006;
      const yScale = height * 0.006;
      return {
        x: width * 0.5 + (normX * xScale),
        y: height * baseY + (normY * yScale) + (options.orderOffset ? order * options.orderOffset : 0),
      };
    }

    function describeObject(model) {
      const state = objectState(model);
      const lines = [
        `Type: ${model.kind || "object"}`,
        `Live: ${model.live ? "yes" : "no"}`,
        `ID: ${model.elementId || "(local)"}`,
        `Slug: ${model.elementSlug || "(local)"}`,
        `Position: ${formatStagePosition(model.position)}`,
        `Locked: ${state.locked ? "yes" : "no"}`,
        `Nameplate: ${state.nameplateVisible ? "visible" : "hidden"}`,
        `Visible: ${state.visible ? "yes" : "no"}`,
      ];

      if (model.kind === "card") {
        lines.push(`Front: ${model.frontText || "(blank)"}`);
        lines.push(`Back: ${model.backText || "(blank)"}`);
        lines.push(`Color: ${model.color || "#d9c7a6"}`);
      }

      return lines.join("\n");
    }

    function selectObject(objectModel, reason = "") {
      currentSelection = objectModel || null;
      if (currentSelection) {
        setSelectionLine(`${currentSelection.label} ${currentSelection.live ? "(live)" : "(local)"}`);
        setLiveFeedLine(describeObject(currentSelection));
        setStageStatus(reason || `${currentSelection.label} selected.`);
      } else {
        setSelectionLine("None");
        setLiveFeedLine("No live object selected.");
        setStageStatus(reason || "Selection cleared.");
      }

      updateShellTargetPresentation();
      syncSelectedActions();
      syncCardEditorWithSelection();

      for (const [key, node] of currentNodeMap.entries()) {
        node.updateSelected?.(currentSelection?.key === key);
      }
    }

    function setMovementReport(text) {
      setMovementLine(text || "Idle");
    }

    function objectFromSnapshotElement(element, index = 0) {
      const data = element?.data || {};
      const state = element?.state || {};
      const visibility = element?.visibility || {};
      return {
        key: `live:${String(element?.element_id || element?.slug || index)}`,
        kind: "card",
        live: true,
        elementId: String(element?.element_id || ""),
        elementSlug: String(element?.slug || ""),
        elementType: String(element?.element_type || ""),
        contextClass: String(element?.context_class || data.context_class || "card"),
        label: String(element?.name || data.front_text || element?.slug || "Index card"),
        frontText: String(data.front_text || element?.name || "").trim(),
        backText: String(data.back_text || "").trim(),
        color: String(data.color || "#d9c7a6").trim() || "#d9c7a6",
        position: element?.position || data.position || {},
        state,
        visibility,
        source: element,
      };
    }

    function fireObjectFromSnapshot(element) {
      return {
        key: `fire:${String(element?.element_id || element?.slug || "fire")}`,
        kind: "fire",
        live: true,
        elementId: String(element?.element_id || ""),
        elementSlug: String(element?.slug || "first-fire"),
        label: String(element?.name || "Fire"),
        position: element?.position || element?.data?.position || {},
        state: element?.state || {},
        visibility: element?.visibility || {},
        source: element,
      };
    }

    function buildObjects(snapshot) {
      const elements = Array.isArray(snapshot?.elements) ? snapshot.elements : [];
      const fireElement = elements.find((element) => element?.slug === "first-fire") || null;
      const stageCards = elements
        .filter((element) => String(element?.element_type || "").toLowerCase() === "index_card")
        .filter((element) => String(element?.surface || element?.data?.surface || "tray").toLowerCase() === "stage")
        .sort((a, b) => {
          const orderA = Number(a?.position?.order ?? a?.data?.position?.order ?? 0);
          const orderB = Number(b?.position?.order ?? b?.data?.position?.order ?? 0);
          if (orderA !== orderB) return orderA - orderB;
          const nameA = String(a?.name || a?.slug || "");
          const nameB = String(b?.name || b?.slug || "");
          return nameA.localeCompare(nameB);
        })
        .map((element, index) => objectFromSnapshotElement(element, index));

      const out = [];
      if (fireElement) {
        out.push(fireObjectFromSnapshot(fireElement));
      }
      if (stageCards.length > 0) {
        out.push(...stageCards);
      }

      return out;
    }

    function clearSceneNodes() {
      currentNodeMap = new Map();
      if (objectLayer) {
        objectLayer.removeChildren();
      }
      if (facadeLayer) {
        facadeLayer.removeChildren();
      }
    }

    function refreshNodeSelection() {
      for (const [key, node] of currentNodeMap.entries()) {
        node.updateSelected?.(currentSelection?.key === key);
      }
    }

    function makeCardNode(model) {
      const container = new PIXI.Container();
      container.sortableChildren = true;
      const state = objectState(model);
      const cardKey = String(model.elementId || model.elementSlug || model.key || "");
      const face = cardFaceForModel(model);
      const titleText = face === "back"
        ? truncateCardText(model.backText || "Back", 22) || "Back"
        : truncateCardText(model.frontText || model.label || "Untitled card", 22) || "Untitled card";

      const shadow = new PIXI.Graphics();
      shadow.beginFill(0x000000, 0.28);
      shadow.drawRoundedRect(8, 10, 192, 132, 16);
      shadow.endFill();

      const body = new PIXI.Graphics();
      body.beginFill(PIXI.utils.string2hex(model.color || "#d9c7a6"), 1);
      body.lineStyle(2, 0x5d4327, 0.52);
      body.drawRoundedRect(0, 0, 192, 132, 16);
      body.endFill();

      const titleStyle = new PIXI.TextStyle({
        fontFamily: "Arial",
        fontSize: 14,
        fontWeight: "700",
        fill: 0x2f2112,
        wordWrap: true,
        wordWrapWidth: 144,
      });

      const title = new PIXI.Text(titleText, titleStyle);
      title.position.set(18, 18);
      title.alpha = 0.98;
      title.eventMode = "static";
      title.cursor = "text";

      const faceButtonBg = new PIXI.Graphics();
      faceButtonBg.beginFill(0x000000, 0.18);
      faceButtonBg.drawRoundedRect(0, 0, 26, 26, 13);
      faceButtonBg.endFill();
      faceButtonBg.position.set(160, 104);
      faceButtonBg.alpha = (canEditLiveCard(model) || canToggleLock(model)) && state.visible ? 1 : 0.34;

      const faceButton = new PIXI.Text(face === "back" ? "↻" : "↺", new PIXI.TextStyle({
        fontFamily: "Arial",
        fontSize: 15,
        fontWeight: "700",
        fill: 0x5d4327,
      }));
      faceButton.position.set(168, 108);
      faceButton.eventMode = "static";
      faceButton.cursor = "pointer";
      faceButton.on("pointerdown", (event) => {
        event.stopPropagation();
        if (objectState(model).locked) {
          selectObject(model, `${model.label} is locked.`);
          return;
        }
        if (!canEditLiveCard(model) && !canManageIndexCards(currentRole)) {
          return;
        }
        cardFaceState.set(cardKey, face === "back" ? "front" : "back");
        renderPixiScene();
      });

      const statusBadgeTextValue = cardStatusBadgeText(model);
      const statusBadge = new PIXI.Container();
      const statusBadgeStyle = new PIXI.TextStyle({
        fontFamily: "Arial",
        fontSize: 10,
        fontWeight: "700",
        fill: 0xf0d49b,
      });
      const statusBadgeLabel = new PIXI.Text(statusBadgeTextValue, statusBadgeStyle);
      statusBadgeLabel.position.set(10, 3);
      const statusBadgeBg = new PIXI.Graphics();
      statusBadgeBg.beginFill(0x1a0f0f, 0.84);
      statusBadgeBg.lineStyle(1, 0xf0d49b, 0.22);
      const statusBadgeWidth = Math.max(54, Math.ceil(statusBadgeLabel.width + 20));
      statusBadgeBg.drawRoundedRect(0, 0, statusBadgeWidth, 18, 9);
      statusBadgeBg.endFill();
      statusBadge.addChild(statusBadgeBg, statusBadgeLabel);
      statusBadge.position.set(Math.max(8, 192 - statusBadgeWidth - 10), 10);
      statusBadge.visible = Boolean(statusBadgeTextValue);

      const focus = new PIXI.Graphics();
      focus.lineStyle(0, 0x000000, 0);
      focus.drawRoundedRect(0, 0, 200, 140, 18);

      container.addChild(shadow, body, title, faceButtonBg, faceButton, statusBadge, focus);
      container.eventMode = "static";
      container.cursor = "pointer";
      container.interactive = true;
      container.visible = state.visible || viewerCanSeeHiddenCards();
      title.visible = true;
      faceButtonBg.visible = true;
      faceButton.visible = true;

      const node = {
        model,
        container,
        updateSelected(selected) {
          focus.clear();
          if (selected) {
            focus.lineStyle(3, 0xf0d49b, 0.9);
            focus.drawRoundedRect(0, 0, 200, 140, 18);
            container.zIndex = 50;
          } else {
            focus.lineStyle(0, 0x000000, 0);
            focus.drawRoundedRect(0, 0, 200, 140, 18);
            container.zIndex = 10;
          }
        },
      };

      const openCardContextMenu = (event) => {
        if (wasContextMenuHandled(event)) {
          return;
        }
        cancelContextMenuEvent(event);
        openResolvedContextMenu(event);
      };

      container.on("pointerdown", (event) => {
        if (isSecondaryPointerEvent(event)) {
          return;
        }
        event.stopPropagation();
        selectObject(model, `${model.label} selected.`);

        if (model.live) {
          if (objectState(model).locked) {
            setStageStatus(`${model.label} is locked.`);
            setMovementReport("Locked objects cannot be dragged.");
            return;
          }
          const local = event.data.getLocalPosition(sceneRoot);
          dragState = {
            node: container,
            model,
            offsetX: local.x - container.x,
            offsetY: local.y - container.y,
          };
          setMovementReport("Dragging live object...");
          setStageStatus("Release to try act/place_element.");
        }
      });

      const inspectFromText = (event) => {
        event?.stopPropagation?.();
        event?.stopImmediatePropagation?.();
        selectObject(model, `${model.label} selected.`);
        openCardEditor(model);
      };

      title.on("click", inspectFromText);
      title.on("contextmenu", openCardContextMenu);
      faceButton.on("contextmenu", openCardContextMenu);
      container.on("contextmenu", openCardContextMenu);

      return node;
    }

    function makeFireNode(model) {
      const container = new PIXI.Container();
      const glow = new PIXI.Graphics();
      glow.beginFill(0xff8b36, 0.14);
      glow.drawCircle(62, 62, 78);
      glow.endFill();

      const halo = new PIXI.Graphics();
      halo.beginFill(0xffc16f, 0.22);
      halo.drawCircle(62, 62, 40);
      halo.endFill();

      const sprite = PIXI.Sprite.from("/assets/fire.jpg");
      sprite.anchor.set(0, 0);
      sprite.position.set(0, 0);
      sprite.width = 124;
      sprite.height = 124;

      const base = new PIXI.Graphics();
      base.beginFill(0x000000, 0.2);
      base.drawEllipse(62, 118, 54, 14);
      base.endFill();

      const labelStyle = new PIXI.TextStyle({
        fontFamily: "Arial",
        fontSize: 12,
        fontWeight: "700",
        fill: 0xf4dfb4,
      });
      const label = new PIXI.Text(model.label || "Fire", labelStyle);
      label.anchor.set(0, 0);
      label.position.set(0, 130);

      container.addChild(glow, halo, base, sprite, label);
      container.eventMode = "static";
      container.cursor = "pointer";
      container.interactive = true;

      const focus = new PIXI.Graphics();
      focus.lineStyle(0, 0x000000, 0);
      focus.drawRoundedRect(0, 0, 124, 150, 18);
      container.addChild(focus);

      const node = {
        model,
        container,
        updateSelected(selected) {
          focus.clear();
          if (selected) {
            focus.lineStyle(3, 0xf0d49b, 0.95);
            focus.drawRoundedRect(0, 0, 124, 150, 18);
          } else {
            focus.lineStyle(0, 0x000000, 0);
            focus.drawRoundedRect(0, 0, 124, 150, 18);
          }
        },
      };

      container.on("pointerdown", (event) => {
        if (isSecondaryPointerEvent(event)) {
          return;
        }
        event.stopPropagation();
        selectObject(model, "Fire selected.");
      });

      container.on("contextmenu", (event) => {
        if (wasContextMenuHandled(event)) {
          return;
        }
        cancelContextMenuEvent(event);
        openResolvedContextMenu(event);
      });

      return node;
    }

    function makePlaceholderNode(model) {
      const node = makeCardNode(model);
      return node;
    }

    function createIndexCardFromMenu() {
      if (!canManageIndexCards(currentRole)) {
        setStageStatus("Only producers and directors can create cards.");
        return;
      }

      if (!stagePlacementCandidate) {
        setStageStatus("No stage placement point available.");
        return;
      }

      pendingStageCardPlacement = stagePlacementCandidate;
      stagePlacementCandidate = null;
      const placementLabel = `x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}`;
      const queued = !!(ws && ws.readyState === WebSocket.CONNECTING && currentSessionId);

      const sent = sendAction("create/index_card", {
        front_text: "New Index Card",
        back_text: "",
        color: "#d9c7a6",
      });
      if (!sent) {
        pendingStageCardPlacement = null;
        setStageStatus("Socket unavailable.");
        closeContextMenu();
        return;
      }

      setStageStatus(queued ? `Index card queued. Placing at ${placementLabel} when the connection opens...` : `Index card created. Placing at ${placementLabel}...`);
      setMovementReport(queued ? `Queued placement at ${placementLabel}.` : `Awaiting placement at ${placementLabel}.`);
      if (smokeMode) {
        setSmokeLine(queued ? `Create queued for ${placementLabel}.` : `Create sent for ${placementLabel}.`);
      }
      closeContextMenu();
    }

    function placeCreatedIndexCard(action) {
      if (!pendingStageCardPlacement) return;

      const targetId = String(action?.target?.element_id || "");
      const targetSlug = String(action?.target?.element_slug || "");
      const snapshotElements = Array.isArray(currentSnapshot?.elements) ? currentSnapshot.elements : [];
      const createdCard = snapshotElements.find((el) =>
        el &&
        String(el.element_type || "").toLowerCase() === "index_card" &&
        (
          (targetId && String(el.element_id || "") === targetId) ||
          (targetSlug && String(el.slug || "") === targetSlug)
        )
      );

      const elementId = targetId || createdCard?.element_id || "";
      const elementSlug = targetSlug || createdCard?.slug || "";

      if (!elementId && !elementSlug) {
        return;
      }

      selectObject({
        key: `live:${elementId || elementSlug}`,
        kind: "card",
        live: true,
        elementId,
        elementSlug,
        label: createdCard?.name || createdCard?.data?.front_text || "Index card",
        frontText: String(createdCard?.data?.front_text || createdCard?.name || "").trim(),
        backText: String(createdCard?.data?.back_text || "").trim(),
        color: String(createdCard?.data?.color || "#d9c7a6").trim() || "#d9c7a6",
        position: {
          ...(createdCard?.position || createdCard?.data?.position || {}),
          x: pendingStageCardPlacement.x,
          y: pendingStageCardPlacement.y,
          frame: "top-left",
        },
      }, `Index card created at x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}.`);
      if (smokeMode) {
        setRecentPlacementMarker(pendingStageCardPlacement, createdCard?.name || createdCard?.data?.front_text || "Index card");
      } else {
        setRecentPlacementMarker(null);
      }

      const ok = sendAction("act/place_element", {
        element_id: elementId,
        element_slug: elementSlug,
        venue_slug: "the-cave",
        layer: "stage",
        x: pendingStageCardPlacement.x,
        y: pendingStageCardPlacement.y,
        order: Number(createdCard?.position?.order ?? 0),
      });

      if (!ok) {
        setStageStatus(`Card created but the placement could not be sent from x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}.`);
        if (smokeMode) {
          setSmokeLine(`Create succeeded, placement failed at ${pendingStageCardPlacement.x}, ${pendingStageCardPlacement.y}.`);
        }
        return;
      }

      setStageStatus(`Index card placed at x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}.`);
      setMovementReport(`Index card placed at x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}.`);
      if (smokeMode) {
        setSmokeLine(`Placed smoke card at ${pendingStageCardPlacement.x}, ${pendingStageCardPlacement.y}.`);
      }
      pendingStageCardPlacement = null;
    }

    function renderPixiScene() {
      if (!pixiApp || !sceneRoot || !backgroundLayer || !objectLayer) return;

      clearSceneNodes();

      const size = getStageSize();
      const width = Math.max(320, size.width);
      const height = Math.max(320, size.height);
      backgroundLayer.removeChildren();

      const atmosphere = new PIXI.Graphics();
      atmosphere.beginFill(0x050405, 1);
      atmosphere.drawRect(0, 0, width, height);
      atmosphere.endFill();
      backgroundLayer.addChild(atmosphere);

      const ambientGlow = new PIXI.Graphics();
      ambientGlow.beginFill(0xf0d49b, 0.08);
      ambientGlow.drawEllipse(width * 0.5, height * 0.48, width * 0.26, height * 0.16);
      ambientGlow.endFill();
      backgroundLayer.addChild(ambientGlow);

      const prosceniumInset = Math.max(18, Math.round(width * 0.04));
      const upperDrapeHeight = Math.max(80, Math.round(height * 0.16));
      const floorBandTop = Math.max(Math.round(height * 0.72), height - 200);
      const footlightY = Math.max(floorBandTop + 10, height - 38);

      const stageOpening = new PIXI.Graphics();
      stageOpening.beginFill(0x090709, 1);
      stageOpening.drawRect(prosceniumInset, upperDrapeHeight, width - (prosceniumInset * 2), height - upperDrapeHeight);
      stageOpening.endFill();
      backgroundLayer.addChild(stageOpening);

      const sceneDisplayMode = String(currentVenueMapState?.display_mode || "theater") === "fullscreen" ? "fullscreen" : "theater";

      if (sceneDisplayMode !== "fullscreen") {
        const topValance = new PIXI.Graphics();
        topValance.beginFill(0x5a1119, 1);
        topValance.drawRect(0, 0, width, upperDrapeHeight);
        topValance.endFill();
        facadeLayer.addChild(topValance);

        const topValanceSheen = new PIXI.Graphics();
        topValanceSheen.beginFill(0xf0d49b, 0.08);
        topValanceSheen.drawRect(0, 0, width, Math.max(24, Math.round(upperDrapeHeight * 0.42)));
        topValanceSheen.endFill();
        facadeLayer.addChild(topValanceSheen);

        const sideCurtains = new PIXI.Graphics();
        sideCurtains.beginFill(0x2b090d, 0.92);
        sideCurtains.drawRect(0, upperDrapeHeight, prosceniumInset, height - upperDrapeHeight);
        sideCurtains.drawRect(width - prosceniumInset, upperDrapeHeight, prosceniumInset, height - upperDrapeHeight);
        sideCurtains.endFill();
        facadeLayer.addChild(sideCurtains);

        const curtainFoldStyle = new PIXI.Graphics();
        const foldWidth = Math.max(64, Math.round(width / 12));
        for (let x = 0; x < width; x += foldWidth) {
          const foldAlpha = (Math.floor(x / foldWidth) % 2 === 0) ? 0.08 : 0.14;
          curtainFoldStyle.beginFill(0x22070a, foldAlpha);
          curtainFoldStyle.drawRect(x, 0, Math.max(16, Math.round(foldWidth * 0.42)), upperDrapeHeight);
          curtainFoldStyle.endFill();
        }
        facadeLayer.addChild(curtainFoldStyle);

        const prosceniumFrame = new PIXI.Graphics();
        prosceniumFrame.lineStyle(3, 0xf0d49b, 0.16);
        prosceniumFrame.drawRoundedRect(prosceniumInset, upperDrapeHeight, width - (prosceniumInset * 2), height - upperDrapeHeight - 2, 18);
        prosceniumFrame.lineStyle(1, 0xffffff, 0.06);
        prosceniumFrame.drawRoundedRect(prosceniumInset + 8, upperDrapeHeight + 8, width - ((prosceniumInset + 8) * 2), height - upperDrapeHeight - 18, 14);
        facadeLayer.addChild(prosceniumFrame);

        const stageFloor = new PIXI.Graphics();
        stageFloor.beginFill(0x0d0a0b, 1);
        stageFloor.drawRect(0, floorBandTop, width, height - floorBandTop);
        stageFloor.endFill();
        facadeLayer.addChild(stageFloor);

        const floorGlow = new PIXI.Graphics();
        floorGlow.beginFill(0x8fb7da, 0.06);
        floorGlow.drawEllipse(width * 0.5, floorBandTop + ((height - floorBandTop) * 0.12), width * 0.3, Math.max(24, (height - floorBandTop) * 0.2));
        floorGlow.endFill();
        facadeLayer.addChild(floorGlow);

        const footlights = new PIXI.Graphics();
        const footlightCount = Math.max(5, Math.min(14, Math.floor(width / 120)));
        for (let index = 0; index < footlightCount; index += 1) {
          const x = (width / (footlightCount + 1)) * (index + 1);
          footlights.beginFill(0xf3d79f, 0.75);
          footlights.drawCircle(x, footlightY, 2.4);
          footlights.endFill();
          footlights.beginFill(0xf3d79f, 0.11);
          footlights.drawCircle(x, footlightY, 9);
          footlights.endFill();
        }
        facadeLayer.addChild(footlights);

        const stageLip = new PIXI.Graphics();
        stageLip.beginFill(0x120b0b, 0.9);
        stageLip.drawRect(0, floorBandTop, width, height - floorBandTop);
        stageLip.endFill();
        facadeLayer.addChild(stageLip);
      }

      renderVenueMapLayer(width, height);
      renderVenueGridLayer(width, height);

      if (smokeGridEnabled) {
        const grid = new PIXI.Graphics();
        grid.lineStyle(1, 0xffffff, 0.06);
        const spacing = 100;
        for (let x = 0; x <= width; x += spacing) {
          grid.moveTo(x, 0);
          grid.lineTo(x, height);
        }
        for (let y = 0; y <= height; y += spacing) {
          grid.moveTo(0, y);
          grid.lineTo(width, y);
        }
        grid.lineStyle(2, 0xf0d49b, 0.16);
        grid.moveTo(0, 0);
        grid.lineTo(Math.min(width, 80), 0);
        grid.moveTo(0, 0);
        grid.lineTo(0, Math.min(height, 80));
        const origin = new PIXI.Graphics();
        origin.beginFill(0xf0d49b, 0.7);
        origin.drawCircle(0, 0, 4);
        origin.endFill();
        const originLabelStyle = new PIXI.TextStyle({
          fontFamily: "Arial",
          fontSize: 11,
          fill: 0xf0d49b,
          fontWeight: "700",
        });
        const originLabel = new PIXI.Text("(0,0)", originLabelStyle);
        originLabel.position.set(8, 6);
        backgroundLayer.addChild(grid, origin, originLabel);
      }

      if (recentPlacementMarker && Date.now() - recentPlacementMarker.at < 15000) {
        const marker = new PIXI.Graphics();
        marker.lineStyle(2, 0xf0d49b, 0.9);
        marker.drawCircle(0, 0, 14);
        marker.moveTo(-20, 0);
        marker.lineTo(20, 0);
        marker.moveTo(0, -20);
        marker.lineTo(0, 20);
        marker.beginFill(0xf0d49b, 0.12);
        marker.drawCircle(0, 0, 24);
        marker.endFill();
        marker.position.set(recentPlacementMarker.x, recentPlacementMarker.y);

        const markerLabelStyle = new PIXI.TextStyle({
          fontFamily: "Arial",
          fontSize: 11,
          fontWeight: "700",
          fill: 0xf0d49b,
        });
        const markerLabel = new PIXI.Text(recentPlacementMarker.label ? `${recentPlacementMarker.label} @ ${recentPlacementMarker.x}, ${recentPlacementMarker.y}` : `${recentPlacementMarker.x}, ${recentPlacementMarker.y}`, markerLabelStyle);
        markerLabel.position.set(recentPlacementMarker.x + 18, recentPlacementMarker.y - 28);
        backgroundLayer.addChild(marker, markerLabel);
      }

      let liveCount = 0;
      currentObjects.forEach((model, index) => {
        let node;
        if (model.kind === "fire") {
          node = makeFireNode(model);
        } else if (model.kind === "card" && model.live) {
          node = makeCardNode(model);
        } else {
          node = makePlaceholderNode(model);
        }

        const pos = toStagePoint(model, { width, height }, { baseY: model.kind === "fire" ? 0.6 : 0.5, orderOffset: 12 });
        node.container.position.set(pos.x, pos.y);
        node.container.zIndex = 10 + index;
        objectLayer.addChild(node.container);
        currentNodeMap.set(model.key, node);
        node.updateSelected(currentSelection?.key === model.key);

        if (model.live) {
          liveCount += 1;
        }
      });

      if (objectCountLine) {
        objectCountLine.textContent = String(currentObjects.length);
      }
      setLiveFeedLine(`${liveCount} live object${liveCount === 1 ? "" : "s"} in the Pixi scene.`);
      const renderedFire = currentObjects.some((object) => object.kind === "fire");
      const renderedCards = currentObjects.filter((object) => object.kind === "card").length;
      setSnapshotSummary(`${renderedFire ? "Fire" : "No fire"} and ${renderedCards} card object${renderedCards === 1 ? "" : "s"} are rendered from the live Cave snapshot.`);
      updateStatusSummary();
      refreshNodeSelection();
      updateStageEmptyState();
    }

    function layoutPixiScene() {
      if (!pixiApp) return;
      const size = getStageSize();
      pixiApp.renderer.resize(size.width, size.height);
      if (sceneRoot) {
        sceneRoot.hitArea = new PIXI.Rectangle(0, 0, size.width, size.height);
      }
      renderPixiScene();
    }

    async function initializePixi() {
      if (!window.PIXI) {
        setRendererFallback(true, "PixiJS failed to load. The stage stays readable in DOM-only mode.");
        setStageStatus("PixiJS failed to load.");
        setMovementReport("Pixi unavailable");
        return;
      }

      setRendererFallback(false);

      const size = getStageSize();
      pixiApp = new PIXI.Application({
        width: size.width,
        height: size.height,
        backgroundAlpha: 0,
        antialias: true,
        resolution: window.devicePixelRatio || 1,
        autoDensity: true,
      });

      stageHost.innerHTML = "";
      stageHost.appendChild(pixiApp.view);
      pixiApp.stage.sortableChildren = true;

      pixiApp.view.addEventListener("contextmenu", handleNativeStageContextMenu, true);
      stageHost.addEventListener("contextmenu", handleNativeStageContextMenu, true);

      sceneRoot = new PIXI.Container();
      sceneRoot.sortableChildren = true;
      sceneRoot.eventMode = "static";
      sceneRoot.hitArea = new PIXI.Rectangle(0, 0, size.width, size.height);
      sceneRoot.on("pointerdown", (event) => {
        if (event.target === sceneRoot) {
          selectObject(null, "Selection cleared.");
          closeContextMenu();
        }
      });
      sceneRoot.on("contextmenu", (event) => {
        if (wasContextMenuHandled(event)) {
          return;
        }
        handleNativeStageContextMenu(event);
      });
      backgroundLayer = new PIXI.Container();
      backgroundLayer.zIndex = 0;
      mapLayer = new PIXI.Container();
      mapLayer.zIndex = 5;
      gridLayer = new PIXI.Container();
      gridLayer.zIndex = 6;
      facadeLayer = new PIXI.Container();
      facadeLayer.zIndex = 8;
      objectLayer = new PIXI.Container();
      objectLayer.zIndex = 10;
      uiLayer = new PIXI.Container();
      uiLayer.zIndex = 20;
      pixiApp.stage.addChild(sceneRoot);
      sceneRoot.addChild(backgroundLayer, mapLayer, gridLayer, facadeLayer, objectLayer, uiLayer);

      stageHost.addEventListener("pointermove", (event) => {
        const point = stagePointFromClient(event.clientX, event.clientY);
        const stageX = Math.round(point.x);
        const stageY = Math.round(point.y);
        if (dragState) {
          dragState.lastStagePoint = { x: stageX, y: stageY };
        }
        updatePointerReadout(event.clientX || 0, event.clientY || 0, { x: stageX, y: stageY }, dragState ? "Dragging" : "Pointer");
        if (currentSelection) {
          syncSelectedActions();
        }
        if (!dragState || !dragState.node) return;
        dragState.node.position.set(point.x - dragState.offsetX, point.y - dragState.offsetY);
        const dropX = Math.round(point.x - dragState.offsetX);
        const dropY = Math.round(point.y - dragState.offsetY);
        setMovementReport(`Dragging to x ${dropX}, y ${dropY}`);
      });

      window.addEventListener("pointerup", finishDrag, true);
      window.addEventListener("pointercancel", finishDrag, true);
      cardEditorHeader?.addEventListener("pointerdown", beginCardEditorDrag);
      window.addEventListener("pointermove", moveCardEditorDrag, true);
      window.addEventListener("pointerup", endCardEditorDrag, true);
      window.addEventListener("pointercancel", endCardEditorDrag, true);
      mapEditorHeader?.addEventListener("pointerdown", beginMapEditorDrag);
      window.addEventListener("pointermove", moveMapEditorDrag, true);
      window.addEventListener("pointerup", endMapEditorDrag, true);
      window.addEventListener("pointercancel", endMapEditorDrag, true);
      gridEditorHeader?.addEventListener("pointerdown", beginGridEditorDrag);
      window.addEventListener("pointermove", moveGridEditorDrag, true);
      window.addEventListener("pointerup", endGridEditorDrag, true);
      window.addEventListener("pointercancel", endGridEditorDrag, true);

      resizeObserver = new ResizeObserver(() => layoutPixiScene());
      resizeObserver.observe(stageShell);
      window.addEventListener("resize", layoutPixiScene);
      layoutPixiScene();
      window.setTimeout(() => {
        void refreshVenueMapState();
        void refreshVenueGridConfig();
      }, 0);
    }

    function toStageCoordinates(x, y) {
      return {
        x: Math.round(Number(x || 0)),
        y: Math.round(Number(y || 0)),
      };
    }

    function finishDrag(event) {
      if (!dragState) return;

      const { model, node } = dragState;
      dragState = null;

      const stagePoint = toStageCoordinates(node.x, node.y);
      setMovementReport(`Dropped at x ${stagePoint.x}, y ${stagePoint.y}`);

      if (!model.live) {
        setStageStatus("Non-live object moved only in Pixi.");
        return;
      }

      const sent = sendAction("act/place_element", {
        element_id: model.elementId || "",
        element_slug: model.elementSlug || "",
        venue_slug: "first-theater",
        layer: "stage",
        x: stagePoint.x,
        y: stagePoint.y,
        order: Number(model.position?.order ?? 0),
      });

      const overrideKey = String(model.elementId || model.elementSlug || model.key || "");
      const overridePos = {
        ...(model.position || {}),
        x: stagePoint.x,
        y: stagePoint.y,
        order: Number(model.position?.order ?? 0),
        frame: "top-left",
      };
      if (overrideKey) {
        localPositionOverrides.set(overrideKey, overridePos);
      }
      updateLocalObjectModel(model, (m) => {
        m.position = overridePos;
      });

      if (sent) {
        setStageStatus("Move sent through act/place_element.");
        if (smokeMode) {
          setSmokeLine(`Drop sent at ${stagePoint.x}, ${stagePoint.y}.`);
        }
      } else {
        setStageStatus("Move could not be sent. Socket unavailable.");
        if (smokeMode) {
          setSmokeLine(`Drop failed at ${stagePoint.x}, ${stagePoint.y}.`);
        }
      }
    }

    function beginCardEditorDrag(event) {
      if (!cardEditorPanel || cardEditorPanel.hidden || !cardEditorHeader) return;
      if (event.button !== 0) return;
      event.preventDefault();
      event.stopPropagation();
      const rect = cardEditorPanel.getBoundingClientRect();
      cardEditorDragState = {
        offsetX: event.clientX - rect.left,
        offsetY: event.clientY - rect.top,
      };
      cardEditorHeader.setPointerCapture?.(event.pointerId);
      cardEditorHeader.style.cursor = "grabbing";
    }

    function moveCardEditorDrag(event) {
      if (!cardEditorDragState || !cardEditorPanel) return;
      const shellRect = stageShell?.getBoundingClientRect?.();
      if (!shellRect) return;
      setCardEditorPosition(
        event.clientX - shellRect.left - cardEditorDragState.offsetX,
        event.clientY - shellRect.top - cardEditorDragState.offsetY,
      );
    }

    function endCardEditorDrag() {
      if (!cardEditorDragState) return;
      cardEditorDragState = null;
      if (cardEditorHeader) {
        cardEditorHeader.style.cursor = "move";
      }
    }

    function beginMapEditorDrag(event) {
      if (!mapEditorPanel || mapEditorPanel.hidden || !mapEditorHeader) return;
      if (event.button !== 0) return;
      event.preventDefault();
      event.stopPropagation();
      const rect = mapEditorPanel.getBoundingClientRect();
      mapEditorDragState = {
        offsetX: event.clientX - rect.left,
        offsetY: event.clientY - rect.top,
      };
      mapEditorHeader.setPointerCapture?.(event.pointerId);
      mapEditorHeader.style.cursor = "grabbing";
    }

    function moveMapEditorDrag(event) {
      if (!mapEditorDragState || !mapEditorPanel) return;
      const shellRect = stageShell?.getBoundingClientRect?.();
      if (!shellRect) return;
      setMapEditorPosition(
        event.clientX - shellRect.left - mapEditorDragState.offsetX,
        event.clientY - shellRect.top - mapEditorDragState.offsetY,
      );
    }

    function endMapEditorDrag() {
      if (!mapEditorDragState) return;
      mapEditorDragState = null;
      if (mapEditorHeader) {
        mapEditorHeader.style.cursor = "move";
      }
    }

    function beginGridEditorDrag(event) {
      if (!gridEditorPanel || gridEditorPanel.hidden || !gridEditorHeader) return;
      if (event.button !== 0) return;
      event.preventDefault();
      event.stopPropagation();
      const rect = gridEditorPanel.getBoundingClientRect();
      gridEditorDragState = {
        offsetX: event.clientX - rect.left,
        offsetY: event.clientY - rect.top,
      };
      gridEditorHeader.setPointerCapture?.(event.pointerId);
      gridEditorHeader.style.cursor = "grabbing";
    }

    function moveGridEditorDrag(event) {
      if (!gridEditorDragState || !gridEditorPanel) return;
      const shellRect = stageShell?.getBoundingClientRect?.();
      if (!shellRect) return;
      setGridEditorPosition(
        event.clientX - shellRect.left - gridEditorDragState.offsetX,
        event.clientY - shellRect.top - gridEditorDragState.offsetY,
      );
    }

    function endGridEditorDrag() {
      if (!gridEditorDragState) return;
      gridEditorDragState = null;
      if (gridEditorHeader) {
        gridEditorHeader.style.cursor = "move";
      }
    }

    function canSendStageAction() {
      return !!ws && ws.readyState === WebSocket.OPEN;
    }

    function scheduleSocketReconnect() {
      if (socketReconnectTimer) return;
      socketReconnectTimer = window.setTimeout(() => {
        socketReconnectTimer = null;
        socketReconnectDelay = Math.min(socketReconnectDelay * 2, 10000);
        connectSocket();
      }, socketReconnectDelay);
    }

    function queuePendingSocketAction(type, extra = {}) {
      pendingSocketActions.push({
        type,
        extra: { ...extra },
      });
    }

    function flushPendingSocketActions() {
      if (!canSendStageAction() || pendingSocketActions.length === 0) return;

      const queued = pendingSocketActions;
      pendingSocketActions = [];
      for (const item of queued) {
        ws.send(JSON.stringify({
          type: item.type,
          session_id: currentSessionId,
          ...item.extra,
        }));
      }
    }

    function sendAction(type, extra = {}) {
      if (!canSendStageAction()) {
        if (ws && ws.readyState === WebSocket.CONNECTING && currentSessionId) {
          queuePendingSocketAction(type, extra);
          return true;
        }
        return false;
      }

      ws.send(JSON.stringify({
        type,
        session_id: currentSessionId,
        ...extra,
      }));
      return true;
    }

    function applySnapshot(snapshot) {
      currentSnapshot = snapshot || null;
      const elements = Array.isArray(snapshot?.elements) ? snapshot.elements : [];
      const overlay = snapshot?.overlay || null;
      const showing = snapshot?.showing || null;
      const latestFire = elements.find((element) => element?.slug === "first-fire") || null;
      const stageCards = elements.filter((element) =>
        String(element?.element_type || "").toLowerCase() === "index_card" &&
        String(element?.surface || element?.data?.surface || "tray").toLowerCase() === "stage"
      );

      const nextObjects = buildObjects(snapshot);
      if (localPositionOverrides.size > 0) {
        for (const obj of nextObjects) {
          const key = String(obj.elementId || obj.elementSlug || obj.key || "");
          const override = key ? localPositionOverrides.get(key) : null;
          if (override) {
            obj.position = override;
          }
        }
      }
      if (nextObjects.length > 0 || currentObjects.length === 0) {
        currentObjects = nextObjects;
      }
      updateStatusSummary();
      setStageStatus(`Snapshot loaded. ${currentObjects.length} rendered object${currentObjects.length === 1 ? "" : "s"}.`);
      setMovementLine("Idle");

      const showingState = showing && String(showing.status || "").toLowerCase() !== "closed"
        ? "active"
        : "closed";
      const overlayState = overlay ? `${overlay.overlay_type || "text"} overlay active` : "no overlay";
      setSnapshotSummary(`Showing is ${showingState}; ${overlayState}; fire ${latestFire ? "is present" : "not present"}; staged cards ${stageCards.length}.`);

      if (!pixiApp) {
        return;
      }

      if (currentSelection && !currentObjects.find((object) => object.key === currentSelection.key)) {
        currentSelection = null;
      }
      if (currentSelection) {
        const refreshedSelection = currentObjects.find((object) => object.key === currentSelection.key);
        if (refreshedSelection) {
          currentSelection = refreshedSelection;
        }
      }

      if (!currentSelection) {
        setSelectionLine("None");
        setLiveFeedLine("No live object selected.");
      }

      if (nextObjects.length > 0 || currentObjects.length === 0) {
        renderPixiScene();
      }
      syncSelectedActions();
      syncCardEditorWithSelection();
    }

    async function refreshWorld() {
      const refreshSerial = ++worldRefreshSerial;
      try {
        if (!currentSessionId) {
          await ensureJoinedCave();
        }
        const response = await fetch("/api/world/the-cave", {
          method: "GET",
          credentials: "include",
          cache: "no-store",
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok || !payload?.data) {
          throw new Error(payload?.error || `HTTP ${response.status}`);
        }
        if (refreshSerial !== worldRefreshSerial) {
          return;
        }
        applySnapshot(payload.data);
      } catch (error) {
        if (refreshSerial !== worldRefreshSerial) {
          return;
        }
        console.error("refreshWorld failed", error);
        setStageStatus(`Snapshot refresh failed: ${error.message || String(error)}`);
      }
    }

    async function joinCave() {
      if (joinPromise) {
        return joinPromise;
      }

      joinPromise = joinCaveOnce().finally(() => {
        joinPromise = null;
      });
      return joinPromise;
    }

    async function ensureJoinedCave() {
      if (currentSessionId && currentActorId) return;
      await joinCave();
    }

    async function joinCaveOnce() {
      setStageStatus("Joining the live Cave session...");

      let sessionHandle = String(currentIdentity?.handle || "web").trim() || "web";
      let sessionDisplayName = String(currentIdentity?.display_name || "Friend").trim() || "Friend";
      if (!currentIdentity) {
        try {
          const sessionResponse = await fetch("/api/session/me", { credentials: "include" });
          const sessionPayload = await sessionResponse.json().catch(() => null);
          const session = sessionPayload?.data || {};
          currentIdentity = session;
          sessionHandle = String(session.handle || sessionHandle).trim() || sessionHandle;
          sessionDisplayName = String(session.display_name || sessionDisplayName).trim() || sessionDisplayName;
        } catch (err) {
          console.warn("session identity lookup failed", err);
        }
      }

      const response = await fetch("/api/session/the-cave/join", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          handle: sessionHandle,
          display_name: sessionDisplayName,
        }),
      });

      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok) {
        throw new Error(payload?.error || `Join failed (HTTP ${response.status})`);
      }

      const data = payload.data || {};
      currentRole = data.role || data.participant?.role || "audience";
      currentSessionId = data.session_id || data.session?.id || data.sessionId || "";
      currentActorId = data.actor_id || data.participant?.actor_id || data.participant?.user_id || data.user_id || "";

      if (!currentSessionId || !currentActorId) {
        throw new Error("Join succeeded but did not return session_id and actor_id.");
      }

      updateStatusSummary();
      updateShellMetaPresentation();
      setStageStatus(`Joined as ${roleLabel(currentRole)}.`);
    }

    function connectSocket() {
      if (socketReconnectTimer) {
        window.clearTimeout(socketReconnectTimer);
        socketReconnectTimer = null;
      }
      const protocol = location.protocol === "https:" ? "wss:" : "ws:";
      ws = new WebSocket(`${protocol}//${location.host}/ws/the-cave`);

      ws.addEventListener("open", () => {
        socketReconnectDelay = 1000;
        setStageStatus("Connected to the Cave WebSocket.");
        updateShellMetaPresentation();
        flushPendingSocketActions();
      });

      ws.addEventListener("close", () => {
        setStageStatus("WebSocket closed.");
        setMovementLine("Socket closed");
        updateShellMetaPresentation();
        pendingSocketActions = [];
        scheduleSocketReconnect();
      });

      ws.addEventListener("error", () => {
        setStageStatus("WebSocket error.");
        updateShellMetaPresentation();
        scheduleSocketReconnect();
      });

      ws.addEventListener("message", async (event) => {
        let msg;
        try {
          msg = JSON.parse(event.data);
        } catch (error) {
          return;
        }

        if (msg.type === "snapshot" && msg.data) {
          applySnapshot(msg.data);
          return;
        }

        if (msg.type === "action" && msg.data) {
          const actionType = msg.data.type || "";
          if (actionType === "chat/message") {
            appendChatActionLine(msg.data);
            return;
          }
          if (
            actionType === "act/place_element" ||
            actionType === "act/remove_element" ||
            actionType === "act/set_element_lock" ||
            actionType === "act/set_nameplate_visibility" ||
            actionType === "act/reveal_element" ||
            actionType === "act/hide_element" ||
            actionType === "create/index_card" ||
            actionType === "update/index_card" ||
            actionType === "delete/index_card"
          ) {
            await refreshWorld();
            if (actionType === "create/index_card") {
              placeCreatedIndexCard(msg.data);
            }
            if (actionType === "act/place_element") {
              setMovementLine("act/place_element accepted by the live action stream.");
            }
            if (actionType === "act/remove_element") {
              setMovementLine("act/remove_element accepted by the live action stream.");
            }
            if (actionType === "act/set_element_lock") {
              setMovementLine("act/set_element_lock accepted by the live action stream.");
            }
            if (actionType === "act/set_nameplate_visibility") {
              setMovementLine("act/set_nameplate_visibility accepted by the live action stream.");
            }
            if (actionType === "act/hide_element") {
              setMovementLine("act/hide_element accepted by the live action stream.");
            }
            if (actionType === "act/reveal_element") {
              setMovementLine("act/reveal_element accepted by the live action stream.");
            }
            return;
          }
        }

        if (msg.type === "showing/update") {
          await refreshWorld();
          updateChatPresentation();
          return;
        }

        if (msg.type === "error") {
          const errorText = String(msg.error || "action_denied");
          setStageStatus(`Action denied: ${errorText}`);
          setMovementLine(`Denied: ${errorText}`);
          if (errorText === "showing_closed") {
            appendSystemChatNotice(chatClosedMessage());
            if (chatInput) {
              chatInput.value = "";
            }
          } else if (errorText && errorText !== "chat/message") {
            appendSystemChatNotice(`Action denied: ${errorText}`);
          }
          pendingStageCardPlacement = null;
          stagePlacementCandidate = null;
        }
      });
    }

    async function bootstrapFirstTheater() {
      try {
        initializeShellChrome(null);
        headerHoverOpen = true;
        chatHoverOpen = true;
        drawerHoverOpen = { left: true, right: true };
        if (stageStatus) {
          stageStatus.hidden = false;
        }
        applyDrawerState("left", true);
        applyDrawerState("right", true);
        updateShellMetaPresentation();
        updateShellTargetPresentation();
        updateHeaderPresentation();
        updateChatPresentation();
        setStageStatus("First Theater shell ready. Load live tools when you want the full stage.");
        window.__firstTheaterRuntimeBootstrapped = true;
      } catch (error) {
        console.error("first theater bootstrap failed", error);
        setStageStatus(error.message || String(error));
        if (forbiddenScreen) {
          forbiddenScreen.hidden = true;
        }
        initializeShellChrome(currentIdentity);
      }
    }

    window.VictoryFirstTheater = {
      start() {
        if (firstTheaterRuntimeStarted) {
          return;
        }
        firstTheaterRuntimeStarted = true;
        void startFirstTheaterRuntime();
      },
      refreshWorld,
      loadPixiLibrary,
    };

    async function startFirstTheaterRuntime() {
      try {
        const pingStartedAt = performance.now();
        const sessionResponse = await fetch("/api/session/me", { credentials: "include" });
        const sessionPayload = await sessionResponse.json().catch(() => null);
        lastPingMs = Math.max(0, Math.round(performance.now() - pingStartedAt));
        currentIdentity = sessionPayload?.data || null;

        if (!sessionPayload?.signed_in) {
          setStageStatus("Shell only. Sign in to load the live First Theater tools.");
          setMovementLine("Unavailable");
          setSnapshotSummary("The shell stays open, but the live Pixi stage needs a signed-in session.");
          setRendererFallback(true, "Sign in to load the live First Theater tools.");
          if (stageEmptyState) {
            stageEmptyState.hidden = false;
          }
          window.__firstTheaterRuntimeLoaded = false;
          window.__firstTheaterRuntimeLoading = false;
          return;
        }

        const visibilityResponse = await fetch("/api/map/visibility", { credentials: "include" });
        const visibilityPayload = await visibilityResponse.json().catch(() => null);
        const visible = Array.isArray(visibilityPayload?.data)
          ? visibilityPayload.data.some((venue) => venue?.slug === "first-theater")
          : false;

        if (!visible) {
          setStageStatus("Shell only. This venue is not visible yet, but the overlay remains available.");
          setMovementLine("Unavailable");
          setSnapshotSummary("The First Theater Pixi stage is hidden until this venue becomes visible.");
          setRendererFallback(true, "The live Pixi stage is hidden until this venue becomes visible.");
          if (stageEmptyState) {
            stageEmptyState.hidden = false;
          }
          window.__firstTheaterRuntimeLoaded = false;
          window.__firstTheaterRuntimeLoading = false;
          return;
        }

        initializeShellChrome(currentIdentity);
        updateShellMetaPresentation();
        updateShellTargetPresentation();
        updateHeaderPresentation();
        updateChatPresentation();
        await joinCave();
        updateShellMetaPresentation();

        connectSocket();
        setStageStatus("Opening the live Cave socket...");

        await refreshWorld();
        setStageStatus("Loading Pixi stage tools...");
        await loadPixiLibrary();
        await new Promise((resolve) => window.requestAnimationFrame(() => resolve()));
        await initializePixi();
        if (stageEmptyState) {
          stageEmptyState.hidden = true;
        }
        window.__firstTheaterRuntimeLoaded = true;
        window.__firstTheaterRuntimeLoading = false;
      } catch (error) {
        console.error("first theater runtime failed", error);
        setStageStatus(error.message || String(error));
        setRendererFallback(true, "PixiJS failed to load. The stage stays readable in DOM-only mode.");
        if (forbiddenScreen) {
          forbiddenScreen.hidden = true;
        }
        window.__firstTheaterRuntimeLoaded = false;
        window.__firstTheaterRuntimeLoading = false;
      }
    }

    topBar?.addEventListener("pointerenter", (event) => syncHeaderHoverState(event.clientY));
    topBar?.addEventListener("pointerleave", (event) => syncHeaderHoverState(event.clientY));
    topBar?.addEventListener("focusin", updateHeaderPresentation);
    topBar?.addEventListener("focusout", (event) => {
      if (!topBar?.contains(event.relatedTarget)) {
        updateHeaderPresentation();
      }
    });
    topBar?.addEventListener("pointerdown", (event) => {
      syncHeaderHoverState(event.clientY);
    });
    document.addEventListener("pointermove", (event) => {
      syncHeaderHoverState(event.clientY);
    }, { passive: true });
    document.addEventListener("pointerdown", (event) => {
      syncHeaderHoverState(event.clientY);
    }, { passive: true });

    headerPinButton?.addEventListener("click", () => {
      setHeaderPinned(!isHeaderPinned());
    });

    headerSettingsButton?.addEventListener("click", (event) => {
      event.stopPropagation();
      toggleHeaderSettings();
    });

    resetShellPrefsButton?.addEventListener("click", (event) => {
      event.stopPropagation();
      resetShellPreferences();
    });

    headerOpacity?.addEventListener("input", () => {
      uiPreferences.header = {
        ...(uiPreferences.header || shellDefaults.header),
        opacity: clampNumber(headerOpacity.value, 0, 100, 100),
      };
      saveUiPreferences();
      updateHeaderPresentation();
    });

    chatOpacity?.addEventListener("input", () => {
      uiPreferences.chat = {
        ...(uiPreferences.chat || shellDefaults.chat),
        opacity: clampNumber(chatOpacity.value, 0, 100, 96),
      };
      saveUiPreferences();
      updateChatPresentation();
    });

    leftDrawer?.addEventListener("pointerenter", () => setDrawerHoverState("left", true));
    leftDrawer?.addEventListener("pointerleave", () => setDrawerHoverState("left", false));
    leftDrawer?.addEventListener("focusin", () => setDrawerHoverState("left", true));
    leftDrawer?.addEventListener("focusout", (event) => {
      if (!leftDrawer?.contains(event.relatedTarget)) {
        setDrawerHoverState("left", false);
      }
    });

    rightDrawer?.addEventListener("pointerenter", () => setDrawerHoverState("right", true));
    rightDrawer?.addEventListener("pointerleave", () => setDrawerHoverState("right", false));
    rightDrawer?.addEventListener("focusin", () => setDrawerHoverState("right", true));
    rightDrawer?.addEventListener("focusout", (event) => {
      if (!rightDrawer?.contains(event.relatedTarget)) {
        setDrawerHoverState("right", false);
      }
    });

    leftSettingsButton?.addEventListener("click", (event) => {
      event.stopPropagation();
      toggleDrawerSettings("left");
      setDrawerHoverState("left", true);
    });
    rightSettingsButton?.addEventListener("click", (event) => {
      event.stopPropagation();
      toggleDrawerSettings("right");
      setDrawerHoverState("right", true);
    });

    leftOpenMode?.addEventListener("change", () => setDrawerSetting("left", "mode", leftOpenMode.value));
    leftOpacity?.addEventListener("input", () => setDrawerSetting("left", "opacity", leftOpacity.value));
    leftPortraitSize?.addEventListener("change", () => setDrawerSetting("left", "portraitSize", leftPortraitSize.value));
    rightOpenMode?.addEventListener("change", () => setDrawerSetting("right", "mode", rightOpenMode.value));
    rightOpacity?.addEventListener("input", () => setDrawerSetting("right", "opacity", rightOpacity.value));

    chatHead?.addEventListener("pointerenter", (event) => syncChatHoverState(event.clientX, event.clientY));
    chatHead?.addEventListener("pointerleave", (event) => syncChatHoverState(event.clientX, event.clientY));
    chatPanel?.addEventListener("pointerenter", (event) => syncChatHoverState(event.clientX, event.clientY));
    chatPanel?.addEventListener("pointerleave", (event) => syncChatHoverState(event.clientX, event.clientY));
    chatPanel?.addEventListener("focusin", syncChatOpenState);
    chatPanel?.addEventListener("focusout", (event) => {
      if (!chatPanel?.contains(event.relatedTarget)) {
        chatHoverOpen = false;
        updateChatPresentation();
      }
    });
    chatPanel?.addEventListener("pointerdown", (event) => {
      syncChatHoverState(event.clientX, event.clientY);
    });
    chatHead?.addEventListener("click", (event) => {
      event.preventDefault();
      if (chatPinnedOpen) {
        closeChatPanel();
        return;
      }
      setChatPinnedOpen(true);
      chatHoverOpen = true;
      chatInput?.focus({ preventScroll: true });
    });
    chatHead?.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        if (chatPinnedOpen) {
          closeChatPanel();
          return;
        }
        setChatPinnedOpen(true);
        chatHoverOpen = true;
        chatInput?.focus({ preventScroll: true });
      }
    });
    chatInput?.addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        sendChatDraft();
      }
    });
    chatSend?.addEventListener("click", sendChatDraft);
    rightCardEditorButton?.addEventListener("click", () => {
      toggleCardEditorFromSelection();
    });

    accountMenuToggle?.addEventListener("click", (event) => {
      event.stopPropagation();
      if (accountMenu) {
        accountMenu.hidden = !accountMenu.hidden;
        accountMenuToggle.setAttribute("aria-expanded", String(!accountMenu.hidden));
        syncHeaderHoverState();
        updateHeaderPresentation();
      }
    });

    accountLogoutButton?.addEventListener("click", async () => {
      if (accountMenu) accountMenu.hidden = true;
      if (accountMenuToggle) accountMenuToggle.setAttribute("aria-expanded", "false");
      try {
        await fetch("/api/auth/logout", {
          method: "POST",
          credentials: "include",
        });
      } catch (error) {
        console.warn("logout request failed", error);
      } finally {
        window.location.href = "/";
      }
    });

    refreshWorldButton?.addEventListener("click", () => {
      refreshWorld();
    });

    shellRefreshWorldButton?.addEventListener("click", () => {
      refreshWorld();
    });

    clearSelectionButton?.addEventListener("click", () => {
      selectObject(null, "Selection cleared.");
      closeContextMenu();
    });

    shellClearSelectionButton?.addEventListener("click", () => {
      selectObject(null, "Selection cleared.");
      closeContextMenu();
    });

    cardEditorFront?.addEventListener("input", () => {
      cardEditorDirty = true;
      if (cardEditorStatus) {
        cardEditorStatus.textContent = "Unsaved changes.";
      }
    });

    cardEditorBack?.addEventListener("input", () => {
      cardEditorDirty = true;
      if (cardEditorStatus) {
        cardEditorStatus.textContent = "Unsaved changes.";
      }
    });

    cardEditorColor?.addEventListener("input", () => {
      cardEditorDirty = true;
      if (cardEditorStatus) {
        cardEditorStatus.textContent = "Unsaved changes.";
      }
    });

    cardEditorSave?.addEventListener("click", () => {
      saveCardEditor();
    });

    cardEditorCancel?.addEventListener("click", () => {
      hideCardEditor();
      if (currentSelection && isCardObject(currentSelection)) {
        setStageStatus(`${currentSelection.label} selected.`);
      }
    });

    mapEditorFile?.addEventListener("change", () => {
      const selected = mapEditorFile?.files?.[0] || null;
      if (!selected) {
        syncMapEditorPreview(currentVenueMapState?.asset?.content_url || "");
        return;
      }
      const previewURL = URL.createObjectURL(selected);
      syncMapEditorPreview(previewURL);
      mapEditorDirty = true;
      setMapEditorStatus(`Previewing ${selected.name}. Save to upload and replace the active map.`);
    });

    mapEditorDisplayMode?.addEventListener("change", () => {
      mapEditorDirty = true;
      const draft = mapEditorDraftFromState();
      currentVenueMapState = {
        ...(currentVenueMapState || {}),
        display_mode: draft.displayMode,
      };
      renderPixiScene();
    });

    mapEditorFit?.addEventListener("change", () => {
      mapEditorDirty = true;
      syncMapEditorPreview(mapEditorPreviewURL || currentVenueMapState?.asset?.content_url || "");
      livePreviewMapOnStage();
      setMapEditorStatus("Map fit updated.");
    });

    mapEditorScale?.addEventListener("input", () => {
      mapEditorDirty = true;
      syncMapEditorPreview(mapEditorPreviewURL || currentVenueMapState?.asset?.content_url || "");
      livePreviewMapOnStage();
    });

    mapEditorCropX?.addEventListener("input", () => {
      mapEditorDirty = true;
      syncMapEditorPreview(mapEditorPreviewURL || currentVenueMapState?.asset?.content_url || "");
      livePreviewMapOnStage();
    });

    mapEditorCropY?.addEventListener("input", () => {
      mapEditorDirty = true;
      syncMapEditorPreview(mapEditorPreviewURL || currentVenueMapState?.asset?.content_url || "");
      livePreviewMapOnStage();
    });

    mapEditorSafeMargin?.addEventListener("input", () => {
      mapEditorDirty = true;
      livePreviewMapOnStage();
    });

    mapEditorSave?.addEventListener("click", async () => {
      try {
        await saveVenueMap();
      } catch (error) {
        console.warn("saveVenueMap failed", error);
        setMapEditorStatus(error.message || String(error));
        setStageStatus(error.message || String(error));
      }
    });

    mapEditorRemove?.addEventListener("click", async () => {
      if (!window.confirm("Remove the active map from the First Theater stage?")) return;
      try {
        await removeVenueMap();
      } catch (error) {
        console.warn("removeVenueMap failed", error);
        setMapEditorStatus(error.message || String(error));
        setStageStatus(error.message || String(error));
      }
    });

    mapEditorCancel?.addEventListener("click", () => {
      hideMapEditor();
    });

    gridEditorType?.addEventListener("change", () => {
      gridEditorDirty = true;
      liveSyncGrid();
    });

    gridEditorHexOrientation?.addEventListener("change", () => {
      gridEditorDirty = true;
      liveSyncGrid();
    });

    gridEditorCellSize?.addEventListener("input", () => {
      gridEditorDirty = true;
      liveSyncGrid();
    });

    gridEditorOffsetX?.addEventListener("input", () => {
      gridEditorDirty = true;
      liveSyncGrid();
    });

    gridEditorOffsetY?.addEventListener("input", () => {
      gridEditorDirty = true;
      liveSyncGrid();
    });

    gridEditorOpacity?.addEventListener("input", () => {
      gridEditorDirty = true;
      liveSyncGrid();
    });

    gridEditorLineWidth?.addEventListener("input", () => {
      gridEditorDirty = true;
      liveSyncGrid();
    });

    gridEditorLineStyle?.addEventListener("change", () => {
      gridEditorDirty = true;
      liveSyncGrid();
    });

    gridEditorCellSizeUp?.addEventListener("click", (event) => {
      nudgeGridNumberField(gridEditorCellSize, event.shiftKey ? 25 : 5, 8, 500);
    });

    gridEditorCellSizeDown?.addEventListener("click", (event) => {
      nudgeGridNumberField(gridEditorCellSize, event.shiftKey ? -25 : -5, 8, 500);
    });

    gridEditorOffsetXUp?.addEventListener("click", (event) => {
      nudgeGridNumberField(gridEditorOffsetX, event.shiftKey ? 20 : 4, -2000, 2000);
    });

    gridEditorOffsetXDown?.addEventListener("click", (event) => {
      nudgeGridNumberField(gridEditorOffsetX, event.shiftKey ? -20 : -4, -2000, 2000);
    });

    gridEditorOffsetYUp?.addEventListener("click", (event) => {
      nudgeGridNumberField(gridEditorOffsetY, event.shiftKey ? 20 : 4, -2000, 2000);
    });

    gridEditorOffsetYDown?.addEventListener("click", (event) => {
      nudgeGridNumberField(gridEditorOffsetY, event.shiftKey ? -20 : -4, -2000, 2000);
    });

    gridEditorVisibility?.addEventListener("click", () => {
      toggleGridVisibilityDraft();
    });

    gridEditorReset?.addEventListener("click", () => {
      resetGridEditorDraft();
    });

    gridEditorSave?.addEventListener("click", async () => {
      try {
        await saveGridConfig();
      } catch (error) {
        console.warn("saveGridConfig failed", error);
        setGridEditorStatus(error.message || String(error));
        setStageStatus(error.message || String(error));
      }
    });

    gridEditorCancel?.addEventListener("click", () => {
      cancelGridEditor();
    });

    document.addEventListener("click", (event) => {
      if (contextMenu && !contextMenu.hidden && !contextMenu.contains(event.target)) {
        closeContextMenu();
      }
      const clickTarget = event.target instanceof Element ? event.target : null;
      if (chatPanel && clickTarget && !clickTarget.closest("#chat-panel") && (chatPinnedOpen || isChatActive() || chatHoverOpen)) {
        closeChatPanel();
      }
      if (accountMenu && !accountMenu.hidden && !event.target.closest("#account-menu") && !event.target.closest("#account-menu-toggle")) {
        accountMenu.hidden = true;
        accountMenuToggle?.setAttribute("aria-expanded", "false");
      }
      if (headerSettingsPanel && !headerSettingsPanel.hidden && !event.target.closest("#header-settings-panel") && !event.target.closest("#header-settings-button")) {
        closeHeaderSettings();
      }
      if (mapEditorPanel && !mapEditorPanel.hidden && !event.target.closest("#map-editor") && !event.target.closest("#map-editor-header")) {
        hideMapEditor();
      }
      if (gridEditorPanel && !gridEditorPanel.hidden && !event.target.closest("#grid-editor") && !event.target.closest("#grid-editor-header")) {
        hideGridEditor();
      }
      if (leftSettingsPanel && !leftSettingsPanel.hidden && !event.target.closest("#left-settings-panel") && !event.target.closest("#left-settings-button")) {
        leftSettingsPanel.hidden = true;
        leftSettingsButton?.setAttribute("aria-expanded", "false");
      }
      if (rightSettingsPanel && !rightSettingsPanel.hidden && !event.target.closest("#right-settings-panel") && !event.target.closest("#right-settings-button")) {
        rightSettingsPanel.hidden = true;
        rightSettingsButton?.setAttribute("aria-expanded", "false");
      }
    });

    document.addEventListener("keydown", (event) => {
      if (smokeMode && event.key && !event.metaKey && !event.ctrlKey && !event.altKey) {
        const tag = String(event.target?.tagName || "").toLowerCase();
        if (tag !== "input" && tag !== "textarea" && tag !== "select" && tag !== "button") {
          if (event.key.toLowerCase() === "g") {
            event.preventDefault();
            smokeToggleGridButton?.click();
            return;
          }
          if (event.key.toLowerCase() === "d") {
            event.preventDefault();
            smokeDropCardButton?.click();
            return;
          }
        }
      }
      if (event.key === "Escape") {
        closeContextMenu();
        if (accountMenu) accountMenu.hidden = true;
        if (headerSettingsPanel) headerSettingsPanel.hidden = true;
        hideMapEditor();
        if (leftSettingsPanel) leftSettingsPanel.hidden = true;
        if (rightSettingsPanel) rightSettingsPanel.hidden = true;
        accountMenuToggle?.setAttribute("aria-expanded", "false");
        headerSettingsButton?.setAttribute("aria-expanded", "false");
        leftSettingsButton?.setAttribute("aria-expanded", "false");
        rightSettingsButton?.setAttribute("aria-expanded", "false");
        if (currentSelection) {
          selectObject(null, "Selection cleared.");
        }
      }
    });

    contextMenu?.addEventListener("click", (event) => {
      event.stopPropagation();
      const button = event.target.closest("[data-menu-action]");
      if (!button) return;
      const action = button.getAttribute("data-menu-action");
      if (button.disabled) return;
      const objectKey = contextMenu.dataset.objectKey || "";
      const objectModel = contextMenuTarget || currentObjects.find((item) => item.key === objectKey) || null;
      if (!objectModel) {
        closeContextMenu();
        return;
      }
      performStageObjectAction(action, objectModel);
    });

    overlayRoot?.addEventListener("contextmenu", (event) => {
      if (event.defaultPrevented) return;
      if (event.target.closest("#pixi-context-menu, textarea, input, select, a")) return;
      event.preventDefault();
      event.stopPropagation();
      const point = stagePointFromClient(event.clientX, event.clientY);
      stagePlacementCandidate = point;
      updatePointerReadout(event.clientX || 0, event.clientY || 0, point, "Overlay menu");
      openContextMenu(event, stageContextMenuModel());
    });

    stageShell?.addEventListener("contextmenu", handleNativeStageContextMenu, true);
    stageHost?.addEventListener("contextmenu", handleNativeStageContextMenu, true);

    smokeToggleGridButton?.addEventListener("click", () => {
      smokeGridEnabled = !smokeGridEnabled;
      setSmokeLine(smokeGridEnabled ? `Grid on · Last pointer ${currentStagePointLabel()}` : `Grid off · Last pointer ${currentStagePointLabel()}`);
      updateSmokeUI();
      renderPixiScene();
    });

    smokeDropCardButton?.addEventListener("click", () => {
      if (!smokeMode) return;
      if (!lastStagePoint) {
        setSmokeLine("Move the pointer over the stage first.");
        return;
      }
      if (!canManageIndexCards(currentRole)) {
        setSmokeLine("Smoke card creation needs producer or director access.");
        return;
      }
      pendingStageCardPlacement = { ...lastStagePoint };
      stagePlacementCandidate = { ...lastStagePoint };
      setSmokeLine(`Spawning smoke card at ${currentStagePointLabel()}.`);
      const sent = sendAction("create/index_card", {
        front_text: "Smoke Test Card",
        back_text: "Drag me to test pointer math.",
        color: "#d9c7a6",
      });
      if (!sent) {
        pendingStageCardPlacement = null;
        stagePlacementCandidate = null;
        setSmokeLine("Socket unavailable.");
      }
    });

    bootstrapFirstTheater();
