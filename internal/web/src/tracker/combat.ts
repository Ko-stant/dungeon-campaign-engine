/** Hero combat stats for display (The Three Plagues rules; see tracker.Combat in Go). */

import type { HeroCombat, MonsterCombat } from './types.ts';

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

/** One line, e.g. "Avoid 14 · Hit 2d10+3 · Damage 12 · strikes 2 in a line" (as the campaign page shows it). */
export function monsterLine(c: MonsterCombat): string {
  const parts = [`Avoid ${c.avoidance}`, `Hit ${c.hitDice}`, `Damage ${c.damage}`];
  if (c.ranged) {
    parts.push('ranged');
  }
  if (c.reach) {
    parts.push('reach');
  }
  if ((c.line ?? 0) > 0) {
    parts.push(`strikes ${(c.line ?? 0) + 1} in a line`);
  }
  if ((c.splashDamage ?? 0) > 0 && (c.splashTargets ?? 0) > 0) {
    parts.push(`blast ${c.splashDamage ?? 0} to ${c.splashTargets ?? 0} beside the target`);
  }
  if (c.undead) {
    parts.push('undead');
  }
  return parts.join(' · ');
}

/** Faltering: at or below a quarter of maximum Body (rounded down, at least 1) a monster has Avoidance -4. */
export const FALTER_PENALTY = 4;

export function falterAt(maxBody: number): number {
  return Math.max(1, Math.floor(maxBody / 4));
}
