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
    const stageMapButton = document.getElementById("stage-map");
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
    const diceTrayRoot = document.getElementById("first-theater-dice-tray");
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
    const tokenPickerApply = document.getElementById("token-picker-apply");
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
    const smokeMode = new URLSearchParams(location.search).has("smoke");
    let smokeGridEnabled = smokeMode;
    const firstTheaterStateModule = window.VictoryFirstTheaterState || null;
    const firstTheaterSocketModule = window.VictoryFirstTheaterSocket || null;
    const firstTheaterLogicModule = window.VictoryFirstTheaterLogic || null;
    const firstTheaterGeometryModule = window.VictoryFirstTheaterGeometry || null;
    const firstTheaterStageControlsModule = window.VictoryFirstTheaterStageControls || null;
    const firstTheaterMapGridModule = window.VictoryFirstTheaterMapGrid || null;
    const firstTheaterSceneNodesModule = window.VictoryFirstTheaterSceneNodes || null;
    const firstTheaterTokenUiModule = window.VictoryFirstTheaterTokenUI || null;
    const firstTheaterDiceModule = window.VictoryFirstTheaterDice || null;
    const firstTheaterActionRouterModule = window.VictoryFirstTheaterActionRouter || null;
    const firstTheaterSocketControllerModule = window.VictoryFirstTheaterSocketController || null;
    const firstTheaterSessionSyncModule = window.VictoryFirstTheaterSessionSync || null;
    const firstTheaterLifecycleModule = window.VictoryFirstTheaterLifecycle || null;
    const firstTheaterContextModule = window.VictoryFirstTheaterContext || null;
    const firstTheaterEditorsModule = window.VictoryFirstTheaterEditors || null;
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
    const contextHelpers = firstTheaterContextModule?.createContextInteractionHelpers?.({
      smokeMode,
      getCurrentObjects: () => currentObjects,
      getCurrentNodeMap: () => currentNodeMap,
      getStageShell: () => stageShell,
      getStageHost: () => stageHost,
      getPixiApp: () => pixiApp,
      getLastStagePoint: () => lastStagePoint,
      setLastStagePoint: (point) => {
        if (point) {
          lastStagePoint = {
            x: Math.round(Number(point.x || 0)),
            y: Math.round(Number(point.y || 0)),
          };
        }
      },
      setPointerLine: (...args) => setPointerLine(...args),
      setSmokeLine: (...args) => setSmokeLine(...args),
      currentSelectionSummary: () => currentSelectionSummary(),
      stageContextMenuModel: () => stageContextMenuModel(),
      objectKind: (...args) => objectKind(...args),
      setStagePlacementCandidate: (point, screenPoint) => {
        stagePlacementCandidate = point || null;
        stagePlacementScreenCandidate = screenPoint || null;
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
    }) || null;
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
    let suppressEditorAutoClose = false;
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
    let diceTray = null;
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
    let editors = null;
    let sessionSync = null;
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
    const characterTrayButton = document.getElementById("character-tray-button");
    const characterTrayLabel = document.getElementById("character-tray-label");
    const rightCardEditorButton = document.getElementById("right-card-editor-button");
    const chatDiceTab = document.getElementById("chat-dice-tab");
    const chatHelpTab = document.getElementById("chat-help-tab");
    const chatDicePanel = document.getElementById("chat-dice-panel");
    const chatHelpPanel = document.getElementById("chat-help-panel");
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
      if (["producer", "director", "operator", "cast", "crew", "audience", "actor"].includes(value)) {
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
        case "operator":
          return "Operator";
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
      return normalized === "producer" || normalized === "director" || normalized === "operator";
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
      const cameraState = firstTheaterStageControlsModule?.cameraControlsState?.(view || stageCamera?.getView?.(), mapDisplayMode, cameraDefaults) || firstTheaterMapGridModule?.cameraControlsState?.(view || stageCamera?.getView?.(), mapDisplayMode, cameraDefaults) || {
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
      return firstTheaterMapGridModule?.getPlayableBounds?.(size) || computeStagePlayableBounds(size.width, size.height);
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
      firstTheaterMapGridModule?.resetCameraToFit?.(stageCamera, activeMapId, stageCamera?.getView?.(), currentVenueMapState, currentVenueMapAssetID);
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
      updateMapPresentation();
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
      const activeCharacter = currentIdentity?.active_character || null;
      const activeCharacterName = String(activeCharacter?.display_name || activeCharacter?.name || "").trim();
      const activeCharacterStatus = String(activeCharacter?.workbook_status || "").trim() || "Draft";
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
        characterValue.textContent = activeCharacterName ? `${activeCharacterName} · ${activeCharacterStatus}` : "No Character";
      }
      if (viewValue) {
        viewValue.textContent = "Self";
      }
      if (pingValue) {
        pingValue.textContent = Number.isFinite(lastPingMs) ? `${lastPingMs}ms` : "—";
      }
      if (rightCharacterValue) {
        rightCharacterValue.textContent = activeCharacterName ? `${activeCharacterName} · ${activeCharacterStatus}` : "No Character";
      }
      if (characterTrayLabel) {
        characterTrayLabel.textContent = activeCharacterName || "New Character";
      }
      if (characterTrayButton) {
        characterTrayButton.title = activeCharacterName
          ? `Open the workbook for ${activeCharacterName}`
          : "Open the workbook";
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

    function updateMapPresentation() {
      if (!stageMapButton) return;
      const canEditMap = canManageIndexCards(currentRole);
      const hasMap = Boolean(String(currentVenueMapState?.asset_id || currentVenueMapAssetID || "").trim());
      stageMapButton.disabled = !canEditMap;
      stageMapButton.textContent = hasMap ? "Edit Map" : "Add Map";
      stageMapButton.title = canEditMap
        ? (hasMap ? "Open the First Theater map editor." : "Add a new map to First Theater.")
        : "Only producers and directors can manage the First Theater map.";
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
      updateMapPresentation();
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

    function setChatCompanionTab(mode) {
      const showDice = String(mode || "dice") !== "help";
      if (chatDicePanel) chatDicePanel.hidden = !showDice;
      if (chatHelpPanel) chatHelpPanel.hidden = showDice;
      chatDiceTab?.classList.toggle("is-active", showDice);
      chatHelpTab?.classList.toggle("is-active", !showDice);
      chatDiceTab?.setAttribute("aria-selected", showDice ? "true" : "false");
      chatHelpTab?.setAttribute("aria-selected", showDice ? "false" : "true");
    }

    async function sendChatDraft() {
      if (!chatInput) return;
      const text = chatInput.value.trim();
      if (!text) return;

      const rollMatch = text.match(/^\/roll(?:\s+(.+))?$/i);
      if (rollMatch) {
        const expression = String(rollMatch[1] || "").trim();
        if (!expression) {
          appendSystemChatNotice("Usage: /roll XdY");
          chatInput.value = "";
          return;
        }
        if (!diceTray?.roll) {
          appendSystemChatNotice("Dice tray unavailable.");
          chatInput.value = "";
          return;
        }
        try {
          const result = await diceTray.roll({
            expression,
            visibility: "public",
          });
          appendSystemChatNotice(diceTray?.formatRollSummary?.(result) || `Rolled ${expression}.`);
        } catch (error) {
          appendSystemChatNotice(String(error?.message || error || "Roll failed"));
        }
        chatInput.value = "";
        return;
      }

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

      try {
        const journalResult = await window.VictoryMicChat?.sendJournalCommand?.(text, { visibility: "private" });
        if (journalResult?.handled) {
          const message = String(journalResult.message || "").replace(/\n+/g, " · ").trim();
          if (journalResult.ok) {
            appendSystemChatNotice(message || "Journal entry saved privately.");
          } else if (message) {
            appendSystemChatNotice(message);
          }
          chatInput.value = "";
          return;
        }
      } catch (error) {
        appendSystemChatNotice(String(error?.message || error || "Journal save failed"));
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

    function syncCurrentObjectsFromProjectedState() {
      const projected = projectedState?.getState?.() || null;
      if (!projected) return false;
      currentObjects = Array.isArray(projected.objects) ? projected.objects : [];
      if (currentSelection) {
        const refreshedSelection = currentObjects.find((item) => updateObjectMatches(item, currentSelection)) || null;
        currentSelection = refreshedSelection;
      }
      updateStatusSummary();
      updateShellTargetPresentation();
      syncSelectedActions();
      syncCardEditorWithSelection();
      syncTokenEditorWithSelection();
      return true;
    }

    let projectedStateRenderQueued = false;
    function scheduleProjectedStateRender() {
      if (projectedStateRenderQueued) return;
      projectedStateRenderQueued = true;
      runtimeLifecycle.animationFrame(() => {
        projectedStateRenderQueued = false;
        if (syncCurrentObjectsFromProjectedState()) {
          renderPixiScene();
        }
      });
    }

    if (projectedState?.subscribe) {
      const unsubscribeProjectedObjects = projectedState.subscribe("objects", () => {
        scheduleProjectedStateRender();
      });
      runtimeLifecycle.track(() => {
        try {
          unsubscribeProjectedObjects?.();
        } catch (error) {
          console.warn("projected state unsubscribe failed", error);
        }
      });
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
      cancelContextMenuEvent: (...args) => contextHelpers?.cancelContextMenuEvent?.(...args) || cancelContextMenuEvent(...args),
      openResolvedContextMenu: (...args) => contextHelpers?.openResolvedContextMenu?.(...args) || openResolvedContextMenu(...args),
      eventClientPoint: (...args) => contextHelpers?.eventClientPoint?.(...args) || eventClientPoint(...args),
      stageScreenPointFromClient: (...args) => contextHelpers?.stageScreenPointFromClient?.(...args) || stageScreenPointFromClient(...args),
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
      defaultPlacementPoint: () => {
        const size = getStageSize();
        const worldPoint = stageCamera?.screenToWorld?.(Math.round(size.width / 2), Math.round(size.height / 2)) || null;
        if (worldPoint) {
          return {
            x: Math.round(Number(worldPoint.x || 0)),
            y: Math.round(Number(worldPoint.y || 0)),
          };
        }
        const bounds = getPlayableBounds();
        return {
          x: Math.round(Number(bounds.x || 0) + (Number(bounds.width || 0) / 2)),
          y: Math.round(Number(bounds.y || 0) + (Number(bounds.height || 0) / 2)),
        };
      },
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
        scale: tokenEditorScale,
        scaleValue: tokenEditorScaleValue,
        snap: tokenEditorSnap,
        layer: tokenEditorLayer,
        status: tokenEditorStatus,
        setPosition: setTokenEditorPosition,
      }),
      getTokenEditorTargetKey: () => tokenEditorTargetKey,
      setTokenEditorTargetKey: (value) => { tokenEditorTargetKey = String(value || ""); },
      getTokenEditorDirty: () => tokenEditorDirty,
      setTokenEditorDirty: (value) => { tokenEditorDirty = Boolean(value); },
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
        apply: tokenPickerApply,
        setPosition: setTokenPickerPosition,
      }),
      refreshWarehouseTokenAssets: () => refreshWarehouseTokenAssets(),
      clampNumber: (...args) => clampNumber(...args),
      objectState: (...args) => objectState(...args),
      isTokenObject: (...args) => isTokenObject(...args),
      currentObjects: () => currentObjects,
      updateTokenLocalModel: (...args) => updateTokenLocalModel(...args),
    }) || null;

    diceTray = firstTheaterDiceModule?.createDiceTrayController?.({
      document,
      window,
      mountRoot: diceTrayRoot,
      sendAction: (...args) => sendAction(...args),
      canSendAction: () => canSendStageAction(),
      getCurrentSnapshot: () => currentSnapshot,
      refreshWorld: () => refreshWorld(),
      setStageStatus: (...args) => setStageStatus(...args),
      setMovementLine: (...args) => setMovementLine(...args),
      appendSystemChatNotice: (...args) => appendSystemChatNotice(...args),
      onAction: () => updateChatPresentation(),
      onPendingChange: () => updateShellMetaPresentation(),
      onHistoryChange: () => updateShellMetaPresentation(),
      historyLimit: 2,
      timeoutMs: 15000,
    }) || null;

    sessionSync = firstTheaterSessionSyncModule?.createSessionSync?.({
      fetch: (...args) => fetch(...args),
      getCurrentIdentity: () => currentIdentity,
      setCurrentIdentity: (value) => { currentIdentity = value; },
      getCurrentRole: () => currentRole,
      setCurrentRole: (value) => { currentRole = value; },
      getCurrentSessionId: () => currentSessionId,
      setCurrentSessionId: (value) => { currentSessionId = String(value || ""); },
      getCurrentActorId: () => currentActorId,
      setCurrentActorId: (value) => { currentActorId = String(value || ""); },
      getCurrentSnapshot: () => currentSnapshot,
      getCurrentObjects: () => currentObjects,
      setCurrentObjects: (value) => { currentObjects = Array.isArray(value) ? value : []; },
      getCurrentSelection: () => currentSelection,
      setCurrentSelection: (value) => { currentSelection = value; },
      replaceProjectedState: (...args) => projectedState?.replaceFromSnapshot?.(...args),
      buildObjects: (...args) => buildObjects(...args),
      applyVenueFocusPing: (...args) => applyVenueFocusPing(...args),
      updateStatusSummary: (...args) => updateStatusSummary(...args),
      setStageStatus: (...args) => setStageStatus(...args),
      setMovementLine: (...args) => setMovementLine(...args),
      setSnapshotSummary: (...args) => setSnapshotSummary(...args),
      setLiveFeedLine: (...args) => setLiveFeedLine(...args),
      setSelectionLine: (...args) => setSelectionLine(...args),
      updateChatPresentation: (...args) => updateChatPresentation(...args),
      syncDiceTrayFromSnapshot: (snapshot) => diceTray?.handleSnapshot?.(snapshot),
      handleDiceTrayAction: (action) => diceTray?.handleAction?.(action),
      handleDiceTrayError: (errorText, message) => diceTray?.handleError?.(errorText, message),
      rejectPendingDiceTrayRolls: (reason) => diceTray?.rejectPendingRolls?.(reason),
      syncCurrentObjectsFromProjectedState: () => syncCurrentObjectsFromProjectedState(),
      canManageIndexCards: (...args) => canManageIndexCards(...args),
      canManageStageTokens: (...args) => canManageStageTokens(...args),
      canActorRevealHideStageObjects: (...args) => canActorRevealHideStageObjects(...args),
      cardFaceForModel: (...args) => cardFaceForModel(...args),
      cardDisplayMode: (...args) => cardDisplayMode(...args),
      tokenScaleForModel: (...args) => tokenScaleForModel(...args),
      tokenSnapModeForModel: (...args) => tokenSnapModeForModel(...args),
      tokenLayerForModel: (...args) => tokenLayerForModel(...args),
      closeContextMenu: () => closeContextMenu(),
      selectObject: (...args) => selectObject(...args),
      syncSelectedActions: () => syncSelectedActions(),
      syncCardEditorWithSelection: () => syncCardEditorWithSelection(),
      syncTokenEditorWithSelection: () => syncTokenEditorWithSelection(),
      refreshVenueGridConfig: (...args) => refreshVenueGridConfig(...args),
      refreshVenueMapState: (...args) => refreshVenueMapState(...args),
      refreshVenueMapAssets: (...args) => refreshVenueMapAssets(...args),
      loadPixiLibrary: () => loadPixiLibrary(),
      initializePixi: () => initializePixi(),
      initializeShellChrome: (...args) => initializeShellChrome(...args),
      updateShellMetaPresentation: () => updateShellMetaPresentation(),
      updateShellTargetPresentation: () => updateShellTargetPresentation(),
      updateHeaderPresentation: () => updateHeaderPresentation(),
      setRendererFallback: (...args) => setRendererFallback(...args),
      appendSystemChatNotice: (...args) => appendSystemChatNotice(...args),
      appendChatActionLine: (...args) => appendChatActionLine(...args),
      chatClosedMessage: () => chatClosedMessage(),
      getCurrentVenueMapState: () => currentVenueMapState,
      getCurrentVenueMapAssetID: () => currentVenueMapAssetID,
      getCurrentVenueGridConfig: () => currentVenueGridConfig,
      roleLabel: (...args) => roleLabel(...args),
      getStageStatus: () => stageStatus?.textContent || "",
      getStagePlacementCandidate: () => stagePlacementCandidate,
      setStagePlacementCandidate: (value) => { stagePlacementCandidate = value; },
      getStagePlacementScreenCandidate: () => stagePlacementScreenCandidate,
      setStagePlacementScreenCandidate: (value) => { stagePlacementScreenCandidate = value; },
      getPendingStageCardPlacement: () => pendingStageCardPlacement,
      setPendingStageCardPlacement: (value) => { pendingStageCardPlacement = value; },
      setSmokeLine: (...args) => setSmokeLine(...args),
      normalizeOverlayPoint: (...args) => normalizeOverlayPoint(...args),
      setRecentPlacementMarker: (...args) => setRecentPlacementMarker(...args),
      getVenueSlug: () => "first-theater",
      smokeMode,
      setLocalPositionOverrideForModel: (...args) => setLocalPositionOverrideForModel(...args),
      sendAction: (...args) => sendAction(...args),
      placeCreatedIndexCard: (action) => sessionSync?.placeCreatedIndexCard?.(action),
      setStageStatus: (...args) => setStageStatus(...args),
    }) || null;

    editors = firstTheaterEditorsModule?.createEditorControllers?.({
      getStageShell: () => stageShell,
      getCurrentRole: () => currentRole,
      getCurrentSelection: () => currentSelection,
      getCurrentObjects: () => currentObjects,
      getCurrentVenueMapState: () => currentVenueMapState,
      getCurrentVenueMapAssetID: () => currentVenueMapAssetID,
      getCurrentVenueMapAssets: () => currentVenueMapAssets,
      setCurrentVenueMapState: (value) => {
        currentVenueMapState = value;
        updateMapPresentation();
      },
      setCurrentVenueMapAssetID: (value) => {
        currentVenueMapAssetID = String(value || "");
        updateMapPresentation();
      },
      setCurrentVenueMapAssets: (value) => {
        currentVenueMapAssets = Array.isArray(value) ? value : [];
      },
      getCurrentVenueGridConfig: () => currentVenueGridConfig,
      setCurrentVenueGridConfig: (value) => {
        currentVenueGridConfig = value;
      },
      getPlayableBounds: () => getPlayableBounds(),
      getMapEditorElements: () => ({
        panel: mapEditorPanel,
        file: mapEditorFile,
        displayMode: mapEditorDisplayMode,
        fit: mapEditorFit,
        scale: mapEditorScale,
        cropX: mapEditorCropX,
        cropY: mapEditorCropY,
        safeMargin: mapEditorSafeMargin,
        preview: mapEditorPreview,
        previewMode: mapEditorPreviewMode,
        previewFocus: mapEditorPreviewFocus,
        previewSafe: mapEditorPreviewSafe,
        assets: mapEditorAssets,
        remove: mapEditorRemove,
      }),
      getGridEditorElements: () => ({
        panel: gridEditorPanel,
        type: gridEditorType,
        hexOrientation: gridEditorHexOrientation,
        hexOrientationField: gridEditorHexOrientationField,
        cellSize: gridEditorCellSize,
        offsetX: gridEditorOffsetX,
        offsetY: gridEditorOffsetY,
        opacity: gridEditorOpacity,
        lineWidth: gridEditorLineWidth,
        lineStyle: gridEditorLineStyle,
        visibility: gridEditorVisibility,
      }),
      getMapEditorPreviewURL: () => mapEditorPreviewURL,
      setMapEditorPreviewURL: (value) => { mapEditorPreviewURL = String(value || ""); },
      getCurrentSnapshot: () => currentSnapshot,
      setCurrentSnapshot: (value) => { currentSnapshot = value; },
      canManageIndexCards: (...args) => canManageIndexCards(...args),
      canManageStageTokens: (...args) => canManageStageTokens(...args),
      isCardObject: (...args) => isCardObject(...args),
      isTokenObject: (...args) => isTokenObject(...args),
      objectState: (...args) => objectState(...args),
      selectObject: (...args) => selectObject(...args),
      setStageStatus: (...args) => setStageStatus(...args),
      setMovementLine: (...args) => setMovementLine(...args),
      renderPixiScene: () => renderPixiScene(),
      renderVenueGridLayer: (...args) => renderVenueGridLayer(...args),
      sendAction: (...args) => sendAction(...args),
      updateLocalObjectModel: (...args) => updateLocalObjectModel(...args),
      updateLocalCardPinModel: (...args) => updateLocalCardPinModel(...args),
      updateTokenLocalModel: (...args) => updateTokenLocalModel(...args),
      setLocalPositionOverrideForModel: (...args) => setLocalPositionOverrideForModel(...args),
      currentDisplayedPointForModel: (...args) => currentDisplayedPointForModel(...args),
      tokenScaleForModel: (...args) => tokenScaleForModel(...args),
      tokenSnapModeForModel: (...args) => tokenSnapModeForModel(...args),
      tokenLayerForModel: (...args) => tokenLayerForModel(...args),
      tokenPlacementPointForCreate: (...args) => tokenPlacementPointForCreate(...args),
      tokenDuplicatePlacementForModel: (...args) => tokenDuplicatePlacementForModel(...args),
      cardDuplicatePlacementForModel: (...args) => cardDuplicatePlacementForModel(...args),
      tokenMoveTargetForModel: (...args) => tokenMoveTargetForModel(...args),
      cardMoveTargetForModel: (...args) => cardMoveTargetForModel(...args),
      refreshVenueMapAssets: (...args) => refreshVenueMapAssets(...args),
      refreshVenueMapState: (...args) => refreshVenueMapState(...args),
      refreshVenueGridConfig: (...args) => refreshVenueGridConfig(...args),
      uploadMapAsset: (...args) => uploadMapAsset(...args),
      saveVenueMap: (...args) => saveVenueMap(...args),
      removeVenueMap: (...args) => removeVenueMap(...args),
      saveGridConfig: (...args) => saveGridConfig(...args),
      renderVenueGridLayer: (...args) => renderVenueGridLayer(...args),
      setMapEditorStatus: (...args) => setMapEditorStatus(...args),
      setGridEditorStatus: (...args) => setGridEditorStatus(...args),
      setMapEditorPosition: (left, top) => setMapEditorPosition(left, top),
      setGridEditorPosition: (left, top) => setGridEditorPosition(left, top),
      clampMapEditorPosition: (...args) => clampMapEditorPosition(...args),
      clampGridEditorPosition: (...args) => clampGridEditorPosition(...args),
      clampCardEditorPosition: (...args) => clampCardEditorPosition(...args),
      getCardEditorTarget: () => getCardEditorTarget(),
      setCardEditorPosition: (...args) => setCardEditorPosition(...args),
      hideMapEditor: () => hideMapEditor(),
      hideGridEditor: () => hideGridEditor(),
      hideCardEditor: () => hideCardEditor(),
      hideTokenEditor: () => hideTokenEditor(),
      closeTokenPicker: () => closeTokenPicker(),
      openTokenPicker: (...args) => openTokenPicker(...args),
      canActorRevealHideStageObjects: (...args) => canActorRevealHideStageObjects(...args),
      setCurrentSelection: (value) => { currentSelection = value; },
      syncSelectedActions: () => syncSelectedActions(),
      syncTokenEditorWithSelection: () => syncTokenEditorWithSelection(),
      setCardEditorTargetKey: (value) => { cardEditorTargetKey = value; },
      setCardEditorDirty: (value) => { cardEditorDirty = Boolean(value); },
      getMapEditorOriginalState: () => mapEditorOriginalState,
      setMapEditorOriginalState: (value) => { mapEditorOriginalState = value; },
      setMapEditorDirty: (value) => { mapEditorDirty = Boolean(value); },
      getGridEditorOriginalState: () => gridEditorOriginalState,
      setGridEditorOriginalState: (value) => { gridEditorOriginalState = value; },
      setGridEditorDirty: (value) => { gridEditorDirty = Boolean(value); },
      cardEditorPanel,
      cardEditorFront,
      cardEditorBack,
      cardEditorColor,
      cardEditorStatus,
      cardEditorDirty: () => cardEditorDirty,
      mapEditorPanel,
      mapEditorFile,
      mapEditorDisplayMode,
      mapEditorFit,
      mapEditorScale,
      mapEditorCropX,
      mapEditorCropY,
      mapEditorSafeMargin,
      mapEditorPreview,
      mapEditorPreviewMode,
      mapEditorPreviewFocus,
      mapEditorPreviewSafe,
      mapEditorAssets: mapEditorAssets,
      gridEditorPanel,
      gridEditorType,
      gridEditorHexOrientation,
      gridEditorCellSize,
      gridEditorOffsetX,
      gridEditorOffsetY,
      gridEditorOpacity,
      gridEditorLineWidth,
      gridEditorLineStyle,
      gridEditorVisibility,
      gridEditorHexOrientationField,
      defaultGridConfig: () => defaultGridConfig(),
      mapEditorDraftFromState: () => mapEditorDraftFromState(),
      mapEditorPayloadFromUI: (assetID) => mapEditorPayloadFromUI(assetID),
      gridEditorDraftFromUI: () => gridEditorDraftFromUI(),
      syncMapEditorHexFieldVisibility: () => syncGridEditorHexFieldVisibility(),
      syncGridEditorVisibilityButton: () => syncGridEditorVisibilityButton(),
      syncGridEditorWithState: () => syncGridEditorWithState(),
      livePreviewMapOnStage: () => livePreviewMapOnStage(),
      syncMapAssetList: () => syncMapAssetList(),
      createIndexCardFromMenu: () => createIndexCardFromMenu(),
      placeCreatedIndexCard: (action) => placeCreatedIndexCard(action),
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
      openCardEditor: (...args) => openCardEditor(...args),
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
      refreshWorld: () => refreshWorld(),
      syncCurrentObjectsFromProjectedState: () => syncCurrentObjectsFromProjectedState(),
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
      isStageEvent: (event) => contextHelpers?.isStageEvent?.(event) || false,
      isSecondaryPointerEvent: (...args) => contextHelpers?.isSecondaryPointerEvent?.(...args) || isSecondaryPointerEvent(...args),
      markContextMenuHandled: (...args) => contextHelpers?.markContextMenuHandled?.(...args) || markContextMenuHandled(...args),
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
      onConnect: () => {
        diceTray?.setActionAvailability?.(canSendStageAction());
      },
      onClose: () => {
        diceTray?.setActionAvailability?.(canSendStageAction());
        diceTray?.rejectPendingRolls?.("Socket closed.");
      },
      onError: () => {
        diceTray?.setActionAvailability?.(canSendStageAction());
        diceTray?.rejectPendingRolls?.("Socket error.");
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

    function showCardEditorFor(model) { return editors?.showCardEditorFor?.(model); }

    function hideCardEditor() { return editors?.hideCardEditor?.(); }

    function syncCardEditorWithSelection() { return editors?.syncCardEditorWithSelection?.(); }

    function mapEditorDraftFromState() { return editors?.mapEditorDraftFromState?.(); }
    function setMapEditorStatus(text) { if (mapEditorStatus) mapEditorStatus.textContent = text || ""; }
    function clampMapEditorPosition(left, top) { return firstTheaterMapGridModule?.clampPanelPosition?.(left, top, stageShell?.getBoundingClientRect?.(), mapEditorPanel?.getBoundingClientRect?.(), 560, 520) || { left: Math.round(left), top: Math.round(top) }; }
    function setMapEditorPosition(left, top) { if (!mapEditorPanel) return; const position = clampMapEditorPosition(left, top); mapEditorPanel.style.left = `${position.left}px`; mapEditorPanel.style.top = `${position.top}px`; mapEditorPanel.style.right = "auto"; mapEditorPanel.style.bottom = "auto"; }
    function clampGridEditorPosition(left, top) { return firstTheaterMapGridModule?.clampPanelPosition?.(left, top, stageShell?.getBoundingClientRect?.(), gridEditorPanel?.getBoundingClientRect?.(), 420, 480) || { left: Math.round(left), top: Math.round(top) }; }
    function setGridEditorPosition(left, top) { if (!gridEditorPanel) return; const position = clampGridEditorPosition(left, top); gridEditorPanel.style.left = `${position.left}px`; gridEditorPanel.style.top = `${position.top}px`; gridEditorPanel.style.right = "auto"; gridEditorPanel.style.bottom = "auto"; }
    function syncMapEditorPreview(url) { return editors?.syncMapEditorPreview?.(url); }
    function syncMapAssetList() { return editors?.syncMapAssetList?.(); }
    function syncMapEditorWithState() { return editors?.syncMapEditorWithState?.(); }
    function livePreviewMapOnStage() { return editors?.livePreviewMapOnStage?.(); }
    function openMapEditor() { return editors?.openMapEditor?.(); }
    function hideMapEditor() { return editors?.hideMapEditor?.(); }
    function cancelMapEditor() { return editors?.cancelMapEditor?.(); }
    function mapEditorPayloadFromUI(assetID) { return editors?.mapEditorPayloadFromUI?.(assetID); }
    async function refreshVenueMapAssets() { return editors?.refreshVenueMapAssets?.(); }
    async function refreshVenueMapState() { return editors?.refreshVenueMapState?.(); }
    async function uploadMapAsset(file) { return editors?.uploadMapAsset?.(file); }
    async function saveVenueMap() { return editors?.saveVenueMap?.(); }
    async function removeVenueMap() { return editors?.removeVenueMap?.(); }
    function defaultGridConfig() { return editors?.defaultGridConfig?.() || firstTheaterMapGridModule?.defaultGridConfig?.() || { grid_type: "none", hex_orientation: "flat-top", cell_size: 50, offset_x: 0, offset_y: 0, line_width: 1, opacity: 0.45, line_style: "neutral", visible: true }; }
    function gridEditorDraftFromUI() { return editors?.gridEditorDraftFromUI?.(); }
    function setGridEditorStatus(text) { if (gridEditorStatus) gridEditorStatus.textContent = text || ""; }
    function syncGridEditorHexFieldVisibility() { return editors?.syncGridEditorHexFieldVisibility?.(); }
    function syncGridEditorVisibilityButton() { return editors?.syncGridEditorVisibilityButton?.(); }
    function syncGridEditorWithState() { return editors?.syncGridEditorWithState?.(); }
    function liveSyncGrid() { return editors?.liveSyncGrid?.(); }
    function nudgeGridNumberField(inputEl, delta, min, max) { return editors?.nudgeGridNumberField?.(inputEl, delta, min, max); }
    function openGridEditor() { return editors?.openGridEditor?.(); }
    function hideGridEditor() { return editors?.hideGridEditor?.(); }
    function cancelGridEditor() { return editors?.cancelGridEditor?.(); }
    function resetGridEditorDraft() { return editors?.resetGridEditorDraft?.(); }
    function toggleGridVisibilityDraft() { return editors?.toggleGridVisibilityDraft?.(); }
    async function saveGridConfig() { return editors?.saveGridConfig?.(); }
    async function refreshVenueGridConfig() { return editors?.refreshVenueGridConfig?.(); }

    function venueMapTextureURLForState(state) {
      const assetURL = String(state?.asset?.content_url || "").trim();
      if (!assetURL) {
        return "";
      }
      const revision = String(state?.updated_at || "").trim();
      if (!revision) {
        return assetURL;
      }
      return `${assetURL}${assetURL.includes("?") ? "&" : "?"}v=${encodeURIComponent(revision)}`;
    }

    async function ensureVenueMapTexture() {
      const assetURL = venueMapTextureURLForState(currentVenueMapState);
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
          if (venueMapTextureURLForState(currentVenueMapState) === assetURL) {
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

    function clearVenueMapTexture() {
      venueMapTexture = null;
      venueMapTextureURL = "";
      venueMapTexturePromise = null;
      venueMapTextureFailedURL = "";
    }

    const computeStagePlayableBounds = (width, height) => {
      const displayMode = String(currentVenueMapState?.display_mode || "theater").trim() === "fullscreen" ? "fullscreen" : "theater";
      if (displayMode === "fullscreen") {
        return { x: 0, y: 0, width, height };
      }
      return firstTheaterGeometryModule?.computeStagePlayableBounds?.(width, height) || { x: 0, y: 0, width, height };
    };

    function renderVenueGridLayer(bounds, config = null) {
      if (!gridLayer || !window.PIXI) return;
      const resolvedBounds = bounds || getPlayableBounds();
      window.VictoryPixiGrid?.render?.(gridLayer, config || currentVenueGridConfig, resolvedBounds);
    }

    function renderVenueMapLayer(width, height) {
      if (!mapLayer || !window.PIXI) return;
      mapLayer.removeChildren();
      mapLayer.mask = null;
      const state = currentVenueMapState;
      const assetURL = venueMapTextureURLForState(state);
      if (!state || !state.asset || !assetURL) {
        clearVenueMapTexture();
        if (stageCamera && String(stageCamera.getView?.().activeMapId || "")) {
          stageCamera.setActiveMapId?.("", { reset: false });
        }
        return computeStagePlayableBounds(width, height);
      }

      if (!venueMapTexture || venueMapTextureURL !== assetURL) {
        if (venueMapTextureFailedURL === assetURL) {
          return computeStagePlayableBounds(width, height);
        }
        ensureVenueMapTexture().then((texture) => {
          if (!texture) return;
          if (venueMapTextureURLForState(currentVenueMapState) === assetURL) {
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
      return contextHelpers?.isSecondaryPointerEvent?.(event)
        || (Number(event?.button ?? event?.data?.button ?? -1) === 2 || (Number(event?.buttons ?? event?.data?.buttons ?? 0) & 2) === 2);
    }

    function markContextMenuHandled(event) {
      if (contextHelpers?.markContextMenuHandled) {
        contextHelpers.markContextMenuHandled(event);
        return;
      }
      if (!event) return;
      event.__firstTheaterContextMenuHandled = true;
      if (event.data?.originalEvent) {
        event.data.originalEvent.__firstTheaterContextMenuHandled = true;
      }
    }

    function wasContextMenuHandled(event) {
      if (contextHelpers?.wasContextMenuHandled) {
        return contextHelpers.wasContextMenuHandled(event);
      }
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
      if (contextHelpers?.hitTestContextMenuTarget) {
        return contextHelpers.hitTestContextMenuTarget(screenPoint, worldPoint);
      }
      return null;
    }

    function resolveContextMenuTargetFromClient(clientX, clientY) {
      if (contextHelpers?.resolveContextMenuTargetFromClient) {
        return contextHelpers.resolveContextMenuTargetFromClient(clientX, clientY);
      }
      const screenPoint = stageScreenPointFromClient(clientX, clientY);
      const stagePoint = stagePointFromClient(clientX, clientY);
      const objectModel = hitTestContextMenuTarget(screenPoint, stagePoint) || stageContextMenuModel();
      return { objectModel, stagePoint, screenPoint };
    }

    function eventClientPoint(event) {
      return contextHelpers?.eventClientPoint?.(event) || { clientX: 0, clientY: 0 };
    }

    function updatePointerReadout(screenX, screenY, stagePoint, note = "") {
      return contextHelpers?.updatePointerReadout?.(screenX, screenY, stagePoint, note);
    }

    function cancelContextMenuEvent(event) {
      return contextHelpers?.cancelContextMenuEvent?.(event);
    }

    function handleNativeStageContextMenu(event) {
      return actionRouter?.handleNativeStageContextMenu?.(event);
    }

    function openResolvedContextMenu(event) {
      return actionRouter?.openResolvedContextMenu?.(event);
    }

    function performStageObjectAction(action, objectModel) {
      return actionRouter?.performStageObjectAction?.(action, objectModel);
    }

    function openContextMenu(event, objectModel) {
      actionRouter?.openContextMenu?.(event, objectModel);
      if (!objectModel) {
        return;
      }
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

    function createIndexCardFromMenu() { return sessionSync?.createIndexCardFromMenu?.(); }

    function placeCreatedIndexCard(action) { return sessionSync?.placeCreatedIndexCard?.(action); }

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

    function renderTokenPickerList() { return tokenUi?.renderTokenPickerList?.(); }
    async function refreshWarehouseTokenAssets() { return tokenUi?.refreshWarehouseTokenAssets?.(); }
    function openTokenPicker(mode = "create", objectModel = null) { return tokenUi?.openTokenPicker?.(mode, objectModel); }
    function closeTokenPicker() { return tokenUi?.closeTokenPicker?.(); }
    function placeTokenAsset(asset, point, replaceTargetKey = "", replacing = false) { return tokenUi?.placeTokenAsset?.(asset, point, replaceTargetKey, replacing); }
    function syncTokenEditorWithSelection() { return tokenUi?.syncTokenEditorWithSelection?.(); }
    function openTokenEditor(model = null) { return tokenUi?.openTokenEditor?.(model); }
    function hideTokenEditor() { return tokenUi?.hideTokenEditor?.(); }
    function saveTokenEditor() { return tokenUi?.saveTokenEditor?.(); }

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

    function handleSocketMessage(msg) { return sessionSync?.handleSocketMessage?.(msg); }

    function applyVenueFocusPing(data) { return sessionSync?.applyVenueFocusPing?.(data); }

    function applySnapshot(snapshot) { return sessionSync?.applySnapshot?.(snapshot); }

    async function refreshWorld() { return sessionSync?.refreshWorld?.(); }

    async function joinCave() { return sessionSync?.joinCave?.(); }

    async function ensureJoinedCave() { return sessionSync?.ensureJoinedCave?.(); }

    async function joinCaveOnce() { return sessionSync?.joinCaveOnce?.(); }

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
    rightCharacterValue?.addEventListener("click", () => {
      const activeCharacterId = String(currentIdentity?.active_character?.character_card_id || "").trim();
      const target = activeCharacterId ? `/venues/greenroom/?character_id=${encodeURIComponent(activeCharacterId)}` : "/venues/greenroom/";
      window.location.href = target;
    });
    characterTrayButton?.addEventListener("click", () => {
      const activeCharacterId = String(currentIdentity?.active_character?.character_card_id || "").trim();
      const target = activeCharacterId ? `/venues/greenroom/?character_id=${encodeURIComponent(activeCharacterId)}` : "/venues/greenroom/";
      window.location.href = target;
    });
    rightCardEditorButton?.addEventListener("click", () => {
      const activeCharacterId = String(currentIdentity?.active_character?.character_card_id || "").trim();
      const target = activeCharacterId ? `/venues/greenroom/?character_id=${encodeURIComponent(activeCharacterId)}` : "/venues/greenroom/";
      window.location.href = target;
    });
    chatDiceTab?.addEventListener("click", () => setChatCompanionTab("dice"));
    chatHelpTab?.addEventListener("click", () => setChatCompanionTab("help"));
    setChatCompanionTab("dice");

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

    stageMapButton?.addEventListener("click", () => {
      openMapEditor();
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

    tokenPickerApply?.addEventListener("click", () => {
      tokenUi?.applySelectedTokenAsset?.();
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
      syncMapEditorPreview(mapEditorPreviewURL || currentVenueMapState?.asset?.content_url || "");
    });

    mapEditorFit?.addEventListener("change", () => {
      mapEditorDirty = true;
      syncMapEditorPreview(mapEditorPreviewURL || currentVenueMapState?.asset?.content_url || "");
      setMapEditorStatus("Map fit updated.");
    });

    mapEditorScale?.addEventListener("input", () => {
      mapEditorDirty = true;
      syncMapEditorPreview(mapEditorPreviewURL || currentVenueMapState?.asset?.content_url || "");
    });

    mapEditorCropX?.addEventListener("input", () => {
      mapEditorDirty = true;
      syncMapEditorPreview(mapEditorPreviewURL || currentVenueMapState?.asset?.content_url || "");
    });

    mapEditorCropY?.addEventListener("input", () => {
      mapEditorDirty = true;
      syncMapEditorPreview(mapEditorPreviewURL || currentVenueMapState?.asset?.content_url || "");
    });

    mapEditorSafeMargin?.addEventListener("input", () => {
      mapEditorDirty = true;
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
      if (suppressStageContextMenu || suppressEditorAutoClose) {
        suppressStageContextMenu = false;
        suppressEditorAutoClose = false;
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
      suppressEditorAutoClose = true;
      window.setTimeout(() => {
        suppressEditorAutoClose = false;
      }, 0);
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
