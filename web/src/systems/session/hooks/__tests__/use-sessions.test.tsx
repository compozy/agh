import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useSession, useSessions } from "../use-sessions";

vi.mock("../../adapters/session-api", () => ({
  fetchSession: vi.fn(),
  fetchSessions: vi.fn(),
  fetchSessionEvents: vi.fn(),
  fetchSessionHistory: vi.fn(),
  fetchSessionTranscript: vi.fn(),
}));

import { fetchSession, fetchSessions } from "../../adapters/session-api";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("useSessions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads sessions for the selected workspace filter", async () => {
    vi.mocked(fetchSessions).mockResolvedValue([
      {
        id: "sess-001",
        agent_name: "claude-agent",
        provider: "claude",
        workspace_id: "ws_alpha",
        workspace_path: "/workspace/alpha",
        state: "active",
        created_at: "2026-04-06T10:00:00Z",
        updated_at: "2026-04-06T10:00:00Z",
      },
    ]);

    const { result } = renderHook(() => useSessions("ws_alpha"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toHaveLength(1);
    });

    expect(result.current.data?.[0]?.provider).toBe("claude");
    expect(fetchSessions).toHaveBeenCalledWith("ws_alpha", expect.any(AbortSignal));
  });

  it("disables the query when the workspace filter is unavailable", async () => {
    renderHook(() => useSessions(null, { enabled: false }), {
      wrapper: createWrapper(),
    });

    expect(fetchSessions).not.toHaveBeenCalled();
  });
});

describe("useSession", () => {
  it("loads a single session detail", async () => {
    vi.mocked(fetchSession).mockResolvedValue({
      id: "sess-001",
      agent_name: "claude-agent",
      provider: "claude",
      workspace_id: "ws_alpha",
      workspace_path: "/workspace/alpha",
      state: "active",
      created_at: "2026-04-06T10:00:00Z",
      updated_at: "2026-04-06T10:00:00Z",
    });

    const { result } = renderHook(() => useSession("sess-001", "ws_alpha"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data?.id).toBe("sess-001");
    });

    expect(result.current.data?.provider).toBe("claude");
    expect(fetchSession).toHaveBeenCalledWith("ws_alpha", "sess-001", expect.any(AbortSignal));
  });
});
