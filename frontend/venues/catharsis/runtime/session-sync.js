(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryCatharsisSessionSync = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function createSessionSync(deps = {}) {
    const fetchFn = typeof deps.fetch === "function" ? deps.fetch : (...args) => fetch(...args);
    const getCurrentIdentity = typeof deps.getCurrentIdentity === "function" ? deps.getCurrentIdentity : () => null;
    const setCurrentIdentity = typeof deps.setCurrentIdentity === "function" ? deps.setCurrentIdentity : () => {};
    const getCurrentRole = typeof deps.getCurrentRole === "function" ? deps.getCurrentRole : () => "audience";
    const setCurrentRole = typeof deps.setCurrentRole === "function" ? deps.setCurrentRole : () => {};
    const getCurrentSessionId = typeof deps.getCurrentSessionId === "function" ? deps.getCurrentSessionId : () => "";
    const setCurrentSessionId = typeof deps.setCurrentSessionId === "function" ? deps.setCurrentSessionId : () => {};
    const getCurrentActorId = typeof deps.getCurrentActorId === "function" ? deps.getCurrentActorId : () => "";
    const setCurrentActorId = typeof deps.setCurrentActorId === "function" ? deps.setCurrentActorId : () => {};
    const getCurrentSnapshot = typeof deps.getCurrentSnapshot === "function" ? deps.getCurrentSnapshot : () => null;
    const setCurrentSnapshot = typeof deps.setCurrentSnapshot === "function" ? deps.setCurrentSnapshot : () => {};
    const getCurrentObjects = typeof deps.getCurrentObjects === "function" ? deps.getCurrentObjects : () => [];
    const setCurrentObjects = typeof deps.setCurrentObjects === "function" ? deps.setCurrentObjects : () => {};
    const getCurrentSelection = typeof deps.getCurrentSelection === "function" ? deps.getCurrentSelection : () => null;
    const setCurrentSelection = typeof deps.setCurrentSelection === "function" ? deps.setCurrentSelection : () => {};
    const replaceProjectedState = typeof deps.replaceProjectedState === "function" ? deps.replaceProjectedState : () => {};
    const buildObjects = typeof deps.buildObjects === "function" ? deps.buildObjects : () => [];
    const applyVenueFocusPing = typeof deps.applyVenueFocusPing === "function" ? deps.applyVenueFocusPing : () => {};
    const updateStatusSummary = typeof deps.updateStatusSummary === "function" ? deps.updateStatusSummary : () => {};
    const setStageStatus = typeof deps.setStageStatus === "function" ? deps.setStageStatus : () => {};
    const setMovementLine = typeof deps.setMovementLine === "function" ? deps.setMovementLine : () => {};
    const setSnapshotSummary = typeof deps.setSnapshotSummary === "function" ? deps.setSnapshotSummary : () => {};
    const setLiveFeedLine = typeof deps.setLiveFeedLine === "function" ? deps.setLiveFeedLine : () => {};
    const setSelectionLine = typeof deps.setSelectionLine === "function" ? deps.setSelectionLine : () => {};
    const updateChatPresentation = typeof deps.updateChatPresentation === "function" ? deps.updateChatPresentation : () => {};
    const renderPixiScene = typeof deps.renderPixiScene === "function" ? deps.renderPixiScene : () => {};
    const syncDiceTrayFromSnapshot = typeof deps.syncDiceTrayFromSnapshot === "function" ? deps.syncDiceTrayFromSnapshot : () => {};
    const handleDiceTrayAction = typeof deps.handleDiceTrayAction === "function" ? deps.handleDiceTrayAction : () => {};
    const handleDiceTrayError = typeof deps.handleDiceTrayError === "function" ? deps.handleDiceTrayError : () => {};
    const rejectPendingDiceTrayRolls = typeof deps.rejectPendingDiceTrayRolls === "function" ? deps.rejectPendingDiceTrayRolls : () => {};
    const syncCurrentObjectsFromProjectedState = typeof deps.syncCurrentObjectsFromProjectedState === "function" ? deps.syncCurrentObjectsFromProjectedState : () => false;
    const canManageIndexCards = typeof deps.canManageIndexCards === "function" ? deps.canManageIndexCards : () => false;
    const canManageStageTokens = typeof deps.canManageStageTokens === "function" ? deps.canManageStageTokens : () => false;
    const canActorRevealHideStageObjects = typeof deps.canActorRevealHideStageObjects === "function" ? deps.canActorRevealHideStageObjects : () => false;
    const cardFaceForModel = typeof deps.cardFaceForModel === "function" ? deps.cardFaceForModel : () => "front";
    const cardDisplayMode = typeof deps.cardDisplayMode === "function" ? deps.cardDisplayMode : () => "overlay";
    const tokenScaleForModel = typeof deps.tokenScaleForModel === "function" ? deps.tokenScaleForModel : () => 100;
    const tokenSnapModeForModel = typeof deps.tokenSnapModeForModel === "function" ? deps.tokenSnapModeForModel : () => "free";
    const tokenLayerForModel = typeof deps.tokenLayerForModel === "function" ? deps.tokenLayerForModel : () => "public";
    const closeContextMenu = typeof deps.closeContextMenu === "function" ? deps.closeContextMenu : () => {};
    const selectObject = typeof deps.selectObject === "function" ? deps.selectObject : () => {};
    const syncSelectedActions = typeof deps.syncSelectedActions === "function" ? deps.syncSelectedActions : () => {};
    const syncCardEditorWithSelection = typeof deps.syncCardEditorWithSelection === "function" ? deps.syncCardEditorWithSelection : () => {};
    const syncTokenEditorWithSelection = typeof deps.syncTokenEditorWithSelection === "function" ? deps.syncTokenEditorWithSelection : () => {};
    const refreshVenueGridConfig = typeof deps.refreshVenueGridConfig === "function" ? deps.refreshVenueGridConfig : async () => {};
    const refreshVenueMapState = typeof deps.refreshVenueMapState === "function" ? deps.refreshVenueMapState : async () => {};
    const refreshVenueMapAssets = typeof deps.refreshVenueMapAssets === "function" ? deps.refreshVenueMapAssets : async () => {};
    const loadPixiLibrary = typeof deps.loadPixiLibrary === "function" ? deps.loadPixiLibrary : async () => {};
    const initializePixi = typeof deps.initializePixi === "function" ? deps.initializePixi : async () => {};
    const initializeShellChrome = typeof deps.initializeShellChrome === "function" ? deps.initializeShellChrome : () => {};
    const updateShellMetaPresentation = typeof deps.updateShellMetaPresentation === "function" ? deps.updateShellMetaPresentation : () => {};
    const updateShellTargetPresentation = typeof deps.updateShellTargetPresentation === "function" ? deps.updateShellTargetPresentation : () => {};
    const updateHeaderPresentation = typeof deps.updateHeaderPresentation === "function" ? deps.updateHeaderPresentation : () => {};
    const setRendererFallback = typeof deps.setRendererFallback === "function" ? deps.setRendererFallback : () => {};
    const setMovementReport = typeof deps.setMovementReport === "function" ? deps.setMovementReport : () => {};
    const appendSystemChatNotice = typeof deps.appendSystemChatNotice === "function" ? deps.appendSystemChatNotice : () => {};
    const chatClosedMessage = typeof deps.chatClosedMessage === "function" ? deps.chatClosedMessage : () => "Chat closed.";
    const getCurrentVenueMapState = typeof deps.getCurrentVenueMapState === "function" ? deps.getCurrentVenueMapState : () => null;
    const getCurrentVenueMapAssetID = typeof deps.getCurrentVenueMapAssetID === "function" ? deps.getCurrentVenueMapAssetID : () => "";
    const getCurrentVenueGridConfig = typeof deps.getCurrentVenueGridConfig === "function" ? deps.getCurrentVenueGridConfig : () => null;
    const roleLabel = typeof deps.roleLabel === "function" ? deps.roleLabel : (role) => String(role || "audience");
    const getSessionResponseHeaders = typeof deps.getSessionResponseHeaders === "function" ? deps.getSessionResponseHeaders : () => ({});
    const getStageStatus = typeof deps.getStageStatus === "function" ? deps.getStageStatus : () => "";
    const getStagePlacementCandidate = typeof deps.getStagePlacementCandidate === "function" ? deps.getStagePlacementCandidate : () => null;
    const setStagePlacementCandidate = typeof deps.setStagePlacementCandidate === "function" ? deps.setStagePlacementCandidate : () => {};
    const getStagePlacementScreenCandidate = typeof deps.getStagePlacementScreenCandidate === "function" ? deps.getStagePlacementScreenCandidate : () => null;
    const setStagePlacementScreenCandidate = typeof deps.setStagePlacementScreenCandidate === "function" ? deps.setStagePlacementScreenCandidate : () => {};
    const getPendingStageCardPlacement = typeof deps.getPendingStageCardPlacement === "function" ? deps.getPendingStageCardPlacement : () => null;
    const setPendingStageCardPlacement = typeof deps.setPendingStageCardPlacement === "function" ? deps.setPendingStageCardPlacement : () => {};
    const setSmokeLine = typeof deps.setSmokeLine === "function" ? deps.setSmokeLine : () => {};
    const normalizeOverlayPoint = typeof deps.normalizeOverlayPoint === "function" ? deps.normalizeOverlayPoint : (point) => ({ screen_x: Number(point?.x || 0), screen_y: Number(point?.y || 0) });
    const setRecentPlacementMarker = typeof deps.setRecentPlacementMarker === "function" ? deps.setRecentPlacementMarker : () => {};
    const smokeMode = Boolean(deps.smokeMode);

    let currentSnapshot = null;
    let currentObjects = [];
    let currentIdentity = null;
    let currentRole = "audience";
    let currentSessionId = "";
    let currentActorId = "";
    let latestFocusEventStamp = 0;
    let worldRefreshSerial = 0;

    function setSnapshot(value) {
      currentSnapshot = value || null;
    }

    function applySnapshot(snapshot) {
      replaceProjectedState(snapshot, { viewerRole: getCurrentRole() });
      setSnapshot(snapshot);
      setCurrentSnapshot(snapshot || null);
      const nextObjects = buildObjects(snapshot);
      currentObjects = nextObjects;
      setCurrentObjects(nextObjects);

      const selection = getCurrentSelection();
      if (selection && !nextObjects.find((object) => object.key === selection.key)) {
        setCurrentSelection(null);
      } else if (selection) {
        const refreshedSelection = nextObjects.find((object) => object.key === selection.key);
        if (refreshedSelection) {
          setCurrentSelection(refreshedSelection);
        }
      }

      updateStatusSummary();
      setStageStatus(`Snapshot loaded. ${nextObjects.length} rendered object${nextObjects.length === 1 ? "" : "s"}.`);
      setMovementLine("Idle");

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
      const showingState = showing && String(showing.status || "").toLowerCase() !== "closed"
        ? "active"
        : "closed";
      const overlayState = overlay ? `${overlay.overlay_type || "text"} overlay active` : "no overlay";
      setSnapshotSummary(`Showing is ${showingState}; ${overlayState}; fire ${latestFire ? "is present" : "not present"}; staged cards ${stageCards.length}; staged tokens ${stageTokens.length}.`);

      if (!getCurrentSelection()) {
        setSelectionLine("None");
        setLiveFeedLine("No live object selected.");
      }

      syncSelectedActions();
      syncCardEditorWithSelection();
      syncTokenEditorWithSelection();
      syncDiceTrayFromSnapshot(snapshot);
      renderPixiScene();
    }

    function applyVenueFocusPingWrapper(data) {
      const stamp = Number(Date.parse(String(data?.ts || "")) || Date.now());
      if (stamp < latestFocusEventStamp) {
        return;
      }
      latestFocusEventStamp = stamp;
      applyVenueFocusPing(data);
    }

    async function refreshWorld() {
      const refreshSerial = ++worldRefreshSerial;
      try {
        if (!currentSessionId) {
          await ensureJoinedCave();
        }
        const response = await fetchFn("/api/world/the-cave", {
          method: "GET",
          credentials: "include",
          cache: "no-store",
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok || !payload?.data) {
          throw new Error(payload?.error || `HTTP ${response.status}`);
        }
        if (refreshSerial !== worldRefreshSerial) return;
        applySnapshot(payload.data);
      } catch (error) {
        if (refreshSerial !== worldRefreshSerial) return;
        console.error("refreshWorld failed", error);
        setStageStatus(`Snapshot refresh failed: ${error.message || String(error)}`);
      }
    }

    async function joinCaveOnce() {
      setStageStatus("Joining the live Cave session...");

      let sessionHandle = String(currentIdentity?.handle || "web").trim() || "web";
      let sessionDisplayName = String(currentIdentity?.display_name || "Friend").trim() || "Friend";
      if (!currentIdentity) {
        try {
          const sessionResponse = await fetchFn("/api/session/me", { credentials: "include" });
          const sessionPayload = await sessionResponse.json().catch(() => null);
          const session = sessionPayload?.data || {};
          currentIdentity = session;
          setCurrentIdentity(session);
          sessionHandle = String(session.handle || sessionHandle).trim() || sessionHandle;
          sessionDisplayName = String(session.display_name || sessionDisplayName).trim() || sessionDisplayName;
        } catch (error) {
          console.warn("session identity lookup failed", error);
        }
      }

      const response = await fetchFn("/api/session/the-cave/join", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
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
      const requestedRole = String(data.role || data.participant?.role || "audience").trim().toLowerCase();
      const isOperator = Boolean(currentIdentity?.is_operator);
      currentRole = isOperator && requestedRole === "audience" ? "producer" : requestedRole;
      currentSessionId = data.session_id || data.session?.id || data.sessionId || "";
      currentActorId = data.actor_id || data.participant?.actor_id || data.participant?.user_id || data.user_id || "";
      setCurrentRole(currentRole);
      setCurrentSessionId(currentSessionId);
      setCurrentActorId(currentActorId);

      if (!currentSessionId || !currentActorId) {
        throw new Error("Join succeeded but did not return session_id and actor_id.");
      }

      updateStatusSummary();
      updateShellMetaPresentation();
      setStageStatus(`Joined as ${roleLabel(currentRole)}.`);
    }

    async function ensureJoinedCave() {
      if (currentSessionId && currentActorId) return;
      await joinCave();
    }

    async function joinCave() {
      return joinCaveOnce();
    }

    function createIndexCardFromMenu() {
      if (!canManageIndexCards(getCurrentRole())) {
        setStageStatus("Only producers and directors can create cards.");
        return;
      }

      const stagePlacementCandidate = getStagePlacementCandidate();
      if (!stagePlacementCandidate) {
        setStageStatus("No stage placement point available.");
        return;
      }

      const stagePlacementScreenCandidate = getStagePlacementScreenCandidate();
      setPendingStageCardPlacement(stagePlacementScreenCandidate || null);
      setStagePlacementCandidate(null);
      setStagePlacementScreenCandidate(null);
      const placement = getPendingStageCardPlacement();
      const placementLabel = placement ? `x ${placement.x}, y ${placement.y}` : "x 0, y 0";
      const sent = deps.sendAction?.("create/index_card", {
        front_text: "New Index Card",
        back_text: "",
        color: "#d9c7a6",
      });
      if (!sent) {
        setPendingStageCardPlacement(null);
        setStageStatus("Socket unavailable.");
        closeContextMenu();
        return;
      }

      setStageStatus(`Index card created. Attaching to screen at ${placementLabel}...`);
      setMovementLine(`Awaiting placement at ${placementLabel}.`);
      if (smokeMode) {
        setSmokeLine(`Create sent for ${placementLabel}.`);
      }
      closeContextMenu();
    }

    function placeCreatedIndexCard(action) {
      const pendingStageCardPlacement = getPendingStageCardPlacement();
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
      if (!elementId && !elementSlug) return;

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
      deps.setLocalPositionOverrideForModel?.(createdCardModel, pendingStageCardPlacement);
      if (smokeMode) {
        setRecentPlacementMarker(pendingStageCardPlacement, createdCard?.name || createdCard?.data?.front_text || "Index card");
      } else {
        setRecentPlacementMarker(null);
      }

      const normalizedPlacement = normalizeOverlayPoint(pendingStageCardPlacement);
      const updated = deps.sendAction?.("update/index_card", {
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
        setPendingStageCardPlacement(null);
        closeContextMenu();
        return;
      }

      const ok = deps.sendAction?.("act/place_element", {
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
        setPendingStageCardPlacement(null);
        closeContextMenu();
        return;
      }

      setStageStatus(`Index card placed at x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}.`);
      setMovementLine(`Index card placed at x ${pendingStageCardPlacement.x}, y ${pendingStageCardPlacement.y}.`);
      if (smokeMode) {
        setSmokeLine(`Placed smoke card at ${pendingStageCardPlacement.x}, ${pendingStageCardPlacement.y}.`);
      }
      setPendingStageCardPlacement(null);
    }

    async function handleSocketMessage(msg) {
      if (!msg) return null;

      if (msg.kind === "snapshot") {
        applySnapshot(msg.snapshot || msg.message?.data || null);
        return msg;
      }

      if (msg.kind === "focus_ping" && msg.data) {
        applyVenueFocusPingWrapper(msg.data);
        return msg;
      }

      if (msg.kind === "action" && msg.action) {
        const actionType = msg.action.type || "";
        if (actionType === "roll/dice") {
          handleDiceTrayAction(msg.action);
          return msg;
        }
        if (actionType === "chat/message") {
          deps.appendChatActionLine?.(msg.action);
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
          const synced = syncCurrentObjectsFromProjectedState();
          if (!synced) {
            await refreshWorld();
          }
          if (actionType === "create/index_card") {
            deps.placeCreatedIndexCard?.(msg.action);
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
              item.kind === "token" &&
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
          if (synced) {
            renderPixiScene();
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
        handleDiceTrayError(errorText, msg);
        setStageStatus(`Action denied: ${errorText}`);
        setMovementLine(`Denied: ${errorText}`);
        if (errorText === "showing_closed") {
          appendSystemChatNotice(chatClosedMessage());
        } else if (errorText && errorText !== "chat/message") {
          appendSystemChatNotice(`Action denied: ${errorText}`);
        }
        return msg;
      }

      return msg;
    }

    return {
      applySnapshot,
      applyVenueFocusPing: applyVenueFocusPingWrapper,
      refreshWorld,
      joinCave,
      ensureJoinedCave,
      joinCaveOnce,
      createIndexCardFromMenu,
      placeCreatedIndexCard,
      handleSocketMessage,
      syncDiceTrayFromSnapshot,
      handleDiceTrayAction,
      handleDiceTrayError,
      rejectPendingDiceTrayRolls,
      getCurrentSnapshot: () => getCurrentSnapshot() || currentSnapshot,
      getCurrentObjects: () => getCurrentObjects() || currentObjects,
      getCurrentRole: () => getCurrentRole() || currentRole,
      getCurrentSessionId: () => getCurrentSessionId() || currentSessionId,
      getCurrentActorId: () => getCurrentActorId() || currentActorId,
    };
  }

  return { createSessionSync };
});
