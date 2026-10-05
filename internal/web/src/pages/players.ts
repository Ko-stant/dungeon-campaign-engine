/**
 * The player screen: a read-only, live view of the quest for the game-room
 * TV, opened in its own tab. It only ever receives the player view from the
 * server (internal/app/player_api.go), so nothing secret reaches this page.
 * The GM can click a piece on the board (or a hero in the party panel) to
 * show its card, open the event feed, and go full screen.
 */
import { pixelToTile } from '../board/geometry.ts';
import { BoardRenderer } from '../board/renderer.ts';
import { cardFor, pickAt, type Pick } from '../players/cards.ts';
import { addEvent } from '../players/feed.ts';
import type { PlayerCatalog, PlayerEvent, PlayerHero, PlayerState, PlayerUpdate } from '../players/types.ts';
import { playerBoardView } from '../players/view.ts';
import { createTrackerApi } from '../tracker/api.ts';
import { effectLabel } from '../tracker/effects.ts';
import { h, replaceChildren } from '../ui/dom.ts';

const FEED_LENGTH = 30;
const btn = 'rounded-md border border-border/60 px-3 py-1 text-base hover:border-amber-500';
const btnActive = 'rounded-md border border-amber-500 bg-amber-500/15 px-3 py-1 text-base';

function bar(value: number, max: number, color: string): HTMLElement {
  const pct = max > 0 ? Math.max(0, Math.min(100, (value / max) * 100)) : 0;
  return h('div', { class: 'h-2.5 w-full overflow-hidden rounded-full bg-border/40' }, h('div', { class: `h-full rounded-full ${color}`, style: `width: ${pct.toFixed(1)}%` }));
}

async function main(): Promise<void> {
  const root = document.querySelector<HTMLElement>('#players');
  const sessionId = root?.dataset.sessionId;
  if (!root || !sessionId) {
    throw new Error('player screen root element or session id missing');
  }
  const api = createTrackerApi();
  const first = await api.player(sessionId);
  let state: PlayerState = first.state;
  const catalog: PlayerCatalog = first.catalog;
  let feed: PlayerEvent[] = first.events;
  let lastSeq = first.eventSeq;
  let pick: Pick | null = null;
  let feedOpen = false;
  let unread = 0;
  let live = false;

  // --- Layout ---
  const canvas = h('canvas', { class: 'block h-full w-full cursor-pointer' });
  const header = h('header', { class: 'flex items-center gap-4 border-b border-border/60 px-4 py-2' });
  const party = h('aside', { class: 'w-96 shrink-0 space-y-3 overflow-y-auto border-r border-border/60 p-4' });
  // The card and the event feed share a right-hand column, shown only while one is open, so they never cover the map.
  const cardHost = h('div');
  const feedHost = h('div', { class: 'flex min-h-0 flex-1 flex-col' });
  const side = h('aside', { class: 'hidden w-[28rem] shrink-0 flex-col gap-3 overflow-hidden border-l border-border/60 p-4' }, cardHost, feedHost);
  replaceChildren(root,
    header,
    h('div', { class: 'flex min-h-0 flex-1' },
      party,
      h('main', { class: 'relative min-w-0 flex-1 p-3' }, canvas),
      side));

  // --- Drawing ---
  let frame = 0;
  const requestDraw = (): void => {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => {
      const selectedId = pick?.id ?? null;
      renderer.draw(playerBoardView(state, catalog), { selectedId });
    });
  };
  const renderer = new BoardRenderer(canvas, requestDraw);
  new ResizeObserver(requestDraw).observe(canvas);

  canvas.addEventListener('click', (ev) => {
    const m = renderer.metrics;
    if (!m) {
      return;
    }
    const rect = canvas.getBoundingClientRect();
    const t = pixelToTile(m, ev.clientX - rect.left, ev.clientY - rect.top);
    const next = t ? pickAt(state, catalog, t) : null;
    pick = next && (next.kind !== pick?.kind || next.id !== pick.id) ? next : null;
    refresh();
  });

  // --- Panels ---
  function renderHeader(): void {
    replaceChildren(header,
      h('h1', { class: 'text-2xl font-semibold' }, state.questName),
      h('span', { class: 'text-2xl font-semibold text-amber-400' }, `Round ${String(state.round)}`),
      state.fight ? h('span', { class: 'rounded-full border border-danger/60 bg-danger/15 px-3 py-0.5 text-lg font-semibold text-danger' }, 'Fight') : null,
      h('span', { class: 'flex-1' }),
      h('button', {
        type: 'button',
        class: feedOpen ? btnActive : btn,
        onclick: () => {
          feedOpen = !feedOpen;
          unread = 0;
          refresh();
        },
      }, unread > 0 && !feedOpen ? `Events (${String(unread)} new)` : 'Events'),
      h('button', {
        type: 'button',
        class: btn,
        onclick: () => {
          if (document.fullscreenElement) {
            void document.exitFullscreen();
          } else {
            void document.documentElement.requestFullscreen();
          }
        },
      }, document.fullscreenElement ? 'Exit full screen' : 'Full screen'),
      h('span', { class: `h-3 w-3 rounded-full ${live ? 'bg-positive' : 'bg-danger'}`, title: live ? 'Live' : 'Reconnecting…' }));
  }

  function heroCard(hero: PlayerHero): HTMLElement {
    const cls = catalog.heroes.find((c) => c.id === hero.class);
    const selected = pick?.kind === 'hero' && pick.id === hero.id;
    const down = hero.status !== 'active';
    return h('button', {
      type: 'button',
      class: `block w-full space-y-1.5 rounded-lg border p-3 text-left ${selected ? 'border-amber-500 bg-amber-500/10' : 'border-border/60'} ${down ? 'opacity-50' : ''}`,
      onclick: () => {
        pick = selected ? null : { kind: 'hero', id: hero.id };
        refresh();
      },
    },
      h('div', { class: 'flex items-baseline justify-between gap-2' },
        h('span', { class: 'text-xl font-semibold' }, hero.name),
        h('span', { class: 'text-sm opacity-70', style: cls?.color ? `color: ${cls.color}` : null }, hero.status === 'dead' ? 'Fallen' : hero.status === 'escaped' ? 'Escaped' : cls?.name ?? '')),
      h('div', { class: 'flex justify-between text-base' }, h('span', {}, 'Body'), h('span', { class: 'font-mono' }, `${String(hero.body)} / ${String(hero.maxBody)}`)),
      bar(hero.body, hero.maxBody, 'bg-danger'),
      h('div', { class: 'flex justify-between text-sm opacity-80' },
        h('span', {}, `Mind ${String(hero.mind)} / ${String(hero.maxMind)}`),
        (hero.manaCap ?? 0) > 0 ? h('span', {}, `Mana ${String(hero.mana ?? 0)} / ${String(hero.manaCap ?? 0)}`) : null),
      (hero.manaCap ?? 0) > 0 ? bar(hero.mana ?? 0, hero.manaCap ?? 0, 'bg-brand') : null,
      (hero.effects ?? []).length
        ? h('div', { class: 'flex flex-wrap gap-1 pt-1' }, ...(hero.effects ?? []).map((e) => h('span', { class: 'rounded-full border border-amber-500/60 bg-amber-500/10 px-2 py-0.5 text-sm' }, effectLabel(e))))
        : null);
  }

  function renderParty(): void {
    replaceChildren(party, h('h2', { class: 'text-lg font-semibold uppercase tracking-wide opacity-70' }, 'Party'), ...state.heroes.map(heroCard));
  }

  function renderCard(): void {
    const card = pick ? cardFor(state, catalog, pick) : null;
    if (!card) {
      pick = null;
      replaceChildren(cardHost);
      return;
    }
    replaceChildren(cardHost, h('section', { class: 'space-y-2 rounded-xl border border-amber-500/60 bg-surface p-4' },
      h('div', { class: 'flex items-start justify-between gap-3' },
        h('div', {},
          h('h2', { class: 'text-2xl font-semibold' }, card.title),
          card.subtitle ? h('p', { class: 'text-base opacity-70' }, card.subtitle) : null),
        h('button', { type: 'button', class: 'text-2xl leading-none opacity-60 hover:opacity-100', 'aria-label': 'Close', onclick: () => { pick = null; refresh(); } }, '×')),
      card.tags.length ? h('div', { class: 'flex flex-wrap gap-2' }, ...card.tags.map((t) => h('span', { class: 'rounded-full border border-danger/60 bg-danger/15 px-3 py-0.5 text-base font-semibold text-danger' }, t))) : null,
      ...card.lines.map((l) => h('p', { class: 'text-lg' }, l)),
      card.effects.length ? h('div', { class: 'flex flex-wrap gap-2 pt-1' }, ...card.effects.map((e) => h('span', { class: 'rounded-full border border-amber-500/60 bg-amber-500/10 px-3 py-0.5 text-base' }, e))) : null));
  }

  function renderFeed(): void {
    if (!feedOpen) {
      replaceChildren(feedHost);
      return;
    }
    const lines = [...feed].reverse();
    replaceChildren(feedHost, h('section', { class: 'min-h-0 flex-1 space-y-1 overflow-y-auto rounded-xl border border-border/60 bg-surface p-4' },
      h('h2', { class: 'text-lg font-semibold uppercase tracking-wide opacity-70' }, 'What happened'),
      lines.length
        ? h('ul', { class: 'space-y-1' }, ...lines.map((e) => h('li', { class: 'flex gap-3 text-lg' }, h('span', { class: 'w-10 shrink-0 font-mono text-base opacity-60' }, `R${String(e.round)}`), h('span', {}, e.summary))))
        : h('p', { class: 'text-lg opacity-60' }, 'Nothing yet.')));
  }

  function refresh(): void {
    renderHeader();
    renderParty();
    renderCard();
    renderFeed();
    const open = pick !== null || feedOpen;
    side.classList.toggle('hidden', !open);
    side.classList.toggle('flex', open);
    requestDraw();
  }

  // --- Live updates ---
  function apply(u: PlayerUpdate): void {
    if (u.eventSeq <= lastSeq) {
      return;
    }
    state = u.state;
    lastSeq = u.eventSeq;
    if (u.event) {
      feed = addEvent(feed, u.event, FEED_LENGTH);
      if (!feedOpen) {
        unread++;
      }
    }
  }

  function connect(delay = 1000): void {
    const ws = new WebSocket(api.playerStreamUrl(sessionId ?? ''));
    ws.addEventListener('open', () => {
      live = true;
      // Catch up on anything missed while disconnected.
      void api.player(sessionId ?? '').then((r) => {
        state = r.state;
        lastSeq = Math.max(lastSeq, r.eventSeq);
        for (const e of r.events) {
          feed = addEvent(feed, e, FEED_LENGTH);
        }
        refresh();
      }).catch(() => undefined);
      refresh();
    });
    ws.addEventListener('message', (ev) => {
      if (typeof ev.data === 'string') {
        apply(JSON.parse(ev.data) as PlayerUpdate);
        refresh();
      }
    });
    ws.addEventListener('close', () => {
      live = false;
      refresh();
      setTimeout(() => { connect(Math.min(delay * 2, 15000)); }, delay);
    });
  }

  document.addEventListener('fullscreenchange', renderHeader);
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && pick) {
      pick = null;
      refresh();
    }
  });
  refresh();
  connect();
}

void main();
