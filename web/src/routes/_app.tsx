import type { ReactNode } from "react";

import {
  Link,
  Outlet,
  createFileRoute,
  useLocation,
  useRouter,
  type ErrorComponentProps,
  type NotFoundRouteProps,
} from "@tanstack/react-router";
import { AlertTriangle, Compass, RefreshCw } from "lucide-react";
import { AnimatePresence, motion, useReducedMotionConfig } from "motion/react";

import { Button, Empty, buttonVariants } from "@agh/ui";

import { AppSidebar } from "@/components/app-sidebar";
import { useAppLayout } from "@/hooks/routes/use-app-layout";
import { SessionCreateDialog } from "@/systems/session";
import { WorkspaceOnboarding, WorkspaceSetupDialog } from "@/systems/workspace";

const ROUTE_FADE_DURATION = 0.2;

function resolveRouteTransitionDuration(reducedMotion: boolean): number {
  return reducedMotion ? 0 : ROUTE_FADE_DURATION;
}

export const Route = createFileRoute("/_app")({
  component: AppLayout,
  errorComponent: AppRouteErrorBoundary,
  notFoundComponent: AppRouteNotFoundBoundary,
});

function AppLayout() {
  const page = useAppLayout();
  const router = useRouter();
  const locationPathname = useLocation({ select: location => location.pathname });
  // Keep the motion shell keyed to the browser's latest path so pending-route
  // resolution cannot remount the active screen later and discard local UI state.
  const pathname = router.latestLocation.pathname || locationPathname;
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
        pendingSessionAgentName={page.pendingSessionAgentName}
        pendingSessionWorkspaceId={page.pendingSessionWorkspaceId}
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
      <SessionCreateDialog
        agents={page.sessionCreate.agents}
        isSubmitting={page.sessionCreate.isSubmitting}
        onAgentChange={page.sessionCreate.onAgentChange}
        onOpenChange={page.sessionCreate.setOpen}
        onProviderChange={page.sessionCreate.onProviderChange}
        onSubmit={page.sessionCreate.submit}
        open={page.sessionCreate.open}
        providerOptions={page.sessionCreate.providerOptions}
        providersError={page.sessionCreate.providersError}
        providersLoading={page.sessionCreate.providersLoading}
        selectedAgentName={page.sessionCreate.selectedAgentName}
        selectedProvider={page.sessionCreate.selectedProvider}
        submitError={page.sessionCreate.submitError}
        workspace={page.sessionCreate.workspace}
      />
    </>
  );
}

function AppRouteErrorBoundary({ error, reset }: ErrorComponentProps) {
  const router = useRouter();

  const handleRetry = () => {
    reset();
    void router.invalidate({ forcePending: true });
  };

  return (
    <AppRouteBoundaryFrame testId="app-route-error">
      <Empty
        className="max-w-xl"
        description={describeRouteError(error, "The requested app route could not be rendered.")}
        icon={AlertTriangle}
        title="Unable to load this page"
        titleAs="h1"
        action={
          <>
            <Button onClick={handleRetry} size="sm" type="button" variant="outline">
              <RefreshCw className="size-3.5" />
              Retry
            </Button>
            <Link className={buttonVariants({ variant: "outline", size: "sm" })} to="/">
              <Compass className="size-3.5" />
              Go home
            </Link>
          </>
        }
      />
    </AppRouteBoundaryFrame>
  );
}

function AppRouteNotFoundBoundary({ routeId }: NotFoundRouteProps) {
  return (
    <AppRouteBoundaryFrame routeId={routeId} testId="app-route-not-found">
      <Empty
        className="max-w-xl"
        description="The requested app route does not exist."
        icon={Compass}
        title="Page not found"
        titleAs="h1"
        action={
          <Link className={buttonVariants({ variant: "outline", size: "sm" })} to="/">
            <Compass className="size-3.5" />
            Go home
          </Link>
        }
      />
    </AppRouteBoundaryFrame>
  );
}

function AppRouteBoundaryFrame({
  children,
  routeId,
  testId,
}: {
  children: ReactNode;
  routeId?: string;
  testId: string;
}) {
  return (
    <main
      data-route-id={routeId}
      data-testid={testId}
      className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto bg-background px-6 py-8"
    >
      {children}
    </main>
  );
}

function describeRouteError(error: unknown, fallback: string) {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }

  return fallback;
}

export { resolveRouteTransitionDuration, ROUTE_FADE_DURATION };
