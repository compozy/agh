import { useCallback, useState } from "react";

import { useSidebarStore } from "@/hooks/use-sidebar-store";
import { useAgents } from "@/systems/agent";
import { useDaemonHealth } from "@/systems/daemon";
import { useCreateSession, useSessions } from "@/systems/session";
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
  const createSession = useCreateSession();

  const handleNewSession = useCallback(
    (agentName: string) => {
      if (!activeWorkspaceId) {
        return;
      }

      createSession.mutate({ agent_name: agentName, workspace: activeWorkspaceId });
    },
    [activeWorkspaceId, createSession]
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
    isCreatingSession: createSession.isPending,
    openWorkspaceSetup,
  };
}

export { useAppLayout };
