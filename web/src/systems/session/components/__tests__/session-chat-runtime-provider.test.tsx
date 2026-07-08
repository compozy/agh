import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { formatMessageError, SessionThread } from "@/components/assistant-ui/session-thread";
import { sessionKeys, useSessionTranscriptThreadState } from "@/systems/session";
import { mergeSessionThreadReadModel } from "@/systems/session/lib/session-thread-read-model";
import { toReadonlyThreadMessages } from "@/systems/session/lib/session-thread-repository";
import { primarySessionFixture, sessionTranscriptFixture } from "@/systems/session/mocks/fixtures";
import type { TranscriptMessage } from "@/systems/session/types";

import { SessionChatRuntimeProvider } from "../session-chat-runtime-provider";

const SESSION_STREAM_QUERY = "frames=transcript&replay=snapshot";

describe("formatMessageError", () => {
  it("extracts provider failure detail from JSON-RPC error envelopes", () => {
    expect(
      formatMessageError(
        '{"code":-32603,"message":"Internal error","data":{"error":"peer disconnected before response"}}'
      )
    ).toBe("peer disconnected before response");
  });

  it("does not produce empty message chrome for blank provider errors", () => {
    expect(formatMessageError("")).toBeNull();
  });

  it("does not render raw JSON when no provider error detail exists", () => {
    expect(formatMessageError('{"type":"abort"}')).toBeNull();
  });
});

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
}

function transcriptEntries(messages: TranscriptMessage[]) {
  return messages.map((message, index) => ({ message, sequence: index + 1 }));
}

function sseResponse(frames: string[]) {
  const encoder = new TextEncoder();
  return new Response(
    new ReadableStream({
      start(controller) {
        for (const frame of frames) {
          controller.enqueue(encoder.encode(frame));
        }
        controller.close();
      },
    }),
    {
      headers: {
        "Content-Type": "text/event-stream",
        "x-vercel-ai-ui-message-stream": "v1",
      },
    }
  );
}

function getPathname(input: RequestInfo | URL): string {
  if (typeof input === "string") {
    return new URL(input, "http://localhost").pathname;
  }

  if (input instanceof URL) {
    return input.pathname;
  }

  return new URL(input.url, "http://localhost").pathname;
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });
  return { promise, reject, resolve };
}

class FakeSessionEventSource {
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

  dispatch(type: string, payload: unknown, lastEventId: string) {
    const event = new MessageEvent(type, { data: JSON.stringify(payload) });
    Object.defineProperty(event, "lastEventId", { value: lastEventId });
    if (type === "message") {
      this.onmessage?.(event);
    }
    for (const listener of this.listeners.get(type) ?? []) {
      if (typeof listener === "function") {
        listener(event);
      } else {
        listener.handleEvent(event);
      }
    }
  }
}

function TranscriptStateProbe() {
  const state = useSessionTranscriptThreadState();

  return (
    <div
      data-testid="transcript-state-probe"
      data-status={state.status}
      data-is-error={String(state.isError)}
      data-is-pending={String(state.isPending)}
      data-error-message={state.error?.message ?? ""}
    >
      {state.messages.length}
    </div>
  );
}

function renderSessionThread(
  options: {
    eventSourceFactory?: (url: string) => FakeSessionEventSource;
    queryClient?: QueryClient;
    includeTranscriptStateProbe?: boolean;
  } = {}
) {
  const queryClient = options.queryClient ?? createQueryClient();

  const utils = render(
    <QueryClientProvider client={queryClient}>
      <SessionChatRuntimeProvider
        sessionId={primarySessionFixture.id}
        workspaceId={fixtureWorkspaceId()}
        eventSourceFactory={options.eventSourceFactory}
      >
        {options.includeTranscriptStateProbe ? <TranscriptStateProbe /> : null}
        <SessionThread
          sessionId={primarySessionFixture.id}
          agentName={primarySessionFixture.agent_name}
          canPrompt
          onCancelPrompt={() => {}}
        />
      </SessionChatRuntimeProvider>
    </QueryClientProvider>
  );
  return { ...utils, queryClient };
}

function countTranscriptFetches(fetchMock: ReturnType<typeof vi.fn>): number {
  return fetchMock.mock.calls.filter(([input]) => {
    return (
      getPathname(input as RequestInfo | URL) ===
      `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}/transcript`
    );
  }).length;
}

function fixtureWorkspaceId(): string {
  const workspaceId = primarySessionFixture.workspace_id;
  if (!workspaceId) {
    throw new Error("primary session fixture must include workspace_id");
  }
  return workspaceId;
}

describe("SessionChatRuntimeProvider", () => {
  it("Should align viewport and composer on the shared thread content rail", async () => {
    renderSessionThread();

    await waitFor(() => {
      expect(screen.getByTestId("chat-view")).toBeInTheDocument();
    });

    const chatView = screen.getByTestId("chat-view");
    const viewportRail = within(chatView).getByTestId("thread-content-rail");
    expect(viewportRail).toHaveClass("px-4");

    const composerShell = screen.getByTestId("composer-shell");
    const composerRail = within(composerShell).getByTestId("thread-content-rail");
    expect(composerRail).toHaveClass("px-4");
  });

  let transcriptMessages = sessionTranscriptFixture.slice(0, 2);
  let sessionDetailResponse = primarySessionFixture;
  let transcriptFetchShouldFail = false;
  let transcriptResponsePromise: Promise<Response> | null = null;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    transcriptMessages = sessionTranscriptFixture.slice(0, 2);
    sessionDetailResponse = primarySessionFixture;
    transcriptFetchShouldFail = false;
    transcriptResponsePromise = null;
    fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const pathname = getPathname(input);

      if (pathname === "/api/sessions") {
        return jsonResponse({ sessions: [primarySessionFixture] });
      }

      if (
        pathname ===
        `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}`
      ) {
        return jsonResponse({ session: sessionDetailResponse });
      }

      if (
        pathname ===
        `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}/transcript`
      ) {
        if (transcriptResponsePromise) {
          return transcriptResponsePromise;
        }
        if (transcriptFetchShouldFail) {
          return jsonResponse({ error: "recorder temporarily unavailable" }, { status: 500 });
        }
        return jsonResponse({ entries: transcriptEntries(transcriptMessages) });
      }

      if (
        pathname ===
        `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}/prompt`
      ) {
        return sseResponse([
          'data: {"type":"start","messageId":"turn-runtime-001"}\n\n',
          'data: {"type":"text-start","id":"turn-runtime-001-text-1"}\n\n',
          'data: {"type":"text-delta","id":"turn-runtime-001-text-1","delta":"Live runtime answer before transcript reconciliation."}\n\n',
          'data: {"type":"text-end","id":"turn-runtime-001-text-1"}\n\n',
          'data: {"type":"finish","finishReason":"stop"}\n\n',
          "data: [DONE]\n\n",
        ]);
      }

      throw new Error(`Unhandled fetch in test: ${pathname}`);
    });

    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("Should still render from cache after fake timers advance beyond the old 5-minute gcTime default", async () => {
    vi.useFakeTimers();
    const queryClient = createQueryClient();
    const workspaceId = fixtureWorkspaceId();
    const sessionId = primarySessionFixture.id;
    queryClient.setQueryData(sessionKeys.detail(workspaceId, sessionId), primarySessionFixture);
    queryClient.setQueryData(sessionKeys.transcript(workspaceId, sessionId), transcriptMessages);

    const firstRender = renderSessionThread({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(
          screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
        ).toBeInTheDocument();
        expect(screen.getByText("Launch readiness snapshot")).toBeInTheDocument();
      });
    });

    firstRender.unmount();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5 * 60 * 1_000 + 1);
    });

    expect(queryClient.getQueryData(sessionKeys.transcript(workspaceId, sessionId))).toEqual(
      transcriptMessages
    );

    renderSessionThread({ queryClient });

    await act(async () => {
      await vi.waitFor(() => {
        expect(screen.queryByTestId("thread-transcript-skeleton")).not.toBeInTheDocument();
        expect(screen.queryByText("Start a conversation.")).not.toBeInTheDocument();
        expect(
          screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
        ).toBeInTheDocument();
        expect(screen.getByText("Launch readiness snapshot")).toBeInTheDocument();
      });
    });
  }, 10_000);

  it("renders transcript rows when the runtime thread is transiently empty", async () => {
    transcriptMessages = sessionTranscriptFixture.slice(0, 3);

    renderSessionThread();

    await waitFor(() => {
      expect(
        screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
      ).toBeInTheDocument();
      expect(screen.getByText("Launch readiness snapshot")).toBeInTheDocument();
      expect(screen.getByTestId("tool-call-row")).toBeInTheDocument();
    });

    expect(screen.queryByText("Start a conversation.")).not.toBeInTheDocument();
  }, 10_000);

  it("Should show the skeleton on cold provider mount until the transcript resolves", async () => {
    const transcriptResponse = createDeferred<Response>();
    transcriptResponsePromise = transcriptResponse.promise;

    renderSessionThread();

    await waitFor(() => {
      expect(countTranscriptFetches(fetchMock)).toBe(1);
    });
    expect(screen.getByTestId("thread-transcript-skeleton")).toBeInTheDocument();
    expect(screen.queryByText("Start a conversation.")).not.toBeInTheDocument();

    transcriptResponse.resolve(jsonResponse({ entries: transcriptEntries(transcriptMessages) }));

    expect(
      await screen.findByText("Summarize the launch blockers before the 18:30 UTC cutover.")
    ).toBeInTheDocument();
    expect(screen.getByText("Launch readiness snapshot")).toBeInTheDocument();
    expect(screen.queryByTestId("thread-transcript-skeleton")).not.toBeInTheDocument();
    expect(screen.queryByText("Start a conversation.")).not.toBeInTheDocument();
  }, 10_000);

  it("Should issue one transcript fetch per cold provider mount after the transcript settles", async () => {
    renderSessionThread();

    await waitFor(() => {
      expect(screen.getByText("Launch readiness snapshot")).toBeInTheDocument();
    });

    expect(countTranscriptFetches(fetchMock)).toBe(1);
  }, 10_000);

  it("Should fail loudly when workspaceId is empty", () => {
    const queryClient = createQueryClient();
    expect(() =>
      render(
        <QueryClientProvider client={queryClient}>
          <SessionChatRuntimeProvider sessionId={primarySessionFixture.id} workspaceId=" ">
            <SessionThread
              sessionId={primarySessionFixture.id}
              agentName={primarySessionFixture.agent_name}
              canPrompt
              onCancelPrompt={() => {}}
            />
          </SessionChatRuntimeProvider>
        </QueryClientProvider>
      )
    ).toThrow("SessionChatRuntimeProvider requires a non-empty workspaceId");
  });

  it("keeps locally sent prompts visible through transcript reconciliation", async () => {
    const user = userEvent.setup();
    const { queryClient } = renderSessionThread();

    await waitFor(() => {
      expect(screen.getByText("Launch readiness snapshot")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByTestId("composer-textarea"), {
      target: { value: "Continue from the reattached thread" },
    });
    await user.click(screen.getByTestId("composer-send-button"));

    await waitFor(() => {
      expect(screen.getByText("Continue from the reattached thread")).toBeInTheDocument();
      expect(
        screen.getByText("Live runtime answer before transcript reconciliation.")
      ).toBeInTheDocument();
    });

    transcriptMessages = [
      ...sessionTranscriptFixture.slice(0, 2),
      {
        id: "transcript_user_after_send_001",
        role: "user",
        parts: [
          {
            type: "text",
            text: "Continue from the reattached thread",
            state: "done",
          },
        ],
      },
      {
        id: "transcript_assistant_after_send_001",
        role: "assistant",
        parts: [
          {
            type: "text",
            text: "Durable transcript answer after reconciliation.",
            state: "done",
          },
        ],
      },
    ];
    await queryClient.invalidateQueries({
      queryKey: sessionKeys.transcript(fixtureWorkspaceId(), primarySessionFixture.id),
    });

    await waitFor(() => {
      expect(screen.getAllByText("Continue from the reattached thread").length).toBeGreaterThan(0);
      expect(
        screen.getByText("Durable transcript answer after reconciliation.")
      ).toBeInTheDocument();
    });

    expect(
      screen.queryByText("Live runtime answer before transcript reconciliation.")
    ).not.toBeInTheDocument();
    expect(
      transcriptMessages.some(message => JSON.stringify(message).includes("Live runtime answer"))
    ).toBe(false);
    expect(
      fetchMock.mock.calls.some(([input]) => {
        return (
          getPathname(input as RequestInfo | URL) ===
          `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}/prompt`
        );
      })
    ).toBe(true);
  }, 10_000);

  it("promotes an optimistic runtime message to the server identity without a duplicate row", () => {
    const runtimeMessages = toReadonlyThreadMessages([
      {
        id: "client_temp_assistant_001",
        role: "assistant",
        metadata: { turn_id: "turn_promote_001" },
        parts: [
          {
            type: "text",
            text: "Promoted assistant response.",
            state: "done",
          },
        ],
      } as TranscriptMessage,
    ]);
    const transcriptMessages = toReadonlyThreadMessages([
      {
        id: "server_assistant_001",
        role: "assistant",
        metadata: {
          turn_id: "turn_promote_001",
          client_temp_id: "client_temp_assistant_001",
        },
        parts: [
          {
            type: "text",
            text: "Promoted assistant response.",
            state: "done",
          },
        ],
      } as TranscriptMessage,
    ]);

    const merged = mergeSessionThreadReadModel({ transcriptMessages, runtimeMessages });

    expect(merged.map(message => message.id)).toEqual(["server_assistant_001"]);
  });

  it("Should let an empty authoritative transcript replace a stale runtime tail", () => {
    const runtimeMessages = toReadonlyThreadMessages([
      {
        id: "stale_runtime_user_001",
        role: "user",
        parts: [{ type: "text", text: "Old prompt before clear", state: "done" }],
      } as TranscriptMessage,
    ]);

    const merged = mergeSessionThreadReadModel({
      transcriptMessages: [],
      runtimeMessages,
      includeRuntimeTail: false,
    });

    expect(merged).toEqual([]);
  });

  it("Should let same-count authoritative transcript content replace stale runtime content", () => {
    const runtimeMessages = toReadonlyThreadMessages([
      {
        id: "stale_runtime_assistant_001",
        role: "assistant",
        parts: [{ type: "text", text: "Old assistant text", state: "done" }],
      } as TranscriptMessage,
    ]);
    const transcriptMessages = toReadonlyThreadMessages([
      {
        id: "server_assistant_replacement_001",
        role: "assistant",
        parts: [{ type: "text", text: "Replacement assistant text", state: "done" }],
      } as TranscriptMessage,
    ]);

    const merged = mergeSessionThreadReadModel({
      transcriptMessages,
      runtimeMessages,
      includeRuntimeTail: false,
    });

    expect(merged.map(message => message.id)).toEqual(["server_assistant_replacement_001"]);
  });

  it("renders runtime progress events as activity notices instead of assistant text", async () => {
    transcriptMessages = [
      ...sessionTranscriptFixture.slice(0, 1),
      {
        id: "transcript_runtime_001",
        role: "assistant",
        parts: [
          {
            type: "data-agh-event",
            data: {
              type: "runtime_progress",
              text: "Still working",
              runtime: {
                turn_id: "turn_001",
                current_tool: "Bash",
                elapsed_ms: 610_000,
                elapsed_seconds: 610,
                idle_seconds: 30,
              },
            },
          },
        ],
      },
    ];

    renderSessionThread();

    await waitFor(() => {
      expect(screen.getByTestId("runtime-activity-notice")).toBeInTheDocument();
    });

    expect(screen.getByTestId("runtime-activity-notice")).toHaveTextContent("Still working");
    expect(screen.getByTestId("runtime-activity-detail")).toHaveTextContent("Using Bash");
  }, 10_000);

  it("renders persisted session error events as failure notices", async () => {
    transcriptMessages = [
      ...sessionTranscriptFixture.slice(0, 1),
      {
        id: "transcript_error_001",
        role: "assistant",
        parts: [
          {
            type: "text",
            text: "Partial response before failure.",
            state: "done",
          },
          {
            type: "data-agh-event",
            data: {
              type: "error",
              error:
                '{"code":-32603,"message":"Internal error","data":{"error":"peer disconnected before response"}}',
              failure: {
                kind: "process_exit",
                summary: "peer disconnected before response",
              },
            },
          },
        ],
      },
    ];

    renderSessionThread();

    await waitFor(() => {
      expect(screen.getByTestId("session-error-notice")).toBeInTheDocument();
    });

    expect(screen.getByText("Partial response before failure.")).toBeInTheDocument();
    expect(screen.getByTestId("session-error-notice")).toHaveTextContent("Session failed");
    expect(screen.getByTestId("session-error-detail")).toHaveTextContent(
      "peer disconnected before response"
    );
  }, 10_000);

  it("does not render empty error chrome for incomplete message status without a detail", async () => {
    transcriptMessages = [
      ...sessionTranscriptFixture.slice(0, 1),
      {
        id: "transcript_empty_error_status_001",
        role: "assistant",
        status: {
          type: "incomplete",
          reason: "error",
          error: "",
        },
        parts: [
          {
            type: "data-agh-event",
            data: {
              type: "error",
            },
          },
        ],
      } as unknown as TranscriptMessage,
    ];

    renderSessionThread();

    await waitFor(() => {
      expect(
        screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
      ).toBeInTheDocument();
    });

    expect(screen.queryByTestId("session-message-error")).not.toBeInTheDocument();
    expect(screen.queryByTestId("session-error-notice")).not.toBeInTheDocument();
  }, 10_000);

  it("renders only unresolved permission events as interactive prompts", async () => {
    transcriptMessages = [
      ...sessionTranscriptFixture.slice(0, 1),
      {
        id: "transcript_permission_001",
        role: "assistant",
        parts: [
          {
            type: "data-agh-permission",
            data: {
              type: "permission",
              request_id: "turn_001:perm_pending",
              title: "Edit pending file",
              resource: "pending.txt",
              action: "session/request_permission",
              raw: { path: "pending.txt" },
            },
          },
          {
            type: "data-agh-permission",
            data: {
              type: "permission",
              request_id: "turn_001:perm_resolved",
              title: "Edit resolved file",
              resource: "resolved.txt",
              action: "session/request_permission",
              decision: "reject-always",
              raw: { path: "resolved.txt" },
            },
          },
        ],
      },
    ];

    renderSessionThread();

    await waitFor(() => {
      expect(screen.getByTestId("permission-prompt")).toBeInTheDocument();
    });

    expect(screen.getAllByTestId("permission-prompt")).toHaveLength(1);
    expect(screen.getByTestId("permission-prompt")).toHaveTextContent("pending.txt");
  }, 10_000);

  it("renders mixed text, reasoning, and unregistered tool parts inline in order", async () => {
    transcriptMessages = [
      ...sessionTranscriptFixture.slice(0, 1),
      {
        id: "transcript_mixed_parts_001",
        role: "assistant",
        parts: [
          {
            type: "text",
            text: "Before search.",
            state: "done",
          },
          {
            type: "reasoning",
            text: "Need the current launch note before answering.",
            state: "streaming",
          },
          {
            type: "tool-WebSearch",
            toolCallId: "tool_web_001",
            state: "output-available",
            input: {
              query: "launch note",
            },
            output: {
              type: "tool_result",
              title: "WebSearch",
              raw: {
                content: "Launch note found.",
              },
            },
          },
          {
            type: "text",
            text: "After search.",
            state: "done",
          },
        ],
      },
    ];

    renderSessionThread();

    await waitFor(() => {
      expect(screen.getByTestId("tool-call-row")).toBeInTheDocument();
    });

    // The reasoning part is still streaming, so the flattened ThinkingBlock renders
    // it auto-open inline — no toggle needed to read it in order.
    const chat = screen.getByTestId("chat-view");
    const chatText = chat.textContent ?? "";
    const beforeIndex = chatText.indexOf("Before search.");
    const reasoningIndex = chatText.indexOf("Need the current launch note before answering.");
    // The row heading is the visible tense-aware verb ("Searched web"), not the raw tool id.
    const toolIndex = chatText.indexOf("Searched web");
    const afterIndex = chatText.indexOf("After search.");

    expect(beforeIndex).toBeGreaterThanOrEqual(0);
    expect(reasoningIndex).toBeGreaterThan(beforeIndex);
    expect(toolIndex).toBeGreaterThan(reasoningIndex);
    expect(afterIndex).toBeGreaterThan(toolIndex);
    expect(within(chat).getByTestId("tool-call-row")).toHaveTextContent("Searched web");
  }, 10_000);

  it("keeps unregistered data parts inside the settled turn fold instead of dropping them", async () => {
    const user = userEvent.setup();
    transcriptMessages = [
      ...sessionTranscriptFixture.slice(0, 1),
      {
        id: "transcript_unknown_data_001",
        role: "assistant",
        parts: [
          {
            type: "text",
            text: "Before data.",
            state: "done",
          },
          {
            type: "data-provider-note",
            data: {
              title: "Provider note",
              detail: "Unregistered data event",
            },
          },
          {
            type: "text",
            text: "After data.",
            state: "done",
          },
        ],
      },
    ];

    renderSessionThread();

    // The settled turn folds its preamble + data behind a "Worked" disclosure and
    // keeps the terminal answer visible; the data part is folded away, not dropped.
    const fold = await screen.findByRole("button", { name: "Worked" });
    expect(screen.getByText("After data.")).toBeInTheDocument();
    expect(screen.queryByTestId("session-data-part")).not.toBeInTheDocument();

    await user.click(fold);

    await waitFor(() => {
      expect(screen.getByTestId("session-data-part")).toBeInTheDocument();
    });

    const chat = screen.getByTestId("chat-view");
    const chatText = chat.textContent ?? "";
    const beforeIndex = chatText.indexOf("Before data.");
    const dataIndex = chatText.indexOf("provider-note");
    const afterIndex = chatText.indexOf("After data.");

    expect(beforeIndex).toBeGreaterThanOrEqual(0);
    expect(dataIndex).toBeGreaterThan(beforeIndex);
    expect(afterIndex).toBeGreaterThan(dataIndex);
    expect(within(chat).getByTestId("session-data-part")).toHaveTextContent("Provider note");
  }, 10_000);

  it("reconciles durable transcript after a live session stream event", async () => {
    transcriptMessages = sessionTranscriptFixture.slice(0, 1);
    const sources: FakeSessionEventSource[] = [];

    renderSessionThread({
      eventSourceFactory: url => {
        const source = new FakeSessionEventSource(url);
        sources.push(source);
        return source;
      },
    });

    await waitFor(() => {
      expect(
        screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
      ).toBeInTheDocument();
    });
    await waitFor(() => {
      expect(sources[0]?.url).toBe(
        `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}/stream?${SESSION_STREAM_QUERY}`
      );
    });

    const liveMessage: TranscriptMessage = {
      id: "transcript_assistant_live_tail_001",
      role: "assistant",
      parts: [{ type: "text", text: "Live reattached answer.", state: "done" }],
    };
    sources[0]?.dispatch(
      "transcript_delta",
      {
        session_id: primarySessionFixture.id,
        epoch: 0,
        entry: { message: liveMessage, sequence: 2 },
        sequence: 2,
      },
      "2"
    );

    await waitFor(() => {
      expect(screen.getByText("Live reattached answer.")).toBeInTheDocument();
    });
  }, 10_000);

  it("recovers stream gaps from durable transcript before resuming from the latest cursor", async () => {
    transcriptMessages = sessionTranscriptFixture.slice(0, 1);
    const sources: FakeSessionEventSource[] = [];

    renderSessionThread({
      eventSourceFactory: url => {
        const source = new FakeSessionEventSource(url);
        sources.push(source);
        return source;
      },
    });

    await waitFor(() => {
      expect(sources).toHaveLength(1);
    });
    await waitFor(() => {
      expect(sources[0]?.listeners.get("transcript_snapshot")?.size).toBeGreaterThan(0);
      expect(
        screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
      ).toBeInTheDocument();
    });

    const recoveredMessage: TranscriptMessage = {
      id: "transcript_gap_recovery_001",
      role: "assistant",
      parts: [{ type: "text", text: "Recovered from the durable transcript.", state: "done" }],
    };
    await act(async () => {
      sources[0]?.dispatch(
        "transcript_snapshot",
        {
          session_id: primarySessionFixture.id,
          epoch: 0,
          entries: [
            { message: sessionTranscriptFixture[0]!, sequence: 1 },
            { message: recoveredMessage, sequence: 4 },
          ],
          min_sequence: 1,
          max_sequence: 4,
          reset_below: true,
        },
        "4"
      );
    });

    await waitFor(() => {
      expect(screen.getByText("Recovered from the durable transcript.")).toBeInTheDocument();
    });

    expect(countTranscriptFetches(fetchMock)).toBe(1);
  }, 10_000);

  it("Should close the provider stream on terminal frames and reopen only after the session becomes live", async () => {
    vi.useFakeTimers();
    transcriptMessages = sessionTranscriptFixture.slice(0, 1);
    const sources: FakeSessionEventSource[] = [];
    const queryClient = createQueryClient();
    const workspaceId = fixtureWorkspaceId();
    const sessionId = primarySessionFixture.id;
    queryClient.setQueryData(sessionKeys.detail(workspaceId, sessionId), primarySessionFixture);

    renderSessionThread({
      queryClient,
      eventSourceFactory: url => {
        const source = new FakeSessionEventSource(url);
        sources.push(source);
        return source;
      },
    });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(1);
      });
    });

    sessionDetailResponse = { ...primarySessionFixture, state: "stopped" };
    act(() => {
      sources[0]?.dispatch(
        "session_stopped",
        {
          id: "session-stopped-fixture",
          session_id: sessionId,
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

    act(() => {
      queryClient.setQueryData(sessionKeys.detail(workspaceId, sessionId), primarySessionFixture);
    });

    await act(async () => {
      await vi.waitFor(() => {
        expect(sources).toHaveLength(2);
      });
    });
    expect(sources[1]?.url).toBe(
      `/api/workspaces/${workspaceId}/sessions/${sessionId}/stream?${SESSION_STREAM_QUERY}&after_sequence=9`
    );
  }, 10_000);

  it("Should expose transcript fetch failure through provider context while attempting the stream", async () => {
    transcriptFetchShouldFail = true;
    const sources: FakeSessionEventSource[] = [];

    renderSessionThread({
      includeTranscriptStateProbe: true,
      eventSourceFactory: url => {
        const source = new FakeSessionEventSource(url);
        sources.push(source);
        return source;
      },
    });

    await waitFor(() => {
      expect(sources).toHaveLength(1);
    });
    await waitFor(() => {
      expect(screen.getByTestId("transcript-state-probe")).toHaveAttribute("data-status", "error");
    });

    const state = screen.getByTestId("transcript-state-probe");
    expect(sources[0]?.url).toBe(
      `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}/stream?${SESSION_STREAM_QUERY}`
    );
    expect(state).toHaveAttribute("data-is-error", "true");
    expect(state).toHaveTextContent("0");
    expect(state.getAttribute("data-error-message")).toContain("recorder temporarily unavailable");
  }, 10_000);

  it("virtualizes large transcript histories while preserving visible message order", async () => {
    transcriptMessages = Array.from({ length: 80 }, (_, index) => ({
      id: `transcript_large_${index}`,
      role: index % 2 === 0 ? "user" : "assistant",
      parts: [{ type: "text", text: `Large transcript message ${index}`, state: "done" }],
    })) as TranscriptMessage[];

    renderSessionThread();

    await waitFor(() => {
      expect(screen.getByTestId("virtualized-thread-messages")).toBeInTheDocument();
    });

    const rowIndexes = screen
      .getAllByTestId("virtualized-thread-row")
      .map(row => Number(row.getAttribute("data-index")));
    expect(rowIndexes.length).toBeGreaterThan(0);
    expect(rowIndexes).toEqual([...rowIndexes].sort((left, right) => left - right));
  }, 10_000);
});
