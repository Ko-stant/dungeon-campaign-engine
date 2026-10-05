import { describe, expect, test } from 'bun:test';
import { combatLine, combatTotals, falterAt, gearBonus, itemStatsLine, manaCap, monsterLine, playerMonsterLine } from './combat.ts';
import type { Hero, Item } from './types.ts';

describe('combatLine', () => {
  test('sums up a hero\'s attack and defense', () => {
    expect(combatLine({ hitDice: '1d20', accuracy: 3, critFrom: 17, damage: 10, defenseDice: '1d6', avoidance: 3, mitigation: 2 })).toBe(
      'Hit 1d20+3 · Crit 17-20 · Damage 10 · Avoid 3+1d6 · Mitigation 2',
    );
  });

  test('leaves out what is zero, and shows mana regeneration', () => {
    expect(combatLine({ hitDice: '2d8', accuracy: 0, critFrom: 20, damage: 5, defenseDice: '1d6', avoidance: 0, mitigation: 0, manaRegen: 3 })).toBe(
      'Hit 2d8 · Crit 20 · Damage 5 · Avoid 1d6 · Mana +3 a fight round',
    );
  });

  test('a hit dice expression with its own modifier keeps both', () => {
    expect(combatLine({ hitDice: '2d10+2', accuracy: 3, critFrom: 15, damage: 6, defenseDice: '1d6', avoidance: 7, mitigation: 0 })).toBe(
      'Hit 2d10+2 +3 · Crit 15-20 · Damage 6 · Avoid 7+1d6',
    );
  });

  test('a negative bonus (a cursed item) shows its sign', () => {
    expect(combatLine({ hitDice: '1d20', accuracy: -1, critFrom: 20, damage: 3, defenseDice: '1d6', avoidance: 0, mitigation: 0 })).toBe(
      'Hit 1d20-1 · Crit 20 · Damage 3 · Avoid 1d6',
    );
  });
});

describe('item stats', () => {
  test('itemStatsLine names each bonus, in the Go order', () => {
    expect(itemStatsLine({ damage: 7 })).toBe('damage +7');
    expect(itemStatsLine({ accuracy: 1, avoidance: 2 })).toBe('Accuracy +1, avoidance +2');
    expect(itemStatsLine({ mana: 2, manaRegen: 1, mitigation: -1 })).toBe('mitigation -1, mana +2, mana regen +1');
    expect(itemStatsLine({})).toBe('');
  });

  const items: Item[] = [
    { id: 'item-1', name: 'Greataxe', quantity: 1, equipped: true, damage: 7 },
    { id: 'item-2', name: 'Hide Cuirass', quantity: 1, equipped: true, mitigation: 1, mana: 2, manaRegen: 1 },
    { id: 'item-3', name: 'Iron-shod Boots', quantity: 1, avoidance: 1 },
  ];
  const hero = {
    id: 'hero-1', name: 'Grom', class: 'custom-barbarian', x: 1, y: 1, placed: true, body: 40, maxBody: 40, mind: 2, maxMind: 2,
    status: 'active', mana: 4, maxMana: 4, items,
    combat: { hitDice: '1d20', accuracy: 3, critFrom: 17, damage: 3, defenseDice: '1d6', avoidance: 2, mitigation: 1 },
  } satisfies Hero;

  test('gearBonus adds up equipped items only, once each', () => {
    expect(gearBonus(items)).toEqual({ damage: 7, accuracy: 0, avoidance: 0, mitigation: 1, mana: 2, manaRegen: 1 });
    expect(gearBonus(null)).toEqual({ damage: 0, accuracy: 0, avoidance: 0, mitigation: 0, mana: 0, manaRegen: 0 });
  });

  test('combatTotals is the class stats plus equipped items', () => {
    expect(combatTotals(hero)).toEqual({ hitDice: '1d20', accuracy: 3, critFrom: 17, damage: 10, defenseDice: '1d6', avoidance: 2, mitigation: 2, manaRegen: 1 });
    expect(combatTotals({ ...hero, combat: null })).toBeNull();
  });

  test('manaCap raises the class maximum by equipped mana', () => {
    expect(manaCap(hero)).toBe(6);
    expect(manaCap({ ...hero, items: [] })).toBe(4);
  });
});

describe('monsterLine', () => {
  test('sums up a monster\'s combat stats and traits', () => {
    expect(monsterLine({ avoidance: 14, hitDice: '2d10+3', damage: 12, line: 1 })).toBe('Avoid 14 · Hit 2d10+3 · Damage 12 · strikes 2 in a line');
    expect(monsterLine({ avoidance: 6, hitDice: '1d12', damage: 5, ranged: true, splashDamage: 2, splashTargets: 2 })).toBe(
      'Avoid 6 · Hit 1d12 · Damage 5 · ranged · blast 2 to 2 beside the target',
    );
    expect(monsterLine({ avoidance: 12, hitDice: '2d8+1', damage: 10, undead: true, reach: true })).toBe('Avoid 12 · Hit 2d8+1 · Damage 10 · reach · undead');
  });
});

describe('playerMonsterLine', () => {
  test('uses the handbook trait names: reach covers striking in a line, AoE covers the blast', () => {
    expect(playerMonsterLine({ avoidance: 14, hitDice: '2d10+3', damage: 12, line: 1 })).toBe('Avoid 14 · Hit 2d10+3 · Damage 12 · reach');
    expect(playerMonsterLine({ avoidance: 12, hitDice: '2d8+1', damage: 10, reach: true, line: 1, undead: true })).toBe(
      'Avoid 12 · Hit 2d8+1 · Damage 10 · reach · undead',
    );
    expect(playerMonsterLine({ avoidance: 6, hitDice: '1d12', damage: 5, ranged: true, splashDamage: 2, splashTargets: 2 })).toBe(
      'Avoid 6 · Hit 1d12 · Damage 5 · ranged · AoE',
    );
  });
});

describe('falterAt', () => {
  test('a quarter of maximum Body, rounded down, and at least 1', () => {
    expect(falterAt(109)).toBe(27);
    expect(falterAt(5)).toBe(1);
    expect(falterAt(3)).toBe(1);
  });
});
