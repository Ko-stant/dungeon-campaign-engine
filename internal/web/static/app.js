/**
 * @typedef {Object} TileAddress
 * @property {string} segmentId
 * @property {number} x
 * @property {number} y
 */
/**
 * @typedef {Object} EntityLite
 * @property {string} id
 * @property {string} kind
 * @property {TileAddress} tile
 */
/**
 * @typedef {Object} Snapshot
 * @property {string} mapId
 * @property {string} packId
 * @property {number} turn
 * @property {number} lastEventId
 * @property {number} mapWidth
 * @property {number} mapHeight
 * @property {number} regionsCount
 * @property {number[]} tileRegionIds
 * @property {number[]} revealedRegionIds
 * @property {EntityLite[]} entities
 * @property {string} protocolVersion
 * @property {ThresholdLite[]} thresholds
 * @property {BlockingWallLite[]} blockingWalls
 * @property {FurnitureLite[]} furniture
 * @property {MonsterLite[]} monsters
 * @property {number[]} visibleRegionIds
 * @property {number} corridorRegionId
 * @property {number[]} knownRegionIds
 */

/**
 * @typedef {Object} ThresholdLite
 * @property {string} id
 * @property {number} x
 * @property {number} y
 * @property {"vertical"|"horizontal"} orientation
 * @property {"DoorSocket"} kind
 * @property {"open"|"closed"} [state]
 */
/**
 * @typedef {Object} BlockingWallLite
 * @property {string} id
 * @property {number} x
 * @property {number} y
 * @property {"vertical"|"horizontal"} orientation
 * @property {number} size
 */

/**
 * @typedef {Object} FurnitureLite
 * @property {string} id
 * @property {string} type
 * @property {TileAddress} tile
 * @property {{width: number, height: number}} gridSize
 * @property {number} [rotation] - 0, 90, 180, 270 degrees
 * @property {boolean} [swapAspectOnRotate] - Whether to swap width/height for 90/270 rotations
 * @property {string} tileImage
 * @property {string} tileImageCleaned
 * @property {{width: number, height: number}} pixelDimensions
 * @property {boolean} blocksLineOfSight
 * @property {boolean} blocksMovement
 * @property {string[]} [contains]
 */

/**
 * @typedef {Object} MonsterLite
 * @property {string} id
 * @property {string} type
 * @property {TileAddress} tile
 * @property {number} body
 * @property {number} MaxBody
 * @property {boolean} isVisible
 * @property {boolean} isAlive
 */

const snapshotElement = document.getElementById("snapshot");
/** @type {Snapshot|null} */
let snapshot = null;
if (snapshotElement) {
  snapshot = JSON.parse(snapshotElement.textContent);
  window.__SNAPSHOT__ = snapshot;
}

/** @type {HTMLCanvasElement} */
const canvas = document.getElementById("board");
/** @type {CanvasRenderingContext2D} */
const canvasContext = canvas.getContext("2d");
/** @type {HTMLElement} */
const patchCountElement = document.getElementById("patchCount");
/** @type {HTMLButtonElement|null} */
const toggleDoorButton = document.getElementById("toggleDoor");

/** @type {string|null} */
let selectedDoorId = null;

let patchCount = 0;
/** @type {Set<number>} */
let revealedRegions = new Set(Array.isArray(snapshot?.revealedRegionIds) ? snapshot.revealedRegionIds : []);
/** @type {WebSocket|null} */
let socketRef = null;
/** @type {Map<string, TileAddress>} */
let entityPositions = new Map();
if (Array.isArray(snapshot?.entities)) {
  for (const e of snapshot.entities) entityPositions.set(e.id, structuredClone(e.tile));
}

/** @type {ThresholdLite[]} */
let thresholds = Array.isArray(snapshot?.thresholds) ? snapshot.thresholds.slice() : [];
/** @type {BlockingWallLite[]} */
let blockingWalls = Array.isArray(snapshot?.blockingWalls) ? snapshot.blockingWalls.slice() : [];
/** @type {FurnitureLite[]} */
let furniture = Array.isArray(snapshot?.furniture) ? snapshot.furniture.slice() : [];
console.log("DEBUG: Furniture data loaded:", furniture.length, "items", furniture);
/** @type {MonsterLite[]} */
let monsters = Array.isArray(snapshot?.monsters) ? snapshot.monsters.slice() : [];
console.log("DEBUG: Monster data loaded:", monsters.length, "items", monsters);
/** @type {Set<number>} */
let visibleNow = new Set(Array.isArray(snapshot?.visibleRegionIds) ? snapshot.visibleRegionIds : []);
/** @type {number} */
const corridorRegionId = typeof snapshot?.corridorRegionId === "number" ? snapshot.corridorRegionId : 0;
/** @type {Set<number>} */
let knownRegions = new Set(Array.isArray(snapshot?.knownRegionIds) ? snapshot.knownRegionIds : []);


function resizeCanvas() {
  const clientRect = canvas.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;
  canvas.width = Math.floor(clientRect.width * dpr);
  canvas.height = Math.floor(clientRect.height * dpr);
  canvasContext.setTransform(dpr, 0, 0, dpr, 0, 0);
}

function gridMetrics() {
  const clientRect = canvas.getBoundingClientRect();
  const cols = snapshot?.mapWidth ?? 26;
  const rows = snapshot?.mapHeight ?? 19;
  const tile = Math.floor(Math.min(clientRect.width / cols, clientRect.height / rows));
  const gridW = tile * cols;
  const gridH = tile * rows;
  const ox = Math.floor((clientRect.width - gridW) / 2);
  const oy = Math.floor((clientRect.height - gridH) / 2);
  return { cols, rows, tile, ox, oy, width: clientRect.width, height: clientRect.height };
}

/**
 * @param {number} x
 * @param {number} y
 * @param {number} tile
 * @param {number} ox
 * @param {number} oy
 * @returns {{x:number,y:number,w:number,h:number,cx:number,cy:number}}
 */
function tileRect(x, y, tile, ox, oy) {
  const px = ox + x * tile;
  const py = oy + y * tile;
  const cx = px + tile / 2;
  const cy = py + tile / 2;
  return { x: px, y: py, w: tile, h: tile, cx, cy };
}

function drawDoorOverlays() {
  const m = gridMetrics();
  if (!Array.isArray(thresholds) || thresholds.length === 0) return;

  const cs = getComputedStyle(document.documentElement);
  const closedRGB = cs.getPropertyValue("--color-brand").trim();
  const openRGB = cs.getPropertyValue("--color-positive").trim();

  canvasContext.save();
  canvasContext.lineWidth = 2;
  for (const t of thresholds) {
    const cs = getComputedStyle(document.documentElement);
    const closedRGB = cs.getPropertyValue("--color-brand").trim();
    const openRGB = cs.getPropertyValue("--color-positive").trim();
    const accentRGB = cs.getPropertyValue("--color-accent").trim();

    const isSelected = selectedDoorId === t.id;
    const base = t.state === "open" ? openRGB : closedRGB;
    canvasContext.strokeStyle = `rgb(${isSelected ? accentRGB : base})`;
    canvasContext.lineWidth = isSelected ? 3 : 2;

    const m = gridMetrics();
    const gap = Math.max(2, Math.floor(m.tile * 0.15));
    if (t.orientation === "vertical") {
      // Vertical door at (x,y) = left edge of tile (x,y)
      const xLine = m.ox + t.x * m.tile + 0.5;
      const y1 = m.oy + t.y * m.tile + gap + 0.5;
      const y2 = y1 + (m.tile - 2 * gap);
      canvasContext.beginPath();
      canvasContext.moveTo(xLine, y1);
      canvasContext.lineTo(xLine, y2);
      canvasContext.stroke();
    } else {
      // Horizontal door at (x,y) = top edge of tile (x,y)
      const yLine = m.oy + t.y * m.tile + 0.5;
      const x1 = m.ox + t.x * m.tile + gap + 0.5;
      const x2 = x1 + (m.tile - 2 * gap);
      canvasContext.beginPath();
      canvasContext.moveTo(x1, yLine);
      canvasContext.lineTo(x2, yLine);
      canvasContext.stroke();
    }
  }

  canvasContext.restore();
}

// Image cache for blocking wall textures
const blockingWallImageCache = new Map();
const monsterImageCache = new Map();

function drawBlockingWalls() {
  const m = gridMetrics();
  if (!Array.isArray(blockingWalls) || blockingWalls.length === 0) return;

  console.log("DEBUG: Drawing", blockingWalls.length, "blocking walls");
  canvasContext.save();

  for (const wall of blockingWalls) {
    const size = wall.size || 1;

    // Try to load and draw the actual blocking wall image
    drawBlockingWallWithImage(wall, size, m);
  }

  canvasContext.restore();
}

function drawBlockingWallWithImage(wall, size, m) {
  // Choose the appropriate blocking wall image based on size
  const imageUrl = size === 1
    ? "assets/tiles_cleaned/general/blocked_tile_1x1.png"
    : "assets/tiles_cleaned/general/blocked_tile_2x1.png";

  // Check if image is already cached
  if (blockingWallImageCache.has(imageUrl)) {
    const cachedImage = blockingWallImageCache.get(imageUrl);
    if (cachedImage && cachedImage.complete && cachedImage.naturalWidth > 0) {
      console.log("DEBUG: Using cached blocking wall image for size", size);
      drawBlockingWallImage(cachedImage, wall, size, m);
      return;
    } else if (cachedImage === null) {
      // Image failed to load, use fallback
      drawBlockingWallFallback(wall, size, m);
      return;
    }
  }

  // Load image if not cached
  const img = new Image();
  img.onload = function() {
    console.log("DEBUG: Blocking wall image loaded successfully for size", size, "from", imageUrl);
    blockingWallImageCache.set(imageUrl, img);
    // Redraw the canvas to show the loaded image
    requestAnimationFrame(() => drawBoard());
  };
  img.onerror = function() {
    console.log("DEBUG: Blocking wall image failed to load for size", size, "from", imageUrl, "using fallback");
    // Mark as failed so we don't try again
    blockingWallImageCache.set(imageUrl, null);
  };
  img.src = imageUrl;

  // Use fallback while image loads
  drawBlockingWallFallback(wall, size, m);
}

function drawBlockingWallImage(img, wall, size, m) {
  for (let i = 0; i < size; i++) {
    let tileX = wall.x;
    let tileY = wall.y;

    // Offset for multi-tile walls
    if (wall.orientation === "horizontal") {
      tileX += i;
    } else {
      tileY += i;
    }

    const r = tileRect(tileX, tileY, m.tile, m.ox, m.oy);

    // For multi-tile walls, draw the appropriate section of the image
    if (size === 1) {
      // Single tile - draw entire image
      canvasContext.drawImage(img, r.x, r.y, r.w, r.h);
    } else {
      // Multi-tile - draw appropriate section
      const srcX = wall.orientation === "horizontal" ? (i / size) * img.naturalWidth : 0;
      const srcY = wall.orientation === "vertical" ? (i / size) * img.naturalHeight : 0;
      const srcWidth = wall.orientation === "horizontal" ? img.naturalWidth / size : img.naturalWidth;
      const srcHeight = wall.orientation === "vertical" ? img.naturalHeight / size : img.naturalHeight;

      canvasContext.drawImage(
        img,
        srcX, srcY, srcWidth, srcHeight,  // source rectangle
        r.x, r.y, r.w, r.h               // destination rectangle
      );
    }
  }
}

function drawBlockingWallFallback(wall, size, m) {
  const cs = getComputedStyle(document.documentElement);
  const wallColor = cs.getPropertyValue("--color-danger").trim(); // Use danger color for blocking walls

  canvasContext.fillStyle = `rgb(${wallColor})`;
  canvasContext.strokeStyle = `rgb(${wallColor})`;
  canvasContext.lineWidth = 2;

  for (let i = 0; i < size; i++) {
    let tileX = wall.x;
    let tileY = wall.y;

    // Offset for multi-tile walls
    if (wall.orientation === "horizontal") {
      tileX += i;
    } else {
      tileY += i;
    }

    // Draw blocking wall as a filled tile with an X pattern
    const r = tileRect(tileX, tileY, m.tile, m.ox, m.oy);

    // Fill the tile with a semi-transparent background
    canvasContext.save();
    canvasContext.globalAlpha = 0.3;
    canvasContext.fillRect(r.x, r.y, r.w, r.h);
    canvasContext.restore();

    // Draw X pattern to indicate blocked tile
    const margin = Math.floor(m.tile * 0.2);
    canvasContext.beginPath();
    canvasContext.moveTo(r.x + margin, r.y + margin);
    canvasContext.lineTo(r.x + r.w - margin, r.y + r.h - margin);
    canvasContext.moveTo(r.x + r.w - margin, r.y + margin);
    canvasContext.lineTo(r.x + margin, r.y + r.h - margin);
    canvasContext.stroke();
  }
}

// Image cache for furniture textures
const furnitureImageCache = new Map();

function drawFurniture() {
  const m = gridMetrics();
  console.log("DEBUG: drawFurniture called, furniture array:", furniture);
  if (!Array.isArray(furniture) || furniture.length === 0) {
    console.log("DEBUG: No furniture to draw");
    return;
  }

  console.log("DEBUG: Drawing", furniture.length, "furniture items");
  canvasContext.save();

  for (const item of furniture) {
    // Calculate the area this furniture occupies
    const startX = item.tile.x;
    const startY = item.tile.y;
    const gridWidth = item.gridSize.width;
    const gridHeight = item.gridSize.height;

    // Try to load and draw the actual image, fall back to colored rectangle if needed
    drawFurnitureWithImage(item, startX, startY, gridWidth, gridHeight, m);
  }

  canvasContext.restore();
}

function drawFurnitureWithImage(item, startX, startY, gridWidth, gridHeight, m) {
  // Prefer cleaned image, fallback to regular tile image
  const imageUrl = item.tileImageCleaned || item.tileImage;

  if (!imageUrl) {
    console.log("DEBUG: No image URL for furniture", item.id, "using fallback");
    drawFurnitureFallback(item, startX, startY, gridWidth, gridHeight, m);
    return;
  }

  // Check if image is already cached
  if (furnitureImageCache.has(imageUrl)) {
    const cachedImage = furnitureImageCache.get(imageUrl);
    if (cachedImage.complete && cachedImage.naturalWidth > 0) {
      console.log("DEBUG: Using cached image for", item.id);
      drawFurnitureImageWithRotation(cachedImage, item, startX, startY, gridWidth, gridHeight, m);
      return;
    }
  }

  // Load image if not cached
  const img = new Image();
  img.onload = function() {
    console.log("DEBUG: Image loaded successfully for", item.id, "from", imageUrl);
    furnitureImageCache.set(imageUrl, img);
    // Redraw the canvas to show the loaded image
    requestAnimationFrame(() => drawBoard());
  };
  img.onerror = function() {
    console.log("DEBUG: Image failed to load for", item.id, "from", imageUrl, "using fallback");
    // Mark as failed so we don't try again
    furnitureImageCache.set(imageUrl, null);
  };
  img.src = imageUrl;

  // Use fallback while image loads
  drawFurnitureFallback(item, startX, startY, gridWidth, gridHeight, m);
}

function drawFurnitureImageWithRotation(img, item, startX, startY, gridWidth, gridHeight, m) {
  const rotation = item.rotation || 0;

  console.log("DEBUG: Drawing furniture", item.id, "rotation:", rotation, "swapAspectOnRotate:", item.swapAspectOnRotate, "gridSize:", gridWidth + "x" + gridHeight);

  if (rotation === 0) {
    // No rotation, use original method
    console.log("DEBUG: No rotation for", item.id, "using original dimensions", gridWidth + "x" + gridHeight);
    drawFurnitureImage(img, item, startX, startY, gridWidth, gridHeight, m);
    return;
  }

  // For aspect-swapped furniture, we need to handle rotation differently
  let renderStartX = startX;
  let renderStartY = startY;
  let renderWidth = gridWidth;
  let renderHeight = gridHeight;

  if (item.swapAspectOnRotate && (rotation === 90 || rotation === 270)) {
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

    console.log("DEBUG: Image - Aspect swap positioning: original", gridWidth + "x" + gridHeight, "in swapped area", swappedWidth + "x" + swappedHeight, "offset:", widthOffset + "," + heightOffset);
  }

  // Calculate the center for rotation (use the render position)
  const centerX = renderStartX + renderWidth / 2;
  const centerY = renderStartY + renderHeight / 2;
  const centerPixelX = centerX * m.tile + m.ox;
  const centerPixelY = centerY * m.tile + m.oy;

  // Save the current canvas state
  canvasContext.save();

  // Move to center, rotate, then move back
  canvasContext.translate(centerPixelX, centerPixelY);
  canvasContext.rotate((rotation * Math.PI) / 180);
  canvasContext.translate(-centerPixelX, -centerPixelY);

  // Draw the furniture using original dimensions (rotation handles the visual effect)
  console.log("DEBUG: About to call drawFurnitureImage with", item.id, "original dimensions:", renderWidth + "x" + renderHeight, "at render position:", renderStartX + "," + renderStartY);
  drawFurnitureImage(img, item, renderStartX, renderStartY, renderWidth, renderHeight, m);

  // Restore the canvas state
  canvasContext.restore();
}

function drawFurnitureImage(img, item, startX, startY, gridWidth, gridHeight, m) {
  console.log("DEBUG: drawFurnitureImage called for", item.id, "with dimensions", gridWidth + "x" + gridHeight, "at", startX + "," + startY);
  // Draw the furniture image across the grid area it occupies
  for (let dy = 0; dy < gridHeight; dy++) {
    for (let dx = 0; dx < gridWidth; dx++) {
      const r = tileRect(startX + dx, startY + dy, m.tile, m.ox, m.oy);

      // Calculate which part of the source image to use for this tile
      const srcX = (dx / gridWidth) * img.naturalWidth;
      const srcY = (dy / gridHeight) * img.naturalHeight;
      const srcWidth = img.naturalWidth / gridWidth;
      const srcHeight = img.naturalHeight / gridHeight;

      canvasContext.drawImage(
        img,
        srcX, srcY, srcWidth, srcHeight,  // source rectangle
        r.x, r.y, r.w, r.h               // destination rectangle
      );
    }
  }
}

function drawFurnitureFallback(item, startX, startY, gridWidth, gridHeight, m) {
  const rotation = item.rotation || 0;

  console.log("DEBUG: Drawing furniture fallback", item.id, "rotation:", rotation, "swapAspectOnRotate:", item.swapAspectOnRotate, "gridSize:", gridWidth + "x" + gridHeight);
  console.log("DEBUG: Full furniture item:", JSON.stringify(item, null, 2));

  if (rotation === 0) {
    // No rotation, use original method
    drawFurnitureFallbackNoRotation(item, startX, startY, gridWidth, gridHeight, m);
    return;
  }

  // For aspect-swapped furniture, we need to handle rotation differently
  let renderStartX = startX;
  let renderStartY = startY;
  let renderWidth = gridWidth;
  let renderHeight = gridHeight;

  if (item.swapAspectOnRotate && (rotation === 90 || rotation === 270)) {
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

    console.log("DEBUG: Fallback - Aspect swap positioning: original", gridWidth + "x" + gridHeight, "in swapped area", swappedWidth + "x" + swappedHeight, "offset:", widthOffset + "," + heightOffset);
  }

  // Calculate the center for rotation (use the render position)
  const centerX = renderStartX + renderWidth / 2;
  const centerY = renderStartY + renderHeight / 2;
  const centerPixelX = centerX * m.tile + m.ox;
  const centerPixelY = centerY * m.tile + m.oy;

  // Save the current canvas state
  canvasContext.save();

  // Move to center, rotate, then move back
  canvasContext.translate(centerPixelX, centerPixelY);
  canvasContext.rotate((rotation * Math.PI) / 180);
  canvasContext.translate(-centerPixelX, -centerPixelY);

  // Draw the furniture using original dimensions (rotation handles the visual effect)
  console.log("DEBUG: About to call drawFurnitureFallbackNoRotation with", item.id, "original dimensions:", renderWidth + "x" + renderHeight, "at render position:", renderStartX + "," + renderStartY);
  drawFurnitureFallbackNoRotation(item, renderStartX, renderStartY, renderWidth, renderHeight, m);

  // Restore the canvas state
  canvasContext.restore();
}

function drawFurnitureFallbackNoRotation(item, startX, startY, gridWidth, gridHeight, m) {
  // Color-coded fallback for different furniture types
  const cs = getComputedStyle(document.documentElement);
  let fillColor, strokeColor;

  switch (item.type) {
    case 'stairwell':
      fillColor = `rgb(${cs.getPropertyValue("--color-accent").trim()})`;
      strokeColor = `rgb(${cs.getPropertyValue("--color-content").trim()})`;
      break;
    case 'chest':
      fillColor = `rgb(${cs.getPropertyValue("--color-positive").trim()})`;
      strokeColor = `rgb(${cs.getPropertyValue("--color-content").trim()})`;
      break;
    case 'table':
    case 'alchemists_table':
    case 'sorcerers_table':
      fillColor = `rgb(${cs.getPropertyValue("--color-surface-2").trim()})`;
      strokeColor = `rgb(${cs.getPropertyValue("--color-border-rgb").trim()})`;
      break;
    default:
      fillColor = `rgb(${cs.getPropertyValue("--color-surface-3", "--color-surface-2").trim()})`;
      strokeColor = `rgb(${cs.getPropertyValue("--color-border-rgb").trim()})`;
  }

  // Draw furniture area
  for (let dy = 0; dy < gridHeight; dy++) {
    for (let dx = 0; dx < gridWidth; dx++) {
      const r = tileRect(startX + dx, startY + dy, m.tile, m.ox, m.oy);

      // Fill the tile
      canvasContext.fillStyle = fillColor;
      canvasContext.fillRect(r.x + 1, r.y + 1, r.w - 2, r.h - 2);

      // Draw border
      canvasContext.strokeStyle = strokeColor;
      canvasContext.lineWidth = 1;
      canvasContext.strokeRect(r.x + 0.5, r.y + 0.5, r.w - 1, r.h - 1);
    }
  }

  // Add type indicator in center for multi-tile furniture
  if (gridWidth > 1 || gridHeight > 1) {
    const centerR = tileRect(startX, startY, m.tile, m.ox, m.oy);
    const centerX = centerR.x + (gridWidth * m.tile) / 2;
    const centerY = centerR.y + (gridHeight * m.tile) / 2;

    canvasContext.fillStyle = strokeColor;
    canvasContext.font = `${Math.floor(m.tile * 0.2)}px monospace`;
    canvasContext.textAlign = "center";
    canvasContext.textBaseline = "middle";
    canvasContext.fillText(item.type.charAt(0).toUpperCase(), centerX, centerY);
  }
}

/**
 * @param {string} entityId
 * @param {number} dx
 * @param {number} dy
 * @returns {string|null}
 */
function findAdjacentDoorIdFromDirection(entityId, dx, dy) {
  const pos = entityPositions.get(entityId);
  if (!pos) return null;
  /** @type {{x:number,y:number,orientation:"vertical"|"horizontal"}|null} */
  let edge = null;
  // Updated to match new coordinate system:
  // Horizontal door (x,y) = top edge of tile (x,y)
  // Vertical door (x,y) = left edge of tile (x,y)
  if (dx === 1 && dy === 0) edge = { x: pos.x + 1, y: pos.y, orientation: "vertical" };      // right: left edge of tile to the right
  else if (dx === -1 && dy === 0) edge = { x: pos.x, y: pos.y, orientation: "vertical" };   // left: left edge of current tile
  else if (dx === 0 && dy === 1) edge = { x: pos.x, y: pos.y + 1, orientation: "horizontal" }; // down: top edge of tile below
  else if (dx === 0 && dy === -1) edge = { x: pos.x, y: pos.y, orientation: "horizontal" };     // up: top edge of current tile
  if (!edge) return null;
  const t = thresholds.find(d => d.orientation === edge.orientation && d.x === edge.x && d.y === edge.y);
  return t ? t.id : null;
}

/**
 * @param {number} dx
 * @param {number} dy
 */
function selectDoorInDirection(dx, dy) {
  selectedDoorId = findAdjacentDoorIdFromDirection("hero-1", dx, dy);
  drawBoard();
}

function drawBoard() {
  const m = gridMetrics();

  const cs = getComputedStyle(document.documentElement);
  const colorSurface = cs.getPropertyValue("--color-surface").trim();
  const colorSurface2 = cs.getPropertyValue("--color-surface-2").trim();
  const colorBorder = cs.getPropertyValue("--color-border-rgb").trim();

  canvasContext.clearRect(0, 0, m.width, m.height);
  canvasContext.fillStyle = `rgb(${colorSurface})`;
  canvasContext.fillRect(0, 0, m.width, m.height);

  if (Array.isArray(snapshot?.tileRegionIds)) {
    for (let y = 0; y < m.rows; y++) {
      for (let x = 0; x < m.cols; x++) {
        const idx = y * m.cols + x;
        const rid = snapshot.tileRegionIds[idx];
        const visible = revealedRegions.has(rid);
        canvasContext.fillStyle = visible ? `rgb(${colorSurface2})` : `rgb(${colorSurface})`;
        const r = tileRect(x, y, m.tile, m.ox, m.oy);
        canvasContext.fillRect(r.x, r.y, r.w, r.h);
      }
    }
  }

  drawGrid();          // subtle per-cell grid for counting
  drawRegionBorders(); // bold room/corridor outlines
  drawFurniture();     // furniture on top of regions
  drawMonsters();      // monsters on top of furniture
  drawDoorOverlays();  // door overlays on top of regions
  drawBlockingWalls(); // blocking walls on top of regions

  for (const [id, t] of entityPositions.entries()) {
    if (!t) continue;
    const r = tileRect(t.x, t.y, m.tile, m.ox, m.oy);
    canvasContext.beginPath();
    canvasContext.arc(r.cx, r.cy, Math.max(2, Math.floor(m.tile * 0.35)), 0, Math.PI * 2);
    canvasContext.closePath();
    canvasContext.fillStyle = `rgb(${cs.getPropertyValue("--color-accent").trim()})`;
    canvasContext.fill();
  }
}

/**
 * Draw a subtle per-cell grid across the whole board for movement counting.
 * Uses half-pixel alignment for crisp 1px lines.
 */
function drawGrid() {
  const m = gridMetrics();
  const cs = getComputedStyle(document.documentElement);
  const borderRGB = cs.getPropertyValue("--color-border").trim(); // "r g b"

  canvasContext.save();
  canvasContext.strokeStyle = `rgb(${borderRGB})`;
  canvasContext.lineWidth = 1;
  canvasContext.globalAlpha = 0.18;

  // Vertical grid lines (x = 0..cols)
  canvasContext.beginPath();
  for (let x = 0; x <= m.cols; x++) {
    const xLine = m.ox + x * m.tile + 0.5;
    canvasContext.moveTo(xLine, m.oy);
    canvasContext.lineTo(xLine, m.oy + m.tile * m.rows);
  }
  canvasContext.stroke();

  // Horizontal grid lines (y = 0..rows)
  canvasContext.beginPath();
  for (let y = 0; y <= m.rows; y++) {
    const yLine = m.oy + y * m.tile + 0.5;
    canvasContext.moveTo(m.ox, yLine);
    canvasContext.lineTo(m.ox + m.tile * m.cols, yLine);
  }
  canvasContext.stroke();

  canvasContext.restore();
}

/**
 * Draw thin region borders only where neighbor tiles belong to different regions.
 * Uses a stronger color so rooms/corridors remain visually distinct when revealed.
 */
function drawRegionBorders() {
  const m = gridMetrics();
  const ids = snapshot?.tileRegionIds;
  if (!Array.isArray(ids)) return;

  const cs = getComputedStyle(document.documentElement);
  const brandRGB = cs.getPropertyValue("--color-brand").trim();

  canvasContext.save();
  canvasContext.strokeStyle = `rgb(${brandRGB})`;
  canvasContext.lineWidth = 1;
  canvasContext.globalAlpha = 0.85;

  // Vertical borders: between (x,y) and (x+1,y)
  for (let y = 0; y < m.rows; y++) {
    for (let x = 0; x < m.cols - 1; x++) {
      const a = ids[y * m.cols + x];
      const b = ids[y * m.cols + (x + 1)];
      if (a !== b) {
        const roomSideA = (a !== corridorRegionId) ? a : null;
        const roomSideB = (b !== corridorRegionId) ? b : null;

        const show =
          (roomSideA !== null && (knownRegions.has(roomSideA) || revealedRegions.has(roomSideA))) ||
          (roomSideB !== null && (knownRegions.has(roomSideB) || revealedRegions.has(roomSideB)));
        if (show) {
          const r = tileRect(x, y, m.tile, m.ox, m.oy);
          const xLine = r.x + r.w + 0.5;
          canvasContext.beginPath();
          canvasContext.moveTo(xLine, r.y);
          canvasContext.lineTo(xLine, r.y + r.h);
          canvasContext.stroke();
        }
      }
    }
  }
  // Horizontal borders: between (x,y) and (x,y+1)
  for (let y = 0; y < m.rows - 1; y++) {
    for (let x = 0; x < m.cols; x++) {
      const a = ids[y * m.cols + x];
      const b = ids[(y + 1) * m.cols + x];
      if (a !== b) {
        const roomSideA = (a !== corridorRegionId) ? a : null;
        const roomSideB = (b !== corridorRegionId) ? b : null;

        const show =
          (roomSideA !== null && (knownRegions.has(roomSideA) || revealedRegions.has(roomSideA))) ||
          (roomSideB !== null && (knownRegions.has(roomSideB) || revealedRegions.has(roomSideB)));
        if (show) {
          const r = tileRect(x, y, m.tile, m.ox, m.oy);
          const yLine = r.y + r.h + 0.5;
          canvasContext.beginPath();
          canvasContext.moveTo(r.x, yLine);
          canvasContext.lineTo(r.x + r.w, yLine);
          canvasContext.stroke();
        }
      }
    }
  }
  canvasContext.restore();
}


function applyPatch(patch) {
  if (patch.type === "VariablesChanged" && patch.payload && patch.payload.entries) {
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
  } else if (patch.type === "RegionsRevealed" && patch.payload && Array.isArray(patch.payload.ids)) {
    for (const id of patch.payload.ids) revealedRegions.add(id);
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  } else if (patch.type === "DoorStateChanged") {
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    const p = patch.payload;
    const idx = thresholds.findIndex(d => d.id === p.thresholdId);
    if (idx !== -1) {
      thresholds[idx] = { ...thresholds[idx], state: p.state };
      drawBoard();
    }
  } else if (patch.type === "EntityUpdated" && patch.payload && patch.payload.tile) {
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    const p = patch.payload;
    entityPositions.set(p.id, p.tile);
    drawBoard();
  } else if (patch.type === "VisibleNow" && patch.payload && Array.isArray(patch.payload.ids)) {
    visibleNow = new Set(patch.payload.ids);
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  } else if (patch.type === "RegionsKnown" && patch.payload && Array.isArray(patch.payload.ids)) {
    for (const id of patch.payload.ids) knownRegions.add(id);
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  } else if (patch.type === "DoorsVisible" && patch.payload && Array.isArray(patch.payload.doors)) {
    console.log("Received DoorsVisible patch:", patch.payload.doors.length, "new doors");
    // Add newly visible doors to existing ones (never remove doors once seen)
    for (const newDoor of patch.payload.doors) {
      const existingIndex = thresholds.findIndex(d => d.id === newDoor.id);
      if (existingIndex === -1) {
        thresholds.push(newDoor);
        console.log("Added new door:", newDoor.id);
      } else {
        // Update existing door (in case state changed)
        thresholds[existingIndex] = newDoor;
      }
    }
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  } else if (patch.type === "BlockingWallsVisible" && patch.payload && Array.isArray(patch.payload.blockingWalls)) {
    console.log("Received BlockingWallsVisible patch:", patch.payload.blockingWalls.length, "new blocking walls");
    // Add newly visible blocking walls to existing ones (never remove blocking walls once seen)
    for (const newWall of patch.payload.blockingWalls) {
      const existingIndex = blockingWalls.findIndex(w => w.id === newWall.id);
      if (existingIndex === -1) {
        blockingWalls.push(newWall);
        console.log("Added new blocking wall:", newWall.id);
      } else {
        // Update existing blocking wall
        blockingWalls[existingIndex] = newWall;
      }
    }
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  } else if (patch.type === "FurnitureVisible" && patch.payload && Array.isArray(patch.payload.furniture)) {
    console.log("Received FurnitureVisible patch:", patch.payload.furniture.length, "new furniture");
    // Add newly visible furniture to existing ones (never remove furniture once seen)
    for (const newFurniture of patch.payload.furniture) {
      const existingIndex = furniture.findIndex(f => f.id === newFurniture.id);
      if (existingIndex === -1) {
        furniture.push(newFurniture);
        console.log("Added new furniture:", newFurniture.id);
      } else {
        // Update existing furniture
        furniture[existingIndex] = newFurniture;
      }
    }
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  } else if (patch.type === "MonstersVisible" && patch.payload && Array.isArray(patch.payload.monsters)) {
    console.log("Received MonstersVisible patch:", patch.payload.monsters.length, "new monsters");
    // Add newly visible monsters to existing ones (never remove monsters once seen)
    for (const newMonster of patch.payload.monsters) {
      const existingIndex = monsters.findIndex(m => m.id === newMonster.id);
      if (existingIndex === -1) {
        monsters.push(newMonster);
        console.log("Added new monster:", newMonster.id);
      } else {
        // Update existing monster
        monsters[existingIndex] = newMonster;
      }
    }
    patchCount += 1;
    if (patchCountElement) patchCountElement.textContent = String(patchCount);
    drawBoard();
  }
}

function openStream() {
  console.log("openStream() called");
  const scheme = location.protocol === "https:" ? "wss" : "ws";
  const url = `${scheme}://${location.host}/stream`;
  console.log("Attempting WebSocket connection to:", url);
  const socket = new WebSocket(url);
  socketRef = socket;

  socket.onmessage = (event) => {
    try {
      console.log("Raw WebSocket message:", event.data);
      const patch = JSON.parse(event.data);
      console.log("Parsed patch:", patch.type, patch);
      applyPatch(patch);
    } catch (err) {
      console.error("Failed to parse WebSocket message:", err, event.data);
    }
  };
  socket.onclose = () => {
    setTimeout(openStream, 2000);
  };
}

function drawMonsters() {
  const m = gridMetrics();
  console.log("DEBUG: drawMonsters called, monsters array:", monsters);
  if (!Array.isArray(monsters) || monsters.length === 0) {
    console.log("DEBUG: No monsters to draw");
    return;
  }

  console.log("DEBUG: Drawing", monsters.length, "monsters");
  canvasContext.save();

  for (const monster of monsters) {
    if (!monster.isVisible || !monster.isAlive) {
      continue; // Only draw visible, alive monsters
    }

    const x = monster.tile.x;
    const y = monster.tile.y;
    const r = tileRect(x, y, m.tile, m.ox, m.oy);

    // Draw monster based on type
    drawMonsterWithImage(monster, r);
  }

  canvasContext.restore();
}

function drawMonsterWithImage(monster, rect) {
  // Prefer cleaned image, fallback to regular tile image
  const imageUrl = `assets/tiles_cleaned/monsters/monster_${monster.type}.png`;

  // Check if image is already cached
  if (monsterImageCache.has(imageUrl)) {
    const cachedImage = monsterImageCache.get(imageUrl);
    if (cachedImage && cachedImage.complete && cachedImage.naturalWidth > 0) {
      console.log("DEBUG: Using cached monster image for", monster.type);
      drawMonsterImage(cachedImage, monster, rect);
      return;
    } else if (cachedImage === null) {
      // Image failed to load, use fallback
      drawMonsterFallback(monster, rect);
      return;
    }
  }

  // Load image if not cached
  const img = new Image();
  img.onload = function() {
    console.log("DEBUG: Monster image loaded successfully for", monster.type, "from", imageUrl);
    monsterImageCache.set(imageUrl, img);
    drawMonsterImage(img, monster, rect);
    requestAnimationFrame(redraw); // Redraw to show the loaded image
  };
  img.onerror = function() {
    console.log("DEBUG: Failed to load monster image for", monster.type, "from", imageUrl, "using fallback");
    // Mark as failed so we don't try again
    monsterImageCache.set(imageUrl, null);
    drawMonsterFallback(monster, rect);
    requestAnimationFrame(redraw); // Redraw to show the fallback
  };
  img.src = imageUrl;

  // Don't draw fallback immediately - wait for image to load or fail
  // This prevents double-rendering issues
}

function drawMonsterImage(img, monster, rect) {
  // Draw the monster image to fill the tile
  canvasContext.drawImage(img, rect.x, rect.y, rect.w, rect.h);

  // Add health indicator if damaged
  if (monster.body < monster.MaxBody) {
    canvasContext.fillStyle = 'red';
    canvasContext.font = '12px monospace';
    canvasContext.textAlign = 'center';
    canvasContext.textBaseline = 'middle';
    canvasContext.fillText(`${monster.body}/${monster.MaxBody}`,
                          rect.x + rect.w / 2, rect.y + rect.h - 10);
  }
}

function drawMonsterFallback(monster, rect) {
  // Color-coded fallback for different monster types
  let fillColor, strokeColor;

  switch (monster.type) {
    case 'goblin':
      fillColor = 'rgb(34, 139, 34)'; // Forest green
      strokeColor = 'rgb(0, 100, 0)';
      break;
    case 'orc':
      fillColor = 'rgb(139, 69, 19)'; // Saddle brown
      strokeColor = 'rgb(101, 67, 33)';
      break;
    case 'skeleton':
      fillColor = 'rgb(245, 245, 220)'; // Beige
      strokeColor = 'rgb(211, 211, 211)';
      break;
    case 'zombie':
      fillColor = 'rgb(128, 128, 0)'; // Olive
      strokeColor = 'rgb(85, 107, 47)';
      break;
    case 'mummy':
      fillColor = 'rgb(222, 184, 135)'; // Burlywood
      strokeColor = 'rgb(139, 69, 19)';
      break;
    case 'dread_warrior':
      fillColor = 'rgb(105, 105, 105)'; // Dim gray
      strokeColor = 'rgb(47, 79, 79)';
      break;
    case 'gargoyle':
      fillColor = 'rgb(112, 128, 144)'; // Slate gray
      strokeColor = 'rgb(47, 79, 79)';
      break;
    case 'abomination':
      fillColor = 'rgb(139, 0, 139)'; // Dark magenta
      strokeColor = 'rgb(75, 0, 130)';
      break;
    default:
      fillColor = 'rgb(220, 20, 60)'; // Crimson (unknown monster)
      strokeColor = 'rgb(139, 0, 0)';
  }

  // Fill the tile
  canvasContext.fillStyle = fillColor;
  canvasContext.fillRect(rect.x + 2, rect.y + 2, rect.w - 4, rect.h - 4);

  // Draw border
  canvasContext.strokeStyle = strokeColor;
  canvasContext.lineWidth = 2;
  canvasContext.strokeRect(rect.x + 1, rect.y + 1, rect.w - 2, rect.h - 2);

  // Add type indicator text
  canvasContext.fillStyle = 'white';
  canvasContext.font = '10px monospace';
  canvasContext.textAlign = 'center';
  canvasContext.textBaseline = 'middle';

  // Abbreviate monster name for display
  let displayText = monster.type.charAt(0).toUpperCase();
  if (monster.type === 'dread_warrior') displayText = 'DW';

  canvasContext.fillText(displayText, rect.x + rect.w / 2, rect.y + rect.h / 2);

  // Add body indicator if damaged
  if (monster.body < monster.MaxBody) {
    canvasContext.fillStyle = 'red';
    canvasContext.font = '8px monospace';
    canvasContext.fillText(`${monster.body}/${monster.MaxBody}`,
                          rect.x + rect.w / 2, rect.y + rect.h - 8);
  }
}

/**
 * @param {KeyboardEvent} ev
 * @returns {{dx:number, dy:number}|null}
 */
function keyToStep(ev) {
  switch (ev.key) {
    case "ArrowLeft":
    case "a":
    case "A":
      return { dx: -1, dy: 0 };
    case "ArrowRight":
    case "d":
    case "D":
      return { dx: 1, dy: 0 };
    case "ArrowUp":
    case "w":
    case "W":
      return { dx: 0, dy: -1 };
    case "ArrowDown":
    case "s":
    case "S":
      return { dx: 0, dy: 1 };
    default:
      return null;
  }
}

window.addEventListener("keydown", (ev) => {
  const step = keyToStep(ev);
  if (step) {
    selectDoorInDirection(step.dx, step.dy); // always update selection
    if (socketRef && socketRef.readyState === WebSocket.OPEN) {
      const msg = { type: "RequestMove", payload: { entityId: "hero-1", dx: step.dx, dy: step.dy } };
      socketRef.send(JSON.stringify(msg));
    }
    ev.preventDefault();
    return;
  }
  if (ev.key === "e" || ev.key === "E") {
    if (selectedDoorId && socketRef && socketRef.readyState === WebSocket.OPEN) {
      const msg = { type: "RequestToggleDoor", payload: { thresholdId: selectedDoorId } };
      socketRef.send(JSON.stringify(msg));
    }
    ev.preventDefault();
  }
});


window.addEventListener("resize", () => {
  resizeCanvas();
  drawBoard();
});

resizeCanvas();
drawBoard();
openStream();
console.log('App initialized');