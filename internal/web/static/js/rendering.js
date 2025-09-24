/**
 * Core canvas rendering utilities
 */

import { calculateGridMetrics, calculateTileRect, calculateCanvasSize } from './geometry.js';
import { gameState } from './gameState.js';

/**
 * Resize canvas to match device pixel ratio
 * @param {HTMLCanvasElement} canvas
 * @param {CanvasRenderingContext2D} ctx
 */
export function resizeCanvas(canvas, ctx) {
  const { width, height, dpr } = calculateCanvasSize(canvas);
  canvas.width = width;
  canvas.height = height;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
}

/**
 * Get current grid metrics
 * @returns {Object}
 */
export function getGridMetrics() {
  const snapshot = gameState.snapshot;
  return calculateGridMetrics(
    gameState.canvas,
    snapshot?.mapWidth ?? 26,
    snapshot?.mapHeight ?? 19,
  );
}

/**
 * Get tile rectangle for given coordinates
 * @param {number} x
 * @param {number} y
 * @param {Object} metrics
 * @returns {Object}
 */
export function getTileRect(x, y, metrics = null) {
  const m = metrics || getGridMetrics();
  return calculateTileRect(x, y, m.tile, m.ox, m.oy);
}

/**
 * Get CSS color value
 * @param {string} property - CSS custom property name
 * @returns {string}
 */
export function getCSSColor(property) {
  const cs = getComputedStyle(document.documentElement);
  return cs.getPropertyValue(property).trim();
}

/**
 * Draw subtle per-cell grid across the whole board
 */
export function drawGrid() {
  const m = getGridMetrics();
  const ctx = gameState.canvasContext;
  const borderRGB = getCSSColor('--color-border');

  ctx.save();
  ctx.strokeStyle = `rgb(${borderRGB})`;
  ctx.lineWidth = 1;
  ctx.globalAlpha = 0.18;

  // Vertical grid lines
  ctx.beginPath();
  for (let x = 0; x <= m.cols; x++) {
    const xLine = m.ox + x * m.tile + 0.5;
    ctx.moveTo(xLine, m.oy);
    ctx.lineTo(xLine, m.oy + m.tile * m.rows);
  }
  ctx.stroke();

  // Horizontal grid lines
  ctx.beginPath();
  for (let y = 0; y <= m.rows; y++) {
    const yLine = m.oy + y * m.tile + 0.5;
    ctx.moveTo(m.ox, yLine);
    ctx.lineTo(m.ox + m.tile * m.cols, yLine);
  }
  ctx.stroke();

  ctx.restore();
}

/**
 * Draw region borders where neighbor tiles belong to different regions
 */
export function drawRegionBorders() {
  const m = getGridMetrics();
  const ctx = gameState.canvasContext;
  const snapshot = gameState.snapshot;
  const ids = snapshot?.tileRegionIds;

  if (!Array.isArray(ids)) {
    return;
  }

  const brandRGB = getCSSColor('--color-brand');

  ctx.save();
  ctx.strokeStyle = `rgb(${brandRGB})`;
  ctx.lineWidth = 1;
  ctx.globalAlpha = 0.85;

  // Vertical borders: between (x,y) and (x+1,y)
  for (let y = 0; y < m.rows; y++) {
    for (let x = 0; x < m.cols - 1; x++) {
      const a = ids[y * m.cols + x];
      const b = ids[y * m.cols + (x + 1)];
      if (a !== b) {
        const roomSideA = (a !== gameState.corridorRegionId) ? a : null;
        const roomSideB = (b !== gameState.corridorRegionId) ? b : null;

        const show =
          (roomSideA !== null && (gameState.knownRegions.has(roomSideA) || gameState.revealedRegions.has(roomSideA))) ||
          (roomSideB !== null && (gameState.knownRegions.has(roomSideB) || gameState.revealedRegions.has(roomSideB)));

        if (show) {
          const r = getTileRect(x, y, m);
          const xLine = r.x + r.w + 0.5;
          ctx.beginPath();
          ctx.moveTo(xLine, r.y);
          ctx.lineTo(xLine, r.y + r.h);
          ctx.stroke();
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
        const roomSideA = (a !== gameState.corridorRegionId) ? a : null;
        const roomSideB = (b !== gameState.corridorRegionId) ? b : null;

        const show =
          (roomSideA !== null && (gameState.knownRegions.has(roomSideA) || gameState.revealedRegions.has(roomSideA))) ||
          (roomSideB !== null && (gameState.knownRegions.has(roomSideB) || gameState.revealedRegions.has(roomSideB)));

        if (show) {
          const r = getTileRect(x, y, m);
          const yLine = r.y + r.h + 0.5;
          ctx.beginPath();
          ctx.moveTo(r.x, yLine);
          ctx.lineTo(r.x + r.w, yLine);
          ctx.stroke();
        }
      }
    }
  }

  ctx.restore();
}

/**
 * Draw background tiles based on region visibility
 */
export function drawBackground() {
  const m = getGridMetrics();
  const ctx = gameState.canvasContext;
  const snapshot = gameState.snapshot;

  const colorSurface = getCSSColor('--color-surface');
  const colorSurface2 = getCSSColor('--color-surface-2');

  ctx.clearRect(0, 0, m.width, m.height);
  ctx.fillStyle = `rgb(${colorSurface})`;
  ctx.fillRect(0, 0, m.width, m.height);

  if (Array.isArray(snapshot?.tileRegionIds)) {
    for (let y = 0; y < m.rows; y++) {
      for (let x = 0; x < m.cols; x++) {
        const idx = y * m.cols + x;
        const rid = snapshot.tileRegionIds[idx];
        const visible = gameState.revealedRegions.has(rid);
        ctx.fillStyle = visible ? `rgb(${colorSurface2})` : `rgb(${colorSurface})`;
        const r = getTileRect(x, y, m);
        ctx.fillRect(r.x, r.y, r.w, r.h);
      }
    }
  }
}

/**
 * Draw entities (heroes)
 */
export function drawEntities() {
  const m = getGridMetrics();
  const ctx = gameState.canvasContext;
  const accentRGB = getCSSColor('--color-accent');

  for (const [, t] of gameState.entityPositions.entries()) {
    if (!t) {
      continue;
    }

    const r = getTileRect(t.x, t.y, m);
    ctx.beginPath();
    ctx.arc(r.cx, r.cy, Math.max(2, Math.floor(m.tile * 0.35)), 0, Math.PI * 2);
    ctx.closePath();
    ctx.fillStyle = `rgb(${accentRGB})`;
    ctx.fill();
  }
}

/**
 * Set up canvas rotation transform
 * @param {CanvasRenderingContext2D} ctx
 * @param {number} centerPixelX
 * @param {number} centerPixelY
 * @param {number} rotationDegrees
 */
export function setupRotationTransform(ctx, centerPixelX, centerPixelY, rotationDegrees) {
  ctx.save();
  ctx.translate(centerPixelX, centerPixelY);
  ctx.rotate((rotationDegrees * Math.PI) / 180);
  ctx.translate(-centerPixelX, -centerPixelY);
}

/**
 * Restore canvas transform
 * @param {CanvasRenderingContext2D} ctx
 */
export function restoreTransform(ctx) {
  ctx.restore();
}

/**
 * Draw image with error handling
 * @param {CanvasRenderingContext2D} ctx
 * @param {HTMLImageElement} img
 * @param {number} srcX
 * @param {number} srcY
 * @param {number} srcWidth
 * @param {number} srcHeight
 * @param {number} destX
 * @param {number} destY
 * @param {number} destWidth
 * @param {number} destHeight
 */
export function drawImageSafe(ctx, img, srcX = 0, srcY = 0, srcWidth = null, srcHeight = null, destX, destY, destWidth, destHeight) {
  if (!img || !img.complete || img.naturalWidth === 0) {
    return false;
  }

  try {
    if (srcWidth !== null && srcHeight !== null) {
      ctx.drawImage(img, srcX, srcY, srcWidth, srcHeight, destX, destY, destWidth, destHeight);
    } else {
      ctx.drawImage(img, destX, destY, destWidth, destHeight);
    }
    return true;
  } catch (error) {
    console.warn('Failed to draw image:', error);
    return false;
  }
}

/**
 * Create image loading promise
 * @param {string} src
 * @returns {Promise<HTMLImageElement>}
 */
export function loadImage(src) {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error(`Failed to load image: ${src}`));
    img.src = src;
  });
}

/**
 * Cache and load image
 * @param {string} url
 * @param {Map} cache
 * @param {Function} onLoad
 * @param {Function} onError
 * @returns {HTMLImageElement|null}
 */
export function getCachedImage(url, cache, onLoad = null, onError = null) {
  if (cache.has(url)) {
    const cachedImage = cache.get(url);
    if (cachedImage && cachedImage.complete && cachedImage.naturalWidth > 0) {
      return cachedImage;
    } else if (cachedImage === null) {
      // Image failed to load previously
      return null;
    }
  }

  // Load image if not cached
  const img = new Image();
  img.onload = function () {
    cache.set(url, img);
    if (onLoad) {
      onLoad(img);
    }
  };
  img.onerror = function () {
    cache.set(url, null);
    if (onError) {
      onError();
    }
  };
  img.src = url;

  return null;
}

/**
 * Schedule redraw on next animation frame
 * @param {Function} drawFunction
 */
export function scheduleRedraw(drawFunction) {
  requestAnimationFrame(drawFunction);
}