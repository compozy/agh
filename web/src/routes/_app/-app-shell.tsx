import { Outlet } from "@tanstack/react-router";
import { AlertTriangle, RefreshCw } from "lucide-react";

import { Button, Empty, Spinner } from "@agh/ui";

import { TopbarShell } from "@/components/topbar-shell";
import { useAppLayout } from "@/hooks/routes/use-app-layout";
import { AgentCreateDialog, AgentCreateHostProvider } from "@/systems/agent";
import { OnboardingWizard, useOnboardingStatus } from "@/systems/onboarding";
import { AppSidebar } from "@/systems/runtime";
import { SessionCreateDialog, SessionCreateProvider } from "@/systems/session";
import { WorkspaceOnboarding, WorkspaceSetupDialog } from "@/systems/workspace";
import { OnboardingGateFrame } from "./-onboarding-gate-frame";
export function AppLayout() {
  const onboarding = useOnboardingStatus();

  if (onboarding.data?.completed === true) {
    return <AppShell />;
  }

  if (onboarding.data?.completed === false) {
    return <OnboardingWizard onComplete={() => void onboarding.refetch()} />;
  }

  if (onboarding.isError) {
    return (
      <OnboardingGateFrame testId="onboarding-gate-error">
        <Empty
          className="max-w-xl"
          description={describeRouteError(
            onboarding.error,
            "AGH could not confirm whether first-run setup is complete."
          )}
          icon={AlertTriangle}
          title="Unable to check onboarding"
          titleAs="h1"
          action={
            <Button
              onClick={() => void onboarding.refetch()}
              size="sm"
              type="button"
              variant="outline"
            >
              <RefreshCw className="size-3" />
              Retry
            </Button>
          }
        />
      </OnboardingGateFrame>
    );
  }

  return (
    <OnboardingGateFrame testId="onboarding-gate-loading">
      <Spinner />
    </OnboardingGateFrame>
  );
}

function AppShell() {
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
      <AgentCreateHostProvider
        openDialog={page.agentCreate.openDialog}
        openForDuplicate={page.agentCreate.openForDuplicate}
      >
        <div
          data-testid="app-grid"
          className="grid min-h-0 flex-1 grid-cols-[56px_minmax(0,1fr)] grid-rows-[48px_1fr] overflow-hidden min-[880px]:grid-cols-[56px_220px_minmax(0,1fr)] min-[1100px]:grid-cols-[56px_244px_minmax(0,1fr)]"
        >
          <AppSidebar
            className="col-span-1 row-span-2 min-[880px]:col-span-2"
            collapsed={page.collapsed}
            onCollapseChange={page.setCollapsed}
            workspaces={
              page.areWorkspacesLoading || page.workspacesError ? undefined : page.workspaces
            }
            activeWorkspaceId={page.activeWorkspaceId}
            onSelectWorkspace={page.setActiveWorkspaceId}
            onAddWorkspace={page.openWorkspaceSetup}
            agentsCount={page.agentsCount}
            activeSessionCount={page.activeSessionCount}
            activeWorkspace={page.activeWorkspace}
          />
          <TopbarShell>
            <main
              id="app-content"
              data-testid="app-content"
              className="relative col-start-2 row-start-2 flex min-h-0 flex-col overflow-hidden bg-canvas min-[880px]:col-start-3"
            >
              <Outlet />
            </main>
          </TopbarShell>
        </div>
        <WorkspaceSetupDialog
          open={page.isWorkspaceSetupOpen}
          onOpenChange={page.setWorkspaceSetupOpen}
          onWorkspaceResolved={page.setActiveWorkspaceId}
        />
        <AgentCreateDialog
          draft={page.agentCreate.draft}
          hasActiveWorkspace={page.agentCreate.hasActiveWorkspace}
          isSubmitting={page.agentCreate.isSubmitting}
          modelCatalogError={page.agentCreate.modelCatalogError}
          modelCatalogLoading={page.agentCreate.modelCatalogLoading}
          modelCatalogLoaded={page.agentCreate.modelCatalogLoaded}
          modelCatalogRefreshing={page.agentCreate.modelCatalogRefreshing}
          onDraftChange={page.agentCreate.onDraftChange}
          onOpenChange={page.agentCreate.onOpenChange}
          onOpenProviderSettings={page.agentCreate.onOpenProviderSettings}
          onRefreshCatalog={page.agentCreate.onRefreshCatalog}
          onSubmit={page.agentCreate.onSubmit}
          open={page.agentCreate.open}
          providerOptions={page.agentCreate.providerOptions}
          providersError={page.agentCreate.providersError}
          providersLoading={page.agentCreate.providersLoading}
          runtimeModels={page.agentCreate.runtimeModels}
          submitError={page.agentCreate.submitError}
          workspaceName={page.agentCreate.workspaceName}
        />
        <SessionCreateDialog
          agents={page.sessionCreate.agents}
          catalogError={page.sessionCreate.catalogError}
          catalogLoading={page.sessionCreate.catalogLoading}
          catalogLoaded={page.sessionCreate.catalogLoaded}
          catalogRefreshError={page.sessionCreate.catalogRefreshError}
          catalogRefreshing={page.sessionCreate.catalogRefreshing}
          catalogStale={page.sessionCreate.catalogStale}
          hasProviderOptions={page.sessionCreate.hasProviderOptions}
          isSubmitting={page.sessionCreate.isSubmitting}
          onAgentChange={page.sessionCreate.onAgentChange}
          onCatalogRefresh={page.sessionCreate.refreshCatalog}
          onOpenChange={page.sessionCreate.setOpen}
          onOpenProviderSettings={page.sessionCreate.openProviderSettings}
          onRuntimeChange={page.sessionCreate.onRuntimeChange}
          onSubmit={page.sessionCreate.submit}
          open={page.sessionCreate.open}
          providersError={page.sessionCreate.providersError}
          providersLoading={page.sessionCreate.providersLoading}
          runtimeModels={page.sessionCreate.runtimeModels}
          runtimeProviders={page.sessionCreate.runtimeProviders}
          runtimeValue={page.sessionCreate.runtimeValue}
          selectedAgentName={page.sessionCreate.selectedAgentName}
          submitError={page.sessionCreate.submitError}
          workspace={page.sessionCreate.workspace}
        />
      </AgentCreateHostProvider>
    </SessionCreateProvider>
  );
}

function describeRouteError(error: unknown, fallback: string) {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }

  return fallback;
}
