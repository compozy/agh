import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, Plus, Server } from "lucide-react";

import { useMcpPage } from "@/hooks/routes/use-mcp-page";
import {
  MCPActionResultBanner,
  MCPAuthorizeDialog,
  MCPSelectionStrip,
  MCPServerDeleteDialog,
  MCPServerEditor,
  MCPServersTable,
} from "@/systems/settings/components";
import { restartBannerPropsFor } from "@/systems/settings/lib/restart-banner-mapper";
import type { TopbarRouteContext } from "@/types/topbar";
import { MCPContextHeader, MCPMatrixSkeleton } from "./-mcp-context";
import { preloadMcpRoute } from "./-settings-preload";
import { Button, Empty, PageShell, PillGroup, RestartBanner, useTopbarSlot } from "@agh/ui";

interface MCPRouteSearch {
  scope: "workspace" | "global";
  server?: string;
  workspace_id?: string;
}

function validateMcpSearch(search: Record<string, unknown>): MCPRouteSearch {
  const server =
    typeof search.server === "string" && search.server.trim() !== "" ? search.server : undefined;
  const workspaceId =
    typeof search.workspace_id === "string" && search.workspace_id.trim() !== ""
      ? search.workspace_id.trim()
      : undefined;
  return {
    scope: search.scope === "global" ? "global" : "workspace",
    server,
    workspace_id: workspaceId,
  };
}

export const Route = createFileRoute("/_app/mcp")({
  validateSearch: validateMcpSearch,
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "MCP servers", icon: Server },
  }),
  loader: ({ context }) => preloadMcpRoute(context.queryClient),
  component: MCPPage,
});

function MCPPage() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const page = useMcpPage({
    scope: search.scope,
    workspaceId: search.workspace_id,
    selectedServer: search.server ?? "",
    onScopeChange: scope =>
      navigate({
        search: prev => ({
          ...prev,
          scope,
          server: undefined,
          workspace_id: scope === "workspace" ? prev.workspace_id : undefined,
        }),
      }),
    onWorkspaceChange: workspaceId =>
      navigate({
        search: prev => ({ ...prev, workspace_id: workspaceId, server: undefined }),
      }),
    onSelectServer: server =>
      navigate({ search: prev => ({ ...prev, server: server || undefined }) }),
  });
  const scopeReady = page.activeScope === "global" || Boolean(page.activeWorkspaceId);
  const bannerProps = restartBannerPropsFor("mcp-servers", page.restart);
  const editor = page.editor;

  useTopbarSlot({
    count: page.envelope ? page.counts.total : undefined,
    tabs: (
      <PillGroup<"workspace" | "global">
        aria-label="MCP scope"
        data-testid="mcp-scope-pills"
        items={[
          { value: "workspace", label: "Workspace", testId: "mcp-scope-workspace" },
          { value: "global", label: "Global", testId: "mcp-scope-global" },
        ]}
        onChange={page.selectScope}
        value={page.activeScope}
      />
    ),
    actions: scopeReady ? (
      <Button
        type="button"
        size="sm"
        onClick={page.openCreate}
        data-testid="settings-page-mcp-servers-create"
      >
        <Plus className="size-3" />
        Add server
      </Button>
    ) : null,
  });

  const banner = bannerProps ? (
    <RestartBanner {...bannerProps} className="px-6 md:px-8 xl:px-10" />
  ) : undefined;

  if (!scopeReady) {
    return (
      <PageShell density="route" data-testid="settings-page-mcp-servers" banner={banner}>
        <Empty
          icon={Server}
          title="Select a workspace"
          description="Choose a workspace in the sidebar to manage workspace MCP overrides, or switch scope to Global."
          data-testid="settings-page-mcp-servers-workspace-guard"
        />
      </PageShell>
    );
  }

  return (
    <PageShell density="route" data-testid="settings-page-mcp-servers" banner={banner}>
      {page.lastAction ? (
        <MCPActionResultBanner action={page.lastAction} onDismiss={page.dismissLastAction} />
      ) : null}

      <MCPContextHeader
        scope={page.activeScope}
        workspaceName={page.activeWorkspace?.name ?? page.activeWorkspaceId ?? "workspace"}
      />

      {page.isLoading ? (
        <MCPMatrixSkeleton />
      ) : page.error || !page.envelope ? (
        <Empty
          icon={AlertCircle}
          title="MCP servers could not be loaded"
          description="The daemon stopped responding before it returned the scoped server list. Existing configuration was not changed."
          action={
            <Button
              type="button"
              variant="neutral"
              size="sm"
              onClick={() => void page.refetch()}
              data-testid="settings-page-mcp-servers-retry"
            >
              Retry load
            </Button>
          }
          data-testid="settings-page-mcp-servers-error"
        />
      ) : (
        <>
          <MCPSelectionStrip
            selectedName={page.selectedServer}
            server={page.selectedEntry}
            scope={page.activeScope}
            onClear={page.clearSelection}
          />
          {page.servers.length === 0 ? (
            <Empty
              icon={Server}
              title="No MCP servers configured"
              description={
                page.activeScope === "global"
                  ? "Add a local stdio server or a remote HTTP/SSE server for the global scope."
                  : "No workspace overrides defined. Add one to shadow the global definition for this workspace."
              }
              action={
                <Button type="button" size="sm" onClick={page.openCreate}>
                  <Plus className="size-3" />
                  Add MCP server
                </Button>
              }
              data-testid="settings-page-mcp-servers-empty"
            />
          ) : (
            <MCPServersTable
              servers={page.servers}
              selectedServer={page.selectedServer}
              onSelect={page.selectServer}
              onEdit={page.openEdit}
              onAuthorize={page.openAuthorize}
            />
          )}
        </>
      )}

      {editor.mode !== "closed" ? (
        <MCPServerEditor
          open
          mode={editor.mode}
          draft={editor.draft}
          scope={page.activeScope}
          errors={page.editorErrors}
          isValid={page.editorIsValid}
          isSaving={page.editorIsSaving}
          saveError={page.editorSaveError}
          warnings={page.editorWarnings}
          vaultInventory={page.editorVaultInventory}
          target={editor.target}
          availableTargets={page.editorAvailableTargets}
          entry={editor.mode === "edit" ? editor.entry : null}
          onChange={page.updateDraft}
          onTargetChange={page.setEditorTarget}
          onClose={page.closeEditor}
          onSave={page.saveEditor}
          onRemove={editor.mode === "edit" ? page.requestRemoveFromEditor : undefined}
        />
      ) : null}

      <MCPAuthorizeDialog
        authorize={page.authorize}
        scope={page.activeScope}
        server={page.authorizeEntry}
      />

      <MCPServerDeleteDialog
        target={page.deleteTarget.mode === "open" ? page.deleteTarget.entry : null}
        selectedTarget={page.deleteTarget.mode === "open" ? page.deleteTarget.target : "auto"}
        availableTargets={page.deleteAvailableTargets}
        error={page.deleteError}
        isDeleting={page.deleteIsPending}
        onTargetChange={page.setDeleteTargetKind}
        onClose={page.closeDelete}
        onConfirm={page.confirmDelete}
      />
    </PageShell>
  );
}
