/**
 * Simulates the party through Quest 1, with abilities and attack-only:
 *   bun scripts/combat-sim.ts [runs] [--specter] [--melee=N] [--swap "Room 8=gargoyle,goblin_warlock"]...
 * Numbers come from scripts/combat-config.ts; encounters from the generated JSON
 * (scripts/quest-encounters.ts). The route rooms come first, the rest in a random
 * order each run. --swap replaces a room's monsters (repeatable); --melee sets how
 * many melee monsters can attack a round.
 */

import { runFight, runQuest, newHero, seededRoller, type MonsterSpec, type QuestPlan, type Tactics } from '../internal/web/src/combat/simulate.ts';
import { MONSTERS, PARTY, QUEST_1, SUPPLIES, TACTICS } from './combat-config.ts';

const args = process.argv.slice(2);
const runs = Number(args.find((a) => /^\d+$/.test(a)) ?? 4000);
const without = args.includes('--specter') ? [] : QUEST_1.without;
const melee = Number(args.find((a) => a.startsWith('--melee='))?.slice('--melee='.length) ?? TACTICS.meleeLimit);
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
    if (r.wipedAt) {
      wipes.set(r.wipedAt, (wipes.get(r.wipedAt) ?? 0) + 1);
    }
    for (const f of r.fought) {
      rounds.set(f.name, [...(rounds.get(f.name) ?? []), f.rounds]);
    }
  }
  const notes = [`${runs} runs`, `melee limit ${melee}`, ...(without.length ? [`without ${without.join(', ')}`] : []), ...[...swaps].map(([k, v]) => `${k} = ${v.join(', ')}`)];
  console.log(`\n## ${mode} (${notes.join('; ')})`);
  console.log(`Quest cleared: ${pct(cleared / runs)} | cleared with no deaths: ${pct(flawless / runs)} | average deaths: ${(deaths / runs).toFixed(2)}`);
  console.log(`Death rate by hero: ${PARTY.map((h) => `${h.name} ${pct((heroDeaths.get(h.name) ?? 0) / runs)}`).join(', ')}`);

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
