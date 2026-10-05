import { describe, expect, test } from 'bun:test';
import { combatLine, falterAt, monsterLine } from './combat.ts';

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

describe('falterAt', () => {
  test('a quarter of maximum Body, rounded down, and at least 1', () => {
    expect(falterAt(109)).toBe(27);
    expect(falterAt(5)).toBe(1);
    expect(falterAt(3)).toBe(1);
  });
});
