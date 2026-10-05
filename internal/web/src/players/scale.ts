/** The player screen's text size steps (the whole page scales with the root font size). */

export const SCALES: readonly number[] = [1, 1.15, 1.3, 1.5, 1.75, 2];

/** The next size up (dir 1) or down (dir -1), stopping at the ends. */
export function nextScale(current: number, dir: 1 | -1): number {
  const i = SCALES.indexOf(current);
  const at = i < 0 ? 0 : i;
  return SCALES[Math.max(0, Math.min(SCALES.length - 1, at + dir))] ?? 1;
}

/** A stored size, or 1 when it is missing or not one of the steps. */
export function parseScale(stored: string | null): number {
  const n = Number(stored);
  return SCALES.includes(n) ? n : 1;
}
