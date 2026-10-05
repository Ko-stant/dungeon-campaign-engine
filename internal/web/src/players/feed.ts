/** The player screen's event feed. */
import type { PlayerEvent } from './types.ts';

/** Adds a line (once) and keeps the newest max lines, oldest first. */
export function addEvent(feed: readonly PlayerEvent[], ev: PlayerEvent, max: number): PlayerEvent[] {
  if (feed.some((e) => e.seq === ev.seq)) {
    return [...feed];
  }
  return [...feed, ev].slice(-max);
}
