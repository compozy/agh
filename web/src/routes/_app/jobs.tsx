import { AlertCircle, Clock3, Plus } from "lucide-react";
import { Outlet, createFileRoute } from "@tanstack/react-router";

import {
  Alert,
  AlertDescription,
  AlertTitle,
  Button,
  Empty,
  ListingPage,
  ListingToolbar,
  PageHead,
  Spinner,
  useTopbarSlot,
} from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  AutomationEditorDialog,
  AutomationJobsCatalog,
  AutomationListFilters,
  parseAutomationEnabled,
  parseAutomationScope,
  parseAutomationSource,
} from "@/systems/automation";
import {
  automationListLoopFilter,
  useAutomationJobsPage,
  type AutomationRouteSearch,
} from "@/hooks/routes/use-automation-page";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import { useActiveWorkspace } from "@/systems/workspace";
import { preloadAutomationJobsRoute } from "./-automation-preload";

function validateJobsSearch(search: Record<string, unknown>): AutomationRouteSearch {
  return {
    create: search.create === "loop" ? "loop" : undefined,
    enabled: parseAutomationEnabled(search.enabled),
    loop: normalizeListingSearchValue(search.loop),
    q: normalizeListingSearchValue(search.q),
    scope: parseAutomationScope(search.scope),
    source: parseAutomationSource(search.source),
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/jobs")({
  validateSearch: validateJobsSearch,
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Jobs", to: "/jobs" } },
  }),
  loaderDeps: ({ search }) => ({
    enabled: search.enabled,
    loop: automationListLoopFilter(search),
    q: search.q,
    scope: search.scope,
    source: search.source,
  }),
  loader: ({ context, deps }) => preloadAutomationJobsRoute(context.queryClient, deps),
  component: JobsPage,
});

function JobsPage() {
  const search = Route.useSearch();
  const page = useAutomationJobsPage(
    search.create === "loop" && search.loop ? { loop: search.loop } : {},
    search
  );
  const { activeWorkspace } = useActiveWorkspace();

  // Publish null while a child route is mounted: a non-null slot from this
  // parent would steal the detail route's publish (layout effects run child
  // first, parent last in the same commit).
  useTopbarSlot(
    page.hasChildMatch
      ? null
      : {
          actions: (
            <Button
              data-testid="create-job-btn"
              onClick={page.handleCreate}
              size="sm"
              type="button"
            >
              <Plus aria-hidden="true" className="size-3" />
              Job
            </Button>
          ),
        }
  );

  if (page.hasChildMatch) {
    return <Outlet />;
  }

  if (page.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center" data-testid="jobs-loading">
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.error) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center py-10"
        data-testid="jobs-error"
      >
        <Empty
          className="max-w-md"
          description={page.error.message ?? "Failed to load jobs"}
          icon={AlertCircle}
          title="Unable to load jobs"
        />
      </div>
    );
  }

  const workspaceLabel = activeWorkspace?.name ?? activeWorkspace?.id ?? "workspace";

  return (
    <>
      <ListingPage
        banner={
          page.runtimeUnavailableMessage ? (
            <div className="border-b border-line px-9 py-3">
              <Alert data-testid="jobs-runtime-alert" variant="warning">
                <AlertCircle aria-hidden="true" className="size-4" />
                <AlertTitle>Automation runtime unavailable</AlertTitle>
                <AlertDescription>{page.runtimeUnavailableMessage}</AlertDescription>
              </Alert>
            </div>
          ) : null
        }
        data-testid="jobs-shell"
      >
        <PageHead
          count={page.total}
          countTestId="jobs-count"
          data-testid="jobs-page-head"
          icon={Clock3}
          meta={
            <>
              <span>Scheduled targets that run agents, tasks, or Loops.</span>
              <PageHead.MetaDot />
              <span>{workspaceLabel}</span>
            </>
          }
          title="Jobs"
        />

        <ListingToolbar>
          <ListingToolbar.Leading>
            <ListingToolbar.Search
              aria-label="Search jobs"
              data-testid="automation-search-input"
              onChange={page.setSearchQuery}
              placeholder="Search jobs"
              value={page.searchQuery}
            />
            <ListingToolbar.Filters>
              <AutomationListFilters
                enabledFilter={page.enabledFilter}
                kind="jobs"
                onEnabledFilterChange={page.setEnabledFilter}
                onScopeFilterChange={page.setScopeFilter}
                onSourceFilterChange={page.setSourceFilter}
                scopeFilter={page.scopeFilter}
                sourceFilter={page.sourceFilter}
              />
            </ListingToolbar.Filters>
          </ListingToolbar.Leading>
          <ListingToolbar.Trailing>
            <ListingToolbar.ViewToggle onChange={page.setView} value={page.view} />
          </ListingToolbar.Trailing>
        </ListingToolbar>

        <AutomationJobsCatalog
          errorMessage={page.errorMessage}
          hasActiveFilters={page.hasActiveFilters}
          isLoading={page.isLoading}
          jobs={page.jobs}
          onClearFilters={page.clearFilters}
          onCreate={page.handleCreate}
          onRun={page.onRunJob}
          pagination={{
            hasNextPage: page.hasNextPage,
            isFetchingNextPage: page.isFetchingNextPage,
            onLoadMore: page.loadMore,
          }}
          runDisabled={page.runDisabled}
          runPendingIds={page.runPendingIds}
          view={page.view}
        />
      </ListingPage>

      <AutomationEditorDialog {...page.editorDialogProps} />
    </>
  );
}
