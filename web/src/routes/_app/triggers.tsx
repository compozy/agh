import { AlertCircle, Plus, Zap } from "lucide-react";
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
  AutomationListFilters,
  AutomationTriggersCatalog,
  parseAutomationEnabled,
  parseAutomationScope,
  parseAutomationSource,
} from "@/systems/automation";
import {
  automationListLoopFilter,
  useAutomationTriggersPage,
  type AutomationRouteSearch,
} from "@/hooks/routes/use-automation-page";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import { useActiveWorkspace } from "@/systems/workspace";
import { preloadAutomationTriggersRoute } from "./-automation-preload";

function validateTriggersSearch(search: Record<string, unknown>): AutomationRouteSearch {
  return {
    create: search.create === "loop" ? "loop" : undefined,
    enabled: parseAutomationEnabled(search.enabled),
    event: normalizeListingSearchValue(search.event),
    loop: normalizeListingSearchValue(search.loop),
    q: normalizeListingSearchValue(search.q),
    scope: parseAutomationScope(search.scope),
    source: parseAutomationSource(search.source),
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/triggers")({
  validateSearch: validateTriggersSearch,
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Triggers", to: "/triggers" } },
  }),
  loaderDeps: ({ search }) => ({
    enabled: search.enabled,
    event: search.event,
    loop: automationListLoopFilter(search),
    q: search.q,
    scope: search.scope,
    source: search.source,
  }),
  loader: ({ context, deps }) => preloadAutomationTriggersRoute(context.queryClient, deps),
  component: TriggersPage,
});

function TriggersPage() {
  const search = Route.useSearch();
  const page = useAutomationTriggersPage(
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
              data-testid="create-trigger-btn"
              onClick={page.handleCreate}
              size="sm"
              type="button"
            >
              <Plus aria-hidden="true" className="size-3" />
              Trigger
            </Button>
          ),
        }
  );

  if (page.hasChildMatch) {
    return <Outlet />;
  }

  if (page.isLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="triggers-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.error) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center py-10"
        data-testid="triggers-error"
      >
        <Empty
          className="max-w-md"
          description={page.error.message ?? "Failed to load triggers"}
          icon={AlertCircle}
          title="Unable to load triggers"
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
              <Alert data-testid="triggers-runtime-alert" variant="warning">
                <AlertCircle aria-hidden="true" className="size-4" />
                <AlertTitle>Automation runtime unavailable</AlertTitle>
                <AlertDescription>{page.runtimeUnavailableMessage}</AlertDescription>
              </Alert>
            </div>
          ) : null
        }
        data-testid="triggers-shell"
      >
        <PageHead
          count={page.total}
          countTestId="triggers-count"
          data-testid="triggers-page-head"
          icon={Zap}
          meta={
            <>
              <span>Runtime events that run agents, tasks, or Loops when they match.</span>
              <PageHead.MetaDot />
              <span>{workspaceLabel}</span>
            </>
          }
          title="Triggers"
        />

        <ListingToolbar>
          <ListingToolbar.Leading>
            <ListingToolbar.Search
              aria-label="Search triggers"
              data-testid="automation-search-input"
              onChange={page.setSearchQuery}
              placeholder="Search triggers"
              value={page.searchQuery}
            />
            <ListingToolbar.Filters>
              <AutomationListFilters
                enabledFilter={page.enabledFilter}
                eventFilter={page.eventFilter}
                kind="triggers"
                onEnabledFilterChange={page.setEnabledFilter}
                onEventFilterChange={page.setEventFilter}
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

        <AutomationTriggersCatalog
          errorMessage={page.errorMessage}
          hasActiveFilters={page.hasActiveFilters}
          isLoading={page.isLoading}
          onClearFilters={page.clearFilters}
          onCreate={page.handleCreate}
          pagination={{
            hasNextPage: page.hasNextPage,
            isFetchingNextPage: page.isFetchingNextPage,
            onLoadMore: page.loadMore,
          }}
          triggers={page.triggers}
          view={page.view}
        />
      </ListingPage>

      <AutomationEditorDialog {...page.editorDialogProps} />
    </>
  );
}
