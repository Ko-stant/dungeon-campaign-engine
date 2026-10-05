/**
 * Simulates the party through Quest 1, with abilities and attack-only:
 *   bun scripts/combat-sim.ts [runs] [--specter] [--melee=N] [--swap "Room 8=gargoyle,goblin_warlock"]...
 *     [--with=ability] [--without=fury,riposte,...] [--mana=12] [--heal=10] [--finds | --geared]
 * --finds picks up the Quest 1 finds as they are found; --geared starts with all of them.
 * Numbers come from scripts/combat-config.ts; encounters from the generated JSON
 * (scripts/quest-encounters.ts). The route rooms come first, the rest in a random
 * order each run. --swap replaces a room's monsters (repeatable); --melee sets how
 * many melee monsters can attack a round.
 */

import { runFight, runQuest, newHero, seededRoller, type HeroSpec, type MonsterSpec, type QuestPlan, type Tactics } from '../internal/web/src/combat/simulate.ts';
import { MONSTERS, PARTY as START_PARTY, PARTY_AFTER_QUEST_1, QUEST_1, QUEST_1_FINDS, SUPPLIES, TACTICS as BASE_TACTICS, TESTING, upgrade } from './combat-config.ts';

const args = process.argv.slice(2);
const runs = Number(args.find((a) => /^\d+$/.test(a)) ?? 4000);
const without = args.includes('--specter') ? [] : QUEST_1.without;
const flag = (name: string): string | undefined => args.find((a) => a.startsWith(`--${name}=`))?.slice(name.length + 3);
const list = (name: string): string[] => (flag(name) ?? '').split(',').map((x) => x.trim()).filter(Boolean);
const melee = Number(flag('melee') ?? BASE_TACTICS.meleeLimit);
const tested = list('with');
const dropped = list('without');
const TACTICS: Tactics = { ...BASE_TACTICS };
const tacticsMap = TACTICS as unknown as Record<string, unknown>;
for (const name of tested) {
  if (!(name in TESTING)) {
    throw new Error(`--with: no tested ability "${name}" (have ${Object.keys(TESTING).join(', ')})`);
  }
  tacticsMap[name] = (TESTING as Record<string, unknown>)[name];
}
for (const name of dropped) {
  if (!(name in TACTICS)) {
    throw new Error(`--without: no ability "${name}"`);
  }
  tacticsMap[name] = typeof tacticsMap[name] === 'boolean' ? false : null;
}
if (flag('heal') && TACTICS.heal) {
  TACTICS.heal = { ...TACTICS.heal, amount: Number(flag('heal')) };
}
const geared = args.includes('--geared');
const finds = args.includes('--finds');
const PARTY = (geared ? PARTY_AFTER_QUEST_1 : START_PARTY).map((h) => (h.cls === 'cleric' && flag('mana') ? { ...h, mana: Number(flag('mana')) } : h));
const swaps = new Map<string, string[]>();
args.forEach((a, i) => {
  const value = a === '--swap' ? args[i + 1] : undefined;
  const [room, list] = value?.split('=') ?? [];
  if (room && list !== undefined) {
    swaps.set(room.trim(), list.split(',').map((m) => m.trim()).filter(Boolean));
  }
});

const file = (await Bun.file(QUEST_1.file).json()) as { encounters: { name: string; monsters: string[] }[] };
const all = file.encounters
  .map((e) => ({ name: e.name, monsters: (swaps.get(e.name) ?? e.monsters).filter((m) => !without.includes(m)) }))
  .filter((e) => e.monsters.length > 0);
const first = QUEST_1.route.flatMap((name) => all.filter((e) => e.name === name));
const plan: QuestPlan = {
  encounters: [...first, ...all.filter((e) => !QUEST_1.route.includes(e.name))],
  poolAfter: QUEST_1.poolAfter,
  shuffleFrom: first.length,
  ...(finds ? { finds: QUEST_1_FINDS.map((f) => ({ after: f.after, hero: f.item.hero, apply: (h: HeroSpec) => upgrade(h, f.item) })) } : {}),
};

const pct = (x: number): string => `${Math.round(x * 100)}%`;

function monsterSpec(type: string): MonsterSpec {
  const m = MONSTERS[type];
  if (!m) {
    throw new Error(`no stats for monster type "${type}" in scripts/combat-config.ts`);
  }
  return m;
}

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

const modes: [string, Tactics][] = [
  ['With abilities', { ...TACTICS, meleeLimit: melee }],
  ['Attack only', { ...TACTICS, meleeLimit: melee, abilities: false }],
];

for (const [mode, tactics] of modes) {
  const roller = seededRoller(2026);
  let cleared = 0;
  let flawless = 0;
  let deaths = 0;
  const heroDeaths = new Map<string, number>();
  const uses = new Map<string, Map<string, number>>();
  const wipes = new Map<string, number>();
  const rounds = new Map<string, number[]>();
  for (let i = 0; i < runs; i++) {
    const r = runQuest(PARTY, MONSTERS, plan, tactics, SUPPLIES, roller);
    cleared += r.cleared ? 1 : 0;
    flawless += r.cleared && r.deaths.length === 0 ? 1 : 0;
    deaths += r.deaths.length;
    for (const d of r.deaths) {
      heroDeaths.set(d, (heroDeaths.get(d) ?? 0) + 1);
    }
    for (const [hero, u] of Object.entries(r.uses)) {
      const m = uses.get(hero) ?? new Map<string, number>();
      for (const [what, n] of Object.entries(u)) {
        m.set(what, (m.get(what) ?? 0) + n);
      }
      uses.set(hero, m);
    }
    if (r.wipedAt) {
      wipes.set(r.wipedAt, (wipes.get(r.wipedAt) ?? 0) + 1);
    }
    for (const f of r.fought) {
      rounds.set(f.name, [...(rounds.get(f.name) ?? []), f.rounds]);
    }
  }
  const notes = [
    `${runs} runs`,
    `melee limit ${melee}`,
    ...(without.length ? [`without ${without.join(', ')}`] : []),
    ...[...swaps].map(([k, v]) => `${k} = ${v.join(', ')}`),
    ...(tested.length ? [`testing ${tested.join(', ')}`] : []),
    ...(dropped.length ? [`no ${dropped.join(', ')}`] : []),
    ...(flag('mana') ? [`Cleric mana ${flag('mana') ?? ''}`] : []),
    ...(flag('heal') ? [`heal ${flag('heal') ?? ''}`] : []),
    ...(finds ? ['Quest 1 finds picked up'] : []),
    ...(geared ? ['all Quest 1 finds from the start'] : []),
  ];
  console.log(`\n## ${mode} (${notes.join('; ')})`);
  console.log(`Quest cleared: ${pct(cleared / runs)} | cleared with no deaths: ${pct(flawless / runs)} | average deaths: ${(deaths / runs).toFixed(2)}`);
  console.log(`Death rate by hero: ${PARTY.map((h) => `${h.name} ${pct((heroDeaths.get(h.name) ?? 0) / runs)}`).join(', ')}`);

  if (tactics.abilities) {
    // Turns spent on each action; "free" abilities ride along with another action.
    const FREE = new Set(['challenge', 'cleave', 'vanish', 'riposte', 'potion', 'manaPotion']);
    const MAIN: Record<string, string> = { Cleric: 'smite' };
    table(
      'How heroes spend their turns (share of turns; free abilities and reactions per 10 turns)',
      ['Hero', 'Basic attack', 'Abilities', 'Breakdown', 'Free / reactions per 10 turns'],
      PARTY.map((h) => {
        const m = uses.get(h.name) ?? new Map<string, number>();
        const turns = [...m].filter(([k]) => !FREE.has(k)).reduce((n, [, v]) => n + v, 0) || 1;
        const basic = (m.get('attack') ?? 0) + (m.get(MAIN[h.name] ?? '') ?? 0);
        const actions = [...m].filter(([k]) => !FREE.has(k) && k !== 'attack' && k !== MAIN[h.name] && k !== 'skip' && k !== 'potion' && k !== 'manaPotion');
        const abilityTurns = actions.reduce((n, [, v]) => n + v, 0);
        const free = [...m].filter(([k]) => FREE.has(k));
        return [
          h.name,
          pct(basic / turns) + (MAIN[h.name] ? ` (${pct((m.get('attack') ?? 0) / turns)} plain, rest ${MAIN[h.name] ?? ''})` : ''),
          pct(abilityTurns / turns),
          actions.sort((a, b) => b[1] - a[1]).map(([k, v]) => `${k} ${pct(v / turns)}`).join(', '),
          free.map(([k, v]) => `${k} ${((10 * v) / turns).toFixed(1)}`).join(', '),
        ];
      }),
    );
  }

  // Each encounter alone, against a fresh party: how hard is it on its own?
  const fresh = plan.encounters.map((e) => {
    const roll = seededRoller(7);
    let won = 0;
    let lost = 0;
    let n = 0;
    const tries = Math.max(500, Math.floor(runs / 4));
    for (let i = 0; i < tries; i++) {
      const heroes = PARTY.map(newHero);
      const r = runFight(heroes, e.monsters.map(monsterSpec), tactics, { ...SUPPLIES, healPotions: 0, manaPotions: 0 }, roll);
      won += r.won ? 1 : 0;
      n += r.rounds;
      lost += heroes.reduce((s, h) => s + (h.maxBody - h.body), 0) / heroes.reduce((s, h) => s + h.maxBody, 0);
    }
    return { won: won / tries, rounds: n / tries, lost: lost / tries };
  });
  table(
    'Encounters (route rooms first, the rest in a random order each run)',
    ['Encounter', 'Monsters', 'Quest runs wiped here', 'Avg rounds', 'Fresh party: wins / rounds / Body lost'],
    plan.encounters.map((e, j) => {
      const rs = rounds.get(e.name) ?? [];
      const f = fresh[j];
      return [
        e.name,
        e.monsters.join(', '),
        pct((wipes.get(e.name) ?? 0) / runs),
        rs.length ? (rs.reduce((s, v) => s + v, 0) / rs.length).toFixed(1) : '-',
        f ? `${pct(f.won)} / ${f.rounds.toFixed(1)} / ${pct(f.lost)}` : '-',
      ];
    }),
  );
}
