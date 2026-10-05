/**
 * Exact combat odds for The Three Plagues house rules (see
 * docs/campaigns/three-plagues/RULES_AND_CLASSES.md, "Combat"). A design aid for tuning
 * hero and monster numbers; nothing here enforces a rule.
 */

import type { DiceExpr } from '../dice/dice.ts';

/** Total (or damage) -> probability. */
export type Distribution = Map<number, number>;

/** A monster crits only on double 20s on its 2d20 crit check. */
export const MONSTER_CRIT_CHANCE = 1 / 400;

/** The exact chance of each total of a dice expression. */
export function distribution(expr: DiceExpr): Distribution {
  let d: Distribution = new Map([[expr.modifier, 1]]);
  for (const t of expr.terms) {
    for (let i = 0; i < t.count; i++) {
      const next: Distribution = new Map();
      for (const [v, p] of d) {
        for (let face = 1; face <= t.sides; face++) {
          add(next, t.negative ? v - face : v + face, p / t.sides);
        }
      }
      d = next;
    }
  }
  return d;
}

/** The chance a roll totals at least target. */
export function chanceAtLeast(d: Distribution, target: number): number {
  let p = 0;
  for (const [v, q] of d) {
    if (v >= target) {
      p += q;
    }
  }
  return p;
}

export interface HeroAttacker {
  /** The class hit dice, rolled with the d20 crit die. */
  hitDice: DiceExpr;
  accuracy: number;
  /** Lowest natural crit-die roll (2-20) that is a crit. */
  critFrom: number;
  critMultiplier: number;
  /** Added to a missed hit total when the crit die crits. */
  nearMiss: number;
  damage: number;
}

export interface HeroAttackOdds {
  /** Any damage dealt (near-miss hits and special crits included, critical misses excluded). */
  hit: number;
  /** Multiplied damage dealt. */
  crit: number;
  /** Misses turned into normal hits by the near-miss bonus. */
  nearMissHit: number;
  /** Every die (hit dice and crit die) shows 1: a miss, and the hero loses their next turn. */
  criticalMiss: number;
  /** Every die shows its maximum: a crit plus a flourish. */
  specialCrit: number;
  expectedDamage: number;
  damage: Distribution;
}

/** One hero attack against a monster's Avoidance: hit if the total meets or beats it. */
export function heroAttack(a: HeroAttacker, avoidance: number): HeroAttackOdds {
  if (!Number.isInteger(a.critFrom) || a.critFrom < 2 || a.critFrom > 20) {
    throw new Error(`critFrom must be 2-20, got ${a.critFrom}`);
  }
  const rolls = distribution(a.hitDice);
  const pCrit = (21 - a.critFrom) / 20;
  let pHit = 0;
  let pNear = 0;
  for (const [v, p] of rolls) {
    const total = v + a.accuracy;
    if (total >= avoidance) {
      pHit += p;
    } else if (total + a.nearMiss >= avoidance) {
      pNear += p;
    }
  }
  let crit = pHit * pCrit;
  let normal = pHit * (1 - pCrit) + pNear * pCrit;

  // The all-ones and all-max hit dice each come up once in prod(sides) throws; a crit die
  // of 1 never crits and a 20 always does.
  let sides = 1;
  let ones = a.hitDice.modifier + a.accuracy;
  let maxes = ones;
  for (const t of a.hitDice.terms) {
    sides *= t.sides ** t.count;
    ones += (t.negative ? -1 : 1) * t.count;
    maxes += (t.negative ? -1 : 1) * t.count * t.sides;
  }
  const extreme = 1 / sides / 20;
  if (ones >= avoidance) {
    normal -= extreme;
  }
  if (maxes < avoidance) {
    if (maxes + a.nearMiss >= avoidance) {
      normal -= extreme;
    }
    crit += extreme;
  }

  const damage: Distribution = new Map();
  add(damage, 0, 1 - crit - normal);
  add(damage, a.damage, normal);
  add(damage, a.damage * a.critMultiplier, crit);
  return {
    hit: crit + normal,
    crit,
    nearMissHit: pNear * pCrit,
    criticalMiss: extreme,
    specialCrit: extreme,
    expectedDamage: expected(damage),
    damage,
  };
}

/** An attack on a target that cannot defend: it always hits, and only the crit die is rolled. */
export function sureHit(a: HeroAttacker): HeroAttackOdds {
  const crit = (21 - a.critFrom) / 20;
  const damage: Distribution = new Map();
  add(damage, a.damage, 1 - crit);
  add(damage, a.damage * a.critMultiplier, crit);
  return { hit: 1, crit, nearMissHit: 0, criticalMiss: 0, specialCrit: 0, expectedDamage: expected(damage), damage };
}

/** Rules that change over a run of attacks on one monster. */
export interface KillRules {
  /** Accuracy added per miss in a row (Determination), up to determinationCap. */
  determinationStep: number;
  determinationCap: number;
  /** Body at or below which the monster falters (0 = never). */
  falterAt: number;
  /** Avoidance lost while faltering. */
  falterPenalty: number;
}

/** Determination +2 per miss in a row up to +4; set falterAt per monster. */
export const CAMPAIGN_RULES: KillRules = { determinationStep: 2, determinationCap: 4, falterAt: 0, falterPenalty: 4 };

export interface KillOdds {
  expected: number;
  /** byAttack[i]: the chance the monster dies on attack i + 1. */
  byAttack: number[];
}

const MAX_ATTACKS = 1000;

/** Exact odds of how many attacks one hero needs to bring body to 0; startBonus is a Determination bonus the hero already has. */
export function killOdds(a: HeroAttacker, avoidance: number, body: number, rules: KillRules, startBonus = 0): KillOdds {
  const steps = rules.determinationStep > 0 ? Math.ceil(rules.determinationCap / rules.determinationStep) : 0;
  const startStreak = rules.determinationStep > 0 ? Math.min(steps, Math.ceil(startBonus / rules.determinationStep)) : 0;
  const cache = new Map<string, [number, number][]>();
  const outcomes = (hp: number, streak: number): [number, number][] => {
    const bonus = Math.min(rules.determinationCap, streak * rules.determinationStep);
    const av = rules.falterAt > 0 && hp <= rules.falterAt ? avoidance - rules.falterPenalty : avoidance;
    const key = `${av}:${bonus}`;
    let o = cache.get(key);
    if (!o) {
      o = [...heroAttack({ ...a, accuracy: a.accuracy + bonus }, av).damage];
      cache.set(key, o);
    }
    return o;
  };
  // Alive states: "hp:streak" -> probability.
  let alive = new Map<string, [number, number, number]>([[`${body}:${String(startStreak)}`, [body, startStreak, 1]]]);
  const byAttack: number[] = [];
  let expectedAttacks = 0;
  let left = 1;
  for (let n = 1; n <= MAX_ATTACKS && left > 1e-12; n++) {
    const next = new Map<string, [number, number, number]>();
    let killed = 0;
    for (const [hp, streak, p] of alive.values()) {
      for (const [x, q] of outcomes(hp, streak)) {
        const mass = p * q;
        if (x <= 0) {
          const s = Math.min(steps, streak + 1);
          bump(next, hp, s, mass);
        } else if (hp - x <= 0) {
          killed += mass;
        } else {
          bump(next, hp - x, 0, mass);
        }
      }
    }
    byAttack.push(killed);
    expectedAttacks += n * killed;
    left -= killed;
    alive = next;
  }
  return { expected: left > 1e-9 ? Infinity : expectedAttacks, byAttack };
}

/** The attack number by which at least share q of kills are done. */
export function percentile(byAttack: number[], q: number): number {
  let total = 0;
  for (let i = 0; i < byAttack.length; i++) {
    total += byAttack[i] ?? 0;
    if (total >= q - 1e-12) {
      return i + 1;
    }
  }
  return Infinity;
}

function bump(m: Map<string, [number, number, number]>, hp: number, streak: number, p: number): void {
  const key = `${hp}:${streak}`;
  const cur = m.get(key);
  if (cur) {
    cur[2] += p;
  } else {
    m.set(key, [hp, streak, p]);
  }
}

export interface MonsterAttacker {
  hitDice: DiceExpr;
  damage: number;
}

export interface HeroDefender {
  defenseDice: DiceExpr;
  baseAvoidance: number;
  mitigation: number;
}

export interface MonsterAttackOdds {
  /** The monster's roll beats the hero's total (damage may still be mitigated to 0). */
  hit: number;
  /** A hit with double 20s on the crit check: damage doubles before mitigation. */
  crit: number;
  expectedDamage: number;
  damage: Distribution;
}

/** One monster attack against a hero's opposed defense roll; ties go to the hero. */
export function monsterAttack(m: MonsterAttacker, h: HeroDefender): MonsterAttackOdds {
  const attack = distribution(m.hitDice);
  const defense = distribution(h.defenseDice);
  let hit = 0;
  for (const [a, pa] of attack) {
    for (const [d, pd] of defense) {
      if (a > d + h.baseAvoidance) {
        hit += pa * pd;
      }
    }
  }
  const crit = hit * MONSTER_CRIT_CHANCE;
  const damage: Distribution = new Map();
  add(damage, 0, 1 - hit);
  add(damage, Math.max(0, m.damage - h.mitigation), hit - crit);
  add(damage, Math.max(0, 2 * m.damage - h.mitigation), crit);
  return { hit, crit, expectedDamage: expected(damage), damage };
}

/** Expected attacks to take body to 0, given the damage of one attack. */
export function attacksToKill(damage: Distribution, body: number): number {
  const miss = damage.get(0) ?? 0;
  if (miss >= 1) {
    return Infinity;
  }
  const memo = new Map<number, number>();
  const left = (hp: number): number => {
    if (hp <= 0) {
      return 0;
    }
    const known = memo.get(hp);
    if (known !== undefined) {
      return known;
    }
    let sum = 1;
    for (const [x, p] of damage) {
      if (x > 0) {
        sum += p * left(hp - x);
      }
    }
    const e = sum / (1 - miss);
    memo.set(hp, e);
    return e;
  };
  return left(body);
}

/**
 * Damage from one attack in original HeroQuest: skulls (1/2 per die) minus the defender's
 * shields. Monsters block on the black shield (1/6), heroes on the two white shields (1/3).
 */
export function heroQuestDamage(attackDice: number, defendDice: number, defender: 'hero' | 'monster'): Distribution {
  const skulls = binomial(attackDice, 1 / 2);
  const shields = binomial(defendDice, defender === 'hero' ? 1 / 3 : 1 / 6);
  const d: Distribution = new Map();
  skulls.forEach((ps, s) => {
    shields.forEach((pb, b) => {
      add(d, Math.max(0, s - b), ps * pb);
    });
  });
  return d;
}

function binomial(n: number, p: number): number[] {
  const out: number[] = [];
  let choose = 1;
  for (let k = 0; k <= n; k++) {
    out.push(choose * p ** k * (1 - p) ** (n - k));
    choose = (choose * (n - k)) / (k + 1);
  }
  return out;
}

function expected(d: Distribution): number {
  let e = 0;
  for (const [x, p] of d) {
    e += x * p;
  }
  return e;
}

function add(d: Distribution, key: number, p: number): void {
  if (p !== 0) {
    d.set(key, (d.get(key) ?? 0) + p);
  }
}
