import { describe, expect, test } from 'bun:test';
import type { Catalog } from '../maps/types.ts';
import { addBlockedSquare, emptyQuest, placeDoor, placeFurniture, placeMonster, placeNote, placeTeleport, placeTrap } from './model.ts';
import { activeOnSquare, describeItem, squareStack } from './selection.ts';

const catalog: Catalog = {
  furniture: [{ id: 'chest', name: 'Chest', width: 1, height: 1, blocksMovement: true, blocksLineOfSight: false }],
  monsters: [{ id: 'orc', name: 'Orc', body: 1, mind: 2, attack: 3, defense: 2, movement: 8 }],
  heroes: [],
  traps: [{ id: 'pit', name: 'Pit Trap', width: 1, height: 1 }],
};

describe('squareStack', () => {
  test('lists everything on the square topmost first, then a door under the pointer', () => {
    let q = placeFurniture(emptyQuest(), 'chest', { x: 4, y: 23 }, 0);
    q = placeTrap(q, 'chest', { x: 4, y: 23 });
    q = placeNote(q, { x: 4, y: 23 }, 'A dusty chest');
    q = placeDoor(q, { x: 4, y: 23, orientation: 'vertical' }, 'normal', false);
    const edge = { x: 4, y: 23, orientation: 'vertical' as const };
    expect(squareStack(q, catalog, { x: 4, y: 23 }, null)).toEqual(['trap-1', 'note-1', 'furniture-1']);
    expect(squareStack(q, catalog, { x: 4, y: 23 }, edge)).toEqual(['trap-1', 'note-1', 'furniture-1', 'door-1']);
  });

  test('a door alone, or nothing', () => {
    const q = placeDoor(emptyQuest(), { x: 2, y: 2, orientation: 'horizontal' }, 'normal', false);
    expect(squareStack(q, catalog, { x: 2, y: 2 }, { x: 2, y: 2, orientation: 'horizontal' })).toEqual(['door-1']);
    expect(squareStack(q, catalog, null, { x: 2, y: 2, orientation: 'horizontal' })).toEqual(['door-1']);
    expect(squareStack(q, catalog, { x: 5, y: 5 }, null)).toEqual([]);
  });
});

describe('activeOnSquare', () => {
  const stack = ['monster-1', 'note-1', 'blocked-1'];
  const here = { x: 4, y: 23 };

  test('picks the topmost item on a newly clicked square', () => {
    expect(activeOnSquare(stack, here, null, null)).toBe('monster-1');
    expect(activeOnSquare(stack, here, { x: 1, y: 1 }, 'blocked-1')).toBe('monster-1');
  });

  test('keeps the item chosen in the panel when the same square is clicked again', () => {
    expect(activeOnSquare(stack, here, { x: 4, y: 23 }, 'blocked-1')).toBe('blocked-1');
  });

  test('falls back to the top when the chosen item is no longer there', () => {
    expect(activeOnSquare(stack, here, { x: 4, y: 23 }, 'trap-9')).toBe('monster-1');
    expect(activeOnSquare(stack, here, { x: 4, y: 23 }, null)).toBe('monster-1');
  });

  test('nothing on the square selects nothing', () => {
    expect(activeOnSquare([], here, here, 'monster-1')).toBeNull();
  });
});

describe('describeItem', () => {
  test('names each kind of item', () => {
    let q = placeMonster(emptyQuest(), 'orc', { x: 1, y: 1 });
    q = placeMonster(q, 'mystery', { x: 1, y: 1 });
    q = placeTrap(q, 'pit', { x: 1, y: 1 });
    q = placeTrap(q, 'trigger', { x: 1, y: 1 });
    q = placeTrap(q, 'chest', { x: 1, y: 1 });
    q = placeFurniture(q, 'chest', { x: 1, y: 1 }, 0);
    q = placeNote(q, { x: 1, y: 1 }, '');
    q = placeTeleport(q, { x: 1, y: 1 });
    q = addBlockedSquare(q, { x: 1, y: 1, w: 2, h: 3 });
    q = placeDoor(q, { x: 1, y: 1, orientation: 'vertical' }, 'secret', false);
    expect(describeItem(q, catalog, 'monster-1')).toEqual({ kind: 'Monster', name: 'Orc' });
    expect(describeItem(q, catalog, 'monster-2')).toEqual({ kind: 'Monster', name: 'mystery' });
    expect(describeItem(q, catalog, 'trap-1')).toEqual({ kind: 'Trap', name: 'Pit Trap' });
    expect(describeItem(q, catalog, 'trap-2')).toEqual({ kind: 'Trap', name: 'Trigger' });
    expect(describeItem(q, catalog, 'trap-3')).toEqual({ kind: 'Trap', name: 'chest trap' });
    expect(describeItem(q, catalog, 'furniture-1')).toEqual({ kind: 'Furniture', name: 'Chest' });
    expect(describeItem(q, catalog, 'note-1')).toEqual({ kind: 'Note', name: 'A' });
    expect(describeItem(q, catalog, 'teleport-1')).toEqual({ kind: 'Teleport', name: 'square' });
    expect(describeItem(q, catalog, 'blocked-1')).toEqual({ kind: 'Blocked', name: '2×3 squares' });
    expect(describeItem(q, catalog, 'door-1')).toEqual({ kind: 'Door', name: 'secret' });
    expect(describeItem(q, catalog, 'nope')).toBeNull();
  });

  test('labels show on traps and teleports', () => {
    let q = placeTeleport(emptyQuest(), { x: 1, y: 1 });
    q = { ...q, teleports: (q.teleports ?? []).map((t) => ({ ...t, label: '2' })) };
    q = placeTrap(q, 'trigger', { x: 1, y: 1 });
    q = { ...q, traps: q.traps.map((t) => ({ ...t, label: 'Gate' })) };
    expect(describeItem(q, catalog, 'teleport-1')).toEqual({ kind: 'Teleport', name: '2' });
    expect(describeItem(q, catalog, 'trap-1')).toEqual({ kind: 'Trap', name: 'Trigger "Gate"' });
  });
});
