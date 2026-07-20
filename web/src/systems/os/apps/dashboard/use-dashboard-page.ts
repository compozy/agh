import type { ConnectionStatus, PillTone } from "@agh/ui";

import { formatUptimeSeconds } from "@/lib/format-time";
import { useAgents } from "@/systems/agent";
import { useDaemonHealth } from "@/systems/status";
import type { HealthPayload } from "@/systems/status";
import { useSessions } from "@/systems/session";
import { useActiveWorkspace } from "@/systems/workspace";

export type DaemonStatusKey = "healthy" | "degraded" | "disconnected" | "unknown";

interface DaemonStatusDescriptor {
  key: DaemonStatusKey;
  tone: PillTone;
  label: string;
  description: string;
}

interface DashboardMetricEntry {
  key: "active-sessions" | "workspaces" | "agents" | "uptime";
  label: string;
  value: string;
  detail?: string;
}

interface DashboardPageView {
  isLoading: boolean;
  hasFatalError: boolean;
  errorMessage: string | null;
  connectionStatus: ConnectionStatus;
  daemonStatus: DaemonStatusDescriptor;
  daemonVersion: string | null;
  metrics: DashboardMetricEntry[];
  hasWorkspaces: boolean;
  activeWorkspaceName: string | null;
}

function deriveDaemonStatus(
  connectionStatus: ConnectionStatus,
  health: HealthPayload | undefined
): DaemonStatusDescriptor {
  if (connectionStatus === "disconnected") {
    return {
      key: "disconnected",
      tone: "danger",
      label: "Disconnected",
      description:
        "The daemon is unreachable. Start it with `agh daemon` and the dashboard will reconnect automatically.",
    };
  }

  if (connectionStatus === "connecting") {
    return {
      key: "unknown",
      tone: "neutral",
      label: "Connecting",
      description: "Re-establishing the connection to the local daemon.",
    };
  }

  if (connectionStatus === "error") {
    return {
      key: "disconnected",
      tone: "danger",
      label: "Connection error",
      description: "The daemon health endpoint did not return a usable response.",
    };
  }

  if (!health) {
    return {
      key: "unknown",
      tone: "neutral",
      label: "Unknown",
      description: "Waiting for the first health response from the daemon.",
    };
  }

  const status = health.status?.toLowerCase();
  if (status === "ok" || status === "healthy" || status === "running") {
    return {
      key: "healthy",
      tone: "success",
      label: "Healthy",
      description: "All subsystems are reporting healthy status.",
    };
  }

  return {
    key: "degraded",
    tone: "warning",
    label: "Degraded",
    description: "The daemon responded but reported a non-healthy status.",
  };
}

function useDashboardPage(): DashboardPageView {
  const { health, connectionStatus, isInitialLoading: isHealthInitialLoading } = useDaemonHealth();
  const {
    workspaces,
    hasWorkspaces,
    activeWorkspace,
    activeWorkspaceId,
    isLoading: areWorkspacesLoading,
    isError: workspacesError,
    error: workspacesErrorObject,
  } = useActiveWorkspace();
  const {
    data: agents,
    isLoading: agentsLoading,
    error: agentsError,
  } = useAgents(activeWorkspaceId, {
    enabled: activeWorkspaceId !== null,
  });
  const {
    total: activeSessionsTotal,
    isLoading: areSessionsLoading,
    isError: sessionsError,
  } = useSessions(activeWorkspaceId, {
    enabled: activeWorkspaceId !== null,
    filters: { state: "active", type: "user", limit: 1 },
  });

  const daemonStatus = deriveDaemonStatus(connectionStatus, health);

  let activeSessionsMetric: DashboardMetricEntry;
  if (activeWorkspaceId === null) {
    activeSessionsMetric = {
      key: "active-sessions",
      label: "Active Sessions",
      value: String(health?.active_sessions ?? 0),
    };
  } else if (sessionsError) {
    activeSessionsMetric = {
      key: "active-sessions",
      label: "Active Sessions",
      value: "—",
      detail: activeWorkspace ? `unavailable for ${activeWorkspace.name}` : "unavailable",
    };
  } else {
    activeSessionsMetric = {
      key: "active-sessions",
      label: "Active Sessions",
      value: String(activeSessionsTotal),
      detail: activeWorkspace ? `in ${activeWorkspace.name}` : undefined,
    };
  }

  const agentsCount = agents?.length ?? 0;
  const workspacesCount = workspaces.length;
  const uptimeLabel = formatUptimeSeconds(health?.uptime_seconds);

  const metrics: DashboardMetricEntry[] = [
    activeSessionsMetric,
    { key: "workspaces", label: "Workspaces", value: String(workspacesCount) },
    { key: "agents", label: "Agents", value: String(agentsCount) },
    { key: "uptime", label: "Daemon Uptime", value: uptimeLabel },
  ];

  const isLoading =
    isHealthInitialLoading ||
    areWorkspacesLoading ||
    (activeWorkspaceId !== null && agentsLoading) ||
    (activeWorkspaceId !== null && areSessionsLoading);

  const fatalError = workspacesError ? workspacesErrorObject : agentsError ? agentsError : null;
  const errorMessage = fatalError instanceof Error ? fatalError.message : null;

  return {
    isLoading,
    hasFatalError: Boolean(fatalError),
    errorMessage,
    connectionStatus,
    daemonStatus,
    daemonVersion: health?.version ?? null,
    metrics,
    hasWorkspaces,
    activeWorkspaceName: activeWorkspace?.name ?? null,
  };
}

export {
  useDashboardPage,
  type DaemonStatusDescriptor,
  type DashboardMetricEntry,
  type DashboardPageView,
};
