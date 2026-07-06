(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryCatharsisTokenUI = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function createTokenUi(deps) {
    const state = deps.state;
    const canManageStageTokens = deps.canManageStageTokens;
    const tokenPlacementPointForCreate = deps.tokenPlacementPointForCreate;
    const tokenScaleForModel = deps.tokenScaleForModel;
    const tokenSnapModeForModel = deps.tokenSnapModeForModel;
    const tokenLayerForModel = deps.tokenLayerForModel;
    const renderPixiScene = deps.renderPixiScene;
    const setStageStatus = deps.setStageStatus;
    const setMovementLine = deps.setMovementLine;
    const formatByteSize = deps.formatByteSize;
    const escapeHtml = deps.escapeHtml;
    const sendAction = deps.sendAction;
    const refreshVenueMapAssets = deps.refreshVenueMapAssets;
    const currentGridConfig = deps.currentGridConfig || (() => null);
    const lastStagePoint = deps.lastStagePoint || (() => null);
    const stagePlacementCandidate = deps.stagePlacementCandidate || (() => null);
    const stagePlacementScreenCandidate = deps.stagePlacementScreenCandidate || (() => null);
    const defaultPlacementPoint = deps.defaultPlacementPoint || (() => null);
    const setPlacementState = deps.setPlacementState;
    const openEditor = deps.openEditor;
    const closeEditor = deps.closeEditor;
    const setEditorStatus = deps.setEditorStatus;
    const getEditorState = deps.getEditorState;
    const getTokenEditorTargetKey = typeof deps.getTokenEditorTargetKey === "function" ? deps.getTokenEditorTargetKey : () => "";
    const setTokenEditorTargetKey = typeof deps.setTokenEditorTargetKey === "function" ? deps.setTokenEditorTargetKey : () => {};
    const getTokenEditorDirty = typeof deps.getTokenEditorDirty === "function" ? deps.getTokenEditorDirty : () => false;
    const setTokenEditorDirty = typeof deps.setTokenEditorDirty === "function" ? deps.setTokenEditorDirty : () => {};
    const getTokenAssets = deps.getTokenAssets;
    const setTokenAssets = deps.setTokenAssets;
    const getTokenPickerElements = deps.getTokenPickerElements;
    const refreshWarehouseTokenAssetsFn = deps.refreshWarehouseTokenAssets;

    function tokenPickerFilteredAssets() {
      const search = String(state.search || "").trim().toLowerCase();
      const shape = String(state.filterShape || "all").trim().toLowerCase();
      return getTokenAssets().filter((asset) => {
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
        if (shape !== "all" && assetShape !== shape) return false;
        if (search && !haystack.includes(search)) return false;
        return true;
      });
    }

    function tokenPickerPreviewForAsset(asset) {
      const elements = getTokenPickerElements();
      if (!elements.preview) return;
      if (state.previewURL && state.previewURL.startsWith("blob:")) {
        try { URL.revokeObjectURL(state.previewURL); } catch {}
      }
      state.previewURL = String(asset?.thumbnail_url || asset?.content_url || "").trim();
      elements.preview.src = state.previewURL || "";
      if (elements.previewBadge) {
        const dims = `${Number(asset?.default_grid_width || 1)} x ${Number(asset?.default_grid_height || 1)}`;
        elements.previewBadge.textContent = asset ? `${asset.name || asset.id || "Token"} · ${asset.shape || "circle"} · ${dims}` : "No token selected";
      }
    }

    function renderTokenPickerList() {
      const elements = getTokenPickerElements();
      if (!elements.list) return;
      const assets = tokenPickerFilteredAssets();
      elements.list.innerHTML = "";
      if (!assets.length) {
        const empty = document.createElement("div");
        empty.className = "token-picker-status";
        empty.textContent = "No matching active token assets.";
        elements.list.appendChild(empty);
        tokenPickerPreviewForAsset(null);
        return;
      }
      if (!state.selectedAssetID || !assets.some((asset) => asset.id === state.selectedAssetID)) {
        state.selectedAssetID = String(assets[0].id || assets[0].asset_id || "");
      }
      const selected = assets.find((asset) => asset.id === state.selectedAssetID) || assets[0] || null;
      if (selected) tokenPickerPreviewForAsset(selected);
      assets.forEach((asset) => {
        const assetID = String(asset.id || asset.asset_id || "").trim();
        const assetName = String(asset.name || asset.asset_name || asset.original_filename || assetID || "Token asset");
        const assetShape = String(asset.shape || asset.asset_shape || "circle");
        const sourceMime = String(asset.source_mime || asset.sourceMime || asset.sniffed_mime || asset.sniffedMime || "");
        const thumbURL = String(asset.thumbnail_url || asset.thumbnailURL || asset.content_url || asset.contentURL || "");
        const button = document.createElement("button");
        button.type = "button";
        button.className = "token-picker-card";
        if (assetID === state.selectedAssetID) button.classList.add("is-selected");
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
          state.selectedAssetID = assetID;
          state.search = String(elements.search?.value || state.search || "");
          renderTokenPickerList();
          if (state.mode !== "replace") {
            const assetToPlace = selectedTokenAsset();
            if (assetToPlace) {
              placeTokenAsset(assetToPlace, state.placementPoint, "", false);
            }
            return;
          }
          if (elements.status) {
            elements.status.textContent = state.mode === "replace"
              ? `Selected ${assetName}. Click Replace Token to apply.`
              : `Selected ${assetName}. Click Add Token to place it.`;
          }
        });
        elements.list.appendChild(button);
      });
    }

    function selectedTokenAsset() {
      const assets = getTokenAssets();
      const selectedAssetID = String(state.selectedAssetID || "").trim();
      if (!selectedAssetID) return null;
      const asset = assets.find((item) => String(item.id || item.asset_id || "").trim() === selectedAssetID) || null;
      if (!asset) return null;
      const assetName = String(asset.name || asset.asset_name || asset.original_filename || selectedAssetID || "Token asset");
      const assetShape = String(asset.shape || asset.asset_shape || "circle");
      const thumbURL = String(asset.thumbnail_url || asset.thumbnailURL || asset.content_url || asset.contentURL || "");
      return {
        ...asset,
        id: selectedAssetID,
        asset_id: selectedAssetID,
        name: assetName,
        shape: assetShape,
        thumbnail_url: thumbURL,
        content_url: String(asset.content_url || asset.contentURL || ""),
      };
    }

    async function refreshWarehouseTokenAssets() {
      if (!canManageStageTokens()) {
        setTokenAssets([]);
        renderTokenPickerList();
        return;
      }
      try {
        if (getTokenPickerElements().status) getTokenPickerElements().status.textContent = "Loading active token assets...";
        const params = new URLSearchParams({
          asset_type: "token",
          status: "active",
          search: String(getTokenPickerElements().search?.value || "").trim(),
        });
        const response = await fetch(`/api/warehouse/assets?${params.toString()}`, { credentials: "include", cache: "no-store" });
        const payload = await response.json().catch(() => null);
        if (!response.ok || !payload?.ok) throw new Error(payload?.error || `HTTP ${response.status}`);
        setTokenAssets(Array.isArray(payload.data) ? payload.data : []);
        if (getTokenPickerElements().status) {
          const count = getTokenAssets().length;
          getTokenPickerElements().status.textContent = count ? `${count} token asset${count === 1 ? "" : "s"} loaded.` : "No active token assets found.";
        }
        renderTokenPickerList();
      } catch (error) {
        console.warn("refreshWarehouseTokenAssets failed", error);
        setTokenAssets([]);
        if (getTokenPickerElements().status) getTokenPickerElements().status.textContent = error.message || String(error);
        renderTokenPickerList();
      }
    }

    function openTokenPicker(mode = "create", objectModel = null) {
      const elements = getTokenPickerElements();
      if (!elements.panel) return;
      if (!canManageStageTokens()) {
        setStageStatus("Only producers and directors can place Catharsis tokens.");
        return;
      }
      closeEditor("card");
      closeEditor("map");
      closeEditor("grid");
      closeEditor("token");
      state.open = true;
      state.mode = mode === "replace" ? "replace" : "create";
      state.selectedAssetID = "";
      state.filterShape = "all";
      state.search = "";
      state.placementPoint = stagePlacementCandidate()
        ? { ...stagePlacementCandidate() }
        : (lastStagePoint()
          ? { ...lastStagePoint() }
          : (defaultPlacementPoint() ? { ...defaultPlacementPoint() } : null));
      state.placementScreenPoint = stagePlacementScreenCandidate() ? { ...stagePlacementScreenCandidate() } : null;
      state.replaceTargetKey = objectModel?.key || "";
      state.replaceTargetElementID = objectModel?.elementId || "";
      state.replaceTargetElementSlug = objectModel?.elementSlug || "";
      state.replaceTargetScale = objectModel ? tokenScaleForModel(objectModel) : 100;
      state.replaceTargetSnapMode = objectModel ? tokenSnapModeForModel(objectModel) : (currentGridConfig()?.grid_type !== "none" ? "grid" : "free");
      state.replaceTargetTokenLayer = objectModel ? tokenLayerForModel(objectModel) : "public";
      if (elements.panel.hidden) {
        state.setPosition?.(16, 220);
      }
      elements.panel.hidden = false;
      if (elements.search) elements.search.value = "";
      if (elements.shape) elements.shape.value = "all";
      if (elements.status) elements.status.textContent = "Choose an active Warehouse token asset.";
      if (elements.apply) elements.apply.textContent = state.mode === "replace" ? "Replace Token" : "Add Token";
      void refreshWarehouseTokenAssetsFn();
      elements.search?.focus?.({ preventScroll: true });
    }

    function closeTokenPicker() {
      state.open = false;
      state.replaceTargetKey = "";
      state.replaceTargetElementID = "";
      state.replaceTargetElementSlug = "";
      state.replaceTargetScale = 100;
      state.replaceTargetSnapMode = "grid";
      state.replaceTargetTokenLayer = "public";
      state.placementPoint = null;
      state.placementScreenPoint = null;
      const elements = getTokenPickerElements();
      if (elements.panel) elements.panel.hidden = true;
      state.previewURL = "";
      if (elements.status) elements.status.textContent = "Choose a reusable Warehouse token.";
    }

    function applySelectedTokenAsset() {
      const asset = selectedTokenAsset();
      if (!asset) {
        setStageStatus("Select a token asset first.");
        const elements = getTokenPickerElements();
        if (elements.status) {
          elements.status.textContent = state.mode === "replace"
            ? "Select a token asset, then click Replace Token."
            : "Select a token asset, then click Add Token.";
        }
        return;
      }
      if (state.mode === "replace" && state.replaceTargetKey) {
        placeTokenAsset(asset, state.placementPoint, state.replaceTargetKey, true);
        return;
      }
      placeTokenAsset(asset, state.placementPoint, "", false);
    }

    function placeTokenAsset(asset, point, replaceTargetKey = "", replacing = false) {
      if (!asset) return;
      const livePlacementPoint = stagePlacementCandidate() ? { ...stagePlacementCandidate() } : null;
      const fallbackPlacementPoint = defaultPlacementPoint() ? { ...defaultPlacementPoint() } : null;
      const placementPoint = livePlacementPoint || point || state.placementPoint || lastStagePoint() || fallbackPlacementPoint || null;
      if (!placementPoint) {
        setStageStatus("No stage placement point available.");
        return;
      }
      state.placementPoint = placementPoint;
      state.placementScreenPoint = stagePlacementScreenCandidate() ? { ...stagePlacementScreenCandidate() } : state.placementScreenPoint;
      const gridConfig = currentGridConfig();
      const snapMode = gridConfig && gridConfig.grid_type !== "none" ? "grid" : "free";
      const snapped = tokenPlacementPointForCreate(placementPoint, { source: { data: { snap_mode: snapMode } } }, snapMode);
      const targetPayload = {
        asset_id: String(asset.id || "").trim(),
        venue_slug: "catharsis",
        layer: "stage",
        x: snapped.x,
        y: snapped.y,
        order: 0,
        snap_mode: replacing ? (state.replaceTargetSnapMode || snapMode) : snapMode,
        token_layer: replacing ? (state.replaceTargetTokenLayer || "public") : "public",
        scale: replacing ? (state.replaceTargetScale || 100) : 100,
      };
      const actionType = replacing ? "update/token" : "create/token";
      const sent = sendAction(actionType, replacing && replaceTargetKey ? { element_id: state.replaceTargetElementID || replaceTargetKey || "", element_slug: state.replaceTargetElementSlug || "", ...targetPayload } : targetPayload);
      if (sent) {
        const elements = getTokenPickerElements();
        if (elements.status) elements.status.textContent = replacing ? `Replacing token asset with ${asset.name || asset.id}.` : `Placing ${asset.name || asset.id}.`;
        setStageStatus(replacing ? `Replacing token with ${asset.name || asset.id}.` : `Placing token ${asset.name || asset.id}.`);
        setMovementLine(replacing ? `Replace sent for ${asset.name || asset.id}.` : `Create sent for ${asset.name || asset.id}.`);
      } else {
        setStageStatus("Socket unavailable.");
      }
      closeTokenPicker();
    }

    function syncTokenEditorWithSelection() {
      const elements = getEditorState();
      if (!elements.panel || elements.panel.hidden) return;
      const target = elements.currentSelection && deps.isTokenObject(elements.currentSelection)
        ? elements.currentSelection
        : deps.currentObjects().find((item) => item.key === getTokenEditorTargetKey()) || null;
      if (!target || !target.live || !deps.isTokenObject(target)) {
        hideTokenEditor();
        return;
      }
      if (getTokenEditorDirty()) return;
      if (elements.scale) elements.scale.value = String(Math.round(tokenScaleForModel(target)));
      if (elements.scaleValue) elements.scaleValue.value = String(Math.round(tokenScaleForModel(target)));
      if (elements.snap) elements.snap.value = tokenSnapModeForModel(target);
      if (elements.layer) elements.layer.value = tokenLayerForModel(target);
      if (elements.status) elements.status.textContent = `${target.label} ready to edit.`;
    }

    function openTokenEditor(model = null) {
      const elements = getEditorState();
      const target = model || elements.currentSelection;
      if (!target || !target.live || !deps.isTokenObject(target)) {
        setStageStatus("Select a live token to edit it.");
        return;
      }
      if (deps.objectState(target).locked) {
        setStageStatus("This token is locked.");
        return;
      }
      closeEditor("map");
      closeEditor("grid");
      closeEditor("card");
      closeTokenPicker();
      setTokenEditorTargetKey(target.key);
      setTokenEditorDirty(false);
      if (elements.panel.hidden) {
        elements.setPosition?.(16, 220);
      }
      elements.panel.hidden = false;
      syncTokenEditorWithSelection();
      elements.scale?.focus?.({ preventScroll: true });
    }

    function hideTokenEditor() {
      const elements = getEditorState();
      setTokenEditorDirty(false);
      setTokenEditorTargetKey("");
      if (elements.panel) elements.panel.hidden = true;
      if (elements.status) elements.status.textContent = "Adjust the selected token's placement scale.";
    }

    function saveTokenEditor() {
      const elements = getEditorState();
      const target = deps.currentObjects().find((item) => item.key === getTokenEditorTargetKey()) || elements.currentSelection;
      if (!target || !target.live || !deps.isTokenObject(target)) {
        setStageStatus("Select a live token to save edits.");
        return;
      }
      const scale = deps.clampNumber(elements.scaleValue?.value ?? elements.scale?.value ?? 100, 25, 500, 100);
      const snapMode = String(elements.snap?.value || tokenSnapModeForModel(target)).trim().toLowerCase() === "free" ? "free" : "grid";
      const tokenLayer = String(elements.layer?.value || tokenLayerForModel(target)).trim().toLowerCase() === "director" ? "director" : "public";
      const sent = sendAction("update/token", {
        element_id: target.elementId || "",
        element_slug: target.elementSlug || "",
        venue_slug: "catharsis",
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
        if (elements.status) elements.status.textContent = "Save failed: socket unavailable.";
        return;
      }
      deps.updateTokenLocalModel(target, (model) => {
        model.scale = scale;
        model.snapMode = snapMode;
        model.gridRelative = snapMode === "grid";
        model.tokenLayer = tokenLayer;
      });
      setTokenEditorDirty(false);
      setStageStatus(`Saved ${target.label}.`);
      if (elements.status) elements.status.textContent = `${target.label} save sent.`;
      setMovementLine(`update/token sent for ${target.label}.`);
      renderPixiScene();
    }

    return {
      tokenPickerFilteredAssets,
      tokenPickerPreviewForAsset,
      renderTokenPickerList,
      refreshWarehouseTokenAssets,
      openTokenPicker,
      closeTokenPicker,
      applySelectedTokenAsset,
      placeTokenAsset,
      syncTokenEditorWithSelection,
      openTokenEditor,
      hideTokenEditor,
      saveTokenEditor,
    };
  }

  return { createTokenUi };
});
