import { useMemo } from "react";

import type { ConnectionStatus, PillTone } from "@agh/ui";

import { useAgents } from "@/systems/agent";
import { useDaemonHealth } from "@/systems/status";
import type { HealthPayload } from "@/systems/status";
import { useSessions, type SessionPayload } from "@/systems/session";
import { useActiveWorkspace, useWorkspace } from "@/systems/workspace";

export type DaemonStatusKey = "healthy" | "degraded" | "disconnected" | "unknown";

interface DaemonStatusDescriptor {
  key: DaemonStatusKey;
  tone: PillTone;
  label: string;
  description: string;
}

interface HomeMetricEntry {
  key: "active-sessions" | "workspaces" | "agents" | "uptime";
  label: string;
  value: string;
  detail?: string;
}

interface HomePageView {
  isLoading: boolean;
  hasFatalError: boolean;
  errorMessage: string | null;
  connectionStatus: ConnectionStatus;
  daemonStatus: DaemonStatusDescriptor;
  daemonVersion: string | null;
  metrics: HomeMetricEntry[];
  hasWorkspaces: boolean;
  activeWorkspaceName: string | null;
}

const SECOND = 1;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;
const ACTIVE_SESSION_STATES = new Set<SessionPayload["state"]>(["active", "starting", "stopping"]);

function isActiveSession(session: SessionPayload): boolean {
  return ACTIVE_SESSION_STATES.has(session.state);
}

function formatUptimeSeconds(seconds: number | null | undefined): string {
  if (typeof seconds !== "number" || !Number.isFinite(seconds) || seconds < 0) {
    return "—";
  }

  if (seconds < MINUTE) {
    return `${Math.round(seconds)}s`;
  }

  if (seconds < HOUR) {
    const minutes = Math.floor(seconds / MINUTE);
    const remainder = Math.floor(seconds % MINUTE);
    return remainder === 0 ? `${minutes}m` : `${minutes}m ${remainder}s`;
  }

  if (seconds < DAY) {
    const hours = Math.floor(seconds / HOUR);
    const remainder = Math.floor((seconds % HOUR) / MINUTE);
    return remainder === 0 ? `${hours}h` : `${hours}h ${remainder}m`;
  }

  const days = Math.floor(seconds / DAY);
  const remainder = Math.floor((seconds % DAY) / HOUR);
  return remainder === 0 ? `${days}d` : `${days}d ${remainder}h`;
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

function useHomePage(): HomePageView {
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
  const { data: agents, isLoading: agentsLoading, error: agentsError } = useAgents();
  const {
    data: workspaceDetail,
    isLoading: isWorkspaceDetailLoading,
    error: workspaceDetailError,
  } = useWorkspace(activeWorkspaceId ?? "", {
    enabled: activeWorkspaceId !== null,
  });
  const {
    data: sessions,
    isLoading: areSessionsLoading,
    isError: sessionsError,
  } = useSessions(activeWorkspaceId, {
    enabled: activeWorkspaceId !== null,
  });

  const daemonStatus = useMemo(
    () => deriveDaemonStatus(connectionStatus, health),
    [connectionStatus, health]
  );

  const activeSessionsMetric = useMemo<HomeMetricEntry>(() => {
    if (activeWorkspaceId === null) {
      return {
        key: "active-sessions",
        label: "Active Sessions",
        value: String(health?.active_sessions ?? 0),
      };
    }

    if (sessionsError) {
      return {
        key: "active-sessions",
        label: "Active Sessions",
        value: "—",
        detail: activeWorkspace ? `unavailable for ${activeWorkspace.name}` : "unavailable",
      };
    }

    return {
      key: "active-sessions",
      label: "Active Sessions",
      value: String(sessions?.filter(isActiveSession).length ?? 0),
      detail: activeWorkspace ? `in ${activeWorkspace.name}` : undefined,
    };
  }, [activeWorkspace, activeWorkspaceId, health?.active_sessions, sessions, sessionsError]);

  const activeWorkspaceAgents = workspaceDetail?.agents ?? agents;
  const agentsCount = activeWorkspaceAgents?.length ?? 0;
  const workspacesCount = workspaces.length;
  const uptimeLabel = formatUptimeSeconds(health?.uptime_seconds);

  const metrics = useMemo<HomeMetricEntry[]>(
    () => [
      activeSessionsMetric,
      {
        key: "workspaces",
        label: "Workspaces",
        value: String(workspacesCount),
      },
      {
        key: "agents",
        label: "Agents",
        value: String(agentsCount),
      },
      {
        key: "uptime",
        label: "Daemon Uptime",
        value: uptimeLabel,
      },
    ],
    [activeSessionsMetric, workspacesCount, agentsCount, uptimeLabel]
  );

  const isLoading =
    isHealthInitialLoading ||
    areWorkspacesLoading ||
    agentsLoading ||
    (activeWorkspaceId !== null && isWorkspaceDetailLoading) ||
    (activeWorkspaceId !== null && areSessionsLoading);

  const fatalError = workspacesError
    ? workspacesErrorObject
    : workspaceDetailError
      ? workspaceDetailError
      : agentsError
        ? agentsError
        : null;
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
  formatUptimeSeconds,
  useHomePage,
  type DaemonStatusDescriptor,
  type HomeMetricEntry,
  type HomePageView,
};
