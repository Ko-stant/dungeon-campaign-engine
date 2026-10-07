import { describe, expect, test } from 'bun:test';
import { currentSection, noteLabels, passageClips, passageNeighbors, passageNotes, sectionProgress } from './script.ts';
import type { Chapter, ScriptSection } from './types.ts';

const passage = (id: string) => ({ id, title: `Title ${id}`, parts: [{ paragraphs: ['Text.'] }] });
const sections: ScriptSection[] = [
  { title: 'Prologue', passages: [passage('P0-01'), passage('P0-02')] },
  { title: 'Quest 1 - The Crumbling Halls', passages: [passage('Q1-01')] },
  { title: 'Quest 2 - The Bloated Fields', passages: [passage('Q2-01'), passage('Q2-02')] },
  { title: 'The Wardens\' Rise extras', passages: [passage('G-01')] },
];
const chapters: Chapter[] = [
  { number: 1, questId: 'q1', questName: 'The Crumbling Halls', boardId: 'b1', boardName: 'Halls' },
  { number: 2, questId: 'q2', questName: 'The Bloated Fields', boardId: 'b2', boardName: 'Fields' },
  { number: 3, questId: 'q3', questName: 'The Wardens\' Rise', boardId: 'b3', boardName: 'Rise' },
];

describe('currentSection', () => {
  test('matches "Quest N" to the chapter number of the active quest', () => {
    expect(currentSection(sections, { questId: 'q2', questName: 'Renamed quest' }, chapters)).toBe(2);
  });

  test('falls back to a section naming the quest', () => {
    expect(currentSection(sections, { questId: 'q3', questName: 'The Wardens\' Rise' }, chapters)).toBe(3);
    expect(currentSection(sections, { questName: 'the bloated fields' }, [])).toBe(2);
  });

  test('is -1 when nothing matches', () => {
    expect(currentSection(sections, { questId: 'x', questName: 'Side cave' }, chapters)).toBe(-1);
    expect(currentSection([], { questName: 'x' }, chapters)).toBe(-1);
  });
});

describe('passageNeighbors', () => {
  test('steps through passages in script order, across sections', () => {
    expect(passageNeighbors(sections, 'P0-02')).toEqual({ prev: 'P0-01', next: 'Q1-01' });
    expect(passageNeighbors(sections, 'P0-01')).toEqual({ prev: null, next: 'P0-02' });
    expect(passageNeighbors(sections, 'G-01')).toEqual({ prev: 'Q2-02', next: null });
    expect(passageNeighbors(sections, 'nope')).toEqual({ prev: null, next: null });
  });
});

describe('sectionProgress', () => {
  test('counts passages read in a section', () => {
    expect(sectionProgress(sections[2] ?? sections[0] ?? { title: '', passages: [] }, ['Q2-02', 'P0-01'])).toBe('1/2 read');
    expect(sectionProgress(sections[0] ?? { title: '', passages: [] }, undefined)).toBe('0/2 read');
  });
});

describe('read-aloud events', () => {
  test('are filed with the narration notes in the log', async () => {
    const { formatEvent } = await import('./format.ts');
    expect(formatEvent({ seq: 1, round: 1, kind: 'passage.read', summary: 's', payload: {}, createdAt: '' }).kind).toBe('note');
  });
});

describe('passageClips', () => {
  const clips = {
    'Q3-09c': '/audio/c/Q3-09c.mp3',
    'Q3-09': '/audio/c/Q3-09.mp3',
    'Q3-09a': '/audio/c/Q3-09a.ogg',
    'Q3-091': '/audio/c/Q3-091.mp3',
    'Q3-09ab': '/audio/c/Q3-09ab.mp3',
    'Q2-03': '/audio/c/Q2-03.wav',
  };

  test('finds the passage clip first, then its lettered clips in order', () => {
    expect(passageClips('Q3-09', clips)).toEqual([
      { id: 'Q3-09', label: 'Clip', url: '/audio/c/Q3-09.mp3' },
      { id: 'Q3-09a', label: 'a', url: '/audio/c/Q3-09a.ogg' },
      { id: 'Q3-09c', label: 'c', url: '/audio/c/Q3-09c.mp3' },
    ]);
    expect(passageClips('Q2-03', clips)).toEqual([{ id: 'Q2-03', label: 'Clip', url: '/audio/c/Q2-03.wav' }]);
  });

  test('is empty when the passage has no clip', () => {
    expect(passageClips('P0-01', clips)).toEqual([]);
    expect(passageClips('P0-01', {})).toEqual([]);
  });
});

describe('passageNotes', () => {
  const note = (label: string, text: string) => ({ id: `note-${label}`, label, x: 1, y: 1, text });
  const notes = [
    note('A', 'Carved into the lid of a Warden\'s tomb, half buried in rubble, are four signs you have seen before.\n\n"Three the plagues..."'),
    note('B', 'Half buried under a fall of rubble lies what is left of a soldier. A faded blue surcoat.\n\nIt reads:'),
    note('E', 'The gate key'),
    note('G', 'Q1-N9 the GM\'s own marker'),
  ];
  const tomb = { id: 'Q1-02', title: 'The verse', parts: [{ paragraphs: ['Carved into the lid of a Warden\'s tomb, half buried in rubble, are four signs you have seen before, on the cracked fountain.'] }] };
  const soldier = { id: 'Q1-N1', title: 'Tomas', parts: [{ speaker: 'Narrator', paragraphs: ['Half buried under a fall of rubble lies what is left of a soldier.'] }, { speaker: 'Tomas', paragraphs: ['Tomas Reed.'] }] };

  test('a note holding the start of the passage\'s text is its note, ignoring case and punctuation', () => {
    expect(passageNotes(tomb, notes).map((n) => n.label)).toEqual(['A']);
    expect(passageNotes({ ...soldier, parts: [{ paragraphs: ['HALF buried -- under a fall of rubble, lies what is left of a soldier!'] }] }, notes).map((n) => n.label)).toEqual(['B']);
  });

  test('one differing word near the start (grey and gray) still matches on the words after it', () => { // spelling:allow (the GM's note spells it this way)
    const gate = { id: 'Q1-03', title: 'Gate', parts: [{ paragraphs: ['Daylight. A thin gray line of it, spilling under a pair of iron-bound doors at the end of the hall.'] }] };
    const withGate = [...notes, note('F', 'Daylight. A thin grey line of it, spilling under a pair of iron-bound doors at the end of the hall. The eastern gate.')]; // spelling:allow
    expect(passageNotes(gate, withGate).map((n) => n.label)).toEqual(['F']);
  });

  test('a note that starts with the passage id is its note too', () => {
    expect(passageNotes({ id: 'Q1-N9', title: 'x', parts: [{ paragraphs: ['Something else entirely, with no match at all.'] }] }, notes).map((n) => n.label)).toEqual(['G']);
  });

  test('no note: none; short notes never match by accident', () => {
    expect(passageNotes({ id: 'Q1-01', title: 'Start', parts: [{ paragraphs: ['The great doors groan shut behind you.'] }] }, notes)).toEqual([]);
    expect(passageNotes({ id: 'X', title: 'x', parts: [{ paragraphs: ['The gate'] }] }, notes)).toEqual([]);
  });

  test('noteLabels maps each passage id to its note letters, for a whole script', () => {
    const labels = noteLabels([{ title: 'Quest 1', passages: [tomb, soldier, passage('Q1-01')] }], notes);
    expect(labels.get('Q1-02')).toEqual(['A']);
    expect(labels.get('Q1-N1')).toEqual(['B']);
    expect(labels.has('Q1-01')).toBe(false);
  });

  test('with a section given, only its passages are matched (the notes belong to the active map)', () => {
    const script = [{ title: 'Quest 1', passages: [tomb] }, { title: 'Quest 2', passages: [{ ...soldier, id: 'Q2-N2' }] }];
    expect([...noteLabels(script, notes, 0).keys()]).toEqual(['Q1-02']);
    expect([...noteLabels(script, notes, 1).keys()]).toEqual(['Q2-N2']);
    expect([...noteLabels(script, notes, -1).keys()]).toEqual(['Q1-02', 'Q2-N2']);
  });
});
