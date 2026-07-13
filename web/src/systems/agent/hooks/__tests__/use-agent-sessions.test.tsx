import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/systems/session/adapters/session-api", async importOriginal => ({
  ...(await importOriginal()),
  fetchSessions: vi.fn(),
}));

import { fetchSessions } from "@/systems/session/adapters/session-api";
import { useAgentSessions } from "../use-agent-sessions";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("useAgentSessions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(fetchSessions).mockResolvedValue({
      sessions: [],
      page: { has_more: false, limit: 50, total: 0 },
    });
  });

  it("Should wait for an explicit workspace before issuing one agent session query", async () => {
    const initialProps: { workspaceId: string | null } = { workspaceId: null };
    const { rerender } = renderHook(
      ({ workspaceId }) => useAgentSessions(workspaceId, "claude-agent"),
      {
        initialProps,
        wrapper: createWrapper(),
      }
    );

    expect(fetchSessions).not.toHaveBeenCalled();

    rerender({ workspaceId: "ws_alpha" });

    await waitFor(() => {
      expect(fetchSessions).toHaveBeenCalledTimes(1);
    });
    for (const [filters] of vi.mocked(fetchSessions).mock.calls) {
      expect(filters?.workspace).toBe("ws_alpha");
    }
  });
});
