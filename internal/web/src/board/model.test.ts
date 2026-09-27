import { describe, expect, test } from 'bun:test';
import { CORRIDOR, VOID, assertBoardShape, deriveWalls, regionAt, tileAt, tileIndex } from './model.ts';

describe('regionAt / tileIndex / tileAt', () => {
  // 3x2, bottom row first: y=1 is "void corridor 1", y=2 is "corridor corridor 2".
  const regions = [
    VOID, CORRIDOR, 1,
    CORRIDOR, CORRIDOR, 2,
  ];

  test('indexes row-major from the bottom-left square (1,1)', () => {
    expect(tileIndex(3, { x: 1, y: 1 })).toBe(0);
    expect(tileIndex(3, { x: 3, y: 1 })).toBe(2);
    expect(tileIndex(3, { x: 1, y: 2 })).toBe(3);
    expect(tileIndex(3, { x: 3, y: 2 })).toBe(5);
  });

  test('tileAt inverts tileIndex', () => {
    for (let i = 0; i < 6; i++) {
      expect(tileIndex(3, tileAt(3, i))).toBe(i);
    }
    expect(tileAt(3, 0)).toEqual({ x: 1, y: 1 });
    expect(tileAt(3, 4)).toEqual({ x: 2, y: 2 });
  });

  test('reads regions', () => {
    expect(regionAt(3, 2, regions, { x: 1, y: 1 })).toBe(VOID);
    expect(regionAt(3, 2, regions, { x: 3, y: 1 })).toBe(1);
    expect(regionAt(3, 2, regions, { x: 3, y: 2 })).toBe(2);
  });

  test('treats off-board squares (including row or column 0) as void', () => {
    expect(regionAt(3, 2, regions, { x: 0, y: 1 })).toBe(VOID);
    expect(regionAt(3, 2, regions, { x: 4, y: 1 })).toBe(VOID);
    expect(regionAt(3, 2, regions, { x: 2, y: 0 })).toBe(VOID);
    expect(regionAt(3, 2, regions, { x: 2, y: 3 })).toBe(VOID);
  });
});

describe('assertBoardShape', () => {
  test('accepts a regions array matching the board size', () => {
    expect(() => { assertBoardShape(3, 2, [0, 0, 0, 0, 0, 0]); }).not.toThrow();
  });

  test('rejects mismatched lengths and invalid region ids', () => {
    expect(() => { assertBoardShape(3, 2, [0, 0, 0]); }).toThrow(RangeError);
    expect(() => { assertBoardShape(1, 1, [-2]); }).toThrow(RangeError);
    expect(() => { assertBoardShape(1, 1, [1.5]); }).toThrow(RangeError);
  });
});

describe('deriveWalls', () => {
  test('a single corridor square is enclosed on all four sides', () => {
    expect(deriveWalls(1, 1, [CORRIDOR])).toEqual([
      { x: 1, y: 1, orientation: 'vertical' },
      { x: 2, y: 1, orientation: 'vertical' },
      { x: 1, y: 1, orientation: 'horizontal' },
      { x: 1, y: 2, orientation: 'horizontal' },
    ]);
  });

  test('squares of the same region share no wall', () => {
    const walls = deriveWalls(2, 1, [CORRIDOR, CORRIDOR]);
    expect(walls).not.toContainEqual({ x: 2, y: 1, orientation: 'vertical' });
    expect(walls).toHaveLength(6); // 2 side walls + 2 top + 2 bottom
  });

  test('a wall separates a room from the corridor and rooms from each other', () => {
    const walls = deriveWalls(3, 1, [CORRIDOR, 1, 2]);
    expect(walls).toContainEqual({ x: 2, y: 1, orientation: 'vertical' });
    expect(walls).toContainEqual({ x: 3, y: 1, orientation: 'vertical' });
  });

  test('a wall between rows sits on the bottom side of the upper square', () => {
    // 1x2: corridor at y=1, room at y=2.
    expect(deriveWalls(1, 2, [CORRIDOR, 1])).toContainEqual({ x: 1, y: 2, orientation: 'horizontal' });
  });

  test('void squares get no walls of their own; walls appear only where they meet open squares', () => {
    const walls = deriveWalls(3, 1, [VOID, VOID, CORRIDOR]);
    expect(walls).toEqual([
      { x: 3, y: 1, orientation: 'vertical' },
      { x: 4, y: 1, orientation: 'vertical' },
      { x: 3, y: 1, orientation: 'horizontal' },
      { x: 3, y: 2, orientation: 'horizontal' },
    ]);
  });

  test('an all-void board has no walls', () => {
    expect(deriveWalls(4, 3, new Array<number>(12).fill(VOID))).toEqual([]);
  });

  test('works for a large landscape board (30x24) and counts perimeter walls', () => {
    const cols = 30;
    const rows = 24;
    const walls = deriveWalls(cols, rows, new Array<number>(cols * rows).fill(CORRIDOR));
    // Open corridor everywhere: only the outer perimeter is walled.
    expect(walls).toHaveLength(2 * cols + 2 * rows);
  });
});
