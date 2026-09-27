import { describe, expect, test } from 'bun:test';
import { ApiError, createApi } from './api.ts';

interface Call {
  url: string;
  method: string;
  body: unknown;
}

function fakeFetch(status: number, body: unknown, calls: Call[]): typeof fetch {
  return ((input: RequestInfo | URL, init?: RequestInit) => {
    calls.push({
      url: typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url,
      method: init?.method ?? 'GET',
      body: typeof init?.body === 'string' ? JSON.parse(init.body) : undefined,
    });
    const text = body === undefined ? '' : JSON.stringify(body);
    return Promise.resolve(new Response(status === 204 ? null : text, { status, headers: { 'Content-Type': 'application/json' } }));
  }) as typeof fetch;
}

describe('api client', () => {
  test('GETs and decodes JSON', async () => {
    const calls: Call[] = [];
    const api = createApi(fakeFetch(200, { id: 'b1', name: 'Keep' }, calls));
    const board = await api.board('b1');
    expect(board.name).toBe('Keep');
    expect(calls).toEqual([{ url: '/api/boards/b1', method: 'GET', body: undefined }]);
  });

  test('PUTs a JSON body when saving', async () => {
    const calls: Call[] = [];
    const api = createApi(fakeFetch(200, { id: 'q1' }, calls));
    await api.saveQuest('q1', 'Rescue', { version: 2, boardChecksum: '', doors: [], blockedSquares: [], furniture: [], monsters: [], traps: [], notes: [], startTiles: [] });
    expect(calls[0]?.method).toBe('PUT');
    expect(calls[0]?.url).toBe('/api/quests/q1');
    expect(calls[0]?.body).toMatchObject({ name: 'Rescue', quest: { version: 2 } });
  });

  test('encodes ids into the path', async () => {
    const calls: Call[] = [];
    await createApi(fakeFetch(200, [], calls)).quests('a/b');
    expect(calls[0]?.url).toBe('/api/boards/a%2Fb/quests');
  });

  test('turns error responses into ApiError with the server message', async () => {
    const api = createApi(fakeFetch(400, { error: 'board size must be 1..200' }, []));
    try {
      await api.createBoard('x', 999, 1);
      throw new Error('expected a rejection');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).status).toBe(400);
      expect((err as ApiError).message).toBe('board size must be 1..200');
    }
  });

  test('handles 204 No Content', async () => {
    const calls: Call[] = [];
    await createApi(fakeFetch(204, undefined, calls)).deleteQuest('q1');
    expect(calls).toEqual([{ url: '/api/quests/q1', method: 'DELETE', body: undefined }]);
  });
});
