/**
 * Monte Carlo combat simulator for The Three Plagues house rules (see
 * docs/campaigns/three-plagues/RULES_AND_CLASSES.md, "Combat"). A design aid: it
 * plays a party through a quest's encounters with simple tactics, so hero and
 * monster numbers can be tuned against survival targets. Positioning, movement,
 * traps and doors are not modeled; a cap on melee attackers per round stands in
 * for heroes holding a doorway.
 */

import { rollDice, type DiceExpr } from '../dice/dice.ts';
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
  /** Added to the hit total (Determination, a mark, or a penalty when negative). */
  bonus?: number;
  /** Roll the hit dice twice and keep the better total (Unleash Fury). */
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

/** Rounds left on a cooldown after a fight: a 1-round one is ready, 2-3 stop at 1, longer ones at 2. */
export function cooldownFloor(remaining: number, cooldown: number): number {
  return Math.min(remaining, cooldown <= 1 ? 0 : cooldown <= 3 ? 1 : 2);
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
  /** Takes Smite's extra damage. */
  undead?: boolean;
}

/**
 * How the party plays, and the class abilities (see RULES_AND_CLASSES.md, "Abilities").
 * An ability set to null is not used, for testing what each one is worth.
 */
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
  /** A hero at or below this share of Body drinks a healing potion (a free action, at any time). */
  potionAt: number;
  /** The Cleric heals an ally at or below this share of Body in a fight... */
  healAt: number;
  /** ...and up to this share between fights. */
  restHealAt: number;
  /** Monsters this hard to hit, or with this much Body, count as elites when choosing abilities. */
  eliteAvoidance: number;
  eliteBody: number;
  // Barbarian
  charge: { cooldown: number; extraDamage: number } | null;
  /** Unleash Fury (with Echoing Roar): advantage on the first attack, allies get rally Accuracy. */
  fury: { cooldown: number; rounds: number; damage: number; mitigation: number; rally: number } | null;
  challenge: { cooldown: number } | null;
  cleave: boolean;
  // Rogue
  fan: { cooldown: number; targets: number } | null;
  vanish: { cooldown: number; below: number } | null;
  riposte: { cooldown: number } | null;
  /**
   * Venom Vial: thrown; on a hit the target loses damage Body (doubled on a crit) at the start
   * of its next turns. With weapon, the hit also deals the Rogue's weapon damage.
   */
  vial: { cooldown: number; damage: number; turns: number; weapon?: boolean } | null;
  /** Exploit Opening: the Rogue's crit range starts here against a monster whose last attack went at someone else. */
  exploitCritFrom: number | null;
  // Ranger
  multi: { cooldown: number; targets: number } | null;
  /** Aimed Shot (with Hunter's Mark): a sure hit that marks the target for everyone. */
  aimed: { cooldown: number; mark: number } | null;
  rain: { cooldown: number; targets: number; minFoes: number; arrows: DiceExpr } | null;
  // Cleric
  /** Smite: the Cleric's attack with these changes, plus extra damage to undead. */
  smite: { cost: number; attack: Partial<HeroAttacker>; undead: number } | null;
  heal: { cost: number; amount: number } | null;
  turnEvil: { cost: number; cooldown: number } | null;
  prayer: { cooldown: number } | null;
}

const D4: DiceExpr = { terms: [{ count: 1, sides: 4, negative: false }], modifier: 0 };

/** The agreed kit (2026-10-04, after pruning). */
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
  eliteAvoidance: 12,
  eliteBody: 30,
  charge: { cooldown: 3, extraDamage: 4 },
  fury: { cooldown: 6, rounds: 3, damage: 3, mitigation: 2, rally: 2 },
  challenge: { cooldown: 4 },
  cleave: true,
  fan: { cooldown: 3, targets: 3 },
  vanish: { cooldown: 5, below: 0.5 },
  riposte: { cooldown: 2 },
  vial: { cooldown: 4, damage: 3, turns: 3, weapon: true },
  exploitCritFrom: 13,
  multi: { cooldown: 5, targets: 3 },
  aimed: { cooldown: 5, mark: 2 },
  rain: { cooldown: 7, targets: 4, minFoes: 3, arrows: D4 },
  smite: { cost: 2, attack: { damage: 5 }, undead: 2 },
  heal: { cost: 6, amount: 12 },
  turnEvil: { cost: 8, cooldown: 5 },
  prayer: { cooldown: 5 },
};

type CooldownName = 'charge' | 'fury' | 'challenge' | 'fan' | 'vanish' | 'riposte' | 'vial' | 'multi' | 'aimed' | 'rain' | 'turnEvil' | 'prayer';

const cooldownOf = (t: Tactics, a: CooldownName): number => t[a]?.cooldown ?? 0;

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
  cooldowns: Partial<Record<CooldownName, number>>;
  /** Misses in a row (Determination). */
  streak: number;
  /** Loses the next turn (critical miss). */
  skip: boolean;
  /** Rolls no defense dice until their next turn (after Prayer). */
  undefended: boolean;
  /** Cannot be targeted for the rest of the round (Vanish From Sight). */
  vanished: boolean;
  alive: boolean;
  /** Rounds of Unleash Fury left. */
  rage: number;
  /** Accuracy added to the next attack (an ally's Unleash Fury roar). */
  rally: number;
  /** Challenge: melee monsters attack this hero on their next turn. */
  challenging: boolean;
  /** What the hero did with each turn (attack, potion, skip, or an ability), and free abilities used. */
  uses: Record<string, number>;
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
    rage: 0,
    rally: 0,
    challenging: false,
    uses: {},
  };
}

interface Foe {
  spec: MonsterSpec;
  body: number;
  /** The hero it last attacked (it faces them; a Rogue elsewhere is behind it). */
  lastTarget: HeroState | null;
  /** Aimed Shot's mark: every hero has extra Accuracy against it. */
  marked: boolean;
  /** Turn Evil: skips its next attack, and attacks on it are sure hits through this round. */
  skipNext: boolean;
  turnedUntil: number;
  /** Venom Vial: Body lost at the start of each of its next turns. */
  poison: { damage: number; turns: number };
}

const count = (h: HeroState, what: string): void => {
  h.uses[what] = (h.uses[what] ?? 0) + 1;
};

const usable = (t: Tactics, h: HeroState, a: CooldownName): boolean => t.abilities && t[a] !== null && (h.cooldowns[a] ?? 0) <= 0;

/** Plays one fight to the end (or maxRounds). Heroes act first each round, in party order. */
export function runFight(heroes: HeroState[], monsters: MonsterSpec[], t: Tactics, supplies: Supplies, roller: Roller, maxRounds = 200): { won: boolean; rounds: number; foesBody: number[] } {
  const foes: Foe[] = monsters.map((spec) => ({ spec, body: spec.body, lastTarget: null, marked: false, skipNext: false, turnedUntil: 0, poison: { damage: 0, turns: 0 } }));
  const liveFoes = (): Foe[] => foes.filter((f) => f.body > 0).sort((a, b) => a.body - b.body);
  const liveHeroes = (): HeroState[] => heroes.filter((h) => h.alive);
  const avoidance = (f: Foe): number => {
    const at = t.falterShare > 0 ? Math.max(1, Math.floor(f.spec.body * t.falterShare)) : 0;
    return f.body <= at ? f.spec.avoidance - t.falterPenalty : f.spec.avoidance;
  };
  const elite = (f: Foe): boolean => f.spec.avoidance >= t.eliteAvoidance || f.spec.body >= t.eliteBody;
  let round = 0;
  const result = (won: boolean) => ({ won, rounds: round, foesBody: foes.map((f) => Math.max(0, f.body)) });

  const use = (h: HeroState, a: CooldownName): void => {
    h.cooldowns[a] = cooldownOf(t, a);
  };

  /** Attacks each target once; returns how many it killed. */
  const attack = (h: HeroState, targets: Foe[], o: StrikeOptions = {}, a: HeroAttacker = h.spec.attack, smite = false, onHit?: (f: Foe, r: StrikeResult) => void): number => {
    const bonus = Math.min(t.determinationCap, h.streak * t.determinationStep) + (o.bonus ?? 0) + h.rally;
    h.rally = 0;
    let hit = false;
    let kills = 0;
    for (const f of targets) {
      if (f.body <= 0) {
        continue;
      }
      const behind = h.spec.cls === 'rogue' && t.exploitCritFrom !== null && f.lastTarget !== null && f.lastTarget !== h;
      const strike = behind && t.exploitCritFrom !== null ? { ...a, critFrom: Math.min(a.critFrom, t.exploitCritFrom) } : a;
      const extra = (o.extraDamage ?? 0) + (h.rage > 0 && t.fury ? t.fury.damage : 0) + (smite && f.spec.undead && t.smite ? t.smite.undead : 0);
      const r = heroStrike(strike, avoidance(f), roller, {
        ...o,
        bonus: bonus + (f.marked && t.aimed ? t.aimed.mark : 0),
        extraDamage: extra,
        ...(f.turnedUntil >= round ? { sure: true } : {}),
      });
      f.body -= r.damage;
      hit ||= r.hit;
      if (r.hit) {
        onHit?.(f, r);
      }
      if (r.criticalMiss) {
        h.skip = true;
      }
      if (f.body <= 0) {
        kills++;
      }
    }
    h.streak = hit ? 0 : h.streak + 1;
    return kills;
  };

  const heal = (patient: HeroState, amount: number): void => {
    patient.body = Math.min(patient.maxBody, patient.body + amount);
  };

  /** Drinking a potion is a free action, even during the monsters' turn. */
  const drinkIfLow = (h: HeroState): void => {
    if (h.alive && h.body <= t.potionAt * h.maxBody && supplies.healPotions > 0) {
      supplies.healPotions--;
      count(h, 'potion');
      heal(h, supplies.healAmount);
    }
  };

  const barbarianTurn = (h: HeroState, targets: Foe[]): void => {
    const melee = targets.filter((f) => !f.spec.ranged).length;
    if (usable(t, h, 'challenge') && melee >= 2 && liveHeroes().length > 1) {
      use(h, 'challenge');
      h.challenging = true;
      count(h, 'challenge');
    }
    const target = targets[0];
    if (!target) {
      return;
    }
    let kills: number;
    if (h.rage <= 0 && t.fury && usable(t, h, 'fury') && targets.some(elite)) {
      use(h, 'fury');
      h.rage = t.fury.rounds;
      for (const ally of liveHeroes()) {
        if (ally !== h) {
          ally.rally = t.fury.rally;
        }
      }
      count(h, 'fury');
      kills = attack(h, [target], { advantage: true });
    } else if (t.charge && usable(t, h, 'charge')) {
      use(h, 'charge');
      count(h, 'charge');
      kills = attack(h, [target], { extraDamage: t.charge.extraDamage });
    } else {
      count(h, 'attack');
      kills = attack(h, [target]);
    }
    if (kills > 0) {
      h.rage = 0;
      const next = liveFoes()[0];
      if (t.abilities && t.cleave && next) {
        count(h, 'cleave');
        attack(h, [next]);
      }
    }
  };

  const rogueTurn = (h: HeroState, targets: Foe[]): void => {
    const vial = t.vial;
    const mark = targets.filter((f) => f.poison.turns === 0 && f.body > (vial ? vial.damage * vial.turns : 0)).sort((a, b) => b.body - a.body)[0];
    if (vial && mark && usable(t, h, 'vial')) {
      use(h, 'vial');
      count(h, 'vial');
      attack(h, [mark], {}, vial.weapon ? h.spec.attack : { ...h.spec.attack, damage: 0 }, false, (f, r) => {
        f.poison = { damage: r.crit ? 2 * vial.damage : vial.damage, turns: vial.turns };
      });
      return;
    }
    if (t.fan && usable(t, h, 'fan') && targets.length >= 2) {
      use(h, 'fan');
      count(h, 'fan');
      attack(h, targets.slice(0, t.fan.targets));
      return;
    }
    count(h, 'attack');
    attack(h, targets.slice(0, 1));
  };

  const rangerTurn = (h: HeroState, targets: Foe[]): void => {
    const target = targets[0];
    if (!target) {
      return;
    }
    if (t.rain && usable(t, h, 'rain') && targets.length >= t.rain.minFoes) {
      use(h, 'rain');
      count(h, 'rain');
      const arrows = rollDice(t.rain.arrows, (sides) => roller.die(sides)).total;
      attack(h, targets.slice(0, t.rain.targets), { bonus: -Math.ceil(h.spec.attack.accuracy / 2), damageFactor: arrows });
      return;
    }
    if (t.multi && usable(t, h, 'multi') && targets.length >= 2) {
      use(h, 'multi');
      count(h, 'multi');
      attack(h, targets.slice(0, t.multi.targets));
      return;
    }
    const big = targets.filter(elite).sort((a, b) => b.body - a.body)[0];
    if (big && usable(t, h, 'aimed')) {
      use(h, 'aimed');
      count(h, 'aimed');
      big.marked = true;
      attack(h, [big], { sure: true });
      return;
    }
    count(h, 'attack');
    attack(h, [target]);
  };

  const clericTurn = (h: HeroState, targets: Foe[]): void => {
    const target = targets[0];
    if (!target) {
      return;
    }
    const hurt = liveHeroes().filter((x) => x.body <= t.healAt * x.maxBody).sort((a, b) => a.body / a.maxBody - b.body / b.maxBody);
    const big = targets.filter(elite).sort((a, b) => b.body - a.body)[0];
    const pay = (cost: number): boolean => {
      if (h.mana < cost) {
        return false;
      }
      h.mana -= cost;
      return true;
    };
    if (t.abilities) {
      const patient = hurt[0];
      if (patient && t.heal && pay(t.heal.cost)) {
        count(h, 'heal');
        heal(patient, t.heal.amount);
        return;
      }
      if (big && t.turnEvil && usable(t, h, 'turnEvil') && pay(t.turnEvil.cost)) {
        use(h, 'turnEvil');
        count(h, 'turnEvil');
        big.skipNext = true;
        big.turnedUntil = round + 1;
        return;
      }
      if (t.smite && h.mana < t.smite.cost) {
        if (usable(t, h, 'prayer')) {
          use(h, 'prayer');
          count(h, 'prayer');
          h.mana = Math.min(h.maxMana, h.mana + Math.ceil(h.maxMana / 2));
          h.undefended = true;
          return;
        }
        if (supplies.manaPotions > 0) {
          supplies.manaPotions--;
          count(h, 'manaPotion');
          h.mana = Math.min(h.maxMana, h.mana + supplies.manaAmount);
        }
      }
      if (t.smite && pay(t.smite.cost)) {
        count(h, 'smite');
        attack(h, [target], {}, { ...h.spec.attack, ...t.smite.attack }, true);
        return;
      }
    }
    count(h, 'attack');
    attack(h, [target]);
  };

  const heroTurn = (h: HeroState): void => {
    if (h.skip) {
      h.skip = false;
      count(h, 'skip');
      return;
    }
    h.undefended = false;
    drinkIfLow(h);
    const targets = liveFoes();
    if (targets.length === 0) {
      return;
    }
    if (!t.abilities) {
      count(h, 'attack');
      attack(h, targets.slice(0, 1));
      return;
    }
    switch (h.spec.cls) {
      case 'barbarian':
        barbarianTurn(h, targets);
        return;
      case 'rogue':
        rogueTurn(h, targets);
        return;
      case 'ranger':
        rangerTurn(h, targets);
        return;
      case 'cleric':
        clericTurn(h, targets);
        return;
    }
  };

  const wound = (h: HeroState, damage: number): void => {
    h.body -= damage;
    if (h.body <= 0) {
      h.body = 0;
      h.alive = false;
    }
    drinkIfLow(h);
  };

  const defenseOf = (h: HeroState): HeroDefender => {
    const d = h.spec.defense;
    const raging = h.rage > 0 && t.fury ? t.fury.mitigation : 0;
    return { ...d, mitigation: d.mitigation + raging };
  };

  const monsterTurn = (): void => {
    let melee = 0;
    for (const f of foes) {
      if (f.body <= 0) {
        continue;
      }
      if (f.poison.turns > 0) {
        f.poison.turns--;
        f.body -= f.poison.damage;
        if (f.body <= 0) {
          continue;
        }
      }
      if (f.skipNext) {
        f.skipNext = false;
        continue;
      }
      const unlimited = (f.spec.ranged ?? false) || (f.spec.reach ?? false);
      if (!unlimited && melee >= t.meleeLimit) {
        continue;
      }
      const candidates = liveHeroes().filter((h) => !h.vanished);
      const challenger = unlimited ? undefined : candidates.find((h) => h.challenging);
      const target = challenger ?? candidates[roller.die(candidates.length) - 1];
      if (!target) {
        continue;
      }
      if (!unlimited) {
        melee++;
      }
      if (t.abilities && t.vanish && target.spec.cls === 'rogue' && usable(t, target, 'vanish') && target.body <= t.vanish.below * target.maxBody) {
        use(target, 'vanish');
        count(target, 'vanish');
        target.vanished = true;
        continue;
      }
      f.lastTarget = target;
      const r = monsterStrike(f.spec.attack, defenseOf(target), roller, !target.undefended);
      wound(target, r.damage);
      if (!r.hit && target.alive && target.spec.cls === 'rogue' && t.riposte && usable(t, target, 'riposte')) {
        use(target, 'riposte');
        count(target, 'riposte');
        f.body -= Math.floor(target.spec.attack.damage / 2);
      }
      const behind = liveHeroes().filter((h) => h !== target && !h.vanished);
      for (let i = 0; i < (f.spec.line ?? 0) && behind.length > 0; i++) {
        const [h] = behind.splice(roller.die(behind.length) - 1, 1);
        if (h) {
          wound(h, monsterStrike(f.spec.attack, defenseOf(h), roller, !h.undefended).damage);
        }
      }
      const splash = f.spec.splash;
      if (splash) {
        const others = liveHeroes().filter((h) => h !== target);
        for (let i = 0; i < splash.targets && others.length > 0; i++) {
          const [h] = others.splice(roller.die(others.length) - 1, 1);
          if (h) {
            wound(h, Math.max(0, splash.damage - defenseOf(h).mitigation));
          }
        }
      }
    }
    for (const h of heroes) {
      h.challenging = false;
    }
  };

  for (round = 1; round <= maxRounds; round++) {
    for (const h of liveHeroes()) {
      if (round > 1) {
        for (const a of Object.keys(h.cooldowns) as CooldownName[]) {
          h.cooldowns[a] = Math.max(0, (h.cooldowns[a] ?? 0) - 1);
        }
        h.rage = Math.max(0, h.rage - 1);
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
      return result(true);
    }
    monsterTurn();
    if (liveHeroes().length === 0) {
      return result(false);
    }
  }
  round = maxRounds;
  return result(false);
}

/**
 * Between fights: cooldowns count down to their floor, the pool (once) heals everyone,
 * the Cleric heals while mana lasts (praying if Prayer is ready), and badly hurt heroes
 * drink potions.
 */
export function restBetweenFights(heroes: HeroState[], t: Tactics, supplies: Supplies, pool: boolean): void {
  const alive = heroes.filter((h) => h.alive);
  for (const h of alive) {
    for (const a of Object.keys(h.cooldowns) as CooldownName[]) {
      h.cooldowns[a] = cooldownFloor(h.cooldowns[a] ?? 0, cooldownOf(t, a));
    }
    h.skip = false;
    h.undefended = false;
    h.streak = 0;
    h.rage = 0;
    h.rally = 0;
    h.challenging = false;
    if (pool) {
      h.body = h.maxBody;
    }
  }
  const low = (h: HeroState): boolean => h.body < t.restHealAt * h.maxBody;
  // Prayer used after a fight starts its cooldown at the out-of-fight floor.
  const spend = (h: HeroState, a: CooldownName): void => {
    h.cooldowns[a] = cooldownFloor(cooldownOf(t, a), cooldownOf(t, a));
    h.uses[a] = (h.uses[a] ?? 0) + 1;
  };
  const cleric = alive.find((h) => h.spec.cls === 'cleric');
  if (t.abilities && cleric) {
    const heal = t.heal;
    for (;;) {
      const patient = alive.filter(low).sort((a, b) => a.body / a.maxBody - b.body / b.maxBody)[0];
      if (!patient || !heal) {
        break;
      }
      if (cleric.mana >= heal.cost) {
        cleric.mana -= heal.cost;
        patient.body = Math.min(patient.maxBody, patient.body + heal.amount);
      } else if (t.prayer && (cleric.cooldowns.prayer ?? 0) <= 0 && cleric.maxMana > 0) {
        cleric.mana = Math.min(cleric.maxMana, cleric.mana + Math.ceil(cleric.maxMana / 2));
        spend(cleric, 'prayer');
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
  /** Gear found after an encounter changes the hero's stats (Body and mana caps follow). */
  finds?: { after: string; hero: string; apply: (spec: HeroSpec) => HeroSpec }[];
}

export interface QuestResult {
  cleared: boolean;
  /** Heroes dead at the end (or at the wipe). */
  deaths: string[];
  wipedAt?: string;
  /** Each encounter fought, in the order fought. */
  fought: { name: string; rounds: number }[];
  bodyLeft: Record<string, number>;
  /** What each hero did with their turns (see HeroState.uses). */
  uses: Record<string, Record<string, number>>;
  /** Each hero's stats at the end, after any finds. */
  specs: Record<string, HeroSpec>;
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
    uses: Object.fromEntries(heroes.map((h) => [h.spec.name, h.uses])),
    specs: Object.fromEntries(heroes.map((h) => [h.spec.name, h.spec])),
  });
  for (const f of fights) {
    const r = runFight(heroes, f.foes, t, stock, roller);
    fought.push({ name: f.name, rounds: r.rounds });
    if (!r.won) {
      return { ...result(false), wipedAt: f.name };
    }
    for (const find of plan.finds ?? []) {
      const h = heroes.find((x) => x.spec.name === find.hero);
      if (find.after === f.name && h?.alive) {
        h.spec = find.apply(h.spec);
        h.maxBody = h.spec.body;
        h.body = Math.min(h.body, h.maxBody);
        h.maxMana = h.spec.mana ?? 0;
        h.mana = Math.min(h.mana, h.maxMana);
      }
    }
    restBetweenFights(heroes, t, stock, plan.poolAfter === f.name);
  }
  return result(true);
}
