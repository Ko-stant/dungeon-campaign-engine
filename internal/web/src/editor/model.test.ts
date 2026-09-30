import { describe, expect, test } from 'bun:test';
import { CORRIDOR, VOID } from '../board/model.ts';
import { DOC_VERSION, type BoardDoc, type Catalog, type QuestDoc } from '../maps/types.ts';
import {
  addBlockedSquare,
  addRoom,
  placeDoor,
  emptyQuest,
  itemsAt,
  moveItem,
  nextId,
  nextNoteLabel,
  roomForStroke,
  notePreview,
  notesInOrder,
  paintTiles,
  placeFurniture,
  placeMonster,
  placeNote,
  placeTeleport,
  questFromSearch,
  placeTrap,
  rectTiles,
  removeItem,
  renameRoom,
  resizeBoard,
  rotateItem,
  setBlockedHiddenDoor,
  setNoteText,
  setRoomColor,
  setTeleportLabel,
  setTrapLabel,
  toBoardView,
  toggleExitTile,
  toggleDoorState,
  toggleStartTile,
  toggleWall,
  trapKindOptions,
  updateDoor,
} from './model.ts';

function board(width: number, height: number, fill = VOID): BoardDoc {
  return { version: DOC_VERSION, width, height, regions: new Array<number>(width * height).fill(fill), rooms: [] };
}

const catalog: Catalog = {
  furniture: [
    { id: 'table', name: 'Table', width: 3, height: 2, blocksMovement: true, blocksLineOfSight: false, image: 'assets/table.png' },
    { id: 'chest', name: 'Chest', width: 1, height: 1, blocksMovement: true, blocksLineOfSight: false },
  ],
  monsters: [
    { id: 'orc', name: 'Orc', body: 1, mind: 2, attack: 3, defense: 2, movement: 8, image: 'assets/orc.png' },
    { id: 'custom-ogre', name: 'Cave Ogre', body: 6, mind: 1, attack: 4, defense: 3, movement: 6, width: 2, height: 2, color: '#aa3300', custom: true },
  ],
  heroes: [],
  traps: [
    { id: 'long_pit', name: 'Long Pit Trap', width: 1, height: 2, image: 'assets/long_pit.png' },
    { id: 'pit', name: 'Pit Trap', width: 1, height: 1 },
  ],
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

  test('placeDoor adds a door of the chosen kind, switches an existing one, and removes an identical one', () => {
    const edge = { x: 2, y: 1, orientation: 'vertical' as const };
    let q = placeDoor(emptyQuest(), edge, 'normal', false);
    expect(q.doors).toEqual([{ id: 'door-1', edge, kind: 'normal', state: 'closed' }]);
    q = placeDoor(q, edge, 'secret', false);
    expect(q.doors).toEqual([{ id: 'door-1', edge, kind: 'secret', state: 'closed' }]);
    q = placeDoor(q, edge, 'gate', true);
    expect(q.doors).toEqual([{ id: 'door-1', edge, kind: 'gate', state: 'closed', locked: true }]);
    q = placeDoor(q, edge, 'gate', false);
    expect(q.doors).toEqual([{ id: 'door-1', edge, kind: 'gate', state: 'closed' }]);
    q = placeDoor(q, edge, 'gate', false);
    expect(q.doors).toEqual([]);
  });

  test('placeDoor places two-wide doors; clicking either half switches or removes it', () => {
    const edge = { x: 2, y: 1, orientation: 'vertical' as const };
    const upper = { x: 2, y: 2, orientation: 'vertical' as const };
    let q = placeDoor(emptyQuest(), edge, 'gate', false, 2);
    expect(q.doors).toEqual([{ id: 'door-1', edge, kind: 'gate', state: 'closed', span: 2 }]);
    // Same choice on the upper half removes the whole door.
    expect(placeDoor(q, upper, 'gate', false, 2).doors).toEqual([]);
    // A different width on either half resizes it; width 1 drops the span.
    q = placeDoor(q, upper, 'gate', false, 1);
    expect(q.doors).toEqual([{ id: 'door-1', edge, kind: 'gate', state: 'closed' }]);
  });

  test('a two-wide door replaces doors under its second half', () => {
    let q = placeDoor(emptyQuest(), { x: 2, y: 2, orientation: 'vertical' }, 'normal', false);
    q = placeDoor(q, { x: 2, y: 1, orientation: 'vertical' }, 'exit', false, 2);
    expect(q.doors).toEqual([{ id: 'door-2', edge: { x: 2, y: 1, orientation: 'vertical' }, kind: 'exit', state: 'closed', span: 2 }]);
  });

  test('updateDoor changes kind and lock of one door', () => {
    let q = placeDoor(emptyQuest(), { x: 2, y: 1, orientation: 'vertical' }, 'normal', false);
    q = updateDoor(q, 'door-1', { kind: 'gate', locked: true });
    expect(q.doors[0]).toMatchObject({ kind: 'gate', locked: true });
    expect(updateDoor(q, 'door-1', { locked: false }).doors[0]).toEqual({ id: 'door-1', edge: { x: 2, y: 1, orientation: 'vertical' }, kind: 'gate', state: 'closed' });
  });

  test('toggleDoorState flips the initial state', () => {
    let q = placeDoor(emptyQuest(), { x: 2, y: 1, orientation: 'vertical' }, 'normal', false);
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

  test('traps can be placed turned and rotate a quarter at a time', () => {
    let q = placeTrap(emptyQuest(), 'long_pit', { x: 2, y: 2 }, 90);
    expect(q.traps).toEqual([{ id: 'trap-1', kind: 'long_pit', x: 2, y: 2, state: 'hidden', rotation: 90 }]);
    q = placeTrap(q, 'pit', { x: 5, y: 5 });
    q = rotateItem(q, 'trap-2'); // saved without a rotation: counts as 0
    expect(q.traps[1]?.rotation).toBe(90);
    q = rotateItem(rotateItem(rotateItem(q, 'trap-1'), 'trap-1'), 'trap-1');
    expect(q.traps[0]?.rotation).toBe(0);
  });

  test('notes get sequential letters', () => {
    let q = placeNote(emptyQuest(), { x: 1, y: 1 }, 'first');
    q = placeNote(q, { x: 2, y: 1 }, 'second');
    expect(q.notes.map((n) => n.label)).toEqual(['A', 'B']);
    expect(nextNoteLabel({ ...q, notes: [{ id: 'note-z', label: 'Z', x: 1, y: 1, text: '' }] })).toBe('AA');
  });

  test('removing a note relabels the notes after it so the letters stay in sequence', () => {
    let q = emptyQuest();
    for (const text of ['a', 'b', 'c', 'd']) {
      q = placeNote(q, { x: 1, y: 1 }, text);
    }
    const removed = removeItem(q, 'note-2');
    expect(removed.notes.map((n) => [n.id, n.label, n.text])).toEqual([['note-1', 'A', 'a'], ['note-3', 'B', 'c'], ['note-4', 'C', 'd']]);
    expect(nextNoteLabel(removed)).toBe('D');
    // Removing the last note changes no other label.
    expect(removeItem(q, 'note-4').notes.map((n) => n.label)).toEqual(['A', 'B', 'C']);
    // Gaps left by older documents close up too, keeping the letters' order.
    const gappy: QuestDoc = { ...emptyQuest(), notes: [
      { id: 'n-f', label: 'F', x: 1, y: 1, text: '' },
      { id: 'n-a', label: 'A', x: 1, y: 1, text: '' },
      { id: 'n-c', label: 'C', x: 1, y: 1, text: '' },
    ] };
    expect(notesInOrder(removeItem(gappy, 'n-c')).map((n) => [n.id, n.label])).toEqual([['n-a', 'A'], ['n-f', 'B']]);
    // Removing something else leaves the notes alone.
    expect(removeItem(gappy, 'monster-9').notes).toBe(gappy.notes);
  });

  test('notesInOrder lists notes by letter, AA after Z', () => {
    const q: QuestDoc = { ...emptyQuest(), notes: [
      { id: 'n1', label: 'AA', x: 1, y: 1, text: '' },
      { id: 'n2', label: 'B', x: 1, y: 1, text: '' },
      { id: 'n3', label: 'Z', x: 1, y: 1, text: '' },
      { id: 'n4', label: 'A', x: 1, y: 1, text: '' },
    ] };
    expect(notesInOrder(q).map((n) => n.label)).toEqual(['A', 'B', 'Z', 'AA']);
  });

  test('notePreview shows the first line, shortened', () => {
    expect(notePreview('Q1-N1 Tomas Reed: remains under the fall\nmore detail', 20)).toBe('Q1-N1 Tomas Reed: r…');
    expect(notePreview('  The gate key  ', 20)).toBe('The gate key');
    expect(notePreview('   ', 20)).toBe('(no text)');
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

  test('itemsAt covers a catalog trap footprint; marker traps cover one square', () => {
    let q = placeTrap(emptyQuest(), 'long_pit', { x: 2, y: 2 }, 0); // 1x2: (2,2) and (2,3)
    q = placeTrap(q, 'long_pit', { x: 5, y: 5 }, 90); // turned 2x1: (5,5) and (6,5)
    q = placeTrap(q, 'chest', { x: 8, y: 1 });
    expect(itemsAt(q, catalog, { x: 2, y: 3 })).toEqual(['trap-1']);
    expect(itemsAt(q, catalog, { x: 3, y: 2 })).toEqual([]);
    expect(itemsAt(q, catalog, { x: 6, y: 5 })).toEqual(['trap-2']);
    expect(itemsAt(q, catalog, { x: 5, y: 6 })).toEqual([]);
    expect(itemsAt(q, catalog, { x: 8, y: 1 })).toEqual(['trap-3']);
    expect(itemsAt(q, catalog, { x: 8, y: 2 })).toEqual([]);
  });

  test('trap kind options list the catalog, then marker kinds without artwork', () => {
    expect(trapKindOptions(catalog)).toEqual([
      ['long_pit', 'Long Pit Trap (1×2)'],
      ['pit', 'Pit Trap'],
      ['trigger', 'trigger'],
      ['chest', 'chest'],
      ['teleport', 'teleport'],
      ['other', 'other'],
    ]);
    const withChest: Catalog = { ...catalog, traps: [{ id: 'chest', name: 'Chest Trap', width: 1, height: 1 }] };
    expect(trapKindOptions(withChest)).toEqual([['chest', 'Chest Trap'], ['trigger', 'trigger'], ['teleport', 'teleport'], ['other', 'other']]);
  });

  test('traps take a short trimmed label, shown in the view; an empty one clears it', () => {
    let q = placeTrap(emptyQuest(), 'trigger', { x: 3, y: 3 });
    q = setTrapLabel(q, 'trap-1', '  1 ');
    expect(q.traps[0]).toEqual({ id: 'trap-1', kind: 'trigger', x: 3, y: 3, state: 'hidden', label: '1' });
    expect(toBoardView(board(4, 4), q, catalog).traps[0]).toEqual({ id: 'trap-1', kind: 'trigger', at: { x: 3, y: 3 }, state: 'hidden', label: '1' });
    expect(setTrapLabel(q, 'trap-1', ' ').traps[0]).toEqual({ id: 'trap-1', kind: 'trigger', x: 3, y: 3, state: 'hidden' });
    expect(setTrapLabel(q, 'trap-1', 'far too long a label').traps[0]?.label).toBe('far too');
  });

  test('a blocked square can be marked as hiding a secret door', () => {
    let q = addBlockedSquare(emptyQuest(), { x: 4, y: 15, w: 1, h: 1, hiddenDoor: true });
    expect(q.blockedSquares[0]).toEqual({ id: 'blocked-1', x: 4, y: 15, w: 1, h: 1, hiddenDoor: true });
    q = setBlockedHiddenDoor(q, 'blocked-1', false);
    expect(q.blockedSquares[0]).toEqual({ id: 'blocked-1', x: 4, y: 15, w: 1, h: 1 });
    expect(toBoardView(board(5, 20), setBlockedHiddenDoor(q, 'blocked-1', true), catalog).blockedSquares).toEqual([{ id: 'blocked-1', x: 4, y: 15, w: 1, h: 1, hiddenDoor: true }]);
  });

  test('teleport squares can be placed, labelled, found, moved and removed', () => {
    let q = placeTeleport(emptyQuest(), { x: 2, y: 2 });
    q = placeTeleport(q, { x: 5, y: 5 });
    expect(q.teleports).toEqual([{ id: 'teleport-1', x: 2, y: 2 }, { id: 'teleport-2', x: 5, y: 5 }]);
    q = setTeleportLabel(q, 'teleport-1', ' 1 ');
    expect(q.teleports?.[0]).toEqual({ id: 'teleport-1', x: 2, y: 2, label: '1' });
    expect(setTeleportLabel(q, 'teleport-1', '').teleports?.[0]).toEqual({ id: 'teleport-1', x: 2, y: 2 });
    expect(itemsAt(q, catalog, { x: 2, y: 2 })).toEqual(['teleport-1']);
    q = moveItem(q, 'teleport-2', { x: 6, y: 6 });
    expect(q.teleports?.[1]).toMatchObject({ x: 6, y: 6 });
    expect(removeItem(q, 'teleport-1').teleports).toEqual([{ id: 'teleport-2', x: 6, y: 6 }]);
    expect(toBoardView(board(8, 8), q, catalog).teleports).toEqual([{ id: 'teleport-1', at: { x: 2, y: 2 }, label: '1' }, { id: 'teleport-2', at: { x: 6, y: 6 } }]);
    const older: QuestDoc = emptyQuest(); // quests saved before teleports existed
    delete older.teleports;
    expect(placeTeleport(older, { x: 1, y: 1 }).teleports).toHaveLength(1);
    expect(toBoardView(board(2, 2), older, catalog).teleports).toEqual([]);
  });

  test('custom monsters cover their whole footprint', () => {
    const q = placeMonster(emptyQuest(), 'custom-ogre', { x: 3, y: 3 });
    expect(itemsAt(q, catalog, { x: 4, y: 4 })).toEqual(['monster-1']);
    expect(itemsAt(q, catalog, { x: 5, y: 4 })).toEqual([]);
    expect(toBoardView(board(6, 6), q, catalog).monsters[0]).toEqual({ id: 'monster-1', type: 'custom-ogre', at: { x: 3, y: 3 }, label: 'Cave Ogre', width: 2, height: 2, color: '#aa3300' });
  });

  test('catalog traps carry their artwork, size and rotation into the view; markers do not', () => {
    let q = placeTrap(emptyQuest(), 'long_pit', { x: 2, y: 2 }, 90);
    q = placeTrap(q, 'pit', { x: 4, y: 4 });
    q = placeTrap(q, 'chest', { x: 5, y: 5 });
    expect(toBoardView(board(6, 6), q, catalog).traps).toEqual([
      { id: 'trap-1', kind: 'long_pit', at: { x: 2, y: 2 }, state: 'hidden', width: 1, height: 2, rotation: 90, image: 'assets/long_pit.png' },
      { id: 'trap-2', kind: 'pit', at: { x: 4, y: 4 }, state: 'hidden', width: 1, height: 1, rotation: 0 },
      { id: 'trap-3', kind: 'chest', at: { x: 5, y: 5 }, state: 'hidden' },
    ]);
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

describe('questFromSearch', () => {
  const quests = [{ id: 'q1' }, { id: 'q2' }];
  test('opens the quest named in ?quest= when the board has it', () => {
    expect(questFromSearch('?quest=q2', quests)).toBe('q2');
    expect(questFromSearch('?quest=zzz', quests)).toBeNull();
    expect(questFromSearch('', quests)).toBeNull();
  });
});

describe('toBoardView', () => {
  test('maps documents into the renderer view using catalog sizes and images', () => {
    const b = board(4, 3, CORRIDOR);
    let q = placeFurniture(emptyQuest(), 'table', { x: 1, y: 1 }, 270);
    q = placeFurniture(q, 'mystery', { x: 4, y: 3 }, 0);
    q = placeMonster(q, 'orc', { x: 2, y: 3 });
    q = placeDoor(q, { x: 2, y: 2, orientation: 'horizontal' }, 'normal', false);
    q = placeNote(q, { x: 3, y: 3 }, 'gold');
    q = toggleStartTile(q, { x: 4, y: 1 });

    const view = toBoardView(b, q, catalog);
    expect(view.cols).toBe(4);
    expect(view.rows).toBe(3);
    expect(view.furniture[0]).toEqual({ id: 'furniture-1', type: 'table', at: { x: 1, y: 1 }, width: 3, height: 2, rotation: 270, image: 'assets/table.png' });
    expect(view.furniture[1]).toMatchObject({ type: 'mystery', width: 1, height: 1 });
    expect(view.monsters[0]).toEqual({ id: 'monster-1', type: 'orc', at: { x: 2, y: 3 }, label: 'Orc', image: 'assets/orc.png' });
    expect(view.doors[0]).toEqual({ id: 'door-1', edge: { x: 2, y: 2, orientation: 'horizontal' }, kind: 'normal', state: 'closed', locked: false });
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

describe('roomForStroke', () => {
  // Squares (1,1) (2,1) (3,1): room 1, corridor, corridor. Room 2 exists but is empty.
  const painted: BoardDoc = { ...board(3, 1, CORRIDOR), regions: [1, 0, 0], rooms: [{ id: 1, name: 'Hall' }, { id: 2, name: 'Empty' }] };
  const onRoom = { x: 1, y: 1 };
  const onCorridor = { x: 2, y: 1 };

  test('without "each shape", strokes paint the active room, or start one when none is active', () => {
    expect(roomForStroke(painted, 1, false, onCorridor)).toBe(1);
    expect(roomForStroke(painted, null, false, onCorridor)).toBe('new');
  });

  test('with "each shape", a stroke starting on a room extends that room', () => {
    expect(roomForStroke(painted, null, true, onRoom)).toBe(1);
    expect(roomForStroke(painted, 1, true, onRoom)).toBe(1);
  });

  test('with "each shape", a stroke starting on corridor or rock starts a new room', () => {
    expect(roomForStroke(painted, 1, true, onCorridor)).toBe('new');
    expect(roomForStroke(painted, null, true, onCorridor)).toBe('new');
  });

  test('an empty active room (just made with N) is painted wherever the stroke starts', () => {
    expect(roomForStroke(painted, 2, true, onRoom)).toBe(2);
    expect(roomForStroke(painted, 2, true, onCorridor)).toBe(2);
  });
});
