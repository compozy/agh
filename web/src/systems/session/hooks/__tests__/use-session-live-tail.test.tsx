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
import { sessionKeys } from "../../lib/query-keys";
import {
  getSessionDebugCounters,
  getSessionDebugEvents,
  resetSessionDebugTelemetry,
  SESSION_DEBUG_EVENTS,
} from "../../lib/session-observability";
import { useSessionLiveTail } from "../use-session-live-tail";
import type { SessionStreamEventSource } from "../use-session-live-tail";

vi.mock("../../adapters/session-api", () => ({
  buildSessionStreamUrl: (workspaceId: string, id: string, afterSequence?: number) => {
    const base = `/api/workspaces/${encodeURIComponent(workspaceId)}/sessions/${encodeURIComponent(id)}/stream`;
    const params = new URLSearchParams({ frames: "transcript", replay: "snapshot" });
    if (afterSequence && afterSequence > 0) {
      params.set("after_sequence", String(afterSequence));
    }
    return `${base}?${params.toString()}`;
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
const STREAM_URL = `/api/workspaces/${WORKSPACE_ID}/sessions/${SESSION_ID}/stream?frames=transcript&replay=snapshot`;

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

  emit(type: string, payload: unknown, lastEventId = "") {
    const event = {
      data: JSON.stringify(payload),
      lastEventId,
    } as MessageEvent;
    for (const listener of this.listeners.get(type) ?? []) {
      if (typeof listener === "function") {
        listener(event);
      } else {
        listener.handleEvent(event);
      }
    }
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
    resetSessionDebugTelemetry();
    vi.mocked(fetchSession).mockResolvedValue(sessionWithState("active"));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    resetSessionDebugTelemetry();
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

  it("Should record transcript fetch failure and SSE lifecycle events with cursors", async () => {
    vi.useFakeTimers();
    const transcriptError = new Error("transcript endpoint returned 500");
    vi.mocked(fetchSessionTranscript).mockRejectedValue(transcriptError);

    const { result, sources } = renderLiveTail({});

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.isError).toBe(true);
      });
    });

    act(() => {
      sources[0]?.onerror?.(new Event("error"));
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(250);
    });

    const counters = getSessionDebugCounters();
    expect(counters[SESSION_DEBUG_EVENTS.transcriptFetchFailed]).toBe(1);
    expect(counters[SESSION_DEBUG_EVENTS.sseOpen]).toBe(2);
    expect(counters[SESSION_DEBUG_EVENTS.sseClose]).toBe(1);
    expect(counters[SESSION_DEBUG_EVENTS.sseReconnect]).toBe(1);
    expect(getSessionDebugEvents()).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          cursor: 0,
          event: SESSION_DEBUG_EVENTS.transcriptFetchFailed,
          error: "transcript endpoint returned 500",
          session_id: SESSION_ID,
          workspace_id: WORKSPACE_ID,
        }),
        expect.objectContaining({
          cursor: 0,
          event: SESSION_DEBUG_EVENTS.sseOpen,
          session_id: SESSION_ID,
          workspace_id: WORKSPACE_ID,
        }),
        expect.objectContaining({
          cursor: 0,
          event: SESSION_DEBUG_EVENTS.sseClose,
          reason: "reconnect",
        }),
        expect.objectContaining({
          attempt: 1,
          cursor: 0,
          delay_ms: 250,
          event: SESSION_DEBUG_EVENTS.sseReconnect,
        }),
      ])
    );
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

  it("Should apply transcript snapshot and delta frames without refetching the transcript", async () => {
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { result, sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.status).toBe("success");
      });
    });
    vi.mocked(fetchSessionTranscript).mockClear();

    await act(async () => {
      sources[0]?.emit(
        "transcript_snapshot",
        {
          session_id: SESSION_ID,
          epoch: 1,
          entries: [{ message: sessionTranscriptFixture[0]!, sequence: 4 }],
          min_sequence: 4,
          max_sequence: 4,
          reset_below: false,
        },
        "4"
      );
    });
    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages[0]?.id).toBe(sessionTranscriptFixture[0]?.id);
      });
    });
    expect(fetchSessionTranscript).not.toHaveBeenCalled();

    await act(async () => {
      sources[0]?.emit(
        "transcript_delta",
        {
          session_id: SESSION_ID,
          epoch: 1,
          entry: { message: sessionTranscriptFixture[1]!, sequence: 5 },
          sequence: 5,
        },
        "5"
      );
    });
    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages.map(message => message.id)).toEqual([
          sessionTranscriptFixture[0]?.id,
          sessionTranscriptFixture[1]?.id,
        ]);
      });
    });
    expect(fetchSessionTranscript).not.toHaveBeenCalled();
  });

  it("Should record gap recovery and refetch the transcript when delta sequence skips the cursor", async () => {
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { result, sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.status).toBe("success");
      });
    });

    await act(async () => {
      sources[0]?.emit(
        "transcript_snapshot",
        {
          session_id: SESSION_ID,
          epoch: 1,
          entries: [{ message: sessionTranscriptFixture[0]!, sequence: 2 }],
          min_sequence: 2,
          max_sequence: 2,
          reset_below: false,
        },
        "2"
      );
    });
    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages[0]?.id).toBe(sessionTranscriptFixture[0]?.id);
      });
    });
    vi.mocked(fetchSessionTranscript).mockClear();

    await act(async () => {
      sources[0]?.emit(
        "transcript_delta",
        {
          session_id: SESSION_ID,
          epoch: 1,
          entry: { message: sessionTranscriptFixture[1]!, sequence: 5 },
          sequence: 5,
        },
        "5"
      );
    });

    await act(async () => {
      await vi.waitFor(() => {
        expect(fetchSessionTranscript).toHaveBeenCalledTimes(1);
      });
    });
    expect(getSessionDebugCounters()[SESSION_DEBUG_EVENTS.gapRecoveryTriggered]).toBe(1);
    expect(getSessionDebugEvents()).toContainEqual(
      expect.objectContaining({
        cursor: 2,
        event: SESSION_DEBUG_EVENTS.gapRecoveryTriggered,
        missing_count: 2,
        next_sequence: 5,
        session_id: SESSION_ID,
        workspace_id: WORKSPACE_ID,
      })
    );
  });

  it("Should record apply failure and refetch when a transcript frame fails validation", async () => {
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { result, sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.status).toBe("success");
      });
    });
    vi.mocked(fetchSessionTranscript).mockClear();

    await act(async () => {
      sources[0]?.emit(
        "transcript_snapshot",
        {
          session_id: SESSION_ID,
          epoch: 1,
          entries: [
            {
              message: {
                id: "invalid-runtime-event",
                role: "assistant",
                parts: [
                  {
                    type: "data-agh-event",
                    data: {
                      type: "runtime_activity",
                      runtime: { turn_id: "turn-invalid" },
                    },
                  },
                ],
              },
              sequence: 8,
            },
          ],
          min_sequence: 8,
          max_sequence: 8,
          reset_below: false,
        },
        "8"
      );
    });

    await act(async () => {
      await vi.waitFor(() => {
        expect(fetchSessionTranscript).toHaveBeenCalledTimes(1);
      });
    });
    expect(result.current.messages).toEqual([]);
    expect(getSessionDebugCounters()[SESSION_DEBUG_EVENTS.transcriptApplyFailed]).toBe(1);
    expect(getSessionDebugEvents()).toContainEqual(
      expect.objectContaining({
        cursor: 8,
        event: SESSION_DEBUG_EVENTS.transcriptApplyFailed,
        frame: "snapshot",
        sequence: 8,
        session_id: SESSION_ID,
        workspace_id: WORKSPACE_ID,
      })
    );
  });

  it("Should reset the reconnect cursor when a snapshot moves to a lower epoch sequence", async () => {
    vi.useFakeTimers();
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { result, sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.status).toBe("success");
      });
    });

    await act(async () => {
      sources[0]?.emit(
        "transcript_snapshot",
        {
          session_id: SESSION_ID,
          epoch: 1,
          entries: [{ message: sessionTranscriptFixture[0]!, sequence: 50 }],
          min_sequence: 50,
          max_sequence: 50,
          reset_below: false,
        },
        "50"
      );
    });
    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages[0]?.id).toBe(sessionTranscriptFixture[0]?.id);
      });
    });

    await act(async () => {
      sources[0]?.emit(
        "transcript_snapshot",
        {
          session_id: SESSION_ID,
          epoch: 2,
          entries: [{ message: sessionTranscriptFixture[1]!, sequence: 5 }],
          min_sequence: 5,
          max_sequence: 5,
          reset_below: true,
        },
        "5"
      );
    });
    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages.map(message => message.id)).toEqual([
          sessionTranscriptFixture[1]?.id,
        ]);
      });
    });

    act(() => {
      sources[0]?.onerror?.(new Event("error"));
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(250);
    });

    expect(sources).toHaveLength(2);
    expect(sources[1]?.url).toBe(`${STREAM_URL}&after_sequence=5`);
  });

  it("Should preserve the ThreadMessage identity of unchanged messages across delta updates", async () => {
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { result, sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.status).toBe("success");
      });
    });

    await act(async () => {
      sources[0]?.emit(
        "transcript_snapshot",
        {
          session_id: SESSION_ID,
          epoch: 1,
          entries: [
            { message: sessionTranscriptFixture[0]!, sequence: 4 },
            { message: sessionTranscriptFixture[1]!, sequence: 5 },
          ],
          min_sequence: 4,
          max_sequence: 5,
          reset_below: false,
        },
        "5"
      );
    });
    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages).toHaveLength(2);
      });
    });
    const settledA = result.current.messages[0];
    const settledB = result.current.messages[1];

    // Appending a message must not re-allocate the two settled rows' ThreadMessages.
    await act(async () => {
      sources[0]?.emit(
        "transcript_delta",
        {
          session_id: SESSION_ID,
          epoch: 1,
          entry: { message: sessionTranscriptFixture[2]!, sequence: 6 },
          sequence: 6,
        },
        "6"
      );
    });
    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages).toHaveLength(3);
      });
    });
    expect(result.current.messages[0]).toBe(settledA);
    expect(result.current.messages[1]).toBe(settledB);

    // A delta that mutates the second message re-allocates only that row; the first
    // settled message keeps its ThreadMessage reference.
    const revisedB = {
      ...sessionTranscriptFixture[1]!,
      parts: [{ type: "text", text: "Revised launch summary.", state: "done" }],
    };
    await act(async () => {
      sources[0]?.emit(
        "transcript_delta",
        {
          session_id: SESSION_ID,
          epoch: 1,
          entry: { message: revisedB, sequence: 7 },
          sequence: 7,
        },
        "7"
      );
    });
    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages[1]).not.toBe(settledB);
      });
    });
    expect(result.current.messages[0]).toBe(settledA);
  });

  it("Should reconnect with after_sequence set to the latest applied sequence", async () => {
    vi.useFakeTimers();
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { result, sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.status).toBe("success");
      });
    });

    await act(async () => {
      sources[0]?.emit(
        "transcript_delta",
        {
          session_id: SESSION_ID,
          epoch: 1,
          entry: { message: sessionTranscriptFixture[0]!, sequence: 5 },
          sequence: 5,
        },
        "5"
      );
    });
    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages[0]?.id).toBe(sessionTranscriptFixture[0]?.id);
      });
    });

    act(() => {
      sources[0]?.onerror?.(new Event("error"));
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(250);
    });

    expect(sources[0]?.closed).toBe(true);
    expect(sources).toHaveLength(2);
    expect(sources[1]?.url).toBe(`${STREAM_URL}&after_sequence=5`);
  });

  it("Should back off exponentially to a bounded delay on persistent stream failures", async () => {
    vi.useFakeTimers();
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { result, sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.status).toBe("success");
      });
    });

    act(() => {
      sources[0]?.onerror?.(new Event("error"));
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(249);
    });
    expect(sources).toHaveLength(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(sources).toHaveLength(2);

    act(() => {
      sources[1]?.onerror?.(new Event("error"));
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(499);
    });
    expect(sources).toHaveLength(2);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(sources).toHaveLength(3);

    act(() => {
      sources[2]?.onerror?.(new Event("error"));
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(999);
    });
    expect(sources).toHaveLength(3);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(sources).toHaveLength(4);
  });

  it("Should not trigger a transcript refetch on stream error", async () => {
    vi.useFakeTimers();
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { result, sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.status).toBe("success");
      });
    });
    vi.mocked(fetchSessionTranscript).mockClear();

    act(() => {
      sources[0]?.onerror?.(new Event("error"));
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(250);
    });

    expect(fetchSessionTranscript).not.toHaveBeenCalled();
    expect(sources).toHaveLength(2);
  });

  it("Should apply session_stopped payload to detail and read-model caches before reconciliation", async () => {
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    vi.mocked(fetchSession).mockResolvedValue({
      ...sessionWithState("stopped"),
      stop_reason: "agent_crashed",
      failure: {
        kind: "process_exit",
        summary: "provider exited before response",
      },
    });
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { result, sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.status).toBe("success");
      });
    });
    vi.mocked(fetchSessionTranscript).mockClear();

    await act(async () => {
      sources[0]?.emit(
        "session_stopped",
        {
          id: "session-stopped-fixture",
          session_id: SESSION_ID,
          sequence: 9,
          type: "session_stopped",
          timestamp: "2026-07-07T12:00:00Z",
          turn_id: "turn-terminal",
          agent_name: primarySessionFixture.agent_name,
          content: {},
          spawn_depth: 0,
          stop_reason: "agent_crashed",
          stop_detail: "provider exited before response",
          failure: {
            kind: "process_exit",
            summary: "provider exited before response",
          },
        },
        "9"
      );
    });

    expect(sources[0]?.closed).toBe(true);
    expect(
      queryClient.getQueryData<SessionPayload>(sessionKeys.detail(WORKSPACE_ID, SESSION_ID))
    ).toMatchObject({
      state: "stopped",
      stop_reason: "agent_crashed",
      failure: {
        kind: "process_exit",
        summary: "provider exited before response",
      },
    });
    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages.at(-1)?.id).toBe(`session-stopped-${SESSION_ID}`);
      });
    });
    expect(JSON.stringify(result.current.messages.at(-1))).toContain(
      "provider exited before response"
    );
    expect(fetchSessionTranscript).not.toHaveBeenCalled();
  });

  it("Should close on session_stopped and never reconnect while stopped", async () => {
    vi.useFakeTimers();
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    vi.mocked(fetchSession).mockResolvedValue(sessionWithState("stopped"));
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
      });
    });

    act(() => {
      sources[0]?.emit(
        "session_stopped",
        {
          id: "session-stopped-fixture",
          session_id: SESSION_ID,
          sequence: 9,
          type: "session_stopped",
          timestamp: "2026-07-07T12:00:00Z",
          turn_id: "turn-terminal",
          agent_name: primarySessionFixture.agent_name,
          content: {},
          spawn_depth: 0,
        },
        "9"
      );
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(4_000);
    });

    expect(sources[0]?.closed).toBe(true);
    expect(sources).toHaveLength(1);
  });

  it("Should treat done as a terminal stream frame without reconnect churn", async () => {
    vi.useFakeTimers();
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    vi.mocked(fetchSession).mockResolvedValue(sessionWithState("stopped"));
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
      });
    });

    act(() => {
      sources[0]?.emit(
        "done",
        {
          id: "session-done-fixture",
          session_id: SESSION_ID,
          sequence: 10,
          type: "done",
          timestamp: "2026-07-07T12:00:00Z",
          turn_id: "turn-terminal",
          agent_name: primarySessionFixture.agent_name,
          content: {},
          spawn_depth: 0,
        },
        "10"
      );
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(4_000);
    });

    expect(sources[0]?.closed).toBe(true);
    expect(sources).toHaveLength(1);
  });

  it("Should reopen only when a stopped session becomes live again", async () => {
    vi.useFakeTimers();
    vi.mocked(fetchSessionTranscript).mockResolvedValue([]);
    vi.mocked(fetchSession).mockResolvedValue(sessionWithState("stopped"));
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
      });
    });

    act(() => {
      sources[0]?.emit(
        "session_stopped",
        {
          id: "session-stopped-fixture",
          session_id: SESSION_ID,
          sequence: 9,
          type: "session_stopped",
          timestamp: "2026-07-07T12:00:00Z",
          turn_id: "turn-terminal",
          agent_name: primarySessionFixture.agent_name,
          content: {},
          spawn_depth: 0,
        },
        "9"
      );
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(4_000);
    });
    expect(sources).toHaveLength(1);

    act(() => {
      queryClient.setQueryData(
        sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
        sessionWithState("active")
      );
    });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(2);
      });
    });
    expect(sources[1]?.url).toBe(`${STREAM_URL}&after_sequence=9`);
  });

  it("Should drop stale local rows when a transcript snapshot resets below the window", async () => {
    vi.mocked(fetchSessionTranscript).mockResolvedValue([sessionTranscriptFixture[0]!]);
    const queryClient = createQueryClient();
    queryClient.setQueryData(
      sessionKeys.detail(WORKSPACE_ID, SESSION_ID),
      sessionWithState("active")
    );

    const { result, sources } = renderLiveTail({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
        expect(result.current.messages.map(message => message.id)).toEqual([
          sessionTranscriptFixture[0]?.id,
        ]);
      });
    });
    vi.mocked(fetchSessionTranscript).mockClear();

    await act(async () => {
      sources[0]?.emit(
        "transcript_snapshot",
        {
          session_id: SESSION_ID,
          epoch: 2,
          entries: [{ message: sessionTranscriptFixture[1]!, sequence: 8 }],
          min_sequence: 8,
          max_sequence: 8,
          reset_below: true,
        },
        "8"
      );
    });

    await act(async () => {
      await vi.waitFor(() => {
        expect(result.current.messages.map(message => message.id)).toEqual([
          sessionTranscriptFixture[1]?.id,
        ]);
      });
    });
    expect(fetchSessionTranscript).not.toHaveBeenCalled();
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
