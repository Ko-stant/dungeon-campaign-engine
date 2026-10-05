/**
 * Pure, immutable editing operations on board and quest documents. Every
 * function returns a new document and never mutates its input, so undo/redo
 * is just a stack of snapshots.
 */
import { doorCovers, doorEdges, edgeTiles, footprintTiles, isInteriorEdge, type Edge, type Rotation, type TileCoord } from '../board/geometry.ts';
import { VOID, covers, withShape, onBoard as squareOnBoard, regionAt, tileIndex, type BlockedSquareView, type BoardView, type DoorKind, type FurnitureView, type PieceView, type TrapState, type TrapView } from '../board/model.ts';
import { DOC_VERSION, MAX_TRAP_LABEL, type BoardDoc, type Catalog, type DoorDoc, type QuestDoc, type RectDoc, type Room, type TeleportDoc, type TrapDoc } from '../maps/types.ts';

export const MAX_BOARD_SIZE = 200;

// --- Board layer ---

function onBoard(b: BoardDoc, t: TileCoord): boolean {
  return squareOnBoard(b.width, b.height, t);
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
      regions[tileIndex(b.width, t)] = region;
    }
  }
  return pruneRooms({ ...b, regions }, keepRoomId ?? (region > 0 ? region : undefined));
}

/**
 * Which room a Room-brush stroke paints, or 'new' to start a room first.
 * Without "each shape", it is the active room (a new one if none is active).
 * With it: an empty active room (just made with N) is used as it is; a stroke
 * starting on a room's square extends that room; any other stroke starts a
 * new room.
 */
export function roomForStroke(b: BoardDoc, activeRoom: number | null, eachShape: boolean, start: TileCoord): number | 'new' {
  if (!eachShape) {
    return activeRoom ?? 'new';
  }
  if (activeRoom !== null && !b.regions.includes(activeRoom)) {
    return activeRoom;
  }
  const under = regionAt(b.width, b.height, b.regions, start);
  return under > 0 ? under : 'new';
}

/** Adds a room with the next free id. */
export function addRoom(b: BoardDoc, name?: string): { board: BoardDoc; roomId: number } {
  const roomId = Math.max(0, ...b.rooms.map((r) => r.id), ...b.regions) + 1;
  return { board: { ...b, rooms: [...b.rooms, { id: roomId, name: name ?? `Room ${roomId}` }] }, roomId };
}

/** Sets a room's "#rrggbb" fill, or clears it with an empty string. */
export function setRoomColor(b: BoardDoc, id: number, color: string): BoardDoc {
  return {
    ...b,
    rooms: b.rooms.map((r) => {
      if (r.id !== id) {
        return r;
      }
      const next: Room = { id: r.id, name: r.name };
      if (color) {
        next.color = color;
      }
      return next;
    }),
  };
}

export function renameRoom(b: BoardDoc, id: number, name: string): BoardDoc {
  return { ...b, rooms: b.rooms.map((r) => (r.id === id ? { ...r, name: name.trim() } : r)) };
}

/**
 * Draws a wall on an interior edge, or removes one drawn there before. Edges on
 * the outer boundary, and edges already walled because the regions on either
 * side differ, are left alone.
 */
export function toggleWall(b: BoardDoc, edge: Edge): BoardDoc {
  const drawn = b.drawnWalls ?? [];
  if (drawn.some((e) => sameEdge(e, edge))) {
    return { ...b, drawnWalls: drawn.filter((e) => !sameEdge(e, edge)) };
  }
  if (!isInteriorEdge(edge, b.width, b.height)) {
    return b;
  }
  const [a, c] = edgeTiles(edge);
  if (regionAt(b.width, b.height, b.regions, a) !== regionAt(b.width, b.height, b.regions, c)) {
    return b;
  }
  return { ...b, drawnWalls: [...drawn, { x: edge.x, y: edge.y, orientation: edge.orientation }] };
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

/**
 * Resizes keeping the bottom-left corner, so every square keeps its
 * coordinates: new columns appear on the right and new rows on top, filled with
 * VOID. Cropped rooms are dropped.
 */
export function resizeBoard(b: BoardDoc, width: number, height: number): BoardDoc {
  if (!Number.isInteger(width) || !Number.isInteger(height) || width < 1 || height < 1 || width > MAX_BOARD_SIZE || height > MAX_BOARD_SIZE) {
    throw new RangeError(`board size must be 1..${MAX_BOARD_SIZE} in each dimension, got ${width}x${height}`);
  }
  const regions = new Array<number>(width * height).fill(VOID);
  for (let y = 1; y <= Math.min(height, b.height); y++) {
    for (let x = 1; x <= Math.min(width, b.width); x++) {
      regions[tileIndex(width, { x, y })] = b.regions[tileIndex(b.width, { x, y })] ?? VOID;
    }
  }
  const next: BoardDoc = { ...b, width, height, regions };
  if (b.drawnWalls) {
    next.drawnWalls = b.drawnWalls.filter((e) => isInteriorEdge(e, width, height));
  }
  return pruneRooms(next);
}

// --- Quest layer ---

export function emptyQuest(): QuestDoc {
  return { version: DOC_VERSION, boardChecksum: '', doors: [], blockedSquares: [], furniture: [], monsters: [], traps: [], notes: [], startTiles: [], exitTiles: [], teleports: [] };
}

function allIds(q: QuestDoc): string[] {
  return [
    ...q.doors.map((d) => d.id),
    ...q.blockedSquares.map((r) => r.id),
    ...q.furniture.map((f) => f.id),
    ...q.monsters.map((m) => m.id),
    ...q.traps.map((t) => t.id),
    ...q.notes.map((n) => n.id),
    ...(q.teleports ?? []).map((t) => t.id),
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

/** Sets a door's kind, lock and width; `locked: false` drops the flag (and the key) and span 1 drops the span. */
export function updateDoor(q: QuestDoc, doorId: string, change: { kind?: DoorKind; locked?: boolean; span?: number }): QuestDoc {
  return {
    ...q,
    doors: q.doors.map((d) => {
      if (d.id !== doorId) {
        return d;
      }
      const next: DoorDoc = { id: d.id, edge: d.edge, kind: change.kind ?? d.kind, state: d.state };
      if (change.locked ?? d.locked) {
        next.locked = true;
        if (d.key) {
          next.key = d.key;
        }
      }
      const span = change.span ?? d.span ?? 1;
      if (span > 1) {
        next.span = span;
      }
      return next;
    }),
  };
}

/**
 * Puts a door of the given kind and width (1 or 2 edges) on an edge. An
 * existing door there (either half of a wide one) is switched to that kind,
 * lock and width; clicking again with the same choice removes it. A new wide
 * door replaces any door under its second half.
 */
export function placeDoor(q: QuestDoc, edge: Edge, kind: DoorKind, locked: boolean, span = 1): QuestDoc {
  const existing = q.doors.find((d) => doorCovers(d, edge));
  if (!existing) {
    const door: DoorDoc = { id: nextId(q, 'door'), edge: { x: edge.x, y: edge.y, orientation: edge.orientation }, kind, state: 'closed' };
    if (locked) {
      door.locked = true;
    }
    if (span > 1) {
      door.span = span;
    }
    const covered = doorEdges(door.edge, door.span);
    const others = q.doors.filter((d) => !covered.some((e) => doorCovers(d, e)));
    return { ...q, doors: [...others, door] };
  }
  if (existing.kind === kind && (existing.locked ?? false) === locked && (existing.span ?? 1) === span) {
    return { ...q, doors: q.doors.filter((d) => d !== existing) };
  }
  return updateDoor(q, existing.id, { kind, locked, span });
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

export function placeTrap(q: QuestDoc, kind: string, at: TileCoord, rotation: Rotation = 0): QuestDoc {
  const trap: TrapDoc = { id: nextId(q, 'trap'), kind, x: at.x, y: at.y, state: 'hidden' };
  if (rotation !== 0) {
    trap.rotation = rotation;
  }
  return { ...q, traps: [...q.traps, trap] };
}

/**
 * Trap kinds without catalog artwork, offered after the catalog's traps. A
 * "trigger" is the GM's own marker for setting off other effects.
 */
export const MARKER_TRAP_KINDS = ['trigger', 'chest', 'teleport', 'other'] as const;

/** Picker options for the trap tool: catalog traps (by id), then marker kinds not in the catalog. */
export function trapKindOptions(catalog: Catalog): [string, string][] {
  const options: [string, string][] = catalog.traps.map((t) => [t.id, t.width > 1 || t.height > 1 ? `${t.name} (${t.width}×${t.height})` : t.name]);
  for (const kind of MARKER_TRAP_KINDS) {
    if (!catalog.traps.some((t) => t.id === kind)) {
      options.push([kind, kind.replaceAll('_', ' ')]);
    }
  }
  return options;
}

/** The squares a quest trap covers: its rotated catalog footprint, or its own square. */
export function trapTiles(catalog: Catalog, t: Pick<TrapDoc, 'kind' | 'x' | 'y' | 'rotation'>): TileCoord[] {
  const def = catalog.traps.find((d) => d.id === t.kind);
  return def ? footprintTiles({ x: t.x, y: t.y }, def.width, def.height, t.rotation ?? 0) : [{ x: t.x, y: t.y }];
}

export function setTrapLabel(q: QuestDoc, id: string, label: string): QuestDoc {
  const trimmed = label.trim().slice(0, MAX_TRAP_LABEL).trim();
  return {
    ...q,
    traps: q.traps.map((t) => {
      if (t.id !== id) {
        return t;
      }
      const next: TrapDoc = { ...t };
      delete next.label;
      if (trimmed) {
        next.label = trimmed;
      }
      return next;
    }),
  };
}

/**
 * The renderer's view of a quest trap in a given state, with catalog artwork
 * and footprint. `at` overrides the quest position (a trap moved during play).
 */
export function trapView(catalog: Catalog, t: TrapDoc, state: TrapState, at?: TileCoord): TrapView {
  const view: TrapView = { id: t.id, kind: t.kind, at: at ? { x: at.x, y: at.y } : { x: t.x, y: t.y }, state };
  if (t.label) {
    view.label = t.label;
  }
  const def = catalog.traps.find((d) => d.id === t.kind);
  if (def) {
    view.width = def.width;
    view.height = def.height;
    view.rotation = t.rotation ?? 0;
    if (def.image) {
      view.image = def.image;
    }
  }
  return view;
}

export function addBlockedSquare(q: QuestDoc, rect: { x: number; y: number; w: number; h: number; hiddenDoor?: boolean }): QuestDoc {
  const block: RectDoc = { id: nextId(q, 'blocked'), x: rect.x, y: rect.y, w: rect.w, h: rect.h };
  if (rect.hiddenDoor) {
    block.hiddenDoor = true;
  }
  return { ...q, blockedSquares: [...q.blockedSquares, block] };
}

/** Marks (or unmarks) a blocked square as hiding a secret door. */
export function setBlockedHiddenDoor(q: QuestDoc, id: string, hiddenDoor: boolean): QuestDoc {
  return {
    ...q,
    blockedSquares: q.blockedSquares.map((r) => {
      if (r.id !== id) {
        return r;
      }
      const next: RectDoc = { id: r.id, x: r.x, y: r.y, w: r.w, h: r.h };
      if (hiddenDoor) {
        next.hiddenDoor = true;
      }
      return next;
    }),
  };
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

/** Notes in letter order: A, B, ... Z, AA, AB ... */
export function notesInOrder(q: QuestDoc): QuestDoc['notes'] {
  return [...q.notes].sort((a, b) => labelValue(a.label) - labelValue(b.label));
}

/** Gives notes the letters A, B, C ... in their current letter order, closing any gaps. */
function relabelNotes(notes: QuestDoc['notes']): QuestDoc['notes'] {
  const label = new Map(notesInOrder({ notes } as QuestDoc).map((n, i) => [n.id, valueLabel(i + 1)]));
  return notes.map((n) => ({ ...n, label: label.get(n.id) ?? n.label }));
}

/** A note's first line for lists, cut to max characters. */
export function notePreview(text: string, max: number): string {
  const line = text.trim().split('\n')[0]?.trim() ?? '';
  if (!line) {
    return '(no text)';
  }
  return line.length > max ? `${line.slice(0, max - 1)}…` : line;
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

export function placeTeleport(q: QuestDoc, at: TileCoord): QuestDoc {
  return { ...q, teleports: [...(q.teleports ?? []), { id: nextId(q, 'teleport'), x: at.x, y: at.y }] };
}

/** Sets (or, when blank, clears) a teleport square's pairing label. */
export function setTeleportLabel(q: QuestDoc, id: string, label: string): QuestDoc {
  const trimmed = label.trim();
  return {
    ...q,
    teleports: (q.teleports ?? []).map((t) => {
      if (t.id !== id) {
        return t;
      }
      const next: TeleportDoc = { id: t.id, x: t.x, y: t.y };
      if (trimmed) {
        next.label = trimmed;
      }
      return next;
    }),
  };
}

export function toggleExitTile(q: QuestDoc, at: TileCoord): QuestDoc {
  const exits = q.exitTiles ?? [];
  const has = exits.some((t) => t.x === at.x && t.y === at.y);
  return {
    ...q,
    exitTiles: has ? exits.filter((t) => t.x !== at.x || t.y !== at.y) : [...exits, { x: at.x, y: at.y }],
  };
}

export function rotateItem(q: QuestDoc, id: string): QuestDoc {
  const turn = (r: Rotation | undefined): Rotation => (((r ?? 0) + 90) % 360) as Rotation;
  return {
    ...q,
    furniture: q.furniture.map((f) => (f.id === id ? { ...f, rotation: turn(f.rotation) } : f)),
    traps: q.traps.map((t) => (t.id === id ? { ...t, rotation: turn(t.rotation) } : t)),
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
  ids.push(...q.monsters.filter((m) => {
    const def = catalog.monsters.find((d) => d.id === m.type);
    return covers(m.x, m.y, def?.width, def?.height, t);
  }).map((m) => m.id));
  ids.push(...q.traps.filter((tr) => trapTiles(catalog, tr).some((tt) => hit(tt.x, tt.y))).map((tr) => tr.id));
  ids.push(...q.notes.filter((n) => hit(n.x, n.y)).map((n) => n.id));
  ids.push(...(q.teleports ?? []).filter((tp) => hit(tp.x, tp.y)).map((tp) => tp.id));
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
    teleports: move(q.teleports ?? []),
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
    // Removing a note moves the letters after it up, so they stay A, B, C ...
    notes: q.notes.some((n) => n.id === id) ? relabelNotes(q.notes.filter((n) => n.id !== id)) : q.notes,
    teleports: (q.teleports ?? []).filter((t) => t.id !== id),
  };
}

/** The quest id in a "?quest=" URL query, if it is one of the board's quests. */
export function questFromSearch(search: string, quests: readonly { id: string }[]): string | null {
  const id = new URLSearchParams(search).get('quest');
  return id && quests.some((q) => q.id === id) ? id : null;
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
    return def ? withShape(view, def) : view;
  });
  return {
    cols: b.width,
    rows: b.height,
    regions: b.regions,
    drawnWalls: b.drawnWalls ?? [],
    doors: (q?.doors ?? []).map((d) => ({ id: d.id, edge: d.edge, kind: d.kind, state: d.state, locked: d.locked ?? false, span: d.span })),
    blockedSquares: (q?.blockedSquares ?? []).map((r) => {
      const view: BlockedSquareView = { id: r.id, x: r.x, y: r.y, w: r.w, h: r.h };
      if (r.hiddenDoor) {
        view.hiddenDoor = true;
      }
      return view;
    }),
    furniture,
    monsters,
    heroes: [],
    traps: (q?.traps ?? []).map((t) => trapView(catalog, t, t.state)),
    notes: (q?.notes ?? []).map((n) => ({ id: n.id, label: n.label, at: { x: n.x, y: n.y } })),
    startTiles: (q?.startTiles ?? []).map((t) => ({ x: t.x, y: t.y })),
    exitTiles: (q?.exitTiles ?? []).map((t) => ({ x: t.x, y: t.y })),
    teleports: (q?.teleports ?? []).map((t) => (t.label ? { id: t.id, at: { x: t.x, y: t.y }, label: t.label } : { id: t.id, at: { x: t.x, y: t.y } })),
    roomColors: new Map(b.rooms.filter((r) => r.color).map((r) => [r.id, r.color ?? ''])),
  };
}

export function setNoteText(q: QuestDoc, id: string, text: string): QuestDoc {
  return { ...q, notes: q.notes.map((n) => (n.id === id ? { ...n, text } : n)) };
}
