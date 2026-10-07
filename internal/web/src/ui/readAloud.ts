/**
 * The tracker's Read aloud panel (a passage list in the side panel) and the
 * reader (a full-screen view of one passage in large type, for reading at the
 * table). Marking a passage read sends one passage.read command.
 */
import { passageClips, passageNeighbors, sectionProgress } from '../tracker/script.ts';
import type { Command, ScriptPassage, ScriptSection } from '../tracker/types.ts';
import { h } from './dom.ts';
import { panel } from './panel.ts';

const btn = 'rounded-md border border-border/60 px-3 py-1 text-sm hover:border-amber-500 disabled:opacity-40';

export interface ReadAloudContext {
  sections: readonly ScriptSection[];
  /** The section for the active map (-1: none); it starts open. */
  current: number;
  read: readonly string[];
  campaignId: string;
  send: (c: Command) => void;
  open: (passageId: string | null) => void;
  /** Section open/closed state, remembered across re-renders (undefined: default). */
  sectionOpen: (index: number) => boolean | undefined;
  setSectionOpen: (index: number, open: boolean) => void;
  reload: () => void;
  /** Clip id -> URL. */
  clips: Readonly<Record<string, string>>;
  /** The one audio element, kept across re-renders so playback never stops on a live update. */
  player: HTMLAudioElement;
  playClip: (clipId: string) => void;
  autoplay: boolean;
  setAutoplay: (on: boolean) => void;
  /** Passage id -> the letters of the quest notes that hold it (on the map). */
  noteLabels: ReadonlyMap<string, readonly string[]>;
  /** Whether the panel is open (its header always shows). */
  panelOpen: boolean;
  setPanelOpen: (open: boolean) => void;
}

/** "Note B" (or "Notes B, D") for a passage's quest notes. */
function noteTag(labels: readonly string[] | undefined): string {
  if (!labels?.length) {
    return '';
  }
  return `${labels.length === 1 ? 'Note' : 'Notes'} ${labels.join(', ')}`;
}

export function readAloudPanel(ctx: ReadAloudContext): HTMLElement {
  const reload = h('button', { type: 'button', class: 'text-xs opacity-60 hover:opacity-100', title: 'Fetch the script again after editing it on the campaign page', onclick: ctx.reload }, 'Reload');
  const opts = { title: 'Read aloud', open: ctx.panelOpen, onToggle: ctx.setPanelOpen, extra: [reload] };
  if (ctx.sections.length === 0) {
    return panel(opts,
      h('p', { class: 'text-xs opacity-60' }, 'No script yet. ',
        h('a', { href: `/campaigns/${encodeURIComponent(ctx.campaignId)}#script`, class: 'underline hover:text-amber-400' }, 'Add one on the campaign page'),
        ', then ', h('button', { type: 'button', class: 'underline hover:text-amber-400', onclick: ctx.reload }, 'reload it'), '.'));
  }
  const groups = ctx.sections.map((sec, i) => {
    const open = ctx.sectionOpen(i) ?? i === ctx.current;
    return h('details', {
      class: 'rounded-md border border-border/40 px-2 py-1',
      open,
      ontoggle: (e: Event) => { ctx.setSectionOpen(i, (e.currentTarget as HTMLDetailsElement).open); },
    },
      h('summary', { class: `cursor-pointer text-xs font-semibold ${i === ctx.current ? 'text-amber-400' : ''}` },
        sec.title, h('span', { class: 'ml-2 font-normal opacity-60' }, sectionProgress(sec, ctx.read))),
      h('ul', { class: 'mt-1 space-y-0.5' }, ...sec.passages.map((p) => {
        const done = ctx.read.includes(p.id);
        const tag = noteTag(ctx.noteLabels.get(p.id));
        return h('li', {},
          h('button', {
            type: 'button',
            class: `flex w-full items-baseline gap-2 rounded px-1 text-left text-sm hover:bg-amber-500/10 ${done ? 'opacity-50' : ''}`,
            onclick: () => { ctx.open(p.id); },
          },
            h('span', { class: 'w-12 shrink-0 font-mono text-xs opacity-60' }, p.id),
            h('span', { class: 'min-w-0 flex-1' }, p.title),
            tag ? h('span', { class: 'shrink-0 rounded border border-amber-500/50 px-1 text-xs font-semibold text-amber-400', title: `On the map as quest ${tag.toLowerCase()}` }, tag) : null,
            done ? h('span', { class: 'text-xs text-positive', 'aria-label': 'read' }, '✓') : null));
      })));
  });
  return panel(opts, ...groups);
}

function findPassage(sections: readonly ScriptSection[], id: string): { passage: ScriptPassage; section: ScriptSection } | null {
  for (const section of sections) {
    const passage = section.passages.find((p) => p.id === id);
    if (passage) {
      return { passage, section };
    }
  }
  return null;
}

/** The reader overlay for one passage, or null when the id is not in the script. */
export function readerOverlay(ctx: ReadAloudContext, id: string, editable: boolean): HTMLElement | null {
  const found = findPassage(ctx.sections, id);
  if (!found) {
    return null;
  }
  const { passage, section } = found;
  const { prev, next } = passageNeighbors(ctx.sections, id);
  const done = ctx.read.includes(id);
  const mark = (read: boolean): void => { ctx.send({ type: 'passage.read', payload: { id, title: passage.title, read } }); };
  const clips = passageClips(id, ctx.clips);
  const loaded = ctx.player.dataset.clip;
  const audioBar = clips.length
    ? h('div', { class: 'flex flex-wrap items-center gap-2 border-b border-border/60 px-6 py-3' },
      ...clips.map((c) => h('button', {
        type: 'button',
        class: `${btn} ${c.id === loaded ? 'border-amber-500 bg-amber-500/15' : ''}`,
        title: `Play ${c.id}`,
        onclick: () => { ctx.playClip(c.id); },
      }, `▶ ${c.label}`)),
      h('div', { class: 'min-w-60 flex-1' }, ctx.player),
      h('label', { class: 'flex items-center gap-1 text-xs opacity-80', title: 'Start the clip when a passage opens' },
        h('input', { type: 'checkbox', checked: ctx.autoplay, onchange: (e: Event) => { ctx.setAutoplay((e.target as HTMLInputElement).checked); } }),
        'Autoplay'))
    : null;

  return h('div', {
    class: 'fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4',
    role: 'dialog',
    'aria-modal': 'true',
    'aria-label': `${passage.id} ${passage.title}`,
    onclick: (e: Event) => {
      if (e.target === e.currentTarget) {
        ctx.open(null);
      }
    },
  },
    h('article', { class: 'flex max-h-full w-full max-w-3xl flex-col rounded-lg border border-border/60 bg-surface shadow-xl' },
      h('header', { class: 'flex items-start justify-between gap-3 border-b border-border/60 px-6 py-4' },
        h('div', {},
          h('p', { class: 'text-xs uppercase tracking-wide opacity-60' }, `${section.title} · ${passage.id}`,
            noteTag(ctx.noteLabels.get(passage.id)) ? h('span', { class: 'ml-2 rounded border border-amber-500/50 px-1 font-semibold text-amber-400' }, noteTag(ctx.noteLabels.get(passage.id))) : null),
          h('h2', { class: 'text-2xl font-bold text-amber-400' }, passage.title)),
        h('button', { type: 'button', class: btn, 'aria-label': 'Close', onclick: () => { ctx.open(null); } }, 'Close')),
      audioBar,
      h('div', { class: 'min-h-0 flex-1 space-y-5 overflow-y-auto px-6 py-5' },
        passage.notes?.length
          ? h('dl', { class: 'grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 rounded-md border border-border/40 bg-surface-2 p-3 text-sm' },
            ...passage.notes.flatMap((n) => [h('dt', { class: 'font-semibold opacity-70' }, n.label), h('dd', { class: 'opacity-80' }, n.text)]))
          : null,
        ...passage.parts.map((part) => h('section', { class: 'space-y-3' },
          part.speaker
            ? h('p', { class: 'text-sm font-semibold uppercase tracking-wide text-amber-400' }, part.speaker,
              part.aside ? h('span', { class: 'ml-2 font-normal normal-case opacity-70' }, `(${part.aside})`) : null)
            : null,
          ...part.paragraphs.map((text) => h('p', { class: 'whitespace-pre-line font-serif text-xl leading-relaxed' }, text)))),
        passage.goals?.length
          ? h('section', { class: 'rounded-md border border-amber-500/40 bg-amber-500/5 p-3' },
            h('p', { class: 'mb-1 text-sm font-semibold' }, passage.goalsTitle ?? 'Goals'),
            h('ul', { class: 'list-disc space-y-1 pl-5' }, ...passage.goals.map((g) => h('li', {}, g))))
          : null),
      h('footer', { class: 'flex flex-wrap items-center gap-2 border-t border-border/60 px-6 py-3' },
        h('button', { type: 'button', class: btn, disabled: !prev, title: 'Previous passage (←)', onclick: () => { ctx.open(prev); } }, '← Previous'),
        h('button', { type: 'button', class: btn, disabled: !next, title: 'Next passage (→)', onclick: () => { ctx.open(next); } }, 'Next →'),
        h('span', { class: 'flex-1' }),
        done ? h('span', { class: 'text-sm text-positive' }, 'Read ✓') : null,
        editable && done
          ? h('button', { type: 'button', class: btn, onclick: () => { mark(false); } }, 'Mark not read')
          : null,
        editable
          ? h('button', {
            type: 'button',
            class: 'rounded-md bg-amber-600 px-4 py-1 text-sm font-semibold text-white hover:bg-amber-700',
            onclick: () => { mark(true); },
          }, done ? 'Read again' : 'Mark as read')
          : null)));
}
