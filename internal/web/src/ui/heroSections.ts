/**
 * The tracker hero card's Abilities and Inventory sections, and the party's
 * purse. Each control sends one command; nothing is refused (cooling down or
 * short of mana is shown, not enforced).
 */
import { abilityLimit, abilityRows, goldChange } from '../tracker/abilities.ts';
import { itemStatsLine } from '../tracker/combat.ts';
import type { Command, Hero, Item, ItemStats } from '../tracker/types.ts';
import { h } from './dom.ts';

const btn = 'rounded-md border border-border/60 px-2 py-0.5 text-xs hover:border-amber-500 disabled:opacity-40';
const smallBtn = 'h-6 w-6 shrink-0 rounded border border-border/60 text-sm leading-none hover:border-amber-500';
const field = 'rounded-md border border-border/60 bg-surface px-2 py-1 text-sm';

export interface SectionContext {
  send: (c: Command) => void;
  /** Shows a message in the header, e.g. a gold amount that cannot be read. */
  warn: (message: string) => void;
  /** Collapsible sections remember whether they are open across re-renders. */
  isOpen: (key: string) => boolean;
  setOpen: (key: string, open: boolean) => void;
}

function section(ctx: SectionContext, key: string, title: string, summary: string, ...body: (HTMLElement | null)[]): HTMLElement {
  return h('details', {
    class: 'rounded-md border border-border/40 px-2 py-1',
    open: ctx.isOpen(key),
    ontoggle: (e: Event) => { ctx.setOpen(key, (e.currentTarget as HTMLDetailsElement).open); },
  },
    h('summary', { class: 'cursor-pointer text-xs font-semibold uppercase tracking-wide opacity-70' }, title, summary ? h('span', { class: 'ml-2 font-normal normal-case' }, summary) : null),
    h('div', { class: 'mt-2 space-y-2' }, ...body));
}

export function abilitySection(hero: Hero, round: number, ctx: SectionContext): HTMLElement | null {
  const rows = abilityRows(hero, round);
  if (rows.length === 0) {
    return null;
  }
  const cooling = rows.filter((r) => r.roundsLeft > 0).length;
  const items = rows.map((r) => {
    const statusClass = r.roundsLeft > 0 ? 'text-warning' : r.usable ? 'text-positive' : 'opacity-60';
    const limit = abilityLimit(r.ability);
    return h('li', { class: 'space-y-0.5' },
      h('div', { class: 'flex items-center gap-2 text-sm' },
        h('span', { class: 'min-w-0 flex-1 truncate font-semibold', title: r.ability.text ?? '' }, r.ability.name),
        h('span', { class: `text-xs ${statusClass}` }, r.status),
        r.roundsLeft > 0
          ? h('button', { type: 'button', class: btn, title: 'Make it ready now (Divine Blessing, or a correction)', onclick: () => { ctx.send({ type: 'ability.reset', payload: { heroId: hero.id, abilityId: r.ability.id } }); } }, 'Ready')
          : null,
        r.usable
          ? h('button', {
            type: 'button',
            class: `${btn} ${r.roundsLeft > 0 || r.shortOfMana ? 'border-warning/60' : ''}`,
            title: r.roundsLeft > 0 ? 'Still cooling down; the use is recorded anyway' : r.shortOfMana ? 'Not enough mana; the use is recorded anyway' : 'Record a use',
            onclick: () => { ctx.send({ type: 'ability.use', payload: { heroId: hero.id, abilityId: r.ability.id } }); },
          }, r.ability.kind === 'spell' ? 'Cast' : 'Use')
          : null),
      h('p', { class: 'text-xs opacity-60' }, [r.ability.kind, limit].filter(Boolean).join(' · ')));
  });
  return section(ctx, `${hero.id}:abilities`, 'Abilities', cooling ? `${String(cooling)} cooling down` : 'all ready',
    h('ul', { class: 'space-y-2' }, ...items),
    cooling > 1
      ? h('button', { type: 'button', class: btn, onclick: () => { ctx.send({ type: 'ability.reset', payload: { heroId: hero.id } }); } }, 'Make all ready')
      : null);
}

/** The party's purse: "+25" adds, "-10" takes away, "40" sets. */
export function purseControl(gold: number, ctx: SectionContext): HTMLElement {
  const input = h('input', { class: `${field} w-24 min-w-0 flex-1`, placeholder: '+25, -10, 40', 'aria-label': "Change the party's gold" });
  const apply = (): void => {
    if (input.value.trim() === '') {
      return;
    }
    const r = goldChange(gold, input.value);
    if (!r.ok) {
      ctx.warn(r.error);
      return;
    }
    input.value = '';
    if (r.gold !== gold) {
      ctx.send({ type: 'gold.set', payload: { gold: r.gold } });
    }
  };
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      apply();
    }
  });
  return h('div', { class: 'flex items-center gap-2 rounded-lg border border-border/60 px-3 py-2 text-sm' },
    h('span', { class: 'whitespace-nowrap opacity-70' }, 'Party gold'),
    h('span', { class: 'font-mono font-semibold text-amber-400' }, String(gold)),
    input,
    h('button', { type: 'button', class: btn, onclick: apply }, 'Apply'));
}

const STAT_INPUTS: [keyof ItemStats, string][] = [
  ['damage', 'Damage'], ['accuracy', 'Accuracy'], ['avoidance', 'Avoidance'], ['mitigation', 'Mitigation'], ['mana', 'Mana'], ['manaRegen', 'Regen'],
];

/** An item's kind and stats editor, closed until opened (remembered across re-renders). */
function itemEditor(hero: Hero, it: Item, ctx: SectionContext): HTMLElement {
  // Sections default to open, so the remembered key here means "opened".
  const key = `${hero.id}:item:${it.id}`;
  const kind = h('input', { class: `${field} w-full text-xs`, maxlength: 40, value: it.kind ?? '', placeholder: 'Kind: weapon, bow, chest...', 'aria-label': `${it.name} kind` });
  const inputs = STAT_INPUTS.map(([k, label]) => [k, h('input', {
    class: `${field} w-full min-w-0 px-1 text-xs`, type: 'number', min: -99, max: 99, value: it[k] ? String(it[k]) : '', placeholder: '0', 'aria-label': `${it.name} ${label}`,
  })] as const);
  const save = (): void => {
    const stats: ItemStats = {};
    for (const [k, input] of inputs) {
      const n = Math.trunc(Number(input.value) || 0);
      if (n !== 0) {
        stats[k] = n;
      }
    }
    ctx.send({ type: 'item.update', payload: { heroId: hero.id, itemId: it.id, kind: kind.value.trim(), stats } });
  };
  return h('details', {
    class: 'pl-1',
    open: !ctx.isOpen(key),
    ontoggle: (e: Event) => { ctx.setOpen(key, !(e.currentTarget as HTMLDetailsElement).open); },
  },
    h('summary', { class: 'cursor-pointer text-xs opacity-60' }, 'Kind and stats'),
    h('div', { class: 'mt-1 space-y-1' },
      kind,
      h('div', { class: 'grid grid-cols-3 gap-x-2 gap-y-1' }, ...inputs.map(([k, input]) => h('label', { class: 'block text-xs' },
        h('span', { class: 'block opacity-70' }, STAT_INPUTS.find(([s]) => s === k)?.[1] ?? k), input))),
      h('button', { type: 'button', class: btn, onclick: save }, 'Save stats')));
}

export function inventorySection(hero: Hero, party: readonly Hero[], ctx: SectionContext): HTMLElement {
  const items = hero.items ?? [];
  const others = party.filter((o) => o.id !== hero.id);

  const rows = items.map((it) => h('li', { class: 'space-y-1 border-b border-border/30 pb-1 last:border-0' },
    h('div', { class: 'flex items-baseline gap-1 text-sm' },
      h('span', { class: `min-w-0 flex-1 break-words ${it.equipped ? 'font-semibold' : ''}`, title: it.notes ?? '' }, it.name, it.notes ? h('span', { class: 'ml-1 text-xs opacity-50' }, '*') : null),
      h('button', { type: 'button', class: 'px-1 text-xs text-danger hover:underline', 'aria-label': `Remove all ${it.name}`, title: 'Remove all', onclick: () => { ctx.send({ type: 'item.remove', payload: { heroId: hero.id, itemId: it.id } }); } }, '✕')),
    itemTag(it) ? h('p', { class: 'text-xs opacity-60' }, itemTag(it)) : null,
    h('div', { class: 'flex flex-wrap items-center gap-1 text-sm' },
      h('button', {
        type: 'button',
        class: `${btn} ${it.equipped ? 'border-positive/60 text-positive' : ''}`,
        title: it.equipped ? 'Equipped: its stats count. Click to unequip.' : 'Equip it so its stats count',
        'aria-label': `${it.equipped ? 'Unequip' : 'Equip'} ${it.name}`,
        onclick: () => { ctx.send({ type: 'item.equip', payload: { heroId: hero.id, itemId: it.id, equipped: !it.equipped } }); },
      }, it.equipped ? 'Equipped' : 'Equip'),
      h('button', { type: 'button', class: smallBtn, 'aria-label': `One less ${it.name}`, onclick: () => { ctx.send({ type: 'item.remove', payload: { heroId: hero.id, itemId: it.id, quantity: 1 } }); } }, '−'),
      h('span', { class: 'w-8 text-center font-mono' }, String(it.quantity)),
      h('button', { type: 'button', class: smallBtn, 'aria-label': `One more ${it.name}`, onclick: () => { ctx.send({ type: 'item.update', payload: { heroId: hero.id, itemId: it.id, quantity: it.quantity + 1 } }); } }, '+'),
      others.length
        ? h('select', {
          class: `${field} w-16 px-1 text-xs`,
          'aria-label': `Give one ${it.name} to`,
          title: 'Give one to another hero',
          onchange: (e: Event) => {
            const to = (e.target as HTMLSelectElement).value;
            if (to) {
              ctx.send({ type: 'item.give', payload: { heroId: hero.id, itemId: it.id, toHeroId: to, quantity: 1 } });
            }
          },
        }, h('option', { value: '' }, 'Give'), ...others.map((o) => h('option', { value: o.id }, o.name)))
        : null),
    itemEditor(hero, it, ctx)));

  const nameInput = h('input', { class: `${field} min-w-0 flex-1`, placeholder: 'Add an item', maxlength: 120, 'aria-label': `New item for ${hero.name}` });
  const qtyInput = h('input', { class: `${field} w-14`, type: 'number', min: 1, max: 9999, value: 1, 'aria-label': 'Quantity' });
  const addItem = (): void => {
    const name = nameInput.value.trim();
    if (name) {
      const quantity = Number(qtyInput.value) || 1;
      nameInput.value = '';
      ctx.send({ type: 'item.add', payload: { heroId: hero.id, name, quantity } });
    }
  };
  nameInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      addItem();
    }
  });

  const count = items.reduce((n, it) => n + it.quantity, 0);
  const equipped = items.filter((it) => it.equipped).length;
  return section(ctx, `${hero.id}:inventory`, 'Inventory', `${String(count)} item${count === 1 ? '' : 's'}${equipped ? ` · ${String(equipped)} equipped` : ''}`,
    items.length ? h('ul', { class: 'space-y-1' }, ...rows) : h('p', { class: 'text-xs opacity-60' }, 'No items.'),
    h('div', { class: 'flex items-center gap-1' }, nameInput, qtyInput, h('button', { type: 'button', class: btn, onclick: addItem }, 'Add')));
}

/** An item's kind and stats on one line, e.g. "bow · damage +6". */
function itemTag(it: Item): string {
  return [it.kind ?? '', itemStatsLine(it)].filter(Boolean).join(' · ');
}
