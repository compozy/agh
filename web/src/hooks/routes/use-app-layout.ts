import { useCallback, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { useSidebarStore } from "@/hooks/use-sidebar-store";
import { useAgents } from "@/systems/agent";
import { useDaemonHealth } from "@/systems/daemon";
import { useCreateSession, useSessions } from "@/systems/session";
import { useActiveWorkspace } from "@/systems/workspace";

interface PendingSessionState {
  agentName: string;
  workspaceId: string;
}

function useAppLayout() {
  const navigate = useNavigate();
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
  const [pendingSession, setPendingSession] = useState<PendingSessionState | null>(null);
  const { data: sessions } = useSessions(activeWorkspaceId, {
    enabled: activeWorkspaceId !== null,
  });
  const createSession = useCreateSession();

  const handleNewSession = useCallback(
    async (agentName: string) => {
      const workspaceId = activeWorkspaceId;
      if (!workspaceId || pendingSession) {
        return;
      }

      setPendingSession({ agentName, workspaceId });

      try {
        const session = await createSession.mutateAsync({
          agent_name: agentName,
          workspace: workspaceId,
        });
        await navigate({ to: "/session/$id", params: { id: session.id } });
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to create session");
      } finally {
        setPendingSession(null);
      }
    },
    [activeWorkspaceId, createSession, navigate, pendingSession]
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
    isCreatingSession: pendingSession !== null,
    pendingSessionAgentName: pendingSession?.agentName ?? null,
    pendingSessionWorkspaceId: pendingSession?.workspaceId ?? null,
    openWorkspaceSetup,
  };
}

export { useAppLayout };
