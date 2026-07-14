import { useState } from "react";

import { useSidebarStore } from "@/hooks/use-sidebar-store";
import { useAgentCatalog, useAgentCreateDialog, useAgents } from "@/systems/agent";
import { useDaemonHealth } from "@/systems/status";
import {
  useSessionCatalogStreams,
  useSessionCreateDialog,
  useSessions,
  useWorkspaceSessionActivity,
} from "@/systems/session";
import { useActiveWorkspace, useWorkspace } from "@/systems/workspace";
import { useSyncUserHomeDir } from "@/systems/workspace/hooks/use-sync-user-home-dir";

function useAppLayout() {
  useSyncUserHomeDir();
  const collapsed = useSidebarStore(state => state.collapsed);
  const setCollapsed = useSidebarStore(state => state.setCollapsed);
  const { connectionStatus } = useDaemonHealth();
  const {
    workspaces,
    hasWorkspaces,
    activeWorkspace,
    activeWorkspaceId,
    setActiveWorkspaceId,
    isLoading: areWorkspacesLoading,
    isError: workspacesError,
  } = useActiveWorkspace();
  const {
    data: agents,
    isLoading: agentsLoading,
    isError: agentsError,
  } = useAgents(activeWorkspaceId, {
    enabled: activeWorkspaceId !== null,
  });
  const activeWorkspaceDetail = useWorkspace(activeWorkspaceId ?? "", {
    enabled: activeWorkspaceId !== null,
  });
  const workspaceAgents = activeWorkspaceId === null ? undefined : agents;
  const [isWorkspaceSetupOpen, setWorkspaceSetupOpen] = useState(false);
  const agentCatalogSummary = useAgentCatalog(
    activeWorkspaceId ?? "",
    { limit: 1 },
    { enabled: activeWorkspaceId !== null }
  );
  const activeSessions = useSessions(activeWorkspaceId, {
    enabled: activeWorkspaceId !== null,
    filters: { state: "active", limit: 1 },
  });
  const workspaceSessionActivity = useWorkspaceSessionActivity(workspaces, hasWorkspaces);
  useSessionCatalogStreams(workspaces, { enabled: hasWorkspaces });
  const sessionCreate = useSessionCreateDialog({
    agents: workspaceAgents,
    activeWorkspace,
  });
  const agentCreate = useAgentCreateDialog({
    activeWorkspace,
    workspaceProviders: activeWorkspaceDetail.data?.providers ?? [],
    workspaceProvidersLoading: activeWorkspaceId !== null && activeWorkspaceDetail.isLoading,
    workspaceProvidersError: activeWorkspaceDetail.error
      ? describeWorkspaceProviderError(activeWorkspaceDetail.error)
      : null,
  });

  const handleNewSession = (agentName: string) => {
    sessionCreate.openForAgent(agentName);
  };

  const openWorkspaceSetup = () => {
    setWorkspaceSetupOpen(true);
  };

  return {
    collapsed,
    setCollapsed,
    connectionStatus,
    workspaces,
    hasWorkspaces,
    activeWorkspace,
    activeWorkspaceId,
    setActiveWorkspaceId,
    areWorkspacesLoading,
    workspacesError,
    agents: workspaceAgents,
    agentsLoading: activeWorkspaceId !== null && agentsLoading,
    agentsError: activeWorkspaceId !== null && agentsError,
    isWorkspaceSetupOpen,
    setWorkspaceSetupOpen,
    agentsCount:
      agentCatalogSummary.sessionsAvailable && agentCatalogSummary.facets
        ? {
            live: agentCatalogSummary.facets.active,
            total: agentCatalogSummary.facets.total,
          }
        : undefined,
    activeSessionCount: activeSessions.total,
    workspaceSessionActivity,
    handleNewSession,
    isCreatingSession: sessionCreate.isSubmitting,
    pendingSessionAgentName: sessionCreate.pendingAgentName,
    sessionCreate,
    agentCreate,
    openWorkspaceSetup,
  };
}

function describeWorkspaceProviderError(error: unknown): string {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  return "Unable to load workspace providers.";
}

export { useAppLayout };
