import { describe, expect, test } from 'bun:test';
import { parseDice, type DiceExpr } from '../dice/dice.ts';
import type { HeroAttacker, HeroDefender } from './odds.ts';
import {
  cooldownFloor,
  DEFAULT_TACTICS,
  heroStrike,
  monsterStrike,
  newHero,
  restBetweenFights,
  runFight,
  runQuest,
  seededRoller,
  type HeroSpec,
  type MonsterSpec,
  type Roller,
  type Supplies,
  type Tactics,
} from './simulate.ts';

function dice(s: string): DiceExpr {
  const r = parseDice(s);
  if (!r.ok) {
    throw new Error(r.error);
  }
  return r.expr;
}

/** Hands out the given die results in order; fails the test when they run out. */
function scripted(values: number[]): Roller & { left: () => number } {
  let i = 0;
  return {
    die(sides: number): number {
      const v = values[i++];
      if (v === undefined || v < 1 || v > sides) {
        throw new Error(`roll ${i}: no scripted value for d${sides} (got ${v})`);
      }
      return v;
    },
    left: () => values.length - i,
  };
}

const barbarian: HeroAttacker = { hitDice: dice('1d20'), accuracy: 3, critFrom: 17, critMultiplier: 2, nearMiss: 2, damage: 10 };

describe('seededRoller', () => {
  test('repeats for the same seed and stays in range', () => {
    const a = seededRoller(42);
    const b = seededRoller(42);
    let sum = 0;
    for (let i = 0; i < 10000; i++) {
      const v = a.die(6);
      expect(v).toBe(b.die(6));
      expect(v).toBeGreaterThanOrEqual(1);
      expect(v).toBeLessThanOrEqual(6);
      sum += v;
    }
    expect(sum / 10000).toBeCloseTo(3.5, 1);
  });
});

describe('heroStrike', () => {
  test('meets Avoidance to hit; the crit die doubles a hit', () => {
    expect(heroStrike(barbarian, 12, scripted([9, 5]))).toEqual({ damage: 10, hit: true, crit: false, criticalMiss: false });
    expect(heroStrike(barbarian, 12, scripted([15, 18]))).toEqual({ damage: 20, hit: true, crit: true, criticalMiss: false });
  });

  test('a crit on a miss adds the near-miss bonus, for normal damage', () => {
    expect(heroStrike(barbarian, 12, scripted([7, 17]))).toEqual({ damage: 10, hit: true, crit: false, criticalMiss: false });
    expect(heroStrike(barbarian, 12, scripted([5, 19])).hit).toBe(false);
  });

  test('all ones is a critical miss; all maximums is a special crit', () => {
    expect(heroStrike(barbarian, 2, scripted([1, 1]))).toEqual({ damage: 0, hit: false, crit: false, criticalMiss: true });
    expect(heroStrike(barbarian, 30, scripted([20, 20]))).toEqual({ damage: 20, hit: true, crit: true, criticalMiss: false });
  });

  test('options: Accuracy bonus, advantage, sure hit, extra damage and a damage factor', () => {
    expect(heroStrike(barbarian, 12, scripted([7, 3]), { bonus: 2 }).hit).toBe(true);
    const r = scripted([4, 15, 3]);
    expect(heroStrike(barbarian, 18, r, { advantage: true }).hit).toBe(true);
    expect(r.left()).toBe(0);
    expect(heroStrike(barbarian, 99, scripted([3]), { sure: true })).toEqual({ damage: 10, hit: true, crit: false, criticalMiss: false });
    expect(heroStrike(barbarian, 99, scripted([18]), { sure: true }).damage).toBe(20);
    expect(heroStrike(barbarian, 12, scripted([15, 3]), { extraDamage: 4 }).damage).toBe(14);
    expect(heroStrike(barbarian, 12, scripted([15, 3]), { damageFactor: 3 }).damage).toBe(30);
  });
});

describe('monsterStrike', () => {
  const orc = { hitDice: dice('2d8'), damage: 5 };
  const hero: HeroDefender = { defenseDice: dice('1d6'), baseAvoidance: 4, mitigation: 1 };

  test('beats the hero total to hit; ties go to the hero; mitigation comes off', () => {
    expect(monsterStrike(orc, hero, scripted([6, 5, 3, 1, 1]))).toEqual({ damage: 4, hit: true, crit: false });
    const tie = scripted([4, 4, 4]);
    expect(monsterStrike(orc, hero, tie)).toEqual({ damage: 0, hit: false, crit: false });
    expect(tie.left()).toBe(0);
  });

  test('double 20s crit for double damage before mitigation', () => {
    expect(monsterStrike(orc, hero, scripted([8, 8, 2, 20, 20]))).toEqual({ damage: 9, hit: true, crit: true });
  });

  test('a hero who cannot defend keeps base avoidance and rolls no defense dice', () => {
    expect(monsterStrike(orc, hero, scripted([3, 2, 1, 1]), false)).toEqual({ damage: 4, hit: true, crit: false });
    expect(monsterStrike(orc, { ...hero, mitigation: 5 }, scripted([8, 8, 1, 1, 2])).damage).toBe(0);
  });
});

describe('cooldownFloor', () => {
  test('out of a fight, short cooldowns stop at 1 left, long ones at 2, lower ones stay', () => {
    expect(cooldownFloor(3, 3)).toBe(1);
    expect(cooldownFloor(5, 5)).toBe(2);
    expect(cooldownFloor(1, 5)).toBe(1);
    expect(cooldownFloor(0, 4)).toBe(0);
  });
});

const sureHits = (damage: number) => ({ hitDice: dice('1d20+90'), damage });
const tough: HeroDefender = { defenseDice: dice('1d4'), baseAvoidance: 0, mitigation: 0 };
const supplies = (): Supplies => ({ healPotions: 0, healAmount: 10, manaPotions: 0, manaAmount: 6 });

function spec(over: Partial<HeroSpec> = {}): HeroSpec {
  return { name: 'Barbarian', cls: 'barbarian', body: 40, attack: barbarian, defense: tough, ...over };
}

describe('runFight', () => {
  const attackOnly: Tactics = { ...DEFAULT_TACTICS, abilities: false };

  test('a goblin that cannot avoid anything dies in the first round', () => {
    const goblin: MonsterSpec = { body: 1, avoidance: -100, attack: sureHits(3) };
    const heroes = [newHero(spec())];
    const r = runFight(heroes, [goblin], attackOnly, supplies(), seededRoller(1));
    expect(r).toMatchObject({ won: true, rounds: 1 });
    expect(heroes[0]?.body).toBe(40);
  });

  test('at most meleeLimit melee monsters attack in a round (a held door); ranged ones always can', () => {
    const wall: MonsterSpec = { body: 1000, avoidance: 1000, attack: sureHits(1) };
    const harmless = { ...barbarian, damage: 0 };
    const pair = () => [newHero(spec({ body: 100, attack: harmless })), newHero(spec({ name: 'Second', body: 100, attack: harmless }))];
    const lost = (hs: { body: number }[]) => hs.reduce((n, h) => n + 100 - h.body, 0);
    const held = pair();
    runFight(held, [wall, wall, wall, wall], { ...attackOnly, meleeLimit: 2 }, supplies(), seededRoller(2), 1);
    expect(lost(held)).toBe(2);
    const shot = pair();
    runFight(shot, [1, 2, 3, 4].map(() => ({ ...wall, ranged: true })), { ...attackOnly, meleeLimit: 2 }, supplies(), seededRoller(3), 1);
    expect(lost(shot)).toBe(4);
  });

  test('a line attack strikes one more hero, who defends separately', () => {
    const gargoyle: MonsterSpec = { body: 1000, avoidance: 1000, attack: sureHits(1), line: 1 };
    const harmless = { ...barbarian, damage: 0 };
    const party = [newHero(spec({ body: 100, attack: harmless })), newHero(spec({ name: 'B', body: 100, attack: harmless })), newHero(spec({ name: 'C', body: 100, attack: harmless }))];
    runFight(party, [gargoyle], attackOnly, supplies(), seededRoller(12), 1);
    expect(party.map((h) => 100 - h.body).sort()).toEqual([0, 1, 1]);
  });

  test('a reaching monster attacks past the held doorway (not counted in meleeLimit)', () => {
    const wall: MonsterSpec = { body: 1000, avoidance: 1000, attack: sureHits(1) };
    const harmless = { ...barbarian, damage: 0 };
    const heroes = [newHero(spec({ body: 100, attack: harmless }))];
    runFight(heroes, [wall, wall, { ...wall, reach: true }], { ...attackOnly, meleeLimit: 2 }, supplies(), seededRoller(13), 1);
    expect(heroes[0]?.body).toBe(97);
  });

  test('a splash attack also hurts other heroes, no roll, less mitigation', () => {
    const warlock: MonsterSpec = { body: 1000, avoidance: 1000, attack: sureHits(0), ranged: true, splash: { damage: 3, targets: 2 } };
    const harmless = { ...barbarian, damage: 0 };
    const party = [
      newHero(spec({ body: 100, attack: harmless })),
      newHero(spec({ name: 'B', body: 100, attack: harmless })),
      newHero(spec({ name: 'C', body: 100, attack: harmless, defense: { ...tough, mitigation: 1 } })),
    ];
    runFight(party, [warlock], attackOnly, supplies(), seededRoller(10), 1);
    const lost = party.map((h) => 100 - h.body).sort();
    // The target takes 0; two others take 3 (or 2 through mitigation 1).
    expect(lost[0]).toBe(0);
    expect(lost.reduce((n, x) => n + x, 0)).toBeGreaterThanOrEqual(5);
    expect(lost.reduce((n, x) => n + x, 0)).toBeLessThanOrEqual(6);
  });

  test('only the first openingAttackers heroes act in the first round (the rest are moving in)', () => {
    const steady = { ...barbarian, critMultiplier: 1, damage: 1 };
    const target: MonsterSpec = { body: 2, avoidance: -100, attack: sureHits(0) };
    const two = () => [newHero(spec({ attack: steady })), newHero(spec({ name: 'Second', attack: steady }))];
    expect(runFight(two(), [target], { ...attackOnly, openingAttackers: 1 }, supplies(), seededRoller(9)).rounds).toBe(2);
    expect(runFight(two(), [target], { ...attackOnly, openingAttackers: 2 }, supplies(), seededRoller(9)).rounds).toBe(1);
  });

  test('the party loses when every hero falls', () => {
    const brute: MonsterSpec = { body: 1000, avoidance: 1000, attack: sureHits(50) };
    const heroes = [newHero(spec({ body: 10 }))];
    expect(runFight(heroes, [brute], attackOnly, supplies(), seededRoller(4)).won).toBe(false);
    expect(heroes[0]?.alive).toBe(false);
  });

  test('drinking a potion is free: a badly hurt hero drinks and still attacks', () => {
    const wall: MonsterSpec = { body: 1000, avoidance: 1000, attack: sureHits(0) };
    const heroes = [newHero(spec())];
    const h = heroes[0];
    if (!h) {
      throw new Error('no hero');
    }
    h.body = 5;
    const s = { ...supplies(), healPotions: 1 };
    runFight(heroes, [wall], attackOnly, s, seededRoller(5), 1);
    expect(h.body).toBe(15);
    expect(s.healPotions).toBe(0);
    expect(h.uses).toEqual({ potion: 1, attack: 1 });
  });

  test('a potion can be drunk during the monsters\' turn, between hits', () => {
    const wall: MonsterSpec = { body: 1000, avoidance: 1000, attack: sureHits(6) };
    const h = newHero(spec({ attack: { ...barbarian, damage: 0 } }));
    h.body = 20;
    const s = { ...supplies(), healPotions: 1 };
    runFight([h], [wall, wall, wall], { ...attackOnly, meleeLimit: 3 }, s, seededRoller(38), 1);
    // 20 -> 14 -> 8 (at or below a quarter: drink, 18) -> 12.
    expect(h.body).toBe(12);
    expect(s.healPotions).toBe(0);
  });
});

describe('abilities in a fight', () => {
  const harmless = { ...barbarian, damage: 0 };
  const wall = (damage: number, over: Partial<MonsterSpec> = {}): MonsterSpec => ({ body: 100, avoidance: 1000, attack: sureHits(damage), ...over });

  test('Riposte: a miss on the Rogue costs the monster half the Rogue\'s weapon damage, no roll', () => {
    const rogue = newHero(spec({ name: 'Rogue', cls: 'rogue', body: 50, attack: { ...barbarian, damage: 6 }, defense: { ...tough, baseAvoidance: 200 } }));
    const r = runFight([rogue], [wall(5, { avoidance: 1000 })], { ...DEFAULT_TACTICS, vanish: null }, supplies(), seededRoller(14), 1);
    expect(r.foesBody).toEqual([97]);
    expect(rogue.body).toBe(50);
  });

  test('Venom Vial: on a hit, the target loses 3 Body at the start of each of its next 3 turns', () => {
    const rogue = newHero(spec({ name: 'Rogue', cls: 'rogue', attack: { ...harmless, critFrom: 20 } }));
    const r = runFight([rogue], [wall(0, { avoidance: -100 })], { ...DEFAULT_TACTICS, fan: null }, supplies(), seededRoller(19), 4);
    expect(r.foesBody).toEqual([91]);
    expect(rogue.uses.vial).toBe(1);
  });

  test('Venom Vial with weapon: the hit also deals weapon damage', () => {
    const rogue = newHero(spec({ name: 'Rogue', cls: 'rogue', attack: { ...barbarian, critFrom: 20, critMultiplier: 1, damage: 6 } }));
    const r = runFight([rogue], [wall(0, { avoidance: -100 })], { ...DEFAULT_TACTICS, fan: null, vial: { cooldown: 4, damage: 3, turns: 3, weapon: true } }, supplies(), seededRoller(20), 1);
    expect(r.foesBody).toEqual([91]);
  });

  test('Challenge pulls the melee attacks onto the Barbarian', () => {
    const barb = newHero(spec({ attack: harmless }));
    const ally = newHero(spec({ name: 'Ranger', cls: 'ranger', attack: harmless }));
    runFight([barb, ally], [wall(1), wall(1)], { ...DEFAULT_TACTICS, fury: null, charge: null, multi: null, aimed: null, rain: null }, supplies(), seededRoller(15), 1);
    expect(barb.body).toBe(38);
    expect(ally.body).toBe(40);
  });

  test('Unleash Fury: +3 damage and +2 mitigation while raging', () => {
    const barb = newHero(spec({ attack: { ...barbarian, critMultiplier: 1 } }));
    const r = runFight([barb], [wall(5, { avoidance: -100 })], { ...DEFAULT_TACTICS, charge: null, challenge: null }, supplies(), seededRoller(16), 1);
    expect(r.foesBody).toEqual([87]);
    expect(barb.body).toBe(37);
    expect(barb.uses.fury).toBe(1);
  });

  test('Turn Evil: the target skips its next attack', () => {
    const cleric = newHero(spec({ name: 'Cleric', cls: 'cleric', body: 30, mana: 20, attack: harmless }));
    runFight([cleric], [wall(5)], DEFAULT_TACTICS, supplies(), seededRoller(17), 1);
    expect(cleric.body).toBe(30);
    expect(cleric.uses.turnEvil).toBe(1);
  });

  test('every hero action is counted', () => {
    const barb = newHero(spec());
    runFight([barb], [{ body: 1, avoidance: -100, attack: sureHits(0) }], { ...DEFAULT_TACTICS, abilities: false }, supplies(), seededRoller(18));
    expect(barb.uses).toEqual({ attack: 1 });
  });
});

describe('restBetweenFights', () => {
  test('applies the cooldown floor and the pool heals every living hero once', () => {
    const h = newHero(spec());
    h.body = 12;
    h.cooldowns.charge = 3;
    restBetweenFights([h], DEFAULT_TACTICS, supplies(), true);
    expect(h.cooldowns.charge).toBe(1);
    expect(h.body).toBe(40);
  });

  test('the Cleric heals hurt heroes out of a fight while mana lasts', () => {
    const cleric = newHero(spec({ name: 'Cleric', cls: 'cleric', body: 30, mana: 12 }));
    const hurt = newHero(spec());
    hurt.body = 10;
    restBetweenFights([cleric, hurt], DEFAULT_TACTICS, supplies(), false);
    expect(hurt.body).toBeGreaterThan(10);
    expect(cleric.mana).toBeLessThan(12);
  });
});

describe('runQuest', () => {
  const monsters: Record<string, MonsterSpec> = {
    goblin: { body: 1, avoidance: -100, attack: sureHits(1) },
    brute: { body: 1000, avoidance: 1000, attack: sureHits(100) },
  };

  test('an overwhelming party clears every encounter', () => {
    const r = runQuest([spec()], monsters, { encounters: [{ name: 'A', monsters: ['goblin'] }, { name: 'B', monsters: ['goblin', 'goblin'] }] }, DEFAULT_TACTICS, supplies(), seededRoller(6));
    expect(r.cleared).toBe(true);
    expect(r.deaths).toEqual([]);
    expect(r.fought.map((f) => f.name)).toEqual(['A', 'B']);
  });

  test('a wipe ends the quest at that encounter', () => {
    const r = runQuest([spec()], monsters, { encounters: [{ name: 'A', monsters: ['brute'] }, { name: 'B', monsters: ['goblin'] }] }, DEFAULT_TACTICS, supplies(), seededRoller(7));
    expect(r).toMatchObject({ cleared: false, wipedAt: 'A', deaths: ['Barbarian'] });
    expect(r.fought).toHaveLength(1);
  });

  test('a find changes a hero\'s stats after its encounter; Body and mana caps follow', () => {
    const plan = {
      encounters: [{ name: 'A', monsters: ['goblin'] }, { name: 'B', monsters: ['goblin'] }],
      finds: [{ after: 'A', hero: 'Barbarian', apply: (h: HeroSpec) => ({ ...h, body: 50, mana: 4, attack: { ...h.attack, damage: 12 } }) }],
    };
    const r = runQuest([spec()], monsters, plan, DEFAULT_TACTICS, supplies(), seededRoller(26));
    expect(r.cleared).toBe(true);
    expect(r.bodyLeft.Barbarian).toBe(40);
    expect(r.specs.Barbarian?.attack.damage).toBe(12);
    expect(r.specs.Barbarian?.body).toBe(50);
  });

  test('encounters from shuffleFrom on come in a new order each run', () => {
    const plan = { encounters: ['A', 'B', 'C', 'D'].map((name) => ({ name, monsters: ['goblin'] })), shuffleFrom: 1 };
    const roller = seededRoller(11);
    const orders = new Set<string>();
    for (let i = 0; i < 50; i++) {
      const names = runQuest([spec()], monsters, plan, DEFAULT_TACTICS, supplies(), roller).fought.map((f) => f.name);
      expect(names[0]).toBe('A');
      expect([...names].sort()).toEqual(['A', 'B', 'C', 'D']);
      orders.add(names.join(''));
    }
    expect(orders.size).toBeGreaterThan(3);
  });

  test('an unknown monster type is an error', () => {
    expect(() => runQuest([spec()], monsters, { encounters: [{ name: 'A', monsters: ['dragon'] }] }, DEFAULT_TACTICS, supplies(), seededRoller(8))).toThrow('dragon');
  });
});
