/** Formatting for the session event log. */
import type { SessionEvent } from './types.ts';

export interface EventLine {
  round: string;
  time: string;
  summary: string;
  /** Category for styling: move, door, trap, hero, monster, reveal, round, note, session, other. */
  kind: string;
}

const CATEGORIES: Record<string, string> = {
  move: 'move',
  door: 'door',
  trap: 'trap',
  hero: 'hero',
  ability: 'hero',
  item: 'hero',
  gold: 'hero',
  monster: 'monster',
  area: 'reveal',
  tiles: 'reveal',
  seen: 'reveal',
  block: 'trap',
  players: 'note',
  round: 'round',
  fight: 'round',
  effect: 'note',
  log: 'note',
  passage: 'note',
  note: 'note',
  session: 'session',
};

export function formatEvent(e: SessionEvent): EventLine {
  const created = new Date(e.createdAt);
  const time = Number.isNaN(created.getTime()) ? '' : created.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
  const prefix = e.kind.split('.')[0] ?? e.kind;
  return { round: `R${e.round}`, time, summary: e.summary, kind: CATEGORIES[prefix] ?? 'other' };
}

/**
 * The log entries matching what the GM typed in the log search: every word must appear in
 * the summary (any order, ignoring case), and a word like "r2" picks round 2 instead.
 */
export function searchEvents<T extends Pick<SessionEvent, 'round' | 'summary'>>(events: readonly T[], query: string): T[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  if (words.length === 0) {
    return [...events];
  }
  return events.filter((e) => {
    const text = e.summary.toLowerCase();
    return words.every((w) => (/^r\d+$/.test(w) ? e.round === Number(w.slice(1)) : text.includes(w)));
  });
}
