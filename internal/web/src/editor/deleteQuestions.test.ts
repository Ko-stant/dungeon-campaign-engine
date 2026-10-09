import { describe, expect, test } from 'bun:test';
import { boardDeleteQuestion, questDeleteQuestion } from './deleteQuestions.ts';

describe('questDeleteQuestion', () => {
  test('names the quest and what happens to campaigns and sessions', () => {
    const q = questDeleteQuestion('Crumbling Halls');
    expect(q).toContain('Crumbling Halls');
    expect(q).toContain("campaign's chapters");
    expect(q).toContain('Sessions already started keep their copy');
  });

  test('an unnamed quest still reads well', () => {
    expect(questDeleteQuestion('  ')).toContain('this quest');
  });
});

describe('boardDeleteQuestion', () => {
  test('a board without quests', () => {
    expect(boardDeleteQuestion('Eastmarch (draft)', [])).toBe('Delete the board Eastmarch (draft) for good? Sessions already started keep their copy of the map.');
  });

  test('lists the quests that go with the board', () => {
    expect(boardDeleteQuestion('Eastmarch', ['The Bloated Fields'])).toContain('Its quest (The Bloated Fields) goes with it');
    expect(boardDeleteQuestion('Keep', ['A', 'B', 'C'])).toContain('Its 3 quests (A, B and C) go with it');
  });
});
