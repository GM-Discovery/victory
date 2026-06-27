(function (root, factory) {
  if (typeof module === "object" && module.exports) {
    module.exports = factory();
  }
  root.VictoryCatharsisGeometry = factory();
})(typeof globalThis !== "undefined" ? globalThis : window, function () {
  function clampNumber(value, min, max, fallback) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(min, Math.min(max, parsed));
  }

  function normalizeBounds(bounds, fallbackWidth = 1, fallbackHeight = 1) {
    const source = bounds && typeof bounds === "object" ? bounds : {};
    return {
      x: Number.isFinite(Number(source.x)) ? Number(source.x) : 0,
      y: Number.isFinite(Number(source.y)) ? Number(source.y) : 0,
      width: Math.max(1, Number(source.width || fallbackWidth || 1)),
      height: Math.max(1, Number(source.height || fallbackHeight || 1)),
    };
  }

  function computeStagePlayableBounds(width, height) {
    const safeWidth = Math.max(1, Number(width || 0));
    const safeHeight = Math.max(1, Number(height || 0));
    const stageWidth = Math.max(320, safeWidth);
    const stageHeight = Math.max(320, safeHeight);
    const insetX = Math.max(18, Math.round(stageWidth * 0.04));
    const upperDrapeHeight = Math.max(80, Math.round(stageHeight * 0.16));
    return {
      x: insetX,
      y: upperDrapeHeight,
      width: Math.max(1, stageWidth - (insetX * 2)),
      height: Math.max(1, stageHeight - upperDrapeHeight),
    };
  }

  function cameraStateFromView(view = {}, playableBounds, worldBounds) {
    const pb = normalizeBounds(playableBounds, 1, 1);
    const wb = normalizeBounds(worldBounds, 1, 1);
    const zoom = clampNumber(view.zoomRelativeToFit, 0.01, 10, 1);
    const scaledWidth = wb.width * zoom;
    const scaledHeight = wb.height * zoom;
    const viewLeft = pb.x;
    const viewTop = pb.y;
    const viewRight = pb.x + pb.width;
    const viewBottom = pb.y + pb.height;

    const leftAtPanZero = wb.x * zoom;
    const topAtPanZero = wb.y * zoom;
    const minPanX = viewRight - ((wb.x + wb.width) * zoom);
    const maxPanX = viewLeft - leftAtPanZero;
    const minPanY = viewBottom - ((wb.y + wb.height) * zoom);
    const maxPanY = viewTop - topAtPanZero;

    let panX = Number(view.panX || 0);
    let panY = Number(view.panY || 0);

    if (scaledWidth <= pb.width) {
      const centeredLeft = viewLeft + ((pb.width - scaledWidth) / 2);
      panX = centeredLeft - leftAtPanZero;
    } else {
      panX = clampNumber(panX, minPanX, maxPanX, panX);
    }

    if (scaledHeight <= pb.height) {
      const centeredTop = viewTop + ((pb.height - scaledHeight) / 2);
      panY = centeredTop - topAtPanZero;
    } else {
      panY = clampNumber(panY, minPanY, maxPanY, panY);
    }

    return {
      panX,
      panY,
      zoomRelativeToFit: zoom,
    };
  }

  function screenToWorld(screenPoint, view = {}) {
    const zoom = clampNumber(view.zoomRelativeToFit, 0.01, 10, 1);
    return {
      x: (Number(screenPoint?.x || 0) - Number(view.panX || 0)) / zoom,
      y: (Number(screenPoint?.y || 0) - Number(view.panY || 0)) / zoom,
    };
  }

  function worldToScreen(worldPoint, view = {}) {
    const zoom = clampNumber(view.zoomRelativeToFit, 0.01, 10, 1);
    return {
      x: (Number(worldPoint?.x || 0) * zoom) + Number(view.panX || 0),
      y: (Number(worldPoint?.y || 0) * zoom) + Number(view.panY || 0),
    };
  }

  function stageScreenPointFromClient(clientX, clientY, rect) {
    const stageRect = rect && typeof rect === "object" ? rect : null;
    if (!stageRect) {
      return { x: Number(clientX || 0), y: Number(clientY || 0) };
    }
    return {
      x: Number(clientX || 0) - Number(stageRect.left || 0),
      y: Number(clientY || 0) - Number(stageRect.top || 0),
    };
  }

  function stagePointFromClient(clientX, clientY, rect, camera) {
    const point = stageScreenPointFromClient(clientX, clientY, rect);
    if (camera && typeof camera.screenToWorld === "function") {
      const worldPoint = camera.screenToWorld(point.x, point.y);
      return {
        x: Number(worldPoint?.x || 0),
        y: Number(worldPoint?.y || 0),
      };
    }
    return point;
  }

  function stagePointForWorldPoint(point, camera) {
    if (camera && typeof camera.worldToScreen === "function") {
      const screenPoint = camera.worldToScreen(point?.x, point?.y);
      return {
        x: Number(screenPoint?.x || 0),
        y: Number(screenPoint?.y || 0),
      };
    }
    return {
      x: Number(point?.x || 0),
      y: Number(point?.y || 0),
    };
  }

  function worldPointForStagePoint(point, camera) {
    if (camera && typeof camera.screenToWorld === "function") {
      const worldPoint = camera.screenToWorld(point?.x, point?.y);
      return {
        x: Number(worldPoint?.x || 0),
        y: Number(worldPoint?.y || 0),
      };
    }
    return {
      x: Number(point?.x || 0),
      y: Number(point?.y || 0),
    };
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

  function normalizeOverlayPoint(point, bounds) {
    const stageBounds = normalizeBounds(bounds, 1, 1);
    const x = Number(point?.x || 0);
    const y = Number(point?.y || 0);
    return {
      screen_x: clampNumber((x - stageBounds.x) / stageBounds.width, 0, 1, 0.5),
      screen_y: clampNumber((y - stageBounds.y) / stageBounds.height, 0, 1, 0.5),
    };
  }

  function clampOverlayPoint(point, bounds) {
    const stageBounds = normalizeBounds(bounds, 1, 1);
    return {
      x: Math.max(stageBounds.x, Math.min(stageBounds.x + stageBounds.width, Math.round(Number(point?.x || 0)))),
      y: Math.max(stageBounds.y, Math.min(stageBounds.y + stageBounds.height, Math.round(Number(point?.y || 0)))),
    };
  }

  function offsetPoint(point, dx = 24, dy = 16) {
    return {
      x: Math.round(Number(point?.x || 0) + Number(dx || 0)),
      y: Math.round(Number(point?.y || 0) + Number(dy || 0)),
    };
  }

  function cardPinData(model) {
    const data = model?.source?.data || model?.data || {};
    const position = model?.position || {};
    const readNumber = (value) => {
      if (value === null || value === undefined || value === "") {
        return Number.NaN;
      }
      const parsed = Number(value);
      return Number.isFinite(parsed) ? parsed : Number.NaN;
    };
    const pinMode = String(data.pin_mode || position.frame || "").trim().toLowerCase();
    return {
      mode: pinMode === "world" ? "world" : "overlay",
      worldX: readNumber(data.world_x),
      worldY: readNumber(data.world_y),
      screenX: readNumber(data.screen_x),
      screenY: readNumber(data.screen_y),
    };
  }

  function cardDisplayMode(model) {
    return cardPinData(model).mode;
  }

  function currentDisplayedPointForModel(model, deps = {}) {
    const pin = cardPinData(model);
    const camera = deps.camera || null;
    const getPlayableBounds = typeof deps.getPlayableBounds === "function" ? deps.getPlayableBounds : () => ({ x: 0, y: 0, width: 1, height: 1 });
    const toStagePoint = typeof deps.toStagePoint === "function" ? deps.toStagePoint : () => ({ x: 0, y: 0 });
    const stagePointForWorld = typeof deps.stagePointForWorldPoint === "function" ? deps.stagePointForWorldPoint : (point) => stagePointForWorldPoint(point, camera);
    const worldPointForStage = typeof deps.worldPointForStagePoint === "function" ? deps.worldPointForStagePoint : (point) => worldPointForStagePoint(point, camera);
    const size = deps.size || { width: 0, height: 0 };

    if (String(model?.kind || "").toLowerCase() === "token") {
      const tokenPosition = model?.position || {};
      const tokenX = Number(tokenPosition.x ?? model?.source?.data?.world_x ?? model?.source?.data?.x ?? 0);
      const tokenY = Number(tokenPosition.y ?? model?.source?.data?.world_y ?? model?.source?.data?.y ?? 0);
      return { x: tokenX, y: tokenY };
    }
    if (pin.mode === "world") {
      if (Number.isFinite(pin.worldX) && Number.isFinite(pin.worldY)) {
        return stagePointForWorld({ x: pin.worldX, y: pin.worldY });
      }
      const fallback = toStagePoint(model, size, { baseY: 0.5, orderOffset: 12 });
      return stagePointForWorld(fallback);
    }
    if (Number.isFinite(pin.screenX) && Number.isFinite(pin.screenY)) {
      const bounds = normalizeBounds(typeof getPlayableBounds === "function" ? getPlayableBounds() : null, 1, 1);
      return {
        x: bounds.x + (bounds.width * pin.screenX),
        y: bounds.y + (bounds.height * pin.screenY),
      };
    }
    return toStagePoint(model, size, { baseY: model?.kind === "fire" ? 0.6 : 0.5, orderOffset: 12 });
  }

  function tokenSnapModeForModel(model, gridConfig) {
    const data = model?.source?.data || {};
    const snapMode = String(data.snap_mode || "").trim().toLowerCase();
    if (snapMode === "grid" || snapMode === "free") {
      return snapMode;
    }
    return gridConfig && gridConfig.grid_type !== "none" ? "grid" : "free";
  }

  function tokenLayerForModel(model) {
    const data = model?.source?.data || {};
    const tokenLayer = String(data.token_layer || "").trim().toLowerCase();
    return tokenLayer === "director" ? "director" : "public";
  }

  function tokenScaleForModel(model) {
    const data = model?.source?.data || {};
    const scale = Number(data.scale ?? 100);
    if (!Number.isFinite(scale) || scale <= 0) return 100;
    return Math.max(25, Math.min(500, scale));
  }

  function tokenFootprintForModel(model) {
    const data = model?.source?.data || {};
    const width = Number(data.default_grid_width ?? model?.defaultGridWidth ?? 1);
    const height = Number(data.default_grid_height ?? model?.defaultGridHeight ?? 1);
    return {
      width: Math.max(1, Math.round(Number.isFinite(width) ? width : 1)),
      height: Math.max(1, Math.round(Number.isFinite(height) ? height : 1)),
    };
  }

  function tokenPlacementBaseSize(model, gridConfig) {
    const snapMode = tokenSnapModeForModel(model, gridConfig);
    const hasGrid = Boolean(gridConfig && gridConfig.grid_type && gridConfig.grid_type !== "none");
    return hasGrid && snapMode === "grid" ? Number(gridConfig.cell_size || 50) : 64;
  }

  function tokenDisplaySizeForModel(model, gridConfig) {
    const footprint = tokenFootprintForModel(model);
    const base = tokenPlacementBaseSize(model, gridConfig);
    const scale = tokenScaleForModel(model) / 100;
    return {
      width: Math.max(8, Math.round(base * footprint.width * scale)),
      height: Math.max(8, Math.round(base * footprint.height * scale)),
    };
  }

  function hexCenterCandidates(point, config, bounds) {
    const normalized = {
      gridType: String(config?.gridType || config?.grid_type || "none").trim(),
      hexOrientation: String(config?.hexOrientation || config?.hex_orientation || "flat-top").trim(),
      cellSize: clampNumber(config?.cellSize ?? config?.cell_size, 8, 500, 50),
      offsetX: Number(config?.offsetX ?? config?.offset_x ?? 0),
      offsetY: Number(config?.offsetY ?? config?.offset_y ?? 0),
    };
    const stageBounds = normalizeBounds(bounds, 1, 1);
    const size = Math.max(4, normalized.cellSize / 2);
    const orientation = normalized.hexOrientation === "pointy-top" ? "pointy-top" : "flat-top";
    const origin = {
      x: stageBounds.x + normalized.offsetX,
      y: stageBounds.y + normalized.offsetY,
    };
    const hexWidth = orientation === "pointy-top" ? Math.sqrt(3) * size : 2 * size;
    const hexHeight = orientation === "pointy-top" ? 2 * size : Math.sqrt(3) * size;
    const horizSpacing = orientation === "pointy-top" ? hexWidth : hexWidth * 0.75;
    const vertSpacing = orientation === "pointy-top" ? hexHeight * 0.75 : hexHeight;
    const approxCol = Math.round((Number(point?.x || 0) - origin.x) / horizSpacing);
    const approxRow = Math.round((Number(point?.y || 0) - origin.y) / vertSpacing);
    const candidates = [];

    for (let col = approxCol - 2; col <= approxCol + 2; col += 1) {
      for (let row = approxRow - 2; row <= approxRow + 2; row += 1) {
        let cx;
        let cy;
        if (orientation === "pointy-top") {
          cx = origin.x + (col * horizSpacing) + ((row % 2 !== 0) ? horizSpacing / 2 : 0);
          cy = origin.y + (row * vertSpacing);
        } else {
          cx = origin.x + (col * horizSpacing);
          cy = origin.y + (row * vertSpacing) + ((col % 2 !== 0) ? vertSpacing / 2 : 0);
        }
        if (cx < stageBounds.x - hexWidth || cx > stageBounds.x + stageBounds.width + hexWidth) continue;
        if (cy < stageBounds.y - hexHeight || cy > stageBounds.y + stageBounds.height + hexHeight) continue;
        candidates.push({ x: cx, y: cy, dx: Math.abs(cx - Number(point?.x || 0)), dy: Math.abs(cy - Number(point?.y || 0)) });
      }
    }
    return candidates;
  }

  function snapSquarePoint(point, config, bounds) {
    const stageBounds = normalizeBounds(bounds, 1, 1);
    const cellSize = Math.max(8, Number(config?.cellSize ?? config?.cell_size ?? 50));
    const offsetX = stageBounds.x + (((Number(config?.offsetX ?? config?.offset_x ?? 0) % cellSize) + cellSize) % cellSize);
    const offsetY = stageBounds.y + (((Number(config?.offsetY ?? config?.offset_y ?? 0) % cellSize) + cellSize) % cellSize);
    return {
      x: Math.round(offsetX + (Math.round((Number(point?.x || 0) - offsetX) / cellSize) * cellSize) + (cellSize / 2)),
      y: Math.round(offsetY + (Math.round((Number(point?.y || 0) - offsetY) / cellSize) * cellSize) + (cellSize / 2)),
    };
  }

  function snapHexPoint(point, config, bounds) {
    const candidates = hexCenterCandidates(point, config, bounds);
    if (!candidates.length) {
      return { x: Math.round(Number(point?.x || 0)), y: Math.round(Number(point?.y || 0)) };
    }
    let best = candidates[0];
    let bestDistance = Infinity;
    candidates.forEach((candidate) => {
      const distance = (candidate.dx * candidate.dx) + (candidate.dy * candidate.dy);
      if (distance < bestDistance) {
        best = candidate;
        bestDistance = distance;
      }
    });
    return { x: Math.round(best.x), y: Math.round(best.y) };
  }

  function snapPoint(rawConfig, bounds, point) {
    const config = rawConfig || {};
    const sourcePoint = {
      x: Number(point?.x || 0),
      y: Number(point?.y || 0),
    };
    if (!bounds || !bounds.width || !bounds.height) {
      return { x: Math.round(sourcePoint.x), y: Math.round(sourcePoint.y) };
    }
    const gridType = String(config.gridType || config.grid_type || "none").trim();
    const visible = config.visible !== false;
    if (!visible || gridType === "none") {
      return { x: Math.round(sourcePoint.x), y: Math.round(sourcePoint.y) };
    }
    if (gridType === "square") {
      return snapSquarePoint(sourcePoint, config, bounds);
    }
    if (gridType === "hex") {
      return snapHexPoint(sourcePoint, config, bounds);
    }
    return { x: Math.round(sourcePoint.x), y: Math.round(sourcePoint.y) };
  }

  function tokenPlacementPointForCreate(point, model, deps = {}) {
    const basePoint = point ? { x: Number(point.x || 0), y: Number(point.y || 0) } : null;
    if (!basePoint) {
      return null;
    }
    const snapMode = deps.forcedSnapMode || tokenSnapModeForModel(model, deps.gridConfig || null);
    const bounds = deps.bounds || deps.currentVenueMapBounds || deps.getPlayableBounds?.() || { x: 0, y: 0, width: 1, height: 1 };
    const snapPointFn = typeof deps.snapPoint === "function" ? deps.snapPoint : snapPoint;
    if (snapMode !== "grid") {
      return {
        x: Number(basePoint.x),
        y: Number(basePoint.y),
      };
    }
    return snapPointFn(deps.gridConfig || {}, bounds, basePoint);
  }

  function cardMoveTargetForModel(model, targetPoint, deps = {}) {
    const pinMode = cardDisplayMode(model);
    const basePoint = targetPoint || deps.stagePlacementCandidate || deps.lastStagePoint || null;
    if (!basePoint) {
      return null;
    }
    if (pinMode === "world") {
      return {
        pinMode,
        world_x: Math.round(Number(basePoint.x || 0)),
        world_y: Math.round(Number(basePoint.y || 0)),
        screen_x: null,
        screen_y: null,
      };
    }
    const overlayPoint = normalizeOverlayPoint(basePoint, deps.bounds || deps.getPlayableBounds?.());
    return {
      pinMode,
      world_x: null,
      world_y: null,
      screen_x: overlayPoint.screen_x,
      screen_y: overlayPoint.screen_y,
    };
  }

  function tokenMoveTargetForModel(model, targetPoint, deps = {}) {
    const basePoint = targetPoint || deps.stagePlacementCandidate || deps.lastStagePoint || null;
    if (!basePoint) {
      return null;
    }
    const snapMode = tokenSnapModeForModel(model, deps.gridConfig || null);
    const snapped = snapMode === "grid"
      ? tokenPlacementPointForCreate(basePoint, model, {
          forcedSnapMode: snapMode,
          gridConfig: deps.gridConfig || null,
          currentVenueMapBounds: deps.currentVenueMapBounds || null,
          getPlayableBounds: deps.getPlayableBounds,
          snapPoint: deps.snapPoint,
        })
      : { x: Number(basePoint.x || 0), y: Number(basePoint.y || 0) };
    return {
      x: snapped.x,
      y: snapped.y,
      snapMode,
    };
  }

  function tokenDuplicatePlacementForModel(model, deps = {}) {
    const basePoint = currentDisplayedPointForModel(model, {
      camera: deps.camera || null,
      getPlayableBounds: deps.getPlayableBounds,
      toStagePoint: deps.toStagePoint,
      stagePointForWorldPoint: deps.stagePointForWorldPoint,
      worldPointForStagePoint: deps.worldPointForStagePoint,
      size: deps.size,
    });
    const duplicatePoint = offsetPoint(basePoint, 24, 16);
    const snapMode = tokenSnapModeForModel(model, deps.gridConfig || null);
    const snapped = snapMode === "grid"
      ? tokenPlacementPointForCreate(duplicatePoint, model, {
          forcedSnapMode: snapMode,
          gridConfig: deps.gridConfig || null,
          currentVenueMapBounds: deps.currentVenueMapBounds || null,
          getPlayableBounds: deps.getPlayableBounds,
          snapPoint: deps.snapPoint,
        })
      : duplicatePoint;
    return {
      x: snapped.x,
      y: snapped.y,
      snapMode,
    };
  }

  function cardDuplicatePlacementForModel(model, deps = {}) {
    const pinMode = cardDisplayMode(model);
    const pin = cardPinData(model);
    const offset = { x: 24, y: 16 };
    if (pinMode === "world") {
      const baseWorldPoint = Number.isFinite(pin.worldX) && Number.isFinite(pin.worldY)
        ? { x: pin.worldX, y: pin.worldY }
        : worldPointForStagePoint(currentDisplayedPointForModel(model, deps), deps.camera || null);
      const worldPoint = offsetPoint(baseWorldPoint, offset.x, offset.y);
      return {
        pinMode,
        x: worldPoint.x,
        y: worldPoint.y,
        world_x: worldPoint.x,
        world_y: worldPoint.y,
        screen_x: null,
        screen_y: null,
      };
    }
    const baseOverlayPoint = currentDisplayedPointForModel(model, deps);
    const overlayStagePoint = offsetPoint(baseOverlayPoint, offset.x, offset.y);
    const overlayPoint = normalizeOverlayPoint(overlayStagePoint, deps.bounds || deps.getPlayableBounds?.());
    return {
      pinMode,
      x: overlayStagePoint.x,
      y: overlayStagePoint.y,
      world_x: null,
      world_y: null,
      screen_x: overlayPoint.screen_x,
      screen_y: overlayPoint.screen_y,
    };
  }

  return {
    clampNumber,
    normalizeBounds,
    computeStagePlayableBounds,
    cameraStateFromView,
    screenToWorld,
    worldToScreen,
    stageScreenPointFromClient,
    stagePointFromClient,
    stagePointForWorldPoint,
    worldPointForStagePoint,
    toStagePoint,
    normalizeOverlayPoint,
    clampOverlayPoint,
    offsetPoint,
    cardPinData,
    cardDisplayMode,
    currentDisplayedPointForModel,
    tokenSnapModeForModel,
    tokenLayerForModel,
    tokenScaleForModel,
    tokenFootprintForModel,
    tokenPlacementBaseSize,
    tokenDisplaySizeForModel,
    snapSquarePoint,
    snapHexPoint,
    snapPoint,
    tokenPlacementPointForCreate,
    cardMoveTargetForModel,
    tokenMoveTargetForModel,
    tokenDuplicatePlacementForModel,
    cardDuplicatePlacementForModel,
  };
});
