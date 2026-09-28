/**
 * Dice expressions such as "2d6+1" or "1d8+1d4", mirroring internal/dice (Go):
 * the same dice (d4 to d20), limits, error wording and canonical form.
 */

export const DICE_SIDES: readonly number[] = [4, 6, 8, 10, 12, 20];
const MAX_COUNT = 20;
const MAX_DICE = 50;
const MAX_MODIFIER = 99;

export interface DiceTerm {
  count: number;
  sides: number;
  negative: boolean;
}

export interface DiceExpr {
  terms: DiceTerm[];
  modifier: number;
}

export interface DieRoll {
  sides: number;
  value: number;
  negative: boolean;
}

export type ParseResult = { ok: true; expr: DiceExpr } | { ok: false; error: string };

const DICE_TERM = /^(\d*)\s*[dD]\s*(\d+)$/;
const CONSTANT_TERM = /^\d{1,4}$/;

/** Reads an expression; constants are summed into the modifier. */
export function parseDice(input: string): ParseResult {
  const s = input.trim();
  if (s === '') {
    return { ok: false, error: 'dice expression is empty' };
  }
  const q = JSON.stringify(s);
  const notDice = `${q} is not a dice expression (e.g. 2d6+1)`;
  const expr: DiceExpr = { terms: [], modifier: 0 };
  let dice = 0;
  let negative = false;
  let start = 0;
  for (let i = 0; i <= s.length; i++) {
    if (i < s.length && s[i] !== '+' && s[i] !== '-') {
      continue;
    }
    const raw = s.slice(start, i).trim();
    const leading = start === 0 && raw === '' && i < s.length;
    if (!leading) {
      if (raw === '') {
        return { ok: false, error: `${q} is missing a term` };
      }
      if (CONSTANT_TERM.test(raw)) {
        expr.modifier += negative ? -Number(raw) : Number(raw);
      } else {
        const m = DICE_TERM.exec(raw);
        if (!m) {
          return { ok: false, error: notDice };
        }
        const count = m[1] === '' ? 1 : Number(m[1]);
        const sides = Number(m[2]);
        if (!DICE_SIDES.includes(sides)) {
          return { ok: false, error: `${q}: dice must be d4, d6, d8, d10, d12 or d20` };
        }
        if (count < 1) {
          return { ok: false, error: `${q}: roll at least 1 die per term` };
        }
        if (count > MAX_COUNT) {
          return { ok: false, error: `${q}: roll at most ${MAX_COUNT} dice per term` };
        }
        expr.terms.push({ count, sides, negative });
        dice += count;
      }
    }
    if (i < s.length) {
      negative = s[i] === '-';
    }
    start = i + 1;
  }
  if (dice > MAX_DICE) {
    return { ok: false, error: `${q} has ${dice} dice; use at most ${MAX_DICE} dice` };
  }
  if (expr.modifier < -MAX_MODIFIER || expr.modifier > MAX_MODIFIER) {
    return { ok: false, error: `${q}: the modifier must be between ${-MAX_MODIFIER} and ${MAX_MODIFIER}` };
  }
  return { ok: true, expr };
}

/** The canonical form, e.g. "2d6+1"; the modifier always comes last. */
export function formatDice(e: DiceExpr): string {
  let out = '';
  e.terms.forEach((t, i) => {
    if (t.negative) {
      out += '-';
    } else if (i > 0) {
      out += '+';
    }
    out += `${t.count}d${t.sides}`;
  });
  if (e.terms.length === 0) {
    return String(e.modifier);
  }
  if (e.modifier > 0) {
    out += `+${e.modifier}`;
  } else if (e.modifier < 0) {
    out += String(e.modifier);
  }
  return out;
}

function randomDie(sides: number): number {
  return 1 + Math.floor(Math.random() * sides);
}

/** Throws every die (die returns 1..sides) and returns the total and each die in order. */
export function rollDice(e: DiceExpr, die: (sides: number) => number = randomDie): { total: number; rolls: DieRoll[] } {
  let total = e.modifier;
  const rolls: DieRoll[] = [];
  for (const t of e.terms) {
    for (let i = 0; i < t.count; i++) {
      const value = die(t.sides);
      total += t.negative ? -value : value;
      rolls.push({ sides: t.sides, value, negative: t.negative });
    }
  }
  return { total, rolls };
}
