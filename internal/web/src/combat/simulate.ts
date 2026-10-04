/**
 * Monte Carlo combat simulator for The Three Plagues house rules (see
 * docs/campaigns/three-plagues/RULES_AND_CLASSES.md, "Combat"). A design aid: it
 * plays a party through a quest's encounters with simple tactics, so hero and
 * monster numbers can be tuned against survival targets. Positioning, movement,
 * traps and doors are not modeled; a cap on melee attackers per round stands in
 * for heroes holding a doorway.
 */

import { rollDice } from '../dice/dice.ts';
import type { HeroAttacker, HeroDefender, MonsterAttacker } from './odds.ts';

export interface Roller {
  /** A whole number from 1 to sides. */
  die(sides: number): number;
}

/** A repeatable roller (mulberry32), so runs and tests can be replayed. */
export function seededRoller(seed: number): Roller {
  let a = seed >>> 0;
  return {
    die(sides: number): number {
      a = (a + 0x6d2b79f5) >>> 0;
      let t = a;
      t = Math.imul(t ^ (t >>> 15), t | 1);
      t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
      return 1 + Math.floor((((t ^ (t >>> 14)) >>> 0) / 4294967296) * sides);
    },
  };
}

export interface StrikeOptions {
  /** Added to the hit total (Determination, Holy Blessing, or a penalty when negative). */
  bonus?: number;
  /** Roll the hit dice twice and keep the better total (Echoing Roar). */
  advantage?: boolean;
  /** The target cannot defend, or the shot cannot miss: only the crit die is rolled. */
  sure?: boolean;
  /** Added to damage before any factor (Unburdened Charge). */
  extraDamage?: number;
  /** Multiplies damage (Rain of Arrows arrows). */
  damageFactor?: number;
}

export interface StrikeResult {
  damage: number;
  hit: boolean;
  crit: boolean;
  criticalMiss: boolean;
}

/** One hero attack. Dice are rolled in order: hit dice (twice with advantage), then the crit die. */
export function heroStrike(a: HeroAttacker, avoidance: number, roller: Roller, o: StrikeOptions = {}): StrikeResult {
  const damage = (a.damage + (o.extraDamage ?? 0)) * (o.damageFactor ?? 1);
  const die = (sides: number): number => roller.die(sides);
  if (o.sure) {
    const crit = roller.die(20) >= a.critFrom;
    return { damage: crit ? damage * a.critMultiplier : damage, hit: true, crit, criticalMiss: false };
  }
  let roll = rollDice(a.hitDice, die);
  if (o.advantage) {
    const second = rollDice(a.hitDice, die);
    if (second.total > roll.total) {
      roll = second;
    }
  }
  const critDie = roller.die(20);
  if (critDie === 1 && roll.rolls.every((r) => r.value === 1)) {
    return { damage: 0, hit: false, crit: false, criticalMiss: true };
  }
  const crit = critDie >= a.critFrom;
  const special = critDie === 20 && roll.rolls.every((r) => r.value === r.sides);
  const total = roll.total + a.accuracy + (o.bonus ?? 0);
  if (special || (crit && total >= avoidance)) {
    return { damage: damage * a.critMultiplier, hit: true, crit: true, criticalMiss: false };
  }
  if (total >= avoidance || (crit && total + a.nearMiss >= avoidance)) {
    return { damage, hit: true, crit: false, criticalMiss: false };
  }
  return { damage: 0, hit: false, crit: false, criticalMiss: false };
}

/**
 * One monster attack against a hero's opposed roll; ties go to the hero. Dice are rolled
 * in order: monster hit dice, hero defense dice (unless the hero cannot defend), then the
 * 2d20 crit check (on a hit only).
 */
export function monsterStrike(m: MonsterAttacker, d: HeroDefender, roller: Roller, canDefend = true): { damage: number; hit: boolean; crit: boolean } {
  const die = (sides: number): number => roller.die(sides);
  const attack = rollDice(m.hitDice, die).total;
  const defense = d.baseAvoidance + (canDefend ? rollDice(d.defenseDice, die).total : 0);
  if (attack <= defense) {
    return { damage: 0, hit: false, crit: false };
  }
  const first = roller.die(20);
  const second = roller.die(20);
  const crit = first === 20 && second === 20;
  return { damage: Math.max(0, (crit ? 2 : 1) * m.damage - d.mitigation), hit: true, crit };
}

/** Rounds left on a cooldown after time out of a fight: short ones stop at 1, long ones at 2. */
export function cooldownFloor(remaining: number, cooldown: number): number {
  return Math.min(remaining, cooldown <= 3 ? 1 : 2);
}

export type HeroClass = 'barbarian' | 'ranger' | 'rogue' | 'cleric';

export interface HeroSpec {
  name: string;
  cls: HeroClass;
  body: number;
  attack: HeroAttacker;
  defense: HeroDefender;
  /** Maximum mana (spellcasters). */
  mana?: number;
  /** Mana regenerated at the start of each fight round. */
  manaRegen?: number;
}

export interface MonsterSpec {
  body: number;
  avoidance: number;
  attack: MonsterAttacker;
  /** Attacks from range: never crowded out by the melee cap. */
  ranged?: boolean;
  /** Each attack also blasts this many other heroes for this damage (no roll; mitigation applies). */
  splash?: { damage: number; targets: number };
  /** Each attack also strikes this many more heroes in a line; each defends separately. */
  line?: number;
  /** Strikes past the heroes holding a doorway: not counted in the melee limit. */
  reach?: boolean;
}

export interface Tactics {
  /** False plays attack-only: no abilities or spells (potions and the pool still count). */
  abilities: boolean;
  /** Most melee monsters that can attack in a round (1-2 when the heroes hold a doorway). */
  meleeLimit: number;
  /** Heroes (in party order) who can attack in a fight's first round; the rest are moving in. */
  openingAttackers: number;
  determinationStep: number;
  determinationCap: number;
  /** Share of maximum Body at or below which a monster falters (0 = never; at least 1 Body). */
  falterShare: number;
  falterPenalty: number;
  /** A hero at or below this share of Body drinks a healing potion. */
  potionAt: number;
  /** The Cleric heals an ally at or below this share of Body in a fight... */
  healAt: number;
  /** ...and up to this share between fights. */
  restHealAt: number;
  charge: { cooldown: number; extraDamage: number };
  roar: { cooldown: number; minAvoidance: number };
  fan: { cooldown: number; targets: number };
  vanish: { cooldown: number; below: number };
  aimed: { cooldown: number; minAvoidance: number };
  multi: { cooldown: number; targets: number };
  rain: { cooldown: number; targets: number; minFoes: number };
  /** Smite: the Cleric's attack with these changes. */
  smite: { cost: number; attack: Partial<HeroAttacker> };
  heal: { cost: number; amount: number };
  prayer: { cooldown: number };
}

/** Provisional ability numbers until step 4 (abilities) sets them. */
export const DEFAULT_TACTICS: Tactics = {
  abilities: true,
  meleeLimit: 2,
  openingAttackers: 4,
  determinationStep: 2,
  determinationCap: 4,
  falterShare: 0.25,
  falterPenalty: 4,
  potionAt: 0.25,
  healAt: 0.5,
  restHealAt: 0.75,
  charge: { cooldown: 3, extraDamage: 4 },
  roar: { cooldown: 5, minAvoidance: 12 },
  fan: { cooldown: 3, targets: 3 },
  vanish: { cooldown: 5, below: 0.5 },
  aimed: { cooldown: 5, minAvoidance: 12 },
  multi: { cooldown: 5, targets: 3 },
  rain: { cooldown: 7, targets: 4, minFoes: 3 },
  smite: { cost: 2, attack: { damage: 6 } },
  heal: { cost: 6, amount: 12 },
  prayer: { cooldown: 5 },
};

type AbilityName = 'charge' | 'roar' | 'fan' | 'vanish' | 'aimed' | 'multi' | 'rain' | 'prayer';

/** Party-wide consumables; the simulator uses them up. */
export interface Supplies {
  healPotions: number;
  healAmount: number;
  manaPotions: number;
  manaAmount: number;
}

export interface HeroState {
  spec: HeroSpec;
  body: number;
  maxBody: number;
  mana: number;
  maxMana: number;
  /** Rounds until each used ability is ready again (0 or missing = ready). */
  cooldowns: Partial<Record<AbilityName, number>>;
  /** Misses in a row (Determination). */
  streak: number;
  /** Loses the next turn (critical miss). */
  skip: boolean;
  /** Rolls no defense dice until their next turn (after Prayer). */
  undefended: boolean;
  /** Cannot be targeted for the rest of the round (Vanish From Sight). */
  vanished: boolean;
  alive: boolean;
}

export function newHero(spec: HeroSpec): HeroState {
  return {
    spec,
    body: spec.body,
    maxBody: spec.body,
    mana: spec.mana ?? 0,
    maxMana: spec.mana ?? 0,
    cooldowns: {},
    streak: 0,
    skip: false,
    undefended: false,
    vanished: false,
    alive: true,
  };
}

interface Foe {
  spec: MonsterSpec;
  body: number;
}

const ready = (h: HeroState, a: AbilityName): boolean => (h.cooldowns[a] ?? 0) <= 0;

/** Plays one fight to the end (or maxRounds). Heroes act first each round, in party order. */
export function runFight(heroes: HeroState[], monsters: MonsterSpec[], t: Tactics, supplies: Supplies, roller: Roller, maxRounds = 200): { won: boolean; rounds: number } {
  const foes: Foe[] = monsters.map((spec) => ({ spec, body: spec.body }));
  const liveFoes = (): Foe[] => foes.filter((f) => f.body > 0).sort((a, b) => a.body - b.body);
  const liveHeroes = (): HeroState[] => heroes.filter((h) => h.alive);
  const avoidance = (f: Foe): number => {
    const at = t.falterShare > 0 ? Math.max(1, Math.floor(f.spec.body * t.falterShare)) : 0;
    return f.body <= at ? f.spec.avoidance - t.falterPenalty : f.spec.avoidance;
  };

  const attack = (h: HeroState, targets: Foe[], o: StrikeOptions = {}, a: HeroAttacker = h.spec.attack): void => {
    const bonus = Math.min(t.determinationCap, h.streak * t.determinationStep) + (o.bonus ?? 0);
    let hit = false;
    for (const f of targets) {
      const r = heroStrike(a, avoidance(f), roller, { ...o, bonus });
      f.body -= r.damage;
      hit ||= r.hit;
      if (r.criticalMiss) {
        h.skip = true;
      }
    }
    h.streak = hit ? 0 : h.streak + 1;
  };
  const use = (h: HeroState, a: AbilityName): void => {
    h.cooldowns[a] = t[a].cooldown;
  };

  const clericTurn = (h: HeroState, target: Foe): void => {
    const hurt = liveHeroes()
      .filter((x) => x.body <= t.healAt * x.maxBody)
      .sort((a, b) => a.body / a.maxBody - b.body / b.maxBody);
    const patient = hurt[0];
    if (patient && h.mana >= t.heal.cost) {
      h.mana -= t.heal.cost;
      patient.body = Math.min(patient.maxBody, patient.body + t.heal.amount);
      return;
    }
    if (h.mana < t.smite.cost) {
      if (ready(h, 'prayer')) {
        use(h, 'prayer');
        h.mana = Math.min(h.maxMana, h.mana + Math.ceil(h.maxMana / 2));
        h.undefended = true;
        return;
      }
      if (supplies.manaPotions > 0) {
        supplies.manaPotions--;
        h.mana = Math.min(h.maxMana, h.mana + supplies.manaAmount);
        return;
      }
      attack(h, [target]);
      return;
    }
    h.mana -= t.smite.cost;
    attack(h, [target], {}, { ...h.spec.attack, ...t.smite.attack });
  };

  const heroTurn = (h: HeroState): void => {
    if (h.skip) {
      h.skip = false;
      return;
    }
    h.undefended = false;
    if (h.body <= t.potionAt * h.maxBody && supplies.healPotions > 0) {
      supplies.healPotions--;
      h.body = Math.min(h.maxBody, h.body + supplies.healAmount);
      return;
    }
    const targets = liveFoes();
    const target = targets[0];
    if (!target) {
      return;
    }
    if (!t.abilities) {
      attack(h, [target]);
      return;
    }
    switch (h.spec.cls) {
      case 'barbarian': {
        const advantage = ready(h, 'roar') && avoidance(target) >= t.roar.minAvoidance;
        if (advantage) {
          use(h, 'roar');
        }
        const charge = ready(h, 'charge');
        if (charge) {
          use(h, 'charge');
        }
        attack(h, [target], { advantage, extraDamage: charge ? t.charge.extraDamage : 0 });
        return;
      }
      case 'rogue':
        if (ready(h, 'fan') && targets.length >= 2) {
          use(h, 'fan');
          attack(h, targets.slice(0, t.fan.targets));
          return;
        }
        attack(h, [target]);
        return;
      case 'ranger':
        if (ready(h, 'rain') && targets.length >= t.rain.minFoes) {
          use(h, 'rain');
          const arrows = roller.die(4) + 1;
          attack(h, targets.slice(0, t.rain.targets), { bonus: -Math.ceil(h.spec.attack.accuracy / 2), damageFactor: arrows });
          return;
        }
        if (ready(h, 'multi') && targets.length >= 2) {
          use(h, 'multi');
          attack(h, targets.slice(0, t.multi.targets));
          return;
        }
        if (ready(h, 'aimed') && avoidance(target) >= t.aimed.minAvoidance) {
          use(h, 'aimed');
          attack(h, [target], { sure: true });
          return;
        }
        attack(h, [target]);
        return;
      case 'cleric':
        clericTurn(h, target);
        return;
    }
  };

  const wound = (h: HeroState, damage: number): void => {
    h.body -= damage;
    if (h.body <= 0) {
      h.body = 0;
      h.alive = false;
    }
  };

  const monsterTurn = (): void => {
    let melee = 0;
    for (const f of foes) {
      if (f.body <= 0) {
        continue;
      }
      const unlimited = (f.spec.ranged ?? false) || (f.spec.reach ?? false);
      if (!unlimited && melee >= t.meleeLimit) {
        continue;
      }
      const candidates = liveHeroes().filter((h) => !h.vanished);
      const target = candidates[roller.die(candidates.length) - 1];
      if (!target) {
        continue;
      }
      if (!unlimited) {
        melee++;
      }
      if (t.abilities && target.spec.cls === 'rogue' && ready(target, 'vanish') && target.body <= t.vanish.below * target.maxBody) {
        use(target, 'vanish');
        target.vanished = true;
        continue;
      }
      wound(target, monsterStrike(f.spec.attack, target.spec.defense, roller, !target.undefended).damage);
      const behind = liveHeroes().filter((h) => h !== target && !h.vanished);
      for (let i = 0; i < (f.spec.line ?? 0) && behind.length > 0; i++) {
        const [h] = behind.splice(roller.die(behind.length) - 1, 1);
        if (h) {
          wound(h, monsterStrike(f.spec.attack, h.spec.defense, roller, !h.undefended).damage);
        }
      }
      const splash = f.spec.splash;
      if (splash) {
        const others = liveHeroes().filter((h) => h !== target);
        for (let i = 0; i < splash.targets && others.length > 0; i++) {
          const [h] = others.splice(roller.die(others.length) - 1, 1);
          if (h) {
            wound(h, Math.max(0, splash.damage - h.spec.defense.mitigation));
          }
        }
      }
    }
  };

  for (let round = 1; round <= maxRounds; round++) {
    for (const h of liveHeroes()) {
      if (round > 1) {
        for (const a of Object.keys(h.cooldowns) as AbilityName[]) {
          h.cooldowns[a] = Math.max(0, (h.cooldowns[a] ?? 0) - 1);
        }
      }
      if (t.abilities && h.maxMana > 0) {
        h.mana = Math.min(h.maxMana, h.mana + (h.spec.manaRegen ?? 0));
      }
      h.vanished = false;
    }
    let acting = round === 1 ? t.openingAttackers : heroes.length;
    for (const h of heroes) {
      if (h.alive && acting > 0 && liveFoes().length > 0) {
        acting--;
        heroTurn(h);
      }
    }
    if (liveFoes().length === 0) {
      return { won: true, rounds: round };
    }
    monsterTurn();
    if (liveHeroes().length === 0) {
      return { won: false, rounds: round };
    }
  }
  return { won: false, rounds: maxRounds };
}

/**
 * Between fights: cooldowns count down to their floor, the pool (once) heals everyone,
 * the Cleric heals while mana lasts (praying if Prayer is ready), and badly hurt heroes
 * drink potions.
 */
export function restBetweenFights(heroes: HeroState[], t: Tactics, supplies: Supplies, pool: boolean): void {
  const alive = heroes.filter((h) => h.alive);
  for (const h of alive) {
    for (const a of Object.keys(h.cooldowns) as AbilityName[]) {
      h.cooldowns[a] = cooldownFloor(h.cooldowns[a] ?? 0, t[a].cooldown);
    }
    h.skip = false;
    h.undefended = false;
    h.streak = 0;
    if (pool) {
      h.body = h.maxBody;
    }
  }
  const cleric = alive.find((h) => h.spec.cls === 'cleric');
  if (t.abilities && cleric) {
    for (;;) {
      const patient = alive.filter((h) => h.body < t.restHealAt * h.maxBody).sort((a, b) => a.body / a.maxBody - b.body / b.maxBody)[0];
      if (!patient) {
        break;
      }
      if (cleric.mana >= t.heal.cost) {
        cleric.mana -= t.heal.cost;
        patient.body = Math.min(patient.maxBody, patient.body + t.heal.amount);
      } else if (ready(cleric, 'prayer') && cleric.maxMana > 0) {
        cleric.mana = Math.min(cleric.maxMana, cleric.mana + Math.ceil(cleric.maxMana / 2));
        cleric.cooldowns.prayer = cooldownFloor(t.prayer.cooldown, t.prayer.cooldown);
      } else {
        break;
      }
    }
  }
  for (const h of alive) {
    if (h.body <= t.potionAt * h.maxBody && supplies.healPotions > 0) {
      supplies.healPotions--;
      h.body = Math.min(h.maxBody, h.body + supplies.healAmount);
    }
  }
}

export interface QuestPlan {
  encounters: { name: string; monsters: string[] }[];
  /** The pool heals everyone fully once, after this encounter. */
  poolAfter?: string;
  /** Encounters from this index on come in a random order each run (the party can go anywhere). */
  shuffleFrom?: number;
}

export interface QuestResult {
  cleared: boolean;
  /** Heroes dead at the end (or at the wipe). */
  deaths: string[];
  wipedAt?: string;
  /** Each encounter fought, in the order fought. */
  fought: { name: string; rounds: number }[];
  bodyLeft: Record<string, number>;
}

/** Plays the party through every encounter in order; a wipe ends the quest. */
export function runQuest(party: HeroSpec[], monsters: Record<string, MonsterSpec>, plan: QuestPlan, t: Tactics, supplies: Supplies, roller: Roller): QuestResult {
  const fights = plan.encounters.map((e) => ({
    name: e.name,
    foes: e.monsters.map((type) => {
      const m = monsters[type];
      if (!m) {
        throw new Error(`unknown monster type "${type}" in ${e.name}`);
      }
      return m;
    }),
  }));
  if (plan.shuffleFrom !== undefined) {
    for (let i = fights.length - 1; i > plan.shuffleFrom; i--) {
      const j = plan.shuffleFrom + roller.die(i - plan.shuffleFrom + 1) - 1;
      const a = fights[i];
      const b = fights[j];
      if (a && b) {
        fights[i] = b;
        fights[j] = a;
      }
    }
  }
  const heroes = party.map(newHero);
  const stock = { ...supplies };
  const fought: QuestResult['fought'] = [];
  const result = (cleared: boolean): QuestResult => ({
    cleared,
    deaths: heroes.filter((h) => !h.alive).map((h) => h.spec.name),
    fought,
    bodyLeft: Object.fromEntries(heroes.map((h) => [h.spec.name, h.body])),
  });
  for (const f of fights) {
    const r = runFight(heroes, f.foes, t, stock, roller);
    fought.push({ name: f.name, rounds: r.rounds });
    if (!r.won) {
      return { ...result(false), wipedAt: f.name };
    }
    restBetweenFights(heroes, t, stock, plan.poolAfter === f.name);
  }
  return result(true);
}
