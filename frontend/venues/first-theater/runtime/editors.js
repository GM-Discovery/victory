(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryFirstTheaterEditors = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function createEditorControllers(deps = {}) {
    const getStageShell = typeof deps.getStageShell === "function" ? deps.getStageShell : () => null;
    const getCurrentRole = typeof deps.getCurrentRole === "function" ? deps.getCurrentRole : () => "audience";
    const getCurrentSelection = typeof deps.getCurrentSelection === "function" ? deps.getCurrentSelection : () => null;
    const getCurrentObjects = typeof deps.getCurrentObjects === "function" ? deps.getCurrentObjects : () => [];
    const getCurrentVenueMapState = typeof deps.getCurrentVenueMapState === "function" ? deps.getCurrentVenueMapState : () => null;
    const getCurrentVenueMapAssetID = typeof deps.getCurrentVenueMapAssetID === "function" ? deps.getCurrentVenueMapAssetID : () => "";
    const getCurrentVenueMapAssets = typeof deps.getCurrentVenueMapAssets === "function" ? deps.getCurrentVenueMapAssets : () => [];
    const setCurrentVenueMapState = typeof deps.setCurrentVenueMapState === "function" ? deps.setCurrentVenueMapState : () => {};
    const setCurrentVenueMapAssetID = typeof deps.setCurrentVenueMapAssetID === "function" ? deps.setCurrentVenueMapAssetID : () => {};
    const setCurrentVenueMapAssets = typeof deps.setCurrentVenueMapAssets === "function" ? deps.setCurrentVenueMapAssets : () => {};
    const getCurrentVenueGridConfig = typeof deps.getCurrentVenueGridConfig === "function" ? deps.getCurrentVenueGridConfig : () => null;
    const setCurrentVenueGridConfig = typeof deps.setCurrentVenueGridConfig === "function" ? deps.setCurrentVenueGridConfig : () => {};
    const getCurrentSnapshot = typeof deps.getCurrentSnapshot === "function" ? deps.getCurrentSnapshot : () => null;
    const canManageIndexCards = typeof deps.canManageIndexCards === "function" ? deps.canManageIndexCards : () => false;
    const canManageStageTokens = typeof deps.canManageStageTokens === "function" ? deps.canManageStageTokens : () => false;
    const isCardObject = typeof deps.isCardObject === "function" ? deps.isCardObject : () => false;
    const isTokenObject = typeof deps.isTokenObject === "function" ? deps.isTokenObject : () => false;
    const objectState = typeof deps.objectState === "function" ? deps.objectState : () => ({ locked: false });
    const selectObject = typeof deps.selectObject === "function" ? deps.selectObject : () => {};
    const setStageStatus = typeof deps.setStageStatus === "function" ? deps.setStageStatus : () => {};
    const setMovementLine = typeof deps.setMovementLine === "function" ? deps.setMovementLine : () => {};
    const renderPixiScene = typeof deps.renderPixiScene === "function" ? deps.renderPixiScene : () => {};
    const sendAction = typeof deps.sendAction === "function" ? deps.sendAction : () => false;
    const escapeHtml = typeof deps.escapeHtml === "function"
      ? deps.escapeHtml
      : (value) => String(value)
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll("\"", "&quot;")
        .replaceAll("'", "&#39;");
    const updateLocalObjectModel = typeof deps.updateLocalObjectModel === "function" ? deps.updateLocalObjectModel : () => {};
    const updateLocalCardPinModel = typeof deps.updateLocalCardPinModel === "function" ? deps.updateLocalCardPinModel : () => {};
    const updateTokenLocalModel = typeof deps.updateTokenLocalModel === "function" ? deps.updateTokenLocalModel : () => {};
    const setLocalPositionOverrideForModel = typeof deps.setLocalPositionOverrideForModel === "function" ? deps.setLocalPositionOverrideForModel : () => null;
    const currentDisplayedPointForModel = typeof deps.currentDisplayedPointForModel === "function" ? deps.currentDisplayedPointForModel : () => ({ x: 0, y: 0 });
    const tokenScaleForModel = typeof deps.tokenScaleForModel === "function" ? deps.tokenScaleForModel : () => 100;
    const tokenSnapModeForModel = typeof deps.tokenSnapModeForModel === "function" ? deps.tokenSnapModeForModel : () => "free";
    const tokenLayerForModel = typeof deps.tokenLayerForModel === "function" ? deps.tokenLayerForModel : () => "public";
    const tokenPlacementPointForCreate = typeof deps.tokenPlacementPointForCreate === "function" ? deps.tokenPlacementPointForCreate : (point) => point;
    const tokenDuplicatePlacementForModel = typeof deps.tokenDuplicatePlacementForModel === "function" ? deps.tokenDuplicatePlacementForModel : () => null;
    const cardDuplicatePlacementForModel = typeof deps.cardDuplicatePlacementForModel === "function" ? deps.cardDuplicatePlacementForModel : () => null;
    const tokenMoveTargetForModel = typeof deps.tokenMoveTargetForModel === "function" ? deps.tokenMoveTargetForModel : () => null;
    const cardMoveTargetForModel = typeof deps.cardMoveTargetForModel === "function" ? deps.cardMoveTargetForModel : () => null;
    const fetchFn = typeof deps.fetch === "function" ? deps.fetch : (...args) => fetch(...args);
    const refreshVenueMapAssetsFn = typeof deps.refreshVenueMapAssets === "function" ? deps.refreshVenueMapAssets : async () => {};
    const refreshVenueMapStateFn = typeof deps.refreshVenueMapState === "function" ? deps.refreshVenueMapState : async () => {};
    const refreshVenueGridConfigFn = typeof deps.refreshVenueGridConfig === "function" ? deps.refreshVenueGridConfig : async () => {};
    const uploadMapAssetFn = typeof deps.uploadMapAsset === "function" ? deps.uploadMapAsset : async () => null;
    const renderVenueGridLayer = typeof deps.renderVenueGridLayer === "function" ? deps.renderVenueGridLayer : () => {};
    const setMapEditorStatus = typeof deps.setMapEditorStatus === "function" ? deps.setMapEditorStatus : () => {};
    const setGridEditorStatus = typeof deps.setGridEditorStatus === "function" ? deps.setGridEditorStatus : () => {};
    const setMapEditorPosition = typeof deps.setMapEditorPosition === "function" ? deps.setMapEditorPosition : () => {};
    const setGridEditorPosition = typeof deps.setGridEditorPosition === "function" ? deps.setGridEditorPosition : () => {};
    const clampMapEditorPosition = typeof deps.clampMapEditorPosition === "function" ? deps.clampMapEditorPosition : (left, top) => ({ left, top });
    const clampGridEditorPosition = typeof deps.clampGridEditorPosition === "function" ? deps.clampGridEditorPosition : (left, top) => ({ left, top });
    const clampCardEditorPosition = typeof deps.clampCardEditorPosition === "function" ? deps.clampCardEditorPosition : (left, top) => ({ left, top });
    const getCardEditorTarget = typeof deps.getCardEditorTarget === "function" ? deps.getCardEditorTarget : () => null;
    const setCardEditorPosition = typeof deps.setCardEditorPosition === "function" ? deps.setCardEditorPosition : () => {};
    const hideTokenEditor = typeof deps.hideTokenEditor === "function" ? deps.hideTokenEditor : () => {};
    const closeTokenPicker = typeof deps.closeTokenPicker === "function" ? deps.closeTokenPicker : () => {};
    const openTokenPicker = typeof deps.openTokenPicker === "function" ? deps.openTokenPicker : () => {};
    const canActorRevealHideStageObjects = typeof deps.canActorRevealHideStageObjects === "function" ? deps.canActorRevealHideStageObjects : () => false;
    const setCurrentSelection = typeof deps.setCurrentSelection === "function" ? deps.setCurrentSelection : () => {};
    const syncSelectedActions = typeof deps.syncSelectedActions === "function" ? deps.syncSelectedActions : () => {};
    const syncTokenEditorWithSelection = typeof deps.syncTokenEditorWithSelection === "function" ? deps.syncTokenEditorWithSelection : () => {};
    const setCardEditorTargetKey = typeof deps.setCardEditorTargetKey === "function" ? deps.setCardEditorTargetKey : () => {};
    const setCardEditorDirty = typeof deps.setCardEditorDirty === "function" ? deps.setCardEditorDirty : () => {};
    const getMapEditorOriginalState = typeof deps.getMapEditorOriginalState === "function" ? deps.getMapEditorOriginalState : () => null;
    const setMapEditorOriginalState = typeof deps.setMapEditorOriginalState === "function" ? deps.setMapEditorOriginalState : () => {};
    const setMapEditorDirty = typeof deps.setMapEditorDirty === "function" ? deps.setMapEditorDirty : () => {};
    const getGridEditorOriginalState = typeof deps.getGridEditorOriginalState === "function" ? deps.getGridEditorOriginalState : () => null;
    const setGridEditorOriginalState = typeof deps.setGridEditorOriginalState === "function" ? deps.setGridEditorOriginalState : () => {};
    const setGridEditorDirty = typeof deps.setGridEditorDirty === "function" ? deps.setGridEditorDirty : () => {};
    let mapEditorSelectedAssetID = "";
    const getStageShellRect = () => getStageShell()?.getBoundingClientRect?.();

    function currentLiveSelection() {
      const selection = getCurrentSelection();
      if (!selection) return null;
      return getCurrentObjects().find((item) => item.key === selection.key) || selection;
    }

    function showCardEditorFor(model) {
      const cardEditorPanel = deps.cardEditorPanel;
      const cardEditorFront = deps.cardEditorFront;
      const cardEditorBack = deps.cardEditorBack;
      const cardEditorColor = deps.cardEditorColor;
      const cardEditorStatus = deps.cardEditorStatus;
      if (!cardEditorPanel || !cardEditorFront || !cardEditorBack || !cardEditorColor) return;
      if (!model || !model.live || !isCardObject(model) || objectState(model).locked) {
        setCardEditorTargetKey("");
        setCardEditorDirty(false);
        cardEditorPanel.hidden = true;
        if (cardEditorStatus) cardEditorStatus.textContent = "Select a live index card.";
        return;
      }
      const draft = deps.cardEditorDraftFromModel?.(model) || {};
      setCardEditorTargetKey(model.key);
      setCardEditorDirty(false);
      if (cardEditorPanel.hidden) {
        setCardEditorPosition(24, 24);
      }
      cardEditorPanel.hidden = false;
      cardEditorFront.value = draft.frontText || "";
      cardEditorBack.value = draft.backText || "";
      cardEditorColor.value = draft.color || "#d9c7a6";
      if (cardEditorStatus) cardEditorStatus.textContent = `${draft.label || model.label || "Card"} ready to inspect.`;
    }

    function syncCardEditorWithSelection() {
      const cardEditorPanel = deps.cardEditorPanel;
      if (!cardEditorPanel || cardEditorPanel.hidden) return;
      const target = getCardEditorTarget();
      if (!target || !target.live || !isCardObject(target) || objectState(target).locked) {
        hideCardEditor();
        return;
      }
      if (getCurrentSelection() && getCurrentSelection().key !== target.key) {
        hideCardEditor();
        return;
      }
      if (deps.cardEditorDirty?.()) {
        return;
      }
      showCardEditorFor(target);
    }

    function hideCardEditor() {
      const cardEditorPanel = deps.cardEditorPanel;
      const cardEditorStatus = deps.cardEditorStatus;
      if (!cardEditorPanel) return;
      setCardEditorTargetKey("");
      setCardEditorDirty(false);
      cardEditorPanel.hidden = true;
      if (cardEditorStatus) cardEditorStatus.textContent = "Select a live index card.";
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
      hideMapEditorController();
      hideGridEditorController();
      showCardEditorFor(target);
      selectObject(target, `${target.label} selected.`);
      deps.cardEditorFront?.focus?.({ preventScroll: true });
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
      const frontText = String(deps.cardEditorFront?.value || "").trim();
      const backText = String(deps.cardEditorBack?.value || "").trim();
      const color = String(deps.cardEditorColor?.value || "").trim() || target.color || "#d9c7a6";
      const sent = sendAction("update/index_card", {
        element_id: target.elementId || "",
        element_slug: target.elementSlug || "",
        front_text: frontText,
        back_text: backText,
        color,
      });
      if (!sent) {
        setStageStatus("Socket unavailable.");
        if (deps.cardEditorStatus) deps.cardEditorStatus.textContent = "Save failed: socket unavailable.";
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
      deps.setCardEditorDirty?.(false);
      setStageStatus(`Saved ${target.label}.`);
      if (deps.cardEditorStatus) deps.cardEditorStatus.textContent = `${target.label} save sent.`;
      setMovementLine(`update/index_card sent for ${target.label}.`);
    }

    function toggleCardEditorFromSelection() {
      const selection = getCurrentSelection();
      if (!selection || !isCardObject(selection) || !selection.live) {
        return;
      }
      if (deps.cardEditorPanel && deps.cardEditorPanel.hidden) {
        openCardEditor(selection);
      } else {
        hideCardEditor();
      }
    }

    function hideCardEditorController() {
      hideCardEditor();
    }

    function hideMapEditorController() {
      setMapEditorDirty(false);
      setMapEditorOriginalState(null);
      mapEditorSelectedAssetID = "";
      const panel = deps.getMapEditorElements?.().panel;
      if (panel) panel.hidden = true;
      setMapEditorStatus("Choose a map asset for the First Theater stage.");
      deps.setMapEditorPreviewURL?.("");
    }

    function hideGridEditorController() {
      const original = getGridEditorOriginalState();
      if (original) {
        setCurrentVenueGridConfig({ ...original });
        deps.renderVenueGridLayer?.(deps.getCurrentVenueMapBounds?.() || deps.getPlayableBounds?.());
      }
      setGridEditorDirty(false);
      setGridEditorOriginalState(null);
      const panel = deps.getGridEditorElements?.().panel;
      if (panel) panel.hidden = true;
      setGridEditorStatus("Choose a grid style for the First Theater stage.");
    }

    function syncMapEditorPreview(url) {
      const elements = deps.getMapEditorElements?.() || {};
      if (!elements.preview) return;
      if (deps.getMapEditorPreviewURL?.() && deps.getMapEditorPreviewURL?.() !== url && String(deps.getMapEditorPreviewURL?.() || "").startsWith("blob:")) {
        try {
          URL.revokeObjectURL(deps.getMapEditorPreviewURL());
        } catch (error) {
          console.warn("map preview revoke failed", error);
        }
      }
      deps.setMapEditorPreviewURL?.(url || "");
      elements.preview.src = deps.getMapEditorPreviewURL?.() || deps.getCurrentVenueMapState?.()?.asset?.content_url || "";
      const draft = deps.mapEditorDraftFromState?.() || {};
      const fitMode = draft.fit === "contain" ? "contain" : "cover";
      elements.preview.style.objectFit = fitMode;
      elements.preview.style.objectPosition = `${Math.round((draft.cropX || 0) * 100)}% ${Math.round((draft.cropY || 0) * 100)}%`;
      elements.preview.style.transform = `scale(${draft.scale || 1})`;
      if (elements.previewMode) elements.previewMode.textContent = fitMode;
      if (elements.previewFocus) {
        elements.previewFocus.style.setProperty("--map-crop-x", `${Math.round((draft.cropX || 0) * 100)}%`);
        elements.previewFocus.style.setProperty("--map-crop-y", `${Math.round((draft.cropY || 0) * 100)}%`);
      }
      if (elements.previewSafe) {
        elements.previewSafe.style.setProperty("--map-safe-margin", `${draft.safeMargin || 24}px`);
      }
    }

    function mapEditorDraftFromState() {
      const state = getCurrentVenueMapState() || {};
      const controls = deps.getMapEditorElements?.() || {};
      const selectedAssetID = String(mapEditorSelectedAssetID || getCurrentVenueMapAssetID() || state.asset_id || "").trim();
      return {
        assetID: selectedAssetID,
        displayMode: String(controls.displayMode?.value || state.display_mode || "theater").trim() === "fullscreen" ? "fullscreen" : "theater",
        fit: String(controls.fit?.value || state.fit || "cover"),
        cropX: Number.isFinite(Number(controls.cropX?.value)) ? Math.max(0, Math.min(1, Number(controls.cropX.value))) : Number(state.crop_x ?? 0.5),
        cropY: Number.isFinite(Number(controls.cropY?.value)) ? Math.max(0, Math.min(1, Number(controls.cropY.value))) : Number(state.crop_y ?? 0.5),
        scale: Number.isFinite(Number(controls.scale?.value)) ? Math.max(0.25, Math.min(4, Number(controls.scale.value))) : Number(state.scale ?? 1),
        safeMargin: Number.isFinite(Number(controls.safeMargin?.value)) ? Math.max(0, Math.min(128, Number(controls.safeMargin.value))) : Number(state.safe_margin ?? 24),
      };
    }

    function mapEditorPayloadFromUI(assetID) {
      const draft = mapEditorDraftFromState();
      const controls = deps.getMapEditorElements?.() || {};
      return {
        asset_id: String(assetID || draft.assetID || getCurrentVenueMapAssetID() || "").trim(),
        display_mode: draft.displayMode,
        fit: String(controls.fit?.value || draft.fit || "cover"),
        crop_x: Number.isFinite(Number(controls.cropX?.value)) ? Math.max(0, Math.min(1, Number(controls.cropX.value))) : draft.cropX,
        crop_y: Number.isFinite(Number(controls.cropY?.value)) ? Math.max(0, Math.min(1, Number(controls.cropY.value))) : draft.cropY,
        scale: Number.isFinite(Number(controls.scale?.value)) ? Math.max(0.25, Math.min(4, Number(controls.scale.value))) : draft.scale,
        safe_margin: Math.round(Number.isFinite(Number(controls.safeMargin?.value)) ? Math.max(0, Math.min(128, Number(controls.safeMargin.value))) : draft.safeMargin),
      };
    }

    function syncMapAssetList() {
      const elements = deps.getMapEditorElements?.() || {};
      const assets = Array.isArray(getCurrentVenueMapAssets()) ? getCurrentVenueMapAssets() : [];
      if (!elements.assets) return;
      elements.assets.innerHTML = "";
      if (!assets.length) {
        const empty = document.createElement("div");
        empty.className = "map-editor-status";
        empty.textContent = "No active map assets found.";
        elements.assets.appendChild(empty);
        return;
      }
      const currentAssetID = String(mapEditorSelectedAssetID || getCurrentVenueMapAssetID() || getCurrentVenueMapState()?.asset_id || "").trim();
      assets.forEach((asset) => {
        const assetID = String(asset.id || asset.asset_id || "").trim();
        const name = String(asset.name || asset.asset_name || asset.original_filename || assetID || "Map asset");
        const thumbURL = String(asset.thumbnail_url || asset.thumbnailURL || asset.content_url || asset.contentURL || "");
        const bytes = Number(asset.stored_bytes || asset.byte_size || 0);
        const button = document.createElement("button");
        button.type = "button";
        button.className = "map-editor-asset map-asset-button";
        if (assetID === currentAssetID) button.classList.add("is-selected");
        button.innerHTML = `
          <img src="${escapeHtml(thumbURL)}" alt="${escapeHtml(name)}" />
          <div class="map-editor-asset-meta">
            <strong>${escapeHtml(name)}</strong>
            <small>${escapeHtml(String(asset.asset_type || "map"))} · ${escapeHtml(String(Math.max(1, Number(asset.default_grid_width || 1))))} × ${escapeHtml(String(Math.max(1, Number(asset.default_grid_height || 1))))}</small>
            <small>${escapeHtml(`${Math.round(bytes / 1024)} KB`)}</small>
          </div>
        `;
        button.addEventListener("click", (event) => {
          event.preventDefault();
          event.stopPropagation();
          mapEditorSelectedAssetID = assetID;
          setMapEditorDirty(true);
          syncMapEditorWithState();
          setMapEditorStatus(`Previewing ${name}. Save Map to apply.`);
        });
        elements.assets.appendChild(button);
      });
    }

    function syncMapEditorWithState() {
      const elements = deps.getMapEditorElements?.() || {};
      const state = getCurrentVenueMapState() || {};
      const selectedAssetID = String(mapEditorSelectedAssetID || getCurrentVenueMapAssetID() || state.asset_id || "").trim();
      const selectedAsset = Array.isArray(getCurrentVenueMapAssets())
        ? getCurrentVenueMapAssets().find((asset) => String(asset.id || asset.asset_id || "").trim() === selectedAssetID) || null
        : null;
      const previewURL = String(
        selectedAsset?.content_url
        || selectedAsset?.contentURL
        || selectedAsset?.thumbnail_url
        || selectedAsset?.thumbnailURL
        || state?.asset?.content_url
        || state?.asset?.contentURL
        || "",
      ).trim();
      if (elements.displayMode) elements.displayMode.value = String(state.display_mode || "theater");
      if (elements.fit) elements.fit.value = String(state.fit || "cover");
      if (elements.scale) elements.scale.value = String(Number(state.scale ?? 1));
      if (elements.cropX) elements.cropX.value = String(Number(state.crop_x ?? 0.5));
      if (elements.cropY) elements.cropY.value = String(Number(state.crop_y ?? 0.5));
      if (elements.safeMargin) elements.safeMargin.value = String(Number(state.safe_margin ?? 24));
      syncMapEditorPreview(previewURL);
      syncMapAssetList();
    }

    function livePreviewMapOnStage() {
      renderPixiScene?.();
      deps.renderVenueGridLayer?.(deps.getCurrentVenueMapBounds?.() || deps.getPlayableBounds?.());
    }

    async function refreshVenueMapAssetsController() {
      try {
        const response = await fetchFn("/api/warehouse/assets?asset_type=map&status=active", {
          credentials: "include",
          cache: "no-store",
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok) {
          throw new Error(payload?.error || `HTTP ${response.status}`);
        }
        setCurrentVenueMapAssets(Array.isArray(payload.data) ? payload.data : []);
        syncMapAssetList();
        return payload.data || [];
      } catch (error) {
        console.warn("refreshVenueMapAssets failed", error);
        setCurrentVenueMapAssets([]);
        syncMapAssetList();
        throw error;
      }
    }

    async function refreshVenueMapStateController() {
      try {
        const response = await fetchFn("/api/venues/first-theater/map", {
          credentials: "include",
          cache: "no-store",
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok) {
          throw new Error(payload?.error || `HTTP ${response.status}`);
        }
        const nextState = payload.data || null;
        setCurrentVenueMapState(nextState);
        setCurrentVenueMapAssetID(String(nextState?.asset_id || ""));
        syncMapEditorWithState();
        livePreviewMapOnStage();
        return nextState;
      } catch (error) {
        console.warn("refreshVenueMapState failed", error);
        throw error;
      }
    }

    async function refreshVenueGridConfigController() {
      try {
        const response = await fetchFn("/api/venues/first-theater/grid", {
          credentials: "include",
          cache: "no-store",
        });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok) {
          throw new Error(payload?.error || `HTTP ${response.status}`);
        }
        const nextConfig = payload.data || defaultGridConfig();
        setCurrentVenueGridConfig(nextConfig);
        syncGridEditorWithState();
        syncGridEditorHexFieldVisibility();
        syncGridEditorVisibilityButton();
        deps.renderVenueGridLayer?.(deps.getCurrentVenueMapBounds?.() || deps.getPlayableBounds?.(), nextConfig);
        return nextConfig;
      } catch (error) {
        console.warn("refreshVenueGridConfig failed", error);
        throw error;
      }
    }

    async function uploadMapAssetController(file) {
      if (!file) {
        return null;
      }
      const formData = new FormData();
      formData.append("asset_type", "map");
      formData.append("file", file, file.name || "map.png");
      const response = await fetchFn("/api/workshop/assets", {
        method: "POST",
        credentials: "include",
        body: formData,
      });
      const payload = await response.json().catch(() => null);
      if (!response.ok || !payload?.ok) {
        throw new Error(payload?.error || `HTTP ${response.status}`);
      }
      return payload.data || payload;
    }

    async function saveVenueMapController() {
      const controls = deps.getMapEditorElements?.() || {};
      const selectedFile = controls.file?.files?.[0] || null;
      const draft = mapEditorPayloadFromUI(selectedFile ? "" : undefined);
      let assetID = String(draft.asset_id || getCurrentVenueMapAssetID() || "").trim();

      if (selectedFile) {
        const uploaded = await uploadMapAssetController(selectedFile);
        assetID = String(uploaded?.asset_id || uploaded?.id || assetID || "").trim();
      } else if (!assetID) {
        throw new Error("asset_id_required");
      }

      const payload = {
        ...draft,
        asset_id: assetID,
      };
      const response = await fetchFn("/api/venues/first-theater/map", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const body = await response.json().catch(() => null);
      if (!response.ok || !body?.ok) {
        throw new Error(body?.error || `HTTP ${response.status}`);
      }
      const nextState = body.data || null;
      setCurrentVenueMapState(nextState);
      setCurrentVenueMapAssetID(String(nextState?.asset_id || assetID || ""));
      setMapEditorDirty(false);
      syncMapEditorWithState();
      livePreviewMapOnStage();
      return nextState;
    }

    async function removeVenueMapController() {
      const response = await fetchFn("/api/venues/first-theater/map", {
        method: "DELETE",
        credentials: "include",
      });
      const body = await response.json().catch(() => null);
      if (!response.ok || !body?.ok) {
        throw new Error(body?.error || `HTTP ${response.status}`);
      }
      setCurrentVenueMapState(null);
      setCurrentVenueMapAssetID("");
      setMapEditorDirty(false);
      deps.setMapEditorPreviewURL?.("");
      syncMapEditorWithState();
      livePreviewMapOnStage();
      return null;
    }

    async function saveGridConfigController() {
      const draft = deps.gridEditorDraftFromUI?.() || defaultGridConfig();
      const response = await fetchFn("/api/venues/first-theater/grid", {
        method: "PUT",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(draft),
      });
      const body = await response.json().catch(() => null);
      if (!response.ok || !body?.ok) {
        throw new Error(body?.error || `HTTP ${response.status}`);
      }
      const nextConfig = body.data || draft;
      setCurrentVenueGridConfig(nextConfig);
      setGridEditorOriginalState({ ...nextConfig });
      setGridEditorDirty(false);
      syncGridEditorWithState();
      syncGridEditorHexFieldVisibility();
      syncGridEditorVisibilityButton();
      renderVenueGridLayer?.(deps.getCurrentVenueMapBounds?.() || deps.getPlayableBounds?.(), nextConfig);
      return nextConfig;
    }

    function openMapEditor() {
      const panel = deps.getMapEditorElements?.().panel;
      if (!panel) return;
      if (!canManageIndexCards(getCurrentRole())) {
        setStageStatus("Only producers and directors can add or replace the First Theater map.");
        return;
      }
      hideCardEditor();
      hideGridEditor();
      deps.setMapEditorOriginalState?.(getCurrentVenueMapState() ? { ...getCurrentVenueMapState() } : null);
      mapEditorSelectedAssetID = String(getCurrentVenueMapState()?.asset_id || getCurrentVenueMapAssetID() || "");
      if (panel.hidden) setMapEditorPosition(24, 24);
      panel.hidden = false;
      deps.setMapEditorDirty?.(false);
      setMapEditorStatus("Choose a map asset for the First Theater stage.");
      void refreshVenueMapAssetsController();
      syncMapEditorWithState();
      deps.getMapEditorElements?.().file?.focus?.({ preventScroll: true });
    }

    function hideMapEditorController() {
      deps.setMapEditorDirty?.(false);
      deps.setMapEditorOriginalState?.(null);
      const panel = deps.getMapEditorElements?.().panel;
      if (panel) panel.hidden = true;
      setMapEditorStatus("Choose a map asset for the First Theater stage.");
      deps.setMapEditorPreviewURL?.("");
    }

    function cancelMapEditor() {
      const original = getMapEditorOriginalState();
      if (original) {
        setCurrentVenueMapState(original);
        setCurrentVenueMapAssetID(String(original.asset_id || ""));
        deps.renderPixiScene?.();
      }
      mapEditorSelectedAssetID = String(original?.asset_id || "");
      hideMapEditorController();
    }

    function openGridEditor() {
      const panel = deps.getGridEditorElements?.().panel;
      if (!panel) return;
      if (!canManageIndexCards(getCurrentRole())) {
        setStageStatus("Only producers and directors can configure the First Theater grid.");
        return;
      }
      hideCardEditor();
      hideMapEditorController();
      setGridEditorOriginalState(getCurrentVenueGridConfig() ? { ...getCurrentVenueGridConfig() } : deps.defaultGridConfig?.());
      if (panel.hidden) setGridEditorPosition(24, 24);
      panel.hidden = false;
      setGridEditorDirty(false);
      setGridEditorStatus("Choose a grid style for the First Theater stage.");
      syncGridEditorWithState();
    }

    function cancelGridEditor() {
      const original = getGridEditorOriginalState();
      if (original) {
        setCurrentVenueGridConfig(original);
        applyGridDraftToControls(original);
        deps.renderVenueGridLayer?.(deps.getCurrentVenueMapBounds?.() || deps.getPlayableBounds?.(), original);
      }
      hideGridEditorController();
    }

    function resetGridEditorDraft() {
      if (!window.confirm("Reset grid fields to default values? Click Save Grid to persist.")) return;
      applyGridDraftToControls(deps.defaultGridConfig?.());
      setGridEditorDirty(true);
      syncGridEditorWithState();
      deps.renderVenueGridLayer?.(deps.getCurrentVenueMapBounds?.() || deps.getPlayableBounds?.(), deps.defaultGridConfig?.());
      setGridEditorStatus("Grid reset to defaults. Save to persist.");
    }

    function toggleGridVisibilityDraft() {
      const draft = deps.gridEditorDraftFromUI?.() || deps.defaultGridConfig?.();
      draft.visible = !(draft.visible !== false);
      applyGridDraftToControls(draft);
      setGridEditorDirty(true);
      syncGridEditorVisibilityButton();
      deps.renderVenueGridLayer?.(deps.getCurrentVenueMapBounds?.() || deps.getPlayableBounds?.(), draft);
      setGridEditorStatus(draft.visible ? "Grid will be shown after save." : "Grid will be hidden after save.");
    }

    function syncGridEditorHexFieldVisibility() {
      const controls = deps.getGridEditorElements?.() || {};
      if (!controls.hexOrientationField) return;
      const gridType = String(controls.type?.value || getCurrentVenueGridConfig()?.grid_type || "none").trim().toLowerCase();
      controls.hexOrientationField.hidden = gridType !== "hex";
    }

    function syncGridEditorVisibilityButton() {
      const controls = deps.getGridEditorElements?.() || {};
      if (!controls.visibility) return;
      const visible = gridEditorDraftFromUI()?.visible !== false;
      controls.visibility.textContent = visible ? "Hide Grid" : "Show Grid";
    }

    function syncGridEditorWithState() {
      const controls = deps.getGridEditorElements?.() || {};
      const state = getCurrentVenueGridConfig() || defaultGridConfig();
      if (controls.type) controls.type.value = String(state.grid_type || "none");
      if (controls.hexOrientation) controls.hexOrientation.value = String(state.hex_orientation || "flat-top");
      if (controls.cellSize) controls.cellSize.value = String(Number(state.cell_size ?? 50));
      if (controls.offsetX) controls.offsetX.value = String(Number(state.offset_x ?? 0));
      if (controls.offsetY) controls.offsetY.value = String(Number(state.offset_y ?? 0));
      if (controls.opacity) controls.opacity.value = String(Number(state.opacity ?? 0.45));
      if (controls.lineWidth) controls.lineWidth.value = String(Number(state.line_width ?? 1));
      if (controls.lineStyle) controls.lineStyle.value = String(state.line_style || "neutral");
      if (controls.visibility) controls.visibility.textContent = state.visible !== false ? "Hide Grid" : "Show Grid";
      syncGridEditorHexFieldVisibility();
      syncGridEditorVisibilityButton();
    }
    function defaultGridConfig() {
      return deps.defaultGridConfig?.() || {
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
      const controls = deps.getGridEditorElements?.() || {};
      const state = getCurrentVenueGridConfig() || defaultGridConfig();
      const gridType = String(controls.type?.value || state.grid_type || "none").trim().toLowerCase();
      return {
        grid_type: gridType === "square" || gridType === "hex" ? gridType : "none",
        hex_orientation: String(controls.hexOrientation?.value || state.hex_orientation || "flat-top").trim() === "pointy-top" ? "pointy-top" : "flat-top",
        cell_size: Math.max(8, Math.min(500, Math.round(Number(controls.cellSize?.value || state.cell_size || 50)))),
        offset_x: Math.max(-2000, Math.min(2000, Math.round(Number(controls.offsetX?.value || state.offset_x || 0)))),
        offset_y: Math.max(-2000, Math.min(2000, Math.round(Number(controls.offsetY?.value || state.offset_y || 0)))),
        line_width: Math.max(0.5, Math.min(8, Number(controls.lineWidth?.value || state.line_width || 1))),
        opacity: Math.max(0, Math.min(1, Number(controls.opacity?.value || state.opacity || 0.45))),
        line_style: String(controls.lineStyle?.value || state.line_style || "neutral").trim(),
        visible: controls.visibility ? controls.visibility.textContent !== "Show Grid" : state.visible !== false,
      };
    }
    function applyGridDraftToControls(draft = null) {
      const controls = deps.getGridEditorElements?.() || {};
      const state = draft || getCurrentVenueGridConfig() || defaultGridConfig();
      if (controls.type) controls.type.value = String(state.grid_type || "none");
      if (controls.hexOrientation) controls.hexOrientation.value = String(state.hex_orientation || "flat-top");
      if (controls.cellSize) controls.cellSize.value = String(Number(state.cell_size ?? 50));
      if (controls.offsetX) controls.offsetX.value = String(Number(state.offset_x ?? 0));
      if (controls.offsetY) controls.offsetY.value = String(Number(state.offset_y ?? 0));
      if (controls.lineWidth) controls.lineWidth.value = String(Number(state.line_width ?? 1));
      if (controls.opacity) controls.opacity.value = String(Number(state.opacity ?? 0.45));
      if (controls.lineStyle) controls.lineStyle.value = String(state.line_style || "neutral");
      if (controls.visibility) controls.visibility.textContent = state.visible !== false ? "Hide Grid" : "Show Grid";
      syncGridEditorHexFieldVisibility();
      syncGridEditorVisibilityButton();
    }
    function liveSyncGrid() {
      const draft = deps.gridEditorDraftFromUI?.() || deps.defaultGridConfig?.();
      syncGridEditorHexFieldVisibility();
      syncGridEditorVisibilityButton();
      deps.renderVenueGridLayer?.(deps.getCurrentVenueMapBounds?.() || deps.getPlayableBounds?.(), draft);
    }

    function nudgeGridNumberField(inputEl, delta, min, max) {
      if (!inputEl) return;
      const current = Number(inputEl.value) || 0;
      const next = Math.max(min, Math.min(max, current + delta));
      inputEl.value = String(next);
      deps.setGridEditorDirty?.(true);
      liveSyncGrid();
    }

    function saveGridConfig() {
      return saveGridConfigController();
    }

    function setMapEditorPositionWrapper(left, top) {
      setMapEditorPosition(left, top);
    }

    function setGridEditorPositionWrapper(left, top) {
      setGridEditorPosition(left, top);
    }

    return {
      currentLiveSelection,
      clampCardEditorPosition,
      setCardEditorPosition,
      showCardEditorFor,
      hideCardEditor: hideCardEditorController,
      syncCardEditorWithSelection,
      openCardEditor,
      saveCardEditor,
      toggleCardEditorFromSelection,
      mapEditorDraftFromState,
      setMapEditorStatus,
      clampMapEditorPosition,
      setMapEditorPosition: setMapEditorPositionWrapper,
      syncMapEditorPreview,
      syncMapAssetList,
      syncMapEditorWithState,
      livePreviewMapOnStage,
      openMapEditor,
      hideMapEditor: hideMapEditorController,
      cancelMapEditor,
      mapEditorPayloadFromUI,
      refreshVenueMapAssets: refreshVenueMapAssetsController,
      refreshVenueMapState: refreshVenueMapStateController,
      uploadMapAsset: uploadMapAssetController,
      saveVenueMap: saveVenueMapController,
      removeVenueMap: removeVenueMapController,
      defaultGridConfig,
      gridEditorDraftFromUI,
      setGridEditorStatus,
      syncGridEditorHexFieldVisibility,
      syncGridEditorVisibilityButton,
      syncGridEditorWithState,
      liveSyncGrid,
      nudgeGridNumberField,
      openGridEditor,
      hideGridEditor: hideGridEditorController,
      cancelGridEditor,
      resetGridEditorDraft,
      toggleGridVisibilityDraft,
      saveGridConfig: saveGridConfigController,
      refreshVenueGridConfig: refreshVenueGridConfigController,
      createIndexCardFromMenu: deps.createIndexCardFromMenu,
      placeCreatedIndexCard: deps.placeCreatedIndexCard,
    };
  }

  return { createEditorControllers };
});
