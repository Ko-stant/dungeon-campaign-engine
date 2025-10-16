/**
 * Entity rendering - monsters, furniture, blocking walls, doors
 */

import {
  getGridMetrics,
  getTileRect,
  getCSSColor,
  getCachedImage,
  scheduleRedraw,
  setupRotationTransform,
  restoreTransform,
  drawImageSafe,
} from './rendering.js';
import {
  calculateRotationPositioning,
  calculateRotationCenter,
} from './geometry.js';
import {
  MONSTER_COLORS,
  FURNITURE_COLORS,
  IMAGE_PATHS,
} from './types.js';
import { gameState } from './gameState.js';

/**
 * Draw all doors with state indicators
 */
export function drawDoors() {
  const m = getGridMetrics();
  const ctx = gameState.canvasContext;
  const thresholds = gameState.thresholds;

  if (!Array.isArray(thresholds) || thresholds.length === 0) {
    return;
  }

  const closedRGB = getCSSColor('--color-brand');
  const openRGB = getCSSColor('--color-positive');
  const accentRGB = getCSSColor('--color-accent');

  ctx.save();

  for (const t of thresholds) {
    const isSelected = gameState.selectedDoorId === t.id;
    const base = t.state === 'open' ? openRGB : closedRGB;
    ctx.strokeStyle = `rgb(${isSelected ? accentRGB : base})`;
    ctx.lineWidth = isSelected ? 3 : 2;

    const gap = Math.max(2, Math.floor(m.tile * 0.15));

    if (t.orientation === 'vertical') {
      const xLine = m.ox + t.x * m.tile + 0.5;
      const y1 = m.oy + t.y * m.tile + gap + 0.5;
      const y2 = y1 + (m.tile - 2 * gap);
      ctx.beginPath();
      ctx.moveTo(xLine, y1);
      ctx.lineTo(xLine, y2);
      ctx.stroke();
    } else {
      const yLine = m.oy + t.y * m.tile + 0.5;
      const x1 = m.ox + t.x * m.tile + gap + 0.5;
      const x2 = x1 + (m.tile - 2 * gap);
      ctx.beginPath();
      ctx.moveTo(x1, yLine);
      ctx.lineTo(x2, yLine);
      ctx.stroke();
    }
  }

  ctx.restore();
}

/**
 * Draw all blocking walls
 */
export function drawBlockingWalls() {
  const m = getGridMetrics();
  const blockingWalls = gameState.blockingWalls;

  if (!Array.isArray(blockingWalls) || blockingWalls.length === 0) {
    return;
  }


  for (const wall of blockingWalls) {
    const size = wall.size || 1;
    drawBlockingWall(wall, size, m);
  }
}

/**
 * Draw a single blocking wall
 * @param {BlockingWallLite} wall
 * @param {number} size
 * @param {Object} metrics
 */
function drawBlockingWall(wall, size, metrics) {
  const imageUrl = size === 1
    ? IMAGE_PATHS.BLOCKING_WALL_1X1
    : IMAGE_PATHS.BLOCKING_WALL_2X1;

  const img = getCachedImage(
    imageUrl,
    gameState.blockingWallImageCache,
    () => scheduleRedraw(() => drawMain()),
    () => console.error('Failed to load blocking wall image for size', size),
  );

  if (img) {
    drawBlockingWallImage(img, wall, size, metrics);
  } else {
    drawBlockingWallFallback(wall, size, metrics);
  }
}

/**
 * Draw blocking wall with image
 * @param {HTMLImageElement} img
 * @param {BlockingWallLite} wall
 * @param {number} size
 * @param {Object} metrics
 */
function drawBlockingWallImage(img, wall, size, metrics) {
  const ctx = gameState.canvasContext;

  for (let i = 0; i < size; i++) {
    let tileX = wall.x;
    let tileY = wall.y;

    if (wall.orientation === 'horizontal') {
      tileX += i;
    } else {
      tileY += i;
    }

    const r = getTileRect(tileX, tileY, metrics);

    if (size === 1) {
      drawImageSafe(ctx, img, 0, 0, null, null, r.x, r.y, r.w, r.h);
    } else {
      const srcX = wall.orientation === 'horizontal' ? (i / size) * img.naturalWidth : 0;
      const srcY = wall.orientation === 'vertical' ? (i / size) * img.naturalHeight : 0;
      const srcWidth = wall.orientation === 'horizontal' ? img.naturalWidth / size : img.naturalWidth;
      const srcHeight = wall.orientation === 'vertical' ? img.naturalHeight / size : img.naturalHeight;

      drawImageSafe(ctx, img, srcX, srcY, srcWidth, srcHeight, r.x, r.y, r.w, r.h);
    }
  }
}

/**
 * Draw blocking wall fallback
 * @param {BlockingWallLite} wall
 * @param {number} size
 * @param {Object} metrics
 */
function drawBlockingWallFallback(wall, size, metrics) {
  const ctx = gameState.canvasContext;
  const wallColor = getCSSColor('--color-danger');

  ctx.save();
  ctx.fillStyle = `rgb(${wallColor})`;
  ctx.strokeStyle = `rgb(${wallColor})`;
  ctx.lineWidth = 2;

  for (let i = 0; i < size; i++) {
    let tileX = wall.x;
    let tileY = wall.y;

    if (wall.orientation === 'horizontal') {
      tileX += i;
    } else {
      tileY += i;
    }

    const r = getTileRect(tileX, tileY, metrics);

    // Fill with semi-transparent background
    ctx.globalAlpha = 0.3;
    ctx.fillRect(r.x, r.y, r.w, r.h);
    ctx.globalAlpha = 1.0;

    // Draw X pattern
    const margin = Math.floor(metrics.tile * 0.2);
    ctx.beginPath();
    ctx.moveTo(r.x + margin, r.y + margin);
    ctx.lineTo(r.x + r.w - margin, r.y + r.h - margin);
    ctx.moveTo(r.x + r.w - margin, r.y + margin);
    ctx.lineTo(r.x + margin, r.y + r.h - margin);
    ctx.stroke();
  }

  ctx.restore();
}

/**
 * Draw all furniture
 */
export function drawFurniture() {
  const m = getGridMetrics();
  const furniture = gameState.furniture;

  if (!Array.isArray(furniture) || furniture.length === 0) {
    return;
  }

  for (const item of furniture) {
    const startX = item.tile.x;
    const startY = item.tile.y;
    const gridWidth = item.gridSize.width;
    const gridHeight = item.gridSize.height;

    drawFurnitureItem(item, startX, startY, gridWidth, gridHeight, m);
  }
}

/**
 * Draw a single furniture item
 * @param {FurnitureLite} item
 * @param {number} startX
 * @param {number} startY
 * @param {number} gridWidth
 * @param {number} gridHeight
 * @param {Object} metrics
 */
function drawFurnitureItem(item, startX, startY, gridWidth, gridHeight, metrics) {
  const imageUrl = item.tileImageCleaned || item.tileImage;

  if (!imageUrl) {
    drawFurnitureFallback(item, startX, startY, gridWidth, gridHeight, metrics);
    return;
  }

  const img = getCachedImage(
    imageUrl,
    gameState.furnitureImageCache,
    () => scheduleRedraw(() => drawMain()),
    () => console.error('Failed to load furniture image for', item.id),
  );

  if (img) {
    drawFurnitureWithRotation(img, item, startX, startY, gridWidth, gridHeight, metrics);
  } else {
    drawFurnitureFallback(item, startX, startY, gridWidth, gridHeight, metrics);
  }
}

/**
 * Draw furniture with rotation handling
 * @param {HTMLImageElement} img
 * @param {FurnitureLite} item
 * @param {number} startX
 * @param {number} startY
 * @param {number} gridWidth
 * @param {number} gridHeight
 * @param {Object} metrics
 */
function drawFurnitureWithRotation(img, item, startX, startY, gridWidth, gridHeight, metrics) {
  const rotation = item.rotation || 0;
  const ctx = gameState.canvasContext;

  if (rotation === 0) {
    drawFurnitureImage(img, item, startX, startY, gridWidth, gridHeight, metrics);
    return;
  }

  const { renderStartX, renderStartY, renderWidth, renderHeight } =
    calculateRotationPositioning(startX, startY, gridWidth, gridHeight, rotation, item.swapAspectOnRotate);

  const { centerPixelX, centerPixelY } =
    calculateRotationCenter(renderStartX, renderStartY, renderWidth, renderHeight, metrics.tile, metrics.ox, metrics.oy);

  setupRotationTransform(ctx, centerPixelX, centerPixelY, rotation);
  drawFurnitureImage(img, item, renderStartX, renderStartY, renderWidth, renderHeight, metrics);
  restoreTransform(ctx);
}

/**
 * Draw furniture image across grid tiles
 * @param {HTMLImageElement} img
 * @param {FurnitureLite} item
 * @param {number} startX
 * @param {number} startY
 * @param {number} gridWidth
 * @param {number} gridHeight
 * @param {Object} metrics
 */
function drawFurnitureImage(img, item, startX, startY, gridWidth, gridHeight, metrics) {
  const ctx = gameState.canvasContext;

  for (let dy = 0; dy < gridHeight; dy++) {
    for (let dx = 0; dx < gridWidth; dx++) {
      const r = getTileRect(startX + dx, startY + dy, metrics);

      const srcX = (dx / gridWidth) * img.naturalWidth;
      const srcY = (dy / gridHeight) * img.naturalHeight;
      const srcWidth = img.naturalWidth / gridWidth;
      const srcHeight = img.naturalHeight / gridHeight;

      drawImageSafe(ctx, img, srcX, srcY, srcWidth, srcHeight, r.x, r.y, r.w, r.h);
    }
  }
}

/**
 * Draw furniture fallback with rotation
 * @param {FurnitureLite} item
 * @param {number} startX
 * @param {number} startY
 * @param {number} gridWidth
 * @param {number} gridHeight
 * @param {Object} metrics
 */
function drawFurnitureFallback(item, startX, startY, gridWidth, gridHeight, metrics) {
  const rotation = item.rotation || 0;
  const ctx = gameState.canvasContext;

  if (rotation === 0) {
    drawFurnitureFallbackNoRotation(item, startX, startY, gridWidth, gridHeight, metrics);
    return;
  }

  const { renderStartX, renderStartY, renderWidth, renderHeight } =
    calculateRotationPositioning(startX, startY, gridWidth, gridHeight, rotation, item.swapAspectOnRotate);

  const { centerPixelX, centerPixelY } =
    calculateRotationCenter(renderStartX, renderStartY, renderWidth, renderHeight, metrics.tile, metrics.ox, metrics.oy);

  setupRotationTransform(ctx, centerPixelX, centerPixelY, rotation);
  drawFurnitureFallbackNoRotation(item, renderStartX, renderStartY, renderWidth, renderHeight, metrics);
  restoreTransform(ctx);
}

/**
 * Draw furniture fallback without rotation
 * @param {FurnitureLite} item
 * @param {number} startX
 * @param {number} startY
 * @param {number} gridWidth
 * @param {number} gridHeight
 * @param {Object} metrics
 */
function drawFurnitureFallbackNoRotation(item, startX, startY, gridWidth, gridHeight, metrics) {
  const ctx = gameState.canvasContext;
  const colors = FURNITURE_COLORS[item.type] || FURNITURE_COLORS.default;

  const fillColor = `rgb(${getCSSColor(colors.fill)})`;
  const strokeColor = `rgb(${getCSSColor(colors.stroke)})`;

  for (let dy = 0; dy < gridHeight; dy++) {
    for (let dx = 0; dx < gridWidth; dx++) {
      const r = getTileRect(startX + dx, startY + dy, metrics);

      ctx.fillStyle = fillColor;
      ctx.fillRect(r.x + 1, r.y + 1, r.w - 2, r.h - 2);

      ctx.strokeStyle = strokeColor;
      ctx.lineWidth = 1;
      ctx.strokeRect(r.x + 0.5, r.y + 0.5, r.w - 1, r.h - 1);
    }
  }

  // Add type indicator for multi-tile furniture
  if (gridWidth > 1 || gridHeight > 1) {
    const centerR = getTileRect(startX, startY, metrics);
    const centerX = centerR.x + (gridWidth * metrics.tile) / 2;
    const centerY = centerR.y + (gridHeight * metrics.tile) / 2;

    ctx.fillStyle = strokeColor;
    ctx.font = `${Math.floor(metrics.tile * 0.2)}px monospace`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(item.type.charAt(0).toUpperCase(), centerX, centerY);
  }
}

/**
 * Draw all monsters
 */
export function drawMonsters() {
  const m = getGridMetrics();
  const monsters = gameState.monsters;

  // Clean up invalid monsters
  gameState.cleanupMonsters();

  if (!Array.isArray(monsters) || monsters.length === 0) {
    return;
  }

  for (const monster of monsters) {
    if (!monster || !monster.tile || !monster.isVisible || !monster.isAlive) {
      continue;
    }

    const r = getTileRect(monster.tile.x, monster.tile.y, m);
    drawMonster(monster, r);
  }
}

/**
 * Draw a single monster
 * @param {MonsterLite} monster
 * @param {Object} rect
 */
function drawMonster(monster, rect) {
  const ctx = gameState.canvasContext;

  // Draw selection highlight
  const isSelected = gameState.selectedMonsterId === monster.id;
  if (isSelected) {
    ctx.save();
    ctx.strokeStyle = '#ff0000';
    ctx.lineWidth = 3;
    ctx.strokeRect(rect.x - 1, rect.y - 1, rect.w + 2, rect.h + 2);
    ctx.restore();
  }

  const imageUrl = `${IMAGE_PATHS.MONSTER_BASE}${monster.type}.png`;

  const img = getCachedImage(
    imageUrl,
    gameState.monsterImageCache,
    () => scheduleRedraw(() => drawMain()),
    () => console.error('Failed to load monster image for', monster.type),
  );

  if (img) {
    drawMonsterImage(img, monster, rect);
  } else {
    drawMonsterFallback(monster, rect);
  }
}

/**
 * Draw monster image
 * @param {HTMLImageElement} img
 * @param {MonsterLite} monster
 * @param {Object} rect
 */
function drawMonsterImage(img, monster, rect) {
  const ctx = gameState.canvasContext;

  drawImageSafe(ctx, img, 0, 0, null, null, rect.x, rect.y, rect.w, rect.h);

  // Add health indicator if damaged
  if (monster.body < monster.maxBody) {
    ctx.fillStyle = 'red';
    ctx.font = '12px monospace';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(`${monster.body}/${monster.maxBody}`,
      rect.x + rect.w / 2, rect.y + rect.h - 10);
  }
}

/**
 * Draw monster fallback
 * @param {MonsterLite} monster
 * @param {Object} rect
 */
function drawMonsterFallback(monster, rect) {
  const ctx = gameState.canvasContext;
  const colors = MONSTER_COLORS[monster.type] || MONSTER_COLORS.default;

  // Fill the tile
  ctx.fillStyle = colors.fill;
  ctx.fillRect(rect.x + 2, rect.y + 2, rect.w - 4, rect.h - 4);

  // Draw border
  ctx.strokeStyle = colors.stroke;
  ctx.lineWidth = 2;
  ctx.strokeRect(rect.x + 1, rect.y + 1, rect.w - 2, rect.h - 2);

  // Add type indicator
  ctx.fillStyle = 'white';
  ctx.font = '10px monospace';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';

  let displayText = monster.type.charAt(0).toUpperCase();
  if (monster.type === 'dread_warrior') {
    displayText = 'DW';
  }

  ctx.fillText(displayText, rect.x + rect.w / 2, rect.y + rect.h / 2);

  // Add health indicator if damaged
  if (monster.body < monster.maxBody) {
    ctx.fillStyle = 'red';
    ctx.font = '8px monospace';
    ctx.fillText(`${monster.body}/${monster.maxBody}`,
      rect.x + rect.w / 2, rect.y + rect.h - 8);
  }
}

// Import the main draw function (will be defined in main app)
let drawMain;
export function setDrawMainFunction(fn) {
  drawMain = fn;
}