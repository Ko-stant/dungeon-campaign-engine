/**
 * The board model shared by the renderer, the map editor and the tracker.
 *
 * A board is a cols x rows grid counted from (1, 1) at the bottom-left. Each
 * tile holds a region id (row-major, bottom row first):
 *   VOID (-1)      solid rock / outside the dungeon
 *   CORRIDOR (0)   open corridor
 *   1, 2, 3 ...    a room
 * Walls are never stored: one exists wherever two neighbouring tiles belong to
 * different regions, treating anything off the board as VOID.
 */
import type { Edge, Rotation, TileCoord } from './geometry.ts';

export const VOID = -1;
export const CORRIDOR = 0;

export type DoorKind = 'normal' | 'secret';
export type DoorState = 'open' | 'closed';
export type TrapState = 'hidden' | 'revealed' | 'triggered' | 'disarmed';

export interface DoorView {
  id: string;
  edge: Edge;
  kind: DoorKind;
  state: DoorState;
}

/** A rectangle of impassable squares (rubble / blocked-square tiles), anchored bottom-left. */
export interface BlockedSquareView {
  x: number;
  y: number;
  w: number;
  h: number;
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
}

export interface PieceView {
  id: string;
  type: string;
  at: TileCoord;
  label?: string;
  image?: string;
  /** Drawn faded, e.g. a monster the heroes have not seen yet. */
  dim?: boolean;
}

export interface TrapView {
  id: string;
  kind: string;
  at: TileCoord;
  state: TrapState;
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
 * Every wall edge on the board: vertical edges row by row, then horizontal
 * edges row by row. A wall sits wherever the regions on either side differ.
 */
export function deriveWalls(cols: number, rows: number, regions: readonly number[]): Edge[] {
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
  return walls;
}
