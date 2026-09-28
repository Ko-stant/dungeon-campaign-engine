/** Ability cooldowns and gold input for the tracker's hero cards. Advice only: nothing here refuses the GM. */
import type { Ability } from '../maps/types.ts';

export interface AbilityRow {
  ability: Ability;
  /** Rounds until ready; 0 means ready. Used in round R with cooldown C, ready again in round R + C. */
  roundsLeft: number;
  /** Active, reaction and spell abilities get a Use button; passives are reminders. */
  usable: boolean;
  /** The hero has less mana than the ability costs (it can still be used). */
  shortOfMana: boolean;
  status: string;
}

/** "2 mana, every 10 rounds", matching the Classes page (views.AbilityLimit). */
export function abilityLimit(a: Ability): string {
  const parts: string[] = [];
  if ((a.manaCost ?? 0) > 0) {
    parts.push(`${String(a.manaCost)} mana`);
  }
  const cd = a.cooldown ?? 0;
  if (cd === 1) {
    parts.push('every round');
  } else if (cd > 1) {
    parts.push(`every ${String(cd)} rounds`);
  }
  return parts.join(', ');
}

export function abilityRows(
  hero: { mana?: number; abilities?: readonly Ability[] | null; cooldowns?: Readonly<Record<string, number>> | null },
  round: number,
): AbilityRow[] {
  return (hero.abilities ?? []).map((ability) => {
    const ready = hero.cooldowns?.[ability.id];
    const roundsLeft = ready !== undefined && ready > round ? ready - round : 0;
    const usable = ability.kind !== 'passive';
    let status = 'ready';
    if (!usable) {
      status = 'passive';
    } else if (roundsLeft > 0) {
      status = roundsLeft === 1 ? '1 round left' : `${String(roundsLeft)} rounds left`;
    }
    return { ability, roundsLeft, usable, shortOfMana: (ability.manaCost ?? 0) > (hero.mana ?? 0), status };
  });
}

const MAX_GOLD = 999999;

/** What the GM typed into a gold box: "+25" adds, "-10" takes away, "40" sets. Mirrors tracker.GoldChange (Go). */
export function goldChange(current: number, input: string): { ok: true; gold: number } | { ok: false; error: string } {
  const typed = input.trim();
  let s = typed.replaceAll(' ', '');
  let sign = 0;
  if (s.startsWith('+')) {
    sign = 1;
    s = s.slice(1);
  } else if (s.startsWith('-')) {
    sign = -1;
    s = s.slice(1);
  }
  if (!/^\d{1,9}$/.test(s)) {
    return { ok: false, error: `gold: ${JSON.stringify(typed)} is not an amount (e.g. 40, +25 or -10)` };
  }
  const n = Number(s);
  const gold = sign === 0 ? n : current + sign * n;
  if (gold < 0) {
    return { ok: false, error: `gold: taking ${String(n)} leaves less than 0 (the hero has ${String(current)})` };
  }
  if (gold > MAX_GOLD) {
    return { ok: false, error: `gold must be at most ${String(MAX_GOLD)}` };
  }
  return { ok: true, gold };
}
