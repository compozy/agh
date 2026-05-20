import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { AgentPayload } from "@/systems/agent";
import type { HealthPayload } from "@/systems/status";
import type { SessionPayload } from "@/systems/session";
import type { WorkspacePayload } from "@/systems/workspace";

vi.mock("@/systems/agent/adapters/agent-api", () => ({
  fetchAgents: vi.fn(),
  fetchAgent: vi.fn(),
}));

vi.mock("@/systems/status/adapters/daemon-api", () => ({
  fetchHealth: vi.fn(),
  fetchDaemonStatus: vi.fn(),
}));

vi.mock("@/systems/session/adapters/session-api", () => ({
  fetchSessions: vi.fn(),
  fetchSession: vi.fn(),
  approveSession: vi.fn(),
  createSession: vi.fn(),
  fetchSessionEvents: vi.fn(),
  fetchSessionHistory: vi.fn(),
  fetchSessionTranscript: vi.fn(),
  resumeSession: vi.fn(),
  stopSession: vi.fn(),
}));

vi.mock("@/systems/workspace/adapters/workspace-api", () => ({
  fetchWorkspaces: vi.fn(),
  fetchWorkspace: vi.fn(),
  resolveWorkspace: vi.fn(),
}));

let mockSelectedWorkspaceId: string | null = "ws_main";

vi.mock("@/systems/workspace/hooks/use-active-workspace-store", () => ({
  useActiveWorkspaceStore: (selector?: (state: unknown) => unknown) => {
    const state = {
      selectedWorkspaceId: mockSelectedWorkspaceId,
      setSelectedWorkspaceId: vi.fn(),
      clearSelectedWorkspaceId: vi.fn(),
    };
    return selector ? selector(state) : state;
  },
}));

import { fetchAgents } from "@/systems/agent/adapters/agent-api";
import { fetchHealth } from "@/systems/status/adapters/daemon-api";
import { fetchSessions } from "@/systems/session/adapters/session-api";
import { fetchWorkspace, fetchWorkspaces } from "@/systems/workspace/adapters/workspace-api";

import { formatUptimeSeconds, useHomePage } from "../use-home-page";

const HEALTH_FIXTURE: HealthPayload = {
  status: "ok",
  uptime_seconds: 3_600 + 900,
  active_sessions: 2,
  active_agents: 5,
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
  version: "0.1.0-test",
};

const WORKSPACES_FIXTURE: WorkspacePayload[] = [
  {
    id: "ws_main",
    root_dir: "/workspaces/main",
    add_dirs: [],
    name: "main",
    created_at: "2026-04-15T09:00:00Z",
    updated_at: "2026-04-15T09:00:00Z",
  },
  {
    id: "ws_secondary",
    root_dir: "/workspaces/sec",
    add_dirs: [],
    name: "secondary",
    created_at: "2026-04-15T09:00:00Z",
    updated_at: "2026-04-15T09:00:00Z",
  },
];

const AGENTS_FIXTURE: AgentPayload[] = [
  { name: "alpha", provider: "claude", prompt: "" },
  { name: "beta", provider: "codex", prompt: "" },
];

const WORKSPACE_AGENTS_FIXTURE: AgentPayload[] = [
  ...AGENTS_FIXTURE,
  { name: "gamma", provider: "codex", prompt: "" },
];

const SESSIONS_FIXTURE: SessionPayload[] = [
  {
    id: "sess_1",
    name: "Session 1",
    agent_name: "alpha",
    provider: "claude",
    workspace_id: "ws_main",
    workspace_path: "/workspaces/main",
    state: "active",
    created_at: "2026-04-15T09:00:00Z",
    updated_at: "2026-04-15T09:00:00Z",
  },
  {
    id: "sess_2",
    name: "Session 2",
    agent_name: "beta",
    provider: "codex",
    workspace_id: "ws_main",
    workspace_path: "/workspaces/main",
    state: "active",
    created_at: "2026-04-15T09:00:00Z",
    updated_at: "2026-04-15T09:00:00Z",
  },
  {
    id: "sess_3",
    name: "Session 3",
    agent_name: "alpha",
    provider: "claude",
    workspace_id: "ws_main",
    workspace_path: "/workspaces/main",
    state: "stopped",
    created_at: "2026-04-15T09:00:00Z",
    updated_at: "2026-04-15T09:00:00Z",
  },
];

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe("useHomePage", () => {
  beforeEach(() => {
    mockSelectedWorkspaceId = "ws_main";
    vi.clearAllMocks();
    vi.mocked(fetchHealth).mockResolvedValue(HEALTH_FIXTURE);
    vi.mocked(fetchWorkspaces).mockResolvedValue(WORKSPACES_FIXTURE);
    vi.mocked(fetchWorkspace).mockResolvedValue({
      workspace: WORKSPACES_FIXTURE[0],
      sessions: SESSIONS_FIXTURE,
      agents: WORKSPACE_AGENTS_FIXTURE,
      skills: [],
      providers: [],
    });
    vi.mocked(fetchAgents).mockResolvedValue(AGENTS_FIXTURE);
    vi.mocked(fetchSessions).mockResolvedValue(SESSIONS_FIXTURE);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("derives healthy daemon status, connected pill, and metrics for a happy path", async () => {
    const { result } = renderHook(() => useHomePage(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
      expect(result.current.daemonStatus.key).toBe("healthy");
    });

    expect(result.current.connectionStatus).toBe("connected");
    expect(result.current.daemonStatus.tone).toBe("success");
    expect(result.current.daemonStatus.label).toBe("Healthy");
    expect(result.current.daemonVersion).toBe("0.1.0-test");
    expect(result.current.hasFatalError).toBe(false);
    expect(result.current.hasWorkspaces).toBe(true);
    expect(result.current.activeWorkspaceName).toBe("main");

    const metricsByKey = Object.fromEntries(
      result.current.metrics.map(metric => [metric.key, metric] as const)
    );
    expect(metricsByKey["active-sessions"].value).toBe("2");
    expect(metricsByKey["active-sessions"].detail).toBe("in main");
    expect(metricsByKey.workspaces.value).toBe("2");
    expect(metricsByKey.agents.value).toBe("3");
    expect(metricsByKey.uptime.value).toBe("1h 15m");
  });

  it("maps a non-ok health status to a warning/degraded descriptor", async () => {
    vi.mocked(fetchHealth).mockResolvedValue({ ...HEALTH_FIXTURE, status: "degraded" });

    const { result } = renderHook(() => useHomePage(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.daemonStatus.key).toBe("degraded");
    });

    expect(result.current.daemonStatus.tone).toBe("warning");
    expect(result.current.daemonStatus.label).toBe("Degraded");
  });

  it("returns an error descriptor after the health query fails", async () => {
    vi.mocked(fetchHealth).mockRejectedValue(new Error("network down"));

    const { result } = renderHook(() => useHomePage(), { wrapper: createWrapper() });

    await waitFor(
      () => {
        expect(result.current.daemonStatus.key).toBe("disconnected");
      },
      { timeout: 2_500 }
    );

    expect(result.current.connectionStatus).toBe("error");
    expect(result.current.daemonStatus.tone).toBe("danger");
    expect(result.current.daemonVersion).toBeNull();
  });

  it("flags a fatal error when workspaces fail to load", async () => {
    vi.mocked(fetchWorkspaces).mockRejectedValue(new Error("workspaces unavailable"));

    const { result } = renderHook(() => useHomePage(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.hasFatalError).toBe(true);
    });

    expect(result.current.errorMessage).toBe("workspaces unavailable");
  });

  it("falls back to health.active_sessions when no workspace is selected", async () => {
    mockSelectedWorkspaceId = null;
    vi.mocked(fetchWorkspaces).mockResolvedValue([]);

    const { result } = renderHook(() => useHomePage(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    const sessionsMetric = result.current.metrics.find(metric => metric.key === "active-sessions");
    expect(sessionsMetric?.value).toBe(String(HEALTH_FIXTURE.active_sessions));
    expect(sessionsMetric?.detail).toBeUndefined();
    expect(result.current.hasWorkspaces).toBe(false);
  });

  it("marks the active-session metric unavailable when the workspace sessions query fails", async () => {
    vi.mocked(fetchSessions).mockRejectedValue(new Error("sessions unavailable"));

    const { result } = renderHook(() => useHomePage(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    const sessionsMetric = result.current.metrics.find(metric => metric.key === "active-sessions");
    expect(sessionsMetric?.value).toBe("—");
    expect(sessionsMetric?.detail).toBe("unavailable for main");
    expect(result.current.hasFatalError).toBe(false);
  });
});

describe("formatUptimeSeconds", () => {
  it("returns an em-dash for invalid input", () => {
    expect(formatUptimeSeconds(undefined)).toBe("—");
    expect(formatUptimeSeconds(null)).toBe("—");
    expect(formatUptimeSeconds(-12)).toBe("—");
    expect(formatUptimeSeconds(Number.NaN)).toBe("—");
  });

  it("formats sub-minute durations as seconds", () => {
    expect(formatUptimeSeconds(0)).toBe("0s");
    expect(formatUptimeSeconds(45)).toBe("45s");
  });

  it("formats minutes/hours/days with their largest two units", () => {
    expect(formatUptimeSeconds(60)).toBe("1m");
    expect(formatUptimeSeconds(125)).toBe("2m 5s");
    expect(formatUptimeSeconds(3_600)).toBe("1h");
    expect(formatUptimeSeconds(3_600 + 1_800)).toBe("1h 30m");
    expect(formatUptimeSeconds(86_400)).toBe("1d");
    expect(formatUptimeSeconds(86_400 + 3_600 * 5)).toBe("1d 5h");
  });
});
