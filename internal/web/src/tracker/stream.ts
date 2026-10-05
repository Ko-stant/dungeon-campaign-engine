/**
 * Messages on the GM's live stream (/api/sessions/{id}/stream): a change
 * (a CommandResponse), or who is connected online (internal/app/seat.go,
 * PresenceUpdate).
 */

import type { CommandResponse } from './types.ts';

/** A player connected to the session online, with the heroes they play. */
export interface Present {
  name: string;
  heroes: string[];
}

export type StreamMessage = { kind: 'change'; response: CommandResponse } | { kind: 'presence'; presence: Present[] };

/** Reads one stream message; anything unrecognized is null. */
export function parseStreamMessage(data: string): StreamMessage | null {
  let msg: unknown;
  try {
    msg = JSON.parse(data);
  } catch {
    return null;
  }
  if (typeof msg !== 'object' || msg === null) {
    return null;
  }
  if ('presence' in msg && Array.isArray(msg.presence)) {
    return { kind: 'presence', presence: msg.presence as Present[] };
  }
  if ('eventSeq' in msg && 'state' in msg) {
    return { kind: 'change', response: msg as CommandResponse };
  }
  return null;
}
