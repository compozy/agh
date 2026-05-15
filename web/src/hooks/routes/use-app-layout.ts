import { useCallback, useState } from "react";

import { useSidebarStore } from "@/hooks/use-sidebar-store";
import { useAgentCreateDialog, useAgents } from "@/systems/agent";
import { useDaemonHealth } from "@/systems/daemon";
import { useSessionCreateDialog, useSessions } from "@/systems/session";
import { useActiveWorkspace, useWorkspace } from "@/systems/workspace";

function useAppLayout() {
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
  const { data: agents, isLoading: agentsLoading, isError: agentsError } = useAgents();
  const activeWorkspaceDetail = useWorkspace(activeWorkspaceId ?? "", {
    enabled: activeWorkspaceId !== null,
  });
  const hasWorkspaceScopedAgents =
    activeWorkspaceId !== null && activeWorkspaceDetail.data?.agents !== undefined;
  const workspaceAgents = hasWorkspaceScopedAgents ? activeWorkspaceDetail.data?.agents : agents;
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
    agentsLoading: hasWorkspaceScopedAgents
      ? false
      : agentsLoading || (activeWorkspaceId !== null && activeWorkspaceDetail.isLoading),
    agentsError: hasWorkspaceScopedAgents
      ? false
      : agentsError ||
        (activeWorkspaceId !== null &&
          activeWorkspaceDetail.isError &&
          workspaceAgents === undefined),
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
