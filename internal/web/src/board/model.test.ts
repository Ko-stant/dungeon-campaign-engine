import { describe, expect, test } from 'bun:test';
import { CORRIDOR, VOID, assertBoardShape, deriveWalls, regionAt, tileIndex } from './model.ts';

describe('regionAt / tileIndex', () => {
  const regions = [
    VOID, CORRIDOR, 1,
    CORRIDOR, CORRIDOR, 2,
  ];

  test('reads row-major regions', () => {
    expect(tileIndex(3, { x: 2, y: 1 })).toBe(5);
    expect(regionAt(3, 2, regions, { x: 0, y: 0 })).toBe(VOID);
    expect(regionAt(3, 2, regions, { x: 2, y: 0 })).toBe(1);
    expect(regionAt(3, 2, regions, { x: 2, y: 1 })).toBe(2);
  });

  test('treats off-board tiles as void', () => {
    expect(regionAt(3, 2, regions, { x: -1, y: 0 })).toBe(VOID);
    expect(regionAt(3, 2, regions, { x: 3, y: 0 })).toBe(VOID);
    expect(regionAt(3, 2, regions, { x: 0, y: 2 })).toBe(VOID);
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
  test('a single corridor tile is enclosed on all four sides', () => {
    expect(deriveWalls(1, 1, [CORRIDOR])).toEqual([
      { x: 0, y: 0, orientation: 'vertical' },
      { x: 1, y: 0, orientation: 'vertical' },
      { x: 0, y: 0, orientation: 'horizontal' },
      { x: 0, y: 1, orientation: 'horizontal' },
    ]);
  });

  test('tiles of the same region share no wall', () => {
    const walls = deriveWalls(2, 1, [CORRIDOR, CORRIDOR]);
    expect(walls).not.toContainEqual({ x: 1, y: 0, orientation: 'vertical' });
    expect(walls).toHaveLength(6); // 2 side walls + 2 top + 2 bottom
  });

  test('a wall separates a room from the corridor and rooms from each other', () => {
    const walls = deriveWalls(3, 1, [CORRIDOR, 1, 2]);
    expect(walls).toContainEqual({ x: 1, y: 0, orientation: 'vertical' });
    expect(walls).toContainEqual({ x: 2, y: 0, orientation: 'vertical' });
  });

  test('void tiles get no walls of their own; walls appear only where they meet open tiles', () => {
    // Row: void, void, corridor
    const walls = deriveWalls(3, 1, [VOID, VOID, CORRIDOR]);
    expect(walls).toEqual([
      { x: 2, y: 0, orientation: 'vertical' },
      { x: 3, y: 0, orientation: 'vertical' },
      { x: 2, y: 0, orientation: 'horizontal' },
      { x: 2, y: 1, orientation: 'horizontal' },
    ]);
  });

  test('an all-void board has no walls', () => {
    expect(deriveWalls(4, 3, new Array<number>(12).fill(VOID))).toEqual([]);
  });

  test('works for a large custom board (24x30) and counts perimeter walls', () => {
    const cols = 24;
    const rows = 30;
    const walls = deriveWalls(cols, rows, new Array<number>(cols * rows).fill(CORRIDOR));
    // Open corridor everywhere: only the outer perimeter is walled.
    expect(walls).toHaveLength(2 * cols + 2 * rows);
  });
});
