/**
 * The GM's rules console (online play): where the round stands, the phase
 * buttons, and in the monsters' phase each monster's attacks. Selecting a
 * monster highlights the squares it may move to; clicking one moves it by
 * the rules (pages/tracker.ts). The GM can still do anything by hand.
 */
import { gmConsole, rulesLines } from '../tracker/rules.ts';
import type { Action } from '../seat/types.ts';
import type { Command, SessionState } from '../tracker/types.ts';
import { h } from './dom.ts';

const btn = 'rounded-md border border-border/60 px-2 py-1 text-sm hover:border-amber-500 disabled:opacity-40';
const btnMain = 'rounded-md bg-amber-600 px-2 py-1 text-sm font-semibold text-white hover:bg-amber-700 disabled:opacity-40';

export interface RulesConsoleContext {
  state: SessionState;
  actions: Action[];
  selectedId: string | null;
  disabled: boolean;
  send: (c: Command) => void;
  select: (id: string) => void;
}

/** The console, or null while the rules are off. */
export function rulesConsole(ctx: RulesConsoleContext): HTMLElement | null {
  const { state: s, disabled, send } = ctx;
  const r = s.rules;
  if (!r) {
    return null;
  }
  const c = gmConsole(s, ctx.actions);
  const [title, ...lines] = rulesLines(s);
  const turnHero = r.turn ? s.heroes.find((x) => x.id === r.turn?.heroId) : undefined;
  const phaseButtons = c.phase.map((a) => h('button', {
    type: 'button',
    class: a.command.type === 'phase.monsters' ? btnMain : btn,
    disabled,
    onclick: () => { send(a.command); },
  }, a.label));
  const monsters = r.phase === 'monsters'
    ? c.monsters.map((m) => h('li', { class: `space-y-1 rounded-md border p-2 ${m.id === ctx.selectedId ? 'border-amber-500 bg-amber-500/10' : 'border-border/60'}` },
      h('button', { type: 'button', class: 'text-left text-sm font-semibold hover:text-amber-400', onclick: () => { ctx.select(m.id); } }, m.label),
      m.moves.size
        ? h('p', { class: 'text-xs opacity-70' }, m.id === ctx.selectedId ? `Click a highlighted square to move it (${String(m.moves.size)} in reach).` : 'Select it to see where it can move.')
        : null,
      m.attacks.length
        ? h('div', { class: 'flex flex-wrap gap-1' }, ...m.attacks.map((a) => h('button', { type: 'button', class: btn, disabled, onclick: () => { send(a.command); } }, a.label)))
        : null))
    : [];
  return h('section', { class: 'space-y-2 rounded-lg border border-amber-500/40 bg-surface-2 p-3', 'aria-label': 'Rules' },
    h('h2', { class: 'text-sm font-semibold text-amber-400' }, title ?? 'Rules'),
    lines.length ? h('ul', { class: 'space-y-0.5 text-sm' }, ...lines.map((l) => h('li', {}, l))) : null,
    turnHero
      ? h('button', {
        type: 'button',
        class: btn,
        disabled,
        title: 'Ends the turn for the player (say, if they dropped out)',
        onclick: () => { send({ type: 'turn.end', payload: {} }); },
      }, `End ${turnHero.name}'s turn`)
      : null,
    monsters.length ? h('ul', { class: 'space-y-2' }, ...monsters) : null,
    r.phase === 'monsters' && !monsters.length ? h('p', { class: 'text-sm opacity-70' }, 'No monster can move or attack.') : null,
    phaseButtons.length ? h('div', { class: 'flex flex-wrap gap-2' }, ...phaseButtons) : null);
}
