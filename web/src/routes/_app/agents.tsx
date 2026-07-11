import { AlertCircle, Plus, RefreshCw, Users2 } from "lucide-react";
import { Outlet, createFileRoute } from "@tanstack/react-router";

import { Button, Empty, ListingPage, useTopbarSlot } from "@agh/ui";

import { useAgentsFleetPage } from "@/hooks/routes/use-agents-fleet-page";
import { AgentFleetList, AgentFleetToolbar, validateAgentsFleetSearch } from "@/systems/agent";
import type { TopbarRouteContext } from "@/types/topbar";

export const Route = createFileRoute("/_app/agents")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Agents", icon: Users2 },
  }),
  validateSearch: validateAgentsFleetSearch,
  component: AgentsFleetRoute,
});

function AgentsFleetRoute() {
  const page = useAgentsFleetPage(Route.useSearch());

  useTopbarSlot({
    count: page.hasChildMatch || page.isFirstRunEmpty ? undefined : page.fleetTotal,
    actions:
      page.hasChildMatch || page.isFirstRunEmpty ? undefined : (
        <div className="flex items-center gap-2" data-testid="agents-topbar-actions">
          <Button
            data-testid="agents-topbar-create"
            onClick={page.openCreate}
            size="sm"
            type="button"
          >
            <Plus aria-hidden="true" className="size-3" />
            New agent
          </Button>
        </div>
      ),
  });

  if (page.hasChildMatch) {
    return <Outlet />;
  }

  if (page.workspaceId === "") {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
        data-testid="agents-no-workspace"
      >
        <Empty
          description="Select a workspace to browse its agents."
          icon={Users2}
          title="No workspace selected"
        />
      </div>
    );
  }

  const agentsErrorEmpty = Boolean(page.agentsError) && page.agents.length === 0 && !page.isLoading;
  const headCount =
    page.isLoading || agentsErrorEmpty || page.isFirstRunEmpty ? 0 : page.fleetTotal;

  return (
    <ListingPage data-testid="agent-fleet-page">
      <ListingPage.Head
        count={headCount}
        countTestId="agents-page-count"
        data-testid="agents-page-head"
        meta={
          <>
            <span>Provider, model, and instructions a session runs with.</span>
            <ListingPage.MetaDot />
            <span>Operate</span>
          </>
        }
        title="Agents"
      />

      <AgentFleetToolbar
        categoryOptions={page.categoryOptions}
        draftQuery={page.draftQuery}
        onDraftQueryChange={page.setDraftQuery}
        onFiltersChange={page.setFilters}
        onViewChange={page.setView}
        search={page.search}
        searchInputRef={page.searchInputRef}
        showFacets={page.showFacets}
        view={page.view}
      />

      {agentsErrorEmpty ? (
        <div
          className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
          data-testid="agent-fleet-error"
        >
          <Empty
            action={
              <Button
                data-testid="agent-fleet-error-retry"
                onClick={page.retryAgents}
                size="sm"
                type="button"
                variant="ghost"
              >
                <RefreshCw aria-hidden="true" className="size-3" />
                Retry
              </Button>
            }
            description="The agents request failed. Check the daemon connection and try again."
            icon={AlertCircle}
            title="Couldn't load agents"
          />
        </div>
      ) : (
        <AgentFleetList
          isFilteredEmpty={page.isFilteredEmpty}
          isFirstRunEmpty={page.isFirstRunEmpty}
          isLoading={page.isLoading}
          newSessionDisabled={page.newSessionDisabled}
          onClearFilters={page.clearFilters}
          onCreateAgent={page.openCreate}
          onNewSession={page.openNewSession}
          rows={page.rows}
          sessionsPartial={page.sessionsPartial}
          view={page.view}
        />
      )}
    </ListingPage>
  );
}
