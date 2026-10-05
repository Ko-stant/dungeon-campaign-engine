/** What a click on the tracker board means in each mode. */
import { doorCovers, footprintTiles, type Edge, type TileCoord } from '../board/geometry.ts';
import { covers, tileIndex } from '../board/model.ts';
import { trapTiles } from '../editor/model.ts';
import type { Catalog, TrapDoc } from '../maps/types.ts';
import type { Command, SessionState } from './types.ts';

/**
 * What board clicks do. Reveal (a room, one corridor square, or the corridor
 * squares along a drag: see revealPathCommand) and picked squares can also
 * mark the monsters there as seen. In 'pickSquares' the page
 * collects squares (see paintPending) and reveals them with one command.
 */
export type Mode =
  | { kind: 'select' }
  | { kind: 'reveal'; seen?: boolean }
  | { kind: 'pickSquares'; seen?: boolean }
  | { kind: 'hide' }
  | { kind: 'block' }
  | { kind: 'addMonster'; monsterType: string };

/** The mode bar's keyboard shortcuts: 1 Select / move (and clear the selection), 2 Reveal, 3 Hide. */
export function hotkey(
  ev: { key: string; ctrlKey: boolean; metaKey: boolean; altKey: boolean },
  mode: Mode,
  revealSeen: boolean,
): { mode: Mode; deselect: boolean } | null {
  if (ev.ctrlKey || ev.metaKey || ev.altKey) {
    return null;
  }
  switch (ev.key) {
    case '1':
      return { mode: { kind: 'select' }, deselect: true };
    case '2':
      return { mode: mode.kind === 'pickSquares' ? mode : { kind: 'reveal', seen: revealSeen }, deselect: false };
    case '3':
      return { mode: { kind: 'hide' }, deselect: false };
    default:
      return null;
  }
}

export interface ClickTarget {
  tile: TileCoord | null;
  edge: Edge | null;
}

export interface ClickResult {
  command: Command | null;
  /** The id to select after the click (null clears the selection). */
  select: string | null;
}

const at = (x: number, y: number, t: TileCoord): boolean => x === t.x && y === t.y;

/** The hero or living monster on a square, heroes first. */
export function pieceAt(s: SessionState, t: TileCoord): string | null {
  const hero = s.heroes.find((h) => h.placed && at(h.x, h.y, t));
  if (hero) {
    return hero.id;
  }
  return s.monsters.find((m) => m.alive && covers(m.x, m.y, m.width, m.height, t))?.id ?? null;
}

export function doorAt(s: SessionState, e: Edge): string | null {
  return s.quest.doors.find((d) => doorCovers(d, e))?.id ?? null;
}

/** Quest traps still on the board, where they are now (moved during play or as placed). */
function boardTraps(s: SessionState): { doc: TrapDoc; at: TileCoord }[] {
  const live = new Map(s.traps.map((t) => [t.id, t]));
  return s.quest.traps.flatMap((doc) => {
    const l = live.get(doc.id);
    return l?.state === 'removed' ? [] : [{ doc, at: l?.at ?? { x: doc.x, y: doc.y } }];
  });
}

/** The trap on a square: with the catalog, anywhere on a multi-square trap's footprint. */
function trapAt(s: SessionState, t: TileCoord, catalog?: Catalog): string | null {
  const hit = boardTraps(s).find(({ doc, at: pos }) =>
    catalog ? trapTiles(catalog, { kind: doc.kind, x: pos.x, y: pos.y, rotation: doc.rotation ?? 0 }).some((tt) => at(tt.x, tt.y, t)) : at(pos.x, pos.y, t));
  return hit?.doc.id ?? null;
}

/** The furniture on a square: with the catalog, anywhere on its rotated footprint. */
function furnitureAt(s: SessionState, t: TileCoord, catalog?: Catalog): string | null {
  const hit = s.quest.furniture.find((f) => {
    const def = catalog?.furniture.find((d) => d.id === f.type);
    return footprintTiles({ x: f.x, y: f.y }, def?.width ?? 1, def?.height ?? 1, f.rotation).some((ft) => at(ft.x, ft.y, t));
  });
  return hit?.id ?? null;
}

function isMovable(s: SessionState, id: string | null, catalog?: Catalog): id is string {
  if (id === null) {
    return false;
  }
  if (s.heroes.some((h) => h.id === id) || s.monsters.some((m) => m.id === id)) {
    return true;
  }
  // Traps move only when the catalog says so (a rolling boulder), so a stray click never drags a pit around.
  const trap = boardTraps(s).find(({ doc }) => doc.id === id);
  return trap !== undefined && (catalog?.traps.find((d) => d.id === trap.doc.kind)?.movable ?? false);
}

/** Adds (add=true) or removes a square from the picked set, without duplicates. */
export function paintPending(pending: readonly TileCoord[], t: TileCoord, add: boolean): TileCoord[] {
  const rest = pending.filter((p) => !at(p.x, p.y, t));
  return add ? [...rest, { x: t.x, y: t.y }].sort((a, b) => a.y - b.y || a.x - b.x) : rest;
}

/** One command revealing every picked square (and, with seen, the monsters on them). */
export function revealSquaresCommand(pending: readonly TileCoord[], seen: boolean): Command | null {
  if (pending.length === 0) {
    return null;
  }
  const tiles = pending.map((p) => ({ x: p.x, y: p.y }));
  return { type: 'tiles.reveal', payload: seen ? { tiles, seen: true } : { tiles } };
}

/**
 * The corridor squares along a reveal drag that the heroes have not discovered
 * yet, in path order without repeats. Room squares and solid rock are skipped:
 * a click reveals a whole room.
 */
export function corridorPath(s: SessionState, path: readonly TileCoord[]): TileCoord[] {
  const { width, height, regions } = s.board;
  const done = new Set(s.discovered);
  const out: TileCoord[] = [];
  for (const t of path) {
    if (t.x < 1 || t.y < 1 || t.x > width || t.y > height) {
      continue;
    }
    const i = tileIndex(width, t);
    if (regions[i] === 0 && !done.has(i)) {
      done.add(i);
      out.push({ x: t.x, y: t.y });
    }
  }
  return out;
}

/** One command revealing the new corridor squares along a drag (null when there are none). */
export function revealPathCommand(s: SessionState, path: readonly TileCoord[], seen: boolean): Command | null {
  return revealSquaresCommand(corridorPath(s, path), seen);
}

/** The catalog (optional) gives multi-square traps their footprint; without it every trap is one square. */
export function clickCommand(s: SessionState, mode: Mode, selectedId: string | null, target: ClickTarget, catalog?: Catalog): ClickResult | null {
  if (mode.kind === 'select' && target.edge) {
    const doorId = doorAt(s, target.edge);
    if (doorId) {
      // Selecting only; the Selected panel opens, closes, locks or shows it.
      return { command: null, select: doorId };
    }
  }
  const t = target.tile;
  if (!t) {
    return null;
  }
  switch (mode.kind) {
    case 'reveal':
      return { command: { type: 'area.reveal', payload: mode.seen ? { x: t.x, y: t.y, seen: true } : { x: t.x, y: t.y } }, select: selectedId };
    case 'pickSquares':
      return null; // the page collects the squares
    case 'hide':
      return { command: { type: 'tiles.hide', payload: { tiles: [{ x: t.x, y: t.y }] } }, select: selectedId };
    case 'block':
      return { command: { type: 'block.add', payload: { x: t.x, y: t.y } }, select: null };
    case 'addMonster':
      return { command: { type: 'monster.add', payload: { type: mode.monsterType, x: t.x, y: t.y } }, select: null };
    case 'select': {
      const piece = pieceAt(s, t);
      if (piece) {
        return { command: null, select: piece };
      }
      if (isMovable(s, selectedId, catalog)) {
        return { command: { type: 'move', payload: { id: selectedId, x: t.x, y: t.y } }, select: selectedId };
      }
      const trap = trapAt(s, t, catalog);
      const note = s.quest.notes.find((n) => at(n.x, n.y, t) && !s.consumedNotes.includes(n.id));
      const removed = new Set(s.removedBlocks ?? []);
      const inside = (r: { x: number; y: number; w: number; h: number }): boolean => t.x >= r.x && t.x < r.x + r.w && t.y >= r.y && t.y < r.y + r.h;
      const block = (s.addedBlocks ?? []).find(inside) ?? s.quest.blockedSquares.find((r) => !removed.has(r.id) && inside(r));
      return { command: null, select: trap ?? note?.id ?? block?.id ?? furnitureAt(s, t, catalog) };
    }
  }
}
