/** Typed client for the tracker session API (internal/app). */
import { createRequester, encodeId as id } from '../api/http.ts';
import type { Catalog } from '../maps/types.ts';
import type { PlayerResponse } from '../players/types.ts';
import type { Action, SeatResponse } from '../seat/types.ts';
import type { Chapter, Command, CommandResponse, Item, ScriptSection, SessionEvent, SessionResponse } from './types.ts';

export function createTrackerApi(fetchFn: typeof fetch = fetch.bind(globalThis)) {
  const request = createRequester(fetchFn);
  return {
    catalog: () => request<Catalog>('GET', '/api/catalog'),
    session: (sessionId: string) => request<SessionResponse>('GET', `/api/sessions/${id(sessionId)}`),
    command: (sessionId: string, command: Command) => request<CommandResponse>('POST', `/api/sessions/${id(sessionId)}/commands`, command),
    /** What the GM may do now by the rules (empty while the rules are off), and the event it was listed for. */
    gmActions: (sessionId: string) => request<{ eventSeq: number; actions: Action[] }>('GET', `/api/sessions/${id(sessionId)}/actions`),
    events: (sessionId: string, after = 0) => request<SessionEvent[]>('GET', `/api/sessions/${id(sessionId)}/events?after=${after}`),
    complete: (sessionId: string) => request<CommandResponse>('POST', `/api/sessions/${id(sessionId)}/complete`),
    reopen: (sessionId: string) => request<CommandResponse>('POST', `/api/sessions/${id(sessionId)}/reopen`),
    /** Moves the party to another map (a quest), mid-game. */
    travel: (sessionId: string, questId: string) => request<CommandResponse>('POST', `/api/sessions/${id(sessionId)}/travel`, { questId }),
    chapters: (campaignId: string) => request<Chapter[]>('GET', `/api/campaigns/${id(campaignId)}/chapters`),
    /** The campaign's read-aloud script (empty sections when it has none). */
    loot: (campaignId: string) => request<Item[]>('GET', `/api/campaigns/${id(campaignId)}/loot`),
    script: (campaignId: string) => request<{ sections: ScriptSection[] }>('GET', `/api/campaigns/${id(campaignId)}/script`),
    /** The campaign's audio clips: clip id ("Q2-03", "Q3-09a") -> URL. */
    audio: (campaignId: string) => request<{ clips: Record<string, string> }>('GET', `/api/campaigns/${id(campaignId)}/audio`),
    /** The player screen's filtered view, latest lines and catalog (see internal/app/player_api.go). */
    player: (sessionId: string) => request<PlayerResponse>('GET', `/api/sessions/${id(sessionId)}/player`),
    /** WebSocket URL for the player screen's live updates. */
    playerStreamUrl: (sessionId: string, loc: { protocol: string; host: string } = window.location) =>
      `${loc.protocol === 'https:' ? 'wss' : 'ws'}://${loc.host}/api/sessions/${id(sessionId)}/player-stream`,
    /** A player's seat: the player view and their own heroes (internal/app/seat.go). */
    seat: (sessionId: string) => request<SeatResponse>('GET', `/api/sessions/${id(sessionId)}/seat`),
    /** Sends a command for one of the player's heroes; answers with the seat after it. */
    seatCommand: (sessionId: string, hero: string, command: Command) =>
      request<SeatResponse>('POST', `/api/sessions/${id(sessionId)}/seat-commands`, { hero, type: command.type, payload: command.payload }),
    /** WebSocket URL for a seat's live updates and who is here. */
    seatStreamUrl: (sessionId: string, loc: { protocol: string; host: string } = window.location) =>
      `${loc.protocol === 'https:' ? 'wss' : 'ws'}://${loc.host}/api/sessions/${id(sessionId)}/seat-stream`,
    /** WebSocket URL for live updates, matching the page's scheme. */
    streamUrl: (sessionId: string, loc: { protocol: string; host: string } = window.location) =>
      `${loc.protocol === 'https:' ? 'wss' : 'ws'}://${loc.host}/api/sessions/${id(sessionId)}/stream`,
  };
}

export type TrackerApi = ReturnType<typeof createTrackerApi>;
