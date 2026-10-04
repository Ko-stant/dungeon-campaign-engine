/**
 * Proposed and draft numbers for The Three Plagues combat, shared by
 * scripts/combat-odds.ts and scripts/combat-sim.ts. Attack numbers are the
 * step 2 proposal; defense, monsters, supplies and the Cleric's spells are
 * drafts to tune with the simulator (see RULES_AND_CLASSES.md, "Combat").
 */

import { parseDice, type DiceExpr } from '../internal/web/src/dice/dice.ts';
import type { HeroAttacker } from '../internal/web/src/combat/odds.ts';
import { DEFAULT_TACTICS, type HeroSpec, type MonsterSpec, type Supplies, type Tactics } from '../internal/web/src/combat/simulate.ts';

export function dice(s: string): DiceExpr {
  const r = parseDice(s);
  if (!r.ok) {
    throw new Error(r.error);
  }
  return r.expr;
}

export const NEAR_MISS = 2;

const attack = (hitDice: string, accuracy: number, critFrom: number, damage: number): HeroAttacker => ({
  hitDice: dice(hitDice),
  accuracy,
  critFrom,
  critMultiplier: 2,
  nearMiss: NEAR_MISS,
  damage,
});

/** Step 2 proposal: hero attacks with the starting kit (2026-10-04). */
export const HERO_ATTACKS: Record<string, HeroAttacker> = {
  Barbarian: attack('1d20', 3, 17, 10),
  Ranger: attack('2d10', 5, 18, 7),
  Rogue: attack('2d10', 2, 15, 6),
  Cleric: attack('2d8', 4, 20, 5),
};

function heroAttack(name: string): HeroAttacker {
  const a = HERO_ATTACKS[name];
  if (!a) {
    throw new Error(`no attack for ${name}`);
  }
  return a;
}

/** Draft: Body and defense (the Barbarian's mitigation includes Tough as Nails). */
export const PARTY: HeroSpec[] = [
  { name: 'Barbarian', cls: 'barbarian', body: 40, attack: heroAttack('Barbarian'), defense: { defenseDice: dice('1d6'), baseAvoidance: 3, mitigation: 2 } },
  { name: 'Ranger', cls: 'ranger', body: 30, attack: heroAttack('Ranger'), defense: { defenseDice: dice('1d6'), baseAvoidance: 6, mitigation: 0 } },
  { name: 'Rogue', cls: 'rogue', body: 28, attack: heroAttack('Rogue'), defense: { defenseDice: dice('1d6'), baseAvoidance: 7, mitigation: 0 } },
  {
    name: 'Cleric',
    cls: 'cleric',
    body: 28,
    attack: heroAttack('Cleric'),
    defense: { defenseDice: dice('1d6'), baseAvoidance: 4, mitigation: 0 },
    mana: 12,
    // 2 per round, +1 from the holy tome.
    manaRegen: 3,
  },
];

const monster = (body: number, avoidance: number, hitDice: string, damage: number, more: Partial<MonsterSpec> = {}): MonsterSpec => ({
  body,
  avoidance,
  attack: { hitDice: dice(hitDice), damage },
  ...more,
});

/**
 * Draft monster numbers (step 3, second calibration 2026-10-04: heroes hold doorways, so
 * 2 melee attackers a round; non-fodder Body x1.25 and damage x1.5 over the first draft).
 * Fodder keeps the original feel (goblins die to any hit); mid and elite monsters get
 * relatively more Body than the original ratios, so four heroes can't fell them in one round.
 */
export const MONSTERS: Record<string, MonsterSpec> = {
  goblin: monster(5, 6, '1d12', 4),
  goblin_archer: monster(5, 6, '1d12', 4, { ranged: true }),
  // Proposed: the warlock's blast also hits 2 heroes beside the target for light damage.
  goblin_warlock: monster(5, 6, '1d12', 5, { ranged: true, splash: { damage: 2, targets: 2 } }),
  orc: monster(15, 8, '2d8', 9),
  orc_archer: monster(15, 8, '2d8', 9, { ranged: true }),
  skeleton: monster(12, 8, '1d12', 6),
  zombie: monster(15, 10, '1d12+1', 8),
  abomination: monster(38, 10, '2d8+1', 10),
  mummy: monster(44, 12, '2d8+1', 10),
  dread_warrior: monster(75, 12, '2d10+2', 15),
  // Proposed: strikes 2 squares in a straight line (the hero behind defends separately).
  gargoyle: monster(75, 14, '2d10+3', 12, { line: 1 }),
  specter: monster(75, 14, '2d10+2', 15),
};

/** Draft spell numbers; abilities keep the simulator's provisional defaults until step 4. */
export const TACTICS: Tactics = {
  ...DEFAULT_TACTICS,
  // After a door opens, about half the party can reach the monsters in the first round.
  openingAttackers: 2,
  // The heroes hold a doorway: 1-2 melee monsters reach them a round.
  meleeLimit: 2,
  smite: { cost: 2, attack: { hitDice: dice('1d20'), accuracy: 5, damage: 6 } },
  heal: { cost: 6, amount: 12 },
};

/** Quest 1 consumables (note V): 3 healing potions and 1 mana potion. */
export const SUPPLIES: Supplies = { healPotions: 3, healAmount: 10, manaPotions: 1, manaAmount: 6 };

/** Quest 1 adjustments to the generated encounters. */
export const QUEST_1 = {
  file: 'docs/campaigns/three-plagues/sim/crumbling-halls.json',
  /** The rooms every party meets first, in order; the rest come in a random order each run. */
  route: ['Room 19', 'Room 20', 'Room 4', 'Room 5'],
  /** The pool (note N) heals fully once, after this encounter. */
  poolAfter: 'Room 9',
  /** Monsters left out by default: the Specter only appears if the Stranger is never freed. */
  without: ['specter'],
};
