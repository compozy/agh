import { Outlet, createFileRoute } from "@tanstack/react-router";

import { AppSidebar } from "@/components/app-sidebar";
import { useAppLayout } from "@/hooks/routes/use-app-layout";
import { WorkspaceOnboarding, WorkspaceSetupDialog } from "@/systems/workspace";

export const Route = createFileRoute("/_app")({
  component: AppLayout,
});

function AppLayout() {
  const page = useAppLayout();

  if (!page.areWorkspacesLoading && !page.workspacesError && !page.hasWorkspaces) {
    return <WorkspaceOnboarding onWorkspaceResolved={page.setActiveWorkspaceId} />;
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <AppSidebar
        collapsed={page.collapsed}
        onCollapseChange={page.setCollapsed}
        workspaces={page.areWorkspacesLoading || page.workspacesError ? undefined : page.workspaces}
        activeWorkspace={page.activeWorkspace}
        activeWorkspaceId={page.activeWorkspaceId}
        onSelectWorkspace={page.setActiveWorkspaceId}
        onAddWorkspace={page.openWorkspaceSetup}
        health={page.health}
        connectionStatus={page.connectionStatus}
        agents={page.agents}
        agentsLoading={page.agentsLoading}
        agentsError={page.agentsError}
        sessions={page.sessions}
        onNewSession={page.handleNewSession}
        isCreatingSession={page.isCreatingSession}
      />
      <div className="flex flex-1 flex-col overflow-hidden">
        <div className="relative flex flex-1 flex-col overflow-hidden bg-background">
          <Outlet />
        </div>
      </div>
      <WorkspaceSetupDialog
        open={page.isWorkspaceSetupOpen}
        onOpenChange={page.setWorkspaceSetupOpen}
        onWorkspaceResolved={page.setActiveWorkspaceId}
      />
    </div>
  );
}
