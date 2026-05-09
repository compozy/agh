import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  deriveDaemonConnectionStatus,
  useDaemonConnectionStatus,
} from "../use-daemon-connection-status";

vi.mock("../../adapters/daemon-api", () => ({
  fetchHealth: vi.fn(),
}));

import { fetchHealth } from "../../adapters/daemon-api";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("deriveDaemonConnectionStatus", () => {
  it("Should map query states to a known connection status", () => {
    expect(
      deriveDaemonConnectionStatus({
        isPending: true,
        isFetching: true,
        isSuccess: false,
        isError: false,
      })
    ).toBe("connecting");
    expect(
      deriveDaemonConnectionStatus({
        data: { status: "ok" },
        isPending: false,
        isFetching: false,
        isSuccess: true,
        isError: false,
      })
    ).toBe("connected");
    expect(
      deriveDaemonConnectionStatus({
        isPending: false,
        isFetching: false,
        isSuccess: false,
        isError: true,
      })
    ).toBe("error");
    expect(
      deriveDaemonConnectionStatus({
        isPending: false,
        isFetching: false,
        isSuccess: false,
        isError: false,
      })
    ).toBe("disconnected");
  });
});

describe("useDaemonConnectionStatus", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("Should return connected after health resolves", async () => {
    vi.mocked(fetchHealth).mockResolvedValue({
      status: "ok",
      uptime_seconds: 12,
      active_sessions: 0,
      active_agents: 0,
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
      failures: {
        status: "ok",
        total: 0,
      },
      agent_probes: [],
      version: "0.1.0",
    });

    const { result } = renderHook(() => useDaemonConnectionStatus(), {
      wrapper: createWrapper(),
    });

    expect(result.current).toBe("connecting");
    await waitFor(() => {
      expect(result.current).toBe("connected");
    });
  });
});
