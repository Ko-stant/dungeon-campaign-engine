/**
 * Writes the GM bestiary page (docs/campaigns/three-plagues/gm/bestiary.html)
 * from the agreed numbers: monsters from combat.json, the party with its
 * starting gear and with every Quest 1 find (scripts/combat-config.ts), and the
 * exact odds from internal/web/src/combat/odds.ts. Rerun after changing the
 * numbers:  bun scripts/gm-bestiary.ts
 */
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import combat from '../docs/campaigns/three-plagues/combat.json';
import { CAMPAIGN_RULES, heroAttack, killOdds, monsterAttack, MONSTER_CRIT_CHANCE, type HeroAttacker } from '../internal/web/src/combat/odds.ts';
import type { HeroSpec, MonsterSpec } from '../internal/web/src/combat/simulate.ts';
import { MONSTERS, PARTY, PARTY_AFTER_QUEST_1, QUEST_1, TACTICS } from './combat-config.ts';

const OUT = 'docs/campaigns/three-plagues/gm/bestiary.html';

interface Line {
  body: number;
  avoidance: number;
  hitDice: string;
  damage: number;
  movement?: number;
  ranged?: boolean;
  reach?: boolean;
  line?: number;
  splashDamage?: number;
  splashTargets?: number;
  undead?: boolean;
}

// Body-only lines (no hit dice: a monster that doesn't fight) have no odds to show.
const LINES = Object.fromEntries(Object.entries(combat.monsters as Record<string, Partial<Line>>).filter((e): e is [string, Line] => e[1].hitDice !== undefined));
const HERO_NAMES: Record<string, string> = { Barbarian: 'Brentanamo', Ranger: 'Mordecai', Rogue: 'Papi', Cleric: 'Derrick' };

const esc = (s: string): string => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
const pct = (p: number): string => (p > 0 && p < 0.005 ? '<1%' : p < 1 && p >= 0.995 ? '>99%' : `${String(Math.round(p * 100))}%`);
const one = (n: number): string => (Number.isFinite(n) ? n.toFixed(1) : '-');
const title = (type: string): string => type.split('_').map((w) => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
const falterAt = (body: number): number => Math.max(1, Math.floor(body / 4));
const band = (p: number): string => (p >= 0.8 ? 'high' : p >= 0.5 ? 'mid' : 'low');

/**
 * Squares a monster moves, as the tracker has it: the campaign stat line's movement, else the
 * base game's from the gitignored content/ catalog (undefined when neither is there).
 */
function movement(t: string): number | undefined {
  const own = LINES[t]?.movement;
  if (own) return own;
  const file = `content/monsters/${t}.json`;
  if (!existsSync(file)) return undefined;
  const base = (JSON.parse(readFileSync(file, 'utf8')) as { stats?: { movementSquares?: number } }).stats?.movementSquares;
  return base !== undefined && base > 0 ? base : undefined;
}

function traits(l: Line): string[] {
  const out: string[] = [];
  if (l.ranged) out.push('ranged');
  if (l.reach) out.push('reach');
  if (l.line) out.push(`strikes ${String(l.line + 1)} in a line`);
  if (l.splashDamage && l.splashTargets) out.push(`blast ${String(l.splashDamage)} to ${String(l.splashTargets)} beside the target`);
  if (l.undead) out.push('undead');
  return out;
}

/** How many of each monster type Quest 1 has (from the generated encounters). */
function quest1Counts(): Map<string, number> {
  const data = JSON.parse(readFileSync(QUEST_1.file, 'utf8')) as { encounters: { monsters: string[] }[] };
  const counts = new Map<string, number>();
  for (const e of data.encounters) {
    for (const m of e.monsters) {
      counts.set(m, (counts.get(m) ?? 0) + 1);
    }
  }
  return counts;
}

interface Attacker {
  label: string;
  attack: HeroAttacker;
  undeadBonus: number;
}

/** Each hero's attacks: their weapon, and Smite for the Cleric. */
function attackers(party: HeroSpec[]): Attacker[] {
  const out: Attacker[] = [];
  for (const h of party) {
    const name = HERO_NAMES[h.name] ?? h.name;
    out.push({ label: h.cls === 'cleric' ? `${name} (mace)` : name, attack: h.attack, undeadBonus: 0 });
    if (h.cls === 'cleric' && TACTICS.smite) {
      out.push({ label: `${name} (Smite)`, attack: { ...h.attack, ...TACTICS.smite.attack }, undeadBonus: TACTICS.smite.undead });
    }
  }
  return out;
}

function vsMonster(a: Attacker, m: MonsterSpec): { hit: number; hitWounded: number; perAttack: number; toKill: number } {
  const atk = m.undead && a.undeadBonus ? { ...a.attack, damage: a.attack.damage + a.undeadBonus } : a.attack;
  const normal = heroAttack(atk, m.avoidance);
  const wounded = heroAttack(atk, m.avoidance - CAMPAIGN_RULES.falterPenalty);
  const kill = killOdds(atk, m.avoidance, m.body, { ...CAMPAIGN_RULES, falterAt: falterAt(m.body) });
  return { hit: normal.hit, hitWounded: wounded.hit, perAttack: normal.expectedDamage, toKill: kill.expected };
}

function defenseRange(h: HeroSpec): string {
  const d = h.defense.defenseDice;
  const lo = d.terms.reduce((n, t) => n + t.count, 0) + d.modifier + h.defense.baseAvoidance;
  const hi = d.terms.reduce((n, t) => n + t.count * t.sides, 0) + d.modifier + h.defense.baseAvoidance;
  return `${String(lo)}-${String(hi)}`;
}

function rollRange(m: MonsterSpec): string {
  const d = m.attack.hitDice;
  const lo = d.terms.reduce((n, t) => n + t.count, 0) + d.modifier;
  const hi = d.terms.reduce((n, t) => n + t.count * t.sides, 0) + d.modifier;
  return `${String(lo)}-${String(hi)}`;
}

const types = Object.keys(MONSTERS).sort((a, b) => (MONSTERS[a]?.body ?? 0) - (MONSTERS[b]?.body ?? 0) || a.localeCompare(b));
const counts = quest1Counts();

// --- "Can the heroes avoid hits?" ---
function defenseGrid(party: HeroSpec[], caption: string): string {
  const head = party.map((h) => `<th scope="col">${esc(HERO_NAMES[h.name] ?? h.name)}<small>defends ${defenseRange(h)}${h.defense.mitigation ? `, mitigation ${String(h.defense.mitigation)}` : ''}</small></th>`).join('');
  const rows = types.map((t) => {
    const m = MONSTERS[t];
    if (!m) return '';
    const cells = party.map((h) => {
      const o = monsterAttack(m.attack, h.defense);
      const perHit = Math.max(0, m.attack.damage - h.defense.mitigation);
      return `<td class="${band(o.hit)}"><b>${pct(o.hit)}</b><small>${String(perHit)} a hit · ${one(o.expectedDamage)} avg</small></td>`;
    }).join('');
    const move = movement(t);
    return `<tr><th scope="row">${esc(title(t))}<small>rolls ${esc(LINES[t]?.hitDice ?? '')} (${rollRange(m)}), ${String(m.attack.damage)} damage${move ? `, moves ${String(move)}` : ''}</small></th>${cells}</tr>`;
  }).join('\n');
  return `<div class="scroll"><table class="grid"><caption>${caption}</caption><thead><tr><th scope="col">Monster</th>${head}</tr></thead><tbody>${rows}</tbody></table></div>`;
}

// --- One entry per monster ---
function entry(t: string): string {
  const m = MONSTERS[t];
  const l = LINES[t];
  if (!m || !l) return '';
  const start = attackers(PARTY);
  const geared = attackers(PARTY_AFTER_QUEST_1);
  const rows = start.map((a, i) => {
    const s = vsMonster(a, m);
    const g = geared[i] ? vsMonster(geared[i], m) : s;
    return `<tr><th scope="row">${esc(a.label)}</th><td>${pct(s.hit)}</td><td>${pct(s.hitWounded)}</td><td>${one(s.perAttack)}</td><td>${one(s.toKill)}</td><td class="after">${one(g.toKill)}</td></tr>`;
  }).join('');
  const defRows = PARTY.map((h, i) => {
    const o = monsterAttack(m.attack, h.defense);
    const after = PARTY_AFTER_QUEST_1[i] ? monsterAttack(m.attack, PARTY_AFTER_QUEST_1[i].defense) : o;
    return `<tr><th scope="row">${esc(HERO_NAMES[h.name] ?? h.name)}</th><td class="${band(o.hit)}">${pct(o.hit)}</td><td>${String(Math.max(0, m.attack.damage - h.defense.mitigation))}</td><td>${one(o.expectedDamage)}</td><td class="after ${band(after.hit)}">${pct(after.hit)}</td></tr>`;
  }).join('');
  // The whole party, one attack each (the Cleric's better of mace and Smite).
  const best = new Map<string, number>();
  for (const a of start) {
    const hero = a.label.split(' ')[0] ?? a.label;
    best.set(hero, Math.max(best.get(hero) ?? 0, vsMonster(a, m).perAttack));
  }
  const partyRound = [...best.values()].reduce((n, v) => n + v, 0);
  const tr = traits(l);
  const n = counts.get(t) ?? 0;
  const move = movement(t);
  return `
  <article class="monster" id="${t}">
    <header>
      <h2>${esc(title(t))}</h2>
      <p class="line"><span><b>${String(l.body)}</b> Body</span><span>Avoidance <b>${String(l.avoidance)}</b></span><span>Hit <b>${esc(l.hitDice)}</b></span><span>Damage <b>${String(l.damage)}</b></span>${move ? `<span>Move <b>${String(move)}</b></span>` : ''}${tr.map((x) => `<span class="trait">${esc(x)}</span>`).join('')}</p>
      <p class="meta">Wounded (Avoidance ${String(l.avoidance - CAMPAIGN_RULES.falterPenalty)}) at ${String(falterAt(l.body))} Body or less · crits on double 20s (${(MONSTER_CRIT_CHANCE * 100).toFixed(2)}%) for ${String(l.damage * 2)} · ${n ? `${String(n)} in Quest 1` : t === 'specter' ? 'Quest 1 only if the Stranger is never freed' : 'not in Quest 1'} · the whole party deals about ${one(partyRound)} a round, so about ${one(l.body / partyRound)} rounds to fell it</p>
    </header>
    <div class="cols">
      <div class="scroll"><table class="small">
        <caption>The heroes attacking it (starting gear)</caption>
        <thead><tr><th>Hero</th><th>Hits</th><th>Hits Wounded</th><th>Avg damage</th><th>Attacks to kill alone</th><th class="after">...with Quest 1 finds</th></tr></thead>
        <tbody>${rows}</tbody>
      </table></div>
      <div class="scroll"><table class="small">
        <caption>It attacking the heroes</caption>
        <thead><tr><th>Hero</th><th>Hits</th><th>Damage a hit</th><th>Avg an attack</th><th class="after">Hits after Quest 1 finds</th></tr></thead>
        <tbody>${defRows}</tbody>
      </table></div>
    </div>
  </article>`;
}

const html = `<title>GM Bestiary</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Pirata+One&family=Alegreya:ital,wght@0,400;0,600;0,700;1,400&family=Alegreya+Sans+SC:wght@500;700&display=swap">
<style>
/* Layout: a working reference for the GM. The defense grid first, then one entry per monster from weakest to strongest. Generated by scripts/gm-bestiary.ts. */
:root {
  color-scheme: dark;
  --stone: #131416;
  --stone-raised: #1b1e21;
  --stone-line: #33383d;
  --gold: #d6a24a;
  --gold-bright: #f0c46c;
  --ink: #ebe4d4;
  --ink-muted: #a59e90;
  --blood: #d0614b;
  --blood-wash: rgb(208 97 75 / 0.16);
  --amber-wash: rgb(214 162 74 / 0.14);
  --moss-wash: rgb(110 160 115 / 0.16);
  --moss: #8fbf8f;
  --display: "Pirata One", Georgia, serif;
  --body: "Alegreya", Georgia, "Times New Roman", serif;
  --label: "Alegreya Sans SC", "Alegreya", Georgia, serif;
}
* { box-sizing: border-box; }
body { background: var(--stone); color: var(--ink); font-family: var(--body); font-size: 1.02rem; line-height: 1.5; padding-inline: 16px; padding-block: 0 4rem; }
.wrap { max-width: 76rem; margin-inline: auto; display: grid; gap: 2.2rem; }
h1, h2, h3 { font-family: var(--display); font-weight: 400; color: var(--gold-bright); line-height: 1.1; margin: 0; text-wrap: balance; }
h1 { font-size: clamp(2.4rem, 7vw, 3.8rem); }
h2 { font-size: 2rem; }
p { margin: 0; }
.masthead { padding-block: 2.4rem 0; display: grid; gap: 0.7rem; }
.secret { font-family: var(--label); letter-spacing: 0.1em; color: var(--blood); font-size: 0.9rem; }
.prose { max-width: 46rem; display: grid; gap: 0.7rem; color: var(--ink); }
.prose ul { margin: 0; padding-left: 1.2rem; display: grid; gap: 0.35rem; }
nav.index { display: flex; flex-wrap: wrap; gap: 0.3rem 1rem; font-family: var(--label); letter-spacing: 0.05em; }
nav.index a { color: var(--ink-muted); text-decoration: none; }
nav.index a:hover { color: var(--gold-bright); }
.scroll { overflow-x: auto; min-width: 0; }
table { border-collapse: collapse; font-variant-numeric: tabular-nums; }
caption { text-align: left; font-family: var(--label); letter-spacing: 0.06em; color: var(--ink-muted); padding-bottom: 0.4rem; }
th, td { padding: 0.4rem 0.6rem; border-bottom: 1px solid var(--stone-line); text-align: left; vertical-align: top; }
thead th { font-family: var(--label); font-weight: 500; font-size: 0.82rem; letter-spacing: 0.05em; color: var(--ink-muted); }
th small, td small { display: block; font-size: 0.78rem; color: var(--ink-muted); font-weight: 400; }
table.grid td { min-width: 7.5rem; }
table.grid td b { font-size: 1.1rem; }
td.high { background: var(--blood-wash); }
td.high b { color: var(--blood); }
td.mid { background: var(--amber-wash); }
td.low { background: var(--moss-wash); }
td.low b { color: var(--moss); }
.legend { display: flex; flex-wrap: wrap; gap: 0.4rem 1.2rem; font-size: 0.9rem; color: var(--ink-muted); }
.legend span::before { content: ""; display: inline-block; width: 0.8rem; height: 0.8rem; margin-right: 0.35rem; vertical-align: -0.1rem; border-radius: 2px; }
.legend .k-high::before { background: var(--blood); }
.legend .k-mid::before { background: var(--gold); }
.legend .k-low::before { background: var(--moss); }
.monster { border-top: 1px solid var(--stone-line); padding-top: 1.4rem; display: grid; gap: 1rem; scroll-margin-top: 1rem; }
.monster header { display: grid; gap: 0.4rem; }
.line { display: flex; flex-wrap: wrap; gap: 0.3rem 1.1rem; font-size: 1.08rem; }
.line b { color: var(--gold-bright); }
.trait { font-style: italic; color: var(--gold-bright); }
.meta { color: var(--ink-muted); font-size: 0.95rem; }
.cols { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 30rem), 1fr)); gap: 1.4rem; }
table.small { font-size: 0.95rem; width: 100%; }
.after { color: var(--ink-muted); }
footer { color: var(--ink-muted); font-size: 0.9rem; border-top: 1px solid var(--stone-line); padding-top: 1rem; }
</style>

<div class="wrap">
  <header class="masthead">
    <span class="secret">For the GM only · do not share with players</span>
    <h1>GM Bestiary</h1>
    <div class="prose">
      <p>Every monster's stat line and how it matches up against the party: how often the heroes hit it and how long it takes to bring down, and how often it gets through each hero's defense. All chances are exact (near misses, crits, Determination and Wounded included) from the same math as the simulator.</p>
      <p>"Starting gear" is the party as it enters Quest 1. "Quest 1 finds" is the party wearing every Quest 1 find: the Wardens' Longbow, Dirk, Chain Shirt, Greatsword and Scale Hauberk, the Quivering Boots and the Prayer Beads.</p>
    </div>
  </header>

  <section class="prose" aria-labelledby="defense">
    <h2 id="defense">Can the heroes avoid hits?</h2>
    <p>A monster hits when its roll beats the hero's defense total (avoidance + defense dice); ties go to the hero. Each cell is the chance it hits, the damage after mitigation, and the average damage per attack (misses and crits included).</p>
    <div class="legend"><span class="k-high">hits 80% or more</span><span class="k-mid">50-79%</span><span class="k-low">under 50%</span></div>
  </section>
  ${defenseGrid(PARTY, 'Starting gear')}
  ${defenseGrid(PARTY_AFTER_QUEST_1, 'With every Quest 1 find')}

  <nav class="index" aria-label="Monsters">${types.map((t) => `<a href="#${t}">${esc(title(t))}</a>`).join('')}</nav>
  ${types.map(entry).join('\n')}

  <footer>Generated from docs/campaigns/three-plagues/combat.json and scripts/combat-config.ts by scripts/gm-bestiary.ts. "Attacks to kill alone" is one hero's expected attacks to take the monster from full Body to 0 (Determination and Wounded included). The party estimate adds one attack per hero per round (the Cleric's better of mace and Smite) and ignores abilities, so real fights go faster. Move is the campaign's movement where combat.json sets one, otherwise the base game's.</footer>
</div>
`;

writeFileSync(OUT, html);
console.log(`Wrote ${OUT} (${String(types.length)} monsters)`);
