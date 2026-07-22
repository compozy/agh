import { Outlet } from "@tanstack/react-router";
import { useRef } from "react";

import { AgentCreateDialog, AgentCreateHostProvider } from "@/systems/agent";
import { SessionCreateDialog, SessionCreateProvider } from "@/systems/session";
import { WorkspaceOnboarding, WorkspaceSetupDialog } from "@/systems/workspace";

import { OsShellContext } from "../contexts/os-shell-context";
import { useDesktopChrome } from "../hooks/use-desktop-chrome";
import { useDesktopOverlays } from "../hooks/use-desktop-overlays";
import { useDesktopShellModel } from "../hooks/use-desktop-shell-model";
import { useDesktop } from "../hooks/use-desktop";
import { useOsShortcuts } from "../hooks/use-os-shortcuts";
import { useOsAttention } from "../hooks/use-os-attention";
import { DesktopGate } from "./desktop-gate";
import { DesktopMenubar } from "./desktop-menubar";
import { DesktopDock } from "./desktop-dock";
import { OsAppPreloader } from "./os-app-preloader";
import { OsCommandPalette } from "./os-command-palette";
import { OsSpacesOverview } from "./os-spaces-overview";
import { OsWallpaper } from "./os-wallpaper";
import { OsWinLayer } from "./os-win-layer";
import { OsSessionsModal } from "./sessions-modal";

/**
 * The desktop shell replaces the AppShell chrome (ADR-001): onboarding gate,
 * menubar, wallpapered win-layer, dock, ⌘K palette, and the desktop-state
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
          <DesktopShellBody model={model} wallpaper={chrome.wallpaper} />
        </AgentCreateHostProvider>
      </SessionCreateProvider>
    </OsShellContext.Provider>
  );
}

function DesktopShellBody({
  model,
  wallpaper,
}: {
  model: ReturnType<typeof useDesktopShellModel>;
  wallpaper: "ember" | "mesh" | "carbon";
}) {
  const desktopRef = useRef<HTMLDivElement>(null);
  const windows = useDesktop(state => state.windows);
  const overlays = useDesktopOverlays();
  const attention = useOsAttention(model.activeWorkspace, model.sessionCatalogStreamStatus);

  useOsShortcuts({
    onPalette: () => overlays.toggleOverlay("palette"),
    onNewSession: () => model.sessionCreate.openForAgent(""),
    onSpaces: () => overlays.toggleOverlay("spaces"),
    onEscape: () => {
      if (overlays.activeOverlay !== null) return;
      if (document.querySelector('[data-slot="dialog-content"]')) return;
      desktopRef.current?.focus();
    },
  });

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
        onOpenSpaces={() => overlays.setOverlayOpen("spaces", true)}
        activeOverlay={overlays.activeOverlay}
        onOverlayOpenChange={overlays.setOverlayOpen}
        attention={attention}
      />
      <div data-slot="os-desk" className="relative min-h-0 flex-1 overflow-hidden">
        <OsWallpaper wallpaper={wallpaper} />
        {Object.keys(windows).map(windowId => (
          <OsAppPreloader key={windowId} windowId={windowId} />
        ))}
        <OsWinLayer />
        <DesktopDock
          onNewSession={() => model.sessionCreate.openForAgent("")}
          badges={attention.badges}
          sessionsOpen={overlays.activeOverlay === "sessions"}
          onToggleSessions={() => overlays.toggleOverlay("sessions")}
        />
      </div>
      {/* Route matches mount here as sync-controllers; they render null. */}
      <Outlet />
      <OsCommandPalette
        open={overlays.activeOverlay === "palette"}
        onOpenChange={open => overlays.setOverlayOpen("palette", open)}
        onOpenSpaces={() => overlays.setOverlayOpen("spaces", true)}
        onToggleSessions={() => overlays.toggleOverlay("sessions")}
      />
      <OsSessionsModal
        open={overlays.activeOverlay === "sessions"}
        onOpenChange={open => overlays.setOverlayOpen("sessions", open)}
        sessions={attention.sessions}
        disconnected={attention.sessionsDisconnected}
      />
      <OsSpacesOverview
        open={overlays.activeOverlay === "spaces"}
        onOpenChange={open => overlays.setOverlayOpen("spaces", open)}
        workspaces={model.workspaces}
        activeWorkspaceId={model.activeWorkspaceId}
        onSelectWorkspace={model.setActiveWorkspaceId}
        onNewSpace={model.openWorkspaceSetup}
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
