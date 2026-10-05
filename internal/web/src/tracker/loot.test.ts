import { describe, expect, test } from 'bun:test';
import { itemSummary, lootAddCommand } from './loot.ts';

describe('lootAddCommand', () => {
  test('a loot item goes to the hero with its kind, stats, use and notes (one of it, unequipped)', () => {
    const bow = { id: 'loot-1', name: "Wardens' Longbow", quantity: 1, kind: 'bow', damage: 6, notes: 'For the Ranger; replaces Hunting Bow' };
    expect(lootAddCommand('hero-1', bow)).toEqual({
      type: 'item.add',
      payload: { heroId: 'hero-1', name: "Wardens' Longbow", quantity: 1, kind: 'bow', notes: 'For the Ranger; replaces Hunting Bow', damage: 6 },
    });
    const potion = { id: 'loot-2', name: 'Healing Potion', quantity: 1, kind: 'potion', healBody: 8 };
    expect(lootAddCommand('hero-2', potion).payload).toEqual({ heroId: 'hero-2', name: 'Healing Potion', quantity: 1, kind: 'potion', healBody: 8 });
  });
});

describe('itemSummary', () => {
  test("stats, then what using it does; mirrors Item.Summary (Go)", () => {
    expect(itemSummary({ damage: 7 })).toBe('damage +7');
    expect(itemSummary({ healBody: 8 })).toBe('heals 8 Body');
    expect(itemSummary({ mana: 2, restoreMana: 6 })).toBe('mana +2, restores 6 mana');
    expect(itemSummary({})).toBe('');
  });
});
