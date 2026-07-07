// Suite: useSessionLiveTail
// Invariant: transcript fetch failure never blocks stream startup or active-session self-heal.
// Boundary IN: the live-tail hook, session query options, and transcript context-facing state.
// Boundary OUT: real HTTP transport and final thread visuals, owned by adapter/provider/thread suites.
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { primarySessionFixture, sessionTranscriptFixture } from "../../mocks/fixtures";
import type { SessionMessage, SessionPayload, SessionState } from "../../types";
import { useSessionLiveTail } from "../use-session-live-tail";
import type { SessionStreamEventSource } from "../use-session-live-tail";

vi.mock("../../adapters/session-api", () => ({
  buildSessionStreamUrl: (workspaceId: string, id: string, afterSequence?: number) => {
    const base = `/api/workspaces/${encodeURIComponent(workspaceId)}/sessions/${encodeURIComponent(id)}/stream`;
    return afterSequence && afterSequence > 0 ? `${base}?after_sequence=${afterSequence}` : base;
  },
  fetchSession: vi.fn(),
  fetchSessionEvents: vi.fn(),
  fetchSessionHistory: vi.fn(),
  fetchSessionLedger: vi.fn(),
  fetchSessionRecap: vi.fn(),
  fetchSessionTranscript: vi.fn(),
  fetchSessions: vi.fn(),
  SessionApiError: class SessionApiError extends Error {
    constructor(
      message: string,
      public readonly status: number,
      public readonly sessionId?: string
    ) {
      super(message);
      this.name = "SessionApiError";
    }
  },
  SessionLedgerUnavailableError: class SessionLedgerUnavailableError extends Error {},
  SessionNotFoundError: class SessionNotFoundError extends Error {
    constructor(public readonly sessionId: string) {
      super(`Session not found: ${sessionId}`);
      this.name = "SessionNotFoundError";
    }
  },
}));

import { fetchSession, fetchSessionTranscript } from "../../adapters/session-api";

function fixtureWorkspaceId(): string {
  const workspaceId = primarySessionFixture.workspace_id;
  if (!workspaceId) {
    throw new Error("primary session fixture must include workspace_id");
  }
  return workspaceId;
}

const WORKSPACE_ID = fixtureWorkspaceId();
const SESSION_ID = primarySessionFixture.id;
const STREAM_URL = `/api/workspaces/${WORKSPACE_ID}/sessions/${SESSION_ID}/stream`;

class FakeSessionEventSource implements SessionStreamEventSource {
  readonly listeners = new Map<string, Set<EventListenerOrEventListenerObject>>();
  closed = false;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;

  constructor(readonly url: string) {}

  addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    const listeners = this.listeners.get(type) ?? new Set<EventListenerOrEventListenerObject>();
    listeners.add(listener);
    this.listeners.set(type, listeners);
  }

  removeEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    this.listeners.get(type)?.delete(listener);
  }

  close() {
    this.closed = true;
  }
}

function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function createWrapper(queryClient: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

function sessionWithState(state: SessionState): SessionPayload {
  return {
    ...primarySessionFixture,
    state,
  };
}

function renderLiveTail(options: {
  queryClient?: QueryClient;
  sources?: FakeSessionEventSource[];
}) {
  const queryClient = options.queryClient ?? createQueryClient();
  const sources = options.sources ?? [];
  const eventSourceFactory = (url: string) => {
    const source = new FakeSessionEventSource(url);
    sources.push(source);
    return source;
  };

  const rendered = renderHook(
    () =>
      useSessionLiveTail({
        workspaceId: WORKSPACE_ID,
        sessionId: SESSION_ID,
        eventSourceFactory,
      }),
    { wrapper: createWrapper(queryClient) }
  );

  return { ...rendered, queryClient, sources };
}

describe("useSessionLiveTail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(fetchSession).mockResolvedValue(sessionWithState("active"));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("Should open the EventSource when transcript fetch returns 500 and expose the error state", async () => {
    const transcriptError = new Error("transcript endpoint returned 500");
    vi.mocked(fetchSessionTranscript).mockRejectedValue(transcriptError);

    const { result, sources } = renderLiveTail({});

    await waitFor(() => {
      expect(sources).toHaveLength(1);
    });
    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    expect(sources[0]?.url).toBe(STREAM_URL);
    expect(result.current.status).toBe("error");
    expect(result.current.error).toBe(transcriptError);
    expect(result.current.messages).toEqual([]);
  });

  it("Should recover a failed transcript fetch through the active-session self-heal refetch", async () => {
    vi.useFakeTimers();
    const transcriptError = new Error("transcript endpoint returned 500");
    const recoveredMessages: SessionMessage[] = [sessionTranscriptFixture[0]!];
    vi.mocked(fetchSessionTranscript)
      .mockRejectedValueOnce(transcriptError)
      .mockResolvedValueOnce(recoveredMessages);

    const { result, sources } = renderLiveTail({});

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.isError).toBe(true);
      });
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5_000);
    });

    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.status).toBe("success");
        expect(result.current.messages).toHaveLength(1);
      });
    });
    expect(result.current.messages[0]?.id).toBe(sessionTranscriptFixture[0]?.id);
    expect(vi.mocked(fetchSessionTranscript)).toHaveBeenCalledTimes(2);
  });

  it("Should tolerate stream events without a lastEventId", async () => {
    vi.useFakeTimers();
    vi.mocked(fetchSessionTranscript).mockResolvedValue([sessionTranscriptFixture[0]!]);

    const { result, sources } = renderLiveTail({});

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.status).toBe("success");
      });
    });
    vi.mocked(fetchSessionTranscript).mockClear();

    await act(async () => {
      expect(() => {
        sources[0]?.onmessage?.({
          data: JSON.stringify({ sequence: 2 }),
        } as MessageEvent);
      }).not.toThrow();
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(120);
    });

    await act(async () => {
      await vi.waitFor(() => {
        expect(fetchSessionTranscript).toHaveBeenCalled();
      });
    });
  });

  it.each([
    ["active", true],
    ["starting", true],
    ["stopping", true],
    ["stopped", false],
  ] as const)(
    "Should run the transcript self-heal interval only for live session state %s",
    async (state, shouldRefetch) => {
      vi.useFakeTimers();
      vi.mocked(fetchSession).mockResolvedValue(sessionWithState(state));
      vi.mocked(fetchSessionTranscript).mockResolvedValue([sessionTranscriptFixture[0]!]);

      const { result, unmount, queryClient } = renderLiveTail({});

      await act(async () => {
        await vi.waitFor(() => {
          expect(result.current.status).toBe("success");
        });
      });
      vi.mocked(fetchSessionTranscript).mockClear();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(5_000);
      });

      if (shouldRefetch) {
        await act(async () => {
          await vi.waitFor(() => {
            expect(fetchSessionTranscript).toHaveBeenCalledTimes(1);
          });
        });
      } else {
        expect(fetchSessionTranscript).not.toHaveBeenCalled();
      }

      unmount();
      queryClient.clear();
    }
  );
});
