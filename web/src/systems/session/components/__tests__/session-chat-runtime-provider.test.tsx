import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { formatMessageError, SessionThread } from "@/components/assistant-ui/session-thread";
import { primarySessionFixture, sessionTranscriptFixture } from "@/systems/session/mocks/fixtures";
import type { TranscriptMessage } from "@/systems/session/types";

import { SessionChatRuntimeProvider } from "../session-chat-runtime-provider";

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

function renderSessionThread(
  options: {
    eventSourceFactory?: (url: string) => FakeSessionEventSource;
  } = {}
) {
  const queryClient = createQueryClient();

  return render(
    <QueryClientProvider client={queryClient}>
      <SessionChatRuntimeProvider
        sessionId={primarySessionFixture.id}
        workspaceId={primarySessionFixture.workspace_id}
        eventSourceFactory={options.eventSourceFactory}
      >
        <SessionThread
          sessionId={primarySessionFixture.id}
          agentName={primarySessionFixture.agent_name}
          canPrompt
          onCancelPrompt={() => {}}
        />
      </SessionChatRuntimeProvider>
    </QueryClientProvider>
  );
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
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    transcriptMessages = sessionTranscriptFixture.slice(0, 2);
    fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const pathname = getPathname(input);

      if (pathname === "/api/sessions") {
        return jsonResponse({ sessions: [primarySessionFixture] });
      }

      if (
        pathname ===
        `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}`
      ) {
        return jsonResponse({ session: primarySessionFixture });
      }

      if (
        pathname ===
        `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}/transcript`
      ) {
        return jsonResponse({ messages: transcriptMessages });
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
    vi.unstubAllGlobals();
  });

  it("rehydrates persisted transcript on initial mount and after remounting the same session", async () => {
    const firstRender = renderSessionThread();

    await waitFor(() => {
      expect(
        screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
      ).toBeInTheDocument();
      expect(screen.getByText("Launch readiness snapshot")).toBeInTheDocument();
    });

    firstRender.unmount();

    renderSessionThread();

    await waitFor(() => {
      expect(
        screen.getByText("Summarize the launch blockers before the 18:30 UTC cutover.")
      ).toBeInTheDocument();
      expect(screen.getByText("Launch readiness snapshot")).toBeInTheDocument();
    });

    expect(
      fetchMock.mock.calls.filter(([input]) => {
        return (
          getPathname(input as RequestInfo | URL) ===
          `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}/transcript`
        );
      })
    ).toHaveLength(2);
  }, 10_000);

  it("keeps live prompt stream output visible before transcript reconciliation", async () => {
    const user = userEvent.setup();
    renderSessionThread();

    await waitFor(() => {
      expect(screen.getByText("Launch readiness snapshot")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByTestId("composer-textarea"), {
      target: { value: "Continue from the reattached thread" },
    });
    await user.click(screen.getByTestId("composer-send-button"));

    await waitFor(() => {
      expect(
        screen.getByText("Live runtime answer before transcript reconciliation.")
      ).toBeInTheDocument();
    });

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
    const user = userEvent.setup();
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
      expect(screen.getByTestId("tool-call-card")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("thinking-trigger"));

    const chat = screen.getByTestId("chat-view");
    const chatText = chat.textContent ?? "";
    const beforeIndex = chatText.indexOf("Before search.");
    const reasoningIndex = chatText.indexOf("Need the current launch note before answering.");
    const toolIndex = chatText.indexOf("WebSearch");
    const afterIndex = chatText.indexOf("After search.");

    expect(beforeIndex).toBeGreaterThanOrEqual(0);
    expect(reasoningIndex).toBeGreaterThan(beforeIndex);
    expect(toolIndex).toBeGreaterThan(reasoningIndex);
    expect(afterIndex).toBeGreaterThan(toolIndex);
    expect(within(chat).getByTestId("tool-call-card")).toHaveTextContent("WebSearch");
  }, 10_000);

  it("renders unregistered data parts inline instead of dropping them", async () => {
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
        `/api/workspaces/${primarySessionFixture.workspace_id}/sessions/${primarySessionFixture.id}/stream`
      );
    });

    transcriptMessages = [
      ...sessionTranscriptFixture.slice(0, 1),
      {
        id: "transcript_assistant_live_tail_001",
        role: "assistant",
        parts: [{ type: "text", text: "Live reattached answer.", state: "done" }],
      },
    ];
    sources[0]?.dispatch(
      "agent_message",
      {
        sequence: 1,
        type: "agent_message",
        session_id: primarySessionFixture.id,
        turn_id: "turn_live_001",
        text: "Live reattached answer.",
      },
      "1"
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

    transcriptMessages = [
      ...sessionTranscriptFixture.slice(0, 1),
      {
        id: "transcript_gap_recovery_001",
        role: "assistant",
        parts: [{ type: "text", text: "Recovered from the durable transcript.", state: "done" }],
      },
    ];
    sources[0]?.dispatch(
      "agent_message",
      {
        sequence: 4,
        type: "agent_message",
        session_id: primarySessionFixture.id,
        turn_id: "turn_gap_001",
        text: "Recovered from the durable transcript.",
      },
      "4"
    );

    await waitFor(() => {
      expect(screen.getByText("Recovered from the durable transcript.")).toBeInTheDocument();
    });

    const transcriptFetches = fetchMock.mock.calls.filter(([input]) =>
      getPathname(input as RequestInfo | URL).endsWith(`/${primarySessionFixture.id}/transcript`)
    );
    expect(transcriptFetches.length).toBeGreaterThanOrEqual(2);
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
