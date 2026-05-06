import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { getResponse, http, HttpResponse } from "msw";
import { createElement, type ReactNode } from "react";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

import { handlers } from "@/systems/tasks/mocks/handlers";

import { useTaskDetailOrchestrationTab } from "./use-task-detail-orchestration-tab";

const originalFetch = globalThis.fetch;
const originalEventSource = (globalThis as unknown as { EventSource?: unknown }).EventSource;

class StubEventSource {
  url: string;
  readonly readyState = 0;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;

  constructor(url: string) {
    this.url = url;
  }

  addEventListener(): void {}
  removeEventListener(): void {}
  close(): void {}
}

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

beforeAll(() => {
  const baseUrl = typeof window === "undefined" ? "http://localhost" : window.location.origin;
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? input.clone() : new Request(input, init);
    const response = await getResponse(handlers, request, { baseUrl });
    if (!response) {
      throw new Error(`No MSW handler matched: ${request.method} ${request.url}`);
    }
    return response;
  }) as typeof globalThis.fetch;
  (globalThis as unknown as { EventSource: typeof StubEventSource }).EventSource = StubEventSource;
});

afterAll(() => {
  globalThis.fetch = originalFetch;
  if (originalEventSource === undefined) {
    delete (globalThis as Record<string, unknown>).EventSource;
  } else {
    (globalThis as Record<string, unknown>).EventSource = originalEventSource;
  }
});

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useTaskDetailOrchestrationTab", () => {
  it("Should expose orchestration data through MSW handlers", async () => {
    const { result } = renderHook(
      () =>
        useTaskDetailOrchestrationTab("task_001", {
          enabled: true,
          latestEventSeq: 14,
        }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.profile?.task_id).toBe("task_001");
      expect(result.current.reviews.length).toBeGreaterThan(0);
      expect(result.current.subscriptions.length).toBeGreaterThan(0);
    });
    expect(result.current.streamSeedSequence).toBe(14);
    expect(result.current.hasLatestEventSeq).toBe(true);
  });

  it("Should disable subscriptions when not enabled", async () => {
    const { result } = renderHook(
      () => useTaskDetailOrchestrationTab("task_001", { enabled: false }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.profileLoading).toBe(false);
    });
    expect(result.current.profile).toBeNull();
    expect(result.current.subscriptions).toEqual([]);
    expect(result.current.streamState).toBe("disabled");
  });

  it("Should report idle stream state when transitioning from disabled to enabled before any event", async () => {
    const { result, rerender } = renderHook(
      ({ enabled }: { enabled: boolean }) =>
        useTaskDetailOrchestrationTab("task_001", { enabled, latestEventSeq: 0 }),
      { wrapper: createWrapper(), initialProps: { enabled: false } }
    );

    expect(result.current.streamState).toBe("disabled");

    rerender({ enabled: true });

    await waitFor(() => {
      expect(result.current.streamState).toBe("idle");
    });
    expect(result.current.streamErrorMessage).toBeNull();
  });

  it("Should clear stream error and revert to disabled when stream is turned off", async () => {
    const { result, rerender } = renderHook(
      ({ enabled }: { enabled: boolean }) =>
        useTaskDetailOrchestrationTab("task_001", { enabled, latestEventSeq: 0 }),
      { wrapper: createWrapper(), initialProps: { enabled: true } }
    );

    await waitFor(() => {
      expect(result.current.streamState).toBe("idle");
    });

    rerender({ enabled: false });

    await waitFor(() => {
      expect(result.current.streamState).toBe("disabled");
    });
    expect(result.current.streamErrorMessage).toBeNull();
  });

  it("Should surface load error message for the profile", async () => {
    const errorHandler = http.get(
      "/api/tasks/:id/execution-profile",
      () => new HttpResponse(JSON.stringify({ error: "boom" }), { status: 500 })
    );
    const wrapper = createWrapper();
    const previousFetch = globalThis.fetch;
    globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input.clone() : new Request(input, init);
      const response = await getResponse([errorHandler], request, {
        baseUrl: typeof window === "undefined" ? "http://localhost" : window.location.origin,
      });
      if (response) {
        return response;
      }
      return previousFetch(input, init);
    }) as typeof globalThis.fetch;

    const { result } = renderHook(
      () => useTaskDetailOrchestrationTab("task_001", { enabled: true, latestEventSeq: 0 }),
      { wrapper }
    );
    await waitFor(() => {
      expect(result.current.profileError).not.toBeNull();
    });
    globalThis.fetch = previousFetch;
  });

  it("Should resolve handleSetProfile when the mutation succeeds", async () => {
    const { result } = renderHook(
      () => useTaskDetailOrchestrationTab("task_001", { enabled: true, latestEventSeq: 0 }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => expect(result.current.profile?.task_id).toBe("task_001"));

    await act(async () => {
      await result.current.handleSetProfile({
        ...result.current.profile!,
        task_id: "task_001",
      });
    });
  });
});
