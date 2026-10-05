/**
 * The board model shared by the renderer, the map editor and the tracker.
 *
 * A board is a cols x rows grid counted from (1, 1) at the bottom-left. Each
 * tile holds a region id (row-major, bottom row first):
 *   VOID (-1)      solid rock / outside the dungeon
 *   CORRIDOR (0)   open corridor
 *   1, 2, 3 ...    a room
 * Walls are derived wherever two neighboring tiles belong to different
 * regions, treating anything off the board as VOID. The GM can also draw walls
 * on interior edges (e.g. between two stretches of corridor); those are the
 * only walls stored.
 */
import type { Edge, Rotation, TileCoord } from './geometry.ts';

export const VOID = -1;
export const CORRIDOR = 0;

/** An exit door leads off the map; it may sit on the board's edge. */
export type DoorKind = 'normal' | 'secret' | 'gate' | 'exit';
export type DoorState = 'open' | 'closed';
export type TrapState = 'hidden' | 'revealed' | 'triggered' | 'disarmed';

export interface DoorView {
  id: string;
  edge: Edge;
  kind: DoorKind;
  state: DoorState;
  locked?: boolean;
  /** 2 for a two-wide door (see doorEdges); missing means 1. */
  span?: number | undefined;
  /** Drawn faded (GM: not shown to the players yet). */
  dim?: boolean;
}

/** A rectangle of impassable squares (rubble / blocked-square tiles), anchored bottom-left. */
export interface BlockedSquareView {
  id: string;
  /** Hides a secret door (marked for the GM). */
  hiddenDoor?: boolean;
  x: number;
  y: number;
  w: number;
  h: number;
  /** Drawn faded (GM: not shown to the players yet). */
  dim?: boolean;
}

export interface FurnitureView {
  id: string;
  type: string;
  /** Bottom-left tile of the rotated footprint. */
  at: TileCoord;
  /** Unrotated size in tiles, from the furniture catalog. */
  width: number;
  height: number;
  rotation: Rotation;
  image?: string;
  /** Drawn faded (GM: not shown to the players yet). */
  dim?: boolean;
}

export interface PieceView {
  id: string;
  type: string;
  at: TileCoord;
  label?: string;
  image?: string;
  /** Drawn faded, e.g. a monster the heroes have not seen yet. */
  dim?: boolean;
  /** Footprint in squares from the bottom-left `at` (default 1x1). */
  width?: number;
  height?: number;
  /** Solid color for pieces without artwork (custom monsters). */
  color?: string;
}

/** Copies a piece's size (when bigger than one square) and color onto a view. */
export function withShape(view: PieceView, shape: { width?: number | undefined; height?: number | undefined; color?: string | undefined }): PieceView {
  const w = shape.width ?? 1;
  const h = shape.height ?? 1;
  if (w > 1 || h > 1) {
    view.width = w;
    view.height = h;
  }
  if (shape.color) {
    view.color = shape.color;
  }
  return view;
}

/**
 * True when square t lies on a piece of the given size anchored at (x, y).
 * A missing or 0 size is one square (sessions from before monster sizes store 0), as in Go.
 */
export function covers(x: number, y: number, width: number | undefined, height: number | undefined, t: TileCoord): boolean {
  return t.x >= x && t.x < x + Math.max(width ?? 1, 1) && t.y >= y && t.y < y + Math.max(height ?? 1, 1);
}

export interface TrapView {
  id: string;
  kind: string;
  at: TileCoord;
  state: TrapState;
  /** Catalog traps only: unrotated footprint, rotation and artwork. Markers are one square. */
  width?: number;
  height?: number;
  rotation?: Rotation;
  image?: string;
  /** Short GM label drawn on the trap, e.g. trigger "1". */
  label?: string;
}

export interface TeleportView {
  id: string;
  at: TileCoord;
  label?: string;
}

export interface NoteView {
  id: string;
  label: string;
  at: TileCoord;
}

export interface BoardView {
  cols: number;
  rows: number;
  /** Row-major region ids from the bottom row up, length cols * rows. */
  regions: readonly number[];
  /** Walls drawn by the GM, in addition to the derived ones. */
  drawnWalls?: readonly Edge[];
  doors: readonly DoorView[];
  blockedSquares: readonly BlockedSquareView[];
  furniture: readonly FurnitureView[];
  monsters: readonly PieceView[];
  heroes: readonly PieceView[];
  traps: readonly TrapView[];
  /** Lettered quest-note markers (editor / GM only). */
  notes?: readonly NoteView[];
  /** Hero start squares (editor / setup only). */
  startTiles?: readonly TileCoord[];
  /** Exit squares, where the heroes leave the dungeon. */
  exitTiles?: readonly TileCoord[];
  /** Teleport squares. */
  teleports?: readonly TeleportView[];
  /** Fill colors for rooms, by region id. */
  roomColors?: ReadonlyMap<number, string>;
  /** Squares drawn in the "seen" color (the player screen's discovered squares). */
  seenTiles?: ReadonlySet<number>;
  /** Discovered tile indexes. Omitted means everything is shown (GM view). */
  discovered?: ReadonlySet<number>;
}

/** Index into row-major regions of an on-board square. */
export function tileIndex(cols: number, t: TileCoord): number {
  return (t.y - 1) * cols + (t.x - 1);
}

/** The square at a regions index (the inverse of tileIndex). */
export function tileAt(cols: number, index: number): TileCoord {
  return { x: (index % cols) + 1, y: Math.floor(index / cols) + 1 };
}

export function onBoard(cols: number, rows: number, t: TileCoord): boolean {
  return t.x >= 1 && t.y >= 1 && t.x <= cols && t.y <= rows;
}

/** Region id of a tile; anything off the board is VOID. */
export function regionAt(cols: number, rows: number, regions: readonly number[], t: TileCoord): number {
  if (!onBoard(cols, rows, t)) {
    return VOID;
  }
  return regions[tileIndex(cols, t)] ?? VOID;
}

export function assertBoardShape(cols: number, rows: number, regions: readonly number[]): void {
  if (regions.length !== cols * rows) {
    throw new RangeError(`regions has ${regions.length} entries, expected ${cols * rows} for a ${cols}x${rows} board`);
  }
  for (const id of regions) {
    if (!Number.isInteger(id) || id < VOID) {
      throw new RangeError(`invalid region id ${id}`);
    }
  }
}

/**
 * Every wall edge on the board: derived vertical edges row by row, then derived
 * horizontal edges row by row (a wall sits wherever the regions on either side
 * differ), then any drawn walls that are not already derived.
 */
export function deriveWalls(cols: number, rows: number, regions: readonly number[], drawn: readonly Edge[] = []): Edge[] {
  const walls: Edge[] = [];
  const at = (x: number, y: number): number => regionAt(cols, rows, regions, { x, y });

  for (let y = 1; y <= rows; y++) {
    for (let x = 1; x <= cols + 1; x++) {
      if (at(x - 1, y) !== at(x, y)) {
        walls.push({ x, y, orientation: 'vertical' });
      }
    }
  }
  for (let y = 1; y <= rows + 1; y++) {
    for (let x = 1; x <= cols; x++) {
      if (at(x, y - 1) !== at(x, y)) {
        walls.push({ x, y, orientation: 'horizontal' });
      }
    }
  }
  for (const e of drawn) {
    const [a, b] = e.orientation === 'vertical' ? [at(e.x - 1, e.y), at(e.x, e.y)] : [at(e.x, e.y - 1), at(e.x, e.y)];
    if (a === b) {
      walls.push(e);
    }
  }
  return walls;
}
