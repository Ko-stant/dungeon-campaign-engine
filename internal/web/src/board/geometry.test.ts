import { describe, expect, test } from 'bun:test';
import {
  computeGridMetrics,
  doorRect,
  furnitureDrawBox,
  edgeBetween,
  edgeTiles,
  footprintTiles,
  isInteriorEdge,
  pixelToEdge,
  pixelToTile,
  rotatedFootprint,
  tileRect,
} from './geometry.ts';

describe('computeGridMetrics', () => {
  test('fits square tiles to the limiting dimension and centres the grid', () => {
    // 26x19 board in an 800x600 view: width limits (800/26 = 30.7 -> 30).
    const m = computeGridMetrics(800, 600, 26, 19);
    expect(m.tile).toBe(30);
    expect(m.cols).toBe(26);
    expect(m.rows).toBe(19);
    expect(m.originX).toBe(10); // (800 - 780) / 2
    expect(m.originY).toBe(15); // (600 - 570) / 2
  });

  test('handles a tall custom board (24x30) where height limits', () => {
    const m = computeGridMetrics(800, 600, 24, 30);
    expect(m.tile).toBe(20); // min(33.3, 20)
    expect(m.originX).toBe(160); // (800 - 480) / 2
    expect(m.originY).toBe(0);
  });

  test('never produces a tile smaller than 1px', () => {
    const m = computeGridMetrics(10, 10, 100, 100);
    expect(m.tile).toBe(1);
  });

  test('rejects non-positive or fractional board sizes', () => {
    expect(() => computeGridMetrics(800, 600, 0, 19)).toThrow(RangeError);
    expect(() => computeGridMetrics(800, 600, 26, -1)).toThrow(RangeError);
    expect(() => computeGridMetrics(800, 600, 2.5, 19)).toThrow(RangeError);
  });
});

describe('tileRect', () => {
  test('returns the pixel rectangle and centre of a tile', () => {
    const m = computeGridMetrics(800, 600, 26, 19);
    expect(tileRect(m, { x: 0, y: 0 })).toEqual({ x: 10, y: 15, w: 30, h: 30, cx: 25, cy: 30 });
    expect(tileRect(m, { x: 3, y: 2 })).toEqual({ x: 100, y: 75, w: 30, h: 30, cx: 115, cy: 90 });
  });
});

describe('pixelToTile', () => {
  const m = computeGridMetrics(800, 600, 26, 19); // tile 30, origin (10, 15)

  test('maps a pixel inside the grid to its tile', () => {
    expect(pixelToTile(m, 10, 15)).toEqual({ x: 0, y: 0 });
    expect(pixelToTile(m, 39.9, 44.9)).toEqual({ x: 0, y: 0 });
    expect(pixelToTile(m, 40, 45)).toEqual({ x: 1, y: 1 });
    expect(pixelToTile(m, 789, 584)).toEqual({ x: 25, y: 18 });
  });

  test('returns null outside the grid', () => {
    expect(pixelToTile(m, 9, 20)).toBeNull();
    expect(pixelToTile(m, 20, 14)).toBeNull();
    expect(pixelToTile(m, 790, 20)).toBeNull(); // right edge is exclusive
    expect(pixelToTile(m, 20, 585)).toBeNull();
  });
});

describe('pixelToEdge', () => {
  const m = computeGridMetrics(800, 600, 26, 19); // tile 30, origin (10, 15)

  test('picks the vertical edge (left side of the tile) near a vertical grid line', () => {
    // Grid line x=3 is at pixel 100. Tile row 2 spans y 75..105.
    expect(pixelToEdge(m, 102, 90)).toEqual({ x: 3, y: 2, orientation: 'vertical' });
    expect(pixelToEdge(m, 98, 90)).toEqual({ x: 3, y: 2, orientation: 'vertical' });
  });

  test('picks the horizontal edge (top side of the tile) near a horizontal grid line', () => {
    // Grid line y=2 is at pixel 75. Tile column 3 spans x 100..130.
    expect(pixelToEdge(m, 115, 77)).toEqual({ x: 3, y: 2, orientation: 'horizontal' });
  });

  test('returns null in the middle of a tile', () => {
    expect(pixelToEdge(m, 115, 90)).toBeNull();
  });

  test('prefers the closer line near a corner', () => {
    // Near the corner of grid lines x=3 (px 100) and y=2 (px 75).
    expect(pixelToEdge(m, 101, 79)).toEqual({ x: 3, y: 2, orientation: 'vertical' });
    expect(pixelToEdge(m, 104, 76)).toEqual({ x: 3, y: 2, orientation: 'horizontal' });
  });

  test('can return outer boundary edges', () => {
    expect(pixelToEdge(m, 11, 90)).toEqual({ x: 0, y: 2, orientation: 'vertical' });
    expect(pixelToEdge(m, 789, 90)).toEqual({ x: 26, y: 2, orientation: 'vertical' });
    expect(pixelToEdge(m, 115, 584)).toEqual({ x: 3, y: 19, orientation: 'horizontal' });
  });

  test('returns null outside the grid', () => {
    expect(pixelToEdge(m, 5, 90)).toBeNull();
    expect(pixelToEdge(m, 115, 590)).toBeNull();
  });
});

describe('edges and tiles', () => {
  test('edgeTiles returns the tiles on either side (left/up first)', () => {
    expect(edgeTiles({ x: 3, y: 2, orientation: 'vertical' })).toEqual([{ x: 2, y: 2 }, { x: 3, y: 2 }]);
    expect(edgeTiles({ x: 3, y: 2, orientation: 'horizontal' })).toEqual([{ x: 3, y: 1 }, { x: 3, y: 2 }]);
  });

  test('edgeBetween finds the shared edge of orthogonal neighbours in either order', () => {
    expect(edgeBetween({ x: 2, y: 2 }, { x: 3, y: 2 })).toEqual({ x: 3, y: 2, orientation: 'vertical' });
    expect(edgeBetween({ x: 3, y: 2 }, { x: 2, y: 2 })).toEqual({ x: 3, y: 2, orientation: 'vertical' });
    expect(edgeBetween({ x: 3, y: 1 }, { x: 3, y: 2 })).toEqual({ x: 3, y: 2, orientation: 'horizontal' });
    expect(edgeBetween({ x: 3, y: 2 }, { x: 3, y: 1 })).toEqual({ x: 3, y: 2, orientation: 'horizontal' });
  });

  test('edgeBetween returns null for non-adjacent or diagonal tiles', () => {
    expect(edgeBetween({ x: 2, y: 2 }, { x: 4, y: 2 })).toBeNull();
    expect(edgeBetween({ x: 2, y: 2 }, { x: 3, y: 3 })).toBeNull();
    expect(edgeBetween({ x: 2, y: 2 }, { x: 2, y: 2 })).toBeNull();
  });

  test('isInteriorEdge excludes the board boundary', () => {
    expect(isInteriorEdge({ x: 3, y: 2, orientation: 'vertical' }, 26, 19)).toBe(true);
    expect(isInteriorEdge({ x: 0, y: 2, orientation: 'vertical' }, 26, 19)).toBe(false);
    expect(isInteriorEdge({ x: 26, y: 2, orientation: 'vertical' }, 26, 19)).toBe(false);
    expect(isInteriorEdge({ x: 3, y: 19, orientation: 'vertical' }, 26, 19)).toBe(false);
    expect(isInteriorEdge({ x: 3, y: 0, orientation: 'horizontal' }, 26, 19)).toBe(false);
    expect(isInteriorEdge({ x: 3, y: 18, orientation: 'horizontal' }, 26, 19)).toBe(true);
    expect(isInteriorEdge({ x: 26, y: 5, orientation: 'horizontal' }, 26, 19)).toBe(false);
  });
});

describe('furniture footprints', () => {
  test('rotatedFootprint swaps width and height at 90 and 270 degrees', () => {
    expect(rotatedFootprint(3, 2, 0)).toEqual({ width: 3, height: 2 });
    expect(rotatedFootprint(3, 2, 90)).toEqual({ width: 2, height: 3 });
    expect(rotatedFootprint(3, 2, 180)).toEqual({ width: 3, height: 2 });
    expect(rotatedFootprint(3, 2, 270)).toEqual({ width: 2, height: 3 });
  });

  test('rotatedFootprint rejects rotations that are not quarter turns', () => {
    expect(() => rotatedFootprint(3, 2, 45)).toThrow(RangeError);
    expect(() => rotatedFootprint(3, 2, 360)).toThrow(RangeError);
  });

  test('footprintTiles lists every covered tile, row by row', () => {
    expect(footprintTiles({ x: 5, y: 7 }, 3, 2, 0)).toEqual([
      { x: 5, y: 7 }, { x: 6, y: 7 }, { x: 7, y: 7 },
      { x: 5, y: 8 }, { x: 6, y: 8 }, { x: 7, y: 8 },
    ]);
    expect(footprintTiles({ x: 5, y: 7 }, 3, 2, 90)).toEqual([
      { x: 5, y: 7 }, { x: 6, y: 7 },
      { x: 5, y: 8 }, { x: 6, y: 8 },
      { x: 5, y: 9 }, { x: 6, y: 9 },
    ]);
  });
});

describe('furnitureDrawBox', () => {
  test('unrotated pieces are centred on their footprint at natural size', () => {
    expect(furnitureDrawBox({ x: 6, y: 5 }, 3, 2, 0)).toEqual({ centerX: 7.5, centerY: 6, width: 3, height: 2, radians: 0 });
  });

  test('quarter-turned pieces centre on the swapped footprint and keep their natural size', () => {
    // Alchemist's bench (3x2) at (5,15) rotated 270: footprint is 2 wide, 3 tall.
    const box = furnitureDrawBox({ x: 5, y: 15 }, 3, 2, 270);
    expect(box.centerX).toBe(6);
    expect(box.centerY).toBe(16.5);
    expect(box.width).toBe(3);
    expect(box.height).toBe(2);
    expect(box.radians).toBeCloseTo((3 * Math.PI) / 2);
  });

  test('half-turned pieces keep their footprint', () => {
    const box = furnitureDrawBox({ x: 5, y: 13 }, 3, 1, 180);
    expect([box.centerX, box.centerY, box.width, box.height]).toEqual([6.5, 13.5, 3, 1]);
    expect(box.radians).toBeCloseTo(Math.PI);
  });
});

describe('doorRect', () => {
  const m = computeGridMetrics(800, 600, 26, 19); // tile 30, origin (10, 15)

  test('a vertical door is a thin bar centred on the grid line, inset from the tile corners', () => {
    // Edge x=3 at pixel 100, row 2 spans y 75..105. Thickness 30*0.2=6, inset 30*0.15=4.5.
    expect(doorRect(m, { x: 3, y: 2, orientation: 'vertical' })).toEqual({ x: 97, y: 79.5, w: 6, h: 21 });
  });

  test('a horizontal door is a thin bar across the grid line', () => {
    expect(doorRect(m, { x: 3, y: 2, orientation: 'horizontal' })).toEqual({ x: 104.5, y: 72, w: 21, h: 6 });
  });
});
