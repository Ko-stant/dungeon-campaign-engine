/**
 * When to suggest starting or ending a fight, so the GM does not forget the
 * Start fight / End fight buttons. Advice only: the buttons pulse, nothing
 * starts or ends by itself.
 *
 * Out of a fight, a fight is suggested when a living monster comes into the
 * players' sight, a monster loses Body, or a hero's Determination goes up (a
 * missed attack) - compared with the calm picture taken when the last fight
 * ended or the GM said "not a fight". In a fight, ending it is suggested once
 * no living monster is in sight.
 */
import type { Visibility } from './types.ts';

export interface FightView {
  fight?: boolean;
  monsters: readonly { id: string; name: string; body: number; maxBody: number; alive: boolean; visibility: Visibility }[];
  heroes: readonly { id: string; name: string; determination?: number }[];
}

export interface FightHint {
  kind: 'start' | 'end';
  /** Why, for the button's title: "Goblin in sight". */
  reason: string;
}

/** The calm picture: monsters in sight (with their Body), every monster's Body, and each hero's Determination. */
export interface Calm {
  /** Living monsters the players could see -> their Body. */
  seen: Record<string, number>;
  /** Every monster -> its Body. */
  body: Record<string, number>;
  determination: Record<string, number>;
}

export function calmFrom(s: FightView): Calm {
  const calm: Calm = { seen: {}, body: {}, determination: {} };
  for (const m of s.monsters) {
    calm.body[m.id] = m.body;
    if (m.alive && m.visibility === 'seen') {
      calm.seen[m.id] = m.body;
    }
  }
  for (const h of s.heroes) {
    calm.determination[h.id] = h.determination ?? 0;
  }
  return calm;
}

export function fightHint(s: FightView, calm: Calm | null): FightHint | null {
  const living = s.monsters.filter((m) => m.alive);
  const inSight = living.filter((m) => m.visibility === 'seen');
  if (s.fight) {
    return inSight.length === 0 ? { kind: 'end', reason: 'No monsters left in sight' } : null;
  }
  const hurt = living.find((m) => m.body < (calm?.body[m.id] ?? m.maxBody));
  if (hurt) {
    return { kind: 'start', reason: `${hurt.name} took damage` };
  }
  const missed = s.heroes.find((h) => (h.determination ?? 0) > (calm?.determination[h.id] ?? 0));
  if (missed) {
    return { kind: 'start', reason: `${missed.name} missed an attack` };
  }
  const fresh = inSight.filter((m) => !(calm && m.id in calm.seen));
  if (fresh.length === 1 && fresh[0]) {
    return { kind: 'start', reason: `${fresh[0].name} in sight` };
  }
  return fresh.length > 1 ? { kind: 'start', reason: `${String(fresh.length)} monsters in sight` } : null;
}

const isNumberMap = (v: unknown): v is Record<string, number> =>
  typeof v === 'object' && v !== null && !Array.isArray(v) && Object.values(v).every((n) => typeof n === 'number');

/** A calm picture stored as JSON, or null when nothing readable is stored. */
export function calmFromStorage(stored: string | null): Calm | null {
  if (!stored) {
    return null;
  }
  try {
    const v = JSON.parse(stored) as Partial<Record<keyof Calm, unknown>>;
    if (isNumberMap(v.seen) && isNumberMap(v.body) && isNumberMap(v.determination)) {
      return { seen: v.seen, body: v.body, determination: v.determination };
    }
  } catch {
    // unreadable
  }
  return null;
}
