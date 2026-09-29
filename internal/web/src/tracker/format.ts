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
  monster: 'monster',
  area: 'reveal',
  tiles: 'reveal',
  round: 'round',
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
