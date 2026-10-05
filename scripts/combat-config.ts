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

/** A hero's own numbers, before gear (Tough as Nails counts in the Barbarian's mitigation). */
interface HeroBase {
  name: string;
  cls: HeroSpec['cls'];
  body: number;
  hitDice: string;
  accuracy: number;
  critFrom: number;
  damage: number;
  avoidance: number;
  defenseDice: string;
  mitigation: number;
  mana?: number;
  manaRegen?: number;
}

/** Class base stats, backfilled so the starting kit gives the step 2 totals (2026-10-04). */
const BASES: HeroBase[] = [
  { name: 'Barbarian', cls: 'barbarian', body: 40, hitDice: '1d20', accuracy: 3, critFrom: 17, damage: 3, avoidance: 2, defenseDice: '1d6', mitigation: 1 },
  { name: 'Ranger', cls: 'ranger', body: 30, hitDice: '2d10', accuracy: 4, critFrom: 18, damage: 2, avoidance: 4, defenseDice: '1d6', mitigation: 0 },
  { name: 'Rogue', cls: 'rogue', body: 28, hitDice: '2d10', accuracy: 2, critFrom: 15, damage: 1, avoidance: 4, defenseDice: '1d6', mitigation: 0 },
  { name: 'Cleric', cls: 'cleric', body: 28, hitDice: '2d8', accuracy: 4, critFrom: 20, damage: 2, avoidance: 3, defenseDice: '1d6', mitigation: 0, mana: 16, manaRegen: 2 },
];

export interface Item {
  name: string;
  hero: string;
  /** What the item is (weapon, chest, feet, hands, off-hand, trinket); no fixed slot list. */
  kind: string;
  damage?: number;
  accuracy?: number;
  avoidance?: number;
  mitigation?: number;
  mana?: number;
  manaRegen?: number;
  /** The starting item this one replaces when found. */
  replaces?: string;
  note?: string;
}

/** Each hero's starting gear (2026-10-04). */
export const STARTING_KIT: Item[] = [
  { name: 'Greataxe', hero: 'Barbarian', kind: 'two-handed weapon', damage: 7 },
  { name: 'Hide Cuirass', hero: 'Barbarian', kind: 'chest, heavy (Barbarian-only)', mitigation: 1 },
  { name: 'Iron-shod Boots', hero: 'Barbarian', kind: 'feet', avoidance: 1 },
  { name: 'Sword', hero: 'Rogue', kind: 'weapon', damage: 3 },
  { name: 'Dirk', hero: 'Rogue', kind: 'weapon', damage: 2 },
  { name: 'Leather Jerkin', hero: 'Rogue', kind: 'chest', avoidance: 2 },
  { name: 'Soft Boots', hero: 'Rogue', kind: 'feet', avoidance: 1 },
  { name: 'Hunting Bow', hero: 'Ranger', kind: 'bow', damage: 5 },
  { name: 'Leather Armor', hero: 'Ranger', kind: 'chest', avoidance: 2 },
  { name: "Archer's Gloves", hero: 'Ranger', kind: 'hands', accuracy: 1 },
  { name: 'Mace', hero: 'Cleric', kind: 'weapon', damage: 3 },
  { name: 'Padded Robes', hero: 'Cleric', kind: 'chest', avoidance: 1 },
  { name: 'Holy Tome', hero: 'Cleric', kind: 'off-hand', manaRegen: 1 },
];

/** Quest 1 finds (approved 2026-10-05), keyed to the board notes, found after these encounters. */
export const QUEST_1_FINDS: { label: string; after: string; item: Item }[] = [
  { label: 'W', after: 'Room 5', item: { name: "Wardens' Longbow", hero: 'Ranger', kind: 'bow', damage: 6, replaces: 'Hunting Bow' } },
  { label: 'S', after: 'Room 5', item: { name: "Wardens' Dirk", hero: 'Rogue', kind: 'weapon', damage: 3, replaces: 'Dirk' } },
  { label: 'S', after: 'Room 5', item: { name: "Wardens' Chain Shirt", hero: 'Cleric', kind: 'chest', avoidance: 2, replaces: 'Padded Robes' } },
  { label: 'O', after: 'Room 11', item: { name: "Wardens' Greatsword", hero: 'Barbarian', kind: 'two-handed weapon', damage: 9, replaces: 'Greataxe' } },
  { label: 'X', after: 'Room 17', item: { name: 'Quivering Boots', hero: 'Rogue', kind: 'feet', avoidance: 2, replaces: 'Soft Boots', note: 'special: traps (see the Rogue)' } },
  { label: 'T', after: 'Corridor at (29,5)', item: { name: "Wardens' Scale Hauberk", hero: 'Barbarian', kind: 'chest, heavy (Barbarian-only)', mitigation: 2, replaces: 'Hide Cuirass' } },
  { label: 'L', after: 'Corridor at (15,24)', item: { name: "Pilgrim's Prayer Beads", hero: 'Cleric', kind: 'trinket', mana: 2, note: "left by Sister Wenna's pilgrims" } },
];

/** Adds an item's stats to a hero (sign -1 takes them away). */
export function withItem(h: HeroSpec, item: Item, sign = 1): HeroSpec {
  const mana = (h.mana ?? 0) + sign * (item.mana ?? 0);
  const regen = (h.manaRegen ?? 0) + sign * (item.manaRegen ?? 0);
  return {
    ...h,
    attack: { ...h.attack, damage: h.attack.damage + sign * (item.damage ?? 0), accuracy: h.attack.accuracy + sign * (item.accuracy ?? 0) },
    defense: { ...h.defense, baseAvoidance: h.defense.baseAvoidance + sign * (item.avoidance ?? 0), mitigation: h.defense.mitigation + sign * (item.mitigation ?? 0) },
    ...(mana > 0 ? { mana } : {}),
    ...(regen > 0 ? { manaRegen: regen } : {}),
  };
}

/** A found item: swaps out the starting item it replaces. */
export function upgrade(h: HeroSpec, item: Item): HeroSpec {
  const old = STARTING_KIT.find((k) => k.name === item.replaces);
  return withItem(old ? withItem(h, old, -1) : h, item);
}

function fromBase(b: HeroBase): HeroSpec {
  return {
    name: b.name,
    cls: b.cls,
    body: b.body,
    attack: { hitDice: dice(b.hitDice), accuracy: b.accuracy, critFrom: b.critFrom, critMultiplier: 2, nearMiss: NEAR_MISS, damage: b.damage },
    defense: { defenseDice: dice(b.defenseDice), baseAvoidance: b.avoidance, mitigation: b.mitigation },
    ...(b.mana !== undefined ? { mana: b.mana } : {}),
    ...(b.manaRegen !== undefined ? { manaRegen: b.manaRegen } : {}),
  };
}

/** The party as it enters Quest 1: class base + starting kit. */
export const PARTY: HeroSpec[] = BASES.map((b) => STARTING_KIT.filter((i) => i.hero === b.name).reduce((h, i) => withItem(h, i), fromBase(b)));

/** The party with every Quest 1 find (how it may enter Quest 2). */
export const PARTY_AFTER_QUEST_1: HeroSpec[] = PARTY.map((h) => QUEST_1_FINDS.filter((f) => f.item.hero === h.name).reduce((s, f) => upgrade(s, f.item), h));

/** Hero attacks with the starting kit. */
export const HERO_ATTACKS: Record<string, HeroAttacker> = Object.fromEntries(PARTY.map((h) => [h.name, h.attack]));

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

/** Quest 1 consumables (note V): 3 healing potions (8 Body) and 1 mana potion (6 mana); drinking is free. */
export const SUPPLIES: Supplies = { healPotions: 3, healAmount: 8, manaPotions: 1, manaAmount: 6 };

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
