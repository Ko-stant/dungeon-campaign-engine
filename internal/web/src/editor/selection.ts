/**
 * What the Select / move tool picks. Several quest items can share a square
 * (a chest marker under a note on some furniture); the editor lists them all
 * and the GM chooses which one a drag on the board moves. Pure.
 */
import { doorCovers, type Edge, type TileCoord } from '../board/geometry.ts';
import type { Catalog, QuestDoc } from '../maps/types.ts';
import { itemsAt } from './model.ts';

/** The items on a square, topmost first, then the door under the pointer (if any). */
export function squareStack(q: QuestDoc, catalog: Catalog, tile: TileCoord | null, edge: Edge | null): string[] {
  const ids = tile ? itemsAt(q, catalog, tile) : [];
  const door = edge ? q.doors.find((d) => doorCovers(d, edge)) : undefined;
  return door ? [...ids, door.id] : ids;
}

/**
 * Which item of a clicked square's stack is active (moved by a drag). Clicking
 * the square already shown keeps the item chosen in the panel; any other click
 * picks the topmost item.
 */
export function activeOnSquare(stack: readonly string[], tile: TileCoord, shown: TileCoord | null, current: string | null): string | null {
  const sameSquare = shown !== null && shown.x === tile.x && shown.y === tile.y;
  if (sameSquare && current !== null && stack.includes(current)) {
    return current;
  }
  return stack[0] ?? null;
}

/** A short kind and name for an item, for the panel's headings. */
export function describeItem(q: QuestDoc, catalog: Catalog, id: string): { kind: string; name: string } | null {
  const monster = q.monsters.find((m) => m.id === id);
  if (monster) {
    return { kind: 'Monster', name: catalog.monsters.find((d) => d.id === monster.type)?.name ?? monster.type };
  }
  const trap = q.traps.find((t) => t.id === id);
  if (trap) {
    const def = catalog.traps.find((d) => d.id === trap.kind);
    const name = def?.name ?? (trap.kind === 'trigger' ? 'Trigger' : `${trap.kind.replaceAll('_', ' ')} trap`);
    return { kind: 'Trap', name: trap.label ? `${name} "${trap.label}"` : name };
  }
  const note = q.notes.find((n) => n.id === id);
  if (note) {
    return { kind: 'Note', name: note.label };
  }
  const teleport = (q.teleports ?? []).find((t) => t.id === id);
  if (teleport) {
    return { kind: 'Teleport', name: teleport.label ?? 'square' };
  }
  const furniture = q.furniture.find((f) => f.id === id);
  if (furniture) {
    return { kind: 'Furniture', name: catalog.furniture.find((d) => d.id === furniture.type)?.name ?? furniture.type };
  }
  const blocked = q.blockedSquares.find((r) => r.id === id);
  if (blocked) {
    return { kind: 'Blocked', name: `${blocked.w}×${blocked.h} squares` };
  }
  const door = q.doors.find((d) => d.id === id);
  if (door) {
    return { kind: 'Door', name: door.kind };
  }
  return null;
}
