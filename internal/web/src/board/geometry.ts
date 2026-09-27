/**
 * Pure grid geometry for boards of any size. No DOM or canvas access, so every
 * function here is unit-testable.
 *
 * Coordinate conventions (shared with the Go server):
 * - Tiles are addressed by column x (0..cols-1) and row y (0..rows-1).
 * - A vertical edge (x, y) is the LEFT side of tile (x, y); x ranges 0..cols.
 * - A horizontal edge (x, y) is the TOP side of tile (x, y); y ranges 0..rows.
 */

export interface TileCoord {
  x: number;
  y: number;
}

export type Orientation = 'vertical' | 'horizontal';

export interface Edge {
  x: number;
  y: number;
  orientation: Orientation;
}

export type Rotation = 0 | 90 | 180 | 270;

export interface GridMetrics {
  cols: number;
  rows: number;
  /** Tile size in CSS pixels. */
  tile: number;
  /** Pixel position of the grid's top-left corner inside the view. */
  originX: number;
  originY: number;
}

export interface TileRect {
  x: number;
  y: number;
  w: number;
  h: number;
  cx: number;
  cy: number;
}

/** Fraction of a tile within which a point counts as "on" a grid line. */
const EDGE_HIT_FRACTION = 0.2;

function assertBoardSize(cols: number, rows: number): void {
  if (!Number.isInteger(cols) || !Number.isInteger(rows) || cols < 1 || rows < 1) {
    throw new RangeError(`board size must be positive integers, got ${cols}x${rows}`);
  }
}

/**
 * Fits a cols x rows grid of square tiles into a view, centred, using whole-pixel
 * tile sizes so grid lines stay crisp.
 */
export function computeGridMetrics(viewWidth: number, viewHeight: number, cols: number, rows: number): GridMetrics {
  assertBoardSize(cols, rows);
  const tile = Math.max(1, Math.floor(Math.min(viewWidth / cols, viewHeight / rows)));
  return {
    cols,
    rows,
    tile,
    originX: Math.max(0, Math.floor((viewWidth - tile * cols) / 2)),
    originY: Math.max(0, Math.floor((viewHeight - tile * rows) / 2)),
  };
}

export function tileRect(m: GridMetrics, t: TileCoord): TileRect {
  const x = m.originX + t.x * m.tile;
  const y = m.originY + t.y * m.tile;
  return { x, y, w: m.tile, h: m.tile, cx: x + m.tile / 2, cy: y + m.tile / 2 };
}

/** The tile under a pixel, or null outside the grid (right/bottom edges exclusive). */
export function pixelToTile(m: GridMetrics, px: number, py: number): TileCoord | null {
  const gx = (px - m.originX) / m.tile;
  const gy = (py - m.originY) / m.tile;
  if (gx < 0 || gy < 0 || gx >= m.cols || gy >= m.rows) {
    return null;
  }
  return { x: Math.floor(gx), y: Math.floor(gy) };
}

/**
 * The tile edge nearest a pixel, if the pixel lies within a fraction of a tile
 * of a grid line. Includes the outer boundary; use isInteriorEdge to exclude it.
 */
export function pixelToEdge(m: GridMetrics, px: number, py: number): Edge | null {
  const gx = (px - m.originX) / m.tile;
  const gy = (py - m.originY) / m.tile;
  if (gx < 0 || gy < 0 || gx > m.cols || gy > m.rows) {
    return null;
  }

  const lineX = Math.round(gx);
  const lineY = Math.round(gy);
  const distX = Math.abs(gx - lineX);
  const distY = Math.abs(gy - lineY);
  if (Math.min(distX, distY) > EDGE_HIT_FRACTION) {
    return null;
  }

  // Clamp the along-line coordinate so a point exactly on the far boundary
  // still maps to the last tile.
  const col = Math.min(Math.floor(gx), m.cols - 1);
  const row = Math.min(Math.floor(gy), m.rows - 1);
  if (distX <= distY) {
    return { x: lineX, y: row, orientation: 'vertical' };
  }
  return { x: col, y: lineY, orientation: 'horizontal' };
}

/** The two tiles an edge separates: [left or up, right or down]. May be off-board. */
export function edgeTiles(e: Edge): [TileCoord, TileCoord] {
  if (e.orientation === 'vertical') {
    return [{ x: e.x - 1, y: e.y }, { x: e.x, y: e.y }];
  }
  return [{ x: e.x, y: e.y - 1 }, { x: e.x, y: e.y }];
}

/** The edge shared by two orthogonally adjacent tiles, or null. */
export function edgeBetween(a: TileCoord, b: TileCoord): Edge | null {
  const dx = b.x - a.x;
  const dy = b.y - a.y;
  if (Math.abs(dx) + Math.abs(dy) !== 1) {
    return null;
  }
  if (dx !== 0) {
    return { x: Math.max(a.x, b.x), y: a.y, orientation: 'vertical' };
  }
  return { x: a.x, y: Math.max(a.y, b.y), orientation: 'horizontal' };
}

/** True when the edge lies strictly inside a cols x rows board (both sides on-board). */
export function isInteriorEdge(e: Edge, cols: number, rows: number): boolean {
  if (e.orientation === 'vertical') {
    return e.x > 0 && e.x < cols && e.y >= 0 && e.y < rows;
  }
  return e.y > 0 && e.y < rows && e.x >= 0 && e.x < cols;
}

function assertRotation(rotation: number): asserts rotation is Rotation {
  if (rotation !== 0 && rotation !== 90 && rotation !== 180 && rotation !== 270) {
    throw new RangeError(`rotation must be 0, 90, 180 or 270, got ${rotation}`);
  }
}

/** A piece's footprint after rotation: quarter turns swap width and height. */
export function rotatedFootprint(width: number, height: number, rotation: number): { width: number; height: number } {
  assertRotation(rotation);
  return rotation === 90 || rotation === 270 ? { width: height, height: width } : { width, height };
}

/** Every tile covered by a piece anchored at its top-left tile, row by row. */
export function footprintTiles(origin: TileCoord, width: number, height: number, rotation: number): TileCoord[] {
  const size = rotatedFootprint(width, height, rotation);
  const tiles: TileCoord[] = [];
  for (let y = 0; y < size.height; y++) {
    for (let x = 0; x < size.width; x++) {
      tiles.push({ x: origin.x + x, y: origin.y + y });
    }
  }
  return tiles;
}

export interface DrawBox {
  /** Centre of the rotated footprint, in tile units. */
  centerX: number;
  centerY: number;
  /** Unrotated size in tiles; draw at this size after rotating by `radians`. */
  width: number;
  height: number;
  radians: number;
}

/**
 * Where to draw a piece's artwork: rotate about the centre of its (rotated)
 * footprint and draw the unrotated image centred there.
 */
export function furnitureDrawBox(at: TileCoord, width: number, height: number, rotation: number): DrawBox {
  const size = rotatedFootprint(width, height, rotation);
  return {
    centerX: at.x + size.width / 2,
    centerY: at.y + size.height / 2,
    width,
    height,
    radians: (rotation * Math.PI) / 180,
  };
}

/** Fraction of a tile used for a door marker's thickness and its inset from tile corners. */
const DOOR_THICKNESS = 0.2;
const DOOR_INSET = 0.15;

/** Pixel rectangle of the marker drawn over a door edge. */
export function doorRect(m: GridMetrics, e: Edge): { x: number; y: number; w: number; h: number } {
  const thickness = m.tile * DOOR_THICKNESS;
  const inset = m.tile * DOOR_INSET;
  const length = m.tile - 2 * inset;
  const lineX = m.originX + e.x * m.tile;
  const lineY = m.originY + e.y * m.tile;
  if (e.orientation === 'vertical') {
    return { x: lineX - thickness / 2, y: lineY + inset, w: thickness, h: length };
  }
  return { x: lineX + inset, y: lineY - thickness / 2, w: length, h: thickness };
}
