/**
 * Prints combat odds tables for tuning The Three Plagues numbers:
 *   bun scripts/combat-odds.ts
 * Edit the config below and rerun. The hero and monster values are samples for comparison,
 * not campaign decisions (see docs/campaigns/three-plagues/RULES_AND_CLASSES.md).
 */

import { parseDice, type DiceExpr } from '../internal/web/src/dice/dice.ts';
import {
  attacksToKill,
  heroAttack,
  heroQuestDamage,
  monsterAttack,
  type HeroAttacker,
  type HeroDefender,
  type MonsterAttacker,
} from '../internal/web/src/combat/odds.ts';

function dice(s: string): DiceExpr {
  const r = parseDice(s);
  if (!r.ok) {
    throw new Error(r.error);
  }
  return r.expr;
}

const NEAR_MISS = 2;

/** Sample hero attacks (the step 2 starting point). */
const HEROES: Record<string, HeroAttacker> = {
  Barbarian: { hitDice: dice('1d20'), accuracy: 1, critFrom: 18, critMultiplier: 2, nearMiss: NEAR_MISS, damage: 8 },
  Rogue: { hitDice: dice('2d10'), accuracy: 0, critFrom: 13, critMultiplier: 2, nearMiss: NEAR_MISS, damage: 4 },
  Ranger: { hitDice: dice('2d10'), accuracy: 4, critFrom: 18, critMultiplier: 2, nearMiss: NEAR_MISS, damage: 5 },
  Cleric: { hitDice: dice('2d8'), accuracy: 2, critFrom: 20, critMultiplier: 2, nearMiss: NEAR_MISS, damage: 3 },
};

/** Sample hero defenses. */
const DEFENDERS: Record<string, HeroDefender> = {
  Barbarian: { defenseDice: dice('1d6'), baseAvoidance: 2, mitigation: 2 },
  Rogue: { defenseDice: dice('1d6'), baseAvoidance: 6, mitigation: 0 },
  Ranger: { defenseDice: dice('1d6'), baseAvoidance: 5, mitigation: 0 },
  Cleric: { defenseDice: dice('1d6'), baseAvoidance: 3, mitigation: 0 },
};

/** Sample monster attacks. */
const MONSTERS: Record<string, MonsterAttacker> = {
  '1d12, 3 dmg': { hitDice: dice('1d12'), damage: 3 },
  '2d8, 4 dmg': { hitDice: dice('2d8'), damage: 4 },
  '2d10, 5 dmg': { hitDice: dice('2d10'), damage: 5 },
  '1d20+2, 6 dmg': { hitDice: dice('1d20+2'), damage: 6 },
};

/** Original HeroQuest monsters: attack dice, defend dice, Body. */
const ORIGINAL: Record<string, [number, number, number]> = {
  Goblin: [2, 1, 1],
  Orc: [3, 2, 1],
  Skeleton: [2, 2, 1],
  Zombie: [2, 3, 1],
  Abomination: [3, 3, 2],
  Mummy: [3, 4, 2],
  'Dread Warrior': [4, 4, 3],
  Gargoyle: [4, 5, 3],
};

const pct = (p: number): string => `${Math.round(p * 100)}%`;
const num = (x: number): string => (Number.isFinite(x) ? x.toFixed(1) : '-');

function table(title: string, head: string[], rows: string[][]): void {
  const widths = head.map((h, i) => Math.max(h.length, ...rows.map((r) => (r[i] ?? '').length)));
  const line = (cells: string[]): string => `| ${cells.map((c, i) => c.padEnd(widths[i] ?? 0)).join(' | ')} |`;
  console.log(`\n${title}\n`);
  console.log(line(head));
  console.log(`|${widths.map((w) => '-'.repeat(w + 2)).join('|')}|`);
  for (const r of rows) {
    console.log(line(r));
  }
}

table(
  'Original HeroQuest: hero hit chance / attacks to kill (1, 2, 3 attack dice); monster vs a 2-defend-dice hero',
  ['Monster', '1 die', '2 dice', '3 dice', 'vs hero: hit / dmg'],
  Object.entries(ORIGINAL).map(([name, [atk, def, body]]) => {
    const cells = [1, 2, 3].map((n) => {
      const d = heroQuestDamage(n, def, 'monster');
      return `${pct(1 - (d.get(0) ?? 0))} / ${num(attacksToKill(d, body))}`;
    });
    const vs = heroQuestDamage(atk, 2, 'hero');
    let e = 0;
    for (const [x, p] of vs) {
      e += x * p;
    }
    return [name, ...cells, `${pct(1 - (vs.get(0) ?? 0))} / ${e.toFixed(2)}`];
  }),
);

const avoidances = [4, 6, 8, 10, 12, 14, 16, 18, 20];
table(
  'Sample heroes vs monster Avoidance: hit chance / expected damage per attack',
  ['Avoidance', ...Object.keys(HEROES)],
  avoidances.map((av) => [
    String(av),
    ...Object.values(HEROES).map((h) => {
      const o = heroAttack(h, av);
      return `${pct(o.hit)} / ${num(o.expectedDamage)}`;
    }),
  ]),
);

table(
  'Sample monsters vs sample heroes: hit chance / expected damage per attack',
  ['Monster', ...Object.keys(DEFENDERS)],
  Object.entries(MONSTERS).map(([name, m]) => [
    name,
    ...Object.values(DEFENDERS).map((h) => {
      const o = monsterAttack(m, h);
      return `${pct(o.hit)} / ${num(o.expectedDamage)}`;
    }),
  ]),
);
