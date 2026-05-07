import type { ReactNode } from "react";

import {
  Link,
  Outlet,
  createFileRoute,
  useRouter,
  type ErrorComponentProps,
  type NotFoundRouteProps,
} from "@tanstack/react-router";
import { AlertTriangle, Compass, RefreshCw } from "lucide-react";

import { Button, Empty, buttonVariants } from "@agh/ui";

import { AppSidebar } from "@/components/app-sidebar";
import { useAppLayout } from "@/hooks/routes/use-app-layout";
import { SessionCreateDialog, SessionCreateProvider } from "@/systems/session";
import { WorkspaceOnboarding, WorkspaceSetupDialog } from "@/systems/workspace";

export const Route = createFileRoute("/_app")({
  component: AppLayout,
  errorComponent: AppRouteErrorBoundary,
  notFoundComponent: AppRouteNotFoundBoundary,
});

function AppLayout() {
  const page = useAppLayout();

  if (!page.areWorkspacesLoading && !page.workspacesError && !page.hasWorkspaces) {
    return <WorkspaceOnboarding onWorkspaceResolved={page.setActiveWorkspaceId} />;
  }

  return (
    <SessionCreateProvider
      value={{
        openForAgent: page.handleNewSession,
        isCreating: page.isCreatingSession,
        pendingAgentName: page.pendingSessionAgentName,
        hasActiveWorkspace: page.activeWorkspace !== undefined,
      }}
    >
      <AppSidebar
        collapsed={page.collapsed}
        onCollapseChange={page.setCollapsed}
        workspaces={page.areWorkspacesLoading || page.workspacesError ? undefined : page.workspaces}
        activeWorkspaceId={page.activeWorkspaceId}
        onSelectWorkspace={page.setActiveWorkspaceId}
        onAddWorkspace={page.openWorkspaceSetup}
        health={page.health}
        connectionStatus={page.connectionStatus}
        agents={page.agents}
        agentsLoading={page.agentsLoading}
        agentsError={page.agentsError}
        sessions={page.sessions}
      />
      <main
        data-testid="app-content"
        className="relative flex min-h-0 flex-1 flex-col overflow-hidden bg-background"
      >
        <Outlet />
      </main>
      <WorkspaceSetupDialog
        open={page.isWorkspaceSetupOpen}
        onOpenChange={page.setWorkspaceSetupOpen}
        onWorkspaceResolved={page.setActiveWorkspaceId}
      />
      <SessionCreateDialog
        agents={page.sessionCreate.agents}
        catalogError={page.sessionCreate.catalogError}
        catalogLoading={page.sessionCreate.catalogLoading}
        catalogRefreshError={page.sessionCreate.catalogRefreshError}
        catalogRefreshing={page.sessionCreate.catalogRefreshing}
        catalogStale={page.sessionCreate.catalogStale}
        defaultReasoning={page.sessionCreate.defaultReasoning}
        isSubmitting={page.sessionCreate.isSubmitting}
        modelOptions={page.sessionCreate.modelOptions}
        onAgentChange={page.sessionCreate.onAgentChange}
        onCatalogRefresh={page.sessionCreate.refreshCatalog}
        onModelChange={page.sessionCreate.onModelChange}
        onOpenChange={page.sessionCreate.setOpen}
        onProviderChange={page.sessionCreate.onProviderChange}
        onReasoningChange={page.sessionCreate.onReasoningChange}
        onSubmit={page.sessionCreate.submit}
        open={page.sessionCreate.open}
        providerOptions={page.sessionCreate.providerOptions}
        providersError={page.sessionCreate.providersError}
        providersLoading={page.sessionCreate.providersLoading}
        reasoningOptions={page.sessionCreate.reasoningOptions}
        reasoningSupported={page.sessionCreate.reasoningSupported}
        selectedAgentName={page.sessionCreate.selectedAgentName}
        selectedModel={page.sessionCreate.selectedModel}
        selectedProvider={page.sessionCreate.selectedProvider}
        selectedProviderOption={page.sessionCreate.selectedProviderOption}
        selectedReasoning={page.sessionCreate.selectedReasoning}
        submitError={page.sessionCreate.submitError}
        workspace={page.sessionCreate.workspace}
      />
    </SessionCreateProvider>
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
