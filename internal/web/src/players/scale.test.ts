import { describe, expect, test } from 'bun:test';
import { nextScale, parseScale, SCALES } from './scale.ts';

describe('text size steps', () => {
  test('steps up and down through the sizes and stops at the ends', () => {
    expect(nextScale(1, 1)).toBe(1.15);
    expect(nextScale(SCALES[1] ?? 0, -1)).toBe(1);
    expect(nextScale(1, -1)).toBe(1);
    const biggest = SCALES[SCALES.length - 1] ?? 0;
    expect(nextScale(biggest, 1)).toBe(biggest);
  });

  test('a stored size is read back, anything else is the normal size', () => {
    expect(parseScale('1.3')).toBe(1.3);
    expect(parseScale('7')).toBe(1);
    expect(parseScale(null)).toBe(1);
    expect(parseScale('big')).toBe(1);
  });
});
