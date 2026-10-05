/**
 * Encounters for the combat simulator, read from a quest's board: the monsters of
 * each room fight together, and corridor monsters standing close together form a
 * group. Encounters are ordered by walking distance from the start squares
 * (through doors, secret ones included), as a first guess at the party's route.
 */

import { doorCovers, edgeBetween, type Edge, type TileCoord } from '../board/geometry.ts';
import { deriveWalls, regionAt } from '../board/model.ts';
import type { BoardDoc, QuestDoc } from '../maps/types.ts';

export interface Encounter {
  name: string;
  /** Monster types, in quest order. */
  monsters: string[];
  /** Fewest steps from a start square to one of the monsters; Infinity if unreachable. */
  distance: number;
}

/** Corridor monsters this many squares apart or closer (in any direction) fight together. */
const CORRIDOR_REACH = 3;

export function questEncounters(board: BoardDoc, quest: QuestDoc, skipTypes: readonly string[] = []): Encounter[] {
  const distances = walkDistances(board, quest);
  const at = (t: TileCoord): number => regionAt(board.width, board.height, board.regions, t);
  const monsters = quest.monsters.filter((m) => !skipTypes.includes(m.type));

  const groups: { name: string; tiles: TileCoord[]; types: string[] }[] = [];
  const byRoom = new Map<number, (typeof groups)[number]>();
  const corridor: (typeof groups)[number][] = [];
  for (const m of monsters) {
    const t = { x: m.x, y: m.y };
    const region = at(t);
    if (region > 0) {
      let g = byRoom.get(region);
      if (!g) {
        const name = board.rooms.find((r) => r.id === region)?.name ?? `Room ${region}`;
        g = { name, tiles: [], types: [] };
        byRoom.set(region, g);
        groups.push(g);
      }
      g.tiles.push(t);
      g.types.push(m.type);
      continue;
    }
    // Join every corridor group within reach, merging groups the monster links up.
    const near = corridor.filter((g) => g.tiles.some((u) => Math.max(Math.abs(u.x - t.x), Math.abs(u.y - t.y)) <= CORRIDOR_REACH));
    const [first, ...rest] = near;
    if (!first) {
      const g = { name: `Corridor at (${t.x},${t.y})`, tiles: [t], types: [m.type] };
      corridor.push(g);
      groups.push(g);
      continue;
    }
    first.tiles.push(t);
    first.types.push(m.type);
    for (const g of rest) {
      first.tiles.push(...g.tiles);
      first.types.push(...g.types);
      corridor.splice(corridor.indexOf(g), 1);
      groups.splice(groups.indexOf(g), 1);
    }
  }

  return groups
    .map((g) => ({
      name: g.name,
      monsters: g.types,
      distance: Math.min(...g.tiles.map((t) => distances.get(key(t)) ?? Infinity)),
    }))
    .sort((a, b) => a.distance - b.distance || a.name.localeCompare(b.name));
}

/** Steps from the nearest start square to every reachable square. */
export function walkDistances(board: BoardDoc, quest: QuestDoc): Map<string, number> {
  const { width, height, regions } = board;
  const walls = new Set(deriveWalls(width, height, regions, board.drawnWalls ?? []).map(edgeKey));
  const blocked = new Set<string>();
  for (const b of quest.blockedSquares) {
    if (b.hiddenDoor) {
      continue;
    }
    for (let x = b.x; x < b.x + b.w; x++) {
      for (let y = b.y; y < b.y + b.h; y++) {
        blocked.add(key({ x, y }));
      }
    }
  }
  const open = (t: TileCoord): boolean => regionAt(width, height, regions, t) >= 0 && !blocked.has(key(t));
  const passable = (e: Edge): boolean => !walls.has(edgeKey(e)) || quest.doors.some((d) => doorCovers(d, e));

  const dist = new Map<string, number>();
  let frontier = quest.startTiles.filter(open);
  for (const t of frontier) {
    dist.set(key(t), 0);
  }
  for (let step = 1; frontier.length > 0; step++) {
    const next: TileCoord[] = [];
    for (const t of frontier) {
      for (const n of [
        { x: t.x + 1, y: t.y },
        { x: t.x - 1, y: t.y },
        { x: t.x, y: t.y + 1 },
        { x: t.x, y: t.y - 1 },
      ]) {
        const e = edgeBetween(t, n);
        if (!e || dist.has(key(n)) || !open(n) || !passable(e)) {
          continue;
        }
        dist.set(key(n), step);
        next.push(n);
      }
    }
    frontier = next;
  }
  return dist;
}

function key(t: TileCoord): string {
  return `${t.x},${t.y}`;
}

function edgeKey(e: Edge): string {
  return `${e.x},${e.y},${e.orientation}`;
}
