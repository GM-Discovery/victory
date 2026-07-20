(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryStageStageControls = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function clampNumber(value, min, max, fallback) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(min, Math.min(max, parsed));
  }

  function cameraViewLabel(view = null, defaults = {}) {
    const current = view || { zoomRelativeToFit: 1 };
    const minZoom = Number(defaults.minZoom ?? 0.7);
    const maxZoom = Number(defaults.maxZoom ?? 4);
    const zoom = clampNumber(Number(current.zoomRelativeToFit || 1) * 100, minZoom * 100, maxZoom * 100, 100);
    return `${Math.round(zoom)}%`;
  }

  function cameraControlsState(view = null, mapDisplayMode = "theater", defaults = {}) {
    const current = view || { zoomRelativeToFit: 1 };
    const minZoom = Number(defaults.minZoom ?? 0.7);
    const maxZoom = Number(defaults.maxZoom ?? 4);
    const locked = String(mapDisplayMode || "theater").trim().toLowerCase() !== "fullscreen";
    const zoom = Number(current.zoomRelativeToFit || 1);
    return {
      label: cameraViewLabel(current, defaults),
      zoomOutDisabled: locked || zoom <= (minZoom + 0.0001),
      zoomInDisabled: locked || zoom >= (maxZoom - 0.0001),
      interactionLocked: locked,
    };
  }

  function mapEditorDraftFromState(state = {}, currentAssetID = "", controls = {}) {
    const liveDisplayMode = String(controls.displayMode ?? "").trim();
    const liveFit = String(controls.fit ?? "").trim();
    const liveScale = controls.scale;
    const liveCropX = controls.cropX;
    const liveCropY = controls.cropY;
    const liveSafeMargin = controls.safeMargin;
    const rawDisplayMode = liveDisplayMode || String(state.display_mode || "theater");
    return {
      assetID: String(state.asset_id || currentAssetID || ""),
      displayMode: rawDisplayMode === "fullscreen" ? "fullscreen" : "theater",
      fit: liveFit || String(state.fit || "cover"),
      cropX: clampNumber(liveCropX ?? state.crop_x ?? 0.5, 0, 1, 0.5),
      cropY: clampNumber(liveCropY ?? state.crop_y ?? 0.5, 0, 1, 0.5),
      scale: clampNumber(liveScale ?? state.scale ?? 1, 0.25, 4, 1),
      safeMargin: clampNumber(liveSafeMargin ?? state.safe_margin ?? 24, 0, 128, 24),
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
    cameraViewLabel,
    cameraControlsState,
    mapEditorDraftFromState,
    defaultGridConfig,
    gridEditorDraftFromUI,
    gridEditorHexFieldVisible,
    gridEditorVisibilityLabel,
  };
});
