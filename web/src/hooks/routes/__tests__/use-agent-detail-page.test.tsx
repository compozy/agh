import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { mockInvalidateQueries, mockNavigate, mockUseAgent, mockUseAgentSessions } = vi.hoisted(
  () => ({
    mockInvalidateQueries: vi.fn<() => Promise<void>>(),
    mockNavigate: vi.fn<(input: unknown) => Promise<void>>(),
    mockUseAgent: vi.fn(),
    mockUseAgentSessions: vi.fn(),
  })
);

let mockActiveWorkspaceId: string | null = "ws_alpha";

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}));

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock("@/systems/agent", () => ({
  agentKeys: { all: ["agents"] },
  useAgent: (name: string, workspace?: string | null) => mockUseAgent(name, workspace),
  useAgentSessions: (workspaceId: string | null, agentName: string) =>
    mockUseAgentSessions(workspaceId, agentName),
}));

vi.mock("@/systems/session", () => ({
  sessionKeys: { lists: () => ["sessions", "list"] },
  useSessionCreate: () => ({
    hasActiveWorkspace: true,
    isCreating: false,
    openForAgent: vi.fn(),
    pendingAgentName: null,
  }),
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    activeWorkspaceId: mockActiveWorkspaceId,
  }),
}));

import { useAgentDetailPage } from "../use-agent-detail-page";

describe("useAgentDetailPage", () => {
  beforeEach(() => {
    mockActiveWorkspaceId = "ws_alpha";
    mockInvalidateQueries.mockReset();
    mockNavigate.mockReset();
    mockUseAgent.mockReset();
    mockUseAgentSessions.mockReset();
    mockUseAgent.mockReturnValue({
      data: { name: "categorized-multi", provider: "codex", prompt: "prompt" },
      error: null,
      isLoading: false,
    });
    mockUseAgentSessions.mockReturnValue({
      sessions: [],
      total: 42,
      activeTotal: 3,
      resumableTotal: 8,
      lastActivityAt: "2026-07-11T12:00:00Z",
      hasMore: true,
      isLoadingMore: false,
      loadMore: vi.fn(),
      isError: false,
      isLoading: false,
    });
  });

  it("loads agent details in the active workspace context", () => {
    renderHook(() => useAgentDetailPage("categorized-multi"));

    expect(mockUseAgent).toHaveBeenCalledWith("categorized-multi", "ws_alpha");
    expect(mockUseAgentSessions).toHaveBeenCalledWith("ws_alpha", "categorized-multi");
  });

  it("falls back to global agent details when no workspace is active", () => {
    mockActiveWorkspaceId = null;

    renderHook(() => useAgentDetailPage("global-agent"));

    expect(mockUseAgent).toHaveBeenCalledWith("global-agent", null);
    expect(mockUseAgentSessions).toHaveBeenCalledWith(null, "global-agent");
  });

  it("exposes exact server-owned session metrics and pagination", () => {
    const loadMore = vi.fn();
    mockUseAgentSessions.mockReturnValue({
      sessions: [{ id: "sess-page-1" }],
      total: 205,
      activeTotal: 7,
      resumableTotal: 13,
      lastActivityAt: "2026-07-11T12:00:00Z",
      hasMore: true,
      isLoadingMore: false,
      loadMore,
      isError: false,
      isLoading: false,
    });

    const { result } = renderHook(() => useAgentDetailPage("categorized-multi"));

    expect(result.current.sessionsTotal).toBe(205);
    expect(result.current.activeSessionsTotal).toBe(7);
    expect(result.current.resumableSessionsTotal).toBe(13);
    expect(result.current.lastSessionActivityAt).toBe("2026-07-11T12:00:00Z");
    expect(result.current.hasMoreSessions).toBe(true);
    expect(result.current.onLoadMoreSessions).toBe(loadMore);
  });
});
