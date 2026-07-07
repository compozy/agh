import type { ThreadMessage } from "@assistant-ui/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { SessionChatRuntimeProvider } from "@/systems/session/components/session-chat-runtime-provider";
import { primarySessionFixture, sessionTranscriptFixture } from "@/systems/session/mocks/fixtures";
import { SessionTranscriptThreadProvider } from "@/systems/session/lib/session-transcript-thread-context";
import { toReadonlyThreadMessages } from "@/systems/session/lib/session-thread-repository";
import type { SessionTranscriptThreadStatus } from "@/systems/session/lib/session-transcript-thread-context-value";

import { SessionThread } from "../session-thread";

// Suite: thread state panes
// Invariant: ThreadEmpty renders only after the transcript fetch succeeds with zero messages.
// Boundary IN: SessionThread state branching and readonly row rendering.
// Boundary OUT: transcript query/refetch wiring, covered by session-chat-runtime-provider.test.tsx.

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
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

function createFetchMock() {
  return vi.fn(async (input: RequestInfo | URL) => {
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
      return jsonResponse({ messages: [] });
    }

    throw new Error(`Unhandled fetch in thread test: ${pathname}`);
  });
}

function renderThreadState({
  messages = [],
  status,
  error = null,
  retry = vi.fn(),
}: {
  messages?: readonly ThreadMessage[];
  status: SessionTranscriptThreadStatus;
  error?: Error | null;
  retry?: () => void;
}) {
  const queryClient = createQueryClient();

  render(
    <QueryClientProvider client={queryClient}>
      <SessionChatRuntimeProvider
        sessionId={primarySessionFixture.id}
        workspaceId={primarySessionFixture.workspace_id}
      >
        <SessionTranscriptThreadProvider
          messages={messages}
          status={status}
          isPending={status === "pending"}
          isError={status === "error"}
          error={error}
          retry={retry}
        >
          <SessionThread
            sessionId={primarySessionFixture.id}
            agentName={primarySessionFixture.agent_name}
            canPrompt
            onCancelPrompt={() => {}}
          />
        </SessionTranscriptThreadProvider>
      </SessionChatRuntimeProvider>
    </QueryClientProvider>
  );
}

describe("SessionThread transcript states", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", createFetchMock());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("Should render the transcript skeleton on pending without empty-state copy", async () => {
    renderThreadState({ status: "pending" });

    expect(
      await screen.findByRole("status", {
        name: /loading transcript/i,
      })
    ).toBeInTheDocument();
    expect(screen.getByTestId("thread-transcript-skeleton")).toBeInTheDocument();
    expect(screen.queryByText(/Start a conversation/i)).not.toBeInTheDocument();
  });

  it("Should render a retryable transcript error pane and call retry", async () => {
    const user = userEvent.setup();
    const retry = vi.fn();

    renderThreadState({
      status: "error",
      error: new Error("recorder temporarily unavailable"),
      retry,
    });

    const pane = await screen.findByTestId("thread-transcript-error");
    expect(pane).toHaveAttribute("role", "alert");
    expect(within(pane).getByText("Transcript unavailable")).toBeInTheDocument();
    expect(screen.getByTestId("thread-transcript-error-detail")).toHaveTextContent(
      "recorder temporarily unavailable"
    );
    expect(screen.queryByText(/Start a conversation/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /retry transcript/i }));

    expect(retry).toHaveBeenCalledTimes(1);
  });

  it("Should render ThreadEmpty only for success with zero messages", async () => {
    renderThreadState({ status: "success" });

    expect(await screen.findByText(/Start a conversation/i)).toBeInTheDocument();
    expect(screen.queryByTestId("thread-transcript-skeleton")).not.toBeInTheDocument();
    expect(screen.queryByTestId("thread-transcript-error")).not.toBeInTheDocument();
  });

  it("Should render rows for success with transcript messages", async () => {
    const messages = toReadonlyThreadMessages(sessionTranscriptFixture.slice(0, 2));

    renderThreadState({ status: "success", messages });

    expect(await screen.findByText("Launch readiness snapshot")).toBeInTheDocument();
    expect(screen.getByTestId("virtualized-thread-messages")).toBeInTheDocument();
    expect(screen.queryByText(/Start a conversation/i)).not.toBeInTheDocument();
  });
});
