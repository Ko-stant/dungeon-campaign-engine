/**
 * Prints combat odds tables for tuning The Three Plagues numbers:
 *   bun scripts/combat-odds.ts
 * Hero attacks come from scripts/combat-config.ts; the defenses and monsters below are
 * samples for comparison (see docs/campaigns/three-plagues/RULES_AND_CLASSES.md).
 */

import {
  attacksToKill,
  CAMPAIGN_RULES,
  heroAttack,
  heroQuestDamage,
  killOdds,
  monsterAttack,
  percentile,
  type HeroDefender,
  type MonsterAttacker,
} from '../internal/web/src/combat/odds.ts';
import { dice, HERO_ATTACKS as HEROES } from './combat-config.ts';

/** Sample monster defenses for the kill table: Avoidance, Body. */
const TARGETS: [number, number][] = [
  [8, 10],
  [10, 25],
  [12, 40],
  [14, 50],
];

/** Share of maximum Body at or below which a monster falters. */
const FALTER_SHARE = 1 / 4;

/** Sample hero defenses (placeholders until step 2's defense pass). */
const DEFENDERS: Record<string, HeroDefender> = {
  Barbarian: { defenseDice: dice('1d6'), baseAvoidance: 2, mitigation: 2 },
  Ranger: { defenseDice: dice('1d6'), baseAvoidance: 5, mitigation: 0 },
  Rogue: { defenseDice: dice('1d6'), baseAvoidance: 6, mitigation: 0 },
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

const rules = (body: number) => ({ ...CAMPAIGN_RULES, falterAt: Math.max(1, Math.floor(body * FALTER_SHARE)) });
table(
  'Proposed heroes: attacks to kill (average / slowest 10%), with Determination and Faltering',
  ['Avoidance, Body', ...Object.keys(HEROES)],
  TARGETS.map(([av, body]) => [
    `${av}, ${body}`,
    ...Object.values(HEROES).map((h) => {
      const k = killOdds(h, av, body, rules(body));
      return `${num(k.expected)} / ${percentile(k.byAttack, 0.9)}`;
    }),
  ]),
);

const WEIGHT_BODY = 60;
table(
  `Effective damage per attack over a fight (${WEIGHT_BODY} Body), and as weights with the Barbarian at 6`,
  ['Avoidance', ...Object.keys(HEROES)],
  [8, 10, 12, 14].map((av) => {
    const dpa = Object.values(HEROES).map((h) => WEIGHT_BODY / killOdds(h, av, WEIGHT_BODY, rules(WEIGHT_BODY)).expected);
    const top = dpa[0] ?? 1;
    return [String(av), ...dpa.map((x) => `${x.toFixed(1)} (${((6 * x) / top).toFixed(1)})`)];
  }),
);
