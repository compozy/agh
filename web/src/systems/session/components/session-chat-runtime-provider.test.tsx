import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { SessionThread } from "@/components/assistant-ui/session-thread";
import { primarySessionFixture, sessionTranscriptFixture } from "@/systems/session/mocks/fixtures";

import { SessionChatRuntimeProvider } from "./session-chat-runtime-provider";

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
      <SessionChatRuntimeProvider sessionId={primarySessionFixture.id}>
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

      if (pathname === `/api/sessions/${primarySessionFixture.id}`) {
        return jsonResponse({ session: primarySessionFixture });
      }

      if (pathname === `/api/sessions/${primarySessionFixture.id}/transcript`) {
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
          `/api/sessions/${primarySessionFixture.id}/transcript`
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
});
