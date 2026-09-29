/** What a click on the tracker board means in each mode. */
import { doorCovers, type Edge, type TileCoord } from '../board/geometry.ts';
import { covers } from '../board/model.ts';
import { trapTiles } from '../editor/model.ts';
import type { Catalog, TrapDoc } from '../maps/types.ts';
import type { Command, SessionState } from './types.ts';

/**
 * What board clicks do. Reveal (a room, or one corridor square) and picked
 * squares can also mark the monsters there as seen. In 'pickSquares' the page
 * collects squares (see paintPending) and reveals them with one command.
 */
export type Mode =
  | { kind: 'select' }
  | { kind: 'reveal'; seen?: boolean }
  | { kind: 'pickSquares'; seen?: boolean }
  | { kind: 'hide' }
  | { kind: 'addMonster'; monsterType: string };

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

/** The catalog (optional) gives multi-square traps their footprint; without it every trap is one square. */
export function clickCommand(s: SessionState, mode: Mode, selectedId: string | null, target: ClickTarget, catalog?: Catalog): ClickResult | null {
  if (mode.kind === 'select' && target.edge) {
    const doorId = doorAt(s, target.edge);
    if (doorId) {
      const open = s.doors.find((d) => d.id === doorId)?.state === 'open';
      return { command: { type: 'door.set', payload: { id: doorId, state: open ? 'closed' : 'open' } }, select: doorId };
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
      const block = s.quest.blockedSquares.find((r) => !removed.has(r.id) && t.x >= r.x && t.x < r.x + r.w && t.y >= r.y && t.y < r.y + r.h);
      return { command: null, select: trap ?? note?.id ?? block?.id ?? null };
    }
  }
}
