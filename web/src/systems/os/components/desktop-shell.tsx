import { Outlet } from "@tanstack/react-router";

import { AgentCreateDialog, AgentCreateHostProvider } from "@/systems/agent";
import { SessionCreateDialog, SessionCreateProvider } from "@/systems/session";
import { WorkspaceOnboarding, WorkspaceSetupDialog } from "@/systems/workspace";

import { OsShellContext } from "../contexts/os-shell-context";
import { useDesktopChrome } from "../hooks/use-desktop-chrome";
import { useDesktopShellBody } from "../hooks/use-desktop-shell-body";
import { useDesktopShellModel, type DesktopShellModel } from "../hooks/use-desktop-shell-model";
import { DesktopGate } from "./desktop-gate";
import { DesktopMenubar } from "./desktop-menubar";
import { DesktopDock } from "./desktop-dock";
import { DesktopManagerSurfaces } from "./desktop-manager-surfaces";
import { DesktopPagerSurface } from "./desktop-pager-surface";
import { OsAppPreloader } from "./os-app-preloader";
import { OsCommandPalette } from "./os-command-palette";
import { OsWorkspacesOverview } from "./os-workspaces-overview";
import { OsWallpaper } from "./os-wallpaper";
import { OsWinLayer } from "./os-win-layer";
import { OsSessionsModal } from "./sessions-modal";

/**
 * The desktop shell replaces the AppShell chrome (ADR-001): onboarding gate,
 * menubar, wallpapered win-layer, dock, ⌘K palette, and the window-manager
 * sync lifecycle. Route matches render through the (invisible) Outlet as
 * sync-controllers; windows render in the layer.
 */
export function DesktopShell() {
  return (
    <DesktopGate>
      <DesktopChrome />
    </DesktopGate>
  );
}

function DesktopChrome() {
  const model = useDesktopShellModel();
  const chrome = useDesktopChrome(model.activeWorkspaceId);

  if (!model.areWorkspacesLoading && !model.workspacesError && !model.hasWorkspaces) {
    return <WorkspaceOnboarding onWorkspaceResolved={model.setActiveWorkspaceId} />;
  }

  return (
    <OsShellContext.Provider value={chrome.shell}>
      <SessionCreateProvider
        value={{
          openForAgent: model.sessionCreate.openForAgent,
          isCreating: model.sessionCreate.isSubmitting,
          pendingAgentName: model.sessionCreate.pendingAgentName,
          hasActiveWorkspace: model.activeWorkspace !== undefined,
        }}
      >
        <AgentCreateHostProvider
          openDialog={model.agentCreate.openDialog}
          openForDuplicate={model.agentCreate.openForDuplicate}
        >
          <DesktopShellBody model={model} />
        </AgentCreateHostProvider>
      </SessionCreateProvider>
    </OsShellContext.Provider>
  );
}

function DesktopShellBody({ model }: { model: DesktopShellModel }) {
  const {
    attention,
    desktop,
    desktopRef,
    gesturePreview,
    manager,
    managerSurfaces,
    onResize,
    onSeamPreview,
    onSeamPreviewEnd,
    overlays,
    pager,
    reducedMotion,
    transition,
    windowManagerActions,
    winLayer,
    workspaceDetails,
  } = useDesktopShellBody(model);

  return (
    <div
      ref={desktopRef}
      data-testid="os-desktop"
      tabIndex={-1}
      className="flex min-h-0 flex-1 flex-col overflow-hidden focus-visible:shadow-focus-inset focus-visible:outline-none"
    >
      <DesktopMenubar
        workspaces={model.workspaces}
        activeWorkspace={model.activeWorkspace}
        onSelectWorkspace={model.setActiveWorkspaceId}
        onAddWorkspace={model.openWorkspaceSetup}
        onNewSession={() => model.sessionCreate.openForAgent("")}
        onOpenPalette={() => overlays.setOverlayOpen("palette", true)}
        onOpenDesktops={() => overlays.setOverlayOpen("desktops", true)}
        onOpenWorkspaces={() => overlays.setOverlayOpen("workspaces", true)}
        activeOverlay={overlays.activeOverlay}
        onOverlayOpenChange={overlays.setOverlayOpen}
        attention={attention}
      />
      <div data-slot="os-desk" className="relative min-h-0 flex-1 overflow-hidden">
        <OsWallpaper wallpaper={desktop.wallpaper} />
        {Object.keys(desktop.windows).map(windowId => (
          <OsAppPreloader key={windowId} windowId={windowId} />
        ))}
        <OsWinLayer
          model={winLayer}
          reducedMotion={reducedMotion}
          transition={transition}
          preview={gesturePreview}
          onTransitionComplete={() => windowManagerActions.setTransitionIntent(null)}
          onResize={onResize}
          onSeamPreview={onSeamPreview}
          onSeamPreviewEnd={onSeamPreviewEnd}
        />
        <DesktopManagerSurfaces
          model={managerSurfaces}
          onCreateDesktop={() => manager.createDesktop()}
          onSwitchDesktop={desktopId => manager.switchDesktop(desktopId)}
          onRenameDesktop={(desktopId, name) => manager.renameDesktop(desktopId, name)}
          onReorderDesktop={(desktopId, order) => manager.reorderDesktop(desktopId, order)}
          onDeleteDesktop={(desktopId, destinationId) =>
            manager.deleteDesktop(desktopId, destinationId)
          }
          onMoveWindow={(windowId, destinationDesktopId) =>
            manager.moveWindowToDesktop(windowId, destinationDesktopId)
          }
          onOpenChange={open => {
            if (open) windowManagerActions.openOverlay({ kind: "desktops-overview" });
            else windowManagerActions.closeOverlay();
          }}
          onRetry={() => manager.refreshSnapshot()}
          onResolveConflict={() => {
            manager.clearConflict();
            manager.refreshSnapshot();
          }}
        />
        <DesktopDock
          onNewSession={() => model.sessionCreate.openForAgent("")}
          badges={attention.badges}
          sessionsOpen={overlays.activeOverlay === "sessions"}
          onToggleSessions={() => overlays.toggleOverlay("sessions")}
          pager={
            <DesktopPagerSurface
              activeDesktopId={pager.activeDesktopId}
              desktops={pager.desktops}
              compact={pager.compact}
              canSwitchDesktop={pager.canSwitchDesktop}
              onSelectDesktop={desktopId => manager.switchDesktop(desktopId)}
              onOpenOverview={request => windowManagerActions.requestOverviewSegment(request)}
            />
          }
        />
      </div>
      {/* Route matches mount here as sync-controllers; they render null. */}
      <Outlet />
      <OsCommandPalette
        open={overlays.activeOverlay === "palette"}
        onOpenChange={open => overlays.setOverlayOpen("palette", open)}
        onOpenDesktops={() => overlays.setOverlayOpen("desktops", true)}
        onToggleSessions={() => overlays.toggleOverlay("sessions")}
      />
      <OsSessionsModal
        open={overlays.activeOverlay === "sessions"}
        onOpenChange={open => overlays.setOverlayOpen("sessions", open)}
        sessions={attention.sessions}
        disconnected={attention.sessionsDisconnected}
      />
      <OsWorkspacesOverview
        open={overlays.activeOverlay === "workspaces"}
        onOpenChange={open => overlays.setOverlayOpen("workspaces", open)}
        workspaces={model.workspaces}
        activeWorkspaceId={model.activeWorkspaceId}
        onSelectWorkspace={model.setActiveWorkspaceId}
        onNewWorkspace={model.openWorkspaceSetup}
        details={workspaceDetails}
        presentation={pager.presentation}
        reducedMotion={reducedMotion}
      />
      <WorkspaceSetupDialog
        open={model.isWorkspaceSetupOpen}
        onOpenChange={model.setWorkspaceSetupOpen}
        onWorkspaceResolved={model.setActiveWorkspaceId}
      />
      <AgentCreateDialog
        draft={model.agentCreate.draft}
        hasActiveWorkspace={model.agentCreate.hasActiveWorkspace}
        isSubmitting={model.agentCreate.isSubmitting}
        modelCatalogError={model.agentCreate.modelCatalogError}
        modelCatalogLoading={model.agentCreate.modelCatalogLoading}
        modelCatalogLoaded={model.agentCreate.modelCatalogLoaded}
        modelCatalogRefreshing={model.agentCreate.modelCatalogRefreshing}
        onDraftChange={model.agentCreate.onDraftChange}
        onOpenChange={model.agentCreate.onOpenChange}
        onOpenProviderSettings={model.agentCreate.onOpenProviderSettings}
        onRefreshCatalog={model.agentCreate.onRefreshCatalog}
        onSubmit={model.agentCreate.onSubmit}
        open={model.agentCreate.open}
        providerOptions={model.agentCreate.providerOptions}
        providersError={model.agentCreate.providersError}
        providersLoading={model.agentCreate.providersLoading}
        runtimeModels={model.agentCreate.runtimeModels}
        submitError={model.agentCreate.submitError}
        workspaceName={model.agentCreate.workspaceName}
      />
      <SessionCreateDialog
        agents={model.sessionCreate.agents}
        catalogError={model.sessionCreate.catalogError}
        catalogLoading={model.sessionCreate.catalogLoading}
        catalogLoaded={model.sessionCreate.catalogLoaded}
        catalogRefreshError={model.sessionCreate.catalogRefreshError}
        catalogRefreshing={model.sessionCreate.catalogRefreshing}
        catalogStale={model.sessionCreate.catalogStale}
        hasProviderOptions={model.sessionCreate.hasProviderOptions}
        isSubmitting={model.sessionCreate.isSubmitting}
        networkParticipation={model.sessionCreate.networkParticipation}
        onAgentChange={model.sessionCreate.onAgentChange}
        onCatalogRefresh={model.sessionCreate.refreshCatalog}
        onNetworkParticipationChange={model.sessionCreate.onNetworkParticipationChange}
        onOpenChange={model.sessionCreate.onOpenChange}
        onOpenProviderSettings={model.sessionCreate.openProviderSettings}
        onRuntimeChange={model.sessionCreate.onRuntimeChange}
        onSubmit={model.sessionCreate.submit}
        open={model.sessionCreate.open}
        providersError={model.sessionCreate.providersError}
        providersLoading={model.sessionCreate.providersLoading}
        runtimeModels={model.sessionCreate.runtimeModels}
        runtimeProviders={model.sessionCreate.runtimeProviders}
        runtimeValue={model.sessionCreate.runtimeValue}
        selectedAgentName={model.sessionCreate.selectedAgentName}
        submitError={model.sessionCreate.submitError}
        workspace={model.sessionCreate.workspace}
      />
    </div>
  );
}
