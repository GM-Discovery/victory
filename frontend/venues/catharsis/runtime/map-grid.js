(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryCatharsisMapGrid = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function clampNumber(value, min, max, fallback) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(min, Math.min(max, parsed));
  }

  function clampPanelPosition(left, top, shellRect, panelRect, fallbackWidth, fallbackHeight) {
    const panelWidth = Number(panelRect?.width || fallbackWidth);
    const panelHeight = Number(panelRect?.height || fallbackHeight);
    const shellWidth = Number(shellRect?.width || 0);
    const shellHeight = Number(shellRect?.height || 0);
    const maxLeft = Math.max(8, shellWidth - panelWidth - 8);
    const maxTop = Math.max(8, shellHeight - panelHeight - 8);
    return {
      left: Math.max(8, Math.min(Math.round(left), maxLeft)),
      top: Math.max(8, Math.min(Math.round(top), maxTop)),
    };
  }

  function cameraControlsState(view = {}, displayMode = "theater", defaults = {}) {
    const zoom = clampNumber(view.zoomRelativeToFit, defaults.minZoom || 0.25, defaults.maxZoom || 4, 1);
    const minZoom = Number(defaults.minZoom || 0.25);
    const maxZoom = Number(defaults.maxZoom || 4);
    return {
      label: `${Math.round(zoom * 100)}%`,
      zoomOutDisabled: zoom <= minZoom,
      zoomInDisabled: zoom >= maxZoom,
      interactionLocked: String(displayMode || "theater") === "fullscreen",
    };
  }

  function getPlayableBounds(stageSize = { width: 0, height: 0 }) {
    const width = Math.max(320, Number(stageSize?.width || 0));
    const height = Math.max(320, Number(stageSize?.height || 0));
    const insetX = Math.max(18, Math.round(width * 0.04));
    const upperDrapeHeight = Math.max(80, Math.round(height * 0.16));
    return {
      x: insetX,
      y: upperDrapeHeight,
      width: Math.max(1, width - (insetX * 2)),
      height: Math.max(1, height - upperDrapeHeight),
    };
  }

  function resetCameraToFit(stageCamera, activeMapId, currentView = null, currentMapState = null, currentMapAssetID = "") {
    if (!stageCamera) return;
    const nextActiveMapId = String(activeMapId || currentMapState?.asset_id || currentMapAssetID || "");
    stageCamera.setActiveMapId?.(nextActiveMapId, { reset: true });
    stageCamera.fit?.(nextActiveMapId);
    const displayMode = String(currentMapState?.display_mode || "theater").trim() === "fullscreen" ? "fullscreen" : "theater";
    const state = cameraControlsState(currentView || stageCamera.getView?.(), displayMode, { minZoom: 0.25, maxZoom: 4 });
    stageCamera.setInteractionLocked?.(state.interactionLocked);
    return state;
  }

  function mapEditorDraftFromState(state = {}, currentAssetID = "", controls = {}) {
    const liveDisplayMode = String(controls.displayMode ?? "").trim();
    const liveFit = String(controls.fit ?? "").trim();
    const rawDisplayMode = liveDisplayMode || String(state.display_mode || "theater");
    return {
      assetID: String(state.asset_id || currentAssetID || ""),
      displayMode: rawDisplayMode === "fullscreen" ? "fullscreen" : "theater",
      fit: liveFit || String(state.fit || "cover"),
      cropX: clampNumber(controls.cropX ?? state.crop_x ?? 0.5, 0, 1, 0.5),
      cropY: clampNumber(controls.cropY ?? state.crop_y ?? 0.5, 0, 1, 0.5),
      scale: clampNumber(controls.scale ?? state.scale ?? 1, 0.25, 4, 1),
      safeMargin: clampNumber(controls.safeMargin ?? state.safe_margin ?? 24, 0, 128, 24),
    };
  }

  function mapEditorPayloadFromUI(state = {}, currentAssetID = "", controls = {}) {
    const draft = mapEditorDraftFromState(state, currentAssetID, controls);
    return {
      asset_id: String(controls.assetID || currentAssetID || state.asset_id || "").trim(),
      display_mode: draft.displayMode,
      fit: String(controls.fit || state.fit || "cover"),
      crop_x: draft.cropX,
      crop_y: draft.cropY,
      scale: draft.scale,
      safe_margin: Math.round(draft.safeMargin),
    };
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

  function gridEditorDraftFromUI(currentGridConfig = null, controls = {}) {
    const state = currentGridConfig || defaultGridConfig();
    const liveGridType = String(controls.gridType || "").trim();
    const liveHexOrientation = String(controls.hexOrientation || "").trim();
    const liveLineStyle = String(controls.lineStyle || "").trim();
    return {
      grid_type: liveGridType === "square" || liveGridType === "hex" ? liveGridType : "none",
      hex_orientation: liveHexOrientation === "pointy-top" ? "pointy-top" : "flat-top",
      cell_size: clampNumber(controls.cellSize, 8, 500, 50),
      offset_x: clampNumber(controls.offsetX, -2000, 2000, 0),
      offset_y: clampNumber(controls.offsetY, -2000, 2000, 0),
      line_width: clampNumber(controls.lineWidth, 0.5, 8, 1),
      opacity: clampNumber(controls.opacity, 0, 1, 0.45),
      line_style: liveLineStyle === "light" || liveLineStyle === "dark" ? liveLineStyle : "neutral",
      visible: state.visible !== false,
    };
  }

  function gridEditorHexFieldVisible(gridType = "") {
    return String(gridType || "").trim().toLowerCase() === "hex";
  }

  function gridEditorVisibilityLabel(currentGridConfig = null) {
    const visible = currentGridConfig ? currentGridConfig.visible !== false : true;
    return visible ? "Hide Grid" : "Show Grid";
  }

  return {
    clampPanelPosition,
    cameraControlsState,
    getPlayableBounds,
    resetCameraToFit,
    mapEditorDraftFromState,
    mapEditorPayloadFromUI,
    defaultGridConfig,
    gridEditorDraftFromUI,
    gridEditorHexFieldVisible,
    gridEditorVisibilityLabel,
  };
});
