import { describe, expect, test } from 'bun:test';
import { travelOptions } from './travel.ts';
import type { Chapter } from './types.ts';

const chapters: Chapter[] = [
  { number: 1, questId: 'q1', questName: 'Upper Halls', boardId: 'b1', boardName: 'Level One' },
  { number: 2, questId: 'q2', questName: 'Lower Vaults', boardId: 'b2', boardName: 'Level Two' },
  { number: 3, questId: 'q3', questName: 'Deep Crypt', boardId: 'b3', boardName: 'Level Three' },
];

describe('travelOptions', () => {
  test('lists the other chapters, marking maps already visited', () => {
    const opts = travelOptions({ questId: 'q2', otherMaps: [{ questId: 'q1', questName: 'Upper Halls' }] }, chapters);
    expect(opts).toEqual([
      { questId: 'q1', label: 'Chapter 1: Upper Halls (return)' },
      { questId: 'q3', label: 'Chapter 3: Deep Crypt' },
    ]);
  });

  test('includes visited maps that are not chapters of the campaign', () => {
    const opts = travelOptions({ questId: 'q1', otherMaps: [{ questId: 'side', questName: 'Side Cave' }] }, chapters);
    expect(opts.map((o) => o.label)).toEqual(['Chapter 2: Lower Vaults', 'Chapter 3: Deep Crypt', 'Side Cave (return)']);
  });

  test('works for sessions without travel data', () => {
    expect(travelOptions({}, chapters)).toHaveLength(3);
  });
});
