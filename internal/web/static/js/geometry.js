/**
 * Geometry utilities for grid calculations and positioning
 * These functions are pure and easily testable
 */

/**
 * Calculate grid metrics based on canvas size and map dimensions
 * @param {HTMLCanvasElement} canvas
 * @param {number} mapWidth
 * @param {number} mapHeight
 * @returns {{cols: number, rows: number, tile: number, ox: number, oy: number, width: number, height: number}}
 */
export function calculateGridMetrics(canvas, mapWidth = 26, mapHeight = 19) {
  const clientRect = canvas.getBoundingClientRect();
  const cols = mapWidth;
  const rows = mapHeight;
  const tile = Math.floor(Math.min(clientRect.width / cols, clientRect.height / rows));
  const gridW = tile * cols;
  const gridH = tile * rows;
  const ox = Math.floor((clientRect.width - gridW) / 2);
  const oy = Math.floor((clientRect.height - gridH) / 2);

  return {
    cols,
    rows,
    tile,
    ox,
    oy,
    width: clientRect.width,
    height: clientRect.height,
  };
}

/**
 * Calculate pixel rectangle for a grid tile
 * @param {number} x - Grid x coordinate
 * @param {number} y - Grid y coordinate
 * @param {number} tileSize - Size of each tile in pixels
 * @param {number} offsetX - Grid offset X
 * @param {number} offsetY - Grid offset Y
 * @returns {{x: number, y: number, w: number, h: number, cx: number, cy: number}}
 */
export function calculateTileRect(x, y, tileSize, offsetX, offsetY) {
  const px = offsetX + x * tileSize;
  const py = offsetY + y * tileSize;
  const cx = px + tileSize / 2;
  const cy = py + tileSize / 2;

  return {
    x: px,
    y: py,
    w: tileSize,
    h: tileSize,
    cx,
    cy,
  };
}

/**
 * Convert keyboard event to directional step
 * @param {KeyboardEvent} ev
 * @returns {{dx: number, dy: number}|null}
 */
export function keyToStep(ev) {
  switch (ev.key) {
    case 'ArrowLeft':
    case 'a':
    case 'A':
      return { dx: -1, dy: 0 };
    case 'ArrowRight':
    case 'd':
    case 'D':
      return { dx: 1, dy: 0 };
    case 'ArrowUp':
    case 'w':
    case 'W':
      return { dx: 0, dy: -1 };
    case 'ArrowDown':
    case 's':
    case 'S':
      return { dx: 0, dy: 1 };
    default:
      return null;
  }
}

/**
 * Find door edge position from movement direction
 * @param {TileAddress} entityPos - Current entity position
 * @param {number} dx - Direction X
 * @param {number} dy - Direction Y
 * @returns {{x: number, y: number, orientation: "vertical"|"horizontal"}|null}
 */
export function calculateDoorEdgeFromDirection(entityPos, dx, dy) {
  if (!entityPos) {
    return null;
  }

  let edge = null;

  // Updated to match coordinate system:
  // Horizontal door (x,y) = top edge of tile (x,y)
  // Vertical door (x,y) = left edge of tile (x,y)
  if (dx === 1 && dy === 0) {
    // right: left edge of tile to the right
    edge = { x: entityPos.x + 1, y: entityPos.y, orientation: 'vertical' };
  } else if (dx === -1 && dy === 0) {
    // left: left edge of current tile
    edge = { x: entityPos.x, y: entityPos.y, orientation: 'vertical' };
  } else if (dx === 0 && dy === 1) {
    // down: top edge of tile below
    edge = { x: entityPos.x, y: entityPos.y + 1, orientation: 'horizontal' };
  } else if (dx === 0 && dy === -1) {
    // up: top edge of current tile
    edge = { x: entityPos.x, y: entityPos.y, orientation: 'horizontal' };
  }

  return edge;
}

/**
 * Calculate adjacent tile position from direction
 * @param {TileAddress} entityPos - Current entity position
 * @param {number} dx - Direction X
 * @param {number} dy - Direction Y
 * @returns {{x: number, y: number}|null}
 */
export function calculateAdjacentTile(entityPos, dx, dy) {
  if (!entityPos) {
    return null;
  }

  return {
    x: entityPos.x + dx,
    y: entityPos.y + dy,
  };
}

/**
 * Calculate rotation positioning for furniture with aspect swap
 * @param {number} startX - Start X position
 * @param {number} startY - Start Y position
 * @param {number} gridWidth - Original grid width
 * @param {number} gridHeight - Original grid height
 * @param {number} rotation - Rotation in degrees
 * @param {boolean} swapAspectOnRotate - Whether to swap aspect ratio
 * @returns {{renderStartX: number, renderStartY: number, renderWidth: number, renderHeight: number}}
 */
export function calculateRotationPositioning(startX, startY, gridWidth, gridHeight, rotation, swapAspectOnRotate) {
  let renderStartX = startX;
  let renderStartY = startY;
  const renderWidth = gridWidth;
  const renderHeight = gridHeight;

  if (swapAspectOnRotate && (rotation === 90 || rotation === 270)) {
    // For aspect-swapped furniture, the visual area changes but we draw the original dimensions
    // The collision area is swapped (handled elsewhere), but we render with original dimensions
    // and let the rotation transform handle the visual effect

    // Calculate where to position the original-dimensioned furniture so it appears in the swapped area
    const swappedWidth = gridHeight;
    const swappedHeight = gridWidth;

    // Center the original dimensions within the swapped area
    const widthOffset = (swappedWidth - gridWidth) / 2;
    const heightOffset = (swappedHeight - gridHeight) / 2;

    renderStartX = startX + widthOffset;
    renderStartY = startY + heightOffset;
  }

  return {
    renderStartX,
    renderStartY,
    renderWidth,
    renderHeight,
  };
}

/**
 * Calculate center point for rotation
 * @param {number} startX - Start X position
 * @param {number} startY - Start Y position
 * @param {number} width - Width in grid units
 * @param {number} height - Height in grid units
 * @param {number} tileSize - Size of each tile in pixels
 * @param {number} offsetX - Grid offset X
 * @param {number} offsetY - Grid offset Y
 * @returns {{centerX: number, centerY: number, centerPixelX: number, centerPixelY: number}}
 */
export function calculateRotationCenter(startX, startY, width, height, tileSize, offsetX, offsetY) {
  const centerX = startX + width / 2;
  const centerY = startY + height / 2;
  const centerPixelX = centerX * tileSize + offsetX;
  const centerPixelY = centerY * tileSize + offsetY;

  return {
    centerX,
    centerY,
    centerPixelX,
    centerPixelY,
  };
}

/**
 * Convert degrees to radians
 * @param {number} degrees
 * @returns {number}
 */
export function degreesToRadians(degrees) {
  return (degrees * Math.PI) / 180;
}

/**
 * Calculate canvas size with device pixel ratio
 * @param {HTMLCanvasElement} canvas
 * @returns {{width: number, height: number, dpr: number}}
 */
export function calculateCanvasSize(canvas) {
  const clientRect = canvas.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;
  const width = Math.floor(clientRect.width * dpr);
  const height = Math.floor(clientRect.height * dpr);

  return { width, height, dpr };
}