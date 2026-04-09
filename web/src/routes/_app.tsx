import { useMemo, useState } from "react";
import { Outlet, createFileRoute } from "@tanstack/react-router";

import { AppHeader } from "@/components/app-header";
import { AppSidebar } from "@/components/app-sidebar";
import { useSidebarStore } from "@/stores/sidebar-store";
import { useAgents } from "@/systems/agent";
import { useDaemonHealth } from "@/systems/daemon";
import { useCreateSession, useSessions } from "@/systems/session";
import { useWorkspaces } from "@/systems/workspace";

export const Route = createFileRoute("/_app")({
  component: AppLayout,
});

function AppLayout() {
  const collapsed = useSidebarStore(state => state.collapsed);
  const toggleCollapsed = useSidebarStore(state => state.toggle);
  const { health, connectionStatus } = useDaemonHealth();
  const {
    data: workspaces,
    isLoading: areWorkspacesLoading,
    isError: workspacesError,
  } = useWorkspaces();
  const { data: agents, isLoading: agentsLoading, isError: agentsError } = useAgents();
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<string | null>(null);

  const activeWorkspaceId = useMemo(() => {
    if (!workspaces || workspaces.length === 0) return null;
    if (selectedWorkspaceId && workspaces.some(workspace => workspace.id === selectedWorkspaceId)) {
      return selectedWorkspaceId;
    }
    return workspaces[0].id;
  }, [selectedWorkspaceId, workspaces]);

  const activeWorkspace = useMemo(() => {
    if (!workspaces || !activeWorkspaceId) return undefined;
    return workspaces.find(workspace => workspace.id === activeWorkspaceId);
  }, [workspaces, activeWorkspaceId]);

  const { data: sessions } = useSessions(activeWorkspaceId, {
    enabled: activeWorkspaceId !== null,
  });
  const createSession = useCreateSession();

  const handleNewSession = (agentName: string) => {
    if (!activeWorkspaceId) return;
    createSession.mutate({ agent_name: agentName, workspace: activeWorkspaceId });
  };

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <AppSidebar
        collapsed={collapsed}
        onToggleCollapsed={toggleCollapsed}
        workspaces={areWorkspacesLoading || workspacesError ? undefined : (workspaces ?? undefined)}
        activeWorkspace={activeWorkspace}
        activeWorkspaceId={activeWorkspaceId}
        onSelectWorkspace={setSelectedWorkspaceId}
        health={health}
        connectionStatus={connectionStatus}
        agents={agents}
        agentsLoading={agentsLoading}
        agentsError={agentsError}
        sessions={sessions}
        onNewSession={handleNewSession}
        isCreatingSession={createSession.isPending}
      />
      <div className="flex flex-1 flex-col overflow-hidden">
        <AppHeader />
        <div className="relative flex flex-1 flex-col overflow-hidden bg-background">
          <Outlet />
        </div>
      </div>
    </div>
  );
}
