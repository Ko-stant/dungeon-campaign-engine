import { describe, expect, test } from 'bun:test';
import { combatLine } from './combat.ts';

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
