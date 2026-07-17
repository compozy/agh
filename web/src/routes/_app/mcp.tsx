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
import { preloadMcpRoute } from "./-settings-preload";
import {
  Button,
  Empty,
  Eyebrow,
  PageShell,
  Pill,
  PillGroup,
  RestartBanner,
  SkeletonRows,
  useTopbarSlot,
} from "@agh/ui";

interface MCPRouteSearch {
  scope: "workspace" | "global";
  server?: string;
}

function validateMcpSearch(search: Record<string, unknown>): MCPRouteSearch {
  const server =
    typeof search.server === "string" && search.server.trim() !== "" ? search.server : undefined;
  return { scope: search.scope === "global" ? "global" : "workspace", server };
}

export const Route = createFileRoute("/_app/mcp")({
  validateSearch: validateMcpSearch,
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "MCP servers", icon: Server },
  }),
  loader: ({ context }) => preloadMcpRoute(context.queryClient),
  component: MCPPage,
});

const LEGEND = [
  { label: "ready or confirmed", tone: "success" as const },
  { label: "operator action", tone: "warning" as const },
  { label: "repair required", tone: "danger" as const },
  { label: "not run or not used", tone: "neutral" as const },
];

function MCPPage() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const page = useMcpPage({
    scope: search.scope,
    selectedServer: search.server ?? "",
    onScopeChange: scope => navigate({ search: prev => ({ ...prev, scope, server: undefined }) }),
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
          vaultRefs={page.editorVaultRefs}
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

function MCPContextHeader({
  scope,
  workspaceName,
}: {
  scope: "workspace" | "global";
  workspaceName: string;
}) {
  const scopeContext =
    scope === "global" ? "global · operator home" : `workspace · ${workspaceName}`;
  return (
    <div className="mb-4 flex items-start gap-4" data-testid="settings-page-mcp-context">
      <div className="min-w-0 flex-1">
        <Eyebrow className="text-muted">Runtime inventory</Eyebrow>
        <h2 className="mt-1 text-lg font-medium tracking-tight text-fg-strong">Server status</h2>
        <p className="mt-1 max-w-2xl text-small-body text-muted">
          Configuration, authorization, runtime, and probe results are independent signals. A
          configured server is not assumed ready.
        </p>
        <div
          className="mt-1 font-mono text-caption text-muted"
          data-testid="settings-page-mcp-servers-scope-label"
        >
          {scopeContext}
        </div>
      </div>
      <div className="hidden max-w-[440px] flex-wrap justify-end gap-x-3 gap-y-1.5 pt-1 md:flex">
        {LEGEND.map(item => (
          <span key={item.label} className="flex items-center gap-1.5 text-caption text-subtle">
            <Pill.Dot tone={item.tone} />
            {item.label}
          </span>
        ))}
      </div>
    </div>
  );
}

function MCPMatrixSkeleton() {
  return (
    <div
      className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
      data-testid="settings-page-mcp-servers-loading"
    >
      <SkeletonRows count={5} className="p-3.5" />
    </div>
  );
}
