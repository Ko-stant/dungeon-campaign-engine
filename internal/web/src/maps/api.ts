/** Typed client for the map creator JSON API (internal/app). */
import type { BoardDoc, BoardResponse, Catalog, QuestDoc, QuestResponse, QuestSummary } from './types.ts';

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

const id = (v: string): string => encodeURIComponent(v);

export function createApi(fetchFn: typeof fetch = fetch.bind(globalThis)) {
  async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
    const headers: Record<string, string> = { Accept: 'application/json' };
    const init: RequestInit = { method, headers };
    if (body !== undefined) {
      init.body = JSON.stringify(body);
      headers['Content-Type'] = 'application/json';
    }
    const res = await fetchFn(url, init);
    if (res.status === 204) {
      return undefined as T;
    }
    const text = await res.text();
    let data: unknown;
    try {
      data = text ? JSON.parse(text) : undefined;
    } catch {
      // Not JSON (e.g. a plain-text error from a proxy); fall back to the raw text below.
    }
    if (!res.ok) {
      const message =
        typeof data === 'object' && data !== null && 'error' in data && typeof data.error === 'string'
          ? data.error
          : text || `request failed with status ${res.status}`;
      throw new ApiError(res.status, message);
    }
    return data as T;
  }

  return {
    catalog: () => request<Catalog>('GET', '/api/catalog'),
    createBoard: (name: string, width: number, height: number) => request<BoardResponse>('POST', '/api/boards', { name, width, height }),
    board: (boardId: string) => request<BoardResponse>('GET', `/api/boards/${id(boardId)}`),
    saveBoard: (boardId: string, name: string, board: BoardDoc) => request<BoardResponse>('PUT', `/api/boards/${id(boardId)}`, { name, board }),
    quests: (boardId: string) => request<QuestSummary[]>('GET', `/api/boards/${id(boardId)}/quests`),
    createQuest: (boardId: string, name: string) => request<QuestResponse>('POST', `/api/boards/${id(boardId)}/quests`, { name }),
    quest: (questId: string) => request<QuestResponse>('GET', `/api/quests/${id(questId)}`),
    saveQuest: (questId: string, name: string, quest: QuestDoc) => request<QuestResponse>('PUT', `/api/quests/${id(questId)}`, { name, quest }),
    deleteQuest: async (questId: string): Promise<void> => {
      await request<unknown>('DELETE', `/api/quests/${id(questId)}`);
    },
  };
}

export type Api = ReturnType<typeof createApi>;
