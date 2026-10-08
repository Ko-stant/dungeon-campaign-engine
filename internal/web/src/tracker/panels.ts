/**
 * The tracker's collapsible panels: which start open, and whether the two
 * sidebars are shown (remembered in the browser).
 */

/** Right panel sections that start closed, so little shows to someone walking by. */
const CLOSED_RIGHT = new Set(['right:monsters', 'right:traps', 'right:blocks', 'right:notes']);

/**
 * Whether a collapsible section starts open: a hero's abilities, inventory
 * and item editors start closed, as do the right panel's lists; Read aloud,
 * the log and anything else start open.
 */
export function defaultOpen(key: string): boolean {
  if (/:(abilities|inventory)$/.test(key) || key.includes(':item:')) {
    return false;
  }
  return !CLOSED_RIGHT.has(key);
}

export interface Sidebars {
  left: boolean;
  right: boolean;
}

/** The sidebars as stored (JSON), both shown when nothing readable is stored. */
export function sidebarsFrom(stored: string | null): Sidebars {
  const out: Sidebars = { left: true, right: true };
  if (!stored) {
    return out;
  }
  try {
    const v: unknown = JSON.parse(stored);
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      const { left, right } = v as Record<string, unknown>;
      if (typeof left === 'boolean') {
        out.left = left;
      }
      if (typeof right === 'boolean') {
        out.right = right;
      }
    }
  } catch {
    // unreadable: both shown
  }
  return out;
}
