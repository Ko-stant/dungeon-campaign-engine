import { describe, expect, test } from 'bun:test';
import { createTrackerApi } from './api.ts';

describe('tracker api client', () => {
  test('posts commands and reads events after a sequence number', async () => {
    const calls: { url: string; method: string; body: unknown }[] = [];
    const fetchFn = ((input: RequestInfo | URL, init?: RequestInit) => {
      calls.push({ url: typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url, method: init?.method ?? 'GET', body: typeof init?.body === 'string' ? JSON.parse(init.body) : undefined });
      return Promise.resolve(new Response('{}', { status: 200 }));
    }) as typeof fetch;
    const api = createTrackerApi(fetchFn);

    await api.command('s1', { type: 'round.advance', payload: {} });
    await api.events('s1', 7);
    await api.complete('s1');
    await api.travel('s1', 'q2');
    await api.chapters('c1');
    await api.script('c1');
    await api.audio('c1');

    expect(calls).toEqual([
      { url: '/api/sessions/s1/commands', method: 'POST', body: { type: 'round.advance', payload: {} } },
      { url: '/api/sessions/s1/events?after=7', method: 'GET', body: undefined },
      { url: '/api/sessions/s1/complete', method: 'POST', body: undefined },
      { url: '/api/sessions/s1/travel', method: 'POST', body: { questId: 'q2' } },
      { url: '/api/campaigns/c1/chapters', method: 'GET', body: undefined },
      { url: '/api/campaigns/c1/script', method: 'GET', body: undefined },
      { url: '/api/campaigns/c1/audio', method: 'GET', body: undefined },
    ]);
  });

  test('streamUrl uses ws or wss to match the page', () => {
    expect(createTrackerApi().streamUrl('s1', { protocol: 'http:', host: 'localhost:8080' })).toBe('ws://localhost:8080/api/sessions/s1/stream');
    expect(createTrackerApi().streamUrl('s1', { protocol: 'https:', host: 'x.test' })).toBe('wss://x.test/api/sessions/s1/stream');
  });
});
