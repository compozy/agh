import { getResponse, type HttpHandler } from "msw";

export function createMswFetch(
  getHandlers: () => ReadonlyArray<HttpHandler>,
  baseUrl = typeof window === "undefined" ? "http://localhost" : window.location.origin
): typeof globalThis.fetch {
  return (async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? input.clone() : new Request(input, init);
    const response = await getResponse([...getHandlers()], request, { baseUrl });

    if (!response) {
      const url = new URL(request.url, baseUrl);
      throw new Error(`No MSW handler matched: ${request.method} ${url.pathname}${url.search}`);
    }

    return response;
  }) as typeof globalThis.fetch;
}
