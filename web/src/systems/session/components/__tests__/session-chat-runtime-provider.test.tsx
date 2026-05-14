import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { SessionThread } from "@/components/assistant-ui/session-thread";
import { primarySessionFixture, sessionTranscriptFixture } from "@/systems/session/mocks/fixtures";

import { SessionChatRuntimeProvider } from "../session-chat-runtime-provider";

function jsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
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

function renderSessionThread() {
  const queryClient = createQueryClient();

  return render(
    <QueryClientProvider client={queryClient}>
      <SessionChatRuntimeProvider
        sessionId={primarySessionFixture.id}
        workspaceId={primarySessionFixture.workspace_id}
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
});
