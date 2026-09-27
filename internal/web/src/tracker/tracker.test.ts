import { describe, expect, test } from 'bun:test';
import { DOC_VERSION, type Catalog } from '../maps/types.ts';
import { formatEvent } from './format.ts';
import { clickCommand, doorAt, pieceAt, type Mode } from './interaction.ts';
import type { SessionState } from './types.ts';
import { trackerView } from './view.ts';

const catalog: Catalog = {
  furniture: [{ id: 'table', name: 'Table', width: 2, height: 1, blocksMovement: true, blocksLineOfSight: false, image: 'assets/table.png' }],
  monsters: [{ id: 'orc', name: 'Orc', body: 1, mind: 2, attack: 3, defense: 2, movement: 8, image: 'assets/orc.png' }],
  heroes: [{ id: 'elf', name: 'Elf', body: 6, mind: 4, attack: 2, defense: 2, movementDice: 2 }],
};

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
      { id: 'hero-1', name: 'Faelyn', class: 'elf', x: 1, y: 2, placed: true, body: 6, maxBody: 6, mind: 4, maxMind: 4, gold: 0, status: 'active' },
      { id: 'hero-2', name: 'Bram', class: 'elf', x: 1, y: 2, placed: false, body: 6, maxBody: 6, mind: 4, maxMind: 4, gold: 0, status: 'active' },
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
  test('pieceAt prefers heroes, ignores unplaced heroes and dead monsters', () => {
    const s = state();
    expect(pieceAt(s, { x: 1, y: 2 })).toBe('hero-1');
    expect(pieceAt(s, { x: 3, y: 1 })).toBe('monster-1');
    expect(pieceAt(s, { x: 4, y: 1 })).toBeNull();
  });

  test('doorAt matches a door on the exact edge', () => {
    expect(doorAt(state(), { x: 3, y: 2, orientation: 'vertical' })).toBe('door-1');
    expect(doorAt(state(), { x: 3, y: 2, orientation: 'horizontal' })).toBeNull();
  });

  const select: Mode = { kind: 'select' };

  test('select mode: a door edge toggles the door', () => {
    const r = clickCommand(state(), select, null, { tile: null, edge: { x: 3, y: 2, orientation: 'vertical' } });
    expect(r).toEqual({ command: { type: 'door.set', payload: { id: 'door-1', state: 'closed' } }, select: 'door-1' });
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
    expect(clickCommand(state(), select, null, { tile: { x: 1, y: 1 }, edge: null })).toEqual({ command: null, select: null });
  });

  test('select mode: clicking a blocked square selects it', () => {
    expect(clickCommand(state(), select, null, { tile: { x: 3, y: 2 }, edge: null })).toEqual({ command: null, select: 'blocked-2' });
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

  test('categorises log notes and session events for styling', () => {
    expect(formatEvent({ seq: 1, round: 1, kind: 'log.note', summary: 'x', payload: {}, createdAt: '' }).kind).toBe('note');
    expect(formatEvent({ seq: 1, round: 1, kind: 'session.start', summary: 'x', payload: {}, createdAt: '' }).kind).toBe('session');
    expect(formatEvent({ seq: 1, round: 1, kind: 'weird', summary: 'x', payload: {}, createdAt: 'not a date' }).time).toBe('');
  });
});
