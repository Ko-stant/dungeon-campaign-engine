/** Typed client for the map creator JSON API (internal/app). */
import { createRequester, encodeId as id } from '../api/http.ts';
import type { BoardDoc, BoardResponse, Catalog, QuestDoc, QuestResponse, QuestSummary } from './types.ts';

export { ApiError } from '../api/http.ts';

export function createApi(fetchFn: typeof fetch = fetch.bind(globalThis)) {
  const request = createRequester(fetchFn);
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
    /** Deletes the board and every quest on it. */
    deleteBoard: async (boardId: string): Promise<void> => {
      await request<unknown>('DELETE', `/api/boards/${id(boardId)}?withQuests=true`);
    },
  };
}

export type Api = ReturnType<typeof createApi>;
