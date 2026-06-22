(function () {
  const MAX_GRID_LINES = 600;

  function clamp(value, min, max, fallback) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;
    return Math.max(min, Math.min(max, parsed));
  }

  function resolveLineColor(lineStyle) {
    switch (String(lineStyle || "neutral").trim()) {
      case "light":
        return 0xffffff;
      case "dark":
        return 0x12090a;
      default:
        return 0xf0d49b;
    }
  }

  function normalizeConfig(config) {
    const source = config || {};
    const gridType = String(source.gridType || source.grid_type || "none").trim();
    const hexOrientation = String(source.hexOrientation || source.hex_orientation || "flat-top").trim();
    return {
      gridType: gridType === "square" || gridType === "hex" ? gridType : "none",
      hexOrientation: hexOrientation === "pointy-top" ? "pointy-top" : "flat-top",
      cellSize: clamp(source.cellSize ?? source.cell_size, 8, 500, 50),
      offsetX: clamp(source.offsetX ?? source.offset_x, -2000, 2000, 0),
      offsetY: clamp(source.offsetY ?? source.offset_y, -2000, 2000, 0),
      lineWidth: clamp(source.lineWidth ?? source.line_width, 0.5, 8, 1),
      opacity: clamp(source.opacity, 0, 1, 0.45),
      lineStyle: String(source.lineStyle || source.line_style || "neutral").trim() || "neutral",
      visible: source.visible !== false,
    };
  }

  function resolveOrigin(bounds, config) {
    return {
      x: Number(bounds?.x || 0) + Number(config.offsetX || 0),
      y: Number(bounds?.y || 0) + Number(config.offsetY || 0),
    };
  }

  function snapSquarePoint(point, config, bounds) {
    const cellSize = Math.max(8, config.cellSize);
    const offsetX = Number(bounds?.x || 0) + (((config.offsetX % cellSize) + cellSize) % cellSize);
    const offsetY = Number(bounds?.y || 0) + (((config.offsetY % cellSize) + cellSize) % cellSize);
    return {
      x: Math.round(offsetX + (Math.round((point.x - offsetX) / cellSize) * cellSize) + (cellSize / 2)),
      y: Math.round(offsetY + (Math.round((point.y - offsetY) / cellSize) * cellSize) + (cellSize / 2)),
    };
  }

  function hexCenterCandidates(point, config, bounds) {
    const size = Math.max(4, config.cellSize / 2);
    const orientation = config.hexOrientation;
    const { x, y } = bounds;
    const origin = resolveOrigin(bounds, config);

    const hexWidth = orientation === "pointy-top" ? Math.sqrt(3) * size : 2 * size;
    const hexHeight = orientation === "pointy-top" ? 2 * size : Math.sqrt(3) * size;
    const horizSpacing = orientation === "pointy-top" ? hexWidth : hexWidth * 0.75;
    const vertSpacing = orientation === "pointy-top" ? hexHeight * 0.75 : hexHeight;

    const approxCol = Math.round((point.x - origin.x) / horizSpacing);
    const approxRow = Math.round((point.y - origin.y) / vertSpacing);
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
        if (cx < x - hexWidth || cx > x + bounds.width + hexWidth) continue;
        if (cy < y - hexHeight || cy > y + bounds.height + hexHeight) continue;
        candidates.push({ x: cx, y: cy, dx: Math.abs(cx - point.x), dy: Math.abs(cy - point.y) });
      }
    }
    return candidates;
  }

  function snapHexPoint(point, config, bounds) {
    const candidates = hexCenterCandidates(point, config, bounds);
    if (!candidates.length) {
      return { x: Math.round(point.x), y: Math.round(point.y) };
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
    const config = normalizeConfig(rawConfig);
    const sourcePoint = {
      x: Number(point?.x || 0),
      y: Number(point?.y || 0),
    };
    if (!bounds || !bounds.width || !bounds.height) {
      return { x: Math.round(sourcePoint.x), y: Math.round(sourcePoint.y) };
    }
    if (!config.visible || config.gridType === "none") {
      return { x: Math.round(sourcePoint.x), y: Math.round(sourcePoint.y) };
    }
    if (config.gridType === "square") {
      return snapSquarePoint(sourcePoint, config, bounds);
    }
    if (config.gridType === "hex") {
      return snapHexPoint(sourcePoint, config, bounds);
    }
    return { x: Math.round(sourcePoint.x), y: Math.round(sourcePoint.y) };
  }

  function clear(layer) {
    if (!layer) return;
    layer.removeChildren();
  }

  function drawSquareGrid(graphics, config, bounds) {
    const cellSize = Math.max(8, config.cellSize);
    const { x, y, width, height } = bounds;
    const startX = x + (((config.offsetX % cellSize) + cellSize) % cellSize);
    const startY = y + (((config.offsetY % cellSize) + cellSize) % cellSize);

    const columns = Math.min(MAX_GRID_LINES, Math.ceil(width / cellSize) + 1);
    const rows = Math.min(MAX_GRID_LINES, Math.ceil(height / cellSize) + 1);

    for (let i = 0; i <= columns; i += 1) {
      const lineX = startX + i * cellSize;
      if (lineX < x || lineX > x + width) continue;
      graphics.moveTo(lineX, y);
      graphics.lineTo(lineX, y + height);
    }
    for (let i = 0; i <= rows; i += 1) {
      const lineY = startY + i * cellSize;
      if (lineY < y || lineY > y + height) continue;
      graphics.moveTo(x, lineY);
      graphics.lineTo(x + width, lineY);
    }
  }

  function hexCorner(cx, cy, size, index, orientation) {
    const angleDeg = orientation === "pointy-top" ? (60 * index) + 30 : 60 * index;
    const angleRad = (Math.PI / 180) * angleDeg;
    return {
      x: cx + size * Math.cos(angleRad),
      y: cy + size * Math.sin(angleRad),
    };
  }

  function drawHexAt(graphics, cx, cy, size, orientation) {
    for (let i = 0; i < 6; i += 1) {
      const corner = hexCorner(cx, cy, size, i, orientation);
      if (i === 0) {
        graphics.moveTo(corner.x, corner.y);
      } else {
        graphics.lineTo(corner.x, corner.y);
      }
    }
    graphics.closePath();
  }

  function drawHexGrid(graphics, config, bounds) {
    const size = Math.max(4, config.cellSize / 2);
    const orientation = config.hexOrientation;
    const { x, y, width, height } = bounds;

    const hexWidth = orientation === "pointy-top" ? Math.sqrt(3) * size : 2 * size;
    const hexHeight = orientation === "pointy-top" ? 2 * size : Math.sqrt(3) * size;
    const horizSpacing = orientation === "pointy-top" ? hexWidth : hexWidth * 0.75;
    const vertSpacing = orientation === "pointy-top" ? hexHeight * 0.75 : hexHeight;

    const originX = x + config.offsetX;
    const originY = y + config.offsetY;

    const colStart = Math.max(-MAX_GRID_LINES, Math.floor((x - originX) / horizSpacing) - 1);
    const colEnd = Math.min(MAX_GRID_LINES, Math.ceil((x + width - originX) / horizSpacing) + 1);
    const rowStart = Math.max(-MAX_GRID_LINES, Math.floor((y - originY) / vertSpacing) - 1);
    const rowEnd = Math.min(MAX_GRID_LINES, Math.ceil((y + height - originY) / vertSpacing) + 1);

    for (let col = colStart; col <= colEnd; col += 1) {
      for (let row = rowStart; row <= rowEnd; row += 1) {
        let cx;
        let cy;
        if (orientation === "pointy-top") {
          cx = originX + (col * horizSpacing) + ((row % 2 !== 0) ? horizSpacing / 2 : 0);
          cy = originY + (row * vertSpacing);
        } else {
          cx = originX + (col * horizSpacing);
          cy = originY + (row * vertSpacing) + ((col % 2 !== 0) ? vertSpacing / 2 : 0);
        }
        if (cx < x - hexWidth || cx > x + width + hexWidth) continue;
        if (cy < y - hexHeight || cy > y + height + hexHeight) continue;
        drawHexAt(graphics, cx, cy, size, orientation);
      }
    }
  }

  function render(layer, rawConfig, bounds) {
    if (!layer || !window.PIXI) return;
    clear(layer);

    const config = normalizeConfig(rawConfig);
    if (!config.visible || config.gridType === "none") {
      return;
    }
    if (!bounds || !bounds.width || !bounds.height) {
      return;
    }

    const graphics = new PIXI.Graphics();
    graphics.lineStyle(config.lineWidth, resolveLineColor(config.lineStyle), config.opacity);

    if (config.gridType === "square") {
      drawSquareGrid(graphics, config, bounds);
    } else if (config.gridType === "hex") {
      drawHexGrid(graphics, config, bounds);
    }

    const mask = new PIXI.Graphics();
    mask.beginFill(0xffffff);
    mask.drawRect(bounds.x, bounds.y, bounds.width, bounds.height);
    mask.endFill();
    mask.renderable = false;
    layer.addChild(mask);
    layer.mask = mask;
    layer.addChild(graphics);
  }

  window.VictoryPixiGrid = {
    render,
    clear,
    normalizeConfig,
    snapPoint,
  };
})();
