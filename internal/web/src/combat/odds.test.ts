import { describe, expect, test } from 'bun:test';
import { parseDice, type DiceExpr } from '../dice/dice.ts';
import {
  attacksToKill,
  chanceAtLeast,
  distribution,
  heroAttack,
  heroQuestDamage,
  monsterAttack,
  MONSTER_CRIT_CHANCE,
  type HeroAttacker,
} from './odds.ts';

function dice(s: string): DiceExpr {
  const r = parseDice(s);
  if (!r.ok) {
    throw new Error(r.error);
  }
  return r.expr;
}

function attacker(hitDice: string, over: Partial<HeroAttacker> = {}): HeroAttacker {
  return { hitDice: dice(hitDice), accuracy: 0, critFrom: 18, critMultiplier: 2, nearMiss: 2, damage: 1, ...over };
}

describe('distribution', () => {
  test('gives exact chances for each total', () => {
    const d = distribution(dice('2d6'));
    expect(d.get(7)).toBeCloseTo(6 / 36, 10);
    expect(d.get(2)).toBeCloseTo(1 / 36, 10);
    expect(d.get(13)).toBeUndefined();
  });

  test('handles subtracted dice and constants', () => {
    const d = distribution(dice('1d6-1d4'));
    expect(d.get(5)).toBeCloseTo(1 / 24, 10);
    expect(d.get(-3)).toBeCloseTo(1 / 24, 10);
    expect([...distribution(dice('3')).entries()]).toEqual([[3, 1]]);
  });

  test('chanceAtLeast sums the tail', () => {
    expect(chanceAtLeast(distribution(dice('1d20')), 9)).toBeCloseTo(0.6, 10);
    expect(chanceAtLeast(distribution(dice('3d6')), 15)).toBeCloseTo(20 / 216, 10);
  });
});

describe('heroAttack', () => {
  test('a near-miss crit turns a close miss into a normal hit', () => {
    // Needs 9+ on the d20; a crit (18-20) also saves rolls of 7 and 8.
    const o = heroAttack(attacker('1d20', { accuracy: 1, damage: 8 }), 10);
    expect(o.hit).toBeCloseTo(0.6 + 0.15 * 0.1, 10);
    expect(o.crit).toBeCloseTo(0.6 * 0.15, 10);
    expect(o.nearMissHit).toBeCloseTo(0.015, 10);
    expect(o.expectedDamage).toBeCloseTo(8 * (0.6 * 0.85 + 0.6 * 0.15 * 2 + 0.015), 10);
  });

  test('the sample Ranger hits 91% of the time against Avoidance 10', () => {
    const o = heroAttack(attacker('2d10+4', { damage: 5 }), 10);
    expect(o.hit).toBeCloseTo(0.9 + 0.15 * 0.07, 10);
    expect(o.expectedDamage).toBeCloseTo(5 * (0.9 * 0.85 + 0.9 * 0.15 * 2 + 0.15 * 0.07), 10);
  });

  test('all ones on every die is a critical miss, even when the total would hit', () => {
    const o = heroAttack(attacker('2d4+10'), 4);
    const allOnes = (1 / 16) * (1 / 20);
    expect(o.criticalMiss).toBeCloseTo(allOnes, 10);
    expect(o.hit).toBeCloseTo(1 - allOnes, 10);
  });

  test('every die at its maximum is a special crit, even when the total would miss', () => {
    const o = heroAttack(attacker('1d4', { critFrom: 20, damage: 3 }), 20);
    expect(o.specialCrit).toBeCloseTo(1 / 80, 10);
    expect(o.hit).toBeCloseTo(1 / 80, 10);
    expect(o.crit).toBeCloseTo(1 / 80, 10);
    expect(o.expectedDamage).toBeCloseTo(6 / 80, 10);
  });

  test('the damage distribution adds up to 1', () => {
    const o = heroAttack(attacker('2d10', { critFrom: 13, critMultiplier: 3, damage: 4 }), 12);
    let total = 0;
    for (const p of o.damage.values()) {
      total += p;
    }
    expect(total).toBeCloseTo(1, 10);
    expect(o.damage.get(12)).toBeCloseTo(o.crit, 10);
  });

  test('rejects a crit range outside 2-20', () => {
    expect(() => heroAttack(attacker('1d20', { critFrom: 1 }), 10)).toThrow();
    expect(() => heroAttack(attacker('1d20', { critFrom: 21 }), 10)).toThrow();
  });
});

describe('monsterAttack', () => {
  const goblinish = { hitDice: dice('1d12'), damage: 3 };

  test('the monster must beat the hero total; ties go to the hero', () => {
    const o = monsterAttack(goblinish, { defenseDice: dice('1d6'), baseAvoidance: 4, mitigation: 0 });
    // Beating 4 + k on the d6 needs 5 + k or more on the d12: 7 + 6 + ... + 2 = 27 of 72.
    expect(o.hit).toBeCloseTo(27 / 72, 10);
    expect(o.crit).toBeCloseTo((27 / 72) * MONSTER_CRIT_CHANCE, 10);
    expect(o.expectedDamage).toBeCloseTo(3 * (27 / 72) * (1 + MONSTER_CRIT_CHANCE), 10);
  });

  test('mitigation comes off after a crit doubles the damage, and can reach 0', () => {
    const o = monsterAttack(goblinish, { defenseDice: dice('1d6'), baseAvoidance: 4, mitigation: 2 });
    const h = 27 / 72;
    expect(o.expectedDamage).toBeCloseTo(h * ((1 - MONSTER_CRIT_CHANCE) * 1 + MONSTER_CRIT_CHANCE * 4), 10);
    const tough = monsterAttack(goblinish, { defenseDice: dice('1d6'), baseAvoidance: 4, mitigation: 3 });
    expect(tough.hit).toBeCloseTo(h, 10);
    expect(tough.damage.get(0)).toBeCloseTo(1 - h * MONSTER_CRIT_CHANCE, 10);
  });
});

describe('attacksToKill', () => {
  test('counts expected attacks, including misses', () => {
    expect(attacksToKill(new Map([[0, 0.5], [2, 0.5]]), 3)).toBeCloseTo(4, 10);
    expect(attacksToKill(new Map([[1, 1]]), 5)).toBeCloseTo(5, 10);
  });

  test('is infinite when an attack can never do damage', () => {
    expect(attacksToKill(new Map([[0, 1]]), 1)).toBe(Infinity);
  });
});

describe('heroQuestDamage (original combat dice)', () => {
  test('a 2-dice hero against a goblin (1 black-shield die, 1 Body)', () => {
    const d = heroQuestDamage(2, 1, 'monster');
    expect(1 - (d.get(0) ?? 0)).toBeCloseTo(2 / 3, 10);
    expect(attacksToKill(d, 1)).toBeCloseTo(1.5, 10);
  });

  test('a goblin (2 dice) against a hero with 2 white-shield dice', () => {
    const d = heroQuestDamage(2, 2, 'hero');
    expect(1 - (d.get(0) ?? 0)).toBeCloseTo(0.4444, 3);
    let expected = 0;
    for (const [x, p] of d) {
      expected += x * p;
    }
    expect(expected).toBeCloseTo(0.5556, 3);
  });

  test('a 2-dice hero needs about 6.4 attacks for a gargoyle (5 defend, 3 Body)', () => {
    expect(attacksToKill(heroQuestDamage(2, 5, 'monster'), 3)).toBeCloseTo(6.38, 2);
  });
});
