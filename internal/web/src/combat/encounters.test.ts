import { describe, expect, test } from 'bun:test';
import type { BoardDoc, DoorDoc, MonsterDoc, QuestDoc } from '../maps/types.ts';
import { questEncounters } from './encounters.ts';

// 9 x 3, bottom row first. Room 1 (x 1-3) and room 2 (x 7-9) with a corridor
// between (x 4-6); the far room 3 (x 8-9, top row) has no door.
//   y3: 1 1 1 . . . 2 3 3
//   y2: 1 1 1 . . . 2 2 2
//   y1: 1 1 1 . . . 2 2 2
const W = 9;
const rows = [
  [1, 1, 1, 0, 0, 0, 2, 2, 2],
  [1, 1, 1, 0, 0, 0, 2, 2, 2],
  [1, 1, 1, 0, 0, 0, 2, 3, 3],
];
const board: BoardDoc = {
  version: 2,
  width: W,
  height: 3,
  regions: rows.flat(),
  rooms: [
    { id: 1, name: 'Guard room' },
    { id: 2, name: 'Crypt' },
    { id: 3, name: 'Vault' },
  ],
};

function door(id: string, x: number, y: number): DoorDoc {
  return { id, edge: { x, y, orientation: 'vertical' }, kind: 'normal', state: 'closed' };
}

function monster(id: string, type: string, x: number, y: number): MonsterDoc {
  return { id, type, x, y };
}

function quest(monsters: MonsterDoc[], doors: DoorDoc[] = [door('d1', 4, 2), door('d2', 7, 2)]): QuestDoc {
  return {
    version: 2,
    boardChecksum: '',
    doors,
    blockedSquares: [],
    furniture: [],
    monsters,
    traps: [],
    notes: [],
    startTiles: [{ x: 1, y: 1 }],
  };
}

describe('questEncounters', () => {
  test('groups monsters by room, named after the room, nearest first', () => {
    const e = questEncounters(board, quest([
      monster('m1', 'orc', 8, 1),
      monster('m2', 'goblin', 2, 2),
      monster('m3', 'orc', 9, 2),
      monster('m4', 'goblin', 3, 3),
    ]));
    expect(e.map((x) => x.name)).toEqual(['Guard room', 'Crypt']);
    expect(e[0]?.monsters).toEqual(['goblin', 'goblin']);
    expect(e[1]?.monsters).toEqual(['orc', 'orc']);
    expect(e[0]?.distance).toBe(2);
    expect(e[1]?.distance).toBe(9);
  });

  test('clusters corridor monsters that stand close together', () => {
    const e = questEncounters(board, quest([
      monster('m1', 'goblin', 4, 1),
      monster('m2', 'goblin', 5, 2),
      monster('m3', 'zombie', 6, 3),
    ]));
    expect(e).toHaveLength(1);
    expect(e[0]?.name).toBe('Corridor at (4,1)');
    expect(e[0]?.monsters).toEqual(['goblin', 'goblin', 'zombie']);
  });

  test('walls block the way unless a door covers the edge; unreachable rooms come last', () => {
    const e = questEncounters(board, quest([monster('m1', 'mummy', 8, 3), monster('m2', 'orc', 7, 1)], [door('d1', 4, 2)]));
    expect(e.map((x) => x.name)).toEqual(['Crypt', 'Vault']);
    expect(e[0]?.distance).toBe(Infinity);
    expect(e[1]?.distance).toBe(Infinity);
  });

  test('skips the given monster types (e.g. NPCs)', () => {
    const e = questEncounters(board, quest([monster('m1', 'stranger', 2, 2), monster('m2', 'orc', 8, 1)]), ['stranger']);
    expect(e.map((x) => x.name)).toEqual(['Crypt']);
  });
});
