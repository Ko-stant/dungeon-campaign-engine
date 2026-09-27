import { describe, expect, test } from 'bun:test';
import {
  computeGridMetrics,
  doorRect,
  edgeBetween,
  edgeSegment,
  edgeTiles,
  footprintRect,
  footprintTiles,
  furnitureDrawBox,
  isInteriorEdge,
  pixelToEdge,
  pixelToTile,
  rotatedFootprint,
  tileRect,
} from './geometry.ts';

// Squares count from (1,1) at the bottom-left. In an 800x600 view a 26x19
// board has 30px tiles with the grid's top-left pixel at (10, 15), so the
// bottom row (y=1) spans pixels 555..585 and the top row (y=19) spans 15..45.

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

  test('handles a large landscape custom board (30x24) where height limits', () => {
    const m = computeGridMetrics(800, 600, 30, 24);
    expect(m.tile).toBe(25); // min(26.7, 25)
    expect(m.originX).toBe(25); // (800 - 750) / 2
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
  const m = computeGridMetrics(800, 600, 26, 19);

  test('(1,1) is the bottom-left square', () => {
    expect(tileRect(m, { x: 1, y: 1 })).toEqual({ x: 10, y: 555, w: 30, h: 30, cx: 25, cy: 570 });
  });

  test('rows count upwards and columns rightwards', () => {
    expect(tileRect(m, { x: 1, y: 19 })).toEqual({ x: 10, y: 15, w: 30, h: 30, cx: 25, cy: 30 });
    expect(tileRect(m, { x: 4, y: 17 })).toEqual({ x: 100, y: 75, w: 30, h: 30, cx: 115, cy: 90 });
    expect(tileRect(m, { x: 26, y: 1 })).toEqual({ x: 760, y: 555, w: 30, h: 30, cx: 775, cy: 570 });
  });
});

describe('footprintRect', () => {
  const m = computeGridMetrics(800, 600, 26, 19);

  test('covers a block anchored at its bottom-left square and extending right and up', () => {
    // Squares x 7..9, y 14..15: left pixel 190, top of row 15 at pixel 135.
    expect(footprintRect(m, { x: 7, y: 14 }, 3, 2)).toEqual({ x: 190, y: 135, w: 90, h: 60, cx: 235, cy: 165 });
  });

  test('a 1x1 footprint is the tile itself', () => {
    expect(footprintRect(m, { x: 4, y: 17 }, 1, 1)).toEqual(tileRect(m, { x: 4, y: 17 }));
  });
});

describe('pixelToTile', () => {
  const m = computeGridMetrics(800, 600, 26, 19);

  test('maps a pixel inside the grid to its square', () => {
    expect(pixelToTile(m, 10, 15)).toEqual({ x: 1, y: 19 }); // top-left pixel
    expect(pixelToTile(m, 39.9, 44.9)).toEqual({ x: 1, y: 19 });
    expect(pixelToTile(m, 40, 45)).toEqual({ x: 2, y: 18 });
    expect(pixelToTile(m, 10, 584)).toEqual({ x: 1, y: 1 }); // bottom-left
    expect(pixelToTile(m, 789, 584)).toEqual({ x: 26, y: 1 });
  });

  test('round-trips with tileRect', () => {
    for (const t of [{ x: 1, y: 1 }, { x: 4, y: 17 }, { x: 26, y: 19 }]) {
      const r = tileRect(m, t);
      expect(pixelToTile(m, r.cx, r.cy)).toEqual(t);
    }
  });

  test('returns null outside the grid', () => {
    expect(pixelToTile(m, 9, 20)).toBeNull();
    expect(pixelToTile(m, 20, 14)).toBeNull();
    expect(pixelToTile(m, 790, 20)).toBeNull(); // right edge is exclusive
    expect(pixelToTile(m, 20, 585)).toBeNull(); // bottom edge is exclusive
  });
});

describe('pixelToEdge', () => {
  const m = computeGridMetrics(800, 600, 26, 19);

  test('picks the vertical edge (left side of the square) near a vertical grid line', () => {
    // The line left of column 4 is at pixel 100. Row 17 spans y 75..105.
    expect(pixelToEdge(m, 102, 90)).toEqual({ x: 4, y: 17, orientation: 'vertical' });
    expect(pixelToEdge(m, 98, 90)).toEqual({ x: 4, y: 17, orientation: 'vertical' });
  });

  test('picks the horizontal edge (bottom side of the square) near a horizontal grid line', () => {
    // Pixel 75 is the bottom of row 18 (and the top of row 17). Column 4 spans x 100..130.
    expect(pixelToEdge(m, 115, 77)).toEqual({ x: 4, y: 18, orientation: 'horizontal' });
    expect(pixelToEdge(m, 115, 73)).toEqual({ x: 4, y: 18, orientation: 'horizontal' });
  });

  test('returns null in the middle of a square', () => {
    expect(pixelToEdge(m, 115, 90)).toBeNull();
  });

  test('prefers the closer line near a corner', () => {
    expect(pixelToEdge(m, 101, 79)).toEqual({ x: 4, y: 17, orientation: 'vertical' });
    expect(pixelToEdge(m, 104, 76)).toEqual({ x: 4, y: 18, orientation: 'horizontal' });
  });

  test('can return outer boundary edges', () => {
    expect(pixelToEdge(m, 11, 90)).toEqual({ x: 1, y: 17, orientation: 'vertical' });
    expect(pixelToEdge(m, 789, 90)).toEqual({ x: 27, y: 17, orientation: 'vertical' });
    expect(pixelToEdge(m, 115, 584)).toEqual({ x: 4, y: 1, orientation: 'horizontal' }); // bottom boundary
    expect(pixelToEdge(m, 115, 16)).toEqual({ x: 4, y: 20, orientation: 'horizontal' }); // top boundary
  });

  test('accepts a wider hit tolerance when asked', () => {
    // 8px from the line at pixel 100 is outside the default 20% (6px) but inside 35% (10.5px).
    expect(pixelToEdge(m, 108, 90)).toBeNull();
    expect(pixelToEdge(m, 108, 90, 0.35)).toEqual({ x: 4, y: 17, orientation: 'vertical' });
  });

  test('returns null outside the grid', () => {
    expect(pixelToEdge(m, 5, 90)).toBeNull();
    expect(pixelToEdge(m, 115, 590)).toBeNull();
  });
});

describe('edges and tiles', () => {
  test('edgeTiles returns the squares on either side (left or below first)', () => {
    expect(edgeTiles({ x: 4, y: 17, orientation: 'vertical' })).toEqual([{ x: 3, y: 17 }, { x: 4, y: 17 }]);
    expect(edgeTiles({ x: 4, y: 18, orientation: 'horizontal' })).toEqual([{ x: 4, y: 17 }, { x: 4, y: 18 }]);
  });

  test('edgeBetween finds the shared edge of orthogonal neighbours in either order', () => {
    expect(edgeBetween({ x: 3, y: 17 }, { x: 4, y: 17 })).toEqual({ x: 4, y: 17, orientation: 'vertical' });
    expect(edgeBetween({ x: 4, y: 17 }, { x: 3, y: 17 })).toEqual({ x: 4, y: 17, orientation: 'vertical' });
    expect(edgeBetween({ x: 4, y: 17 }, { x: 4, y: 18 })).toEqual({ x: 4, y: 18, orientation: 'horizontal' });
    expect(edgeBetween({ x: 4, y: 18 }, { x: 4, y: 17 })).toEqual({ x: 4, y: 18, orientation: 'horizontal' });
  });

  test('edgeBetween returns null for non-adjacent or diagonal squares', () => {
    expect(edgeBetween({ x: 2, y: 2 }, { x: 4, y: 2 })).toBeNull();
    expect(edgeBetween({ x: 2, y: 2 }, { x: 3, y: 3 })).toBeNull();
    expect(edgeBetween({ x: 2, y: 2 }, { x: 2, y: 2 })).toBeNull();
  });

  test('isInteriorEdge excludes the board boundary', () => {
    expect(isInteriorEdge({ x: 4, y: 17, orientation: 'vertical' }, 26, 19)).toBe(true);
    expect(isInteriorEdge({ x: 1, y: 17, orientation: 'vertical' }, 26, 19)).toBe(false);
    expect(isInteriorEdge({ x: 27, y: 17, orientation: 'vertical' }, 26, 19)).toBe(false);
    expect(isInteriorEdge({ x: 26, y: 17, orientation: 'vertical' }, 26, 19)).toBe(true);
    expect(isInteriorEdge({ x: 4, y: 20, orientation: 'vertical' }, 26, 19)).toBe(false);
    expect(isInteriorEdge({ x: 4, y: 0, orientation: 'vertical' }, 26, 19)).toBe(false);
    expect(isInteriorEdge({ x: 4, y: 1, orientation: 'horizontal' }, 26, 19)).toBe(false);
    expect(isInteriorEdge({ x: 4, y: 2, orientation: 'horizontal' }, 26, 19)).toBe(true);
    expect(isInteriorEdge({ x: 4, y: 19, orientation: 'horizontal' }, 26, 19)).toBe(true);
    expect(isInteriorEdge({ x: 4, y: 20, orientation: 'horizontal' }, 26, 19)).toBe(false);
    expect(isInteriorEdge({ x: 27, y: 5, orientation: 'horizontal' }, 26, 19)).toBe(false);
  });
});

describe('edgeSegment', () => {
  const m = computeGridMetrics(800, 600, 26, 19);

  test('a vertical edge runs down the left side of its square', () => {
    expect(edgeSegment(m, { x: 4, y: 17, orientation: 'vertical' })).toEqual({ x1: 100, y1: 75, x2: 100, y2: 105 });
  });

  test('a horizontal edge runs along the bottom side of its square', () => {
    expect(edgeSegment(m, { x: 4, y: 18, orientation: 'horizontal' })).toEqual({ x1: 100, y1: 75, x2: 130, y2: 75 });
    expect(edgeSegment(m, { x: 1, y: 1, orientation: 'horizontal' })).toEqual({ x1: 10, y1: 585, x2: 40, y2: 585 });
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

  test('footprintTiles lists every covered square from the bottom-left anchor, row by row upwards', () => {
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
  const m = computeGridMetrics(800, 600, 26, 19);

  test('unrotated pieces are centred on their footprint at natural size', () => {
    expect(furnitureDrawBox(m, { x: 7, y: 14 }, 3, 2, 0)).toEqual({ cx: 235, cy: 165, width: 90, height: 60, radians: 0 });
  });

  test('quarter-turned pieces centre on the swapped footprint and keep their natural size', () => {
    // Alchemist's bench (3x2) at (6,3) rotated 270: footprint x 6..7, y 3..5.
    const box = furnitureDrawBox(m, { x: 6, y: 3 }, 3, 2, 270);
    expect(box.cx).toBe(190);
    expect(box.cy).toBe(480);
    expect(box.width).toBe(90);
    expect(box.height).toBe(60);
    expect(box.radians).toBeCloseTo((3 * Math.PI) / 2);
  });

  test('half-turned pieces keep their footprint', () => {
    const box = furnitureDrawBox(m, { x: 6, y: 6 }, 3, 1, 180);
    expect([box.cx, box.cy, box.width, box.height]).toEqual([205, 420, 90, 30]);
    expect(box.radians).toBeCloseTo(Math.PI);
  });
});

describe('doorRect', () => {
  const m = computeGridMetrics(800, 600, 26, 19);

  test('a vertical door is a thin bar centred on the grid line, inset from the square corners', () => {
    // Line at pixel 100, row 17 spans y 75..105. Thickness 30*0.2=6, inset 30*0.15=4.5.
    expect(doorRect(m, { x: 4, y: 17, orientation: 'vertical' })).toEqual({ x: 97, y: 79.5, w: 6, h: 21 });
  });

  test('a horizontal door is a thin bar across the bottom line of its square', () => {
    expect(doorRect(m, { x: 4, y: 18, orientation: 'horizontal' })).toEqual({ x: 104.5, y: 72, w: 21, h: 6 });
  });
});
