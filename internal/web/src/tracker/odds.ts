/**
 * Odds hints for the tracker (The Three Plagues rules): exact chances from
 * the combat calculator for a hero attacking a monster and a monster
 * attacking a hero. Advice only; nothing here changes or refuses anything.
 * Counted: class stats plus equipped items, the monster's current Body and
 * Faltering, Determination over a run of misses. Not counted: effects,
 * abilities and spells.
 */
import { CAMPAIGN_RULES, heroAttack, killOdds, monsterAttack } from '../combat/odds.ts';
import { parseDice, type DiceExpr } from '../dice/dice.ts';
import { combatTotals, FALTER_PENALTY, falterAt } from './combat.ts';
import type { Hero, Monster } from './types.ts';

/** A crit doubles damage; a crit on a miss adds this to the hit total (RULES_AND_CLASSES.md). */
const CRIT_MULTIPLIER = 2;
const NEAR_MISS = 2;

export interface AttackOdds {
  /** Any damage dealt by one attack (near misses that land included). */
  hit: number;
  crit: number;
  expectedDamage: number;
  /** Expected attacks by this hero alone to bring the monster's current Body to 0. */
  attacksToKill: number;
  /** The monster is at or below a quarter of its Body: Avoidance is lower. */
  faltering: boolean;
}

export interface DefenseOdds {
  /** The monster's roll beats the hero's defense (the damage may still be 0). */
  hit: number;
  /** Damage per hit after the hero's mitigation (a crit doubles it first). */
  damage: number;
  expectedDamage: number;
}

function dice(s: string): DiceExpr | null {
  const r = parseDice(s);
  return r.ok ? r.expr : null;
}

function fighting(hero: Hero): boolean {
  return hero.status === 'active';
}

/** One hero attacking a monster; null without combat stats on both sides. */
export function attackOdds(hero: Hero, monster: Monster): AttackOdds | null {
  const c = combatTotals(hero);
  const m = monster.combat;
  if (!c || !m || !fighting(hero) || !monster.alive || monster.body <= 0) {
    return null;
  }
  const hitDice = dice(c.hitDice);
  if (!hitDice) {
    return null;
  }
  const attacker = { hitDice, accuracy: c.accuracy, critFrom: c.critFrom, critMultiplier: CRIT_MULTIPLIER, nearMiss: NEAR_MISS, damage: c.damage };
  const falterBody = falterAt(monster.maxBody);
  const faltering = monster.body <= falterBody;
  const one = heroAttack(attacker, faltering ? m.avoidance - FALTER_PENALTY : m.avoidance);
  const kill = killOdds(attacker, m.avoidance, monster.body, { ...CAMPAIGN_RULES, falterAt: falterBody, falterPenalty: FALTER_PENALTY });
  return { hit: one.hit, crit: one.crit, expectedDamage: one.expectedDamage, attacksToKill: kill.expected, faltering };
}

/** A monster attacking one hero; null without combat stats on both sides. */
export function defenseOdds(monster: Monster, hero: Hero): DefenseOdds | null {
  const c = combatTotals(hero);
  const m = monster.combat;
  if (!c || !m || !fighting(hero) || !monster.alive) {
    return null;
  }
  const hitDice = dice(m.hitDice);
  const defenseDice = dice(c.defenseDice);
  if (!hitDice || !defenseDice) {
    return null;
  }
  const odds = monsterAttack({ hitDice, damage: m.damage }, { defenseDice, baseAvoidance: c.avoidance, mitigation: c.mitigation });
  return { hit: odds.hit, damage: Math.max(0, m.damage - c.mitigation), expectedDamage: odds.expectedDamage };
}

/** A chance as a whole percent: "43%", with "<1%" and ">99%" for chances that round away. */
export function oddsPercent(p: number): string {
  if (p > 0 && p < 0.005) {
    return '<1%';
  }
  if (p < 1 && p >= 0.995) {
    return '>99%';
  }
  return `${String(Math.round(p * 100))}%`;
}
