/**
 * The Three Plagues combat numbers for scripts/combat-odds.ts and
 * scripts/combat-sim.ts. Class bases, the starting kit and the monsters come
 * from docs/campaigns/three-plagues/combat.json, which `make fill-campaign`
 * also loads into the app; the Quest 1 finds, supplies and the simulated
 * spells live here (see RULES_AND_CLASSES.md, "Combat").
 */

import combat from '../docs/campaigns/three-plagues/combat.json';
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

const CLASSES: Record<string, HeroSpec['cls']> = { Barbarian: 'barbarian', Ranger: 'ranger', Rogue: 'rogue', Cleric: 'cleric' };

/** Class base stats (combat.json), backfilled so the starting kit gives the step 2 totals (2026-10-04). */
const BASES: HeroBase[] = combat.classes.map((c): HeroBase => {
  const cls = CLASSES[c.name];
  if (!cls) {
    throw new Error(`combat.json: the simulator has no class ${c.name}`);
  }
  const { name, body, hitDice, accuracy, critFrom, damage, avoidance, defenseDice, mitigation } = c;
  return {
    name, cls, body, hitDice, accuracy, critFrom, damage, avoidance, defenseDice, mitigation,
    ...('mana' in c ? { mana: c.mana } : {}),
    ...('manaRegen' in c ? { manaRegen: c.manaRegen } : {}),
  };
});

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

/** Each hero's starting gear (combat.json, 2026-10-04). */
export const STARTING_KIT: Item[] = combat.startingKit;

/** A loot list entry in combat.json: an item, or a potion (healBody / restoreMana). */
interface LootLine extends Partial<Item> {
  name: string;
  label?: string;
  after?: string;
  healBody?: number;
  restoreMana?: number;
}

const LOOT = (combat as unknown as { loot: LootLine[] }).loot;

/** Quest 1 finds (approved 2026-10-05; combat.json "loot"), keyed to the board notes, found after these encounters. */
export const QUEST_1_FINDS: { label: string; after: string; item: Item }[] = LOOT.filter(
  (l): l is LootLine & { hero: string; kind: string; label: string; after: string } => l.hero !== undefined && l.label !== undefined && l.after !== undefined && l.kind !== undefined,
).map((l) => ({
  label: l.label,
  after: l.after,
  item: {
    name: l.name,
    hero: l.hero,
    kind: l.kind,
    ...(l.damage ? { damage: l.damage } : {}),
    ...(l.accuracy ? { accuracy: l.accuracy } : {}),
    ...(l.avoidance ? { avoidance: l.avoidance } : {}),
    ...(l.mitigation ? { mitigation: l.mitigation } : {}),
    ...(l.mana ? { mana: l.mana } : {}),
    ...(l.manaRegen ? { manaRegen: l.manaRegen } : {}),
    ...(l.replaces ? { replaces: l.replaces } : {}),
    ...(l.note ? { note: l.note } : {}),
  },
}));

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

/** A monster's stat line as combat.json (and the app's campaign monster stats) keep it. */
interface MonsterLine {
  body: number;
  /** Missing on a body-only line (a monster that doesn't fight, e.g. the Stranger). */
  avoidance?: number;
  hitDice?: string;
  damage?: number;
  ranged?: boolean;
  reach?: boolean;
  line?: number;
  splashDamage?: number;
  splashTargets?: number;
  undead?: boolean;
}

function monster(m: MonsterLine & { hitDice: string }): MonsterSpec {
  return {
    body: m.body,
    avoidance: m.avoidance ?? 0,
    attack: { hitDice: dice(m.hitDice), damage: m.damage ?? 0 },
    ...(m.ranged ? { ranged: true } : {}),
    ...(m.reach ? { reach: true } : {}),
    ...(m.line ? { line: m.line } : {}),
    ...(m.splashDamage && m.splashTargets ? { splash: { damage: m.splashDamage, targets: m.splashTargets } } : {}),
    ...(m.undead ? { undead: true } : {}),
  };
}

/**
 * Monster numbers (combat.json). Third calibration 2026-10-04, for the step 4 kit: non-fodder
 * Body x1.45 over the second calibration (heroes hold doorways: 2 melee attackers a round).
 * Fodder keeps the original feel (goblins die to any hit); mid and elite monsters get
 * relatively more Body than the original ratios, so four heroes can't fell them in one round.
 * The warlock's blast also hits 2 heroes beside the target; the gargoyle strikes 2 squares in
 * a straight line (the hero behind defends separately).
 */
export const MONSTERS: Record<string, MonsterSpec> = Object.fromEntries(
  Object.entries(combat.monsters as Record<string, MonsterLine>)
    .filter((e): e is [string, MonsterLine & { hitDice: string }] => e[1].hitDice !== undefined)
    .map(([type, m]) => [type, monster(m)]),
);

const CLERIC_MIND = combat.classes.find((c) => c.name === 'Cleric')?.mind ?? 0;

/** The agreed ability kit (step 4) with draft numbers; see the simulator's DEFAULT_TACTICS. */
export const TACTICS: Tactics = {
  ...DEFAULT_TACTICS,
  // After a door opens, about half the party can reach the monsters in the first round.
  openingAttackers: 2,
  // The heroes hold a doorway: 1-2 melee monsters reach them a round.
  meleeLimit: 2,
  // Smite rolls 1d20 + the Cleric's Mind (Will) to hit, not the weapon's hit dice and Accuracy (2026-10-05).
  smite: { cost: 2, attack: { hitDice: dice('1d20'), accuracy: CLERIC_MIND, damage: 5 }, undead: 2 },
  heal: { cost: 6, amount: 12 },
};

/** Abilities being tested, switched on with --with=name (none at the moment). */
export const TESTING: Partial<Tactics> = {};

/** Quest 1 consumables (note V): 3 healing potions and 1 mana potion, their amounts from the loot list; drinking is free. */
export const SUPPLIES: Supplies = {
  healPotions: 3,
  healAmount: LOOT.find((l) => l.name === 'Healing Potion')?.healBody ?? 0,
  manaPotions: 1,
  manaAmount: LOOT.find((l) => l.name === 'Mana Potion')?.restoreMana ?? 0,
};

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
