import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, Plus, Server } from "lucide-react";

import { useMcpPage } from "@/hooks/routes/use-mcp-page";
import {
  MCPActionResultBanner,
  MCPServerDeleteDialog,
  MCPServerEditor,
  MCPServersTable,
} from "@/systems/settings/components";
import { restartBannerPropsFor } from "@/systems/settings/lib/restart-banner-mapper";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  Button,
  Empty,
  PageShell,
  PillGroup,
  RestartBanner,
  Section,
  Spinner,
  StatusLineTopbarSlot,
  useTopbarSlot,
} from "@agh/ui";

export const Route = createFileRoute("/_app/mcp")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "MCP servers", icon: Server },
  }),
  component: MCPPage,
});

function MCPPage() {
  const page = useMcpPage();
  const scopeReady = page.activeScope === "global" || Boolean(page.activeWorkspaceId);
  const bannerProps = restartBannerPropsFor("mcp-servers", page.restart);

  useTopbarSlot({
    tabs: (
      <>
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
        {page.envelope ? (
          <StatusLineTopbarSlot
            data-testid="settings-page-mcp-servers-status-line"
            status="connected"
            items={[
              {
                key: "total",
                value: (
                  <span data-testid="settings-page-mcp-servers-total">
                    {page.counts.total} servers
                  </span>
                ),
                tone: "neutral",
              },
              {
                key: "scope",
                value: (
                  <span data-testid="settings-page-mcp-servers-scope-label">
                    scope:{" "}
                    {page.activeScope === "global"
                      ? "global"
                      : (page.activeWorkspace?.name ?? page.activeWorkspaceId ?? "workspace")}
                  </span>
                ),
                tone: "neutral",
              },
              {
                key: "shadowed",
                value: (
                  <span data-testid="settings-page-mcp-servers-shadowed-total">
                    {page.counts.shadowed} shadowed sources
                  </span>
                ),
                tone: "neutral",
              },
            ]}
          />
        ) : null}
      </>
    ),
  });

  if (!scopeReady) {
    return (
      <PageShell
        density="route"
        data-testid="settings-page-mcp-servers"
        banner={
          bannerProps ? (
            <RestartBanner {...bannerProps} className="px-6 md:px-8 xl:px-10" />
          ) : undefined
        }
      >
        <Empty
          icon={Server}
          title="Select a workspace"
          description="Choose a workspace in the sidebar to manage workspace MCP overrides, or switch scope to Global."
          data-testid="settings-page-mcp-servers-workspace-guard"
        />
      </PageShell>
    );
  }

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-mcp-servers-loading"
      >
        <Spinner className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.error || !page.envelope) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-mcp-servers-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">
            {page.error?.message ?? "Failed to load MCP servers"}
          </p>
        </div>
      </div>
    );
  }

  const scopeEyebrow = page.activeScope === "global" ? "Global servers" : "Workspace overrides";
  const scopeSummary =
    page.activeScope === "global"
      ? `${page.counts.total} defined · injected into every agent`
      : `${page.counts.total} overrides · scoped to ${page.activeWorkspace?.name ?? page.activeWorkspaceId}`;

  return (
    <PageShell
      density="route"
      data-testid="settings-page-mcp-servers"
      banner={
        bannerProps ? (
          <RestartBanner {...bannerProps} className="px-6 md:px-8 xl:px-10" />
        ) : undefined
      }
    >
      {page.lastAction ? (
        <MCPActionResultBanner action={page.lastAction} onDismiss={page.dismissLastAction} />
      ) : null}

      <Section
        data-testid="settings-page-mcp-servers-header-row"
        label={scopeEyebrow}
        note={scopeSummary}
        right={
          <Button
            type="button"
            variant="default"
            size="sm"
            onClick={page.openCreate}
            data-testid="settings-page-mcp-servers-create"
          >
            <Plus className="size-3" />
            Add server
          </Button>
        }
      />

      {page.servers.length === 0 ? (
        <Empty
          icon={Server}
          title="No MCP servers configured"
          description={
            page.activeScope === "global"
              ? 'Use "Add server" to create an entry in mcp.json or config.'
              : "No workspace overrides defined. Add one to shadow the global definition for this workspace."
          }
          data-testid="settings-page-mcp-servers-empty"
        />
      ) : (
        <MCPServersTable servers={page.servers} onEdit={page.openEdit} onDelete={page.openDelete} />
      )}

      <MCPServerEditor
        editor={page.editor}
        scope={page.activeScope}
        isValid={page.editorIsValid}
        isSaving={page.editorIsSaving}
        error={page.editorError}
        warnings={page.editorWarnings}
        existingNames={page.servers.map(entry => entry.name)}
        availableTargets={page.editorAvailableTargets}
        onChange={page.updateDraft}
        onTargetChange={page.setEditorTarget}
        onClose={page.closeEditor}
        onSave={page.saveEditor}
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
