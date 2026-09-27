/**
 * Pure, immutable editing operations on board and quest documents. Every
 * function returns a new document and never mutates its input, so undo/redo
 * is just a stack of snapshots.
 */
import { footprintTiles, type Edge, type Rotation, type TileCoord } from '../board/geometry.ts';
import { VOID, type BoardView, type FurnitureView, type PieceView } from '../board/model.ts';
import type { BoardDoc, Catalog, QuestDoc } from '../maps/types.ts';

export const MAX_BOARD_SIZE = 200;

// --- Board layer ---

function onBoard(b: BoardDoc, t: TileCoord): boolean {
  return t.x >= 0 && t.y >= 0 && t.x < b.width && t.y < b.height;
}

/** Drops rooms no tile uses any more, except `keep` (e.g. the room being painted). */
function pruneRooms(b: BoardDoc, keep?: number): BoardDoc {
  const used = new Set(b.regions);
  const rooms = b.rooms.filter((r) => used.has(r.id) || r.id === keep);
  return rooms.length === b.rooms.length ? b : { ...b, rooms };
}

/** Paints tiles with a region (VOID, CORRIDOR or a room id). Off-board tiles are ignored. */
export function paintTiles(b: BoardDoc, tiles: readonly TileCoord[], region: number, keepRoomId?: number): BoardDoc {
  const regions = [...b.regions];
  for (const t of tiles) {
    if (onBoard(b, t)) {
      regions[t.y * b.width + t.x] = region;
    }
  }
  return pruneRooms({ ...b, regions }, keepRoomId ?? (region > 0 ? region : undefined));
}

/** Adds a room with the next free id. */
export function addRoom(b: BoardDoc, name?: string): { board: BoardDoc; roomId: number } {
  const roomId = Math.max(0, ...b.rooms.map((r) => r.id), ...b.regions) + 1;
  return { board: { ...b, rooms: [...b.rooms, { id: roomId, name: name ?? `Room ${roomId}` }] }, roomId };
}

export function renameRoom(b: BoardDoc, id: number, name: string): BoardDoc {
  return { ...b, rooms: b.rooms.map((r) => (r.id === id ? { ...r, name: name.trim() } : r)) };
}

/** Every tile in the inclusive rectangle spanned by two corners, row by row. */
export function rectTiles(a: TileCoord, b: TileCoord): TileCoord[] {
  const tiles: TileCoord[] = [];
  for (let y = Math.min(a.y, b.y); y <= Math.max(a.y, b.y); y++) {
    for (let x = Math.min(a.x, b.x); x <= Math.max(a.x, b.x); x++) {
      tiles.push({ x, y });
    }
  }
  return tiles;
}

/** Resizes keeping the top-left corner: new tiles are VOID, cropped rooms are dropped. */
export function resizeBoard(b: BoardDoc, width: number, height: number): BoardDoc {
  if (!Number.isInteger(width) || !Number.isInteger(height) || width < 1 || height < 1 || width > MAX_BOARD_SIZE || height > MAX_BOARD_SIZE) {
    throw new RangeError(`board size must be 1..${MAX_BOARD_SIZE} in each dimension, got ${width}x${height}`);
  }
  const regions = new Array<number>(width * height).fill(VOID);
  for (let y = 0; y < Math.min(height, b.height); y++) {
    for (let x = 0; x < Math.min(width, b.width); x++) {
      regions[y * width + x] = b.regions[y * b.width + x] ?? VOID;
    }
  }
  return pruneRooms({ ...b, width, height, regions });
}

// --- Quest layer ---

export function emptyQuest(): QuestDoc {
  return { version: 1, boardChecksum: '', doors: [], blockedSquares: [], furniture: [], monsters: [], traps: [], notes: [], startTiles: [] };
}

function allIds(q: QuestDoc): string[] {
  return [
    ...q.doors.map((d) => d.id),
    ...q.blockedSquares.map((r) => r.id),
    ...q.furniture.map((f) => f.id),
    ...q.monsters.map((m) => m.id),
    ...q.traps.map((t) => t.id),
    ...q.notes.map((n) => n.id),
  ];
}

/** `${prefix}-${n}` with n one past the highest existing number for that prefix. */
export function nextId(q: QuestDoc, prefix: string): string {
  let max = 0;
  for (const id of allIds(q)) {
    if (id.startsWith(`${prefix}-`)) {
      const n = Number(id.slice(prefix.length + 1));
      if (Number.isInteger(n) && n > max) {
        max = n;
      }
    }
  }
  return `${prefix}-${max + 1}`;
}

function sameEdge(a: Edge, b: Edge): boolean {
  return a.x === b.x && a.y === b.y && a.orientation === b.orientation;
}

/** Cycles the door on an edge: none -> normal -> secret -> none. */
export function cycleDoor(q: QuestDoc, edge: Edge): QuestDoc {
  const existing = q.doors.find((d) => sameEdge(d.edge, edge));
  if (!existing) {
    return { ...q, doors: [...q.doors, { id: nextId(q, 'door'), edge, kind: 'normal', state: 'closed' }] };
  }
  if (existing.kind === 'normal') {
    return { ...q, doors: q.doors.map((d) => (d === existing ? { ...d, kind: 'secret' } : d)) };
  }
  return { ...q, doors: q.doors.filter((d) => d !== existing) };
}

export function toggleDoorState(q: QuestDoc, doorId: string): QuestDoc {
  return { ...q, doors: q.doors.map((d) => (d.id === doorId ? { ...d, state: d.state === 'open' ? 'closed' : 'open' } : d)) };
}

export function placeFurniture(q: QuestDoc, type: string, at: TileCoord, rotation: Rotation): QuestDoc {
  return { ...q, furniture: [...q.furniture, { id: nextId(q, 'furniture'), type, x: at.x, y: at.y, rotation }] };
}

export function placeMonster(q: QuestDoc, type: string, at: TileCoord): QuestDoc {
  return { ...q, monsters: [...q.monsters, { id: nextId(q, 'monster'), type, x: at.x, y: at.y }] };
}

export function placeTrap(q: QuestDoc, kind: string, at: TileCoord): QuestDoc {
  return { ...q, traps: [...q.traps, { id: nextId(q, 'trap'), kind, x: at.x, y: at.y, state: 'hidden' }] };
}

export function addBlockedSquare(q: QuestDoc, rect: { x: number; y: number; w: number; h: number }): QuestDoc {
  return { ...q, blockedSquares: [...q.blockedSquares, { id: nextId(q, 'blocked'), ...rect }] };
}

function labelValue(label: string): number {
  let v = 0;
  for (const ch of label.toUpperCase()) {
    const code = ch.charCodeAt(0) - 64;
    if (code < 1 || code > 26) {
      return 0;
    }
    v = v * 26 + code;
  }
  return v;
}

function valueLabel(v: number): string {
  let label = '';
  for (let n = v; n > 0; n = Math.floor((n - 1) / 26)) {
    label = String.fromCharCode(65 + ((n - 1) % 26)) + label;
  }
  return label;
}

/** The letter after the highest existing note label: A, B, ... Z, AA, AB ... */
export function nextNoteLabel(q: QuestDoc): string {
  return valueLabel(Math.max(0, ...q.notes.map((n) => labelValue(n.label))) + 1);
}

export function placeNote(q: QuestDoc, at: TileCoord, text: string): QuestDoc {
  return { ...q, notes: [...q.notes, { id: nextId(q, 'note'), label: nextNoteLabel(q), x: at.x, y: at.y, text }] };
}

export function toggleStartTile(q: QuestDoc, at: TileCoord): QuestDoc {
  const has = q.startTiles.some((t) => t.x === at.x && t.y === at.y);
  return {
    ...q,
    startTiles: has ? q.startTiles.filter((t) => t.x !== at.x || t.y !== at.y) : [...q.startTiles, { x: at.x, y: at.y }],
  };
}

export function rotateItem(q: QuestDoc, id: string): QuestDoc {
  return {
    ...q,
    furniture: q.furniture.map((f) => (f.id === id ? { ...f, rotation: ((f.rotation + 90) % 360) as Rotation } : f)),
  };
}

function furnitureSize(catalog: Catalog, type: string): { width: number; height: number } {
  const def = catalog.furniture.find((f) => f.id === type);
  return def ? { width: def.width, height: def.height } : { width: 1, height: 1 };
}

/** Ids of the items covering a tile, topmost first (pieces, then furniture, then blocked squares). */
export function itemsAt(q: QuestDoc, catalog: Catalog, t: TileCoord): string[] {
  const hit = (x: number, y: number): boolean => x === t.x && y === t.y;
  const ids: string[] = [];
  ids.push(...q.monsters.filter((m) => hit(m.x, m.y)).map((m) => m.id));
  ids.push(...q.traps.filter((tr) => hit(tr.x, tr.y)).map((tr) => tr.id));
  ids.push(...q.notes.filter((n) => hit(n.x, n.y)).map((n) => n.id));
  for (const f of q.furniture) {
    const size = furnitureSize(catalog, f.type);
    if (footprintTiles({ x: f.x, y: f.y }, size.width, size.height, f.rotation).some((ft) => hit(ft.x, ft.y))) {
      ids.push(f.id);
    }
  }
  for (const r of q.blockedSquares) {
    if (t.x >= r.x && t.x < r.x + r.w && t.y >= r.y && t.y < r.y + r.h) {
      ids.push(r.id);
    }
  }
  return ids;
}

export function moveItem(q: QuestDoc, id: string, to: TileCoord): QuestDoc {
  const move = <T extends { id: string; x: number; y: number }>(items: T[]): T[] =>
    items.map((it) => (it.id === id ? { ...it, x: to.x, y: to.y } : it));
  return {
    ...q,
    blockedSquares: move(q.blockedSquares),
    furniture: move(q.furniture),
    monsters: move(q.monsters),
    traps: move(q.traps),
    notes: move(q.notes),
  };
}

export function removeItem(q: QuestDoc, id: string): QuestDoc {
  return {
    ...q,
    doors: q.doors.filter((d) => d.id !== id),
    blockedSquares: q.blockedSquares.filter((r) => r.id !== id),
    furniture: q.furniture.filter((f) => f.id !== id),
    monsters: q.monsters.filter((m) => m.id !== id),
    traps: q.traps.filter((t) => t.id !== id),
    notes: q.notes.filter((n) => n.id !== id),
  };
}

// --- Rendering ---

/** Builds the renderer's view of a board and (optionally) a quest. */
export function toBoardView(b: BoardDoc, q: QuestDoc | null, catalog: Catalog): BoardView {
  const furniture: FurnitureView[] = (q?.furniture ?? []).map((f) => {
    const def = catalog.furniture.find((d) => d.id === f.type);
    const view: FurnitureView = { id: f.id, type: f.type, at: { x: f.x, y: f.y }, width: def?.width ?? 1, height: def?.height ?? 1, rotation: f.rotation };
    if (def?.image) {
      view.image = def.image;
    }
    return view;
  });
  const monsters: PieceView[] = (q?.monsters ?? []).map((m) => {
    const def = catalog.monsters.find((d) => d.id === m.type);
    const view: PieceView = { id: m.id, type: m.type, at: { x: m.x, y: m.y }, label: def?.name ?? m.type };
    if (def?.image) {
      view.image = def.image;
    }
    return view;
  });
  return {
    cols: b.width,
    rows: b.height,
    regions: b.regions,
    doors: (q?.doors ?? []).map((d) => ({ id: d.id, edge: d.edge, kind: d.kind, state: d.state })),
    blockedSquares: (q?.blockedSquares ?? []).map((r) => ({ x: r.x, y: r.y, w: r.w, h: r.h })),
    furniture,
    monsters,
    heroes: [],
    traps: (q?.traps ?? []).map((t) => ({ id: t.id, kind: t.kind, at: { x: t.x, y: t.y }, state: t.state })),
    notes: (q?.notes ?? []).map((n) => ({ id: n.id, label: n.label, at: { x: n.x, y: n.y } })),
    startTiles: (q?.startTiles ?? []).map((t) => ({ x: t.x, y: t.y })),
  };
}

export function setNoteText(q: QuestDoc, id: string, text: string): QuestDoc {
  return { ...q, notes: q.notes.map((n) => (n.id === id ? { ...n, text } : n)) };
}
