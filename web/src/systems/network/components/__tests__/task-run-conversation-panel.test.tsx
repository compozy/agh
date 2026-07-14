import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { TaskRunConversationPanel } from "../task-run-conversation-panel";

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({ workspaceId: "ws-fixture" }),
}));

vi.mock("../../hooks/use-network-coordination", () => ({
  useNetworkUsage: () => ({
    isLoading: false,
    data: {
      workspace_id: "ws-fixture",
      details: [],
      total: {
        wake_count: 2,
        reserved_wake_count: 0,
        actual_wake_count: 2,
        unavailable_wake_count: 0,
        charged_wall_time: "1s",
        input_tokens: 10,
        output_tokens: 4,
      },
    },
  }),
}));

function wrap(node: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{node}</QueryClientProvider>;
}

describe("TaskRunConversationPanel", () => {
  it("Should explain empty conversation silence", () => {
    render(
      wrap(
        <TaskRunConversationPanel
          conversationEmpty
          messageCount={0}
          boundsLabel="Participation local"
        />
      )
    );
    expect(screen.getByTestId("tasks-run-conversation-empty")).toHaveTextContent(
      /Silence is normal/i
    );
    expect(screen.getByTestId("tasks-run-usage-summary")).toHaveTextContent(/actual/i);
  });

  it("Should keep the run view interactive while paginating long transcripts", () => {
    const onLoadMore = vi.fn();
    render(
      wrap(
        <TaskRunConversationPanel
          conversationEmpty={false}
          hasMoreMessages
          messageCount={120}
          onLoadMore={onLoadMore}
        />
      )
    );
    fireEvent.click(screen.getByTestId("tasks-run-conversation-load-more"));
    expect(onLoadMore).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("tasks-run-conversation-summary")).toHaveTextContent("120 messages");
  });
});
