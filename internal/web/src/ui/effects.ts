/**
 * Effects on a hero or monster: a list with remove buttons and a small add form.
 * Each control sends one command; the tracker only counts effects down.
 */
import { EFFECT_SUGGESTIONS, effectLabel } from '../tracker/effects.ts';
import type { Command, Effect } from '../tracker/types.ts';
import { h } from './dom.ts';

const field = 'rounded-md border border-border/60 bg-surface px-2 py-1 text-xs';
const btn = 'rounded-md border border-border/60 px-2 py-0.5 text-xs hover:border-amber-500';

/** One datalist of suggested names, shared by every add form on the page. */
export function effectSuggestions(): HTMLElement {
  return h('datalist', { id: 'effect-suggestions' }, ...EFFECT_SUGGESTIONS.map((name) => h('option', { value: name })));
}

/** targetId is a hero or monster id; who names it in labels. */
export function effectsBlock(targetId: string, who: string, effects: readonly Effect[] | null | undefined, send: (c: Command) => void): HTMLElement {
  const name = h('input', { class: `${field} min-w-0 flex-1`, list: 'effect-suggestions', maxlength: 40, placeholder: 'Effect', 'aria-label': `New effect on ${who}` });
  const rounds = h('input', { class: `${field} w-14`, type: 'number', min: 0, max: 99, placeholder: 'Rnds', title: 'Rounds (counted down each fight round; empty lasts until removed)', 'aria-label': `Rounds for the new effect on ${who}` });
  const note = h('input', { class: `${field} min-w-0 flex-1`, maxlength: 200, placeholder: 'Note', 'aria-label': `Note for the new effect on ${who}` });
  const add = (e: Event): void => {
    e.preventDefault();
    const n = name.value.trim();
    if (!n) {
      return;
    }
    send({ type: 'effect.add', payload: { target: targetId, name: n, rounds: Number(rounds.value) || 0, note: note.value.trim() } });
  };
  return h('div', { class: 'space-y-1' },
    (effects ?? []).length > 0
      ? h('ul', { class: 'flex flex-wrap gap-1' }, ...(effects ?? []).map((e) =>
        h('li', { class: 'flex items-center gap-1 rounded-full border border-amber-500/50 bg-amber-500/10 px-2 py-0.5 text-xs' },
          effectLabel(e),
          h('button', { type: 'button', class: 'opacity-60 hover:text-danger hover:opacity-100', title: `Remove ${e.name}`, 'aria-label': `Remove ${e.name} from ${who}`, onclick: () => { send({ type: 'effect.remove', payload: { target: targetId, id: e.id } }); } }, '×'))))
      : null,
    h('form', { class: 'flex gap-1', onsubmit: add }, name, rounds, note, h('button', { type: 'submit', class: btn }, 'Add')));
}
