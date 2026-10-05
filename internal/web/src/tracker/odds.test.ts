import { describe, expect, test } from 'bun:test';
import { CAMPAIGN_RULES, heroAttack, killOdds, monsterAttack } from '../combat/odds.ts';
import { parseDice } from '../dice/dice.ts';
import { attackOdds, defenseOdds, oddsPercent } from './odds.ts';
import type { Hero, Monster } from './types.ts';

function dice(s: string) {
  const r = parseDice(s);
  if (!r.ok) {
    throw new Error(r.error);
  }
  return r.expr;
}

// The Ranger with the starting kit: class 2d10+4, crit 18, damage 2; Hunting Bow +5, Archer's Gloves +1 Accuracy, Leather Armor +2 avoidance.
const rangerCombat = { hitDice: '2d10', accuracy: 4, critFrom: 18, damage: 2, defenseDice: '1d6', avoidance: 4, mitigation: 0 };
const ranger: Hero = {
  id: 'hero-1', name: 'Mordecai', class: 'custom-ranger', x: 1, y: 1, placed: true, body: 30, maxBody: 30, mind: 3, maxMind: 3, status: 'active',
  combat: rangerCombat,
  items: [
    { id: 'item-1', name: 'Hunting Bow', quantity: 1, equipped: true, damage: 5 },
    { id: 'item-2', name: "Archer's Gloves", quantity: 1, equipped: true, accuracy: 1 },
    { id: 'item-3', name: 'Leather Armor', quantity: 1, equipped: true, avoidance: 2 },
    { id: 'item-4', name: 'Spare Bow', quantity: 1, damage: 50 },
  ],
};

const orc: Monster = {
  id: 'monster-1', type: 'orc', name: 'Orc', x: 2, y: 2, body: 22, maxBody: 22, mind: 2, visibility: 'seen', alive: true,
  combat: { avoidance: 8, hitDice: '2d8', damage: 9 },
};

const attacker = { hitDice: dice('2d10'), accuracy: 5, critFrom: 18, critMultiplier: 2, nearMiss: 2, damage: 7 };

describe('attackOdds', () => {
  test('a hero attacking a monster: totals with equipped gear, against its Avoidance and current Body', () => {
    const odds = attackOdds(ranger, orc);
    const exact = heroAttack(attacker, 8);
    const kill = killOdds(attacker, 8, 22, { ...CAMPAIGN_RULES, falterAt: 5 });
    expect(odds).not.toBeNull();
    expect(odds?.hit).toBeCloseTo(exact.hit, 10);
    expect(odds?.crit).toBeCloseTo(exact.crit, 10);
    expect(odds?.expectedDamage).toBeCloseTo(exact.expectedDamage, 10);
    expect(odds?.attacksToKill).toBeCloseTo(kill.expected, 10);
    expect(odds?.faltering).toBe(false);
  });

  test('a faltering monster is easier to hit and nearly dead', () => {
    const hurt = { ...orc, body: 5 };
    const odds = attackOdds(ranger, hurt);
    expect(odds?.faltering).toBe(true);
    expect(odds?.hit).toBeCloseTo(heroAttack(attacker, 4).hit, 10);
    expect(odds?.attacksToKill).toBeLessThan(attackOdds(ranger, orc)?.attacksToKill ?? 0);
  });

  test('no odds without combat stats on both sides, or for a dead monster', () => {
    expect(attackOdds({ ...ranger, combat: null }, orc)).toBeNull();
    expect(attackOdds(ranger, { ...orc, combat: null })).toBeNull();
    expect(attackOdds(ranger, { ...orc, alive: false })).toBeNull();
    expect(attackOdds({ ...ranger, combat: { ...rangerCombat, hitDice: 'lots' } }, orc)).toBeNull();
  });
});

describe('defenseOdds', () => {
  test("a monster attacking a hero: the hero's avoidance with gear, and mitigation", () => {
    const odds = defenseOdds(orc, ranger);
    const exact = monsterAttack({ hitDice: dice('2d8'), damage: 9 }, { defenseDice: dice('1d6'), baseAvoidance: 6, mitigation: 0 });
    expect(odds?.hit).toBeCloseTo(exact.hit, 10);
    expect(odds?.damage).toBe(9);
    expect(odds?.expectedDamage).toBeCloseTo(exact.expectedDamage, 10);
  });

  test('mitigation takes damage off each hit, down to 0', () => {
    const tank: Hero = { ...ranger, combat: { ...rangerCombat, mitigation: 12 } };
    expect(defenseOdds(orc, tank)?.damage).toBe(0);
    expect(defenseOdds(orc, { ...ranger, combat: { ...rangerCombat, mitigation: 2 } })?.damage).toBe(7);
  });

  test('no odds for a hero who is dead or has escaped', () => {
    expect(defenseOdds(orc, { ...ranger, status: 'dead' })).toBeNull();
    expect(attackOdds({ ...ranger, status: 'escaped' }, orc)).toBeNull();
  });
});

describe('oddsPercent', () => {
  test('whole percents, with "<1%" and ">99%" at the ends', () => {
    expect(oddsPercent(0.426)).toBe('43%');
    expect(oddsPercent(0.004)).toBe('<1%');
    expect(oddsPercent(0)).toBe('0%');
    expect(oddsPercent(0.9995)).toBe('>99%');
    expect(oddsPercent(1)).toBe('100%');
  });
});
