/**
 * Pure grid geometry for boards of any size. No DOM or canvas access, so every
 * function here is unit-testable.
 *
 * Coordinate conventions (shared with the Go server):
 * - Squares count from (1, 1) at the BOTTOM-LEFT: column x runs 1..cols left
 *   to right, row y runs 1..rows bottom to top.
 * - A vertical edge (x, y) is the LEFT side of tile (x, y); x ranges 1..cols+1.
 * - A horizontal edge (x, y) is the BOTTOM side of tile (x, y); y ranges 1..rows+1.
 * - Blocks (furniture, blocked squares) are anchored at their bottom-left
 *   square and extend right and up.
 *
 * Only the functions that take GridMetrics deal in screen pixels, where y grows
 * downwards; they are the one place rows are flipped.
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

/** Default fraction of a tile within which a point counts as "on" a grid line. */
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

/** Pixel x of the vertical grid line on the left side of column x. */
function lineX(m: GridMetrics, x: number): number {
  return m.originX + (x - 1) * m.tile;
}

/** Pixel y of the horizontal grid line on the top side of row y. */
function topOfRow(m: GridMetrics, y: number): number {
  return m.originY + (m.rows - y) * m.tile;
}

export function tileRect(m: GridMetrics, t: TileCoord): TileRect {
  return footprintRect(m, t, 1, 1);
}

/** Pixel rectangle of a w x h block of squares anchored at its bottom-left square. */
export function footprintRect(m: GridMetrics, at: TileCoord, w: number, h: number): TileRect {
  const x = lineX(m, at.x);
  const y = topOfRow(m, at.y + h - 1);
  const pw = w * m.tile;
  const ph = h * m.tile;
  return { x, y, w: pw, h: ph, cx: x + pw / 2, cy: y + ph / 2 };
}

/** The tile under a pixel, or null outside the grid (right/bottom edges exclusive). */
export function pixelToTile(m: GridMetrics, px: number, py: number): TileCoord | null {
  const gx = (px - m.originX) / m.tile;
  const gy = (py - m.originY) / m.tile;
  if (gx < 0 || gy < 0 || gx >= m.cols || gy >= m.rows) {
    return null;
  }
  return { x: Math.floor(gx) + 1, y: m.rows - Math.floor(gy) };
}

/**
 * The tile edge nearest a pixel, if the pixel lies within a fraction of a tile
 * of a grid line. Includes the outer boundary; use isInteriorEdge to exclude it.
 */
export function pixelToEdge(m: GridMetrics, px: number, py: number, hitFraction = EDGE_HIT_FRACTION): Edge | null {
  const gx = (px - m.originX) / m.tile;
  const gy = (py - m.originY) / m.tile;
  if (gx < 0 || gy < 0 || gx > m.cols || gy > m.rows) {
    return null;
  }

  // Nearest grid lines, counted in screen order from the top-left corner.
  const nearCol = Math.round(gx);
  const nearRow = Math.round(gy);
  const distX = Math.abs(gx - nearCol);
  const distY = Math.abs(gy - nearRow);
  if (Math.min(distX, distY) > hitFraction) {
    return null;
  }

  // Clamp the along-line coordinate so a point exactly on the far boundary
  // still maps to the last tile.
  const col = Math.min(Math.floor(gx), m.cols - 1) + 1;
  const row = m.rows - Math.min(Math.floor(gy), m.rows - 1);
  if (distX <= distY) {
    return { x: nearCol + 1, y: row, orientation: 'vertical' };
  }
  // Screen line k (from the top) is the bottom side of row rows - k + 1.
  return { x: col, y: m.rows - nearRow + 1, orientation: 'horizontal' };
}

/** Pixel end points of the grid line under an edge (top-to-bottom or left-to-right). */
export function edgeSegment(m: GridMetrics, e: Edge): { x1: number; y1: number; x2: number; y2: number } {
  if (e.orientation === 'vertical') {
    const x = lineX(m, e.x);
    const y = topOfRow(m, e.y);
    return { x1: x, y1: y, x2: x, y2: y + m.tile };
  }
  const x = lineX(m, e.x);
  const y = topOfRow(m, e.y - 1);
  return { x1: x, y1: y, x2: x + m.tile, y2: y };
}

/** The two tiles an edge separates: [left or below, right or above]. May be off-board. */
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
    return e.x > 1 && e.x <= cols && e.y >= 1 && e.y <= rows;
  }
  return e.y > 1 && e.y <= rows && e.x >= 1 && e.x <= cols;
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

/** Pixel layout for previewing a piece before it is placed (see piecePreviewLayout). */
export interface PiecePreviewLayout {
  /** Pixels per square. */
  tile: number;
  /** The rotated footprint on screen. */
  box: { width: number; height: number };
  /** The image at its unrotated size, turned clockwise about the box centre, as on the board. */
  image: { width: number; height: number; degrees: number };
  /** The anchor square (the piece's bottom-left square once placed), in box pixels, y down. */
  anchor: { x: number; y: number; size: number };
}

/**
 * Lays out a piece preview the way the board draws it (furnitureDrawBox): the
 * footprint turned by rotation, the image drawn unrotated and turned about the
 * centre. Squares are at most 40px and shrink so the footprint fits maxPx.
 */
export function piecePreviewLayout(width: number, height: number, rotation: number, maxPx: number): PiecePreviewLayout {
  const size = rotatedFootprint(width, height, rotation);
  const tile = Math.min(40, Math.floor(maxPx / Math.max(size.width, size.height)));
  return {
    tile,
    box: { width: size.width * tile, height: size.height * tile },
    image: { width: width * tile, height: height * tile, degrees: rotation },
    anchor: { x: 0, y: (size.height - 1) * tile, size: tile },
  };
}

/** Every tile covered by a piece anchored at its bottom-left tile, row by row upwards. */
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
  /** Pixel centre of the rotated footprint. */
  cx: number;
  cy: number;
  /** Unrotated size in pixels; draw at this size after rotating by `radians` (clockwise on screen). */
  width: number;
  height: number;
  radians: number;
}

/**
 * Where to draw a piece's artwork: rotate about the centre of its (rotated)
 * footprint and draw the unrotated image centred there.
 */
export function furnitureDrawBox(m: GridMetrics, at: TileCoord, width: number, height: number, rotation: number): DrawBox {
  const size = rotatedFootprint(width, height, rotation);
  const r = footprintRect(m, at, size.width, size.height);
  return {
    cx: r.cx,
    cy: r.cy,
    width: width * m.tile,
    height: height * m.tile,
    radians: (rotation * Math.PI) / 180,
  };
}

/**
 * The edges a door covers: its own, then (for span 2) the next edge along the
 * wall, to the right for a horizontal edge and upward for a vertical one.
 */
export function doorEdges(edge: Edge, span: number | undefined): Edge[] {
  const out: Edge[] = [];
  for (let i = 0; i < Math.max(1, span ?? 1); i++) {
    out.push(edge.orientation === 'vertical' ? { x: edge.x, y: edge.y + i, orientation: edge.orientation } : { x: edge.x + i, y: edge.y, orientation: edge.orientation });
  }
  return out;
}

/** Whether a door (either half of a two-wide one) is on edge e. */
export function doorCovers(door: { edge: Edge; span?: number | undefined }, e: Edge): boolean {
  return doorEdges(door.edge, door.span).some((d) => d.x === e.x && d.y === e.y && d.orientation === e.orientation);
}

/** Fraction of a tile used for a door marker's thickness and its inset from tile corners. */
const DOOR_THICKNESS = 0.2;
const DOOR_INSET = 0.15;

/** Pixel rectangle of the marker drawn over a door edge. */
export function doorRect(m: GridMetrics, e: Edge): { x: number; y: number; w: number; h: number } {
  const thickness = m.tile * DOOR_THICKNESS;
  const inset = m.tile * DOOR_INSET;
  const length = m.tile - 2 * inset;
  const s = edgeSegment(m, e);
  if (e.orientation === 'vertical') {
    return { x: s.x1 - thickness / 2, y: s.y1 + inset, w: thickness, h: length };
  }
  return { x: s.x1 + inset, y: s.y1 - thickness / 2, w: length, h: thickness };
}

/** Pixel rectangle of a door marker across every edge it covers (one bar for a two-wide door). */
export function doorSpanRect(m: GridMetrics, edge: Edge, span: number | undefined): { x: number; y: number; w: number; h: number } {
  const rects = doorEdges(edge, span).map((e) => doorRect(m, e));
  const x = Math.min(...rects.map((r) => r.x));
  const y = Math.min(...rects.map((r) => r.y));
  return { x, y, w: Math.max(...rects.map((r) => r.x + r.w)) - x, h: Math.max(...rects.map((r) => r.y + r.h)) - y };
}
