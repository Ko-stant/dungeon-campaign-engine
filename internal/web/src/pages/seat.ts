/**
 * A player's game screen online: the board as the players see it, the
 * player's own heroes' sheets, buttons for what they may do now, squares to
 * click to move (or a monster to attack), and what happened, dice included.
 * It talks only to the seat API (internal/app/seat.go): the player view and
 * the player's own heroes, never the GM's state.
 */
import { pixelToTile } from '../board/geometry.ts';
import { BoardRenderer } from '../board/renderer.ts';
import { heroChips } from '../players/cards.ts';
import { feedAfter } from '../players/feed.ts';
import type { PlayerCatalog, PlayerEvent, PlayerHero, PlayerState } from '../players/types.ts';
import { playerBoardView } from '../players/view.ts';
import { activeHero, clickAction, groupActions, statusLine, turnDetail } from '../seat/model.ts';
import type { Action, Present, SeatHero, SeatResponse, SeatState, SeatUpdate } from '../seat/types.ts';
import { ApiError } from '../api/http.ts';
import { createTrackerApi } from '../tracker/api.ts';
import { h, replaceChildren } from '../ui/dom.ts';

const FEED_LENGTH = 40;
const btn = 'rounded-md border border-border/60 px-3 py-1.5 text-sm hover:border-amber-500 disabled:opacity-40';
const btnMain = 'rounded-md bg-amber-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-amber-700 disabled:opacity-40';

function bar(value: number, max: number, color: string): HTMLElement {
  const pct = max > 0 ? Math.max(0, Math.min(100, (value / max) * 100)) : 0;
  return h('div', { class: 'h-2 w-full overflow-hidden rounded-full bg-border/40' }, h('div', { class: `h-full rounded-full ${color}`, style: `width: ${pct.toFixed(1)}%` }));
}

async function main(): Promise<void> {
  const root = document.querySelector<HTMLElement>('#seat');
  const sessionId = root?.dataset.sessionId;
  if (!root || !sessionId) {
    throw new Error('seat root element or session id missing');
  }
  const api = createTrackerApi();
  const first = await api.seat(sessionId);
  let state: PlayerState = first.state;
  let seat: SeatState = first.seat;
  let feed: PlayerEvent[] = first.events;
  let presence: Present[] = first.presence;
  let lastSeq = first.eventSeq;
  const catalog: PlayerCatalog = first.catalog;
  let picked: string | null = null;
  let busy = false;
  let message = '';
  let live = false;

  // --- Layout ---
  const canvas = h('canvas', { class: 'block h-full w-full cursor-pointer' });
  const header = h('header', { class: 'flex flex-wrap items-center gap-x-4 gap-y-1 border-b border-border/60 px-4 py-2' });
  // Three columns on a wide screen; on a phone the board comes first, then
  // the turn and the feed, then the sheet and the party.
  const left = h('aside', { class: 'order-3 space-y-3 border-t border-border/60 p-3 lg:order-1 lg:w-80 lg:shrink-0 lg:overflow-y-auto lg:border-t-0 lg:border-r' });
  const right = h('aside', { class: 'order-2 flex flex-col gap-3 border-t border-border/60 p-3 lg:order-3 lg:w-96 lg:shrink-0 lg:overflow-hidden lg:border-t-0 lg:border-l' });
  const board = h('main', { class: 'relative order-1 h-[60vh] min-w-0 p-3 lg:order-2 lg:h-auto lg:flex-1' }, canvas);
  replaceChildren(root, header, h('div', { class: 'flex min-h-0 flex-1 flex-col lg:flex-row' }, left, board, right));

  // --- Board ---
  let frame = 0;
  const requestDraw = (): void => {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => {
      const hero = activeHero(seat, picked);
      const moves = hero ? [...groupActions(hero).moves.keys()].map((k) => { const [x = 0, y = 0] = k.split(',').map(Number); return { x, y }; }) : [];
      renderer.draw(playerBoardView(state, catalog), { tiles: moves, selectedId: hero?.id ?? null });
    });
  };
  const renderer = new BoardRenderer(canvas, requestDraw);
  new ResizeObserver(requestDraw).observe(canvas);
  canvas.addEventListener('click', (ev) => {
    const m = renderer.metrics;
    const hero = activeHero(seat, picked);
    if (!m || !hero || busy) {
      return;
    }
    const rect = canvas.getBoundingClientRect();
    const t = pixelToTile(m, ev.clientX - rect.left, ev.clientY - rect.top);
    const action = t ? clickAction(groupActions(hero), state, t) : null;
    if (action) {
      void send(hero, action);
    }
  });

  // --- Sending ---
  function take(r: SeatResponse): void {
    state = r.state;
    seat = r.seat;
    feed = r.events.slice(-FEED_LENGTH);
    presence = r.presence;
    lastSeq = Math.max(lastSeq, r.eventSeq);
  }

  async function send(hero: SeatHero, action: Action): Promise<void> {
    busy = true;
    message = '';
    refresh();
    try {
      take(await api.seatCommand(sessionId ?? '', hero.id, action.command));
    } catch (e) {
      message = e instanceof ApiError ? e.message : 'Something went wrong; try again.';
    } finally {
      busy = false;
      refresh();
    }
  }

  // --- Panels ---
  function renderHeader(): void {
    replaceChildren(header,
      h('a', { href: '/lobby', class: 'text-sm opacity-70 hover:text-amber-400' }, '← Games'),
      h('h1', { class: 'text-lg font-semibold' }, state.questName),
      h('span', { class: 'font-semibold text-amber-400', 'aria-live': 'polite' }, statusLine(seat, state)),
      state.goal ? h('span', { class: 'text-sm opacity-70', title: 'The quest' }, state.goal) : null,
      presence.length ? h('span', { class: 'ml-auto text-xs opacity-70', title: 'Players connected' }, `At the table: ${presence.map((p) => p.name).join(', ')}`) : h('span', { class: 'ml-auto' }),
      h('span', { class: `h-2 w-2 rounded-full ${live ? 'bg-positive' : 'bg-danger'}`, title: live ? 'Live' : 'Reconnecting…' }));
  }

  function partyCard(p: PlayerHero): HTMLElement {
    const mine = seat.heroes.some((s) => s.id === p.id);
    const turn = seat.turnHero === p.id;
    return h('div', { class: `space-y-1 rounded-md border p-2 ${turn ? 'border-amber-500 bg-amber-500/10' : 'border-border/60'} ${p.status !== 'active' ? 'opacity-50' : ''}` },
      h('div', { class: 'flex items-baseline justify-between text-sm' },
        h('span', { class: 'font-semibold' }, p.name, mine ? h('span', { class: 'ml-1 text-xs text-amber-400' }, '(you)') : null),
        h('span', { class: 'font-mono text-xs' }, `${String(p.body)} / ${String(p.maxBody)}`)),
      bar(p.body, p.maxBody, 'bg-danger'),
      heroChips(p).length ? h('div', { class: 'flex flex-wrap gap-1 text-xs' }, ...heroChips(p).map((c) => h('span', { class: 'rounded-full border border-amber-500/60 px-1.5' }, c))) : null);
  }

  function heroSheet(hero: SeatHero): HTMLElement {
    const c = hero.combat;
    return h('section', { class: 'space-y-2 rounded-lg border border-border/60 bg-surface-2 p-3' },
      h('div', { class: 'flex items-baseline justify-between' },
        h('h2', { class: 'text-lg font-semibold' }, hero.name),
        h('span', { class: 'text-sm opacity-70' }, hero.status === 'active' ? hero.className : hero.status === 'dead' ? 'Fallen' : 'Escaped')),
      h('div', { class: 'flex justify-between text-sm' }, h('span', {}, 'Body'), h('span', { class: 'font-mono' }, `${String(hero.body)} / ${String(hero.maxBody)}`)),
      bar(hero.body, hero.maxBody, 'bg-danger'),
      h('div', { class: 'flex justify-between text-xs opacity-80' },
        h('span', {}, `Mind ${String(hero.mind)} / ${String(hero.maxMind)}`),
        (hero.maxMana ?? 0) > 0 ? h('span', {}, `Mana ${String(hero.mana ?? 0)} / ${String(hero.maxMana ?? 0)}`) : null,
        hero.movement ? h('span', {}, `Moves ${hero.movement}`) : null),
      c ? h('p', { class: 'text-xs opacity-80' }, `Hit ${c.hitDice}${c.accuracy ? ` +${String(c.accuracy)}` : ''} · crit ${String(c.critFrom)}+ · damage ${String(c.damage)} · defense ${c.defenseDice} +${String(c.avoidance)} · mitigation ${String(c.mitigation)}`) : null,
      (hero.determination ?? 0) > 0 ? h('p', { class: 'text-xs text-amber-400' }, `Determination +${String(hero.determination)}`) : null,
      hero.abilities?.length
        ? h('ul', { class: 'space-y-0.5 text-xs' }, ...hero.abilities.map((a) => h('li', { title: a.text ?? '' },
          h('span', { class: 'font-semibold' }, a.name),
          h('span', { class: 'opacity-60' }, ` · ${a.kind}${a.manaCost ? ` · ${String(a.manaCost)} mana` : ''}`),
          (a.readyIn ?? 0) > 0 ? h('span', { class: 'text-danger' }, ` · ready in ${String(a.readyIn)}`) : null)))
        : null,
      hero.items.length
        ? h('p', { class: 'text-xs opacity-70' }, 'Carries: ', hero.items.map((i) => `${i.name}${i.quantity > 1 ? ` ×${String(i.quantity)}` : ''}${i.equipped ? ' (equipped)' : ''}`).join(', '))
        : null);
  }

  function renderLeft(): void {
    const hero = activeHero(seat, picked);
    replaceChildren(left,
      seat.heroes.length > 1
        ? h('div', { class: 'flex flex-wrap gap-1' }, ...seat.heroes.map((s) => h('button', {
          type: 'button',
          class: s.id === hero?.id ? `${btn} border-amber-500 bg-amber-500/15` : btn,
          onclick: () => { picked = s.id; refresh(); },
        }, s.name)))
        : null,
      hero ? heroSheet(hero) : h('p', { class: 'text-sm opacity-70' }, 'You have no hero in this game yet. Join from the games list.'),
      h('h2', { class: 'pt-2 text-xs font-semibold uppercase tracking-wide opacity-60' }, 'Party'),
      ...state.heroes.map(partyCard));
  }

  /** The hero's turn: what is left of it, and a button for each action. */
  function actionsFor(hero: SeatHero): (HTMLElement | null)[] {
    const groups = groupActions(hero);
    const detail = turnDetail(hero);
    return [
      h('h2', { class: 'text-sm font-semibold uppercase tracking-wide opacity-60' }, `${hero.name}'s turn`),
      detail ? h('p', { class: 'text-sm' }, detail) : null,
      groups.moves.size ? h('p', { class: 'text-xs opacity-70' }, 'Click a highlighted square to move there, or a monster in reach to attack it.') : null,
      groups.buttons.length
        ? h('div', { class: 'flex flex-wrap gap-2' }, ...groups.buttons.map((a) => h('button', {
          type: 'button',
          class: a.command.type === 'turn.start' || a.command.type === 'turn.roll-move' ? btnMain : btn,
          disabled: busy,
          onclick: () => { void send(hero, a); },
        }, a.label)))
        : h('p', { class: 'text-sm opacity-60' }, 'Nothing to do right now.'),
    ];
  }

  function renderRight(): void {
    const hero = activeHero(seat, picked);
    const lines = [...feed].reverse();
    replaceChildren(right,
      h('section', { class: 'space-y-2' },
        ...(hero ? actionsFor(hero) : [h('p', { class: 'text-sm opacity-60' }, 'No hero of yours here yet.')]),
        message ? h('p', { class: 'rounded-md border border-danger/60 bg-danger/10 px-2 py-1 text-sm text-danger', role: 'alert' }, message) : null),
      h('section', { class: 'flex min-h-0 flex-1 flex-col' },
        h('h2', { class: 'mb-1 text-sm font-semibold uppercase tracking-wide opacity-60' }, 'What happened'),
        h('ul', { class: 'max-h-80 min-h-0 flex-1 space-y-1 overflow-y-auto text-sm lg:max-h-none' },
          ...lines.map((e) => h('li', { class: 'flex gap-2' }, h('span', { class: 'w-8 shrink-0 font-mono text-xs opacity-60' }, `R${String(e.round)}`), h('span', {}, e.summary))))));
  }

  function refresh(): void {
    renderHeader();
    renderLeft();
    renderRight();
    requestDraw();
  }

  // --- Live updates ---
  function apply(u: SeatUpdate): void {
    if (u.presence) {
      presence = u.presence;
    }
    if (u.player && u.player.eventSeq > lastSeq) {
      state = u.player.state;
      lastSeq = u.player.eventSeq;
      feed = feedAfter(feed, u.player, FEED_LENGTH);
      if (u.seat) {
        seat = u.seat;
      }
    }
  }

  function connect(delay = 1000): void {
    const ws = new WebSocket(api.seatStreamUrl(sessionId ?? ''));
    ws.addEventListener('open', () => {
      live = true;
      delay = 1000;
      // Catch up on anything missed while disconnected.
      void api.seat(sessionId ?? '').then((r) => { take(r); refresh(); }).catch(() => undefined);
      refresh();
    });
    ws.addEventListener('message', (ev) => {
      if (typeof ev.data === 'string') {
        apply(JSON.parse(ev.data) as SeatUpdate);
        refresh();
      }
    });
    ws.addEventListener('close', () => {
      live = false;
      refresh();
      setTimeout(() => { connect(Math.min(delay * 2, 15000)); }, delay);
    });
  }

  refresh();
  connect();
}

void main();
