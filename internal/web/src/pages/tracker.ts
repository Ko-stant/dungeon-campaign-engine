/**
 * Tracker page (/play/{sessionId}): the GM's live record of a game at the
 * table. Every change is a command sent to the server, which saves the new
 * state and a readable event, then pushes both to every open tab.
 */
import { pixelToEdge, pixelToTile, type TileCoord } from '../board/geometry.ts';
import { tileIndex } from '../board/model.ts';
import { monsterOptionLabel, type TrapDoc } from '../maps/types.ts';
import { BoardRenderer } from '../board/renderer.ts';
import { ApiError } from '../api/http.ts';
import { createTrackerApi } from '../tracker/api.ts';
import { combatLine, combatTotals, FALTER_PENALTY, falterAt, manaCap, monsterLine } from '../tracker/combat.ts';
import { formatEvent } from '../tracker/format.ts';
import { travelOptions } from '../tracker/travel.ts';
import { clickCommand, corridorPath, hotkey, paintPending, revealPathCommand, revealSquaresCommand, type ClickTarget, type Mode } from '../tracker/interaction.ts';
import { triggerButton } from '../tracker/traps.ts';
import { shownToPlayers } from '../tracker/visibility.ts';
import type { Command, CommandResponse, Hero, LiveTrapState, Monster, ScriptSection, SessionEvent, SessionState } from '../tracker/types.ts';
import { trackerView } from '../tracker/view.ts';
import { lineTiles } from '../editor/tools.ts';
import { h, preserveFocus, replaceChildren } from '../ui/dom.ts';
import { abilitySection, inventorySection, purseControl, type SectionContext } from '../ui/heroSections.ts';
import { oddsBlock } from '../ui/odds.ts';
import { effectSuggestions, effectsBlock } from '../ui/effects.ts';
import { readAloudPanel, readerOverlay, type ReadAloudContext } from '../ui/readAloud.ts';
import { currentSection, passageClips } from '../tracker/script.ts';

const TRAP_STATES: readonly LiveTrapState[] = ['hidden', 'revealed', 'triggered', 'disarmed', 'removed'];
const btn = 'rounded-md border border-border/60 px-2 py-1 text-sm hover:border-amber-500 disabled:opacity-40';
const btnActive = 'rounded-md border border-amber-500 bg-amber-500/15 px-2 py-1 text-sm';
const smallBtn = 'h-6 w-6 rounded border border-border/60 text-sm leading-none hover:border-amber-500 disabled:opacity-40';
const field = 'rounded-md border border-border/60 bg-surface px-2 py-1 text-sm';
const KIND_STYLES: Record<string, string> = {
  move: 'border-sky-500/50',
  door: 'border-brand/60',
  trap: 'border-danger/60',
  hero: 'border-positive/60',
  monster: 'border-accent/60',
  reveal: 'border-warning/50',
  round: 'border-amber-500',
  note: 'border-content/50',
  session: 'border-purple-400/60',
  other: 'border-border/60',
};

async function main(): Promise<void> {
  const root = document.querySelector<HTMLElement>('#tracker');
  const sessionId = root?.dataset.sessionId;
  if (!root || !sessionId) {
    throw new Error('tracker root element or session id missing');
  }
  const api = createTrackerApi();
  const [session, catalog, initialEvents] = await Promise.all([api.session(sessionId), api.catalog(), api.events(sessionId)]);
  // The campaign's chapters, for traveling to another map mid-game.
  const chapters = await api.chapters(session.campaignId).catch(() => []);
  // The campaign's read-aloud script (empty when it has none).
  const loadScript = (): Promise<ScriptSection[]> => api.script(session.campaignId).then((r) => r.sections).catch(() => []);
  let scriptSections = await loadScript();
  const loadClips = (): Promise<Record<string, string>> => api.audio(session.campaignId).then((r) => r.clips).catch(() => ({}));
  let clips = await loadClips();

  let state: SessionState = session.state;
  let status = session.status;
  let lastSeq = session.eventSeq;
  const events: SessionEvent[] = initialEvents;
  let selectedId: string | null = null;
  let mode: Mode = { kind: 'select' };
  let monsterType = catalog.monsters[0]?.id ?? '';
  let fog = true;
  // Reveal options: also mark monsters on revealed squares as seen; squares picked in 'pickSquares'.
  let revealSeen = true;
  let pending: TileCoord[] = [];
  let painting: { add: boolean; last: TileCoord } | null = null;
  // A drag in Reveal mode: the squares passed over (revealPathCommand keeps the new corridor ones).
  let revealDrag: { path: TileCoord[]; last: TileCoord; moved: boolean } | null = null;
  // The click that ends a reveal drag must not also reveal the square it ends on.
  let swallowClick = false;
  let busy = false;
  let message = '';
  let live = false;
  // The passage open in the reader, and script sections the GM opened or closed.
  let readerId: string | null = null;
  const scriptSectionOpen = new Map<number, boolean>();
  // The reader's audio player: one element, moved into each re-rendered reader, so playback survives live updates.
  const player = h('audio', { controls: true, preload: 'none', class: 'h-9 w-full' });
  const AUTOPLAY_KEY = 'dce.readerAutoplay';
  let autoplay = false;
  try {
    autoplay = localStorage.getItem(AUTOPLAY_KEY) === '1';
  } catch {
    // storage unavailable: autoplay stays off
  }
  // Hero card sections the GM has collapsed ("hero-1:inventory").
  const closedSections = new Set<string>();

  // --- Layout ---
  const canvas = h('canvas', { class: 'block h-full w-full' });
  const header = h('header', { class: 'flex items-center gap-3 overflow-x-auto whitespace-nowrap border-b border-border/60 px-3 py-2' });
  const modeBar = h('div', { class: 'flex flex-wrap items-center gap-2 px-2 pb-2' });
  const leftPanel = h('aside', { class: 'w-80 shrink-0 space-y-3 overflow-y-auto border-r border-border/60 p-3' });
  const rightPanel = h('aside', { class: 'w-80 shrink-0 space-y-4 overflow-y-auto border-l border-border/60 p-3' });
  const readerHost = h('div');
  const hoverInfo = h('span', { class: 'pointer-events-none absolute bottom-3 left-3 rounded bg-surface/80 px-2 py-0.5 font-mono text-xs opacity-80 empty:hidden' });
  replaceChildren(
    root,
    header,
    h('div', { class: 'flex min-h-0 flex-1' },
      leftPanel,
      h('main', { class: 'relative flex min-w-0 flex-1 flex-col p-2' }, modeBar, h('div', { class: 'min-h-0 flex-1' }, canvas), hoverInfo),
      rightPanel),
    readerHost,
  );

  // --- Drawing ---
  let frame = 0;
  const requestDraw = (): void => {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => {
      const tiles = mode.kind === 'pickSquares' ? pending : revealDrag?.moved ? corridorPath(state, revealDrag.path) : null;
      renderer.draw(trackerView(state, catalog, { fog }), { selectedId, tiles });
    });
  };
  const renderer = new BoardRenderer(canvas, requestDraw);

  // --- Commands ---
  function applyResponse(res: CommandResponse): void {
    if (res.eventSeq <= lastSeq) {
      return; // already applied (our own change echoed back by the stream)
    }
    state = res.state;
    lastSeq = res.eventSeq;
    events.push(res.event);
    if (res.event.kind === 'session.complete') {
      status = 'completed';
    } else if (res.event.kind === 'session.reopen') {
      status = 'active';
    } else if (res.event.kind === 'map.travel') {
      selectedId = null; // pieces on the old map are gone from view
    }
  }

  async function send(command: Command): Promise<boolean> {
    if (status !== 'active') {
      message = 'This session is completed. Reopen it to make changes.';
      refresh();
      return false;
    }
    busy = true;
    try {
      applyResponse(await api.command(sessionId ?? '', command));
      message = '';
      return true;
    } catch (err) {
      message = err instanceof ApiError ? err.message : 'Could not save that change.';
      return false;
    } finally {
      busy = false;
      refresh();
    }
  }

  function heroCmd(hero: Hero, changes: Record<string, unknown>): void {
    void send({ type: 'hero.update', payload: { id: hero.id, ...changes } });
  }

  function monsterCmd(m: Monster, changes: Record<string, unknown>): void {
    void send({ type: 'monster.update', payload: { id: m.id, ...changes } });
  }

  // --- Board input ---
  function targetAt(ev: MouseEvent): ClickTarget {
    const m = renderer.metrics;
    if (!m) {
      return { tile: null, edge: null };
    }
    const rect = canvas.getBoundingClientRect();
    const px = ev.clientX - rect.left;
    const py = ev.clientY - rect.top;
    // Doors are easier to hit at the table than in the editor.
    return { tile: pixelToTile(m, px, py), edge: pixelToEdge(m, px, py, 0.35) };
  }

  canvas.addEventListener('click', (ev) => {
    if (swallowClick) {
      swallowClick = false;
      return;
    }
    const result = clickCommand(state, mode, selectedId, targetAt(ev), catalog);
    if (!result) {
      return;
    }
    selectedId = result.select;
    if (result.command) {
      void send(result.command);
    } else {
      refresh();
    }
  });

  // Picking squares to reveal: press on a square to add it (or remove it if picked), drag to paint more.
  canvas.addEventListener('mousedown', (ev) => {
    const t = targetAt(ev).tile;
    swallowClick = false;
    if (mode.kind === 'reveal' && t) {
      revealDrag = { path: [t], last: t, moved: false };
      return;
    }
    if (mode.kind !== 'pickSquares' || !t) {
      return;
    }
    painting = { add: !pending.some((p) => p.x === t.x && p.y === t.y), last: t };
    pending = paintPending(pending, t, painting.add);
    refresh();
  });
  /** Paints every square from the last painted one to t, so a quick drag does not skip any. */
  function paintTo(t: TileCoord | null): void {
    if (!painting || !t || mode.kind !== 'pickSquares' || (t.x === painting.last.x && t.y === painting.last.y)) {
      return;
    }
    const { add } = painting;
    pending = lineTiles(painting.last, t).reduce((acc, tile) => paintPending(acc, tile, add), pending);
    painting.last = t;
    refresh();
  }
  /** Extends a reveal drag to t, through every square in between. */
  function dragTo(t: TileCoord | null): void {
    if (!revealDrag || !t || (t.x === revealDrag.last.x && t.y === revealDrag.last.y)) {
      return;
    }
    revealDrag.path.push(...lineTiles(revealDrag.last, t));
    revealDrag.last = t;
    revealDrag.moved = true;
    requestDraw();
  }
  window.addEventListener('mouseup', (ev) => {
    if (ev.target === canvas) {
      paintTo(targetAt(ev).tile);
      dragTo(targetAt(ev).tile);
    }
    painting = null;
    if (revealDrag?.moved) {
      // A drag reveals the corridor it crossed; a press without a drag is a click (a room or one square).
      swallowClick = ev.target === canvas;
      const command = revealPathCommand(state, revealDrag.path, revealSeen);
      revealDrag = null;
      requestDraw();
      if (command) {
        void send(command);
      }
    }
    revealDrag = null;
  });

  canvas.addEventListener('mousemove', (ev) => {
    const t = targetAt(ev).tile;
    paintTo(t);
    dragTo(t);
    if (!t) {
      hoverInfo.textContent = '';
      return;
    }
    const region = state.board.regions[tileIndex(state.board.width, t)] ?? -1;
    const room = state.board.rooms.find((r) => r.id === region)?.name;
    hoverInfo.textContent = `(${t.x}, ${t.y}) ${room ?? (region === 0 ? 'corridor' : region === -1 ? 'solid rock' : '')}`;
  });
  canvas.addEventListener('mouseleave', () => {
    hoverInfo.textContent = '';
  });

  window.addEventListener('keydown', (ev) => {
    const typing = ev.target instanceof HTMLInputElement || ev.target instanceof HTMLTextAreaElement || ev.target instanceof HTMLSelectElement;
    if (typing) {
      return;
    }
    if (readerId) {
      const ids = scriptSections.flatMap((sec) => sec.passages.map((p) => p.id));
      const i = ids.indexOf(readerId);
      if (ev.key === 'Escape') {
        openPassage(null);
      } else if (ev.key === 'ArrowLeft' && i > 0) {
        openPassage(ids[i - 1] ?? null);
      } else if (ev.key === 'ArrowRight' && i >= 0 && i < ids.length - 1) {
        openPassage(ids[i + 1] ?? null);
      } else if (ev.key === ' ' && player.src) {
        if (player.paused) {
          void player.play().catch(() => undefined);
        } else {
          player.pause();
        }
      } else {
        return;
      }
      ev.preventDefault();
      return;
    }
    if (ev.key === 'Escape') {
      // Cancels a reveal drag in progress: letting go then reveals nothing.
      swallowClick = revealDrag !== null;
      revealDrag = null;
      selectedId = null;
      mode = { kind: 'select' };
      pending = [];
      refresh();
      return;
    }
    const key = hotkey(ev, mode, revealSeen);
    if (key) {
      ev.preventDefault();
      if (key.deselect) {
        selectedId = null;
      }
      setMode(key.mode);
    }
  });

  // --- Live updates ---
  function connect(delay = 1000): void {
    const ws = new WebSocket(api.streamUrl(sessionId ?? ''));
    ws.addEventListener('open', () => {
      live = true;
      delay = 1000;
      // Catch up on anything missed while disconnected.
      void api.session(sessionId ?? '').then((s) => {
        if (s.eventSeq > lastSeq) {
          state = s.state;
          status = s.status;
          void api.events(sessionId ?? '', lastSeq).then((missed) => {
            events.push(...missed);
            lastSeq = s.eventSeq;
            refresh();
          });
        }
      });
      refresh();
    });
    ws.addEventListener('message', (ev) => {
      if (typeof ev.data === 'string') {
        applyResponse(JSON.parse(ev.data) as CommandResponse);
        refresh();
      }
    });
    ws.addEventListener('close', () => {
      live = false;
      refresh();
      setTimeout(() => { connect(Math.min(delay * 2, 15000)); }, delay);
    });
  }

  // --- Rendering panels ---
  function travelControl(completed: boolean): HTMLElement | null {
    const options = travelOptions(state, chapters);
    if (completed || options.length === 0) {
      return null;
    }
    const picker = h('select', { class: field, 'aria-label': 'Map to travel to' },
      ...options.map((o) => h('option', { value: o.questId }, o.label)));
    return h('span', { class: 'flex items-center gap-1' },
      picker,
      h('button', {
        type: 'button',
        class: btn,
        disabled: busy,
        title: 'Move the whole party to another map. Heroes keep their body, mind and items, and the party its gold; the map you leave is kept as it is.',
        onclick: () => {
          const choice = options.find((o) => o.questId === picker.value);
          if (!choice || !confirm(`Travel the party to ${choice.label}?`)) {
            return;
          }
          busy = true;
          refresh();
          void api.travel(sessionId ?? '', choice.questId).then((r) => {
            applyResponse(r);
            message = '';
          }).catch((err: unknown) => {
            message = err instanceof ApiError ? err.message : 'Could not travel.';
          }).finally(() => {
            busy = false;
            refresh();
          });
        },
      }, 'Travel'));
  }

  function renderHeader(): void {
    const completed = status !== 'active';
    replaceChildren(
      header,
      h('a', { href: `/campaigns/${session.campaignId}`, class: 'text-sm opacity-70 hover:text-amber-400' }, '← Campaign'),
      h('div', { class: 'flex flex-col leading-tight' },
        h('span', { class: 'font-semibold' }, session.name),
        h('span', { class: 'text-xs opacity-60' }, state.questName)),
      travelControl(completed),
      h('span', { class: 'mx-2 h-6 w-px bg-border/60' }),
      h('span', { class: 'text-lg font-bold text-amber-400', 'aria-live': 'polite' }, `Round ${state.round}`),
      state.fight
        ? h('span', { class: 'rounded-full border border-danger/60 bg-danger/10 px-2 py-0.5 text-xs font-semibold text-danger', title: 'Rounds finish cooldowns, regenerate mana and count effects down' }, 'Fight')
        : null,
      h('button', { type: 'button', class: btn, disabled: completed || busy, onclick: () => { void send({ type: 'round.advance', payload: {} }); } }, 'Next round'),
      h('button', {
        type: 'button',
        class: btn,
        disabled: completed || busy,
        title: state.fight ? 'End the fight: effects with a countdown end; cooldowns drop to 1-2 rounds left (a 1-round one is ready) and wait for the next fight' : 'Start a fight: rounds finish cooldowns, regenerate mana and count effects down',
        onclick: () => { void send({ type: state.fight ? 'fight.end' : 'fight.start', payload: {} }); },
      }, state.fight ? 'End fight' : 'Start fight'),
      h('label', { class: 'flex items-center gap-1 text-sm', title: 'Darken undiscovered squares and fade the furniture, doors, blocked squares and monsters the players have not been shown' },
        h('input', { type: 'checkbox', checked: fog, onchange: (e: Event) => { fog = (e.target as HTMLInputElement).checked; requestDraw(); } }),
        'Show what the heroes have seen'),
      h('a', {
        class: btn,
        href: `/play/${encodeURIComponent(sessionId ?? '')}/players`,
        target: 'player-screen',
        title: 'Open the player screen in a new tab (drag it to the TV and go full screen)',
      }, 'Open player screen'),
      h('label', { class: 'flex items-center gap-1 text-sm', title: "Show monsters' Body on the player screen" },
        h('input', {
          type: 'checkbox',
          checked: !(state.players?.hideMonsterBody ?? false),
          disabled: completed || busy,
          onchange: (e: Event) => { void send({ type: 'players.set', payload: { hideMonsterBody: !(e.target as HTMLInputElement).checked } }); },
        }),
        'Players see monster Body'),
      h('span', { class: `ml-auto text-xs ${message ? 'text-danger' : 'opacity-60'}`, role: 'status' }, message || (busy ? 'Saving…' : `Saved · ${lastSeq} events`)),
      h('span', { class: `h-2 w-2 rounded-full ${live ? 'bg-positive' : 'bg-danger'}`, title: live ? 'Live' : 'Reconnecting…' }),
      completed
        ? h('button', { type: 'button', class: btn, onclick: () => { void api.reopen(sessionId ?? '').then((r) => { applyResponse(r); refresh(); }); } }, 'Reopen')
        : h('button', {
          type: 'button',
          class: btn,
          onclick: () => {
            if (confirm('Complete this quest? Gold, equipment and notes will be saved to the campaign.')) {
              void api.complete(sessionId ?? '').then((r) => { applyResponse(r); refresh(); }).catch((err: unknown) => {
                message = err instanceof ApiError ? err.message : 'Could not complete the quest.';
                refresh();
              });
            }
          },
        }, 'Complete quest'),
    );
  }

  function setMode(m: Mode): void {
    if (m.kind !== 'pickSquares') {
      pending = [];
    }
    mode = m;
    refresh();
  }

  async function revealPicked(): Promise<void> {
    const command = revealSquaresCommand(pending, revealSeen);
    if (command && (await send(command))) {
      pending = [];
      refresh();
    }
  }

  function renderModeBar(): void {
    const revealing = mode.kind === 'reveal' || mode.kind === 'pickSquares';
    const modeBtn = (m: Mode, label: string, title: string, active = mode.kind === m.kind, key = ''): HTMLButtonElement =>
      h('button', {
        type: 'button',
        title: key ? `${title} (key ${key})` : title,
        class: active ? btnActive : btn,
        onclick: () => { setMode(m); },
      }, label, key ? h('kbd', { class: 'ml-1.5 rounded border border-current/30 px-1 font-mono text-[0.7rem] opacity-60' }, key) : null);
    const checkbox = (label: string, checked: boolean, onChange: (v: boolean) => void): HTMLElement =>
      h('label', { class: 'flex items-center gap-1 text-sm' },
        h('input', { type: 'checkbox', checked, onchange: (e: Event) => { onChange((e.target as HTMLInputElement).checked); } }), label);
    const revealOptions: (Node | null)[] = revealing
      ? [
          checkbox('Show contents too', revealSeen, (v) => {
            revealSeen = v;
            setMode(mode.kind === 'pickSquares' ? { kind: 'pickSquares', seen: v } : { kind: 'reveal', seen: v });
          }),
          checkbox('Pick squares', mode.kind === 'pickSquares', (v) => { setMode(v ? { kind: 'pickSquares', seen: revealSeen } : { kind: 'reveal', seen: revealSeen }); }),
          mode.kind === 'pickSquares'
            ? h('button', { type: 'button', class: btnActive, disabled: pending.length === 0 || busy, onclick: () => { void revealPicked(); } },
              pending.length === 0 ? 'Reveal squares' : pending.length === 1 ? 'Reveal 1 square' : `Reveal ${pending.length} squares`)
            : null,
          mode.kind === 'pickSquares' && pending.length > 0
            ? h('button', { type: 'button', class: btn, onclick: () => { pending = []; refresh(); } }, 'Clear')
            : null,
        ]
      : [];
    const monsterSelect = h('select', {
      class: field,
      'aria-label': 'Monster to add',
      onchange: (e: Event) => {
        monsterType = (e.target as HTMLSelectElement).value;
        setMode({ kind: 'addMonster', monsterType });
      },
    }, ...catalog.monsters.map((m) => h('option', { value: m.id, selected: m.id === monsterType }, monsterOptionLabel(m))));
    const hint: Record<Mode['kind'], string> = {
      select: 'Click a hero, monster or movable trap (boulder), then a square to move it; press 1 to let go of it. Click a door to select it, then open, close or lock it from the Selected panel. Click furniture or blocked squares to show them to the players.',
      reveal: revealSeen
        ? 'Click a room to reveal it, or drag along a corridor to reveal its squares, and show the players what is there: monsters, furniture, blocked squares and doors (never traps or unfound secret doors).'
        : 'Click a room to reveal it, or drag along a corridor to reveal its squares. What is there stays hidden from the players.',
      pickSquares: 'Click or drag across squares to pick them, then reveal them all at once.',
      hide: 'Click a square to hide it again.',
      block: 'Click a square to block it (a falling block, where a boulder stopped). Select an added block to clear it.',
      addMonster: 'Click a square to place the monster.',
    };
    replaceChildren(
      modeBar,
      modeBtn({ kind: 'select' }, 'Select / move', 'Select pieces, move them, open and close doors; the key also clears the selection', mode.kind === 'select', '1'),
      modeBtn({ kind: 'reveal', seen: revealSeen }, 'Reveal', 'Mark areas the heroes have discovered', revealing, '2'),
      ...revealOptions,
      modeBtn({ kind: 'hide' }, 'Hide', 'Un-discover a square', mode.kind === 'hide', '3'),
      modeBtn({ kind: 'block' }, 'Block square', 'Put a blocked square down during play'),
      modeBtn({ kind: 'addMonster', monsterType }, 'Add monster', 'Place a new monster'),
      monsterSelect,
      h('span', { class: 'text-xs opacity-60' }, hint[mode.kind]),
    );
  }

  function statControl(label: string, value: number, max: number, onChange: (v: number) => void): HTMLElement {
    return h('div', { class: 'flex items-center gap-1 text-sm' },
      h('span', { class: 'w-10 opacity-70' }, label),
      h('button', { type: 'button', class: smallBtn, 'aria-label': `Decrease ${label}`, disabled: value <= 0, onclick: () => { onChange(value - 1); } }, '−'),
      h('span', { class: `w-12 text-center font-mono ${value <= 0 ? 'text-danger' : ''}` }, `${value}/${max}`),
      h('button', { type: 'button', class: smallBtn, 'aria-label': `Increase ${label}`, onclick: () => { onChange(value + 1); } }, '+'));
  }

  const sections: SectionContext = {
    send: (c) => { void send(c); },
    warn: (text) => {
      message = text;
      refresh();
    },
    isOpen: (key) => !closedSections.has(key),
    setOpen: (key, open) => {
      if (open) {
        closedSections.delete(key);
      } else {
        closedSections.add(key);
      }
    },
  };

  function renderHeroes(): void {
    const cards = state.heroes.map((hero) => {
      const cls = catalog.heroes.find((c) => c.id === hero.class)?.name ?? hero.class;
      const selected = selectedId === hero.id;
      return h('section', { class: `space-y-2 rounded-lg border p-3 ${selected ? 'border-amber-500 bg-amber-500/5' : 'border-border/60'}` },
        h('button', {
          type: 'button',
          class: 'flex w-full items-baseline justify-between text-left',
          onclick: () => {
            selectedId = selected ? null : hero.id;
            mode = { kind: 'select' };
            refresh();
          },
        },
          h('span', { class: 'font-semibold' }, hero.name, h('span', { class: 'ml-1 text-xs opacity-60' }, `${cls}${hero.player ? ` · ${hero.player}` : ''}`)),
          h('span', { class: 'text-xs opacity-60' }, hero.placed ? `(${hero.x}, ${hero.y})` : 'not on board')),
        totalsLine(hero),
        statControl('Body', hero.body, hero.maxBody, (v) => { heroCmd(hero, { body: v }); }),
        statControl('Mind', hero.mind, hero.maxMind, (v) => { heroCmd(hero, { mind: v }); }),
        manaCap(hero) > 0 ? statControl('Mana', hero.mana ?? 0, manaCap(hero), (v) => { heroCmd(hero, { mana: v }); }) : null,
        h('select', { class: field, 'aria-label': `${hero.name} status`, onchange: (e: Event) => { heroCmd(hero, { status: (e.target as HTMLSelectElement).value }); } },
          ...(['active', 'dead', 'escaped'] as const).map((s) => h('option', { value: s, selected: hero.status === s }, s))),
        effectsBlock(hero.id, hero.name, hero.effects, (c) => { void send(c); }),
        abilitySection(hero, state.round, sections),
        inventorySection(hero, state.heroes, sections),
        hero.equipment ? h('textarea', { class: `${field} h-14 w-full`, placeholder: 'Equipment', title: 'Equipment notes (items are tracked in the inventory)', onchange: (e: Event) => { heroCmd(hero, { equipment: (e.target as HTMLTextAreaElement).value }); } }, hero.equipment) : null,
        h('textarea', { class: `${field} h-14 w-full`, placeholder: 'Notes', onchange: (e: Event) => { heroCmd(hero, { notes: (e.target as HTMLTextAreaElement).value }); } }, hero.notes ?? ''),
        selected ? h('p', { class: 'text-xs text-amber-400' }, hero.placed ? 'Click a square to move this hero.' : 'Click a square to place this hero.') : null,
      );
    });
    replaceChildren(leftPanel, h('h2', { class: 'text-sm font-semibold' }, 'Heroes'), purseControl(state.gold ?? 0, sections), effectSuggestions(), ...cards);
  }

  /** The hero's combat totals (class plus equipped items); the title shows the class's own line. */
  function totalsLine(hero: Hero): HTMLElement | null {
    const totals = combatTotals(hero);
    if (!totals || !hero.combat) {
      return null;
    }
    return h('p', { class: 'text-xs opacity-70', title: `With equipped items. Class: ${combatLine(hero.combat)}` }, combatLine(totals));
  }

  /** A living monster at or below its Faltering threshold (a quarter of maximum Body). */
  function faltering(m: Monster): boolean {
    return m.alive && m.body > 0 && m.body <= falterAt(m.maxBody);
  }

  function renderSelection(): HTMLElement | null {
    if (!selectedId) {
      return null;
    }
    const id = selectedId;
    const monster = state.monsters.find((m) => m.id === id);
    const door = state.quest.doors.find((d) => d.id === id);
    const trap = state.quest.traps.find((t) => t.id === id);
    const note = state.quest.notes.find((n) => n.id === id);
    const block = state.quest.blockedSquares.find((r) => r.id === id);
    const added = (state.addedBlocks ?? []).find((r) => r.id === id);
    const furniture = state.quest.furniture.find((f) => f.id === id);
    const rows: (Node | null)[] = [];
    if (monster) {
      rows.push(
        h('p', { class: 'font-semibold' }, `${monster.name} `, h('span', { class: 'text-xs opacity-60' }, monster.id)),
        monster.combat ? h('p', { class: 'text-xs opacity-80', title: 'Combat stats from the campaign' }, monsterLine(monster.combat)) : null,
        monster.combat?.abilities ? h('p', { class: 'whitespace-pre-wrap text-xs italic opacity-70', title: "Abilities from the campaign's stat line (on the player screen card)" }, monster.combat.abilities) : null,
        monster.combat && faltering(monster)
          ? h('p', { class: 'text-xs font-semibold text-amber-400' }, `Faltering: Avoidance ${monster.combat.avoidance - FALTER_PENALTY} (at ${falterAt(monster.maxBody)} Body or less)`)
          : null,
        statControl('Body', monster.body, monster.maxBody, (v) => { monsterCmd(monster, { body: v }); }),
        oddsBlock(monster, state.heroes, sections.isOpen('odds'), (open) => { sections.setOpen('odds', open); }),
        effectsBlock(monster.id, monster.name, monster.effects, (c) => { void send(c); }),
        h('div', { class: 'flex flex-wrap gap-2' },
          playersButton(monster.id),
          h('button', { type: 'button', class: btn, onclick: () => { monsterCmd(monster, { alive: !monster.alive }); } }, monster.alive ? 'Kill' : 'Revive'),
          h('button', { type: 'button', class: `${btn} text-danger`, onclick: () => { selectedId = null; void send({ type: 'monster.remove', payload: { id: monster.id } }); } }, 'Remove')),
      );
    } else if (door) {
      const live = state.doors.find((d) => d.id === door.id);
      rows.push(
        h('p', { class: 'font-semibold' }, `${door.kind === 'secret' ? 'Secret door' : door.kind === 'gate' ? 'Gate' : 'Door'} ${door.id}`,
          live?.locked ? h('span', { class: 'ml-2 text-xs text-amber-400' }, 'locked') : null),
        h('div', { class: 'flex flex-wrap gap-2' },
          h('button', { type: 'button', class: btn, onclick: () => { void send({ type: 'door.set', payload: { id: door.id, state: live?.state === 'open' ? 'closed' : 'open' } }); } },
            live?.state === 'open' ? 'Close' : 'Open'),
          h('button', { type: 'button', class: btn, onclick: () => { void send({ type: 'door.set', payload: { id: door.id, locked: !(live?.locked ?? false) } }); } },
            live?.locked ? 'Unlock' : 'Lock'),
          door.kind === 'secret'
            ? h('button', { type: 'button', class: btn, onclick: () => { void send({ type: 'door.set', payload: { id: door.id, found: !(live?.found ?? false) } }); } },
              live?.found ? 'Mark not found' : 'Mark found')
            : null,
          playersButton(door.id)),
      );
    } else if (trap) {
      const movable = catalog.traps.find((t) => t.id === trap.kind)?.movable ?? false;
      const at = trapPosition(trap);
      rows.push(
        h('p', { class: 'font-semibold' }, `${trapName(trap)} `, h('span', { class: 'text-xs opacity-60' }, `${trap.id} (${at.x}, ${at.y})`)),
        trapButtons(trap.id),
        movable && trapLiveState(trap.id) !== 'removed' ? h('p', { class: 'text-xs opacity-70' }, 'Click a square to move it there.') : null,
      );
    } else if (added) {
      rows.push(
        h('p', { class: 'font-semibold' }, 'Blocked squares (added during play) ', h('span', { class: 'text-xs opacity-60' }, `${added.id} (${added.x}, ${added.y})`)),
        h('button', { type: 'button', class: btn, onclick: () => { selectedId = null; void send({ type: 'block.remove', payload: { id: added.id } }); } }, 'Clear'),
        h('p', { class: 'text-xs opacity-60' }, 'Always on the player screen.'),
      );
    } else if (block) {
      rows.push(
        h('p', { class: 'font-semibold' }, block.hiddenDoor ? 'Blocked square (hides a secret door) ' : 'Blocked squares ', h('span', { class: 'text-xs opacity-60' }, `${block.id} (${block.x}, ${block.y})`)),
        h('div', { class: 'flex flex-wrap gap-2' }, blockButton(block.id, block.hiddenDoor ?? false), playersButton(block.id)),
      );
    } else if (furniture) {
      rows.push(
        h('p', { class: 'font-semibold' }, `${catalog.furniture.find((f) => f.id === furniture.type)?.name ?? furniture.type.replaceAll('_', ' ')} `,
          h('span', { class: 'text-xs opacity-60' }, `${furniture.id} (${furniture.x}, ${furniture.y})`)),
        playersButton(furniture.id),
      );
    } else if (note) {
      rows.push(h('p', { class: 'font-semibold' }, `Note ${note.label}`), h('p', { class: 'whitespace-pre-wrap text-sm' }, note.text || '(no text)'), noteButton(note.id));
    } else {
      return null;
    }
    return h('section', { class: 'space-y-2 rounded-lg border border-amber-500/50 p-3' },
      h('div', { class: 'flex items-center justify-between' },
        h('h2', { class: 'text-sm font-semibold' }, 'Selected'),
        h('button', { type: 'button', class: 'text-xs opacity-60 hover:opacity-100', onclick: () => { selectedId = null; refresh(); } }, 'Clear')),
      ...rows);
  }

  /** A trap's catalog name (or "Trigger" / "chest trap" for markers), plus its label. */
  function trapName(t: TrapDoc): string {
    const name = catalog.traps.find((d) => d.id === t.kind)?.name ?? (t.kind === 'trigger' ? 'Trigger' : `${t.kind.replaceAll('_', ' ')} trap`);
    return t.label ? `${name} ${t.label}` : name;
  }

  function trapLiveState(trapId: string): LiveTrapState | undefined {
    return state.traps.find((t) => t.id === trapId)?.state;
  }

  /** Where a trap is now: moved during play, or where the quest put it. */
  function trapPosition(t: TrapDoc): TileCoord {
    return state.traps.find((l) => l.id === t.id)?.at ?? { x: t.x, y: t.y };
  }

  function trapButtons(trapId: string): HTMLElement {
    const current = trapLiveState(trapId);
    const kind = state.quest.traps.find((t) => t.id === trapId)?.kind ?? '';
    const trigger = triggerButton(kind);
    return h('div', { class: 'space-y-1' },
      current !== 'removed'
        ? h('div', { class: 'flex flex-wrap gap-1' },
          h('button', { type: 'button', class: `${btn} border-danger/60`, title: trigger.title, onclick: () => { void send({ type: 'trap.trigger', payload: { id: trapId } }); } }, trigger.label),
          h('button', {
            type: 'button',
            class: btn,
            title: 'Take the trap off and block the squares it covers where it is now (a boulder where it stopped)',
            onclick: () => { void send({ type: 'trap.block', payload: { id: trapId } }); },
          }, 'Turn into blocked squares'))
        : null,
      h('div', { class: 'flex flex-wrap gap-1' },
        ...TRAP_STATES.map((s) => h('button', {
          type: 'button',
          title: s === 'removed' ? 'Take it off the board (pick another state to bring it back)' : `Set to ${s}`,
          class: current === s ? btnActive : s === 'removed' ? `${btn} text-danger` : btn,
          onclick: () => { if (current !== s) { void send({ type: 'trap.set', payload: { id: trapId, state: s } }); } },
        }, s === 'removed' ? 'remove' : s))));
  }

  /** Shows or hides a piece on the player screen (null for things that can't be shown). */
  function playersButton(id: string): HTMLElement | null {
    const shown = shownToPlayers(state, id);
    if (shown === null) {
      return null;
    }
    return h('button', {
      type: 'button',
      class: shown ? `${btn} border-positive/60 text-positive` : btn,
      title: shown ? 'On the player screen. Click to hide it from the players.' : 'Not on the player screen. Click to show it to the players.',
      onclick: () => { void send({ type: 'seen.set', payload: { id, seen: !shown } }); },
    }, shown ? 'Shown to players' : 'Show to players');
  }

  function blockButton(blockId: string, hiddenDoor: boolean): HTMLElement {
    const removed = (state.removedBlocks ?? []).includes(blockId);
    const label = removed ? 'Put back' : hiddenDoor ? 'Found the secret door (remove)' : 'Remove';
    return h('button', { type: 'button', class: btn, onclick: () => { void send({ type: 'blocked.set', payload: { id: blockId, removed: !removed } }); } }, label);
  }

  function noteButton(noteId: string): HTMLElement {
    const used = state.consumedNotes.includes(noteId);
    return h('button', { type: 'button', class: btn, onclick: () => { void send({ type: 'note.consume', payload: { id: noteId, consumed: !used } }); } }, used ? 'Mark unused' : 'Mark used');
  }

  /** Loads a clip into the player and plays it. */
  function playClip(clipId: string): void {
    const url = clips[clipId];
    if (!url) {
      return;
    }
    if (player.dataset.clip !== clipId) {
      player.src = url;
      player.dataset.clip = clipId;
    }
    void player.play().catch(() => undefined);
    refresh();
  }

  /** Opens a passage in the reader (null closes it); its clip is loaded, and played with autoplay on. */
  function openPassage(id: string | null): void {
    readerId = id;
    player.pause();
    const first = id ? passageClips(id, clips)[0] : undefined;
    if (first) {
      player.src = first.url;
      player.dataset.clip = first.id;
      if (autoplay) {
        void player.play().catch(() => undefined);
      }
    } else {
      player.removeAttribute('src');
      delete player.dataset.clip;
      player.load();
    }
    refresh();
  }

  function readAloudContext(): ReadAloudContext {
    return {
      sections: scriptSections,
      current: currentSection(scriptSections, state, chapters),
      read: state.readPassages ?? [],
      campaignId: session.campaignId,
      send: (c) => { void send(c); },
      open: openPassage,
      sectionOpen: (i) => scriptSectionOpen.get(i),
      setSectionOpen: (i, open) => { scriptSectionOpen.set(i, open); },
      reload: () => {
        void Promise.all([loadScript(), loadClips()]).then(([sections, loadedClips]) => {
          scriptSections = sections;
          clips = loadedClips;
          scriptSectionOpen.clear();
          refresh();
        });
      },
      clips,
      player,
      playClip,
      autoplay,
      setAutoplay: (on) => {
        autoplay = on;
        try {
          localStorage.setItem(AUTOPLAY_KEY, on ? '1' : '0');
        } catch {
          // storage unavailable: the choice lasts until the page reloads
        }
      },
    };
  }

  function renderReader(): void {
    const overlay = readerId ? readerOverlay(readAloudContext(), readerId, status === 'active') : null;
    if (!overlay && readerId) {
      readerId = null;
      player.pause();
    }
    replaceChildren(readerHost, overlay);
  }

  function renderRight(): void {
    const logInput = h('textarea', { class: `${field} h-16 w-full`, placeholder: 'What happened? (a bargain, a rule bend, a story beat…)', maxlength: 2000 });
    const monsters = state.monsters.map((m) =>
      h('li', { class: 'flex items-center justify-between gap-2 text-sm' },
        h('button', { type: 'button', class: `text-left ${m.alive ? '' : 'line-through opacity-50'}`, onclick: () => { selectedId = m.id; refresh(); } },
          `${m.name} `, h('span', { class: 'text-xs opacity-60' }, `${m.id}${m.visibility === 'hidden' ? ' · hidden' : ''}`),
          m.combat && faltering(m) ? h('span', { class: 'ml-1 text-xs font-semibold text-amber-400' }, 'faltering') : null),
        h('span', { class: 'font-mono text-xs' }, `${m.body}/${m.maxBody}`)));
    const traps = state.quest.traps.map((t) => {
      const at = trapPosition(t);
      const removed = trapLiveState(t.id) === 'removed';
      return h('li', { class: 'space-y-1 text-sm' },
        h('button', { type: 'button', class: `text-left ${removed ? 'line-through opacity-50' : ''}`, onclick: () => { selectedId = t.id; setMode({ kind: 'select' }); } },
          `${trapName(t)} `, h('span', { class: 'text-xs opacity-60' }, `${t.id} (${at.x}, ${at.y})${removed ? ' · removed' : ''}`)),
        trapButtons(t.id));
    });
    // Blocks worth listing: ones hiding a secret door, and any already removed (to put back).
    const removedBlocks = new Set(state.removedBlocks ?? []);
    const blocks = state.quest.blockedSquares.filter((r) => (r.hiddenDoor ?? false) || removedBlocks.has(r.id)).map((r) =>
      h('li', { class: 'flex items-center justify-between gap-2 text-sm' },
        h('span', { class: removedBlocks.has(r.id) ? 'line-through opacity-50' : '' }, r.hiddenDoor ? 'Secret door ' : 'Blocked ', h('span', { class: 'text-xs opacity-60' }, `${r.id} (${r.x}, ${r.y})`)),
        blockButton(r.id, r.hiddenDoor ?? false)));
    const notes = state.quest.notes.map((n) =>
      h('li', { class: 'flex items-start justify-between gap-2 text-sm' },
        h('span', { class: state.consumedNotes.includes(n.id) ? 'line-through opacity-50' : '' }, h('strong', {}, `${n.label}: `), n.text || '(no text)'),
        noteButton(n.id)));
    const eventItems = [...events].reverse().slice(0, 300).map((e) => {
      const line = formatEvent(e);
      return h('li', { class: `border-l-2 pl-2 text-sm ${KIND_STYLES[line.kind] ?? KIND_STYLES.other}` },
        h('div', { class: 'text-xs opacity-60' }, `${line.round} · ${line.time}`), line.summary);
    });

    replaceChildren(
      rightPanel,
      status !== 'active' ? h('p', { class: 'rounded-md border border-purple-400/60 bg-purple-400/10 p-2 text-sm' }, 'This quest is completed. Reopen it to make changes.') : null,
      renderSelection(),
      readAloudPanel(readAloudContext()),
      h('section', { class: 'space-y-2' },
        h('h2', { class: 'text-sm font-semibold' }, 'Log'),
        logInput,
        h('button', {
          type: 'button',
          class: btn,
          onclick: () => {
            const text = logInput.value.trim();
            if (text) {
              void send({ type: 'log.note', payload: { text } });
            }
          },
        }, 'Add to log'),
        h('ol', { class: 'max-h-80 space-y-2 overflow-y-auto' }, ...eventItems)),
      h('section', { class: 'space-y-1' }, h('h2', { class: 'text-sm font-semibold' }, `Monsters (${state.monsters.filter((m) => m.alive).length} alive)`), h('ul', { class: 'space-y-1' }, ...monsters)),
      traps.length ? h('section', { class: 'space-y-1' }, h('h2', { class: 'text-sm font-semibold' }, 'Traps'), h('ul', { class: 'space-y-2' }, ...traps)) : null,
      blocks.length ? h('section', { class: 'space-y-1' }, h('h2', { class: 'text-sm font-semibold' }, 'Blocked squares'), h('ul', { class: 'space-y-2' }, ...blocks)) : null,
      notes.length ? h('section', { class: 'space-y-1' }, h('h2', { class: 'text-sm font-semibold' }, 'Quest notes'), h('ul', { class: 'space-y-2' }, ...notes)) : null,
    );
  }

  function refresh(): void {
    const restoreFocus = preserveFocus(document.body);
    renderHeader();
    renderModeBar();
    renderHeroes();
    renderRight();
    renderReader();
    restoreFocus();
    requestDraw();
  }

  new ResizeObserver(requestDraw).observe(canvas);
  refresh();
  connect();
}

void main().catch((err: unknown) => {
  const root = document.querySelector('#tracker');
  if (root) {
    root.textContent = `The tracker failed to load: ${err instanceof Error ? err.message : String(err)}`;
  }
  console.error(err);
});
