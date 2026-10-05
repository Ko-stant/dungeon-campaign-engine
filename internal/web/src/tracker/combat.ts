/** Hero combat stats for display (The Three Plagues rules; see tracker.Combat in Go). */

import type { HeroCombat } from './types.ts';

/** A flat bonus on a dice expression: "1d20+3", or "2d10+2 +3" when it has its own modifier or several terms. */
function withBonus(dice: string, bonus: number): string {
  if (bonus === 0) {
    return dice;
  }
  return /[+-]/.test(dice) ? `${dice} +${bonus}` : `${dice}+${bonus}`;
}

/** One line, e.g. "Hit 1d20+3 · Crit 17-20 · Damage 10 · Avoid 3+1d6 · Mitigation 2". */
export function combatLine(c: HeroCombat): string {
  const parts = [
    `Hit ${withBonus(c.hitDice, c.accuracy)}`,
    c.critFrom < 20 ? `Crit ${c.critFrom}-20` : 'Crit 20',
    `Damage ${c.damage}`,
    `Avoid ${c.avoidance > 0 ? `${c.avoidance}+${c.defenseDice}` : c.defenseDice}`,
  ];
  if (c.mitigation > 0) {
    parts.push(`Mitigation ${c.mitigation}`);
  }
  if ((c.manaRegen ?? 0) > 0) {
    parts.push(`Mana +${c.manaRegen ?? 0} a fight round`);
  }
  return parts.join(' · ');
}
