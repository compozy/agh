import { useCallback, useState } from "react";

import { useSidebarStore } from "@/hooks/use-sidebar-store";
import { useAgentCreateDialog, useAgents } from "@/systems/agent";
import { useDaemonHealth } from "@/systems/status";
import { useSessionCreateDialog, useSessions } from "@/systems/session";
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
  const { data: sessions } = useSessions(activeWorkspaceId, {
    enabled: activeWorkspaceId !== null,
  });
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

  const handleNewSession = useCallback(
    (agentName: string) => {
      sessionCreate.openForAgent(agentName);
    },
    [sessionCreate]
  );

  const openWorkspaceSetup = useCallback(() => {
    setWorkspaceSetupOpen(true);
  }, []);

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
    sessions,
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
