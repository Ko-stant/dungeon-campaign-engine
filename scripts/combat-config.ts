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
    mana: 16,
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
 * Draft monster numbers (step 3). Third calibration 2026-10-04, for the step 4 kit: non-fodder
 * Body x1.45 over the second calibration (heroes hold doorways: 2 melee attackers a round).
 * Fodder keeps the original feel (goblins die to any hit); mid and elite monsters get
 * relatively more Body than the original ratios, so four heroes can't fell them in one round.
 */
export const MONSTERS: Record<string, MonsterSpec> = {
  goblin: monster(5, 6, '1d12', 4),
  goblin_archer: monster(5, 6, '1d12', 4, { ranged: true }),
  // Proposed: the warlock's blast also hits 2 heroes beside the target for light damage.
  goblin_warlock: monster(5, 6, '1d12', 5, { ranged: true, splash: { damage: 2, targets: 2 } }),
  orc: monster(22, 8, '2d8', 9),
  orc_archer: monster(22, 8, '2d8', 9, { ranged: true }),
  skeleton: monster(17, 8, '1d12', 6, { undead: true }),
  zombie: monster(22, 10, '1d12+1', 8, { undead: true }),
  abomination: monster(55, 10, '2d8+1', 10),
  mummy: monster(64, 12, '2d8+1', 10, { undead: true }),
  dread_warrior: monster(109, 12, '2d10+2', 15),
  // Proposed: strikes 2 squares in a straight line (the hero behind defends separately).
  gargoyle: monster(109, 14, '2d10+3', 12, { line: 1 }),
  specter: monster(109, 14, '2d10+2', 15, { undead: true }),
};

/** The agreed ability kit (step 4) with draft numbers; see the simulator's DEFAULT_TACTICS. */
export const TACTICS: Tactics = {
  ...DEFAULT_TACTICS,
  // After a door opens, about half the party can reach the monsters in the first round.
  openingAttackers: 2,
  // The heroes hold a doorway: 1-2 melee monsters reach them a round.
  meleeLimit: 2,
  smite: { cost: 2, attack: { hitDice: dice('1d20'), accuracy: 5, damage: 5 }, undead: 2 },
  heal: { cost: 6, amount: 12 },
};

/** Abilities being tested, switched on with --with=name (none at the moment). */
export const TESTING: Partial<Tactics> = {};

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
