/** Typed client for the tracker session API (internal/app). */
import { createRequester, encodeId as id } from '../api/http.ts';
import type { Catalog } from '../maps/types.ts';
import type { Command, CommandResponse, SessionEvent, SessionResponse } from './types.ts';

export function createTrackerApi(fetchFn: typeof fetch = fetch.bind(globalThis)) {
  const request = createRequester(fetchFn);
  return {
    catalog: () => request<Catalog>('GET', '/api/catalog'),
    session: (sessionId: string) => request<SessionResponse>('GET', `/api/sessions/${id(sessionId)}`),
    command: (sessionId: string, command: Command) => request<CommandResponse>('POST', `/api/sessions/${id(sessionId)}/commands`, command),
    events: (sessionId: string, after = 0) => request<SessionEvent[]>('GET', `/api/sessions/${id(sessionId)}/events?after=${after}`),
    complete: (sessionId: string) => request<CommandResponse>('POST', `/api/sessions/${id(sessionId)}/complete`),
    reopen: (sessionId: string) => request<CommandResponse>('POST', `/api/sessions/${id(sessionId)}/reopen`),
    /** WebSocket URL for live updates, matching the page's scheme. */
    streamUrl: (sessionId: string, loc: { protocol: string; host: string } = window.location) =>
      `${loc.protocol === 'https:' ? 'wss' : 'ws'}://${loc.host}/api/sessions/${id(sessionId)}/stream`,
  };
}

export type TrackerApi = ReturnType<typeof createTrackerApi>;
