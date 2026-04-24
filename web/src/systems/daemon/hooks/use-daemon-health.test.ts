import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useDaemonHealth } from "./use-daemon-health";

vi.mock("../adapters/daemon-api", () => ({
  fetchHealth: vi.fn(),
}));

import { fetchHealth } from "../adapters/daemon-api";

const healthFixture = {
  status: "ok",
  uptime_seconds: 100,
  active_sessions: 1,
  active_agents: 1,
  bridges: {
    total_instances: 0,
    route_count: 0,
    delivery_backlog: 0,
    delivery_dropped_total: 0,
    delivery_failures_total: 0,
    auth_failures_total: 0,
    status_counts: {
      disabled: 0,
      starting: 0,
      ready: 0,
      degraded: 0,
      auth_required: 0,
      error: 0,
    },
  },
  global_db_size_bytes: 0,
  session_db_size_bytes: 0,
  persistence: {
    status: "ok",
    global_db_size_bytes: 0,
    session_db_size_bytes: 0,
  },
  retention: {
    enabled: false,
    retention_days: 0,
    sweep_interval_seconds: 86_400,
    last_sweep_status: "disabled",
    deleted_event_summaries: 0,
    deleted_token_stats: 0,
    deleted_permission_log_rows: 0,
  },
  version: "0.1.0",
} as const;

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
    vi.mocked(fetchHealth).mockResolvedValue(healthFixture);

    const { result } = renderHook(() => useDaemonHealth(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.connectionStatus).toBe("connected");
    });

    expect(result.current.health).toBeDefined();
    expect(result.current.health?.status).toBe("ok");
  });

  it('derives "reconnecting" while the first health query is still pending', () => {
    vi.mocked(fetchHealth).mockReturnValue(new Promise(() => undefined));

    const { result } = renderHook(() => useDaemonHealth(), {
      wrapper: createWrapper(),
    });

    expect(result.current.connectionStatus).toBe("reconnecting");
    expect(result.current.isInitialLoading).toBe(true);
  });

  it('derives "reconnecting" while the health query is retrying after an error', async () => {
    vi.mocked(fetchHealth).mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useDaemonHealth(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.connectionStatus).toBe("reconnecting");
    });

    expect(result.current.health).toBeUndefined();
  });
});
