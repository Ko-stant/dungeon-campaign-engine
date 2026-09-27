/** Shared JSON request helper for the app's API clients. */

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

export type Requester = <T>(method: string, url: string, body?: unknown) => Promise<T>;

/** Returns a request function bound to `fetchFn` that throws ApiError on failure. */
export function createRequester(fetchFn: typeof fetch): Requester {
  return async <T>(method: string, url: string, body?: unknown): Promise<T> => {
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
  };
}

export const encodeId = (v: string): string => encodeURIComponent(v);
