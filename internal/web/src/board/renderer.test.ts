import { describe, expect, test } from 'bun:test';
import { CORRIDOR, VOID } from './model.ts';
import { tileFill, type BoardTheme } from './renderer.ts';

describe('tileFill', () => {
  const theme = { rock: 'rock', corridor: 'corridor', room: 'room', seen: 'seen room', seenCorridor: 'seen corridor' } as BoardTheme;
  test('revealed corridors and rooms get different light colors, so they tell apart at a glance', () => {
    expect(tileFill(theme, 3, true)).toBe('seen room');
    expect(tileFill(theme, CORRIDOR, true)).toBe('seen corridor');
  });
  test('unrevealed squares keep the map colors, and solid rock is never revealed', () => {
    expect(tileFill(theme, 3, false)).toBe('room');
    expect(tileFill(theme, CORRIDOR, false)).toBe('corridor');
    expect(tileFill(theme, VOID, true)).toBe('rock');
  });
});
