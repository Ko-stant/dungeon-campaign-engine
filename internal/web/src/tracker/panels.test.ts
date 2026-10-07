import { describe, expect, test } from 'bun:test';
import { defaultOpen, sidebarsFrom } from './panels.ts';

describe('defaultOpen', () => {
  test('hero abilities, inventory and item editors start closed', () => {
    expect(defaultOpen('hero-1:abilities')).toBe(false);
    expect(defaultOpen('hero-1:inventory')).toBe(false);
    expect(defaultOpen('hero-1:item:item-2')).toBe(false);
  });

  test('the right panel: Read aloud and the log start open, the lists closed', () => {
    expect(defaultOpen('right:read')).toBe(true);
    expect(defaultOpen('right:log')).toBe(true);
    for (const key of ['right:monsters', 'right:traps', 'right:blocks', 'right:notes']) {
      expect(defaultOpen(key)).toBe(false);
    }
  });

  test('anything else starts open (the odds block)', () => {
    expect(defaultOpen('odds')).toBe(true);
  });
});

describe('sidebarsFrom', () => {
  test('both sidebars show unless stored as hidden', () => {
    expect(sidebarsFrom(null)).toEqual({ left: true, right: true });
    expect(sidebarsFrom('{"left":false}')).toEqual({ left: false, right: true });
    expect(sidebarsFrom('{"left":true,"right":false}')).toEqual({ left: true, right: false });
  });

  test('anything unreadable shows both', () => {
    expect(sidebarsFrom('not json')).toEqual({ left: true, right: true });
    expect(sidebarsFrom('[1]')).toEqual({ left: true, right: true });
    expect(sidebarsFrom('{"left":"no"}')).toEqual({ left: true, right: true });
  });
});
