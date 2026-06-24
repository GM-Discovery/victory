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
    const cameraControls = document.getElementById("camera-controls");
    const cameraZoomOutButton = document.getElementById("camera-zoom-out");
    const cameraZoomValue = document.getElementById("camera-zoom-value");
    const cameraZoomInButton = document.getElementById("camera-zoom-in");
    const cameraFitButton = document.getElementById("camera-fit");
    const pingButton = document.getElementById("ping-button");
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
    const tokenPickerPanel = document.getElementById("token-picker");
    const tokenPickerHeader = document.getElementById("token-picker-header");
    const tokenPickerStatus = document.getElementById("token-picker-status");
    const tokenPickerPreview = document.getElementById("token-picker-preview");
    const tokenPickerPreviewBadge = document.getElementById("token-picker-preview-badge");
    const tokenPickerSearch = document.getElementById("token-picker-search");
    const tokenPickerShape = document.getElementById("token-picker-shape");
    const tokenPickerList = document.getElementById("token-picker-list");
    const tokenPickerRefresh = document.getElementById("token-picker-refresh");
    const tokenPickerCancel = document.getElementById("token-picker-cancel");
    const tokenEditorPanel = document.getElementById("token-editor");
    const tokenEditorHeader = document.getElementById("token-editor-header");
    const tokenEditorStatus = document.getElementById("token-editor-status");
    const tokenEditorScale = document.getElementById("token-editor-scale");
    const tokenEditorScaleValue = document.getElementById("token-editor-scale-value");
    const tokenEditorSnap = document.getElementById("token-editor-snap");
    const tokenEditorLayer = document.getElementById("token-editor-layer");
    const tokenEditorReset = document.getElementById("token-editor-reset");
    const tokenEditorSave = document.getElementById("token-editor-save");
    const tokenEditorCancel = document.getElementById("token-editor-cancel");
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
    const firstTheaterStateModule = window.VictoryFirstTheaterState || null;
    const firstTheaterSocketModule = window.VictoryFirstTheaterSocket || null;
    const firstTheaterLogicModule = window.VictoryFirstTheaterLogic || null;
    const firstTheaterGeometryModule = window.VictoryFirstTheaterGeometry || null;
    const firstTheaterStageControlsModule = window.VictoryFirstTheaterStageControls || null;
    const firstTheaterMapGridModule = window.VictoryFirstTheaterMapGrid || null;
    const firstTheaterSceneNodesModule = window.VictoryFirstTheaterSceneNodes || null;
    const firstTheaterTokenUiModule = window.VictoryFirstTheaterTokenUI || null;
    const firstTheaterActionRouterModule = window.VictoryFirstTheaterActionRouter || null;
    const firstTheaterSocketControllerModule = window.VictoryFirstTheaterSocketController || null;
    const firstTheaterLifecycleModule = window.VictoryFirstTheaterLifecycle || null;
    const projectedState = firstTheaterStateModule?.createProjectedState?.({
      viewerRole: currentRole,
    }) || null;
    const firstTheaterDispatcher = firstTheaterSocketModule?.createDispatcher?.({
      state: projectedState,
      parseMessage(raw) {
        return JSON.parse(raw);
      },
    }) || null;
    const runtimeLifecycle = firstTheaterLifecycleModule?.createLifecycleBag?.() || {
      track: (dispose) => dispose,
      listen: (target, type, handler, options) => {
        target?.addEventListener?.(type, handler, options);
        return () => target?.removeEventListener?.(type, handler, options);
      },
      timeout: (callback, delay) => window.setTimeout(callback, delay),
      interval: (callback, delay) => window.setInterval(callback, delay),
      animationFrame: (callback) => window.requestAnimationFrame(callback),
      dispose: () => {},
    };
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
    let stagePlacementScreenCandidate = null;
    let pendingStageCardPlacement = null;
    let recentPlacementMarker = null;
    let recentFocusMarker = null;
    let cardEditorTargetKey = "";
    let cardEditorDirty = false;
    let cardEditorDragState = null;
    let cardFaceState = new Map();
    let tokenPickerState = {
      open: false,
      mode: "create",
      selectedAssetID: "",
      filterShape: "all",
      search: "",
      placementPoint: null,
      placementScreenPoint: null,
      replaceTargetKey: "",
      replaceTargetElementID: "",
      replaceTargetElementSlug: "",
      replaceTargetScale: 100,
      replaceTargetSnapMode: "grid",
      replaceTargetTokenLayer: "public",
    };
    let tokenEditorTargetKey = "";
    let tokenEditorDirty = false;
    let warehouseTokenAssets = [];
    let warehouseTokenAssetPreviewURL = "";
    let tokenPickerDragState = null;
    let tokenEditorDragState = null;
    let localPositionOverrides = new Map();
    let currentVenueMapState = null;
    let currentVenueMapAssets = [];
    let currentVenueMapAssetID = "";
    let currentVenueMapBounds = null;
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
    let worldLayer = null;
    let mapLayer = null;
    let gridLayer = null;
    let facadeLayer = null;
    let pinnedObjectLayer = null;
    let overlayObjectLayer = null;
    let floatingObjectLayer = null;
    let uiLayer = null;
    let stageCamera = null;
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
    let pendingPingStartedAt = null;
    let currentPresenceUsers = [];
    let chatHoverOpen = false;
    let chatPinnedOpen = false;
    let chatDismissedUntilPointerLeavesRail = false;
    let chatHoverTransitionTimer = null;
    let lastChatPointerY = Number.NaN;
    let headerHoverOpen = false;
    let lastHeaderPointerY = Number.NaN;
    let drawerHoverOpen = { left: false, right: false };
    let drawerHoverCloseTimers = { left: null, right: null };
    let latestFocusEventStamp = 0;
    const stageSceneState = {
      get currentNodeMap() {
        return currentNodeMap;
      },
      set currentNodeMap(value) {
        currentNodeMap = value;
      },
      get pinnedObjectLayer() {
        return pinnedObjectLayer;
      },
      get overlayObjectLayer() {
        return overlayObjectLayer;
      },
      get floatingObjectLayer() {
        return floatingObjectLayer;
      },
      get facadeLayer() {
        return facadeLayer;
      },
      get currentSelection() {
        return currentSelection;
      },
    };
    let sceneNodeFactory = null;
    let tokenUi = null;
    let actionRouter = null;
    let socketController = null;
    const shellDefaults = {
      header: { pinned: false, opacity: 100 },
      chat: { opacity: 96 },
    };
    const cameraDefaults = {
      minZoom: 0.7,
      maxZoom: 4,
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
    const leftDrawerEdgeTrigger = document.getElementById("left-drawer-edge-trigger");
    const rightDrawerEdgeTrigger = document.getElementById("right-drawer-edge-trigger");
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

    function formatByteSize(bytes) {
      const parsed = Number(bytes);
      if (!Number.isFinite(parsed) || parsed < 0) {
        return "0 B";
      }
      if (parsed < 1024) {
        return `${Math.round(parsed)} B`;
      }
      const units = ["KB", "MB", "GB", "TB"];
      let value = parsed / 1024;
      let unit = "KB";
      for (let index = 0; index < units.length - 1 && value >= 1024; index += 1) {
        value /= 1024;
        unit = units[index + 1];
      }
      return `${value >= 10 || unit === "TB" ? value.toFixed(0) : value.toFixed(1)} ${unit}`;
    }

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

    function canManageStageTokens(role) {
      return canManageIndexCards(role);
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

    function updateCameraControls(view = null) {
      const mapDisplayMode = String(currentVenueMapState?.display_mode || "theater").trim() === "fullscreen" ? "fullscreen" : "theater";
      const cameraState = firstTheaterStageControlsModule?.cameraControlsState?.(view || stageCamera?.getView?.(), mapDisplayMode, cameraDefaults) || {
        label: "100%",
        zoomOutDisabled: false,
        zoomInDisabled: false,
        interactionLocked: false,
      };
      if (cameraZoomValue) {
        cameraZoomValue.textContent = cameraState.label;
      }
      if (cameraZoomOutButton && cameraZoomInButton) {
        cameraZoomOutButton.disabled = cameraState.zoomOutDisabled;
        cameraZoomInButton.disabled = cameraState.zoomInDisabled;
      }
      stageCamera?.setInteractionLocked?.(cameraState.interactionLocked);
    }

    function getPlayableBounds() {
      const size = getStageSize();
      return computeStagePlayableBounds(size.width, size.height);
    }

    function currentCameraView() {
      return stageCamera?.getView?.() || {
        activeMapId: String(currentVenueMapState?.asset_id || currentVenueMapAssetID || ""),
        panX: 0,
        panY: 0,
        zoomRelativeToFit: 1,
      };
    }

    function resetCameraToFit(activeMapId = currentVenueMapState?.asset_id || currentVenueMapAssetID || "") {
      stageCamera?.setActiveMapId?.(activeMapId, { reset: true });
      stageCamera?.fit?.(activeMapId);
      updateCameraControls();
    }

    function setCameraWorldBounds(bounds) {
      stageCamera?.setWorldBounds?.(bounds);
      updateCameraControls();
    }

    function setCameraPlayableBounds() {
      stageCamera?.setPlayableBounds?.(getPlayableBounds());
      updateCameraControls();
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
      const pin = cardDisplayMode(currentSelection);
      return `${currentSelection.label || "selection"} · ${currentSelection.kind || "object"} · ${frame} · ${Number(position.x ?? 0)}, ${Number(position.y ?? 0)} · pin ${pin} · locked ${state.locked ? "yes" : "no"} · nameplate ${state.nameplateVisible ? "visible" : "hidden"} · ${state.visible ? "visible" : "hidden"}`;
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
      const socket = socketController?.getWebSocket?.() || ws;
      const socketOpen = socket && socket.readyState === WebSocket.OPEN;
      const socketConnecting = socket && socket.readyState === WebSocket.CONNECTING;
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
      if (prefs.pinned || isHeaderDetailsVisible()) {
        if (!headerHoverOpen) {
          headerHoverOpen = true;
          updateHeaderPresentation();
        }
        return;
      }

      const pointerY = Number.isFinite(lastHeaderPointerY) ? lastHeaderPointerY : clientY;
      if (!Number.isFinite(pointerY)) return;

      const headerRect = topBar.getBoundingClientRect();
      const triggerLine = 20;
      const releaseLine = headerRect.bottom + 6;
      const nextOpen = headerHoverOpen ? pointerY <= releaseLine : pointerY <= triggerLine;

      if (nextOpen !== headerHoverOpen) {
        headerHoverOpen = nextOpen;
        updateHeaderPresentation();
      }
    }

    function closeHeaderHoverState() {
      if (!headerHoverOpen) return;
      const prefs = uiPreferences?.header || shellDefaults.header;
      if (prefs.pinned || isHeaderDetailsVisible()) return;
      headerHoverOpen = false;
      updateHeaderPresentation();
    }

    function setHeaderPinned(pinned) {
      uiPreferences.header = {
        ...(uiPreferences.header || shellDefaults.header),
        pinned: Boolean(pinned),
      };
      saveUiPreferences();
      if (!pinned) {
        closeHeaderHoverState();
      }
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

    function scheduleDrawerHoverClose(side) {
      if (drawerHoverCloseTimers[side]) {
        window.clearTimeout(drawerHoverCloseTimers[side]);
      }
      drawerHoverCloseTimers[side] = window.setTimeout(() => {
        drawerHoverCloseTimers[side] = null;
        const state = side === "left" ? leftDrawer : rightDrawer;
        if (!state) return;
        const prefs = uiPreferences?.[side] || drawerDefaults[side];
        if (prefs.mode !== "hover") return;
        if (state.matches(":hover") || state.contains(document.activeElement) || isDrawerDetailsVisible(side)) {
          return;
        }
        drawerHoverOpen[side] = false;
        applyDrawerState(side, false);
      }, 120);
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
      if (open) {
        if (drawerHoverCloseTimers[side]) {
          window.clearTimeout(drawerHoverCloseTimers[side]);
          drawerHoverCloseTimers[side] = null;
        }
        drawerHoverOpen[side] = true;
        applyDrawerState(side, true);
        return;
      }
      scheduleDrawerHoverClose(side);
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
      if (chatHoverTransitionTimer) {
        window.clearTimeout(chatHoverTransitionTimer);
        chatHoverTransitionTimer = null;
      }
      chatHoverOpen = Boolean(open);
      if (!chatHoverOpen && !chatPinnedOpen && !isChatActive()) {
        lastChatPointerY = Number.NaN;
      }
      updateChatPresentation();
    }

    function setChatPinnedOpen(open) {
      if (chatHoverTransitionTimer) {
        window.clearTimeout(chatHoverTransitionTimer);
        chatHoverTransitionTimer = null;
      }
      chatPinnedOpen = Boolean(open);
      if (!chatPinnedOpen && !isChatActive()) {
        chatHoverOpen = false;
        lastChatPointerY = Number.NaN;
      }
      updateChatPresentation();
    }

    function closeChatPanel() {
      if (chatHoverTransitionTimer) {
        window.clearTimeout(chatHoverTransitionTimer);
        chatHoverTransitionTimer = null;
      }
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
      const insideRail = pointerX >= chatRect.left - 6 && pointerX <= chatRect.right + 6 && pointerY >= chatRect.top - 4 && pointerY <= chatRect.bottom + 4;
      const nearRail = pointerY >= window.innerHeight - 24;

      const nextOpen = insideRail || nearRail;

      if (chatDismissedUntilPointerLeavesRail) {
        if (nextOpen) {
          if (chatHoverOpen) {
            chatHoverOpen = false;
            updateChatPresentation();
          }
          return;
        }
        if (pointerY < window.innerHeight - 72) {
          chatDismissedUntilPointerLeavesRail = false;
        } else {
          return;
        }
      }

      if (chatHoverTransitionTimer) {
        window.clearTimeout(chatHoverTransitionTimer);
      }
      chatHoverTransitionTimer = window.setTimeout(() => {
        chatHoverTransitionTimer = null;
        if (chatPinnedOpen || isChatActive()) {
          return;
        }
        if (chatHoverOpen !== nextOpen) {
          chatHoverOpen = nextOpen;
          updateChatPresentation();
        }
      }, nextOpen ? 70 : 180);
      if (nextOpen) {
        chatDismissedUntilPointerLeavesRail = false;
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
      if (chatHoverTransitionTimer) {
        window.clearTimeout(chatHoverTransitionTimer);
        chatHoverTransitionTimer = null;
      }
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
      stagePlacementScreenCandidate = null;
      suppressStageContextMenu = false;
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

    function setRecentFocusMarker(point, label = "") {
      if (!point) {
        recentFocusMarker = null;
        return;
      }
      recentFocusMarker = {
        x: Math.round(Number(point.x || 0)),
        y: Math.round(Number(point.y || 0)),
        label: String(label || "").trim(),
        at: Date.now(),
      };
    }

    const venueConfigFlag = firstTheaterLogicModule?.venueConfigFlag || ((config, key, fallback = false) => fallback);
    const objectKind = firstTheaterLogicModule?.objectKind || (() => "");
    function objectState(model) {
      return firstTheaterLogicModule?.objectState?.(model) || { locked: false, nameplateVisible: true, visible: true };
    }
    const updateObjectMatches = firstTheaterLogicModule?.updateObjectMatches || (() => false);
    const viewerCanSeeHiddenCards = firstTheaterLogicModule?.viewerCanSeeHiddenCards || (() => true);
    const cardStatusBadgeText = firstTheaterLogicModule?.cardStatusBadgeText || (() => "");

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
          if (item.kind === "token") {
            item.source.data.asset_id = item.assetID ?? item.source.data.asset_id;
            item.source.data.asset_name = item.assetName ?? item.source.data.asset_name;
            item.source.data.asset_shape = item.assetShape ?? item.source.data.asset_shape;
            item.source.data.asset_content_url = item.assetContentURL ?? item.source.data.asset_content_url;
            item.source.data.asset_thumbnail_url = item.assetThumbnailURL ?? item.source.data.asset_thumbnail_url;
            item.source.data.default_grid_width = item.defaultGridWidth ?? item.source.data.default_grid_width;
            item.source.data.default_grid_height = item.defaultGridHeight ?? item.source.data.default_grid_height;
            item.source.data.snap_mode = item.snapMode ?? item.source.data.snap_mode;
            item.source.data.grid_relative = item.gridRelative ?? item.source.data.grid_relative;
            item.source.data.token_layer = item.tokenLayer ?? item.source.data.token_layer;
            item.source.data.scale = item.scale ?? item.source.data.scale;
          }
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

    const canActorRevealHideStageObjects = firstTheaterLogicModule?.canActorRevealHideStageObjects || (() => false);
    const isCardObject = firstTheaterLogicModule?.isCardObject || (() => false);
    const isTokenObject = firstTheaterLogicModule?.isTokenObject || (() => false);
    const isLiveStageObject = firstTheaterLogicModule?.isLiveStageObject || (() => false);
    const canEditLiveCard = firstTheaterLogicModule?.canEditLiveCard || (() => false);
    const canMoveLiveStageObject = firstTheaterLogicModule?.canMoveLiveStageObject || (() => false);
    const canDuplicateLiveStageObject = firstTheaterLogicModule?.canDuplicateLiveStageObject || (() => false);
    const canRemoveLiveStageObject = firstTheaterLogicModule?.canRemoveLiveStageObject || (() => false);
    const canDeleteLiveCard = firstTheaterLogicModule?.canDeleteLiveCard || (() => false);

    sceneNodeFactory = firstTheaterSceneNodesModule?.createSceneNodeFactory?.({
      PIXI: window.PIXI,
      objectState: (...args) => objectState(...args),
      canEditLiveCard: (...args) => canEditLiveCard(...args),
      canToggleLock: (...args) => canToggleLock(...args),
      canManageIndexCards: (...args) => canManageIndexCards(...args),
      currentRole: () => currentRole,
      cardFaceForModel: (...args) => cardFaceForModel(...args),
      cardStatusBadgeText: (...args) => cardStatusBadgeText(...args),
      viewerCanSeeHiddenCards: (...args) => viewerCanSeeHiddenCards(...args),
      tokenDisplayNameForModel: (...args) => tokenDisplayNameForModel(...args),
      tokenStatusPercentForModel: (...args) => tokenStatusPercentForModel(...args),
      tokenLayerForModel: (...args) => tokenLayerForModel(...args),
      tokenDisplaySizeForModel: (...args) => tokenDisplaySizeForModel(...args),
      truncateCardText: (...args) => truncateCardText(...args),
      isTokenObject: (...args) => isTokenObject(...args),
      selectObject: (...args) => selectObject(...args),
      setStageStatus: (...args) => setStageStatus(...args),
      setMovementReport: (...args) => setMovementReport(...args),
      openCardEditor: (...args) => openCardEditor(...args),
      wasContextMenuHandled: (...args) => wasContextMenuHandled(...args),
      cancelContextMenuEvent: (...args) => cancelContextMenuEvent(...args),
      openResolvedContextMenu: (...args) => openResolvedContextMenu(...args),
      eventClientPoint: (...args) => eventClientPoint(...args),
      stageScreenPointFromClient: (...args) => stageScreenPointFromClient(...args),
      cardDisplayMode: (...args) => cardDisplayMode(...args),
      tokenSnapModeForModel: (...args) => tokenSnapModeForModel(...args),
      renderPixiScene: () => renderPixiScene(),
      stageState: stageSceneState,
      cardFaceState,
      setDragState(value) {
        dragState = value;
      },
    }) || null;

    tokenUi = firstTheaterTokenUiModule?.createTokenUi?.({
      state: tokenPickerState,
      canManageStageTokens: () => canManageStageTokens(currentRole),
      tokenPlacementPointForCreate: (...args) => tokenPlacementPointForCreate(...args),
      tokenScaleForModel: (...args) => tokenScaleForModel(...args),
      tokenSnapModeForModel: (...args) => tokenSnapModeForModel(...args),
      tokenLayerForModel: (...args) => tokenLayerForModel(...args),
      renderPixiScene: () => renderPixiScene(),
      setStageStatus: (...args) => setStageStatus(...args),
      setMovementLine: (...args) => setMovementLine(...args),
      formatByteSize: (...args) => formatByteSize(...args),
      escapeHtml: (...args) => escapeHtml(...args),
      sendAction: (...args) => sendAction(...args),
      refreshVenueMapAssets: (...args) => refreshVenueMapAssets(...args),
      currentGridConfig: () => currentVenueGridConfig,
      lastStagePoint: () => lastStagePoint,
      stagePlacementCandidate: () => stagePlacementCandidate,
      stagePlacementScreenCandidate: () => stagePlacementScreenCandidate,
      setPlacementState: (next) => {
        if (next?.point !== undefined) stagePlacementCandidate = next.point;
        if (next?.screenPoint !== undefined) stagePlacementScreenCandidate = next.screenPoint;
      },
      openEditor: (kind) => {
        if (kind === "card") hideCardEditor();
        if (kind === "map") hideMapEditor();
        if (kind === "grid") hideGridEditor();
        if (kind === "token") hideTokenEditor();
      },
      closeEditor: (kind) => {
        if (kind === "card") hideCardEditor();
        if (kind === "map") hideMapEditor();
        if (kind === "grid") hideGridEditor();
        if (kind === "token") hideTokenEditor();
      },
      setEditorStatus: (kind, text) => {
        if (kind === "token") {
          if (tokenEditorStatus) tokenEditorStatus.textContent = text || "";
        }
      },
      getEditorState: () => ({
        panel: tokenEditorPanel,
        currentSelection,
        tokenEditorTargetKey,
        tokenEditorDirty,
        scale: tokenEditorScale,
        scaleValue: tokenEditorScaleValue,
        snap: tokenEditorSnap,
        layer: tokenEditorLayer,
        status: tokenEditorStatus,
        setPosition: setTokenEditorPosition,
      }),
      getTokenAssets: () => warehouseTokenAssets,
      setTokenAssets: (assets) => {
        warehouseTokenAssets = Array.isArray(assets) ? assets : [];
      },
      getTokenPickerElements: () => ({
        panel: tokenPickerPanel,
        status: tokenPickerStatus,
        preview: tokenPickerPreview,
        previewBadge: tokenPickerPreviewBadge,
        search: tokenPickerSearch,
        shape: tokenPickerShape,
        list: tokenPickerList,
        setPosition: setTokenPickerPosition,
      }),
      refreshWarehouseTokenAssets: () => refreshWarehouseTokenAssets(),
      clampNumber: (...args) => clampNumber(...args),
      objectState: (...args) => objectState(...args),
      isTokenObject: (...args) => isTokenObject(...args),
      currentObjects: () => currentObjects,
      updateTokenLocalModel: (...args) => updateTokenLocalModel(...args),
    }) || null;

    actionRouter = firstTheaterActionRouterModule?.createActionRouter?.({
      objectState: (...args) => objectState(...args),
      objectKind: (...args) => objectKind(...args),
      isTokenObject: (...args) => isTokenObject(...args),
      isLiveStageObject: (...args) => isLiveStageObject(...args),
      setStageStatus: (...args) => setStageStatus(...args),
      setMovementReport: (...args) => setMovementReport(...args),
      selectObject: (...args) => selectObject(...args),
      closeContextMenu: () => closeContextMenu(),
      openTokenPicker: (...args) => openTokenPicker(...args),
      openTokenEditor: (...args) => openTokenEditor(...args),
      openMapEditor: (...args) => openMapEditor(...args),
      openGridEditor: (...args) => openGridEditor(...args),
      updateLocalObjectModel: (...args) => updateLocalObjectModel(...args),
      updateLocalCardPinModel: (...args) => updateLocalCardPinModel(...args),
      updateTokenLocalModel: (...args) => updateTokenLocalModel(...args),
      setLocalPositionOverrideForModel: (...args) => setLocalPositionOverrideForModel(...args),
      tokenMoveTargetForModel: (...args) => tokenMoveTargetForModel(...args),
      cardMoveTargetForModel: (...args) => cardMoveTargetForModel(...args),
      tokenLayerForModel: (...args) => tokenLayerForModel(...args),
      tokenScaleForModel: (...args) => tokenScaleForModel(...args),
      tokenPlacementPointForCreate: (...args) => tokenPlacementPointForCreate(...args),
      currentDisplayedPointForModel: (...args) => currentDisplayedPointForModel(...args),
      tokenDuplicatePlacementForModel: (...args) => tokenDuplicatePlacementForModel(...args),
      cardDuplicatePlacementForModel: (...args) => cardDuplicatePlacementForModel(...args),
      sendAction: (...args) => sendAction(...args),
      removeLocalObject: (objectModel) => {
        currentObjects = currentObjects.filter((item) => !updateObjectMatches(item, objectModel));
      },
      setPlacementState: (point, screenPoint) => {
        stagePlacementCandidate = point || null;
        stagePlacementScreenCandidate = screenPoint || null;
      },
      getStagePoint: () => stagePlacementCandidate || lastStagePoint || null,
      stageScreenPointFromClient: (...args) => stageScreenPointFromClient(...args),
      stagePointFromClient: (...args) => stagePointFromClient(...args),
      stagePlacementFromEvent: (...args) => stagePlacementFromEvent(...args),
      hitTestContextMenuTarget: (...args) => hitTestContextMenuTarget(...args),
      stageContextMenuModel: () => stageContextMenuModel(),
      eventClientPoint: (...args) => eventClientPoint(...args),
      resolveStageObjectActions: (...args) => resolveStageObjectActions(...args),
      resolveStageObjectActionsContext: () => ({
        role: currentRole,
        canManageIndexCards: canManageIndexCards(currentRole),
        canManageStageTokens: canManageStageTokens(currentRole),
        canActorRevealHideStageObjects: canActorRevealHideStageObjects(currentRole, currentSnapshot?.venue?.config || {}),
        hasSelection: Boolean(currentSelection),
        gridConfig: currentVenueGridConfig,
        cardFaceForModel,
        cardDisplayMode,
        tokenScaleForModel,
        tokenSnapModeForModel,
        tokenLayerForModel,
      }),
      contextMenu: () => contextMenu,
      pixiApp: () => pixiApp,
      cameraControlsContains: (target) => cameraControls?.contains?.(target),
      isStageEvent: (event) => {
        const target = event?.target;
        const composedPath = typeof event?.composedPath === "function" ? event.composedPath() : [];
        return target === stageHost || target === stageShell || stageShell?.contains?.(target) || stageHost?.contains?.(target) || composedPath.includes(stageHost) || composedPath.includes(stageShell) || composedPath.includes(pixiApp?.view);
      },
      isSecondaryPointerEvent: (...args) => isSecondaryPointerEvent(...args),
      markContextMenuHandled: (...args) => markContextMenuHandled(...args),
      openMapEditor: (...args) => openMapEditor(...args),
      openGridEditor: (...args) => openGridEditor(...args),
      renderContextMenu: (items) => {
        contextMenu.innerHTML = items
          .map((item, index) => {
            let separator = "";
            if (index > 0 && items[index - 1].group !== item.group) {
              separator = '<div class="menu-separator"></div>';
            }
            const disabled = item.disabled ? " disabled" : "";
            return `${separator}<button type="button" data-menu-action="${escapeHtml(item.action)}"${disabled}>${escapeHtml(item.label)}</button>`;
          })
          .join("") + `<div class="menu-separator"></div><button type="button" data-menu-action="copy" disabled>Copy</button><button type="button" data-menu-action="paste" disabled>Paste</button>`;
      },
      positionContextMenu: (x, y) => {
        contextMenu.hidden = false;
        contextMenu.style.left = "0px";
        contextMenu.style.top = "0px";
        const rect = contextMenu.getBoundingClientRect();
        const maxX = Math.max(8, window.innerWidth - rect.width - 8);
        const maxY = Math.max(8, window.innerHeight - rect.height - 8);
        contextMenu.style.left = `${Math.max(8, Math.min(x || 0, maxX))}px`;
        contextMenu.style.top = `${Math.max(8, Math.min(y || 0, maxY))}px`;
      },
      setContextMenuTarget: (model) => {
        contextMenuTarget = model;
      },
      closeContextMenuUI: () => {
        if (contextMenu) {
          contextMenu.hidden = true;
          contextMenu.innerHTML = "";
        }
        contextMenuTarget = null;
        stagePlacementCandidate = null;
        stagePlacementScreenCandidate = null;
      },
      setStagePlacementCandidate: (point, screenPoint) => {
        stagePlacementCandidate = point;
        stagePlacementScreenCandidate = screenPoint;
      },
    }) || null;

    socketController = firstTheaterSocketControllerModule?.createSocketController?.({
      getSessionId: () => currentSessionId,
      getStageCamera: () => stageCamera,
      getLastStagePoint: () => lastStagePoint,
      getStageSize: () => getStageSize(),
      getLocation: () => location,
      getWebSocketCtor: () => window.WebSocket,
      getPerformanceNow: () => performance.now(),
      setStageStatus: (...args) => setStageStatus(...args),
      setMovementLine: (...args) => setMovementLine(...args),
      updateShellMetaPresentation: () => updateShellMetaPresentation(),
      refreshWorld: () => refreshWorld(),
      onMessage: (raw) => {
        const msg = firstTheaterDispatcher?.dispatch?.(raw) || null;
        if (msg) {
          void handleSocketMessage(msg);
        }
        return msg;
      },
      onPong: (value) => {
        if (Number.isFinite(Number(value))) {
          lastPingMs = Number(value);
          pendingPingStartedAt = null;
          updateShellMetaPresentation();
        }
      },
    }) || null;

    const cardFaceForModel = (model) => {
      const cardKey = String(model?.elementId || model?.elementSlug || model?.key || "");
      return String(cardFaceState.get(cardKey) || "front").toLowerCase() === "back" ? "back" : "front";
    };
    const cardPinData = firstTheaterGeometryModule?.cardPinData || (() => ({ mode: "overlay", worldX: Number.NaN, worldY: Number.NaN, screenX: Number.NaN, screenY: Number.NaN }));
    const cardDisplayMode = firstTheaterGeometryModule?.cardDisplayMode || (() => "overlay");
    const stagePointForWorldPoint = (point) => firstTheaterGeometryModule?.stagePointForWorldPoint?.(point, stageCamera) || { x: Number(point?.x || 0), y: Number(point?.y || 0) };
    const worldPointForStagePoint = (point) => firstTheaterGeometryModule?.worldPointForStagePoint?.(point, stageCamera) || { x: Number(point?.x || 0), y: Number(point?.y || 0) };
    const currentDisplayedPointForModel = (model) => firstTheaterGeometryModule?.currentDisplayedPointForModel?.(model, {
      camera: stageCamera,
      getPlayableBounds,
      toStagePoint,
      stagePointForWorldPoint,
      worldPointForStagePoint,
      size: getStageSize(),
    }) || { x: 0, y: 0 };
    const normalizeOverlayPoint = (point) => firstTheaterGeometryModule?.normalizeOverlayPoint?.(point, getPlayableBounds()) || { screen_x: 0.5, screen_y: 0.5 };
    const clampOverlayPoint = (point) => firstTheaterGeometryModule?.clampOverlayPoint?.(point, getPlayableBounds()) || { x: Number(point?.x || 0), y: Number(point?.y || 0) };
    const offsetPoint = firstTheaterGeometryModule?.offsetPoint || ((point, dx = 24, dy = 16) => ({ x: Math.round(Number(point?.x || 0) + Number(dx || 0)), y: Math.round(Number(point?.y || 0) + Number(dy || 0)) }));
    const tokenSnapModeForModel = (model) => firstTheaterGeometryModule?.tokenSnapModeForModel?.(model, currentVenueGridConfig) || "free";
    const tokenLayerForModel = firstTheaterGeometryModule?.tokenLayerForModel || (() => "public");
    const tokenScaleForModel = firstTheaterGeometryModule?.tokenScaleForModel || (() => 100);
    const tokenDisplayNameForModel = firstTheaterLogicModule?.tokenDisplayNameForModel || (() => "Token");
    const tokenStatusPercentForModel = firstTheaterLogicModule?.tokenStatusPercentForModel || (() => null);
    const tokenFootprintForModel = firstTheaterGeometryModule?.tokenFootprintForModel || (() => ({ width: 1, height: 1 }));
    const tokenPlacementBaseSize = (model) => firstTheaterGeometryModule?.tokenPlacementBaseSize?.(model, currentVenueGridConfig) || 64;
    const tokenDisplaySizeForModel = (model) => firstTheaterGeometryModule?.tokenDisplaySizeForModel?.(model, currentVenueGridConfig) || { width: 64, height: 64 };
    const tokenPlacementPointForModel = (model) => currentDisplayedPointForModel(model);
    const tokenPlacementPointForCreate = (point, model, forcedSnapMode = "") => firstTheaterGeometryModule?.tokenPlacementPointForCreate?.(point, model, {
      forcedSnapMode,
      gridConfig: currentVenueGridConfig,
      currentVenueMapBounds,
      getPlayableBounds,
      snapPoint: firstTheaterGeometryModule?.snapPoint,
    }) || point;
    const cardDuplicatePlacementForModel = (model) => firstTheaterGeometryModule?.cardDuplicatePlacementForModel?.(model, {
      camera: stageCamera,
      getPlayableBounds,
      toStagePoint,
      stagePointForWorldPoint,
      worldPointForStagePoint,
      size: getStageSize(),
      bounds: getPlayableBounds(),
    }) || null;
    const tokenMoveTargetForModel = (model, targetPoint) => firstTheaterGeometryModule?.tokenMoveTargetForModel?.(model, targetPoint, {
      gridConfig: currentVenueGridConfig,
      currentVenueMapBounds,
      getPlayableBounds,
      snapPoint: firstTheaterGeometryModule?.snapPoint,
      stagePlacementCandidate,
      lastStagePoint,
    }) || null;
    const tokenDuplicatePlacementForModel = (model) => firstTheaterGeometryModule?.tokenDuplicatePlacementForModel?.(model, {
      camera: stageCamera,
      getPlayableBounds,
      toStagePoint,
      stagePointForWorldPoint,
      worldPointForStagePoint,
      size: getStageSize(),
      gridConfig: currentVenueGridConfig,
      currentVenueMapBounds,
      snapPoint: firstTheaterGeometryModule?.snapPoint,
    }) || null;

    function updateLocalCardPinModel(matchModel, updater) {
      return updateLocalObjectModel(matchModel, (model) => {
        const data = model.source?.data || {};
        model.source = model.source || {};
        model.source.data = { ...data };
        updater(model);
      });
    }

    function updateTokenLocalModel(matchModel, updater) {
      if (!matchModel || typeof updater !== "function") return null;
      return updateLocalObjectModel(matchModel, (model) => {
        model.source = model.source || {};
        model.source.data = { ...(model.source.data || {}) };
        updater(model);
      });
    }

    function setLocalPositionOverrideForModel(model, position) {
      const key = String(model?.elementId || model?.elementSlug || model?.key || "");
      if (!key || !position) return null;
      const overridePos = {
        ...(model?.position || {}),
        x: Math.round(Number(position.x || 0)),
        y: Math.round(Number(position.y || 0)),
        order: Number(model?.position?.order ?? 0),
        frame: "top-left",
      };
      localPositionOverrides.set(key, overridePos);
      return overridePos;
    }

    function floatingCardStateForModel(model) {
      if (!dragState || !model || !updateObjectMatches(dragState.model, model)) {
        return null;
      }
      const point = dragState.floatingPoint || dragState.screenPoint || null;
      if (!point) {
        return null;
      }
      return {
        x: Math.round(Number(point.x || 0)),
        y: Math.round(Number(point.y || 0)),
      };
    }

    function truncateCardText(text, maxChars = 24) {
      const value = String(text || "").replace(/\s+/g, " ").trim();
      if (!value) return "";
      if (value.length <= maxChars) return value;
      return `${value.slice(0, Math.max(1, maxChars - 1)).trimEnd()}…`;
    }

    const canToggleLiveVisibility = firstTheaterLogicModule?.canToggleLiveVisibility || (() => false);
    const canToggleNameplate = firstTheaterLogicModule?.canToggleNameplate || (() => false);
    const canToggleLock = firstTheaterLogicModule?.canToggleLock || (() => false);
    const canTogglePinState = firstTheaterLogicModule?.canTogglePinState || (() => false);
    const cardEditorDraftFromModel = firstTheaterLogicModule?.cardEditorDraftFromModel || (() => ({}));
    const stageContextMenuModel = firstTheaterLogicModule?.stageContextMenuModel || (() => ({ key: "", kind: "stage" }));
    const resolveStageObjectActions = firstTheaterLogicModule?.resolveStageObjectActions || (() => []);

    function syncSelectedActions() {
      if (!selectedActions) return;
      selectedActions.innerHTML = "";
      if (!currentSelection) return;

      const actions = resolveStageObjectActions(currentSelection, {
        role: currentRole,
        canManageIndexCards: canManageIndexCards(currentRole),
        canManageStageTokens: canManageStageTokens(currentRole),
        canActorRevealHideStageObjects: canActorRevealHideStageObjects(currentRole, currentSnapshot?.venue?.config || {}),
        hasSelection: Boolean(currentSelection),
        gridConfig: currentVenueGridConfig,
        cardFaceForModel,
        cardDisplayMode,
        tokenScaleForModel,
        tokenSnapModeForModel,
        tokenLayerForModel,
      });
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
      return firstTheaterMapGridModule?.mapEditorDraftFromState?.(state, currentVenueMapAssetID || "", {
        displayMode: mapEditorDisplayMode?.value,
        fit: mapEditorFit?.value,
        scale: mapEditorScale?.value,
        cropX: mapEditorCropX?.value,
        cropY: mapEditorCropY?.value,
        safeMargin: mapEditorSafeMargin?.value,
      }) || {
        assetID: String(state.asset_id || currentVenueMapAssetID || ""),
        displayMode: String(state.display_mode || "theater"),
        fit: String(state.fit || "cover"),
        cropX: Number(state.crop_x ?? 0.5),
        cropY: Number(state.crop_y ?? 0.5),
        scale: Number(state.scale ?? 1),
        safeMargin: Number(state.safe_margin ?? 24),
      };
    }

    function setMapEditorStatus(text) {
      if (mapEditorStatus) {
        mapEditorStatus.textContent = text || "";
      }
    }

    function clampMapEditorPosition(left, top) {
      return firstTheaterMapGridModule?.clampPanelPosition?.(
        left,
        top,
        stageShell?.getBoundingClientRect?.(),
        mapEditorPanel?.getBoundingClientRect?.(),
        560,
        520,
      ) || { left: Math.round(left), top: Math.round(top) };
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
      return firstTheaterMapGridModule?.clampPanelPosition?.(
        left,
        top,
        stageShell?.getBoundingClientRect?.(),
        gridEditorPanel?.getBoundingClientRect?.(),
        420,
        480,
      ) || { left: Math.round(left), top: Math.round(top) };
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
            mapEditorStatus.textContent = `Selected ${asset.original_filename || asset.asset_id}. Saving to activate it...`;
          }
          void saveVenueMap().catch((error) => {
            console.warn("map asset selection save failed", error);
            setMapEditorStatus(error.message || String(error));
            setStageStatus(error.message || String(error));
          });
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
        const previousAssetID = String(currentVenueMapState?.asset_id || currentVenueMapAssetID || "");
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
        if (stageCamera && previousAssetID !== currentVenueMapAssetID) {
          resetCameraToFit(currentVenueMapAssetID);
        }
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
        resetCameraToFit("");
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
      const previousAssetID = String(currentVenueMapState?.asset_id || "");
      currentVenueMapState = result.data || null;
      currentVenueMapAssetID = String(currentVenueMapState?.asset_id || assetID);
      mapEditorOriginalState = currentVenueMapState ? { ...currentVenueMapState } : null;
      if (previousAssetID !== currentVenueMapAssetID) {
        resetCameraToFit(currentVenueMapAssetID);
      }
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
      resetCameraToFit("");
      void ensureVenueMapTexture();
      renderPixiScene();
      setStageStatus("First Theater map removed.");
      hideMapEditor();
    }

    function defaultGridConfig() {
      return firstTheaterMapGridModule?.defaultGridConfig?.() || {
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
      return firstTheaterMapGridModule?.gridEditorDraftFromUI?.(currentVenueGridConfig || defaultGridConfig(), {
        gridType: gridEditorType?.value,
        hexOrientation: gridEditorHexOrientation?.value,
        cellSize: gridEditorCellSize?.value,
        offsetX: gridEditorOffsetX?.value,
        offsetY: gridEditorOffsetY?.value,
        opacity: gridEditorOpacity?.value,
        lineWidth: gridEditorLineWidth?.value,
        lineStyle: gridEditorLineStyle?.value,
      }) || {
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

    function setGridEditorStatus(text) {
      if (gridEditorStatus) {
        gridEditorStatus.textContent = text || "";
      }
    }

    function syncGridEditorHexFieldVisibility() {
      if (!gridEditorHexOrientationField) return;
      gridEditorHexOrientationField.style.display = firstTheaterMapGridModule?.gridEditorHexFieldVisible?.(gridEditorType?.value) ? "" : "none";
    }

    function syncGridEditorVisibilityButton() {
      if (!gridEditorVisibility) return;
      gridEditorVisibility.textContent = firstTheaterMapGridModule?.gridEditorVisibilityLabel?.(currentVenueGridConfig) || "Hide Grid";
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
      renderVenueGridLayer(currentVenueMapBounds || getPlayableBounds());
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
        renderVenueGridLayer(currentVenueMapBounds || getPlayableBounds());
      }
      hideGridEditor();
    }

    function resetGridEditorDraft() {
      if (!window.confirm("Reset grid fields to default values? Click Save Grid to persist.")) return;
      currentVenueGridConfig = defaultGridConfig();
      gridEditorDirty = true;
      syncGridEditorWithState();
      renderVenueGridLayer(currentVenueMapBounds || getPlayableBounds());
      setGridEditorStatus("Grid reset to defaults. Save to persist.");
    }

    function toggleGridVisibilityDraft() {
      const draft = gridEditorDraftFromUI();
      draft.visible = !(currentVenueGridConfig ? currentVenueGridConfig.visible !== false : true);
      currentVenueGridConfig = draft;
      gridEditorDirty = true;
      syncGridEditorVisibilityButton();
      renderVenueGridLayer(currentVenueMapBounds || getPlayableBounds());
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

    const computeStagePlayableBounds = (width, height) => {
      const displayMode = String(currentVenueMapState?.display_mode || "theater").trim() === "fullscreen" ? "fullscreen" : "theater";
      if (displayMode === "fullscreen") {
        return { x: 0, y: 0, width, height };
      }
      return firstTheaterGeometryModule?.computeStagePlayableBounds?.(width, height) || { x: 0, y: 0, width, height };
    };

    function renderVenueGridLayer(bounds) {
      if (!gridLayer || !window.PIXI) return;
      const resolvedBounds = bounds || getPlayableBounds();
      window.VictoryPixiGrid?.render?.(gridLayer, currentVenueGridConfig, resolvedBounds);
    }

    function renderVenueMapLayer(width, height) {
      if (!mapLayer || !window.PIXI) return;
      mapLayer.removeChildren();
      mapLayer.mask = null;
      const state = currentVenueMapState;
      if (!state || !state.asset || !String(state.asset.content_url || "").trim()) {
        return computeStagePlayableBounds(width, height);
      }

      if (!venueMapTexture || venueMapTextureURL !== state.asset.content_url) {
        if (venueMapTextureFailedURL === state.asset.content_url) {
          return computeStagePlayableBounds(width, height);
        }
        ensureVenueMapTexture().then((texture) => {
          if (!texture) return;
          if (currentVenueMapState?.asset?.content_url === state.asset.content_url) {
            renderPixiScene();
          }
        });
        return computeStagePlayableBounds(width, height);
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
      const theaterCropY = Math.max(Number(state.crop_y ?? 0.5), 0.58);
      window.VictoryPixiStage?.fitSpriteToBounds?.(sprite, boundsWidth, boundsHeight, {
        fit: state.fit || "cover",
        x: boundsX,
        y: boundsY,
        cropX: Number(state.crop_x ?? 0.5),
        cropY: displayMode === "theater" ? theaterCropY : Number(state.crop_y ?? 0.5),
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

      const renderedWidth = Number(sprite.width || boundsWidth);
      const renderedHeight = Number(sprite.height || boundsHeight);
      return {
        x: Number(sprite.x || boundsX) - (renderedWidth / 2),
        y: Number(sprite.y || boundsY) - (renderedHeight / 2),
        width: renderedWidth,
        height: renderedHeight,
      };
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

    const stageScreenPointFromClient = (clientX, clientY) => firstTheaterGeometryModule?.stageScreenPointFromClient?.(clientX, clientY, stageHost?.getBoundingClientRect?.()) || {
      x: Number(clientX || 0),
      y: Number(clientY || 0),
    };

    const stagePointFromClient = (clientX, clientY) => {
      const point = stageScreenPointFromClient(clientX, clientY);
      if (stageCamera) {
        const worldPoint = stageCamera.screenToWorld?.(point.x, point.y);
        if (worldPoint) {
          return {
            x: Math.round(worldPoint.x),
            y: Math.round(worldPoint.y),
          };
        }
      }
      const size = getStageSize();
      return {
        x: Math.max(0, Math.min(Math.round(size.width), Math.round(point.x))),
        y: Math.max(0, Math.min(Math.round(size.height), Math.round(point.y))),
      };
    };

    function hitTestContextMenuTarget(screenPoint, worldPoint) {
      for (let index = currentObjects.length - 1; index >= 0; index -= 1) {
        const model = currentObjects[index];
        const node = currentNodeMap.get(model.key);
        const bounds = node?.container?.getBounds?.();
        const point = screenPoint;
        if (bounds?.contains?.(point.x, point.y)) {
          return model;
        }

        const localPoint = node?.container?.toLocal && window.PIXI
          ? node.container.toLocal(new PIXI.Point(point.x, point.y))
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
      if (actionRouter?.resolveContextMenuTargetFromClient) {
        return actionRouter.resolveContextMenuTargetFromClient(clientX, clientY);
      }
      const screenPoint = stageScreenPointFromClient(clientX, clientY);
      const stagePoint = stagePointFromClient(clientX, clientY);
      const objectModel = hitTestContextMenuTarget(screenPoint, stagePoint) || stageContextMenuModel();
      return { objectModel, stagePoint, screenPoint };
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
      if (actionRouter?.handleNativeStageContextMenu) {
        return actionRouter.handleNativeStageContextMenu(event);
      }
      if (!event) return;
      if (wasContextMenuHandled(event)) return;
      const target = event.target;
      const composedPath = typeof event.composedPath === "function" ? event.composedPath() : [];
      if (cameraControls?.contains?.(target)) {
        return;
      }
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
      if (actionRouter?.openResolvedContextMenu) {
        return actionRouter.openResolvedContextMenu(event);
      }
      if (!event || !contextMenu || !pixiApp) return;
      const clientPoint = eventClientPoint(event);
      const resolved = resolveContextMenuTargetFromClient(clientPoint.clientX, clientPoint.clientY);
      const nativeEvent = event?.data?.originalEvent || event?.originalEvent || event;
      markContextMenuHandled(nativeEvent);
      openContextMenu(nativeEvent, resolved.objectModel);
    }

    function performStageObjectAction(action, objectModel) {
      if (actionRouter?.performStageObjectAction) {
        return actionRouter.performStageObjectAction(action, objectModel);
      }
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

      if (action === "pin" || action === "unpin") {
        if (state.locked) {
          setStageStatus(`${objectModel.label} is locked.`);
          closeContextMenu();
          return;
        }
        const currentPoint = currentDisplayedPointForModel(objectModel);
        const pinMode = action === "pin" ? "world" : "overlay";
        const worldPoint = pinMode === "world"
          ? worldPointForStagePoint(currentPoint)
          : null;
        const overlayPoint = pinMode === "overlay" ? normalizeOverlayPoint(currentPoint) : null;
        const payload = {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          front_text: objectModel.frontText || "",
          back_text: objectModel.backText || "",
          color: objectModel.color || "#d9c7a6",
          pin_mode: pinMode,
        };
        if (pinMode === "world") {
          payload.world_x = worldPoint.x;
          payload.world_y = worldPoint.y;
          payload.screen_x = null;
          payload.screen_y = null;
        } else {
          payload.world_x = null;
          payload.world_y = null;
          payload.screen_x = overlayPoint.screen_x;
          payload.screen_y = overlayPoint.screen_y;
        }
        const sent = sendAction("update/index_card", payload);
        const placementSent = sendAction("act/place_element", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
          x: currentPoint.x,
          y: currentPoint.y,
          order: Number(objectModel.position?.order ?? 0),
        });
        if (sent) {
          updateLocalCardPinModel(objectModel, (model) => {
            model.source.data = {
              ...(model.source.data || {}),
              pin_mode: pinMode,
              world_x: pinMode === "world" ? worldPoint.x : null,
              world_y: pinMode === "world" ? worldPoint.y : null,
              screen_x: pinMode === "overlay" ? overlayPoint.screen_x : null,
              screen_y: pinMode === "overlay" ? overlayPoint.screen_y : null,
            };
            model.position = setLocalPositionOverrideForModel(model, currentPoint) || model.position;
          });
        }
        setStageStatus(sent && placementSent ? `${action === "pin" ? "Attached" : "Pinned"} ${objectModel.label}.` : "Socket unavailable.");
        setMovementReport(sent && placementSent ? `${action === "pin" ? "Attached" : "Pinned"} ${objectModel.label}.` : "Socket unavailable.");
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

      if (action === "add-token") {
        openTokenPicker("create", null);
        closeContextMenu();
        return;
      }

      if (action === "clear") {
        selectObject(null, "Selection cleared.");
        hideCardEditor();
        hideMapEditor();
        hideGridEditor();
        hideTokenEditor();
        closeTokenPicker();
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

      if (action === "scale" && isTokenObject(objectModel)) {
        openTokenEditor(objectModel);
        closeContextMenu();
        return;
      }

      if ((action === "snap-to-grid" || action === "free-placement") && isTokenObject(objectModel)) {
        const snapMode = action === "snap-to-grid" ? "grid" : "free";
        const snappedPoint = snapMode === "grid"
          ? tokenPlacementPointForCreate(currentDisplayedPointForModel(objectModel), objectModel, snapMode)
          : currentDisplayedPointForModel(objectModel);
        const sent = sendAction("update/token", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
          x: Number(snappedPoint?.x ?? objectModel.position?.x ?? 0),
          y: Number(snappedPoint?.y ?? objectModel.position?.y ?? 0),
          order: Number(objectModel.position?.order ?? 0),
          snap_mode: snapMode,
        });
        if (sent) {
          updateTokenLocalModel(objectModel, (model) => {
            model.snapMode = snapMode;
            model.gridRelative = snapMode === "grid";
            if (snapMode === "grid" && snappedPoint) {
              model.position = {
                ...(model.position || {}),
                x: Math.round(Number(snappedPoint.x || 0)),
                y: Math.round(Number(snappedPoint.y || 0)),
                order: Number(model.position?.order ?? 0),
                frame: "center",
              };
            }
          });
        }
        setStageStatus(sent ? `${objectModel.label} set to ${snapMode === "grid" ? "Snap to Grid" : "Free Placement"}.` : "Socket unavailable.");
        closeContextMenu();
        return;
      }

      if ((action === "move-director-layer" || action === "move-public-layer") && isTokenObject(objectModel)) {
        const tokenLayer = action === "move-director-layer" ? "director" : "public";
        const sent = sendAction("update/token", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
          x: Number(objectModel.position?.x ?? 0),
          y: Number(objectModel.position?.y ?? 0),
          order: Number(objectModel.position?.order ?? 0),
          token_layer: tokenLayer,
        });
        if (sent) {
          updateTokenLocalModel(objectModel, (model) => {
            model.tokenLayer = tokenLayer;
          });
        }
        setStageStatus(sent ? `${objectModel.label} moved to the ${tokenLayer === "director" ? "Director" : "Public"} layer.` : "Socket unavailable.");
        closeContextMenu();
        return;
      }

      if (action === "replace-asset" && isTokenObject(objectModel)) {
        openTokenPicker("replace", objectModel);
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
            model.state = { ...(model.state || {}), nameplate_visible: visible, nameplateVisible: visible };
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
        const moveTarget = isTokenObject(objectModel)
          ? tokenMoveTargetForModel(objectModel, point)
          : cardMoveTargetForModel(objectModel, point);
        if (!moveTarget) {
          setStageStatus("Move here needs a stage point.");
          closeContextMenu();
          return;
        }
        const sent = isTokenObject(objectModel)
          ? sendAction("update/token", {
              element_id: objectModel.elementId || "",
              element_slug: objectModel.elementSlug || "",
              venue_slug: "the-cave",
              layer: "stage",
              x: point.x,
              y: point.y,
              order: Number(objectModel.position?.order ?? 0),
              snap_mode: moveTarget.snapMode,
              token_layer: tokenLayerForModel(objectModel),
              scale: tokenScaleForModel(objectModel),
            })
          : sendAction("update/index_card", {
              element_id: objectModel.elementId || "",
              element_slug: objectModel.elementSlug || "",
              front_text: objectModel.frontText || "",
              back_text: objectModel.backText || "",
              color: objectModel.color || "#d9c7a6",
              pin_mode: moveTarget.pinMode,
              world_x: moveTarget.world_x,
              world_y: moveTarget.world_y,
              screen_x: moveTarget.screen_x,
              screen_y: moveTarget.screen_y,
            });
        const placementSent = isTokenObject(objectModel)
          ? true
          : sendAction("act/place_element", {
              element_id: objectModel.elementId || "",
              element_slug: objectModel.elementSlug || "",
              venue_slug: "the-cave",
              layer: "stage",
              x: point.x,
              y: point.y,
              order: Number(objectModel.position?.order ?? 0),
            });
        if (sent) {
          updateLocalCardPinModel(objectModel, (model) => {
            if (isTokenObject(model)) {
              model.source.data = {
                ...(model.source.data || {}),
                snap_mode: moveTarget.snapMode,
                token_layer: tokenLayerForModel(model),
                scale: tokenScaleForModel(model),
              };
              model.position = {
                ...(model.position || {}),
                x: point.x,
                y: point.y,
                order: Number(model.position?.order ?? 0),
                frame: "center",
              };
            } else {
              model.source.data = {
                ...(model.source.data || {}),
                pin_mode: moveTarget.pinMode,
                world_x: moveTarget.world_x,
                world_y: moveTarget.world_y,
                screen_x: moveTarget.screen_x,
                screen_y: moveTarget.screen_y,
              };
              model.position = setLocalPositionOverrideForModel(model, point) || model.position;
            }
          });
        }
        const moveLabel = isTokenObject(objectModel)
          ? `${moveTarget.snapMode === "grid" ? "grid" : "free"} x ${point.x}, y ${point.y}`
          : moveTarget.pinMode === "world"
            ? `world x ${moveTarget.world_x}, y ${moveTarget.world_y}`
            : `overlay x ${moveTarget.screen_x}, y ${moveTarget.screen_y}`;
        const moveOk = isTokenObject(objectModel) ? sent : (sent && placementSent);
        setMovementLine(moveOk ? `Move sent for ${objectModel.label} to ${moveLabel}.` : "Socket unavailable.");
        setStageStatus(moveOk ? `Moving ${objectModel.label} to ${moveLabel}.` : "Socket unavailable.");
        closeContextMenu();
        return;
      }

      if (action === "duplicate") {
        const duplicatePlacement = isTokenObject(objectModel)
          ? tokenDuplicatePlacementForModel(objectModel)
          : cardDuplicatePlacementForModel(objectModel);
        const duplicatePinMode = isTokenObject(objectModel)
          ? (tokenSnapModeForModel(objectModel) === "grid" ? "world" : "overlay")
          : duplicatePlacement.pinMode;
        const sent = sendAction("act/duplicate_element", {
          element_id: objectModel.elementId || "",
          element_slug: objectModel.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
          x: duplicatePlacement.x,
          y: duplicatePlacement.y,
          order: Number(objectModel.position?.order ?? 0),
          pin_mode: duplicatePinMode,
          world_x: duplicatePlacement.world_x,
          world_y: duplicatePlacement.world_y,
          screen_x: duplicatePlacement.screen_x,
          screen_y: duplicatePlacement.screen_y,
        });
        const duplicateLabel = isTokenObject(objectModel)
          ? `x ${duplicatePlacement.x}, y ${duplicatePlacement.y}`
          : duplicatePlacement.pinMode === "world"
            ? `world x ${duplicatePlacement.world_x}, y ${duplicatePlacement.world_y}`
            : `overlay x ${duplicatePlacement.screen_x}, y ${duplicatePlacement.screen_y}`;
        setMovementLine(sent ? `Duplicate sent for ${objectModel.label} to ${duplicateLabel}.` : "Socket unavailable.");
        setStageStatus(sent ? `Duplicating ${objectModel.label} to ${duplicateLabel}.` : "Socket unavailable.");
        if (sent) {
          if (isTokenObject(objectModel)) {
            updateTokenLocalModel(objectModel, (model) => {
              model.position = {
                ...(model.position || {}),
                x: duplicatePlacement.x,
                y: duplicatePlacement.y,
                order: Number(model.position?.order ?? 0),
                frame: "center",
              };
            });
          } else {
            setLocalPositionOverrideForModel(objectModel, { x: duplicatePlacement.x, y: duplicatePlacement.y });
          }
        }
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
          currentObjects = currentObjects.filter((item) => !updateObjectMatches(item, objectModel));
          if (currentSelection && updateObjectMatches(currentSelection, objectModel)) {
            currentSelection = null;
          }
          if (tokenEditorTargetKey && updateObjectMatches({ key: tokenEditorTargetKey }, objectModel)) {
            hideTokenEditor();
          }
          if (cardEditorTargetKey && updateObjectMatches({ key: cardEditorTargetKey }, objectModel)) {
            cardEditorTargetKey = "";
          }
          updateStatusSummary();
          updateShellTargetPresentation();
          renderPixiScene();
          syncSelectedActions();
          syncCardEditorWithSelection();
          syncTokenEditorWithSelection();
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
      if (actionRouter?.openContextMenu) {
        return actionRouter.openContextMenu(event, objectModel);
      }
      if (!contextMenu || !objectModel) return;
      contextMenuTarget = objectModel;
      const clientPoint = eventClientPoint(event);
      const stagePoint = stagePlacementFromEvent(event);
      const screenPoint = stageScreenPointFromClient(clientPoint.clientX, clientPoint.clientY);
      if (objectModel.kind === "stage" || objectModel.kind === "fire" || (objectModel.live && isLiveStageObject(objectModel))) {
        stagePlacementCandidate = stagePoint;
        stagePlacementScreenCandidate = screenPoint;
        if (stagePlacementCandidate) {
          updatePointerReadout(clientPoint.clientX || 0, clientPoint.clientY || 0, stagePlacementCandidate, `${objectModel.kind === "fire" ? "Fire" : "Stage"} menu`);
        }
      }

      const items = resolveStageObjectActions(objectModel, {
        role: currentRole,
        canManageIndexCards: canManageIndexCards(currentRole),
        canManageStageTokens: canManageStageTokens(currentRole),
        canActorRevealHideStageObjects: canActorRevealHideStageObjects(currentRole, currentSnapshot?.venue?.config || {}),
        hasSelection: Boolean(currentSelection),
        gridConfig: currentVenueGridConfig,
        cardFaceForModel,
        cardDisplayMode,
        tokenScaleForModel,
        tokenSnapModeForModel,
        tokenLayerForModel,
      });
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
      suppressStageContextMenu = true;
      window.setTimeout(() => {
        suppressStageContextMenu = false;
      }, 0);
    }

    function getStageSize() {
      const rect = stageShell?.getBoundingClientRect();
      if (!rect || rect.width <= 0 || rect.height <= 0) {
        return { width: 960, height: 640 };
      }
      return { width: rect.width, height: rect.height };
    }

    const toStagePoint = (model, size, options = {}) => firstTheaterGeometryModule?.toStagePoint?.(model, size, options) || {
      x: 0,
      y: 0,
    };

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
        const pin = cardDisplayMode(model);
        lines.push(`Pin: ${pin}`);
        lines.push(`Front: ${model.frontText || "(blank)"}`);
        lines.push(`Back: ${model.backText || "(blank)"}`);
        lines.push(`Color: ${model.color || "#d9c7a6"}`);
      } else if (isTokenObject(model)) {
        const footprint = tokenFootprintForModel(model);
        const displaySize = tokenDisplaySizeForModel(model);
        lines.push(`Asset: ${model.assetName || model.source?.data?.asset_name || "(unknown)"}`);
        lines.push(`Shape: ${model.assetShape || model.source?.data?.asset_shape || "circle"}`);
        lines.push(`Footprint: ${footprint.width} x ${footprint.height}`);
        lines.push(`Scale: ${tokenScaleForModel(model)}%`);
        lines.push(`Placement: ${tokenSnapModeForModel(model)}`);
        lines.push(`Layer: ${tokenLayerForModel(model)}`);
        lines.push(`Display: ${displaySize.width} x ${displaySize.height}px`);
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
      syncTokenEditorWithSelection();

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
      const kind = String(element?.context_class || data.context_class || element?.element_type || "").trim().toLowerCase();
      const inferredKind = (kind === "token" || String(data.asset_id || "").trim() || String(data.asset_content_url || "").trim() || String(data.asset_thumbnail_url || "").trim()) ? "token" : "card";
      const assetID = String(data.asset_id || "");
      return {
        key: `live:${String(element?.element_id || element?.slug || index)}`,
        kind: inferredKind,
        live: true,
        elementId: String(element?.element_id || ""),
        elementSlug: String(element?.slug || ""),
        elementType: String(element?.element_type || ""),
        contextClass: String(element?.context_class || data.context_class || inferredKind),
        label: String(element?.name || data.asset_name || data.front_text || element?.slug || (inferredKind === "token" ? "Token" : "Index card")),
        frontText: String(data.front_text || element?.name || "").trim(),
        backText: String(data.back_text || "").trim(),
        color: String(data.color || "#d9c7a6").trim() || "#d9c7a6",
        position: element?.position || data.position || {},
        state,
        visibility,
        assetID,
        assetName: String(data.asset_name || element?.name || ""),
        assetShape: String(data.asset_shape || data.shape || "circle"),
        assetContentURL: String(data.asset_content_url || ""),
        assetThumbnailURL: String(data.asset_thumbnail_url || ""),
        defaultGridWidth: Number(data.default_grid_width || 1),
        defaultGridHeight: Number(data.default_grid_height || 1),
        snapMode: String(data.snap_mode || ""),
        tokenLayer: String(data.token_layer || "public"),
        scale: Number(data.scale || 100),
        gridRelative: Boolean(data.grid_relative ?? false),
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
      const stageTokens = elements
        .filter((element) => String(element?.element_type || "").toLowerCase() === "token" || String(element?.context_class || element?.data?.context_class || "").toLowerCase() === "token")
        .filter((element) => String(element?.surface || element?.data?.surface || "stage").toLowerCase() === "stage")
        .sort((a, b) => {
          const layerA = String(a?.data?.token_layer || "public");
          const layerB = String(b?.data?.token_layer || "public");
          if (layerA !== layerB) {
            return layerA === "public" ? -1 : 1;
          }
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
      if (stageTokens.length > 0) {
        out.push(...stageTokens);
      }
      if (stageCards.length > 0) {
        out.push(...stageCards);
      }

      return out;
    }

    const clearSceneNodes = () => sceneNodeFactory?.clearSceneNodes?.();
    const refreshNodeSelection = () => sceneNodeFactory?.refreshNodeSelection?.();
    const makeCardNode = (model) => sceneNodeFactory?.makeCardNode?.(model) || null;
    const makeFireNode = (model) => sceneNodeFactory?.makeFireNode?.(model) || null;
    const makeTokenNode = (model) => sceneNodeFactory?.makeTokenNode?.(model) || null;
    const makePlaceholderNode = (model) => (isTokenObject(model) ? makeTokenNode(model) : makeCardNode(model));

    function createIndexCardFromMenu() {
      if (!canManageIndexCards(currentRole)) {
        setStageStatus("Only producers and directors can create cards.");
        return;
      }

      if (!stagePlacementCandidate) {
        setStageStatus("No stage placement point available.");
        return;
      }

      pendingStageCardPlacement = stagePlacementScreenCandidate || null;
      stagePlacementCandidate = null;
      stagePlacementScreenCandidate = null;
      const placementLabel = `x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}`;
      const socket = socketController?.getWebSocket?.() || ws;
      const queued = !!(socket && socket.readyState === WebSocket.CONNECTING && currentSessionId);

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

      setStageStatus(queued ? `Index card queued. Attaching to screen at ${placementLabel} when the connection opens...` : `Index card created. Attaching to screen at ${placementLabel}...`);
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
      const createdCardModel = {
        elementId,
        elementSlug,
        key: `live:${elementId || elementSlug}`,
      };
      createdCardModel.position = setLocalPositionOverrideForModel(createdCardModel, pendingStageCardPlacement) || createdCardModel.position;
      if (smokeMode) {
        setRecentPlacementMarker(pendingStageCardPlacement, createdCard?.name || createdCard?.data?.front_text || "Index card");
      } else {
        setRecentPlacementMarker(null);
      }

      const normalizedPlacement = normalizeOverlayPoint(pendingStageCardPlacement);
      const updated = sendAction("update/index_card", {
        element_id: elementId,
        element_slug: elementSlug,
        front_text: createdCard?.data?.front_text || createdCard?.name || "New Index Card",
        back_text: String(createdCard?.data?.back_text || ""),
        color: String(createdCard?.data?.color || "#d9c7a6"),
        pin_mode: "overlay",
        screen_x: normalizedPlacement.screen_x,
        screen_y: normalizedPlacement.screen_y,
        world_x: null,
        world_y: null,
      });

      if (!updated) {
        setStageStatus(`Card created but the screen attachment could not be sent from x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}.`);
        pendingStageCardPlacement = null;
        closeContextMenu();
        return;
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
        setStageStatus(`Card created but the stage placement could not be sent from x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}.`);
        if (smokeMode) {
          setSmokeLine(`Create succeeded, placement failed at ${pendingStageCardPlacement.x}, ${pendingStageCardPlacement.y}.`);
        }
        pendingStageCardPlacement = null;
        closeContextMenu();
        return;
      }

      setStageStatus(`Index card placed at x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}.`);
      setMovementReport(`Index card placed at x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}.`);
      if (smokeMode) {
        setSmokeLine(`Placed smoke card at ${pendingStageCardPlacement.x}, ${pendingStageCardPlacement.y}.`);
      }
      pendingStageCardPlacement = null;
    }

    function tokenPickerFilteredAssets() {
      const search = String(tokenPickerState.search || "").trim().toLowerCase();
      const shape = String(tokenPickerState.filterShape || "all").trim().toLowerCase();
      return warehouseTokenAssets.filter((asset) => {
        const assetShape = String(asset.shape || asset.asset_shape || "").trim().toLowerCase();
        const haystack = [
          asset.name || asset.asset_name,
          asset.original_filename,
          asset.asset_type,
          asset.shape || asset.asset_shape,
          asset.status,
          asset.source_mime || asset.sourceMime,
          asset.sniffed_mime || asset.sniffedMime,
        ].join(" ").toLowerCase();
        if (shape !== "all" && assetShape !== shape) {
          return false;
        }
        if (search && !haystack.includes(search)) {
          return false;
        }
        return true;
      });
    }

    function tokenPickerPreviewForAsset(asset) {
      if (!tokenPickerPreview) return;
      if (warehouseTokenAssetPreviewURL && warehouseTokenAssetPreviewURL.startsWith("blob:")) {
        try {
          URL.revokeObjectURL(warehouseTokenAssetPreviewURL);
        } catch (error) {
          console.warn("token preview revoke failed", error);
        }
      }
      warehouseTokenAssetPreviewURL = String(asset?.thumbnail_url || asset?.content_url || "").trim();
      tokenPickerPreview.src = warehouseTokenAssetPreviewURL || "";
      if (tokenPickerPreviewBadge) {
        const dims = `${Number(asset?.default_grid_width || 1)} x ${Number(asset?.default_grid_height || 1)}`;
        tokenPickerPreviewBadge.textContent = asset
          ? `${asset.name || asset.id || "Token"} · ${asset.shape || "circle"} · ${dims}`
          : "No token selected";
      }
    }

    function renderTokenPickerList() {
      if (!tokenPickerList) return;
      const assets = tokenPickerFilteredAssets();
      tokenPickerList.innerHTML = "";
      if (!assets.length) {
        const empty = document.createElement("div");
        empty.className = "token-picker-status";
        empty.textContent = "No matching active token assets.";
        tokenPickerList.appendChild(empty);
        tokenPickerPreviewForAsset(null);
        return;
      }

      if (!tokenPickerState.selectedAssetID || !assets.some((asset) => asset.id === tokenPickerState.selectedAssetID)) {
        tokenPickerState.selectedAssetID = String(assets[0].id || assets[0].asset_id || "");
      }

      const selected = assets.find((asset) => asset.id === tokenPickerState.selectedAssetID) || assets[0] || null;
      if (selected) {
        tokenPickerPreviewForAsset(selected);
      }

      assets.forEach((asset) => {
        const assetID = String(asset.id || asset.asset_id || "").trim();
        const assetName = String(asset.name || asset.asset_name || asset.original_filename || assetID || "Token asset");
        const assetShape = String(asset.shape || asset.asset_shape || "circle");
        const sourceMime = String(asset.source_mime || asset.sourceMime || asset.sniffed_mime || asset.sniffedMime || "");
        const thumbURL = String(asset.thumbnail_url || asset.thumbnailURL || asset.content_url || asset.contentURL || "");
        const button = document.createElement("button");
        button.type = "button";
        button.className = "token-picker-card";
        if (assetID === tokenPickerState.selectedAssetID) {
          button.classList.add("is-selected");
        }
        const bytes = formatByteSize(asset.stored_bytes || asset.byte_size || 0);
        const dims = `${asset.default_grid_width || asset.defaultGridWidth || 1} x ${asset.default_grid_height || asset.defaultGridHeight || 1}`;
        button.innerHTML = `
          <img src="${escapeHtml(thumbURL)}" alt="${escapeHtml(assetName)}" />
          <div class="token-picker-meta">
            <strong>${escapeHtml(assetName)}</strong>
            <small>${escapeHtml(assetShape)} · ${escapeHtml(dims)} · ${escapeHtml(bytes)}</small>
            <small>${escapeHtml(sourceMime)}</small>
          </div>
        `;
        button.addEventListener("click", () => {
          tokenPickerState.selectedAssetID = assetID;
          tokenPickerState.search = String(tokenPickerSearch?.value || tokenPickerState.search || "");
          renderTokenPickerList();
          const target = {
            ...asset,
            id: assetID,
            asset_id: assetID,
            name: assetName,
            shape: assetShape,
            thumbnail_url: thumbURL,
            content_url: String(asset.content_url || asset.contentURL || ""),
          };
          if (tokenPickerState.mode === "replace" && tokenPickerState.replaceTargetKey) {
            placeTokenAsset(target, tokenPickerState.placementPoint, tokenPickerState.replaceTargetKey, true);
          } else {
            placeTokenAsset(target, tokenPickerState.placementPoint, "", false);
          }
        });
        tokenPickerList.appendChild(button);
      });
    }

    async function refreshWarehouseTokenAssets() {
      if (!canManageStageTokens(currentRole)) {
        warehouseTokenAssets = [];
        renderTokenPickerList();
        return;
      }
      try {
        if (tokenPickerStatus) {
          tokenPickerStatus.textContent = "Loading active token assets...";
        }
        const params = new URLSearchParams({
          asset_type: "token",
          status: "active",
          search: String(tokenPickerSearch?.value || "").trim(),
        });
        const response = await fetch(`/api/warehouse/assets?${params.toString()}`, {
          credentials: "include",
          cache: "no-store",
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok) {
          throw new Error(payload?.error || `HTTP ${response.status}`);
        }
        warehouseTokenAssets = Array.isArray(payload.data) ? payload.data : [];
        if (tokenPickerStatus) {
          tokenPickerStatus.textContent = warehouseTokenAssets.length
            ? `${warehouseTokenAssets.length} token asset${warehouseTokenAssets.length === 1 ? "" : "s"} loaded.`
            : "No active token assets found.";
        }
        renderTokenPickerList();
      } catch (error) {
        console.warn("refreshWarehouseTokenAssets failed", error);
        warehouseTokenAssets = [];
        if (tokenPickerStatus) {
          tokenPickerStatus.textContent = error.message || String(error);
        }
        renderTokenPickerList();
      }
    }

    function openTokenPicker(mode = "create", objectModel = null) {
      if (!tokenPickerPanel) return;
      if (!canManageStageTokens(currentRole)) {
        setStageStatus("Only producers and directors can place First Theater tokens.");
        return;
      }
      hideCardEditor();
      hideMapEditor();
      hideGridEditor();
      hideTokenEditor();
        tokenPickerState = {
        open: true,
        mode: mode === "replace" ? "replace" : "create",
        selectedAssetID: "",
        filterShape: "all",
        search: "",
        placementPoint: stagePlacementCandidate ? { ...stagePlacementCandidate } : (lastStagePoint ? { ...lastStagePoint } : null),
        placementScreenPoint: stagePlacementScreenCandidate ? { ...stagePlacementScreenCandidate } : null,
        replaceTargetKey: objectModel?.key || "",
        replaceTargetElementID: objectModel?.elementId || "",
        replaceTargetElementSlug: objectModel?.elementSlug || "",
        replaceTargetScale: objectModel ? tokenScaleForModel(objectModel) : 100,
        replaceTargetSnapMode: objectModel ? tokenSnapModeForModel(objectModel) : (currentVenueGridConfig && currentVenueGridConfig.grid_type !== "none" ? "grid" : "free"),
        replaceTargetTokenLayer: objectModel ? tokenLayerForModel(objectModel) : "public",
      };
      if (tokenPickerPanel.hidden) {
        setTokenPickerPosition(16, 220);
      }
      tokenPickerPanel.hidden = false;
      if (tokenPickerSearch) tokenPickerSearch.value = "";
      if (tokenPickerShape) tokenPickerShape.value = "all";
      if (tokenPickerStatus) {
        tokenPickerStatus.textContent = "Choose an active Warehouse token asset.";
      }
      void refreshWarehouseTokenAssets();
      tokenPickerSearch?.focus?.({ preventScroll: true });
    }

    function closeTokenPicker() {
      tokenPickerState.open = false;
      tokenPickerState.replaceTargetKey = "";
      tokenPickerState.replaceTargetElementID = "";
      tokenPickerState.replaceTargetElementSlug = "";
      tokenPickerState.replaceTargetScale = 100;
      tokenPickerState.replaceTargetSnapMode = "grid";
      tokenPickerState.replaceTargetTokenLayer = "public";
      tokenPickerState.placementPoint = null;
      tokenPickerState.placementScreenPoint = null;
      if (tokenPickerPanel) {
        tokenPickerPanel.hidden = true;
      }
      if (warehouseTokenAssetPreviewURL) {
        warehouseTokenAssetPreviewURL = "";
      }
      if (tokenPickerStatus) {
        tokenPickerStatus.textContent = "Choose a reusable Warehouse token.";
      }
    }

    function placeTokenAsset(asset, point, replaceTargetKey = "", replacing = false) {
      if (!asset) return;
      const placementPoint = point || tokenPickerState.placementPoint || lastStagePoint || null;
      if (!placementPoint) {
        setStageStatus("No stage placement point available.");
        return;
      }
      const snapMode = currentVenueGridConfig && currentVenueGridConfig.grid_type !== "none" ? "grid" : "free";
      const snapped = tokenPlacementPointForCreate(placementPoint, { source: { data: { snap_mode: snapMode } } }, snapMode);
      const targetPayload = {
        asset_id: String(asset.id || "").trim(),
        venue_slug: "the-cave",
        layer: "stage",
        x: snapped.x,
        y: snapped.y,
        order: 0,
        snap_mode: replacing ? (tokenPickerState.replaceTargetSnapMode || snapMode) : snapMode,
        token_layer: replacing ? (tokenPickerState.replaceTargetTokenLayer || "public") : "public",
        scale: replacing ? (tokenPickerState.replaceTargetScale || 100) : 100,
      };
      const actionType = replacing ? "update/token" : "create/token";
      const sent = sendAction(actionType, replacing && replaceTargetKey ? {
        element_id: tokenPickerState.replaceTargetElementID || replaceTargetKey || "",
        element_slug: tokenPickerState.replaceTargetElementSlug || "",
        ...targetPayload,
      } : targetPayload);
      if (sent) {
        if (tokenPickerStatus) {
          tokenPickerStatus.textContent = replacing ? `Replacing token asset with ${asset.name || asset.id}.` : `Placing ${asset.name || asset.id}.`;
        }
        setStageStatus(replacing ? `Replacing token with ${asset.name || asset.id}.` : `Placing token ${asset.name || asset.id}.`);
        setMovementLine(replacing ? `Replace sent for ${asset.name || asset.id}.` : `Create sent for ${asset.name || asset.id}.`);
      } else {
        setStageStatus("Socket unavailable.");
      }
      closeTokenPicker();
    }

    function syncTokenEditorWithSelection() {
      if (!tokenEditorPanel || tokenEditorPanel.hidden) return;
      const target = currentSelection && isTokenObject(currentSelection)
        ? currentSelection
        : currentObjects.find((item) => item.key === tokenEditorTargetKey) || null;
      if (!target || !target.live || !isTokenObject(target)) {
        hideTokenEditor();
        return;
      }
      if (tokenEditorDirty) {
        return;
      }
      if (tokenEditorScale) tokenEditorScale.value = String(Math.round(tokenScaleForModel(target)));
      if (tokenEditorScaleValue) tokenEditorScaleValue.value = String(Math.round(tokenScaleForModel(target)));
      if (tokenEditorSnap) tokenEditorSnap.value = tokenSnapModeForModel(target);
      if (tokenEditorLayer) tokenEditorLayer.value = tokenLayerForModel(target);
      if (tokenEditorStatus) {
        tokenEditorStatus.textContent = `${target.label} ready to edit.`;
      }
    }

    function openTokenEditor(model = null) {
      const target = model || currentSelection;
      if (!target || !target.live || !isTokenObject(target)) {
        setStageStatus("Select a live token to edit it.");
        return;
      }
      if (objectState(target).locked) {
        setStageStatus("This token is locked.");
        return;
      }
      hideMapEditor();
      hideGridEditor();
      hideCardEditor();
      closeTokenPicker();
      tokenEditorTargetKey = target.key;
      tokenEditorDirty = false;
      if (tokenEditorPanel.hidden) {
        setTokenEditorPosition(16, 220);
      }
      tokenEditorPanel.hidden = false;
      syncTokenEditorWithSelection();
      tokenEditorScale?.focus?.({ preventScroll: true });
    }

    function hideTokenEditor() {
      tokenEditorDirty = false;
      tokenEditorTargetKey = "";
      if (tokenEditorPanel) {
        tokenEditorPanel.hidden = true;
      }
      if (tokenEditorStatus) {
        tokenEditorStatus.textContent = "Adjust the selected token's placement scale.";
      }
    }

    function saveTokenEditor() {
      const target = currentObjects.find((item) => item.key === tokenEditorTargetKey) || currentSelection;
      if (!target || !target.live || !isTokenObject(target)) {
        setStageStatus("Select a live token to save edits.");
        return;
      }
      const scale = clampNumber(tokenEditorScaleValue?.value ?? tokenEditorScale?.value ?? 100, 25, 500, 100);
      const snapMode = String(tokenEditorSnap?.value || tokenSnapModeForModel(target)).trim().toLowerCase() === "free" ? "free" : "grid";
      const tokenLayer = String(tokenEditorLayer?.value || tokenLayerForModel(target)).trim().toLowerCase() === "director" ? "director" : "public";
      const sent = sendAction("update/token", {
        element_id: target.elementId || "",
        element_slug: target.elementSlug || "",
        venue_slug: "the-cave",
        layer: "stage",
        x: Number(target.position?.x ?? 0),
        y: Number(target.position?.y ?? 0),
        order: Number(target.position?.order ?? 0),
        scale,
        snap_mode: snapMode,
        token_layer: tokenLayer,
      });
      if (!sent) {
        setStageStatus("Socket unavailable.");
        if (tokenEditorStatus) {
          tokenEditorStatus.textContent = "Save failed: socket unavailable.";
        }
        return;
      }
      updateTokenLocalModel(target, (model) => {
        model.scale = scale;
        model.snapMode = snapMode;
        model.gridRelative = snapMode === "grid";
        model.tokenLayer = tokenLayer;
      });
      tokenEditorDirty = false;
      setStageStatus(`Saved ${target.label}.`);
      if (tokenEditorStatus) {
        tokenEditorStatus.textContent = `${target.label} save sent.`;
      }
      setMovementLine(`update/token sent for ${target.label}.`);
      renderPixiScene();
    }

    if (tokenUi) {
      tokenPickerFilteredAssets = (...args) => tokenUi.tokenPickerFilteredAssets(...args);
      tokenPickerPreviewForAsset = (...args) => tokenUi.tokenPickerPreviewForAsset(...args);
      renderTokenPickerList = (...args) => tokenUi.renderTokenPickerList(...args);
      refreshWarehouseTokenAssets = (...args) => tokenUi.refreshWarehouseTokenAssets(...args);
      openTokenPicker = (...args) => tokenUi.openTokenPicker(...args);
      closeTokenPicker = (...args) => tokenUi.closeTokenPicker(...args);
      placeTokenAsset = (...args) => tokenUi.placeTokenAsset(...args);
      syncTokenEditorWithSelection = (...args) => tokenUi.syncTokenEditorWithSelection(...args);
      openTokenEditor = (...args) => tokenUi.openTokenEditor(...args);
      hideTokenEditor = (...args) => tokenUi.hideTokenEditor(...args);
      saveTokenEditor = (...args) => tokenUi.saveTokenEditor(...args);
    }

    function renderPixiScene() {
      if (!pixiApp || !sceneRoot || !backgroundLayer || !worldLayer || !mapLayer || !overlayObjectLayer) return;

      clearSceneNodes();

      const size = getStageSize();
      const width = Math.max(320, size.width);
      const height = Math.max(320, size.height);
      const playableBounds = computeStagePlayableBounds(width, height);
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
      const floorBandTop = Math.max(Math.round(height * 0.84), height - 120);
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
        stageFloor.drawRoundedRect(0, floorBandTop, width, height - floorBandTop, 20);
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
        stageLip.drawRoundedRect(0, floorBandTop, width, height - floorBandTop, 18);
        stageLip.endFill();
        facadeLayer.addChild(stageLip);
      }

      const worldBounds = renderVenueMapLayer(width, height) || playableBounds;
      currentVenueMapBounds = worldBounds || playableBounds;
      renderVenueGridLayer(currentVenueMapBounds);

      if (stageCamera) {
        stageCamera.setPlayableBounds?.(playableBounds);
        stageCamera.setWorldBounds?.(worldBounds);
        const activeMapId = String(currentVenueMapState?.asset_id || currentVenueMapAssetID || "");
        if (activeMapId && String(stageCamera.getView?.().activeMapId || "") !== activeMapId) {
          stageCamera.setActiveMapId?.(activeMapId, { reset: false });
        }
        updateCameraControls(stageCamera.getView?.());
      }

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
        pinnedObjectLayer.addChild(marker, markerLabel);
      }

      if (recentFocusMarker && Date.now() - recentFocusMarker.at < 15000) {
        const marker = new PIXI.Graphics();
        marker.lineStyle(2, 0x7fd7ff, 0.95);
        marker.drawCircle(0, 0, 18);
        marker.moveTo(-24, 0);
        marker.lineTo(24, 0);
        marker.moveTo(0, -24);
        marker.lineTo(0, 24);
        marker.beginFill(0x7fd7ff, 0.14);
        marker.drawCircle(0, 0, 30);
        marker.endFill();
        marker.position.set(recentFocusMarker.x, recentFocusMarker.y);

        const markerLabelStyle = new PIXI.TextStyle({
          fontFamily: "Arial",
          fontSize: 11,
          fontWeight: "700",
          fill: 0x7fd7ff,
        });
        const markerLabel = new PIXI.Text(recentFocusMarker.label ? `${recentFocusMarker.label} @ ${recentFocusMarker.x}, ${recentFocusMarker.y}` : `${recentFocusMarker.x}, ${recentFocusMarker.y}`, markerLabelStyle);
        markerLabel.position.set(recentFocusMarker.x + 18, recentFocusMarker.y - 34);
        pinnedObjectLayer.addChild(marker, markerLabel);
      }

      let liveCount = 0;
      currentObjects.forEach((model, index) => {
        let node;
        if (model.kind === "fire") {
          node = makeFireNode(model);
        } else if (model.kind === "token" && model.live) {
          node = makeTokenNode(model);
        } else if (model.kind === "card" && model.live) {
          node = makeCardNode(model);
        } else {
          node = makePlaceholderNode(model);
        }

        const floatingPoint = floatingCardStateForModel(model);
        if (floatingPoint && floatingObjectLayer) {
          node.container.position.set(floatingPoint.x, floatingPoint.y);
          node.container.zIndex = 100 + index;
          floatingObjectLayer.addChild(node.container);
          if (dragState && updateObjectMatches(dragState.model, model)) {
            dragState.node = node.container;
          }
        } else {
          const pin = cardPinData(model);
          const isWorldObject = model.kind === "fire" || model.kind === "token" || pin.mode === "world";
          const displayedPoint = currentDisplayedPointForModel(model);
          const position = model.kind === "token"
            ? displayedPoint
            : (isWorldObject ? worldPointForStagePoint(displayedPoint) : displayedPoint);
          node.container.position.set(position.x, position.y);
          let zIndex = 10 + index;
          if (model.kind === "token") {
            zIndex = tokenLayerForModel(model) === "director" ? 34 + index : 24 + index;
          } else if (model.kind === "card") {
            zIndex = 44 + index;
          } else if (model.kind === "fire") {
            zIndex = 14 + index;
          }
          node.container.zIndex = zIndex;
          const targetLayer = isWorldObject ? pinnedObjectLayer : overlayObjectLayer;
          targetLayer.addChild(node.container);
        }
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
      const renderedTokens = currentObjects.filter((object) => object.kind === "token").length;
      setSnapshotSummary(`${renderedFire ? "Fire" : "No fire"} and ${renderedCards} card object${renderedCards === 1 ? "" : "s"} plus ${renderedTokens} token object${renderedTokens === 1 ? "" : "s"} are rendered from the live Cave snapshot.`);
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

      runtimeLifecycle.listen(pixiApp.view, "contextmenu", handleNativeStageContextMenu, true);
      runtimeLifecycle.listen(stageHost, "contextmenu", handleNativeStageContextMenu, true);

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
      worldLayer = new PIXI.Container();
      worldLayer.zIndex = 5;
      mapLayer = new PIXI.Container();
      mapLayer.zIndex = 0;
      gridLayer = new PIXI.Container();
      gridLayer.zIndex = 1;
      pinnedObjectLayer = new PIXI.Container();
      pinnedObjectLayer.zIndex = 2;
      facadeLayer = new PIXI.Container();
      facadeLayer.zIndex = 8;
      overlayObjectLayer = new PIXI.Container();
      overlayObjectLayer.zIndex = 10;
      floatingObjectLayer = new PIXI.Container();
      floatingObjectLayer.zIndex = 15;
      uiLayer = new PIXI.Container();
      uiLayer.zIndex = 20;
      pixiApp.stage.addChild(sceneRoot);
      worldLayer.addChild(mapLayer, gridLayer, pinnedObjectLayer);
      sceneRoot.addChild(backgroundLayer, worldLayer, facadeLayer, overlayObjectLayer, floatingObjectLayer, uiLayer);
      runtimeLifecycle.track(() => {
        try {
          pixiApp?.destroy?.(true);
        } catch (error) {
          console.warn("pixi cleanup failed", error);
        }
      });

      stageCamera = window.VictoryStageCamera?.mount?.({
        stageElement: stageHost,
        worldLayer,
        venueSlug: "first-theater",
        userScope: String(currentIdentity?.user_id || currentIdentity?.handle || currentIdentity?.display_name || "browser"),
        minZoom: cameraDefaults.minZoom,
        maxZoom: cameraDefaults.maxZoom,
        getPlayableBounds,
        worldBounds: getPlayableBounds(),
        onChange(view) {
          updateCameraControls(view);
        },
      }) || null;
      runtimeLifecycle.track(() => {
        try {
          stageCamera?.destroy?.();
        } catch (error) {
          console.warn("stage camera cleanup failed", error);
        }
      });
      updateCameraControls(stageCamera?.getView?.());

      runtimeLifecycle.listen(stageHost, "pointermove", (event) => {
        const screenPoint = stageScreenPointFromClient(event.clientX, event.clientY);
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
        dragState.floatingPoint = {
          x: Math.round(Number(screenPoint.x || 0)) - dragState.offsetX,
          y: Math.round(Number(screenPoint.y || 0)) - dragState.offsetY,
        };
        dragState.node.position.set(dragState.floatingPoint.x, dragState.floatingPoint.y);
        const dropX = Math.round(dragState.floatingPoint.x);
        const dropY = Math.round(dragState.floatingPoint.y);
        setMovementReport(`Dragging to x ${dropX}, y ${dropY}`);
      });

      runtimeLifecycle.listen(window, "pointerup", finishDrag, true);
      runtimeLifecycle.listen(window, "pointercancel", cancelDrag, true);
      cardEditorHeader?.addEventListener("pointerdown", beginCardEditorDrag);
      runtimeLifecycle.listen(window, "pointermove", moveCardEditorDrag, true);
      runtimeLifecycle.listen(window, "pointerup", endCardEditorDrag, true);
      runtimeLifecycle.listen(window, "pointercancel", endCardEditorDrag, true);
      mapEditorHeader?.addEventListener("pointerdown", beginMapEditorDrag);
      runtimeLifecycle.listen(window, "pointermove", moveMapEditorDrag, true);
      runtimeLifecycle.listen(window, "pointerup", endMapEditorDrag, true);
      runtimeLifecycle.listen(window, "pointercancel", endMapEditorDrag, true);
      gridEditorHeader?.addEventListener("pointerdown", beginGridEditorDrag);
      runtimeLifecycle.listen(window, "pointermove", moveGridEditorDrag, true);
      runtimeLifecycle.listen(window, "pointerup", endGridEditorDrag, true);
      runtimeLifecycle.listen(window, "pointercancel", endGridEditorDrag, true);
      tokenPickerHeader?.addEventListener("pointerdown", beginTokenPickerDrag);
      runtimeLifecycle.listen(window, "pointermove", moveTokenPickerDrag, true);
      runtimeLifecycle.listen(window, "pointerup", endTokenPickerDrag, true);
      runtimeLifecycle.listen(window, "pointercancel", endTokenPickerDrag, true);
      tokenEditorHeader?.addEventListener("pointerdown", beginTokenEditorDrag);
      runtimeLifecycle.listen(window, "pointermove", moveTokenEditorDrag, true);
      runtimeLifecycle.listen(window, "pointerup", endTokenEditorDrag, true);
      runtimeLifecycle.listen(window, "pointercancel", endTokenEditorDrag, true);

      resizeObserver = new ResizeObserver(() => layoutPixiScene());
      runtimeLifecycle.track(() => resizeObserver?.disconnect());
      resizeObserver.observe(stageShell);
      runtimeLifecycle.listen(window, "resize", layoutPixiScene);
      layoutPixiScene();
      runtimeLifecycle.timeout(() => {
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

      const { model, node, space } = dragState;
      dragState = null;

      const screenPoint = toStageCoordinates(node.x, node.y);
      const worldPoint = space === "world" && stageCamera?.screenToWorld
        ? stageCamera.screenToWorld(screenPoint.x, screenPoint.y)
        : null;
      const stagePoint = worldPoint
        ? toStageCoordinates(worldPoint.x, worldPoint.y)
        : screenPoint;
      setMovementReport(`Dropped at x ${screenPoint.x}, y ${screenPoint.y}`);

      if (!model.live) {
        setStageStatus("Non-live object moved only in Pixi.");
        return;
      }

      if (isCardObject(model)) {
        const pinMode = space === "world" ? "world" : "overlay";
        const payload = {
          element_id: model.elementId || "",
          element_slug: model.elementSlug || "",
          front_text: model.frontText || "",
          back_text: model.backText || "",
          color: model.color || "#d9c7a6",
          pin_mode: pinMode,
        };
        if (pinMode === "world") {
          payload.world_x = stagePoint.x;
          payload.world_y = stagePoint.y;
        } else {
          const normalized = normalizeOverlayPoint(screenPoint);
          payload.screen_x = normalized.screen_x;
          payload.screen_y = normalized.screen_y;
        }

        const sent = sendAction("update/index_card", payload);
        const placementSent = sendAction("act/place_element", {
          element_id: model.elementId || "",
          element_slug: model.elementSlug || "",
          venue_slug: "the-cave",
          layer: "stage",
          x: stagePoint.x,
          y: stagePoint.y,
          order: Number(model.position?.order ?? 0),
        });
        if (sent) {
          updateLocalCardPinModel(model, (m) => {
            m.source.data = {
              ...(m.source.data || {}),
              pin_mode: pinMode,
              world_x: pinMode === "world" ? stagePoint.x : null,
              world_y: pinMode === "world" ? stagePoint.y : null,
              screen_x: pinMode === "overlay" ? normalizeOverlayPoint(screenPoint).screen_x : null,
              screen_y: pinMode === "overlay" ? normalizeOverlayPoint(screenPoint).screen_y : null,
            };
            m.position = setLocalPositionOverrideForModel(m, stagePoint) || m.position;
          });
        }
        setStageStatus(sent && placementSent ? `${model.label} updated.` : "Move could not be sent. Socket unavailable.");
        if (smokeMode) {
          setSmokeLine(sent && placementSent ? `Drop sent at ${screenPoint.x}, ${screenPoint.y}.` : `Drop failed at ${screenPoint.x}, ${screenPoint.y}.`);
        }
        return;
      }

      const tokenSnapMode = isTokenObject(model) ? tokenSnapModeForModel(model) : "";
      const snappedStagePoint = isTokenObject(model) && tokenSnapMode === "grid"
        ? tokenPlacementPointForCreate(stagePoint, model, tokenSnapMode)
        : stagePoint;

      const sent = isTokenObject(model)
        ? (() => {
            const snapMode = tokenSnapMode;
            return sendAction("update/token", {
              element_id: model.elementId || "",
              element_slug: model.elementSlug || "",
              venue_slug: "the-cave",
              layer: "stage",
              x: Number(snappedStagePoint?.x ?? stagePoint.x ?? 0),
              y: Number(snappedStagePoint?.y ?? stagePoint.y ?? 0),
              order: Number(model.position?.order ?? 0),
              snap_mode: snapMode,
              token_layer: tokenLayerForModel(model),
              scale: tokenScaleForModel(model),
            });
          })()
        : sendAction("act/place_element", {
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
        x: Number(snappedStagePoint?.x ?? stagePoint.x ?? 0),
        y: Number(snappedStagePoint?.y ?? stagePoint.y ?? 0),
        order: Number(model.position?.order ?? 0),
        frame: isTokenObject(model) && tokenSnapMode === "grid" ? "center" : "top-left",
      };
      if (overrideKey) {
        localPositionOverrides.set(overrideKey, overridePos);
      }
      updateLocalObjectModel(model, (m) => {
        m.position = overridePos;
        if (isTokenObject(m)) {
          m.source.data = {
            ...(m.source.data || {}),
            snap_mode: tokenSnapModeForModel(m),
            token_layer: tokenLayerForModel(m),
            scale: tokenScaleForModel(m),
          };
        }
      });

      if (sent) {
        setStageStatus(isTokenObject(model) ? "Token move sent through update/token." : "Move sent through act/place_element.");
        if (smokeMode) {
          setSmokeLine(`Drop sent at ${screenPoint.x}, ${screenPoint.y}.`);
        }
      } else {
        setStageStatus("Move could not be sent. Socket unavailable.");
        if (smokeMode) {
          setSmokeLine(`Drop failed at ${screenPoint.x}, ${screenPoint.y}.`);
        }
      }
    }

    function cancelDrag() {
      if (!dragState) return;
      const { model, originalPinMode, originalData } = dragState;
      dragState = null;
      if (model?.live && isCardObject(model) && originalData) {
        updateLocalCardPinModel(model, (m) => {
          m.source.data = {
            ...(originalData || {}),
            pin_mode: originalPinMode === "world" ? "world" : "overlay",
          };
        });
      } else {
        renderPixiScene();
      }
      setMovementReport("Drag cancelled.");
      setStageStatus("Drag cancelled.");
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

    function clampTokenPickerPosition(left, top) {
      const shellRect = stageShell?.getBoundingClientRect?.();
      const panelRect = tokenPickerPanel?.getBoundingClientRect?.();
      const panelWidth = Number(panelRect?.width || 640);
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

    function setTokenPickerPosition(left, top) {
      if (!tokenPickerPanel) return;
      const position = clampTokenPickerPosition(left, top);
      tokenPickerPanel.style.left = `${position.left}px`;
      tokenPickerPanel.style.top = `${position.top}px`;
      tokenPickerPanel.style.right = "auto";
      tokenPickerPanel.style.bottom = "auto";
    }

    function clampTokenEditorPosition(left, top) {
      const shellRect = stageShell?.getBoundingClientRect?.();
      const panelRect = tokenEditorPanel?.getBoundingClientRect?.();
      const panelWidth = Number(panelRect?.width || 420);
      const panelHeight = Number(panelRect?.height || 360);
      const shellWidth = Number(shellRect?.width || 0);
      const shellHeight = Number(shellRect?.height || 0);
      const maxLeft = Math.max(8, shellWidth - panelWidth - 8);
      const maxTop = Math.max(8, shellHeight - panelHeight - 8);
      return {
        left: Math.max(8, Math.min(Math.round(left), maxLeft)),
        top: Math.max(8, Math.min(Math.round(top), maxTop)),
      };
    }

    function setTokenEditorPosition(left, top) {
      if (!tokenEditorPanel) return;
      const position = clampTokenEditorPosition(left, top);
      tokenEditorPanel.style.left = `${position.left}px`;
      tokenEditorPanel.style.top = `${position.top}px`;
      tokenEditorPanel.style.right = "auto";
      tokenEditorPanel.style.bottom = "auto";
    }

    function beginTokenPickerDrag(event) {
      if (!tokenPickerPanel || tokenPickerPanel.hidden || !tokenPickerHeader) return;
      if (event.button !== 0) return;
      event.preventDefault();
      event.stopPropagation();
      const rect = tokenPickerPanel.getBoundingClientRect();
      tokenPickerDragState = {
        offsetX: event.clientX - rect.left,
        offsetY: event.clientY - rect.top,
      };
      tokenPickerHeader.setPointerCapture?.(event.pointerId);
      tokenPickerHeader.style.cursor = "grabbing";
    }

    function moveTokenPickerDrag(event) {
      if (!tokenPickerDragState || !tokenPickerPanel) return;
      const shellRect = stageShell?.getBoundingClientRect?.();
      if (!shellRect) return;
      setTokenPickerPosition(
        event.clientX - shellRect.left - tokenPickerDragState.offsetX,
        event.clientY - shellRect.top - tokenPickerDragState.offsetY,
      );
    }

    function endTokenPickerDrag() {
      if (!tokenPickerDragState) return;
      tokenPickerDragState = null;
      if (tokenPickerHeader) {
        tokenPickerHeader.style.cursor = "move";
      }
    }

    function beginTokenEditorDrag(event) {
      if (!tokenEditorPanel || tokenEditorPanel.hidden || !tokenEditorHeader) return;
      if (event.button !== 0) return;
      event.preventDefault();
      event.stopPropagation();
      const rect = tokenEditorPanel.getBoundingClientRect();
      tokenEditorDragState = {
        offsetX: event.clientX - rect.left,
        offsetY: event.clientY - rect.top,
      };
      tokenEditorHeader.setPointerCapture?.(event.pointerId);
      tokenEditorHeader.style.cursor = "grabbing";
    }

    function moveTokenEditorDrag(event) {
      if (!tokenEditorDragState || !tokenEditorPanel) return;
      const shellRect = stageShell?.getBoundingClientRect?.();
      if (!shellRect) return;
      setTokenEditorPosition(
        event.clientX - shellRect.left - tokenEditorDragState.offsetX,
        event.clientY - shellRect.top - tokenEditorDragState.offsetY,
      );
    }

    function endTokenEditorDrag() {
      if (!tokenEditorDragState) return;
      tokenEditorDragState = null;
      if (tokenEditorHeader) {
        tokenEditorHeader.style.cursor = "move";
      }
    }

    function canSendStageAction() {
      return !!socketController?.canSendStageAction?.();
    }

    function scheduleSocketReconnect() {
      return socketController?.scheduleSocketReconnect?.();
    }

    function queuePendingSocketAction(type, extra = {}) {
      return socketController?.queuePendingSocketAction?.(type, extra);
    }

    function flushPendingSocketActions() {
      return socketController?.flushPendingSocketActions?.();
    }

    function sendAction(type, extra = {}) {
      return socketController?.sendAction?.(type, extra) || false;
    }

    function buildFocusPingPayload() {
      return socketController?.buildFocusPingPayload?.() || null;
    }

    function sendPing() {
      return socketController?.sendPing?.() || false;
    }

    function sendFocusPing() {
      return socketController?.sendFocusPing?.() || false;
    }

    async function handleSocketMessage(msg) {
      if (!msg) return null;

      if (msg.kind === "snapshot") {
        applySnapshot(msg.snapshot || msg.message?.data || null);
        return msg;
      }

      if (msg.kind === "focus_ping" && msg.data) {
        applyVenueFocusPing(msg.data);
        return msg;
      }

      if (msg.kind === "action" && msg.action) {
        const actionType = msg.action.type || "";
        if (actionType === "chat/message") {
          appendChatActionLine(msg.action);
          return msg;
        }
        if (
          actionType === "act/place_element" ||
          actionType === "act/remove_element" ||
          actionType === "act/set_element_lock" ||
          actionType === "act/set_nameplate_visibility" ||
          actionType === "act/reveal_element" ||
          actionType === "act/hide_element" ||
          actionType === "create/index_card" ||
          actionType === "create/token" ||
          actionType === "update/token" ||
          actionType === "update/index_card" ||
          actionType === "delete/index_card"
        ) {
          await refreshWorld();
          if (actionType === "create/index_card") {
            placeCreatedIndexCard(msg.action);
          }
          if (actionType === "create/token") {
            setMovementLine("create/token accepted by the live action stream.");
          }
          if (actionType === "update/token") {
            setMovementLine("update/token accepted by the live action stream.");
          }
          if (actionType === "create/token" || actionType === "update/token") {
            const targetId = String(msg.action?.target?.element_id || msg.action?.payload?.element_id || "").trim();
            const targetSlug = String(msg.action?.target?.element_slug || msg.action?.payload?.element_slug || "").trim();
            const refreshedToken = currentObjects.find((item) =>
              item &&
              item.live &&
              isTokenObject(item) &&
              (
                (targetId && item.elementId === targetId) ||
                (targetSlug && item.elementSlug === targetSlug)
              )
            ) || null;
            if (refreshedToken) {
              selectObject(refreshedToken, actionType === "create/token" ? "Token created." : "Token updated.");
            }
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
          return msg;
        }
      }

      if (msg.kind === "showing_update") {
        await refreshWorld();
        updateChatPresentation();
        return msg;
      }

      if (msg.kind === "error") {
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
        return msg;
      }

      return msg;
    }

    function applyVenueFocusPing(data) {
      if (!data) return;
      const stamp = Number(Date.parse(String(data.ts || "")) || Date.now());
      if (stamp < latestFocusEventStamp) {
        return;
      }
      latestFocusEventStamp = stamp;

      const targetPanX = Number.isFinite(Number(data.camera_pan_x)) ? Number(data.camera_pan_x) : Number(data.camera_center_x || 0);
      const targetPanY = Number.isFinite(Number(data.camera_pan_y)) ? Number(data.camera_pan_y) : Number(data.camera_center_y || 0);
      const targetZoom = Number.isFinite(Number(data.camera_zoom_relative_to_fit))
        ? Number(data.camera_zoom_relative_to_fit)
        : 1;
      stageCamera?.animateToView?.({
        panX: targetPanX,
        panY: targetPanY,
        zoomRelativeToFit: targetZoom,
      }, { duration: 250 });

      setRecentFocusMarker({
        x: Number(data.focus_x || 0),
        y: Number(data.focus_y || 0),
      }, "Director focus");
      const isSelf = String(data.sender_user_id || "") === String(currentActorId || "");
      setStageStatus(isSelf ? "Focused venue." : "Director focus.");
      setMovementLine(isSelf ? "Focused venue." : "Director focus.");
      renderPixiScene();
    }

    function applySnapshot(snapshot) {
      projectedState?.replaceFromSnapshot?.(snapshot, { viewerRole: currentRole });
      currentSnapshot = snapshot || null;
      const elements = Array.isArray(snapshot?.elements) ? snapshot.elements : [];
      const overlay = snapshot?.overlay || null;
      const showing = snapshot?.showing || null;
      const latestFire = elements.find((element) => element?.slug === "first-fire") || null;
      const stageCards = elements.filter((element) =>
        String(element?.element_type || "").toLowerCase() === "index_card" &&
        String(element?.surface || element?.data?.surface || "tray").toLowerCase() === "stage"
      );
      const stageTokens = elements.filter((element) =>
        String(element?.element_type || "").toLowerCase() === "token" &&
        String(element?.surface || element?.data?.surface || "stage").toLowerCase() === "stage"
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
      setSnapshotSummary(`Showing is ${showingState}; ${overlayState}; fire ${latestFire ? "is present" : "not present"}; staged cards ${stageCards.length}; staged tokens ${stageTokens.length}.`);

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
      syncTokenEditorWithSelection();
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
      return socketController?.connectSocket?.() || null;
    }

    async function bootstrapFirstTheater() {
      try {
        initializeShellChrome(null);
        headerHoverOpen = true;
        chatHoverOpen = false;
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
    runtimeLifecycle.listen(window, "beforeunload", () => runtimeLifecycle.dispose());
    window.VictoryFirstTheaterCleanup = () => runtimeLifecycle.dispose();

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
    topBar?.addEventListener("pointerleave", () => closeHeaderHoverState());
    topBar?.addEventListener("pointerdown", (event) => {
      syncHeaderHoverState(event.clientY);
    });

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
    leftDrawerEdgeTrigger?.addEventListener("pointerenter", () => setDrawerHoverState("left", true));
    leftDrawerEdgeTrigger?.addEventListener("pointerleave", () => setDrawerHoverState("left", false));
    leftDrawerEdgeTrigger?.addEventListener("focusin", () => setDrawerHoverState("left", true));

    rightDrawer?.addEventListener("pointerenter", () => setDrawerHoverState("right", true));
    rightDrawer?.addEventListener("pointerleave", () => setDrawerHoverState("right", false));
    rightDrawer?.addEventListener("focusin", () => setDrawerHoverState("right", true));
    rightDrawer?.addEventListener("focusout", (event) => {
      if (!rightDrawer?.contains(event.relatedTarget)) {
        setDrawerHoverState("right", false);
      }
    });
    rightDrawerEdgeTrigger?.addEventListener("pointerenter", () => setDrawerHoverState("right", true));
    rightDrawerEdgeTrigger?.addEventListener("pointerleave", () => setDrawerHoverState("right", false));
    rightDrawerEdgeTrigger?.addEventListener("focusin", () => setDrawerHoverState("right", true));

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

    cameraZoomOutButton?.addEventListener("click", () => {
      if (!stageCamera) return;
      const view = stageCamera.getView?.();
      stageCamera.setView?.({
        zoomRelativeToFit: Math.max(cameraDefaults.minZoom, Number(view?.zoomRelativeToFit || 1) * 0.9),
      });
      updateCameraControls(stageCamera.getView?.());
    });

    cameraZoomInButton?.addEventListener("click", () => {
      if (!stageCamera) return;
      const view = stageCamera.getView?.();
      stageCamera.setView?.({
        zoomRelativeToFit: Math.min(cameraDefaults.maxZoom, Number(view?.zoomRelativeToFit || 1) * 1.1),
      });
      updateCameraControls(stageCamera.getView?.());
    });

    cameraFitButton?.addEventListener("click", () => {
      resetCameraToFit(currentVenueMapState?.asset_id || currentVenueMapAssetID || "");
    });

    pingButton?.addEventListener("click", (event) => {
      event.preventDefault();
      event.stopPropagation();
      if (event.shiftKey) {
        sendFocusPing();
      } else {
        const sent = sendPing();
        setStageStatus(sent ? "Ping sent." : "Socket unavailable.");
      }
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

    tokenPickerSearch?.addEventListener("input", () => {
      tokenPickerState.search = String(tokenPickerSearch.value || "");
      renderTokenPickerList();
    });

    tokenPickerShape?.addEventListener("change", () => {
      tokenPickerState.filterShape = String(tokenPickerShape.value || "all");
      renderTokenPickerList();
    });

    tokenPickerRefresh?.addEventListener("click", () => {
      void refreshWarehouseTokenAssets();
    });

    tokenPickerCancel?.addEventListener("click", () => {
      closeTokenPicker();
    });

    tokenEditorScale?.addEventListener("input", () => {
      tokenEditorDirty = true;
      const value = clampNumber(tokenEditorScale.value || 100, 25, 500, 100);
      if (tokenEditorScaleValue) tokenEditorScaleValue.value = String(value);
      if (tokenEditorStatus) {
        tokenEditorStatus.textContent = `Scale set to ${value}%.`;
      }
    });

    tokenEditorScaleValue?.addEventListener("input", () => {
      tokenEditorDirty = true;
      const value = clampNumber(tokenEditorScaleValue.value || 100, 25, 500, 100);
      if (tokenEditorScale) tokenEditorScale.value = String(value);
      if (tokenEditorStatus) {
        tokenEditorStatus.textContent = `Scale set to ${value}%.`;
      }
    });

    tokenEditorSnap?.addEventListener("change", () => {
      tokenEditorDirty = true;
      if (tokenEditorStatus) {
        tokenEditorStatus.textContent = tokenEditorSnap.value === "grid" ? "Snap to Grid selected." : "Free Placement selected.";
      }
    });

    tokenEditorLayer?.addEventListener("change", () => {
      tokenEditorDirty = true;
      if (tokenEditorStatus) {
        tokenEditorStatus.textContent = tokenEditorLayer.value === "director" ? "Director layer selected." : "Public layer selected.";
      }
    });

    tokenEditorReset?.addEventListener("click", () => {
      tokenEditorDirty = true;
      if (tokenEditorScale) tokenEditorScale.value = "100";
      if (tokenEditorScaleValue) tokenEditorScaleValue.value = "100";
      if (tokenEditorSnap) tokenEditorSnap.value = currentVenueGridConfig && currentVenueGridConfig.grid_type !== "none" ? "grid" : "free";
      if (tokenEditorLayer) tokenEditorLayer.value = "public";
      if (tokenEditorStatus) {
        tokenEditorStatus.textContent = "Reset to default token settings.";
      }
    });

    tokenEditorSave?.addEventListener("click", () => {
      saveTokenEditor();
    });

    tokenEditorCancel?.addEventListener("click", () => {
      hideTokenEditor();
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
      if (suppressStageContextMenu) {
        suppressStageContextMenu = false;
        return;
      }
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
      if (tokenPickerPanel && !tokenPickerPanel.hidden && !event.target.closest("#token-picker") && !event.target.closest("#token-picker-header")) {
        closeTokenPicker();
      }
      if (tokenEditorPanel && !tokenEditorPanel.hidden && !event.target.closest("#token-editor") && !event.target.closest("#token-editor-header")) {
        hideTokenEditor();
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
        closeTokenPicker();
        hideTokenEditor();
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
      const screenPoint = stageScreenPointFromClient(event.clientX, event.clientY);
      stagePlacementCandidate = point;
      stagePlacementScreenCandidate = screenPoint;
      updatePointerReadout(event.clientX || 0, event.clientY || 0, point, "Overlay menu");
      openContextMenu(event, stageContextMenuModel());
    });

    runtimeLifecycle.listen(document, "contextmenu", (event) => {
      const target = event.target;
      const composedPath = typeof event.composedPath === "function" ? event.composedPath() : [];
      const isStageEvent = target === stageHost || target === stageShell || stageShell?.contains?.(target) || stageHost?.contains?.(target) || composedPath.includes(stageHost) || composedPath.includes(stageShell) || composedPath.includes(pixiApp?.view);
      if (!isStageEvent) return;
      handleNativeStageContextMenu(event);
    }, true);

    runtimeLifecycle.listen(document, "pointerdown", (event) => {
      if (!isSecondaryPointerEvent(event)) return;
      const target = event.target;
      const composedPath = typeof event.composedPath === "function" ? event.composedPath() : [];
      const isStageEvent = target === stageHost || target === stageShell || stageShell?.contains?.(target) || stageHost?.contains?.(target) || composedPath.includes(stageHost) || composedPath.includes(stageShell) || composedPath.includes(pixiApp?.view);
      if (!isStageEvent) return;
      handleNativeStageContextMenu(event);
    }, true);

    runtimeLifecycle.listen(document, "mousedown", (event) => {
      if (!isSecondaryPointerEvent(event)) return;
      const target = event.target;
      const composedPath = typeof event.composedPath === "function" ? event.composedPath() : [];
      const isStageEvent = target === stageHost || target === stageShell || stageShell?.contains?.(target) || stageHost?.contains?.(target) || composedPath.includes(stageHost) || composedPath.includes(stageShell) || composedPath.includes(pixiApp?.view);
      if (!isStageEvent) return;
      handleNativeStageContextMenu(event);
    }, true);

    stageShell?.addEventListener("contextmenu", handleNativeStageContextMenu, true);
    stageShell?.addEventListener("mousedown", handleNativeStageContextMenu, true);
    stageHost?.addEventListener("contextmenu", handleNativeStageContextMenu, true);
    stageHost?.addEventListener("mousedown", handleNativeStageContextMenu, true);

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
