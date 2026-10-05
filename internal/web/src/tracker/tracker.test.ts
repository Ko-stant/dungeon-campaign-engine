import { describe, expect, test } from 'bun:test';
import { DOC_VERSION, type Catalog } from '../maps/types.ts';
import { formatEvent, searchEvents } from './format.ts';
import { clickCommand, corridorPath, doorAt, hotkey, paintPending, pieceAt, revealPathCommand, revealSquaresCommand, type Mode } from './interaction.ts';
import type { SessionState } from './types.ts';
import { trackerView } from './view.ts';
import { shownToPlayers } from './visibility.ts';

const catalog: Catalog = {
  furniture: [{ id: 'table', name: 'Table', width: 2, height: 1, blocksMovement: true, blocksLineOfSight: false, image: 'assets/table.png' }],
  monsters: [{ id: 'orc', name: 'Orc', body: 1, mind: 2, attack: 3, defense: 2, movement: 8, image: 'assets/orc.png' }],
  heroes: [{ id: 'elf', name: 'Elf', body: 6, mind: 4, attack: 2, defense: 2, movementDice: 2 }],
  traps: [
    { id: 'long_pit', name: 'Long Pit Trap', width: 1, height: 2, image: 'assets/long_pit.png' },
    { id: 'boulder', name: 'Boulder', width: 1, height: 1, image: 'assets/boulder.png', movable: true },
  ],
};

/** state() plus a boulder on (2,1), which the GM can move. */
function withBoulder(): SessionState {
  const s = state();
  s.quest.traps.push({ id: 'trap-3', kind: 'boulder', x: 2, y: 1, state: 'hidden' });
  s.traps.push({ id: 'trap-3', state: 'triggered' });
  return s;
}

/** state() plus a 1x2 long pit standing on (4,1) and (4,2), under note A. */
function withLongPit(): SessionState {
  const s = state();
  s.quest.traps.push({ id: 'trap-2', kind: 'long_pit', x: 4, y: 1, state: 'hidden' });
  s.traps.push({ id: 'trap-2', state: 'revealed' });
  return s;
}

/**
 * 4x2 corridor board (squares (1,1) bottom-left to (4,2) top-right): one door,
 * one secret door, a trap, a note, two monsters, two heroes.
 */
function state(): SessionState {
  return {
    version: 2,
    board: { version: DOC_VERSION, width: 4, height: 2, regions: [0, 0, 0, 0, 0, 0, 0, 0], rooms: [] },
    quest: {
      version: DOC_VERSION,
      boardChecksum: '',
      doors: [
        { id: 'door-1', edge: { x: 3, y: 2, orientation: 'vertical' }, kind: 'normal', state: 'closed' },
        { id: 'door-2', edge: { x: 4, y: 1, orientation: 'vertical' }, kind: 'secret', state: 'closed' },
      ],
      blockedSquares: [
        { id: 'blocked-1', x: 1, y: 2, w: 1, h: 1, hiddenDoor: true },
        { id: 'blocked-2', x: 3, y: 2, w: 1, h: 1 },
      ],
      furniture: [{ id: 'furniture-1', type: 'table', x: 1, y: 1, rotation: 0 }],
      monsters: [],
      traps: [{ id: 'trap-1', kind: 'pit', x: 2, y: 2, state: 'hidden' }],
      notes: [{ id: 'note-A', label: 'A', x: 4, y: 2, text: 'gold' }],
      startTiles: [],
    },
    questName: 'The Trial',
    round: 1,
    heroes: [
      { id: 'hero-1', name: 'Faelyn', class: 'elf', x: 1, y: 2, placed: true, body: 6, maxBody: 6, mind: 4, maxMind: 4, status: 'active' },
      { id: 'hero-2', name: 'Bram', class: 'elf', x: 1, y: 2, placed: false, body: 6, maxBody: 6, mind: 4, maxMind: 4, status: 'active' },
    ],
    monsters: [
      { id: 'monster-1', type: 'orc', name: 'Orc', x: 3, y: 1, body: 1, maxBody: 1, mind: 2, visibility: 'hidden', alive: true },
      { id: 'monster-2', type: 'orc', name: 'Orc', x: 4, y: 1, body: 0, maxBody: 1, mind: 2, visibility: 'seen', alive: false },
    ],
    doors: [
      { id: 'door-1', state: 'open', found: true, locked: false },
      { id: 'door-2', state: 'closed', found: false, locked: false },
    ],
    traps: [{ id: 'trap-1', state: 'triggered' }],
    removedBlocks: [],
    consumedNotes: [],
    discovered: [0, 1],
  };
}

describe('trackerView', () => {
  test("the heroes' view (fog) dims furniture, doors and blocked squares the players have not been shown", () => {
    const s = state();
    s.seenFurniture = [];
    s.seenBlocks = ['blocked-2'];
    s.doors = [{ id: 'door-1', state: 'open', found: true, locked: false, seen: true }, { id: 'door-2', state: 'closed', found: false, locked: false }];
    s.addedBlocks = [{ id: 'added-1', x: 2, y: 1, w: 1, h: 1 }];
    const fog = trackerView(s, catalog, { fog: true });
    expect(fog.furniture[0]?.dim).toBe(true);
    expect(fog.doors.map((d) => d.dim ?? false)).toEqual([false, true]);
    expect(fog.blockedSquares.map((b) => [b.id, b.dim ?? false])).toEqual([['blocked-1', true], ['blocked-2', false], ['added-1', false]]);
    const gm = trackerView(s, catalog, { fog: false });
    expect([gm.furniture[0]?.dim, ...gm.doors.map((d) => d.dim), ...gm.blockedSquares.map((b) => b.dim)].every((d) => d === undefined)).toBe(true);
  });

  test('blocked squares added during play are drawn with the quest ones', () => {
    const s = state();
    s.addedBlocks = [{ id: 'added-1', x: 2, y: 1, w: 1, h: 1 }];
    const view = trackerView(s, catalog, { fog: false });
    expect(view.blockedSquares.map((b) => b.id)).toContain('added-1');
  });

  test('removed traps are not drawn; moved traps are drawn where they are now', () => {
    const s = withBoulder();
    s.traps = [{ id: 'trap-1', state: 'removed' }, { id: 'trap-3', state: 'triggered', at: { x: 4, y: 2 } }];
    const view = trackerView(s, catalog, { fog: false });
    expect(view.traps.map((t) => [t.id, t.at])).toEqual([['trap-3', { x: 4, y: 2 }]]);
  });

  test('catalog traps carry their artwork and footprint with the live state', () => {
    const view = trackerView(withLongPit(), catalog, { fog: false });
    expect(view.traps[1]).toEqual({ id: 'trap-2', kind: 'long_pit', at: { x: 4, y: 1 }, state: 'revealed', width: 1, height: 2, rotation: 0, image: 'assets/long_pit.png' });
  });

  test('uses live door, trap and piece state on top of the frozen quest', () => {
    const view = trackerView(state(), catalog, { fog: false });
    expect(view.cols).toBe(4);
    expect(view.doors).toEqual([
      { id: 'door-1', edge: { x: 3, y: 2, orientation: 'vertical' }, kind: 'normal', state: 'open', locked: false },
      { id: 'door-2', edge: { x: 4, y: 1, orientation: 'vertical' }, kind: 'secret', state: 'closed', locked: false },
    ]);
    expect(view.traps).toEqual([{ id: 'trap-1', kind: 'pit', at: { x: 2, y: 2 }, state: 'triggered' }]);
    expect(view.furniture[0]).toMatchObject({ id: 'furniture-1', width: 2, height: 1, image: 'assets/table.png' });
    expect(view.discovered).toBeUndefined();
  });

  test('shows placed heroes, living monsters (hidden ones dimmed) and unused notes', () => {
    const view = trackerView(state(), catalog, { fog: false });
    expect(view.heroes).toEqual([{ id: 'hero-1', type: 'elf', at: { x: 1, y: 2 }, label: 'Faelyn' }]);
    expect(view.monsters).toEqual([{ id: 'monster-1', type: 'orc', at: { x: 3, y: 1 }, label: 'Orc', image: 'assets/orc.png', dim: true }]);
    expect(view.notes).toEqual([{ id: 'note-A', label: 'A', at: { x: 4, y: 2 } }]);

    const used = { ...state(), consumedNotes: ['note-A'] };
    expect(trackerView(used, catalog, { fog: false }).notes).toEqual([]);
  });

  test('draws the frozen board\'s drawn walls', () => {
    const s = state();
    s.board = { ...s.board, drawnWalls: [{ x: 2, y: 1, orientation: 'vertical' }] };
    expect(trackerView(s, catalog, { fog: false }).drawnWalls).toEqual([{ x: 2, y: 1, orientation: 'vertical' }]);
  });

  test('shows exit squares and room colors but not start squares', () => {
    const s = state();
    s.quest = { ...s.quest, startTiles: [{ x: 1, y: 1 }], exitTiles: [{ x: 4, y: 2 }] };
    s.board = { ...s.board, regions: [1, 0, 0, 0, 0, 0, 0, 0], rooms: [{ id: 1, name: 'Hall', color: '#aa0000' }] };
    const view = trackerView(s, catalog, { fog: false });
    expect(view.exitTiles).toEqual([{ x: 4, y: 2 }]);
    expect(view.startTiles).toEqual([]);
    expect(view.roomColors?.get(1)).toBe('#aa0000');
  });

  test('gates keep their kind and doors show their live lock', () => {
    const s = state();
    s.quest = { ...s.quest, doors: [...s.quest.doors, { id: 'door-3', edge: { x: 2, y: 2, orientation: 'horizontal' }, kind: 'gate', state: 'closed', locked: true }] };
    s.doors = [...s.doors, { id: 'door-3', state: 'closed', found: true, locked: false }];
    s.doors[0] = { id: 'door-1', state: 'open', found: true, locked: true };
    const view = trackerView(s, catalog, { fog: false });
    expect(view.doors[2]).toEqual({ id: 'door-3', edge: { x: 2, y: 2, orientation: 'horizontal' }, kind: 'gate', state: 'closed', locked: false });
    expect(view.doors[0]?.locked).toBe(true);
  });

  test('removed blocked squares are not drawn', () => {
    const s = { ...state(), removedBlocks: ['blocked-2'] };
    expect(trackerView(s, catalog, { fog: false }).blockedSquares).toEqual([{ id: 'blocked-1', x: 1, y: 2, w: 1, h: 1, hiddenDoor: true }]);
    const older: SessionState = state();
    delete older.removedBlocks; // sessions started before blocks could be removed
    expect(trackerView(older, catalog, { fog: false }).blockedSquares).toHaveLength(2);
  });

  test('custom monsters are drawn with their size and color and can be picked anywhere on them', () => {
    const s = state();
    s.monsters = [...s.monsters, { id: 'monster-3', type: 'custom-ogre', name: 'Cave Ogre', x: 1, y: 1, body: 6, maxBody: 6, mind: 1, visibility: 'seen', alive: true, width: 2, height: 2, color: '#aa3300' }];
    const view = trackerView(s, catalog, { fog: false });
    expect(view.monsters[1]).toEqual({ id: 'monster-3', type: 'custom-ogre', at: { x: 1, y: 1 }, label: 'Cave Ogre', width: 2, height: 2, color: '#aa3300' });
    expect(pieceAt(s, { x: 2, y: 2 })).toBe('monster-3');
  });

  test('shows teleport squares', () => {
    const s = state();
    s.quest = { ...s.quest, teleports: [{ id: 'teleport-1', x: 1, y: 1, label: 'A' }] };
    expect(trackerView(s, catalog, { fog: false }).teleports).toEqual([{ id: 'teleport-1', at: { x: 1, y: 1 }, label: 'A' }]);
  });

  test('a found secret door is drawn as a normal door', () => {
    const s = state();
    s.doors[1] = { id: 'door-2', state: 'closed', found: true, locked: false };
    expect(trackerView(s, catalog, { fog: false }).doors[1]?.kind).toBe('normal');
  });

  test('fog shows the discovered squares to the GM', () => {
    const view = trackerView(state(), catalog, { fog: true });
    expect(view.discovered).toEqual(new Set([0, 1]));
  });
});

describe('interaction', () => {
  test('select picks furniture anywhere on its footprint (with the catalog)', () => {
    expect(clickCommand(state(), { kind: 'select' }, null, { tile: { x: 2, y: 1 }, edge: null }, catalog)).toEqual({ command: null, select: 'furniture-1' });
  });

  test('block mode blocks the clicked square; select picks an added block', () => {
    expect(clickCommand(state(), { kind: 'block' }, null, { tile: { x: 2, y: 2 }, edge: null })).toEqual({
      command: { type: 'block.add', payload: { x: 2, y: 2 } }, select: null,
    });
    const s = state();
    s.addedBlocks = [{ id: 'added-1', x: 4, y: 1, w: 1, h: 1 }];
    expect(clickCommand(s, { kind: 'select' }, null, { tile: { x: 4, y: 1 }, edge: null })).toEqual({ command: null, select: 'added-1' });
  });

  test('pieceAt prefers heroes, ignores unplaced heroes and dead monsters', () => {
    const s = state();
    expect(pieceAt(s, { x: 1, y: 2 })).toBe('hero-1');
    expect(pieceAt(s, { x: 3, y: 1 })).toBe('monster-1');
    expect(pieceAt(s, { x: 4, y: 1 })).toBeNull();
  });

  test('a monster saved with size 0 (sessions from before monster sizes) is one square and can be selected', () => {
    const s = state();
    const orc = s.monsters[0];
    if (!orc) {
      throw new Error('fixture has no monster');
    }
    orc.width = 0;
    orc.height = 0;
    expect(pieceAt(s, { x: 3, y: 1 })).toBe('monster-1');
    expect(pieceAt(s, { x: 4, y: 1 })).toBeNull();
    expect(clickCommand(s, select, null, { tile: { x: 3, y: 1 }, edge: null })).toEqual({ command: null, select: 'monster-1' });
  });

  test('doorAt matches a door on the exact edge', () => {
    expect(doorAt(state(), { x: 3, y: 2, orientation: 'vertical' })).toBe('door-1');
    expect(doorAt(state(), { x: 3, y: 2, orientation: 'horizontal' })).toBeNull();
  });

  test('doorAt matches either half of a two-wide door, and the view keeps its width and exit kind', () => {
    const s = state();
    s.quest.doors.push({ id: 'door-3', edge: { x: 1, y: 3, orientation: 'horizontal' }, kind: 'exit', state: 'closed', span: 2 });
    expect(doorAt(s, { x: 2, y: 3, orientation: 'horizontal' })).toBe('door-3');
    const view = trackerView(s, catalog, { fog: false }).doors.find((d) => d.id === 'door-3');
    expect(view).toMatchObject({ kind: 'exit', span: 2 });
  });

  const select: Mode = { kind: 'select' };

  test('select mode: a door edge selects the door without opening or closing it', () => {
    const r = clickCommand(state(), select, null, { tile: null, edge: { x: 3, y: 2, orientation: 'vertical' } });
    expect(r).toEqual({ command: null, select: 'door-1' });
    // Even with a movable piece selected, a door click only selects the door.
    expect(clickCommand(state(), select, 'hero-1', { tile: { x: 3, y: 2 }, edge: { x: 3, y: 2, orientation: 'vertical' } })).toEqual({ command: null, select: 'door-1' });
  });

  test('select mode: clicking a piece selects it; clicking a square moves the selection there', () => {
    expect(clickCommand(state(), select, null, { tile: { x: 3, y: 1 }, edge: null })).toEqual({ command: null, select: 'monster-1' });
    expect(clickCommand(state(), select, 'hero-1', { tile: { x: 1, y: 1 }, edge: null })).toEqual({
      command: { type: 'move', payload: { id: 'hero-1', x: 1, y: 1 } },
      select: 'hero-1',
    });
    // Unplaced heroes are placed the same way.
    expect(clickCommand(state(), select, 'hero-2', { tile: { x: 1, y: 1 }, edge: null })?.command?.type).toBe('move');
  });

  test('select mode: clicking an empty square with nothing selected clears the selection', () => {
    expect(clickCommand(state(), select, null, { tile: { x: 4, y: 1 }, edge: null })).toEqual({ command: null, select: null });
  });

  test('select mode: clicking a blocked square selects it', () => {
    expect(clickCommand(state(), select, null, { tile: { x: 3, y: 2 }, edge: null })).toEqual({ command: null, select: 'blocked-2' });
  });

  test('select mode: any square of a catalog trap selects it (traps before notes)', () => {
    expect(clickCommand(withLongPit(), select, null, { tile: { x: 4, y: 2 }, edge: null }, catalog)).toEqual({ command: null, select: 'trap-2' });
    // Without the catalog a trap is one square, so the note on (4,2) is found.
    expect(clickCommand(withLongPit(), select, null, { tile: { x: 4, y: 2 }, edge: null })).toEqual({ command: null, select: 'note-A' });
  });

  test('select mode: a selected movable trap moves to the next clicked square; other traps do not', () => {
    expect(clickCommand(withBoulder(), select, null, { tile: { x: 2, y: 1 }, edge: null }, catalog)).toEqual({ command: null, select: 'trap-3' });
    expect(clickCommand(withBoulder(), select, 'trap-3', { tile: { x: 4, y: 1 }, edge: null }, catalog)).toEqual({
      command: { type: 'move', payload: { id: 'trap-3', x: 4, y: 1 } },
      select: 'trap-3',
    });
    expect(clickCommand(withBoulder(), select, 'trap-1', { tile: { x: 4, y: 1 }, edge: null }, catalog)).toEqual({ command: null, select: null });
  });

  test('select mode: removed traps cannot be clicked; moved traps are found where they are now', () => {
    const s = withBoulder();
    s.traps = [{ id: 'trap-1', state: 'removed' }, { id: 'trap-3', state: 'triggered', at: { x: 4, y: 1 } }];
    expect(clickCommand(s, select, null, { tile: { x: 2, y: 2 }, edge: null }, catalog)).toEqual({ command: null, select: null });
    // The boulder's old square: only the table under it is found there now.
    expect(clickCommand(s, select, null, { tile: { x: 2, y: 1 }, edge: null }, catalog)).toEqual({ command: null, select: 'furniture-1' });
    expect(clickCommand(s, select, null, { tile: { x: 4, y: 1 }, edge: null }, catalog)).toEqual({ command: null, select: 'trap-3' });
  });

  test('reveal mode can also show the monsters in the area', () => {
    expect(clickCommand(state(), { kind: 'reveal', seen: true }, null, { tile: { x: 4, y: 1 }, edge: null })?.command).toEqual({
      type: 'area.reveal',
      payload: { x: 4, y: 1, seen: true },
    });
  });

  test('picked squares: painting adds or removes, and one command reveals them all', () => {
    let pending = paintPending([], { x: 1, y: 1 }, true);
    pending = paintPending(pending, { x: 2, y: 1 }, true);
    pending = paintPending(pending, { x: 2, y: 1 }, true); // already there
    expect(pending).toEqual([{ x: 1, y: 1 }, { x: 2, y: 1 }]);
    expect(paintPending(pending, { x: 1, y: 1 }, false)).toEqual([{ x: 2, y: 1 }]);
    expect(revealSquaresCommand([], true)).toBeNull();
    expect(revealSquaresCommand(pending, true)).toEqual({ type: 'tiles.reveal', payload: { tiles: [{ x: 1, y: 1 }, { x: 2, y: 1 }], seen: true } });
    expect(revealSquaresCommand(pending, false)).toEqual({ type: 'tiles.reveal', payload: { tiles: [{ x: 1, y: 1 }, { x: 2, y: 1 }] } });
  });

  test('a reveal drag reveals the undiscovered corridor squares along it, never room squares or solid rock', () => {
    const s = state();
    // (4,1) and (4,2) are a room, (1,2) is solid rock; (1,1) and (2,1) are already discovered.
    s.board.regions = [0, 0, 0, 1, -1, 0, 0, 1];
    const path = [
      { x: 1, y: 1 }, { x: 2, y: 1 }, { x: 3, y: 1 }, { x: 4, y: 1 }, { x: 4, y: 2 },
      { x: 3, y: 2 }, { x: 2, y: 2 }, { x: 1, y: 2 }, { x: 2, y: 2 },
    ];
    expect(corridorPath(s, path)).toEqual([{ x: 3, y: 1 }, { x: 3, y: 2 }, { x: 2, y: 2 }]);
    // The trap on (2,2) is only a square to reveal: trap states change by trap.set alone.
    expect(revealPathCommand(s, path, true)).toEqual({ type: 'tiles.reveal', payload: { tiles: [{ x: 3, y: 1 }, { x: 3, y: 2 }, { x: 2, y: 2 }], seen: true } });
    expect(revealPathCommand(s, path, false)).toEqual({ type: 'tiles.reveal', payload: { tiles: [{ x: 3, y: 1 }, { x: 3, y: 2 }, { x: 2, y: 2 }] } });
    // Nothing new along the drag: no command.
    expect(revealPathCommand(s, [{ x: 1, y: 1 }, { x: 4, y: 1 }], true)).toBeNull();
  });

  test('select mode: clicking a trap or note square selects it when no piece is there', () => {
    expect(clickCommand(state(), select, null, { tile: { x: 2, y: 2 }, edge: null })).toEqual({ command: null, select: 'trap-1' });
    expect(clickCommand(state(), select, null, { tile: { x: 4, y: 2 }, edge: null })).toEqual({ command: null, select: 'note-A' });
  });

  test('reveal and hide modes act on the clicked area or square', () => {
    expect(clickCommand(state(), { kind: 'reveal' }, null, { tile: { x: 4, y: 1 }, edge: null })?.command).toEqual({
      type: 'area.reveal',
      payload: { x: 4, y: 1 },
    });
    expect(clickCommand(state(), { kind: 'hide' }, null, { tile: { x: 4, y: 1 }, edge: null })?.command).toEqual({
      type: 'tiles.hide',
      payload: { tiles: [{ x: 4, y: 1 }] },
    });
  });

  test('add-monster mode places the chosen type', () => {
    expect(clickCommand(state(), { kind: 'addMonster', monsterType: 'orc' }, null, { tile: { x: 1, y: 1 }, edge: null })).toEqual({
      command: { type: 'monster.add', payload: { type: 'orc', x: 1, y: 1 } },
      select: null,
    });
  });

  test('clicks off the board do nothing', () => {
    expect(clickCommand(state(), select, 'hero-1', { tile: null, edge: null })).toBeNull();
  });
});

describe('formatEvent', () => {
  test('shows round, local time and summary', () => {
    const line = formatEvent({ seq: 3, round: 2, kind: 'door.set', summary: 'Opened door-1', payload: {}, createdAt: '2026-09-27T18:05:00Z' });
    expect(line.round).toBe('R2');
    expect(line.summary).toBe('Opened door-1');
    expect(line.time).toMatch(/\d{1,2}:\d{2}/);
    expect(line.kind).toBe('door');
  });

  test('categorizes log notes and session events for styling', () => {
    expect(formatEvent({ seq: 1, round: 1, kind: 'log.note', summary: 'x', payload: {}, createdAt: '' }).kind).toBe('note');
    expect(formatEvent({ seq: 1, round: 1, kind: 'session.start', summary: 'x', payload: {}, createdAt: '' }).kind).toBe('session');
    expect(formatEvent({ seq: 1, round: 1, kind: 'weird', summary: 'x', payload: {}, createdAt: 'not a date' }).time).toBe('');
  });
});

describe('searchEvents', () => {
  const ev = (seq: number, round: number, summary: string) => ({ seq, round, kind: 'log.note', summary, payload: {}, createdAt: '' });
  const events = [ev(1, 1, 'Opened door-1'), ev(2, 2, 'Orc (monster-3): body 22 → 15'), ev(3, 12, 'Vex used Healing Potion'), ev(4, 2, 'Fight over')];
  test('matches words in any order, ignoring case; "r2" picks a round', () => {
    expect(searchEvents(events, '').map((e) => e.seq)).toEqual([1, 2, 3, 4]);
    expect(searchEvents(events, 'ORC').map((e) => e.seq)).toEqual([2]);
    expect(searchEvents(events, 'potion vex').map((e) => e.seq)).toEqual([3]);
    expect(searchEvents(events, 'r2').map((e) => e.seq)).toEqual([2, 4]);
    expect(searchEvents(events, 'r2 orc').map((e) => e.seq)).toEqual([2]);
    expect(searchEvents(events, 'dragon')).toEqual([]);
  });
});

describe('shownToPlayers', () => {
  test('whether each kind of piece is on the player screen; null for things that cannot be shown', () => {
    const s = state();
    s.seenFurniture = ['furniture-1'];
    s.seenBlocks = [];
    s.doors = [{ id: 'door-1', state: 'open', found: true, locked: false, seen: true }, { id: 'door-2', state: 'closed', found: false, locked: false }];
    s.addedBlocks = [{ id: 'added-1', x: 2, y: 1, w: 1, h: 1 }];
    expect(shownToPlayers(s, 'furniture-1')).toBe(true);
    expect(shownToPlayers(s, 'blocked-2')).toBe(false);
    expect(shownToPlayers(s, 'door-1')).toBe(true);
    expect(shownToPlayers(s, 'door-2')).toBe(false);
    expect(shownToPlayers(s, 'monster-1')).toBe(false);
    expect(shownToPlayers(s, 'added-1')).toBe(true);
    expect(shownToPlayers(s, 'trap-1')).toBeNull();
    expect(shownToPlayers(s, 'note-A')).toBeNull();
  });
});

describe('hotkey', () => {
  const plain = { ctrlKey: false, metaKey: false, altKey: false };
  test('1 is Select / move and clears the selection, 2 Reveal, 3 Hide', () => {
    expect(hotkey({ key: '1', ...plain }, { kind: 'hide' }, true)).toEqual({ mode: { kind: 'select' }, deselect: true });
    expect(hotkey({ key: '1', ...plain }, { kind: 'select' }, true)).toEqual({ mode: { kind: 'select' }, deselect: true });
    expect(hotkey({ key: '2', ...plain }, { kind: 'select' }, false)).toEqual({ mode: { kind: 'reveal', seen: false }, deselect: false });
    expect(hotkey({ key: '3', ...plain }, { kind: 'reveal', seen: true }, true)).toEqual({ mode: { kind: 'hide' }, deselect: false });
  });
  test('2 keeps square picking on, and other keys or shortcuts with modifiers do nothing', () => {
    const picking: Mode = { kind: 'pickSquares', seen: true };
    expect(hotkey({ key: '2', ...plain }, picking, true)).toEqual({ mode: picking, deselect: false });
    expect(hotkey({ key: '4', ...plain }, picking, true)).toBeNull();
    expect(hotkey({ key: '1', ...plain, metaKey: true }, picking, true)).toBeNull();
    expect(hotkey({ key: '2', ...plain, ctrlKey: true }, picking, true)).toBeNull();
  });
});
