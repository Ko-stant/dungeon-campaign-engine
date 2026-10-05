import { describe, expect, test } from 'bun:test';
import { effectLabel, EFFECT_SUGGESTIONS } from './effects.ts';
import { formatEvent } from './format.ts';

describe('effectLabel', () => {
  test('name, rounds left and note', () => {
    expect(effectLabel({ id: 'effect-1', name: 'Poisoned', rounds: 3, note: '3 a turn' })).toBe('Poisoned (3 rounds): 3 a turn');
    expect(effectLabel({ id: 'effect-2', name: 'Raging', rounds: 1 })).toBe('Raging (1 round)');
    expect(effectLabel({ id: 'effect-3', name: 'Cursed' })).toBe('Cursed');
  });

  test('suggests the campaign\'s effects', () => {
    expect(EFFECT_SUGGESTIONS).toContain('Poisoned');
    expect(EFFECT_SUGGESTIONS).toContain('Turned');
  });
});

describe('formatEvent for fights and effects', () => {
  const base = { seq: 1, round: 2, summary: 's', payload: {}, createdAt: '' };
  test('fight events style like rounds, effects like notes', () => {
    expect(formatEvent({ ...base, kind: 'fight.start' }).kind).toBe('round');
    expect(formatEvent({ ...base, kind: 'effect.add' }).kind).toBe('note');
  });

  test('party gold and equipping style like hero changes', () => {
    expect(formatEvent({ ...base, kind: 'gold.set' }).kind).toBe('hero');
    expect(formatEvent({ ...base, kind: 'item.equip' }).kind).toBe('hero');
  });
});
