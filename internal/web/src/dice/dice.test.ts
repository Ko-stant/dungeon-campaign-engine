import { describe, expect, test } from 'bun:test';
import { formatDice, parseDice, rollDice } from './dice.ts';

describe('parseDice', () => {
  test('reads expressions into canonical form, matching internal/dice (Go)', () => {
    const cases: Record<string, string> = {
      '1d8': '1d8',
      ' 2 D6 + 1 ': '2d6+1',
      d20: '1d20',
      '1d8+1d6': '1d8+1d6',
      '1d12-2': '1d12-2',
      '3': '3',
      '0': '0',
      '2+1d10': '1d10+2',
      '1d6+2+1': '1d6+3',
      '1d8-1d4': '1d8-1d4',
      '-1+1d6': '1d6-1',
      '20d20+99': '20d20+99',
    };
    for (const [input, want] of Object.entries(cases)) {
      const r = parseDice(input);
      if (!r.ok) {
        throw new Error(`${input}: ${r.error}`);
      }
      expect(formatDice(r.expr)).toBe(want);
    }
  });

  test('rejects bad expressions with a reason', () => {
    const cases: Record<string, string> = {
      '': 'empty',
      '1d7': 'd4, d6, d8, d10, d12 or d20',
      '0d6': 'at least 1',
      '21d6': 'at most 20',
      '1d6+': 'missing',
      '1d6++1': 'missing',
      abc: 'not a dice expression',
      '1dd6': 'not a dice expression',
      '1d6 1': 'not a dice expression',
      '1d6+100': 'between -99 and 99',
      '20d6+20d6+20d6': 'at most 50 dice',
    };
    for (const [input, want] of Object.entries(cases)) {
      const r = parseDice(input);
      expect(r.ok).toBe(false);
      if (!r.ok) {
        expect(r.error).toContain(want);
      }
    }
  });
});

describe('rollDice', () => {
  test('throws each die in order and adds the modifier', () => {
    const r = parseDice('2d6+1d4-1');
    if (!r.ok) {
      throw new Error(r.error);
    }
    const asked: number[] = [];
    const result = rollDice(r.expr, (sides) => {
      asked.push(sides);
      return sides;
    });
    expect(asked).toEqual([6, 6, 4]);
    expect(result.total).toBe(15);
    expect(result.rolls).toEqual([
      { sides: 6, value: 6, negative: false },
      { sides: 6, value: 6, negative: false },
      { sides: 4, value: 4, negative: false },
    ]);
  });

  test('subtracts negative terms', () => {
    const r = parseDice('1d8-1d4');
    if (!r.ok) {
      throw new Error(r.error);
    }
    expect(rollDice(r.expr, () => 2).total).toBe(0);
  });

  test('the default die stays within its faces', () => {
    const r = parseDice('20d20');
    if (!r.ok) {
      throw new Error(r.error);
    }
    const { rolls } = rollDice(r.expr);
    expect(rolls.every((d) => d.value >= 1 && d.value <= 20 && Number.isInteger(d.value))).toBe(true);
  });
});
