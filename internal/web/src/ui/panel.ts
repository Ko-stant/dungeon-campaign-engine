/**
 * A collapsible side-panel section: the title is a button that opens and
 * closes it, extra controls (Reload, search) sit at the right of the header,
 * and a closed section renders only its header.
 */
import { h } from './dom.ts';

export interface PanelOptions {
  title: string;
  /** Shown after the title in a lighter weight, e.g. "3 alive". */
  summary?: string;
  open: boolean;
  onToggle: (open: boolean) => void;
  /** Controls at the right of the header, shown only while open. */
  extra?: (Node | null)[];
  /** Tailwind spacing class for the body, e.g. 'space-y-2'. */
  spacing?: string;
}

export function panel(opts: PanelOptions, ...body: (Node | null)[]): HTMLElement {
  return h('section', { class: opts.open ? (opts.spacing ?? 'space-y-1') : '' },
    h('div', { class: 'flex items-center justify-between gap-2' },
      h('button', {
        type: 'button',
        class: 'flex min-w-0 flex-1 items-baseline gap-1 text-left text-sm font-semibold hover:text-amber-400',
        'aria-expanded': opts.open ? 'true' : 'false',
        title: opts.open ? `Hide ${opts.title}` : `Show ${opts.title}`,
        onclick: () => { opts.onToggle(!opts.open); },
      },
        h('span', { class: 'w-3 shrink-0 text-xs opacity-60', 'aria-hidden': 'true' }, opts.open ? '▾' : '▸'),
        opts.title,
        opts.summary ? h('span', { class: 'ml-1 text-xs font-normal opacity-60' }, opts.summary) : null),
      ...(opts.open ? opts.extra ?? [] : [])),
    ...(opts.open ? body : []));
}
