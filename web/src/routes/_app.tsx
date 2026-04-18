import { Outlet, createFileRoute, useLocation } from "@tanstack/react-router";
import { AnimatePresence, motion, useReducedMotionConfig } from "motion/react";

import { AppSidebar } from "@/components/app-sidebar";
import { useAppLayout } from "@/hooks/routes/use-app-layout";
import { WorkspaceOnboarding, WorkspaceSetupDialog } from "@/systems/workspace";

const ROUTE_FADE_DURATION = 0.2;

function resolveRouteTransitionDuration(reducedMotion: boolean): number {
  return reducedMotion ? 0 : ROUTE_FADE_DURATION;
}

export const Route = createFileRoute("/_app")({
  component: AppLayout,
});

function AppLayout() {
  const page = useAppLayout();
  const pathname = useLocation({ select: location => location.pathname });
  const reducedMotion = useReducedMotionConfig();
  const duration = resolveRouteTransitionDuration(Boolean(reducedMotion));

  if (!page.areWorkspacesLoading && !page.workspacesError && !page.hasWorkspaces) {
    return <WorkspaceOnboarding onWorkspaceResolved={page.setActiveWorkspaceId} />;
  }

  return (
    <>
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
      <main
        data-testid="app-content"
        className="relative flex min-h-0 flex-1 flex-col overflow-hidden bg-background"
      >
        <AnimatePresence mode="wait" initial={false}>
          <motion.div
            key={pathname}
            data-testid="app-route-motion"
            data-route-key={pathname}
            data-route-duration={duration}
            className="flex min-h-0 flex-1 flex-col"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration, ease: "easeOut" }}
          >
            <Outlet />
          </motion.div>
        </AnimatePresence>
      </main>
      <WorkspaceSetupDialog
        open={page.isWorkspaceSetupOpen}
        onOpenChange={page.setWorkspaceSetupOpen}
        onWorkspaceResolved={page.setActiveWorkspaceId}
      />
    </>
  );
}

export { resolveRouteTransitionDuration, ROUTE_FADE_DURATION };
