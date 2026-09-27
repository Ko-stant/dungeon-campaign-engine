/** What a click on the tracker board means in each mode. */
import type { Edge, TileCoord } from '../board/geometry.ts';
import type { Command, SessionState } from './types.ts';

export type Mode = { kind: 'select' } | { kind: 'reveal' } | { kind: 'hide' } | { kind: 'addMonster'; monsterType: string };

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
  return s.monsters.find((m) => m.alive && at(m.x, m.y, t))?.id ?? null;
}

export function doorAt(s: SessionState, e: Edge): string | null {
  return s.quest.doors.find((d) => d.edge.x === e.x && d.edge.y === e.y && d.edge.orientation === e.orientation)?.id ?? null;
}

function isMovable(s: SessionState, id: string | null): id is string {
  return id !== null && (s.heroes.some((h) => h.id === id) || s.monsters.some((m) => m.id === id));
}

export function clickCommand(s: SessionState, mode: Mode, selectedId: string | null, target: ClickTarget): ClickResult | null {
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
      return { command: { type: 'area.reveal', payload: { x: t.x, y: t.y } }, select: selectedId };
    case 'hide':
      return { command: { type: 'tiles.hide', payload: { tiles: [{ x: t.x, y: t.y }] } }, select: selectedId };
    case 'addMonster':
      return { command: { type: 'monster.add', payload: { type: mode.monsterType, x: t.x, y: t.y } }, select: null };
    case 'select': {
      const piece = pieceAt(s, t);
      if (piece) {
        return { command: null, select: piece };
      }
      if (isMovable(s, selectedId)) {
        return { command: { type: 'move', payload: { id: selectedId, x: t.x, y: t.y } }, select: selectedId };
      }
      const trap = s.quest.traps.find((tr) => at(tr.x, tr.y, t));
      const note = s.quest.notes.find((n) => at(n.x, n.y, t) && !s.consumedNotes.includes(n.id));
      const removed = new Set(s.removedBlocks ?? []);
      const block = s.quest.blockedSquares.find((r) => !removed.has(r.id) && t.x >= r.x && t.x < r.x + r.w && t.y >= r.y && t.y < r.y + r.h);
      return { command: null, select: trap?.id ?? note?.id ?? block?.id ?? null };
    }
  }
}
