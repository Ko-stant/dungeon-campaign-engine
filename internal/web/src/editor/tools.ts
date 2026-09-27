/**
 * What each editor tool does to the documents. Pure: the page turns pointer
 * events into clicks (a tile and/or edge) or drags (the tiles visited) and
 * pushes the returned document onto the undo history.
 */
import type { Edge, Rotation, TileCoord } from '../board/geometry.ts';
import type { DoorKind } from '../board/model.ts';
import type { BoardDoc, Catalog, QuestDoc } from '../maps/types.ts';
import {
  addBlockedSquare,

  itemsAt,
  paintTiles,
  placeDoor,
  placeFurniture,
  placeMonster,
  placeNote,
  placeTrap,
  rectTiles,
  removeItem,
  toggleExitTile,
  toggleStartTile,
  toggleWall,
} from './model.ts';

export interface EditorDoc {
  board: BoardDoc;
  quest: QuestDoc | null;
}

export type Tool =
  | { kind: 'paint'; region: number }
  | { kind: 'fill'; region: number }
  | { kind: 'wall' }
  | { kind: 'door'; doorKind: DoorKind; locked: boolean }
  | { kind: 'blocked' }
  | { kind: 'furniture'; type: string; rotation: Rotation }
  | { kind: 'monster'; type: string }
  | { kind: 'trap'; trapKind: string }
  | { kind: 'note' }
  | { kind: 'start' }
  | { kind: 'exit' }
  | { kind: 'select' }
  | { kind: 'erase' };

/** Tools that act on a drag rather than a single click. */
export function isDragTool(tool: Tool): boolean {
  return tool.kind === 'paint' || tool.kind === 'fill' || tool.kind === 'blocked';
}

/** Tools that need an open quest. */
export function isQuestTool(tool: Tool): boolean {
  return tool.kind !== 'paint' && tool.kind !== 'fill' && tool.kind !== 'wall';
}

export interface ClickTarget {
  tile: TileCoord | null;
  edge: Edge | null;
}

function withQuest(d: EditorDoc, quest: QuestDoc): EditorDoc {
  return quest === d.quest ? d : { ...d, quest };
}

/** Applies a drag over `points` (in visiting order). */
export function applyDrag(d: EditorDoc, tool: Tool, points: readonly TileCoord[]): EditorDoc {
  const first = points[0];
  const last = points[points.length - 1];
  if (!first || !last) {
    return d;
  }
  switch (tool.kind) {
    case 'paint':
      return { ...d, board: paintTiles(d.board, points, tool.region) };
    case 'fill':
      return { ...d, board: paintTiles(d.board, rectTiles(first, last), tool.region) };
    case 'blocked': {
      if (!d.quest) {
        return d;
      }
      const x = Math.min(first.x, last.x);
      const y = Math.min(first.y, last.y);
      return withQuest(d, addBlockedSquare(d.quest, { x, y, w: Math.abs(last.x - first.x) + 1, h: Math.abs(last.y - first.y) + 1 }));
    }
    default:
      return d;
  }
}

/** Applies a single click. */
export function applyClick(d: EditorDoc, tool: Tool, target: ClickTarget, catalog: Catalog): EditorDoc {
  if (tool.kind === 'paint' || tool.kind === 'fill') {
    return target.tile ? applyDrag(d, tool, [target.tile]) : d;
  }
  if (tool.kind === 'wall') {
    if (!target.edge) {
      return d;
    }
    const board = toggleWall(d.board, target.edge);
    return board === d.board ? d : { ...d, board };
  }
  const q = d.quest;
  if (!q) {
    return d;
  }
  if (tool.kind === 'door') {
    return target.edge ? withQuest(d, placeDoor(q, target.edge, tool.doorKind, tool.locked)) : d;
  }
  if (tool.kind === 'erase' && target.edge) {
    const e = target.edge;
    const door = q.doors.find((dr) => dr.edge.x === e.x && dr.edge.y === e.y && dr.edge.orientation === e.orientation);
    return door ? withQuest(d, removeItem(q, door.id)) : d;
  }

  const t = target.tile;
  if (!t) {
    return d;
  }
  switch (tool.kind) {
    case 'furniture':
      return withQuest(d, placeFurniture(q, tool.type, t, tool.rotation));
    case 'monster':
      return withQuest(d, placeMonster(q, tool.type, t));
    case 'trap':
      return withQuest(d, placeTrap(q, tool.trapKind, t));
    case 'note':
      return withQuest(d, placeNote(q, t, ''));
    case 'start':
      return withQuest(d, toggleStartTile(q, t));
    case 'exit':
      return withQuest(d, toggleExitTile(q, t));
    case 'erase': {
      const top = itemsAt(q, catalog, t)[0];
      return top ? withQuest(d, removeItem(q, top)) : d;
    }
    default:
      return d;
  }
}

/**
 * Tiles from a to b inclusive, stepping one orthogonal neighbour at a time so
 * a fast brush stroke leaves no gaps (including diagonal ones).
 */
export function lineTiles(a: TileCoord, b: TileCoord): TileCoord[] {
  const tiles: TileCoord[] = [{ x: a.x, y: a.y }];
  let { x, y } = a;
  const dx = Math.abs(b.x - a.x);
  const dy = Math.abs(b.y - a.y);
  const sx = Math.sign(b.x - a.x);
  const sy = Math.sign(b.y - a.y);
  let err = dx - dy;
  while (x !== b.x || y !== b.y) {
    // Move along whichever axis keeps the path closest to the ideal line.
    if (2 * err > -dy && x !== b.x && (2 * err >= dx || y === b.y)) {
      err -= dy;
      x += sx;
    } else {
      err += dx;
      y += sy;
    }
    tiles.push({ x, y });
  }
  return tiles;
}
