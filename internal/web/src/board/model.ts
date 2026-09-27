/**
 * The board model shared by the renderer, the map editor and the tracker.
 *
 * A board is a cols x rows grid. Each tile holds a region id (row-major):
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

/** A rectangle of impassable squares (rubble / blocked-square tiles). */
export interface BlockedSquareView {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface FurnitureView {
  id: string;
  type: string;
  /** Top-left tile of the rotated footprint. */
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
}

export interface TrapView {
  id: string;
  kind: string;
  at: TileCoord;
  state: TrapState;
}

export interface BoardView {
  cols: number;
  rows: number;
  /** Row-major region ids, length cols * rows. */
  regions: readonly number[];
  doors: readonly DoorView[];
  blockedSquares: readonly BlockedSquareView[];
  furniture: readonly FurnitureView[];
  monsters: readonly PieceView[];
  heroes: readonly PieceView[];
  traps: readonly TrapView[];
  /** Discovered tile indexes. Omitted means everything is shown (GM view). */
  discovered?: ReadonlySet<number>;
}

export function tileIndex(cols: number, t: TileCoord): number {
  return t.y * cols + t.x;
}

/** Region id of a tile; anything off the board is VOID. */
export function regionAt(cols: number, rows: number, regions: readonly number[], t: TileCoord): number {
  if (t.x < 0 || t.y < 0 || t.x >= cols || t.y >= rows) {
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

  for (let y = 0; y < rows; y++) {
    for (let x = 0; x <= cols; x++) {
      if (at(x - 1, y) !== at(x, y)) {
        walls.push({ x, y, orientation: 'vertical' });
      }
    }
  }
  for (let y = 0; y <= rows; y++) {
    for (let x = 0; x < cols; x++) {
      if (at(x, y - 1) !== at(x, y)) {
        walls.push({ x, y, orientation: 'horizontal' });
      }
    }
  }
  return walls;
}
