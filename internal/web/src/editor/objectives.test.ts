import { describe, expect, test } from 'bun:test';
import type { Catalog } from '../maps/types.ts';
import type { QuestDoc } from '../maps/types.ts';
import { emptyQuest, placeDoor, placeMonster, removeItem } from './model.ts';
import { addObjective, MAX_GOAL, objectiveText, removeObjective, setCollectItem, setDoorKey, setGoal, setKillTargets } from './objectives.ts';

const catalog = { furniture: [], monsters: [{ id: 'orc', name: 'Orc', body: 1, mind: 2 }], heroes: [], traps: [] } as unknown as Catalog;

function withOrcs(): QuestDoc {
  let q = placeMonster(emptyQuest(), 'orc', { x: 1, y: 1 });
  q = placeMonster(q, 'orc', { x: 2, y: 1 });
  return q;
}

describe('quest goal', () => {
  test('is set trimmed and dropped when empty', () => {
    const q = setGoal(emptyQuest(), '  Slay the orc chief and escape  ');
    expect(q.goal).toBe('Slay the orc chief and escape');
    expect('goal' in setGoal(q, '   ')).toBe(false);
  });

  test('is cut to the limit', () => {
    expect(setGoal(emptyQuest(), 'x'.repeat(MAX_GOAL + 10)).goal).toHaveLength(MAX_GOAL);
  });
});

describe('objectives', () => {
  test('are added, changed and removed by position', () => {
    let q = addObjective(withOrcs(), { kind: 'kill' });
    q = addObjective(q, { kind: 'collect', item: ' Soul Gem ' });
    q = addObjective(q, { kind: 'escape' });
    expect(q.objectives).toEqual([{ kind: 'kill' }, { kind: 'collect', item: 'Soul Gem' }, { kind: 'escape' }]);

    q = setKillTargets(q, 0, ['monster-2']);
    expect(q.objectives?.[0]).toEqual({ kind: 'kill', monsters: ['monster-2'] });
    q = setKillTargets(q, 0, []);
    expect(q.objectives?.[0]).toEqual({ kind: 'kill' });

    q = setCollectItem(q, 1, 'Crown');
    expect(q.objectives?.[1]).toEqual({ kind: 'collect', item: 'Crown' });
    expect(setCollectItem(q, 1, '  ').objectives?.[1]).toEqual({ kind: 'collect', item: 'Crown' });

    q = removeObjective(q, 0);
    expect(q.objectives).toEqual([{ kind: 'collect', item: 'Crown' }, { kind: 'escape' }]);
    q = removeObjective(removeObjective(q, 0), 0);
    expect('objectives' in q).toBe(false);
  });

  test('a collect objective needs an item', () => {
    const q = emptyQuest();
    expect(addObjective(q, { kind: 'collect', item: '  ' })).toBe(q);
  });

  test('removing a monster drops it from kill objectives, and a kill left naming nobody goes', () => {
    let q = addObjective(withOrcs(), { kind: 'kill', monsters: ['monster-1', 'monster-2'] });
    q = addObjective(q, { kind: 'kill', monsters: ['monster-2'] });
    q = addObjective(q, { kind: 'kill' });
    q = removeItem(q, 'monster-2');
    expect(q.objectives).toEqual([{ kind: 'kill', monsters: ['monster-1'] }, { kind: 'kill' }]);
  });

  test('read as a line for the GM', () => {
    const q = withOrcs();
    expect(objectiveText({ kind: 'kill' }, q, catalog)).toBe('Kill every monster');
    expect(objectiveText({ kind: 'kill', monsters: ['monster-1'] }, q, catalog)).toBe('Kill Orc (monster-1)');
    expect(objectiveText({ kind: 'collect', item: 'Soul Gem' }, q, catalog)).toBe('Carry the Soul Gem');
    expect(objectiveText({ kind: 'escape' }, q, catalog)).toBe('Leave by the exits');
  });
});

describe('door keys', () => {
  test('a locked door names its key; unlocked doors and empty names have none', () => {
    let q = placeDoor(emptyQuest(), { x: 2, y: 1, orientation: 'vertical' }, 'gate', true);
    q = setDoorKey(q, 'door-1', ' Iron Key ');
    expect(q.doors[0]?.key).toBe('Iron Key');
    expect('key' in (setDoorKey(q, 'door-1', '').doors[0] ?? {})).toBe(false);
    const open = placeDoor(emptyQuest(), { x: 2, y: 1, orientation: 'vertical' }, 'normal', false);
    expect(setDoorKey(open, 'door-1', 'Iron Key').doors[0]?.key).toBeUndefined();
  });
});
