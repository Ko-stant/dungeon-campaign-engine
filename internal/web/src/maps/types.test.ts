import { describe, expect, test } from 'bun:test';
import { monsterOptionLabel, monsterPlayLabel, type MonsterDef } from './types.ts';

const base: MonsterDef = { id: 'orc', name: 'Orc', body: 1, mind: 2, attack: 3, defense: 2, movement: 8 };

describe('monsterOptionLabel', () => {
  test('a single-square catalog monster is just its name', () => {
    expect(monsterOptionLabel(base)).toBe('Orc');
    expect(monsterOptionLabel({ ...base, width: 1, height: 1 })).toBe('Orc');
  });

  test('a multi-square catalog monster shows its size', () => {
    expect(monsterOptionLabel({ ...base, id: 'giant_wolf', name: 'Giant Wolf', width: 2, height: 1 })).toBe('Giant Wolf (2×1)');
  });

  test('a custom monster always shows that it is custom and its size', () => {
    expect(monsterOptionLabel({ ...base, name: 'Troll', custom: true })).toBe('Troll (custom 1×1)');
    expect(monsterOptionLabel({ ...base, name: 'Troll', custom: true, width: 2, height: 2 })).toBe('Troll (custom 2×2)');
  });
});

describe('monsterPlayLabel', () => {
  test('in play a custom monster reads like any other: its name, and its size when bigger than one square', () => {
    expect(monsterPlayLabel({ ...base, name: 'Troll', custom: true })).toBe('Troll');
    expect(monsterPlayLabel({ ...base, name: 'Troll', custom: true, width: 2, height: 2 })).toBe('Troll (2×2)');
    expect(monsterPlayLabel(base)).toBe('Orc');
  });
});
