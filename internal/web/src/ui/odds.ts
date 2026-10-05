/**
 * The monster panel's odds hints: each hero attacking the monster, and the
 * monster attacking each hero. Advice only (see tracker/odds.ts).
 */
import { attackOdds, defenseOdds, oddsPercent, smiteOdds } from '../tracker/odds.ts';
import type { Hero, Monster } from '../tracker/types.ts';
import { h } from './dom.ts';

/** "~3.2 attacks", "1 attack" or "can't hurt it". */
function attacksText(n: number): string {
  if (!Number.isFinite(n)) {
    return "can't hurt it";
  }
  return n < 1.05 ? '1 attack' : `~${n.toFixed(1)} attacks`;
}

function row(name: string, text: string): HTMLElement {
  return h('li', {}, h('span', { class: 'font-semibold' }, name), h('span', { class: 'opacity-80' }, `: ${text}`));
}

/** Null when no hero and the monster both have combat stats. open/onToggle keep the block open or closed across re-renders. */
export function oddsBlock(monster: Monster, heroes: readonly Hero[], open: boolean, onToggle: (open: boolean) => void): HTMLElement | null {
  const attacks: HTMLElement[] = [];
  const defenses: HTMLElement[] = [];
  for (const hero of heroes) {
    const a = attackOdds(hero, monster);
    if (a) {
      attacks.push(row(hero.name, `hits ${oddsPercent(a.hit)} · crit ${oddsPercent(a.crit)} · ${attacksText(a.attacksToKill)} to finish`));
    }
    const s = smiteOdds(hero, monster);
    if (s) {
      attacks.push(row(`${hero.name} (Smite)`, `hits ${oddsPercent(s.hit)} · crit ${oddsPercent(s.crit)} · ${attacksText(s.attacksToKill)} to finish`));
    }
    const d = defenseOdds(monster, hero);
    if (d) {
      defenses.push(row(hero.name, `${oddsPercent(d.hit)} to hit · ${String(d.damage)} a hit`));
    }
  }
  if (attacks.length === 0 && defenses.length === 0) {
    return null;
  }
  return h('details', {
    class: 'rounded-md border border-border/40 px-2 py-1 text-xs',
    open,
    ontoggle: (e: Event) => { onToggle((e.currentTarget as HTMLDetailsElement).open); },
  },
    h('summary', { class: 'cursor-pointer font-semibold uppercase tracking-wide opacity-70' }, 'Odds', h('span', { class: 'ml-2 font-normal normal-case' }, 'advice only')),
    h('div', { class: 'mt-1 space-y-1' },
      attacks.length ? h('p', { class: 'opacity-60' }, `Heroes attacking the ${monster.name}`) : null,
      attacks.length ? h('ul', { class: 'space-y-0.5' }, ...attacks) : null,
      defenses.length ? h('p', { class: 'opacity-60' }, `The ${monster.name} attacking`) : null,
      defenses.length ? h('ul', { class: 'space-y-0.5' }, ...defenses) : null,
      h('p', { class: 'opacity-50' }, 'Class stats and equipped items, Smite, current Body, Faltering and Determination; other effects, abilities and spells are not counted.')));
}
