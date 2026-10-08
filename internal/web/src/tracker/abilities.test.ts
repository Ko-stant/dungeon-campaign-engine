import { describe, expect, test } from 'bun:test';
import { abilityLimit, abilityRows, goldChange } from './abilities.ts';
import { formatEvent } from './format.ts';
import type { Ability } from '../maps/types.ts';

const rain: Ability = { id: 'ability-1', name: 'Rain of Arrows', kind: 'active', cooldown: 6 };
const heal: Ability = { id: 'ability-2', name: 'Heal Minor Wounds', kind: 'spell', manaCost: 3 };
const blessing: Ability = { id: 'ability-3', name: 'Divine Blessing', kind: 'spell', manaCost: 2, cooldown: 10 };
const tough: Ability = { id: 'ability-4', name: 'Tough as Nails', kind: 'passive' };

describe('abilityLimit', () => {
  test('describes cost and cooldown like the Classes page', () => {
    expect(abilityLimit(rain)).toBe('every 6 rounds');
    expect(abilityLimit(heal)).toBe('3 mana');
    expect(abilityLimit(blessing)).toBe('2 mana, every 10 rounds');
    expect(abilityLimit({ ...rain, cooldown: 1 })).toBe('every round');
    expect(abilityLimit(tough)).toBe('');
  });
});

describe('abilityRows', () => {
  test('shows rounds left until ready (ready round = round used + cooldown)', () => {
    const hero = { mana: 1, abilities: [rain, heal, blessing, tough], cooldowns: { 'ability-1': 10, 'ability-3': 5 } };
    const rows = abilityRows(hero, 5);
    expect(rows.map((r) => [r.ability.name, r.roundsLeft, r.usable, r.shortOfMana])).toEqual([
      ['Rain of Arrows', 5, true, false],
      ['Heal Minor Wounds', 0, true, true],
      ['Divine Blessing', 0, true, true],
      ['Tough as Nails', 0, false, false],
    ]);
    expect(rows[0]?.status).toBe('5 rounds left');
    expect(abilityRows({ ...hero, cooldowns: { 'ability-1': 6 } }, 5)[0]?.status).toBe('1 round left');
    expect(rows[1]?.status).toBe('needs 3 mana');
    expect(rows[3]?.status).toBe('passive');
  });

  test('off cooldown but short of mana says so, instead of "ready"', () => {
    const rows = abilityRows({ mana: 6, abilities: [heal, { ...heal, id: 'ability-5', name: 'Turn Evil', manaCost: 8, cooldown: 5 }] }, 1);
    expect(rows.map((r) => r.status)).toEqual(['ready', 'needs 8 mana']);
    // Cooling down: the rounds left come first.
    expect(abilityRows({ mana: 0, abilities: [blessing], cooldowns: { 'ability-3': 3 } }, 1)[0]?.status).toBe('2 rounds left');
  });

  test('works for heroes from sessions without abilities', () => {
    expect(abilityRows({}, 1)).toEqual([]);
    expect(abilityRows({ abilities: null, cooldowns: null }, 1)).toEqual([]);
  });
});

describe('goldChange', () => {
  test('adds, takes away or sets, matching internal/tracker (Go)', () => {
    expect(goldChange(30, '+25')).toEqual({ ok: true, gold: 55 });
    expect(goldChange(30, ' - 10 ')).toEqual({ ok: true, gold: 20 });
    expect(goldChange(30, '=40')).toEqual({ ok: true, gold: 40 });
    expect(goldChange(30, '= 0')).toEqual({ ok: true, gold: 0 });
    expect(goldChange(30, '-30')).toEqual({ ok: true, gold: 0 });
  });

  test('rejects amounts it cannot read or that leave less than 0', () => {
    // A bare number is refused: +, - or = says what it does.
    for (const input of ['', '+', '=', '40', 'abc', '1.5', '+1000000', '=-5']) {
      expect(goldChange(30, input).ok).toBe(false);
    }
    const r = goldChange(5, '-6');
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.error).toContain('less than 0');
    }
  });
});

describe('formatEvent for abilities and items', () => {
  test('files ability and item events under heroes', () => {
    const base = { seq: 1, round: 1, summary: 's', payload: {}, createdAt: '' };
    expect(formatEvent({ ...base, kind: 'ability.use' }).kind).toBe('hero');
    expect(formatEvent({ ...base, kind: 'item.give' }).kind).toBe('hero');
  });
});
