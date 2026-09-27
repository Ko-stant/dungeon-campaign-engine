/**
 * Tracker page (/play/{sessionId}): the GM's live record of a game at the
 * table. Every change is a command sent to the server, which saves the new
 * state and a readable event, then pushes both to every open tab.
 */
import { pixelToEdge, pixelToTile } from '../board/geometry.ts';
import { tileIndex } from '../board/model.ts';
import { monsterOptionLabel } from '../maps/types.ts';
import { BoardRenderer } from '../board/renderer.ts';
import { ApiError } from '../api/http.ts';
import { createTrackerApi } from '../tracker/api.ts';
import { formatEvent } from '../tracker/format.ts';
import { clickCommand, type ClickTarget, type Mode } from '../tracker/interaction.ts';
import type { Command, CommandResponse, Hero, Monster, SessionEvent, SessionState } from '../tracker/types.ts';
import { trackerView } from '../tracker/view.ts';
import { h, replaceChildren } from '../ui/dom.ts';

const TRAP_STATES = ['hidden', 'revealed', 'triggered', 'disarmed'] as const;
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

  let state: SessionState = session.state;
  let status = session.status;
  let lastSeq = session.eventSeq;
  const events: SessionEvent[] = initialEvents;
  let selectedId: string | null = null;
  let mode: Mode = { kind: 'select' };
  let monsterType = catalog.monsters[0]?.id ?? '';
  let fog = true;
  let busy = false;
  let message = '';
  let live = false;

  // --- Layout ---
  const canvas = h('canvas', { class: 'block h-full w-full' });
  const header = h('header', { class: 'flex items-center gap-3 overflow-x-auto whitespace-nowrap border-b border-border/60 px-3 py-2' });
  const modeBar = h('div', { class: 'flex flex-wrap items-center gap-2 px-2 pb-2' });
  const leftPanel = h('aside', { class: 'w-80 shrink-0 space-y-3 overflow-y-auto border-r border-border/60 p-3' });
  const rightPanel = h('aside', { class: 'w-80 shrink-0 space-y-4 overflow-y-auto border-l border-border/60 p-3' });
  const hoverInfo = h('span', { class: 'pointer-events-none absolute bottom-3 left-3 rounded bg-surface/80 px-2 py-0.5 font-mono text-xs opacity-80 empty:hidden' });
  replaceChildren(
    root,
    header,
    h('div', { class: 'flex min-h-0 flex-1' },
      leftPanel,
      h('main', { class: 'relative flex min-w-0 flex-1 flex-col p-2' }, modeBar, h('div', { class: 'min-h-0 flex-1' }, canvas), hoverInfo),
      rightPanel),
  );

  // --- Drawing ---
  let frame = 0;
  const requestDraw = (): void => {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => {
      renderer.draw(trackerView(state, catalog, { fog }), { selectedId });
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
    const result = clickCommand(state, mode, selectedId, targetAt(ev));
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

  canvas.addEventListener('mousemove', (ev) => {
    const t = targetAt(ev).tile;
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
    if (ev.key === 'Escape') {
      selectedId = null;
      mode = { kind: 'select' };
      refresh();
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
  function renderHeader(): void {
    const completed = status !== 'active';
    replaceChildren(
      header,
      h('a', { href: `/campaigns/${session.campaignId}`, class: 'text-sm opacity-70 hover:text-amber-400' }, '← Campaign'),
      h('div', { class: 'flex flex-col leading-tight' },
        h('span', { class: 'font-semibold' }, session.name),
        h('span', { class: 'text-xs opacity-60' }, state.questName)),
      h('span', { class: 'mx-2 h-6 w-px bg-border/60' }),
      h('span', { class: 'text-lg font-bold text-amber-400', 'aria-live': 'polite' }, `Round ${state.round}`),
      h('button', { type: 'button', class: btn, disabled: completed || busy, onclick: () => { void send({ type: 'round.advance', payload: {} }); } }, 'Next round'),
      h('label', { class: 'flex items-center gap-1 text-sm' },
        h('input', { type: 'checkbox', checked: fog, onchange: (e: Event) => { fog = (e.target as HTMLInputElement).checked; requestDraw(); } }),
        'Show what the heroes have seen'),
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

  function renderModeBar(): void {
    const modeBtn = (m: Mode, label: string, title: string): HTMLButtonElement =>
      h('button', {
        type: 'button',
        title,
        class: mode.kind === m.kind ? btnActive : btn,
        onclick: () => {
          mode = m;
          refresh();
        },
      }, label);
    const monsterSelect = h('select', {
      class: field,
      'aria-label': 'Monster to add',
      onchange: (e: Event) => {
        monsterType = (e.target as HTMLSelectElement).value;
        mode = { kind: 'addMonster', monsterType };
        refresh();
      },
    }, ...catalog.monsters.map((m) => h('option', { value: m.id, selected: m.id === monsterType }, monsterOptionLabel(m))));
    const hint: Record<Mode['kind'], string> = {
      select: 'Click a hero or monster, then a square to move it. Click a door to open or close it.',
      reveal: 'Click a room to reveal it (or a corridor square).',
      hide: 'Click a square to hide it again.',
      addMonster: 'Click a square to place the monster.',
    };
    replaceChildren(
      modeBar,
      modeBtn({ kind: 'select' }, 'Select / move', 'Select pieces, move them, open and close doors'),
      modeBtn({ kind: 'reveal' }, 'Reveal', 'Mark areas the heroes have discovered'),
      modeBtn({ kind: 'hide' }, 'Hide', 'Un-discover a square'),
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
        statControl('Body', hero.body, hero.maxBody, (v) => { heroCmd(hero, { body: v }); }),
        statControl('Mind', hero.mind, hero.maxMind, (v) => { heroCmd(hero, { mind: v }); }),
        h('div', { class: 'flex items-center gap-2 text-sm' },
          h('label', { class: 'flex items-center gap-1' }, h('span', { class: 'opacity-70' }, 'Gold'),
            h('input', { type: 'number', min: 0, class: `${field} w-20`, value: hero.gold, onchange: (e: Event) => { heroCmd(hero, { gold: Number((e.target as HTMLInputElement).value) }); } })),
          h('select', { class: field, 'aria-label': `${hero.name} status`, onchange: (e: Event) => { heroCmd(hero, { status: (e.target as HTMLSelectElement).value }); } },
            ...(['active', 'dead', 'escaped'] as const).map((s) => h('option', { value: s, selected: hero.status === s }, s)))),
        h('textarea', { class: `${field} h-14 w-full`, placeholder: 'Equipment', onchange: (e: Event) => { heroCmd(hero, { equipment: (e.target as HTMLTextAreaElement).value }); } }, hero.equipment ?? ''),
        h('textarea', { class: `${field} h-14 w-full`, placeholder: 'Notes', onchange: (e: Event) => { heroCmd(hero, { notes: (e.target as HTMLTextAreaElement).value }); } }, hero.notes ?? ''),
        selected ? h('p', { class: 'text-xs text-amber-400' }, hero.placed ? 'Click a square to move this hero.' : 'Click a square to place this hero.') : null,
      );
    });
    replaceChildren(leftPanel, h('h2', { class: 'text-sm font-semibold' }, 'Heroes'), ...cards);
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
    const rows: (Node | null)[] = [];
    if (monster) {
      rows.push(
        h('p', { class: 'font-semibold' }, `${monster.name} `, h('span', { class: 'text-xs opacity-60' }, monster.id)),
        statControl('Body', monster.body, monster.maxBody, (v) => { monsterCmd(monster, { body: v }); }),
        h('div', { class: 'flex flex-wrap gap-2' },
          h('button', { type: 'button', class: btn, onclick: () => { monsterCmd(monster, { visibility: monster.visibility === 'hidden' ? 'seen' : 'hidden' }); } },
            monster.visibility === 'hidden' ? 'Mark seen' : 'Mark hidden'),
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
            : null),
      );
    } else if (trap) {
      rows.push(h('p', { class: 'font-semibold' }, `${trap.kind.replaceAll('_', ' ')} trap ${trap.id}`), trapButtons(trap.id));
    } else if (block) {
      rows.push(
        h('p', { class: 'font-semibold' }, block.hiddenDoor ? 'Blocked square (hides a secret door) ' : 'Blocked squares ', h('span', { class: 'text-xs opacity-60' }, `${block.id} (${block.x}, ${block.y})`)),
        blockButton(block.id, block.hiddenDoor ?? false),
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

  function trapButtons(trapId: string): HTMLElement {
    const current = state.traps.find((t) => t.id === trapId)?.state;
    return h('div', { class: 'flex flex-wrap gap-1' },
      ...TRAP_STATES.map((s) => h('button', {
        type: 'button',
        class: current === s ? btnActive : btn,
        onclick: () => { if (current !== s) { void send({ type: 'trap.set', payload: { id: trapId, state: s } }); } },
      }, s)));
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

  function renderRight(): void {
    const logInput = h('textarea', { class: `${field} h-16 w-full`, placeholder: 'What happened? (a bargain, a rule bend, a story beat…)', maxlength: 2000 });
    const monsters = state.monsters.map((m) =>
      h('li', { class: 'flex items-center justify-between gap-2 text-sm' },
        h('button', { type: 'button', class: `text-left ${m.alive ? '' : 'line-through opacity-50'}`, onclick: () => { selectedId = m.id; refresh(); } },
          `${m.name} `, h('span', { class: 'text-xs opacity-60' }, `${m.id}${m.visibility === 'hidden' ? ' · hidden' : ''}`)),
        h('span', { class: 'font-mono text-xs' }, `${m.body}/${m.maxBody}`)));
    const traps = state.quest.traps.map((t) =>
      h('li', { class: 'space-y-1 text-sm' }, h('span', {}, `${t.kind.replaceAll('_', ' ')} `, h('span', { class: 'text-xs opacity-60' }, `${t.id} (${t.x}, ${t.y})`)), trapButtons(t.id)));
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
    renderHeader();
    renderModeBar();
    renderHeroes();
    renderRight();
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
