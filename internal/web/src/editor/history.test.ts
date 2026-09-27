import { describe, expect, test } from 'bun:test';
import { History } from './history.ts';

describe('History', () => {
  test('starts at the initial state with nothing to undo or redo', () => {
    const h = new History('a');
    expect(h.current).toBe('a');
    expect(h.canUndo).toBe(false);
    expect(h.canRedo).toBe(false);
    expect(h.undo()).toBe('a');
  });

  test('undo and redo walk the stack', () => {
    const h = new History(1);
    h.push(2);
    h.push(3);
    expect(h.undo()).toBe(2);
    expect(h.undo()).toBe(1);
    expect(h.canUndo).toBe(false);
    expect(h.redo()).toBe(2);
    expect(h.redo()).toBe(3);
    expect(h.canRedo).toBe(false);
    expect(h.redo()).toBe(3);
  });

  test('pushing after an undo discards the redo branch', () => {
    const h = new History('a');
    h.push('b');
    h.undo();
    h.push('c');
    expect(h.canRedo).toBe(false);
    expect(h.undo()).toBe('a');
  });

  test('pushing the identical state is a no-op', () => {
    const state = { n: 1 };
    const h = new History(state);
    h.push(state);
    expect(h.canUndo).toBe(false);
  });

  test('keeps at most `limit` undo steps', () => {
    const h = new History(0, 3);
    for (let i = 1; i <= 10; i++) {
      h.push(i);
    }
    expect(h.undo()).toBe(9);
    expect(h.undo()).toBe(8);
    expect(h.undo()).toBe(7);
    expect(h.canUndo).toBe(false);
  });

  test('markSaved tracks unsaved changes across undo and redo', () => {
    const h = new History('a');
    expect(h.dirty).toBe(false);
    h.push('b');
    expect(h.dirty).toBe(true);
    h.markSaved();
    expect(h.dirty).toBe(false);
    h.undo();
    expect(h.dirty).toBe(true);
    h.redo();
    expect(h.dirty).toBe(false);
  });
});
