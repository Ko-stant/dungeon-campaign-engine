/** Effects on heroes and monsters (see internal/tracker/fights.go): a name, an optional countdown in fight rounds, a note. */

import type { Effect } from './types.ts';

/** Names offered when adding an effect; any name may be typed. */
export const EFFECT_SUGGESTIONS: readonly string[] = [
  'Poisoned',
  'Raging',
  'Turned',
  'Marked',
  'Vanished',
  'Cannot defend',
  'Stunned',
];

/** e.g. "Poisoned (3 rounds): 3 a turn". */
export function effectLabel(e: Effect): string {
  let label = e.name;
  const rounds = e.rounds ?? 0;
  if (rounds > 0) {
    label += ` (${String(rounds)} ${rounds === 1 ? 'round' : 'rounds'})`;
  }
  if (e.note) {
    label += `: ${e.note}`;
  }
  return label;
}
