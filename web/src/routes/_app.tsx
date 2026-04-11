import { useState } from "react";
import { Outlet, createFileRoute } from "@tanstack/react-router";

import { AppHeader } from "@/components/app-header";
import { AppSidebar } from "@/components/app-sidebar";
import { useSidebarStore } from "@/stores/sidebar-store";
import { useAgents } from "@/systems/agent";
import { useDaemonHealth } from "@/systems/daemon";
import { useCreateSession, useSessions } from "@/systems/session";
import { useActiveWorkspace, WorkspaceOnboarding, WorkspaceSetupDialog } from "@/systems/workspace";

export const Route = createFileRoute("/_app")({
  component: AppLayout,
});

function AppLayout() {
  const collapsed = useSidebarStore(state => state.collapsed);
  const toggleCollapsed = useSidebarStore(state => state.toggle);
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

  const handleNewSession = (agentName: string) => {
    if (!activeWorkspaceId) return;
    createSession.mutate({ agent_name: agentName, workspace: activeWorkspaceId });
  };

  if (!areWorkspacesLoading && !workspacesError && !hasWorkspaces) {
    return <WorkspaceOnboarding onWorkspaceResolved={setActiveWorkspaceId} />;
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <AppSidebar
        collapsed={collapsed}
        onToggleCollapsed={toggleCollapsed}
        workspaces={areWorkspacesLoading || workspacesError ? undefined : workspaces}
        activeWorkspace={activeWorkspace}
        activeWorkspaceId={activeWorkspaceId}
        onSelectWorkspace={setActiveWorkspaceId}
        onAddWorkspace={() => setWorkspaceSetupOpen(true)}
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
      <WorkspaceSetupDialog
        open={isWorkspaceSetupOpen}
        onOpenChange={setWorkspaceSetupOpen}
        onWorkspaceResolved={setActiveWorkspaceId}
      />
    </div>
  );
}
