/** The player screen's event feed. */
import type { PlayerEvent, PlayerUpdate } from './types.ts';

/** Adds a line (once) and keeps the newest max lines, oldest first. */
export function addEvent(feed: readonly PlayerEvent[], ev: PlayerEvent, max: number): PlayerEvent[] {
  if (feed.some((e) => e.seq === ev.seq)) {
    return [...feed];
  }
  return [...feed, ev].slice(-max);
}

/** The feed after a live update: the server's whole feed when it sends one (earlier lines changed), else plus the new line. */
export function feedAfter(feed: readonly PlayerEvent[], u: PlayerUpdate, max: number): PlayerEvent[] {
  if (u.feed) {
    return u.feed.slice(-max);
  }
  return u.event ? addEvent(feed, u.event, max) : [...feed];
}
