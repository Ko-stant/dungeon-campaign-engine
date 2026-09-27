/**
 * Map editor page (/maps/{id}/edit). Wiring only: every edit goes through the
 * pure functions in editor/model.ts and editor/tools.ts and lands on the undo
 * history; the renderer draws toBoardView() of the current documents.
 */
import { pixelToEdge, pixelToTile, type Rotation, type TileCoord } from '../board/geometry.ts';
import { CORRIDOR, VOID, tileAt, tileIndex, type DoorKind } from '../board/model.ts';
import { BoardRenderer, type Highlights } from '../board/renderer.ts';
import { History } from '../editor/history.ts';
import {
  addRoom,
  itemsAt,
  moveItem,
  removeItem,
  renameRoom,
  setBlockedHiddenDoor,
  setRoomColor,
  resizeBoard,
  rotateItem,
  setNoteText,
  toBoardView,
  toggleDoorState,
  updateDoor,
} from '../editor/model.ts';
import { applyClick, applyDrag, isDragTool, lineTiles, type ClickTarget, type EditorDoc, type Tool } from '../editor/tools.ts';
import { ApiError, createApi } from '../maps/api.ts';
import { monsterOptionLabel, type Issue, type QuestSummary } from '../maps/types.ts';
import { h, replaceChildren } from '../ui/dom.ts';

const TRAP_KINDS = ['pit', 'spear', 'falling_block', 'chest', 'other'] as const;

type Layer = 'board' | 'quest';
type BoardBrush = 'corridor' | 'void' | 'room' | 'wall';
type QuestToolKind = 'select' | 'door' | 'blocked' | 'furniture' | 'monster' | 'trap' | 'note' | 'start' | 'exit' | 'erase';

const QUEST_TOOLS: { kind: QuestToolKind; label: string; hint: string }[] = [
  { kind: 'select', label: 'Select / move', hint: 'Click a piece or door to edit it; drag a piece to move it.' },
  { kind: 'door', label: 'Door', hint: 'Pick a kind below, then click a wall edge. Click again with the same choice to remove it.' },
  { kind: 'blocked', label: 'Blocked squares', hint: 'Drag to cover impassable squares.' },
  { kind: 'furniture', label: 'Furniture', hint: 'Click the bottom-left square. R rotates the selection.' },
  { kind: 'monster', label: 'Monster', hint: 'Click a square to place.' },
  { kind: 'trap', label: 'Trap', hint: 'Click a square to place (starts hidden).' },
  { kind: 'note', label: 'Note', hint: 'Click a square, then type the note text.' },
  { kind: 'start', label: 'Start square', hint: 'Click to toggle a hero start square (green).' },
  { kind: 'exit', label: 'Exit square', hint: 'Click to toggle an exit square (magenta).' },
  { kind: 'erase', label: 'Erase', hint: 'Click a piece or door to remove it.' },
];

const DOOR_KINDS: [DoorKind, string][] = [['normal', 'Door'], ['secret', 'Secret door'], ['gate', 'Gate']];

function doorKindLabel(kind: DoorKind): string {
  return DOOR_KINDS.find(([k]) => k === kind)?.[1] ?? 'Door';
}

const btn = 'rounded-md border border-border/60 px-2 py-1 text-sm hover:border-amber-500 disabled:opacity-40 disabled:hover:border-border/60';
const btnActive = 'rounded-md border border-amber-500 bg-amber-500/15 px-2 py-1 text-sm';
const field = 'rounded-md border border-border/60 bg-surface px-2 py-1 text-sm';
const input = `w-full ${field}`;

async function main(): Promise<void> {
  const root = document.querySelector<HTMLElement>('#map-editor');
  const boardId = root?.dataset.boardId;
  if (!root || !boardId) {
    throw new Error('map editor root element or board id missing');
  }
  const api = createApi();
  const [boardRes, catalog, initialQuests] = await Promise.all([api.board(boardId), api.catalog(), api.quests(boardId)]);

  // --- State ---
  let boardName = boardRes.name;
  let quests: QuestSummary[] = initialQuests;
  let questId: string | null = null;
  let questName = '';
  let issues: Issue[] = [];
  let history = new History<EditorDoc>({ board: boardRes.board, quest: null });
  let saving = false;
  let status = 'Loaded';

  let layer: Layer = 'board';
  let brush: BoardBrush = 'corridor';
  let rectMode = false;
  let activeRoom: number | null = null;
  let questTool: QuestToolKind = 'select';
  let furnitureType = catalog.furniture[0]?.id ?? '';
  let furnitureRotation: Rotation = 0;
  let monsterType = catalog.monsters[0]?.id ?? '';
  let trapKind: string = TRAP_KINDS[0];
  let doorKind: DoorKind = 'normal';
  let doorLocked = false;
  let blockHidesDoor = false;
  let selectedId: string | null = null;
  let hover: ClickTarget = { tile: null, edge: null };
  let drag: { points: TileCoord[] } | null = null;
  let moving: { id: string; from: TileCoord; to: TileCoord } | null = null;
  let focusNoteText = false;

  const doc = (): EditorDoc => history.current;

  // --- Layout ---
  const canvas = h('canvas', { class: 'block h-full w-full touch-none' });
  const nameInput = h('input', { class: `${field} w-64 font-semibold`, value: boardName, maxlength: 120, 'aria-label': 'Board name' });
  const widthInput = h('input', { class: `${field} w-16`, type: 'number', min: 1, max: 200, 'aria-label': 'Columns (width)', title: 'Columns, left to right' });
  const heightInput = h('input', { class: `${field} w-16`, type: 'number', min: 1, max: 200, 'aria-label': 'Rows (height)', title: 'Rows, bottom to top' });
  const resizeBtn = h('button', { class: btn, type: 'button' }, 'Resize');
  const undoBtn = h('button', { class: btn, type: 'button', title: 'Undo (Ctrl+Z)' }, 'Undo');
  const redoBtn = h('button', { class: btn, type: 'button', title: 'Redo (Ctrl+Shift+Z)' }, 'Redo');
  const saveBtn = h('button', { class: 'rounded-md bg-amber-600 px-3 py-1 text-sm font-semibold text-white hover:bg-amber-700 disabled:opacity-50', type: 'button', title: 'Save (Ctrl+S)' }, 'Save');
  const statusEl = h('span', { class: 'text-xs opacity-70', role: 'status' });
  const leftPanel = h('aside', { class: 'w-72 shrink-0 space-y-4 overflow-y-auto border-r border-border/60 p-3' });
  const rightPanel = h('aside', { class: 'w-72 shrink-0 space-y-4 overflow-y-auto border-l border-border/60 p-3' });
  const hoverInfo = h('span', { class: 'pointer-events-none absolute bottom-3 left-3 rounded bg-surface/80 px-2 py-0.5 font-mono text-xs opacity-80 empty:hidden' });

  replaceChildren(
    root,
    h(
      'header',
      { class: 'flex items-center gap-2 overflow-x-auto whitespace-nowrap border-b border-border/60 px-3 py-2' },
      h('a', { href: '/maps', class: 'text-sm opacity-70 hover:text-amber-400' }, '← Maps'),
      nameInput,
      h('span', { class: 'ml-2 text-xs opacity-70' }, 'Columns × rows'),
      widthInput,
      h('span', { class: 'text-xs opacity-70' }, '×'),
      heightInput,
      resizeBtn,
      h('span', { class: 'mx-2 h-5 w-px bg-border/60' }),
      undoBtn,
      redoBtn,
      saveBtn,
      statusEl,
    ),
    h('div', { class: 'flex min-h-0 flex-1' }, leftPanel, h('main', { class: 'relative min-w-0 flex-1 p-2' }, canvas, hoverInfo), rightPanel),
  );

  // --- Drawing ---
  let frame = 0;
  const requestDraw = (): void => {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(draw);
  };
  const renderer = new BoardRenderer(canvas, requestDraw);

  function previewDoc(): EditorDoc {
    if (drag) {
      return applyDrag(doc(), currentTool(), drag.points);
    }
    if (moving && (moving.to.x !== moving.from.x || moving.to.y !== moving.from.y) && doc().quest) {
      const q = doc().quest;
      return q ? { ...doc(), quest: moveItem(q, moving.id, moving.to) } : doc();
    }
    return doc();
  }

  function draw(): void {
    const d = previewDoc();
    const highlights: Highlights = { selectedId };
    const tool = currentTool();
    if (drag && (tool.kind === 'fill' || tool.kind === 'blocked')) {
      const first = drag.points[0];
      const last = drag.points[drag.points.length - 1];
      if (first && last) {
        highlights.rect = { from: first, to: last };
      }
    } else if (layer === 'board' && brush === 'room' && activeRoom !== null) {
      highlights.tiles = roomTiles(activeRoom);
    }
    if (tool.kind === 'door' || tool.kind === 'wall' || tool.kind === 'erase' || tool.kind === 'select') {
      highlights.edge = hover.edge;
    }
    if (!hover.edge || tool.kind === 'paint' || tool.kind === 'fill') {
      highlights.tile = hover.tile;
    }
    renderer.draw(toBoardView(d.board, layer === 'quest' ? d.quest : d.quest ?? null, catalog), highlights);
  }

  function roomTiles(roomId: number): TileCoord[] {
    const b = doc().board;
    const tiles: TileCoord[] = [];
    b.regions.forEach((r, i) => {
      if (r === roomId) {
        tiles.push(tileAt(b.width, i));
      }
    });
    return tiles;
  }

  // --- Tools ---
  function currentTool(): Tool {
    if (layer === 'board') {
      if (brush === 'wall') {
        return { kind: 'wall' };
      }
      const region = brush === 'corridor' ? CORRIDOR : brush === 'void' ? VOID : (activeRoom ?? CORRIDOR);
      return rectMode ? { kind: 'fill', region } : { kind: 'paint', region };
    }
    switch (questTool) {
      case 'furniture':
        return { kind: 'furniture', type: furnitureType, rotation: furnitureRotation };
      case 'monster':
        return { kind: 'monster', type: monsterType };
      case 'trap':
        return { kind: 'trap', trapKind };
      case 'door':
        return { kind: 'door', doorKind, locked: doorLocked };
      case 'blocked':
        return { kind: 'blocked', hiddenDoor: blockHidesDoor };
      default:
        return { kind: questTool };
    }
  }

  function commit(next: EditorDoc): void {
    if (next !== doc()) {
      history.push(next);
      refresh();
    }
  }

  function targetAt(ev: PointerEvent): ClickTarget {
    const m = renderer.metrics;
    if (!m) {
      return { tile: null, edge: null };
    }
    const rect = canvas.getBoundingClientRect();
    const px = ev.clientX - rect.left;
    const py = ev.clientY - rect.top;
    return { tile: pixelToTile(m, px, py), edge: pixelToEdge(m, px, py) };
  }

  function ensureRoomForPainting(): void {
    if (layer === 'board' && brush === 'room' && activeRoom === null) {
      const added = addRoom(doc().board);
      history.push({ ...doc(), board: added.board });
      activeRoom = added.roomId;
    }
  }

  canvas.addEventListener('pointerdown', (ev) => {
    if (ev.button !== 0) {
      return;
    }
    const target = targetAt(ev);
    hover = target;
    const tool = currentTool();

    if (layer === 'quest' && !doc().quest) {
      status = 'Create or open a quest first (right panel).';
      refresh();
      return;
    }

    if (tool.kind === 'select') {
      const q = doc().quest;
      if (!q) {
        return;
      }
      const edge = target.edge;
      const door = edge ? q.doors.find((d) => d.edge.x === edge.x && d.edge.y === edge.y && d.edge.orientation === edge.orientation) : undefined;
      const top = target.tile ? itemsAt(q, catalog, target.tile)[0] : undefined;
      selectedId = top ?? door?.id ?? null;
      if (top && target.tile) {
        moving = { id: top, from: target.tile, to: target.tile };
        canvas.setPointerCapture(ev.pointerId);
      }
      refresh();
      return;
    }

    if (isDragTool(tool)) {
      if (!target.tile) {
        return;
      }
      ensureRoomForPainting();
      drag = { points: [target.tile] };
      canvas.setPointerCapture(ev.pointerId);
      refresh();
      return;
    }

    const before = doc().quest;
    const next = applyClick(doc(), tool, target, catalog);
    if (tool.kind === 'wall' && target.edge && next === doc()) {
      status = 'That edge is already a wall (the outer edge, or where two areas meet).';
      refresh();
      return;
    }
    commit(next);
    const after = doc().quest;
    if (tool.kind === 'note' && after && after !== before) {
      selectedId = after.notes[after.notes.length - 1]?.id ?? null;
      focusNoteText = true;
      refresh();
    }
  });

  canvas.addEventListener('pointermove', (ev) => {
    hover = targetAt(ev);
    const tile = hover.tile;
    const b = doc().board;
    hoverInfo.textContent = tile ? `(${tile.x}, ${tile.y}) ${regionName(b.regions[tileIndex(b.width, tile)] ?? VOID)}` : '';
    if (drag && tile) {
      const tool = currentTool();
      const last = drag.points[drag.points.length - 1];
      if (last && (last.x !== tile.x || last.y !== tile.y)) {
        if (tool.kind === 'paint') {
          drag.points.push(...lineTiles(last, tile).slice(1));
        } else {
          drag.points = [drag.points[0] ?? tile, tile];
        }
      }
    }
    if (moving && tile) {
      moving.to = tile;
    }
    requestDraw();
  });

  const endPointer = (): void => {
    if (!drag && !moving) {
      return; // plain clicks were already handled on pointerdown
    }
    if (drag) {
      const points = drag.points;
      drag = null;
      commit(applyDrag(doc(), currentTool(), points));
    }
    if (moving) {
      const m = moving;
      moving = null;
      const q = doc().quest;
      if (q && (m.to.x !== m.from.x || m.to.y !== m.from.y)) {
        commit({ ...doc(), quest: moveItem(q, m.id, m.to) });
      }
    }
    refresh();
  };
  canvas.addEventListener('pointerup', endPointer);
  canvas.addEventListener('pointercancel', endPointer);
  canvas.addEventListener('pointerleave', () => {
    hover = { tile: null, edge: null };
    hoverInfo.textContent = '';
    requestDraw();
  });

  function regionName(region: number): string {
    if (region === VOID) {
      return 'solid rock';
    }
    if (region === CORRIDOR) {
      return 'corridor';
    }
    return doc().board.rooms.find((r) => r.id === region)?.name ?? `room ${region}`;
  }

  // --- Commands ---
  function undo(): void {
    history.undo();
    selectedId = null;
    refresh();
  }

  function redo(): void {
    history.redo();
    selectedId = null;
    refresh();
  }

  function deleteSelected(): void {
    const q = doc().quest;
    if (q && selectedId) {
      commit({ ...doc(), quest: removeItem(q, selectedId) });
      selectedId = null;
      refresh();
    }
  }

  function rotateSelected(): void {
    const q = doc().quest;
    if (q && selectedId && q.furniture.some((f) => f.id === selectedId)) {
      commit({ ...doc(), quest: rotateItem(q, selectedId) });
    } else if (layer === 'quest' && questTool === 'furniture') {
      furnitureRotation = ((furnitureRotation + 90) % 360) as Rotation;
      refresh();
    }
  }

  async function save(): Promise<void> {
    if (saving) {
      return;
    }
    const name = nameInput.value.trim();
    if (!name) {
      status = 'The board needs a name.';
      refresh();
      return;
    }
    saving = true;
    status = 'Saving…';
    refresh();
    try {
      const d = doc();
      const savedBoard = await api.saveBoard(boardId ?? '', name, d.board);
      boardName = savedBoard.name;
      document.title = `${boardName} - Dungeon Campaign Engine`;
      if (questId && d.quest) {
        const savedQuest = await api.saveQuest(questId, questName.trim() || 'Untitled quest', d.quest);
        issues = savedQuest.issues;
        questName = savedQuest.name;
        quests = await api.quests(boardId ?? '');
      }
      history.markSaved();
      status = `Saved at ${new Date().toLocaleTimeString()}`;
    } catch (err) {
      status = err instanceof ApiError ? `Save failed: ${err.message}` : 'Save failed.';
    } finally {
      saving = false;
      refresh();
    }
  }

  async function openQuest(id: string | null): Promise<void> {
    if (history.dirty && !confirm('Discard unsaved changes?')) {
      refresh();
      return;
    }
    if (!id) {
      questId = null;
      questName = '';
      issues = [];
      history = new History<EditorDoc>({ board: doc().board, quest: null });
      layer = 'board';
      refresh();
      return;
    }
    const res = await api.quest(id);
    questId = res.id;
    questName = res.name;
    issues = res.issues;
    history = new History<EditorDoc>({ board: doc().board, quest: res.quest });
    layer = 'quest';
    selectedId = null;
    status = `Opened quest "${res.name}"`;
    refresh();
  }

  async function createQuest(name: string): Promise<void> {
    if (history.dirty) {
      await save();
    }
    try {
      const res = await api.createQuest(boardId ?? '', name);
      quests = await api.quests(boardId ?? '');
      questId = res.id;
      questName = res.name;
      issues = res.issues;
      history = new History<EditorDoc>({ board: doc().board, quest: res.quest });
      layer = 'quest';
      status = `Created quest "${res.name}"`;
    } catch (err) {
      status = err instanceof ApiError ? err.message : 'Could not create quest.';
    }
    refresh();
  }

  function resize(): void {
    const w = Number(widthInput.value);
    const hgt = Number(heightInput.value);
    const b = doc().board;
    if (w === b.width && hgt === b.height) {
      return;
    }
    try {
      const next = resizeBoard(b, w, hgt);
      const lost = b.regions.filter((r) => r !== VOID).length - next.regions.filter((r) => r !== VOID).length;
      if (w < b.width || hgt < b.height) {
        const warn = lost > 0 ? ` ${lost} painted square(s) will be cropped.` : '';
        if (!confirm(`Shrink the board to ${w} × ${hgt}?${warn} (Undo restores it.)`)) {
          refresh();
          return;
        }
      }
      commit({ ...doc(), board: next });
      status = `Resized to ${w} × ${hgt}`;
    } catch (err) {
      status = err instanceof Error ? err.message : 'Invalid size';
    }
    refresh();
  }

  undoBtn.addEventListener('click', undo);
  redoBtn.addEventListener('click', redo);
  saveBtn.addEventListener('click', () => void save());
  resizeBtn.addEventListener('click', resize);
  nameInput.addEventListener('input', () => {
    status = 'Unsaved name change';
    refresh();
  });

  window.addEventListener('keydown', (ev) => {
    const typing = ev.target instanceof HTMLInputElement || ev.target instanceof HTMLTextAreaElement || ev.target instanceof HTMLSelectElement;
    const mod = ev.metaKey || ev.ctrlKey;
    if (mod && ev.key.toLowerCase() === 's') {
      ev.preventDefault();
      void save();
    } else if (!typing && mod && ev.key.toLowerCase() === 'z') {
      ev.preventDefault();
      if (ev.shiftKey) {
        redo();
      } else {
        undo();
      }
    } else if (!typing && mod && ev.key.toLowerCase() === 'y') {
      ev.preventDefault();
      redo();
    } else if (!typing && (ev.key === 'Delete' || ev.key === 'Backspace')) {
      deleteSelected();
    } else if (!typing && ev.key.toLowerCase() === 'r') {
      rotateSelected();
    } else if (!typing && ev.key === 'Escape') {
      selectedId = null;
      refresh();
    }
  });

  window.addEventListener('beforeunload', (ev) => {
    if (history.dirty || nameInput.value.trim() !== boardName) {
      ev.preventDefault();
    }
  });

  // --- Panels ---
  function renderLeft(): void {
    const tab = (value: Layer, label: string): HTMLButtonElement =>
      h('button', {
        type: 'button',
        class: `flex-1 ${layer === value ? btnActive : btn}`,
        onclick: () => {
          layer = value;
          selectedId = null;
          refresh();
        },
      }, label);

    const children: (Node | null)[] = [h('div', { class: 'flex gap-2' }, tab('board', 'Board'), tab('quest', 'Quest'))];

    if (layer === 'board') {
      const brushBtn = (value: BoardBrush, label: string): HTMLButtonElement =>
        h('button', {
          type: 'button',
          class: brush === value ? btnActive : btn,
          onclick: () => {
            brush = value;
            refresh();
          },
        }, label);
      const b = doc().board;
      const counts = new Map<number, number>();
      for (const r of b.regions) {
        counts.set(r, (counts.get(r) ?? 0) + 1);
      }
      const roomRows = b.rooms.map((room) => {
        const nameField = h('input', {
          class: input,
          value: room.name,
          'aria-label': `Name of room ${room.id}`,
          onchange: (e: Event) => {
            const value = (e.target as HTMLInputElement).value;
            if (value.trim()) {
              commit({ ...doc(), board: renameRoom(doc().board, room.id, value) });
            }
          },
        });
        return h(
          'li',
          { class: `flex items-center gap-2 rounded-md p-1 ${activeRoom === room.id && brush === 'room' ? 'bg-amber-500/15' : ''}` },
          h('button', {
            type: 'button',
            class: activeRoom === room.id && brush === 'room' ? btnActive : btn,
            title: 'Paint with this room',
            onclick: () => {
              activeRoom = room.id;
              brush = 'room';
              refresh();
            },
          }, String(room.id)),
          nameField,
          h('input', {
            type: 'color',
            class: 'h-7 w-8 shrink-0 cursor-pointer rounded border border-border/60 bg-surface',
            value: room.color ?? '#1a1d24',
            title: 'Room color',
            'aria-label': `Color of room ${room.id}`,
            onchange: (e: Event) => { commit({ ...doc(), board: setRoomColor(doc().board, room.id, (e.target as HTMLInputElement).value) }); },
          }),
          room.color
            ? h('button', { type: 'button', class: 'text-xs opacity-60 hover:opacity-100', title: 'Clear color', 'aria-label': `Clear color of room ${room.id}`, onclick: () => { commit({ ...doc(), board: setRoomColor(doc().board, room.id, '') }); } }, '×')
            : null,
          h('span', { class: 'w-8 text-right text-xs opacity-60' }, String(counts.get(room.id) ?? 0)),
        );
      });
      children.push(
        h('section', { class: 'space-y-2' },
          h('h2', { class: 'text-sm font-semibold' }, 'Paint'),
          h('div', { class: 'flex flex-wrap gap-2' }, brushBtn('corridor', 'Corridor'), brushBtn('void', 'Solid rock'), brushBtn('room', 'Room'), brushBtn('wall', 'Wall')),
          brush === 'wall'
            ? h('p', { class: 'text-xs opacity-60' }, 'Click an edge to draw a wall, for example between two corridors that touch. Click it again to remove it. Walls between different areas are automatic.')
            : h('div', { class: 'space-y-2' },
              h('label', { class: 'flex items-center gap-2 text-sm' },
                h('input', { type: 'checkbox', checked: rectMode, onchange: (e: Event) => { rectMode = (e.target as HTMLInputElement).checked; refresh(); } }),
                'Rectangle fill (drag corner to corner)'),
              h('p', { class: 'text-xs opacity-60' }, 'Drag on the board to paint. Walls appear automatically wherever areas meet; use Wall for any others.')),
        ),
        h('section', { class: 'space-y-2' },
          h('div', { class: 'flex items-center justify-between' },
            h('h2', { class: 'text-sm font-semibold' }, 'Rooms'),
            h('button', {
              type: 'button',
              class: btn,
              onclick: () => {
                const added = addRoom(doc().board);
                commit({ ...doc(), board: added.board });
                activeRoom = added.roomId;
                brush = 'room';
                refresh();
              },
            }, '+ New room')),
          b.rooms.length ? h('ul', { class: 'space-y-1' }, ...roomRows) : h('p', { class: 'text-xs opacity-60' }, 'No rooms yet.'),
        ),
      );
    } else {
      const toolBtns = QUEST_TOOLS.map((t) =>
        h('button', {
          type: 'button',
          class: `text-left ${questTool === t.kind ? btnActive : btn}`,
          onclick: () => {
            questTool = t.kind;
            refresh();
          },
        }, t.label));
      const hint = QUEST_TOOLS.find((t) => t.kind === questTool)?.hint ?? '';
      const options: (Node | null)[] = [];
      if (questTool === 'furniture') {
        options.push(
          select('Furniture', catalog.furniture.map((f) => [f.id, `${f.name} (${f.width}×${f.height})`]), furnitureType, (v) => { furnitureType = v; }),
          select('Rotation', [['0', '0°'], ['90', '90°'], ['180', '180°'], ['270', '270°']], String(furnitureRotation), (v) => { furnitureRotation = Number(v) as Rotation; }),
        );
      } else if (questTool === 'monster') {
        options.push(select('Monster', catalog.monsters.map((m) => [m.id, monsterOptionLabel(m)]), monsterType, (v) => { monsterType = v; }));
      } else if (questTool === 'trap') {
        options.push(select('Trap', TRAP_KINDS.map((k) => [k, k.replaceAll('_', ' ')]), trapKind, (v) => { trapKind = v; }));
      } else if (questTool === 'blocked') {
        options.push(checkbox('Hides a secret door', blockHidesDoor, (v) => { blockHidesDoor = v; refresh(); }));
      } else if (questTool === 'door') {
        options.push(
          select('Kind', DOOR_KINDS.map(([k, label]) => [k, label]), doorKind, (v) => { doorKind = v as DoorKind; }),
          checkbox('Locked', doorLocked, (v) => { doorLocked = v; refresh(); }),
        );
      }
      children.push(
        h('section', { class: 'space-y-2' },
          h('h2', { class: 'text-sm font-semibold' }, 'Tools'),
          h('div', { class: 'grid grid-cols-2 gap-2' }, ...toolBtns),
          h('p', { class: 'text-xs opacity-60' }, hint),
          ...options,
        ),
      );
      if (!doc().quest) {
        children.push(h('p', { class: 'rounded-md border border-warning/50 bg-warning/10 p-2 text-xs' }, 'Open or create a quest in the right panel to place doors and pieces.'));
      }
    }
    replaceChildren(leftPanel, ...children);
  }

  function checkbox(label: string, checked: boolean, onChange: (v: boolean) => void): HTMLLabelElement {
    return h('label', { class: 'flex items-center gap-2 text-sm' },
      h('input', { type: 'checkbox', checked, onchange: (e: Event) => { onChange((e.target as HTMLInputElement).checked); } }),
      label);
  }

  function select(label: string, opts: string[][], value: string, onChange: (v: string) => void): HTMLLabelElement {
    const el = h('select', { class: input, onchange: (e: Event) => { onChange((e.target as HTMLSelectElement).value); refresh(); } },
      ...opts.map(([v, text]) => h('option', { value: v ?? '', selected: v === value }, text ?? '')));
    return h('label', { class: 'block space-y-1 text-sm' }, h('span', { class: 'opacity-80' }, label), el);
  }

  function renderRight(): void {
    const children: (Node | null)[] = [];

    // Quest management.
    const questSelect = h('select', {
      class: input,
      'aria-label': 'Open quest',
      onchange: (e: Event) => { void openQuest((e.target as HTMLSelectElement).value || null); },
    },
      h('option', { value: '', selected: questId === null }, '— board only —'),
      ...quests.map((q) => h('option', { value: q.id, selected: q.id === questId }, q.name)));
    const newQuestName = h('input', { class: input, placeholder: 'New quest name', maxlength: 120 });
    children.push(
      h('section', { class: 'space-y-2' },
        h('h2', { class: 'text-sm font-semibold' }, 'Quest'),
        questSelect,
        questId
          ? h('label', { class: 'block space-y-1 text-sm' }, h('span', { class: 'opacity-80' }, 'Quest name'),
            h('input', { class: input, value: questName, maxlength: 120, oninput: (e: Event) => { questName = (e.target as HTMLInputElement).value; } }))
          : null,
        h('div', { class: 'flex gap-2' }, newQuestName, h('button', {
          type: 'button',
          class: btn,
          onclick: () => {
            const name = newQuestName.value.trim();
            if (name) {
              void createQuest(name);
            }
          },
        }, 'Create')),
      ),
    );

    // Selection.
    const q = doc().quest;
    if (q && selectedId) {
      children.push(renderSelection(selectedId));
    }

    // Issues.
    children.push(
      h('section', { class: 'space-y-2' },
        h('h2', { class: 'text-sm font-semibold' }, `Checks ${issues.length ? `(${issues.length})` : ''}`),
        issues.length
          ? h('ul', { class: 'space-y-1 text-xs' }, ...issues.map((i) =>
            h('li', { class: 'rounded-md border border-warning/40 bg-warning/10 p-2' },
              i.itemId
                ? h('button', { type: 'button', class: 'text-left underline decoration-dotted', onclick: () => { selectedId = i.itemId ?? null; layer = 'quest'; refresh(); } }, i.message)
                : i.message)))
          : h('p', { class: 'text-xs opacity-60' }, questId ? 'No placement problems found at last save.' : 'Checks run when a quest is saved.'),
        h('p', { class: 'text-xs opacity-50' }, 'Checks are advice only; you can save anything.'),
      ),
      h('section', { class: 'space-y-1 text-xs opacity-60' },
        h('h2', { class: 'text-sm font-semibold opacity-100' }, 'Shortcuts'),
        h('p', {}, 'Ctrl/Cmd+S save · Ctrl/Cmd+Z undo · Shift+Ctrl/Cmd+Z redo'),
        h('p', {}, 'R rotate · Delete remove · Esc deselect'),
      ),
    );
    replaceChildren(rightPanel, ...children);
  }

  function renderSelection(id: string): HTMLElement {
    const q = doc().quest;
    const rows: (Node | null)[] = [];
    const door = q?.doors.find((d) => d.id === id);
    const furniture = q?.furniture.find((f) => f.id === id);
    const monster = q?.monsters.find((m) => m.id === id);
    const trap = q?.traps.find((t) => t.id === id);
    const note = q?.notes.find((n) => n.id === id);
    const blocked = q?.blockedSquares.find((r) => r.id === id);

    if (door && q) {
      rows.push(
        h('p', { class: 'text-sm' }, `${doorKindLabel(door.kind)} at (${door.edge.x}, ${door.edge.y}) ${door.edge.orientation}`),
        select('Kind', DOOR_KINDS.map(([k, label]) => [k, label]), door.kind, (v) => { commit({ ...doc(), quest: updateDoor(q, door.id, { kind: v as DoorKind }) }); }),
        checkbox('Locked', door.locked ?? false, (v) => { commit({ ...doc(), quest: updateDoor(q, door.id, { locked: v }) }); }),
        h('button', { type: 'button', class: btn, onclick: () => { commit({ ...doc(), quest: toggleDoorState(q, door.id) }); } }, `Starts ${door.state} (toggle)`),
      );
    } else if (furniture && q) {
      const def = catalog.furniture.find((f) => f.id === furniture.type);
      rows.push(
        h('p', { class: 'text-sm' }, `${def?.name ?? furniture.type} at (${furniture.x}, ${furniture.y}), ${furniture.rotation}°`),
        h('button', { type: 'button', class: btn, onclick: rotateSelected }, 'Rotate (R)'),
      );
    } else if (monster) {
      const def = catalog.monsters.find((m) => m.id === monster.type);
      rows.push(h('p', { class: 'text-sm' }, `${def?.name ?? monster.type} at (${monster.x}, ${monster.y})`));
      if (def) {
        rows.push(h('p', { class: 'text-xs opacity-70' }, `Body ${def.body} · Mind ${def.mind} · Attack ${def.attack} · Defend ${def.defense} · Move ${def.movement}`));
        if (def.custom) {
          rows.push(h('p', { class: 'text-xs opacity-70' }, `Custom monster, ${def.width ?? 1}×${def.height ?? 1} squares${def.notes ? ` · ${def.notes}` : ''}`));
        }
      }
    } else if (trap) {
      rows.push(h('p', { class: 'text-sm' }, `${trap.kind.replaceAll('_', ' ')} trap at (${trap.x}, ${trap.y})`));
    } else if (note && q) {
      rows.push(
        h('p', { class: 'text-sm' }, `Note ${note.label} at (${note.x}, ${note.y})`),
        h('textarea', {
          id: 'note-text',
          class: `${input} h-28`,
          placeholder: 'What the heroes find here…',
          onchange: (e: Event) => {
            const current = doc().quest;
            if (current) {
              commit({ ...doc(), quest: setNoteText(current, note.id, (e.target as HTMLTextAreaElement).value) });
            }
          },
        }, note.text),
      );
    } else if (blocked && q) {
      rows.push(
        h('p', { class: 'text-sm' }, `Blocked squares ${blocked.w}×${blocked.h} at (${blocked.x}, ${blocked.y})`),
        checkbox('Hides a secret door', blocked.hiddenDoor ?? false, (v) => { commit({ ...doc(), quest: setBlockedHiddenDoor(q, blocked.id, v) }); }),
      );
    } else {
      return h('section', {});
    }

    return h('section', { class: 'space-y-2 rounded-md border border-amber-500/40 p-2' },
      h('div', { class: 'flex items-center justify-between' },
        h('h2', { class: 'text-sm font-semibold' }, 'Selected'),
        h('span', { class: 'font-mono text-xs opacity-50' }, id)),
      ...rows,
      h('button', { type: 'button', class: `${btn} text-danger`, onclick: deleteSelected }, 'Delete'));
  }

  function refresh(): void {
    const b = doc().board;
    if (document.activeElement !== widthInput) {
      widthInput.value = String(b.width);
    }
    if (document.activeElement !== heightInput) {
      heightInput.value = String(b.height);
    }
    undoBtn.disabled = !history.canUndo;
    redoBtn.disabled = !history.canRedo;
    saveBtn.disabled = saving;
    const dirty = history.dirty || nameInput.value.trim() !== boardName;
    statusEl.textContent = dirty && !saving && !status.startsWith('Save failed') ? `${status} · unsaved changes` : status;
    renderLeft();
    renderRight();
    if (focusNoteText) {
      focusNoteText = false;
      document.querySelector<HTMLTextAreaElement>('#note-text')?.focus();
    }
    requestDraw();
  }

  new ResizeObserver(requestDraw).observe(canvas);
  refresh();
}

void main().catch((err: unknown) => {
  const root = document.querySelector('#map-editor');
  if (root) {
    root.textContent = `The map editor failed to load: ${err instanceof Error ? err.message : String(err)}`;
  }
  console.error(err);
});
