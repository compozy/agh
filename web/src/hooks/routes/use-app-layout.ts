import { useCallback, useState } from "react";

import { useSidebarStore } from "@/hooks/use-sidebar-store";
import { useAgents } from "@/systems/agent";
import { useDaemonHealth } from "@/systems/daemon";
import { useSessionCreateDialog, useSessions } from "@/systems/session";
import { useActiveWorkspace } from "@/systems/workspace";

function useAppLayout() {
  const collapsed = useSidebarStore(state => state.collapsed);
  const setCollapsed = useSidebarStore(state => state.setCollapsed);
  const { health, connectionStatus } = useDaemonHealth();
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
  const [isWorkspaceSetupOpen, setWorkspaceSetupOpen] = useState(false);
  const { data: sessions } = useSessions(activeWorkspaceId, {
    enabled: activeWorkspaceId !== null,
  });
  const sessionCreate = useSessionCreateDialog({
    agents,
    activeWorkspace,
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
    health,
    connectionStatus,
    workspaces,
    hasWorkspaces,
    activeWorkspace,
    activeWorkspaceId,
    setActiveWorkspaceId,
    areWorkspacesLoading,
    workspacesError,
    agents,
    agentsLoading,
    agentsError,
    isWorkspaceSetupOpen,
    setWorkspaceSetupOpen,
    sessions,
    handleNewSession,
    isCreatingSession: sessionCreate.isSubmitting,
    pendingSessionAgentName: sessionCreate.pendingAgentName,
    pendingSessionWorkspaceId: sessionCreate.pendingWorkspaceId,
    sessionCreate,
    openWorkspaceSetup,
  };
}

export { useAppLayout };
