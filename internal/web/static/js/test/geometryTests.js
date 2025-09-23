/**
 * Tests for geometry utilities
 */

import { createTestSuite, assert } from './testFramework.js';
import {
  calculateTileRect,
  keyToStep,
  calculateDoorEdgeFromDirection,
  calculateAdjacentTile,
  calculateRotationPositioning,
  calculateRotationCenter,
  degreesToRadians,
} from '../geometry.js';

const suite = createTestSuite('Geometry Utils');

// Test calculateTileRect
suite.test('calculateTileRect - basic positioning', () => {
  const rect = calculateTileRect(2, 3, 32, 10, 20);

  assert.equals(rect.x, 74, 'X position should be offsetX + x * tileSize');
  assert.equals(rect.y, 116, 'Y position should be offsetY + y * tileSize');
  assert.equals(rect.w, 32, 'Width should equal tileSize');
  assert.equals(rect.h, 32, 'Height should equal tileSize');
  assert.equals(rect.cx, 90, 'Center X should be x + width/2');
  assert.equals(rect.cy, 132, 'Center Y should be y + height/2');
});

suite.test('calculateTileRect - origin tile', () => {
  const rect = calculateTileRect(0, 0, 16, 0, 0);

  assert.equals(rect.x, 0, 'Origin tile X should be 0');
  assert.equals(rect.y, 0, 'Origin tile Y should be 0');
  assert.equals(rect.cx, 8, 'Center X should be half tile size');
  assert.equals(rect.cy, 8, 'Center Y should be half tile size');
});

// Test keyToStep
suite.test('keyToStep - arrow keys', () => {
  const up = keyToStep({ key: 'ArrowUp' });
  const down = keyToStep({ key: 'ArrowDown' });
  const left = keyToStep({ key: 'ArrowLeft' });
  const right = keyToStep({ key: 'ArrowRight' });

  assert.deepEquals(up, { dx: 0, dy: -1 }, 'Up arrow should move up');
  assert.deepEquals(down, { dx: 0, dy: 1 }, 'Down arrow should move down');
  assert.deepEquals(left, { dx: -1, dy: 0 }, 'Left arrow should move left');
  assert.deepEquals(right, { dx: 1, dy: 0 }, 'Right arrow should move right');
});

suite.test('keyToStep - WASD keys', () => {
  const w = keyToStep({ key: 'w' });
  const s = keyToStep({ key: 's' });
  const a = keyToStep({ key: 'a' });
  const d = keyToStep({ key: 'd' });

  assert.deepEquals(w, { dx: 0, dy: -1 }, 'W should move up');
  assert.deepEquals(s, { dx: 0, dy: 1 }, 'S should move down');
  assert.deepEquals(a, { dx: -1, dy: 0 }, 'A should move left');
  assert.deepEquals(d, { dx: 1, dy: 0 }, 'D should move right');
});

suite.test('keyToStep - uppercase keys', () => {
  const W = keyToStep({ key: 'W' });
  const A = keyToStep({ key: 'A' });

  assert.deepEquals(W, { dx: 0, dy: -1 }, 'Uppercase W should work');
  assert.deepEquals(A, { dx: -1, dy: 0 }, 'Uppercase A should work');
});

suite.test('keyToStep - invalid keys', () => {
  const invalid = keyToStep({ key: 'x' });
  const space = keyToStep({ key: ' ' });

  assert.isNull(invalid, 'Invalid key should return null');
  assert.isNull(space, 'Space key should return null');
});

// Test calculateDoorEdgeFromDirection
suite.test('calculateDoorEdgeFromDirection - all directions', () => {
  const pos = { x: 5, y: 3 };

  const right = calculateDoorEdgeFromDirection(pos, 1, 0);
  const left = calculateDoorEdgeFromDirection(pos, -1, 0);
  const down = calculateDoorEdgeFromDirection(pos, 0, 1);
  const up = calculateDoorEdgeFromDirection(pos, 0, -1);

  assert.deepEquals(
    right,
    { x: 6, y: 3, orientation: 'vertical' },
    'Right should give vertical door at x+1',
  );
  assert.deepEquals(
    left,
    { x: 5, y: 3, orientation: 'vertical' },
    'Left should give vertical door at x',
  );
  assert.deepEquals(
    down,
    { x: 5, y: 4, orientation: 'horizontal' },
    'Down should give horizontal door at y+1',
  );
  assert.deepEquals(
    up,
    { x: 5, y: 3, orientation: 'horizontal' },
    'Up should give horizontal door at y',
  );
});

suite.test('calculateDoorEdgeFromDirection - null position', () => {
  const result = calculateDoorEdgeFromDirection(null, 1, 0);
  assert.isNull(result, 'Null position should return null');
});

suite.test('calculateDoorEdgeFromDirection - invalid direction', () => {
  const pos = { x: 5, y: 3 };
  const result = calculateDoorEdgeFromDirection(pos, 1, 1); // diagonal
  assert.isNull(result, 'Diagonal direction should return null');
});

// Test calculateAdjacentTile
suite.test('calculateAdjacentTile - basic movement', () => {
  const pos = { x: 2, y: 3 };

  const right = calculateAdjacentTile(pos, 1, 0);
  const left = calculateAdjacentTile(pos, -1, 0);
  const down = calculateAdjacentTile(pos, 0, 1);
  const up = calculateAdjacentTile(pos, 0, -1);

  assert.deepEquals(right, { x: 3, y: 3 }, 'Right movement');
  assert.deepEquals(left, { x: 1, y: 3 }, 'Left movement');
  assert.deepEquals(down, { x: 2, y: 4 }, 'Down movement');
  assert.deepEquals(up, { x: 2, y: 2 }, 'Up movement');
});

suite.test('calculateAdjacentTile - null position', () => {
  const result = calculateAdjacentTile(null, 1, 0);
  assert.isNull(result, 'Null position should return null');
});

// Test calculateRotationPositioning
suite.test('calculateRotationPositioning - no rotation', () => {
  const result = calculateRotationPositioning(1, 2, 3, 4, 0, false);

  assert.equals(result.renderStartX, 1, 'No rotation should preserve X');
  assert.equals(result.renderStartY, 2, 'No rotation should preserve Y');
  assert.equals(result.renderWidth, 3, 'No rotation should preserve width');
  assert.equals(result.renderHeight, 4, 'No rotation should preserve height');
});

suite.test('calculateRotationPositioning - with aspect swap', () => {
  const result = calculateRotationPositioning(5, 6, 2, 4, 90, true);

  // For 90 degree rotation with aspect swap:
  // swappedWidth = 4, swappedHeight = 2
  // widthOffset = (4 - 2) / 2 = 1
  // heightOffset = (2 - 4) / 2 = -1
  assert.equals(result.renderStartX, 6, 'Aspect swap should adjust X position');
  assert.equals(result.renderStartY, 5, 'Aspect swap should adjust Y position');
  assert.equals(result.renderWidth, 2, 'Render width should be original width');
  assert.equals(
    result.renderHeight,
    4,
    'Render height should be original height',
  );
});

// Test calculateRotationCenter
suite.test('calculateRotationCenter - basic calculation', () => {
  const result = calculateRotationCenter(2, 3, 4, 6, 16, 10, 20);

  const expectedCenterX = 2 + 4 / 2; // 4
  const expectedCenterY = 3 + 6 / 2; // 6
  const expectedPixelX = 4 * 16 + 10; // 74
  const expectedPixelY = 6 * 16 + 20; // 116

  assert.equals(result.centerX, expectedCenterX, 'Center X calculation');
  assert.equals(result.centerY, expectedCenterY, 'Center Y calculation');
  assert.equals(
    result.centerPixelX,
    expectedPixelX,
    'Pixel center X calculation',
  );
  assert.equals(
    result.centerPixelY,
    expectedPixelY,
    'Pixel center Y calculation',
  );
});

// Test degreesToRadians
suite.test('degreesToRadians - common angles', () => {
  assert.equals(degreesToRadians(0), 0, '0 degrees = 0 radians');
  assert.equals(degreesToRadians(90), Math.PI / 2, '90 degrees = π/2 radians');
  assert.equals(degreesToRadians(180), Math.PI, '180 degrees = π radians');
  assert.equals(
    degreesToRadians(270),
    (3 * Math.PI) / 2,
    '270 degrees = 3π/2 radians',
  );
  assert.equals(degreesToRadians(360), 2 * Math.PI, '360 degrees = 2π radians');
});

suite.test('degreesToRadians - negative angles', () => {
  assert.equals(
    degreesToRadians(-90),
    -Math.PI / 2,
    '-90 degrees = -π/2 radians',
  );
  assert.equals(degreesToRadians(-180), -Math.PI, '-180 degrees = -π radians');
});

export default suite;
