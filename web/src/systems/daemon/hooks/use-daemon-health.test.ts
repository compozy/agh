import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useDaemonHealth } from "./use-daemon-health";

vi.mock("../adapters/daemon-api", () => ({
  fetchHealth: vi.fn(),
}));

import { fetchHealth } from "../adapters/daemon-api";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("useDaemonHealth", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('derives "connected" when query succeeds', async () => {
    vi.mocked(fetchHealth).mockResolvedValue({
      status: "ok",
      uptime_seconds: 100,
      active_sessions: 1,
      active_agents: 1,
      global_db_size_bytes: 0,
      session_db_size_bytes: 0,
      version: "0.1.0",
    });

    const { result } = renderHook(() => useDaemonHealth(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.connectionStatus).toBe("connected");
    });

    expect(result.current.health).toBeDefined();
    expect(result.current.health?.status).toBe("ok");
  });

  it('derives "disconnected" when query errors', async () => {
    vi.mocked(fetchHealth).mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useDaemonHealth(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.connectionStatus).toBe("disconnected");
    });

    expect(result.current.health).toBeUndefined();
  });
});
