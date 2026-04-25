import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { expectFetchRequest, mockJsonResponse } from "@/test/fetch-test-utils";

import { fetchDaemonStatus, fetchHealth } from "./daemon-api";

describe("fetchHealth", () => {
  const validResponse = {
    health: {
      status: "ok",
      uptime_seconds: 3600,
      active_sessions: 2,
      active_agents: 3,
      bridges: {
        total_instances: 1,
        route_count: 2,
        delivery_backlog: 0,
        delivery_dropped_total: 0,
        delivery_failures_total: 0,
        auth_failures_total: 0,
        status_counts: {
          disabled: 0,
          starting: 0,
          ready: 1,
          degraded: 0,
          auth_required: 0,
          error: 0,
        },
      },
      global_db_size_bytes: 1048576,
      session_db_size_bytes: 524288,
      persistence: {
        status: "ok",
        global_db_size_bytes: 1048576,
        session_db_size_bytes: 524288,
      },
      retention: {
        enabled: true,
        retention_days: 7,
        sweep_interval_seconds: 86400,
        last_sweep_status: "ok",
        last_sweep_at: "2026-04-17T18:00:00Z",
        last_cutoff_at: "2026-04-10T18:00:00Z",
        deleted_event_summaries: 0,
        deleted_token_stats: 0,
        deleted_permission_log_rows: 0,
      },
      failures: {
        status: "degraded",
        total: 1,
        by_kind: {
          process_exit: 1,
        },
        recent: [
          {
            session_id: "sess-crash",
            agent_name: "coder",
            provider: "claude",
            workspace_id: "ws-main",
            state: "stopped",
            failure_kind: "process_exit",
            summary: "provider exited with status 1",
            crash_bundle_path: "/Users/pedro/.agh/logs/crash-bundles/sess-crash.json",
            updated_at: "2026-04-17T18:00:00Z",
          },
        ],
      },
      agent_probes: [
        {
          agent_name: "coder",
          provider: "claude",
          command: "claude --acp",
          executable: "/usr/local/bin/claude",
          status: "ok",
          checked_at: "2026-04-17T18:00:00Z",
          duration_ms: 4,
        },
      ],
      version: "0.1.0",
    },
  };

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns parsed HealthPayload on success", async () => {
    mockJsonResponse(validResponse);

    const result = await fetchHealth();

    expect(result).toEqual(validResponse.health);
    await expectFetchRequest({ path: "/api/observe/health" });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse(validResponse);

    const controller = new AbortController();
    await fetchHealth(controller.signal);

    await expectFetchRequest({
      path: "/api/observe/health",
      signal: controller.signal,
    });
  });

  it("throws on network error (non-ok response)", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(fetchHealth()).rejects.toThrow("Daemon health check failed: 500");
  });

  it("throws on fetch rejection", async () => {
    vi.mocked(globalThis.fetch).mockRejectedValue(new TypeError("Failed to fetch"));

    await expect(fetchHealth()).rejects.toThrow("Failed to fetch");
  });
});

describe("fetchDaemonStatus", () => {
  const validResponse = {
    daemon: {
      status: "running",
      pid: 4242,
      started_at: "2026-04-10T12:00:00Z",
      socket: "/tmp/agh.sock",
      http_host: "127.0.0.1",
      http_port: 2123,
      user_home_dir: "/Users/pedro",
      active_sessions: 2,
      total_sessions: 7,
      version: "0.1.0",
    },
  };

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns parsed DaemonStatusPayload on success", async () => {
    mockJsonResponse(validResponse);

    const result = await fetchDaemonStatus();

    expect(result).toEqual(validResponse.daemon);
    expect(result.user_home_dir).toBe("/Users/pedro");
    await expectFetchRequest({ path: "/api/daemon/status" });
  });

  it("passes abort signal to fetch", async () => {
    mockJsonResponse(validResponse);

    const controller = new AbortController();
    await fetchDaemonStatus(controller.signal);

    await expectFetchRequest({
      path: "/api/daemon/status",
      signal: controller.signal,
    });
  });

  it("throws on network error (non-ok response)", async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue(new Response(null, { status: 500 }));

    await expect(fetchDaemonStatus()).rejects.toThrow("Daemon status check failed: 500");
  });

  it("throws on fetch rejection", async () => {
    vi.mocked(globalThis.fetch).mockRejectedValue(new TypeError("Failed to fetch"));

    await expect(fetchDaemonStatus()).rejects.toThrow("Failed to fetch");
  });
});
