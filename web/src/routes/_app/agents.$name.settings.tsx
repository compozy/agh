import { createFileRoute } from "@tanstack/react-router";
import { AlertCircle, Settings2 } from "lucide-react";

import { Button, Empty, PageActionsTopbarSlot, Pill, Skeleton, useTopbarSlot, cn } from "@agh/ui";

import { useAgentSettingsPage } from "@/systems/agent/hooks/use-agent-settings-page";
import { useLatestAgentSettingsActions } from "@/systems/agent/hooks/use-latest-agent-settings-actions";
import { AgentSettingsPanels } from "@/systems/agent/components/agent-settings-panels";
import {
  AGENT_SETTINGS_SECTIONS,
  resolveAgentSettingsSearch,
  validateAgentSettingsSearch,
  type AgentSettingsSection,
} from "@/systems/agent/lib/agent-settings-search";
import {
  ACTIVE_NAV_INDICATOR_CLASS,
  ACTIVE_NAV_ROW_CLASS,
  NAV_ROW_CLASS,
} from "@/components/sidebar-nav-classes";
import type { TopbarRouteContext } from "@/types/topbar";

const SECTION_LABELS: Record<AgentSettingsSection, string> = {
  basics: "Basics",
  runtime: "Runtime",
  instructions: "Instructions",
  access: "Access",
  mcp: "MCP servers",
  danger: "Danger zone",
};

export const Route = createFileRoute("/_app/agents/$name/settings")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: {
      title: "Settings",
      icon: Settings2,
    },
  }),
  validateSearch: validateAgentSettingsSearch,
  component: AgentSettingsPage,
});

function AgentSettingsPage() {
  const { name } = Route.useParams();
  const rawSearch = Route.useSearch();
  const search = resolveAgentSettingsSearch(rawSearch);
  const page = useAgentSettingsPage({ name, section: search.section });
  const topbarActions = useLatestAgentSettingsActions(page.onSave, page.onDiscard);

  useTopbarSlot({
    back: page.onBackToDetail,
    backLabel: name,
    title: "Settings",
    meta: page.dirty ? (
      <Pill tone="warning" size="sm" data-testid="agent-settings-unsaved">
        Unsaved
      </Pill>
    ) : undefined,
    actions:
      page.dirty || page.isSaving ? (
        <PageActionsTopbarSlot
          dirty={page.dirty}
          saving={page.isSaving}
          saveBlocked={page.saveBlocked}
          saveBlockedCaption={page.saveBlockedCaption}
          onSave={topbarActions.onSave}
          onDiscard={topbarActions.onDiscard}
          data-testid="agent-settings-page-actions"
        />
      ) : undefined,
  });

  if (page.agentLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 flex-col gap-4 px-6 py-5"
        data-testid="agent-settings-loading"
      >
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 rounded-md" />
      </div>
    );
  }

  if (page.agentError || !page.agent || !page.draft) {
    return (
      <div className="flex flex-1 items-center justify-center px-6 py-8">
        <Empty
          icon={AlertCircle}
          title="Agent not found"
          description={
            page.agentError?.message ?? `No agent named "${name}" was found in this workspace.`
          }
          action={
            <Button type="button" variant="ghost" size="sm" onClick={page.onBackToDetail}>
              Back to agent
            </Button>
          }
          data-testid="agent-settings-not-found"
        />
      </div>
    );
  }

  return (
    <div
      className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden xl:flex-row"
      data-testid="agent-settings-page"
    >
      {page.unsavedGuardDialog}
      {page.deleteFlow.confirmDialog}
      <nav
        aria-label="Agent settings sections"
        className="flex w-full shrink-0 flex-wrap gap-1 overflow-y-auto border-b border-line bg-canvas px-2 py-2 xl:w-settings-nav xl:flex-col xl:flex-nowrap xl:border-r xl:border-b-0 xl:py-3"
        data-testid="agent-settings-section-nav"
      >
        {AGENT_SETTINGS_SECTIONS.map(section => {
          const isActive = search.section === section;
          const isDanger = section === "danger";
          return (
            <button
              key={section}
              type="button"
              data-testid={`agent-settings-nav-${section}`}
              data-active={isActive ? "true" : "false"}
              aria-current={isActive ? "page" : undefined}
              onClick={() => page.setSection(section)}
              className={cn(
                NAV_ROW_CLASS,
                "w-auto shrink-0 xl:w-full",
                isActive && ACTIVE_NAV_ROW_CLASS,
                isDanger && "text-danger hover:text-danger",
                isDanger && isActive && "bg-danger-tint text-danger"
              )}
            >
              {isActive ? (
                <span
                  aria-hidden="true"
                  className={cn(ACTIVE_NAV_INDICATOR_CLASS, "hidden xl:block")}
                />
              ) : null}
              <span className="whitespace-nowrap xl:truncate">{SECTION_LABELS[section]}</span>
            </button>
          );
        })}
      </nav>
      <div className="relative flex min-w-0 flex-1 flex-col overflow-hidden">
        <AgentSettingsPanels
          section={search.section}
          agent={page.agent}
          draft={page.draft}
          validation={page.validation}
          disabled={page.fieldsDisabled}
          readOnly={page.fieldsReadOnly}
          mutationDenied={page.mutationDenied}
          saveError={page.saveError}
          conflictBanner={page.conflictBanner}
          onReloadAndRetry={() => {
            void page.onReloadAndRetry();
          }}
          onPatch={page.patchDraft}
          onDelete={page.deleteFlow.openDialog}
          isDeleting={page.deleteFlow.isDeleting}
          providerOptions={page.providerOptions}
          providersLoading={page.providersLoading}
          runtimeModels={page.runtimeModels}
          modelCatalogLoading={page.modelCatalogLoading}
          modelCatalogLoaded={page.modelCatalogLoaded}
          modelCatalogRefreshing={page.modelCatalogRefreshing}
          modelCatalogError={page.modelCatalogError}
          onRefreshCatalog={page.onRefreshCatalog}
          onOpenProviderSettings={page.onOpenProviderSettings}
        />
      </div>
    </div>
  );
}
