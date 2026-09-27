import { describe, expect, test } from 'bun:test';
import { CORRIDOR, VOID } from '../board/model.ts';
import { DOC_VERSION, type BoardDoc, type Catalog, type QuestDoc } from '../maps/types.ts';
import {
  addBlockedSquare,
  addRoom,
  cycleDoor,
  emptyQuest,
  itemsAt,
  moveItem,
  nextId,
  nextNoteLabel,
  paintTiles,
  placeFurniture,
  placeMonster,
  placeNote,
  placeTrap,
  rectTiles,
  removeItem,
  renameRoom,
  resizeBoard,
  rotateItem,
  setNoteText,
  setRoomColor,
  toBoardView,
  toggleExitTile,
  toggleDoorState,
  toggleStartTile,
  toggleWall,
} from './model.ts';

function board(width: number, height: number, fill = VOID): BoardDoc {
  return { version: DOC_VERSION, width, height, regions: new Array<number>(width * height).fill(fill), rooms: [] };
}

const catalog: Catalog = {
  furniture: [
    { id: 'table', name: 'Table', width: 3, height: 2, blocksMovement: true, blocksLineOfSight: false, image: 'assets/table.png' },
    { id: 'chest', name: 'Chest', width: 1, height: 1, blocksMovement: true, blocksLineOfSight: false },
  ],
  monsters: [{ id: 'orc', name: 'Orc', body: 1, mind: 2, attack: 3, defense: 2, movement: 8, image: 'assets/orc.png' }],
  heroes: [],
};

describe('board painting', () => {
  test('paintTiles sets regions and ignores off-board tiles without mutating the input', () => {
    const before = board(3, 2);
    // (1,1) is the bottom-left square (index 0); (3,2) the top-right (index 5).
    const after = paintTiles(before, [{ x: 1, y: 1 }, { x: 3, y: 2 }, { x: 5, y: 5 }, { x: 0, y: 1 }, { x: 1, y: 0 }], CORRIDOR);
    expect(after.regions).toEqual([CORRIDOR, VOID, VOID, VOID, VOID, CORRIDOR]);
    expect(before.regions).toEqual(new Array<number>(6).fill(VOID));
  });

  test('addRoom allocates the next id and a default name', () => {
    const b1 = addRoom(board(2, 2));
    expect(b1.roomId).toBe(1);
    expect(b1.board.rooms).toEqual([{ id: 1, name: 'Room 1' }]);
    const b2 = addRoom(b1.board, 'Vault');
    expect(b2.roomId).toBe(2);
    expect(b2.board.rooms[1]).toEqual({ id: 2, name: 'Vault' });
  });

  test('rooms that lose all their tiles are pruned; rooms still painted are kept', () => {
    let b = addRoom(board(2, 1)).board;
    b = paintTiles(b, [{ x: 1, y: 1 }], 1);
    expect(b.rooms).toHaveLength(1);
    b = paintTiles(b, [{ x: 1, y: 1 }], CORRIDOR);
    expect(b.rooms).toHaveLength(0);
  });

  test('a freshly added room survives until it is painted over', () => {
    const { board: b } = addRoom(board(2, 1));
    // Painting unrelated tiles must not prune the brand-new (still empty) room.
    expect(paintTiles(b, [{ x: 2, y: 1 }], CORRIDOR, 1).rooms).toHaveLength(1);
  });

  test('renameRoom trims and keeps other rooms', () => {
    let b = addRoom(board(1, 1)).board;
    b = addRoom(b).board;
    b = renameRoom(b, 2, '  Throne Room ');
    expect(b.rooms).toEqual([{ id: 1, name: 'Room 1' }, { id: 2, name: 'Throne Room' }]);
  });

  test('setRoomColor sets or clears a room\'s color', () => {
    let b = addRoom(board(1, 1)).board;
    b = setRoomColor(b, 1, '#3a7bd5');
    expect(b.rooms[0]).toEqual({ id: 1, name: 'Room 1', color: '#3a7bd5' });
    expect(setRoomColor(b, 1, '').rooms[0]).toEqual({ id: 1, name: 'Room 1' });
  });

  test('rectTiles covers an inclusive rectangle from either corner', () => {
    expect(rectTiles({ x: 2, y: 1 }, { x: 1, y: 2 })).toEqual([
      { x: 1, y: 1 }, { x: 2, y: 1 },
      { x: 1, y: 2 }, { x: 2, y: 2 },
    ]);
  });
});

describe('toggleWall', () => {
  const between = { x: 2, y: 1, orientation: 'vertical' as const }; // (1,1) | (2,1)

  test('draws a wall between two squares of the same region, and a second click removes it', () => {
    const before = board(3, 2, CORRIDOR);
    const walled = toggleWall(before, between);
    expect(walled.drawnWalls).toEqual([between]);
    expect(before.drawnWalls).toBeUndefined();
    expect(toggleWall(walled, between).drawnWalls).toEqual([]);
  });

  test('ignores the outer edge of the board', () => {
    const b = board(3, 2, CORRIDOR);
    expect(toggleWall(b, { x: 1, y: 1, orientation: 'vertical' })).toBe(b);
    expect(toggleWall(b, { x: 1, y: 3, orientation: 'horizontal' })).toBe(b);
  });

  test('ignores edges that are already walls because the regions differ', () => {
    const b = paintTiles(board(3, 2, CORRIDOR), [{ x: 1, y: 1 }], VOID);
    expect(toggleWall(b, between)).toBe(b);
  });

  test('still removes a drawn wall whose sides were repainted into different regions', () => {
    let b = toggleWall(board(3, 2, CORRIDOR), between);
    b = paintTiles(b, [{ x: 1, y: 1 }], VOID);
    expect(toggleWall(b, between).drawnWalls).toEqual([]);
  });
});

describe('resizeBoard', () => {
  test('drops drawn walls that are no longer inside the board', () => {
    let b = board(3, 3, CORRIDOR);
    b = toggleWall(b, { x: 3, y: 1, orientation: 'vertical' });
    b = toggleWall(b, { x: 1, y: 3, orientation: 'horizontal' });
    b = toggleWall(b, { x: 2, y: 1, orientation: 'vertical' });
    // At 2 wide, x=3 is the outer edge; at 2 tall, the horizontal edge at y=3 is the top edge.
    expect(resizeBoard(b, 2, 2).drawnWalls).toEqual([{ x: 2, y: 1, orientation: 'vertical' }]);
  });

  test('keeps the bottom-left: new columns appear on the right and new rows on top', () => {
    let b = board(2, 2, CORRIDOR);
    b = addRoom(b).board;
    b = paintTiles(b, [{ x: 2, y: 2 }], 1); // top-right square

    const bigger = resizeBoard(b, 3, 3);
    expect(bigger.regions).toEqual([
      CORRIDOR, CORRIDOR, VOID, // y=1
      CORRIDOR, 1, VOID, //        y=2
      VOID, VOID, VOID, //         y=3 (new)
    ]);
    expect(bigger.rooms).toHaveLength(1);
  });

  test('shrinking crops the right-hand columns and the top rows', () => {
    let b = board(2, 2, CORRIDOR);
    b = addRoom(b).board;
    b = paintTiles(b, [{ x: 2, y: 2 }], 1);

    const narrower = resizeBoard(b, 1, 2);
    expect(narrower.regions).toEqual([CORRIDOR, CORRIDOR]);
    expect(narrower.rooms).toHaveLength(0); // room 1 was cropped away

    const shorter = resizeBoard(b, 2, 1);
    expect(shorter.regions).toEqual([CORRIDOR, CORRIDOR]);
    expect(shorter.rooms).toHaveLength(0);
  });

  test('rejects sizes outside 1..200', () => {
    expect(() => resizeBoard(board(2, 2), 0, 2)).toThrow(RangeError);
    expect(() => resizeBoard(board(2, 2), 2, 201)).toThrow(RangeError);
  });
});

describe('quest items', () => {
  test('emptyQuest uses the current (bottom-left) document version', () => {
    expect(emptyQuest().version).toBe(DOC_VERSION);
  });

  test('nextId counts past the highest existing id with that prefix', () => {
    let q = emptyQuest();
    expect(nextId(q, 'door')).toBe('door-1');
    q = { ...q, doors: [{ id: 'door-7', edge: { x: 2, y: 1, orientation: 'vertical' }, kind: 'normal', state: 'closed' }] };
    expect(nextId(q, 'door')).toBe('door-8');
    expect(nextId(q, 'monster')).toBe('monster-1');
  });

  test('cycleDoor goes none -> normal -> secret -> none on the same edge', () => {
    const edge = { x: 2, y: 1, orientation: 'vertical' as const };
    let q = cycleDoor(emptyQuest(), edge);
    expect(q.doors).toEqual([{ id: 'door-1', edge, kind: 'normal', state: 'closed' }]);
    q = cycleDoor(q, edge);
    expect(q.doors[0]?.kind).toBe('secret');
    q = cycleDoor(q, edge);
    expect(q.doors).toEqual([]);
  });

  test('toggleDoorState flips the initial state', () => {
    let q = cycleDoor(emptyQuest(), { x: 2, y: 1, orientation: 'vertical' });
    q = toggleDoorState(q, 'door-1');
    expect(q.doors[0]?.state).toBe('open');
  });

  test('placing pieces assigns ids; rotateItem turns furniture a quarter at a time', () => {
    let q = placeFurniture(emptyQuest(), 'table', { x: 1, y: 1 }, 0);
    q = placeMonster(q, 'orc', { x: 4, y: 4 });
    q = placeTrap(q, 'pit', { x: 5, y: 5 });
    expect(q.furniture).toEqual([{ id: 'furniture-1', type: 'table', x: 1, y: 1, rotation: 0 }]);
    expect(q.monsters).toEqual([{ id: 'monster-1', type: 'orc', x: 4, y: 4 }]);
    expect(q.traps).toEqual([{ id: 'trap-1', kind: 'pit', x: 5, y: 5, state: 'hidden' }]);

    q = rotateItem(q, 'furniture-1');
    q = rotateItem(q, 'furniture-1');
    q = rotateItem(q, 'furniture-1');
    q = rotateItem(q, 'furniture-1');
    expect(q.furniture[0]?.rotation).toBe(0);
    expect(rotateItem(q, 'furniture-1').furniture[0]?.rotation).toBe(90);
  });

  test('notes get sequential letters', () => {
    let q = placeNote(emptyQuest(), { x: 1, y: 1 }, 'first');
    q = placeNote(q, { x: 2, y: 1 }, 'second');
    expect(q.notes.map((n) => n.label)).toEqual(['A', 'B']);
    expect(nextNoteLabel({ ...q, notes: [{ id: 'note-z', label: 'Z', x: 1, y: 1, text: '' }] })).toBe('AA');
  });

  test('setNoteText edits only the chosen note', () => {
    let q = placeNote(emptyQuest(), { x: 1, y: 1 }, '');
    q = placeNote(q, { x: 2, y: 1 }, 'keep');
    q = setNoteText(q, 'note-1', 'The chest holds 84 gold coins.');
    expect(q.notes.map((n) => n.text)).toEqual(['The chest holds 84 gold coins.', 'keep']);
  });

  test('toggleStartTile adds and removes', () => {
    let q = toggleStartTile(emptyQuest(), { x: 2, y: 3 });
    expect(q.startTiles).toEqual([{ x: 2, y: 3 }]);
    q = toggleStartTile(q, { x: 2, y: 3 });
    expect(q.startTiles).toEqual([]);
  });

  test('toggleExitTile adds and removes, separately from start squares', () => {
    let q = toggleStartTile(emptyQuest(), { x: 2, y: 3 });
    q = toggleExitTile(q, { x: 2, y: 3 });
    expect(q.exitTiles).toEqual([{ x: 2, y: 3 }]);
    expect(q.startTiles).toEqual([{ x: 2, y: 3 }]);
    expect(toggleExitTile(q, { x: 2, y: 3 }).exitTiles).toEqual([]);
    const older: QuestDoc = emptyQuest(); // quests saved before exits existed
    delete older.exitTiles;
    expect(toggleExitTile(older, { x: 1, y: 1 }).exitTiles).toEqual([{ x: 1, y: 1 }]);
  });

  test('itemsAt finds items covering a tile, including rotated furniture footprints', () => {
    let q = placeFurniture(emptyQuest(), 'table', { x: 1, y: 1 }, 90); // 2 wide x 3 tall: x 1..2, y 1..3 (up from the anchor)
    q = placeMonster(q, 'orc', { x: 2, y: 3 });
    q = addBlockedSquare(q, { x: 5, y: 1, w: 2, h: 1 });
    expect(itemsAt(q, catalog, { x: 2, y: 3 })).toEqual(['monster-1', 'furniture-1']);
    expect(itemsAt(q, catalog, { x: 3, y: 1 })).toEqual([]);
    expect(itemsAt(q, catalog, { x: 6, y: 1 })).toEqual(['blocked-1']);
  });

  test('moveItem and removeItem work for every layer', () => {
    let q = placeMonster(emptyQuest(), 'orc', { x: 1, y: 1 });
    q = placeNote(q, { x: 2, y: 2 }, 'x');
    q = moveItem(q, 'monster-1', { x: 3, y: 3 });
    expect(q.monsters[0]).toMatchObject({ x: 3, y: 3 });
    q = removeItem(q, 'note-1');
    expect(q.notes).toEqual([]);
    expect(q.monsters).toHaveLength(1);
  });
});

describe('toBoardView', () => {
  test('maps documents into the renderer view using catalog sizes and images', () => {
    const b = board(4, 3, CORRIDOR);
    let q = placeFurniture(emptyQuest(), 'table', { x: 1, y: 1 }, 270);
    q = placeFurniture(q, 'mystery', { x: 4, y: 3 }, 0);
    q = placeMonster(q, 'orc', { x: 2, y: 3 });
    q = cycleDoor(q, { x: 2, y: 2, orientation: 'horizontal' });
    q = placeNote(q, { x: 3, y: 3 }, 'gold');
    q = toggleStartTile(q, { x: 4, y: 1 });

    const view = toBoardView(b, q, catalog);
    expect(view.cols).toBe(4);
    expect(view.rows).toBe(3);
    expect(view.furniture[0]).toEqual({ id: 'furniture-1', type: 'table', at: { x: 1, y: 1 }, width: 3, height: 2, rotation: 270, image: 'assets/table.png' });
    expect(view.furniture[1]).toMatchObject({ type: 'mystery', width: 1, height: 1 });
    expect(view.monsters[0]).toEqual({ id: 'monster-1', type: 'orc', at: { x: 2, y: 3 }, label: 'Orc', image: 'assets/orc.png' });
    expect(view.doors[0]).toEqual({ id: 'door-1', edge: { x: 2, y: 2, orientation: 'horizontal' }, kind: 'normal', state: 'closed' });
    expect(view.notes).toEqual([{ id: 'note-1', label: 'A', at: { x: 3, y: 3 } }]);
    expect(view.startTiles).toEqual([{ x: 4, y: 1 }]);
  });

  test('passes room colors and exit squares to the renderer', () => {
    let b = addRoom(board(2, 1, CORRIDOR)).board;
    b = setRoomColor(b, 1, '#3a7bd5');
    const q = toggleExitTile(emptyQuest(), { x: 2, y: 1 });
    const view = toBoardView(b, q, catalog);
    expect(view.roomColors).toEqual(new Map([[1, '#3a7bd5']]));
    expect(view.exitTiles).toEqual([{ x: 2, y: 1 }]);
    expect(toBoardView(b, null, catalog).exitTiles).toEqual([]);
  });

  test('passes the board\'s drawn walls to the renderer', () => {
    const b = toggleWall(board(2, 1, CORRIDOR), { x: 2, y: 1, orientation: 'vertical' });
    expect(toBoardView(b, null, catalog).drawnWalls).toEqual([{ x: 2, y: 1, orientation: 'vertical' }]);
  });

  test('works without a quest (board layer only)', () => {
    const view = toBoardView(board(2, 2), null, catalog);
    expect(view.doors).toEqual([]);
    expect(view.furniture).toEqual([]);
  });
});

// Keep QuestDoc imported for type-level coverage of the helpers above.
export type _QuestDocCheck = QuestDoc;
