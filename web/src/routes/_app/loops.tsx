import { AlertCircle, Activity, RefreshCw, Repeat2 } from "lucide-react";
import { Link, Outlet, createFileRoute } from "@tanstack/react-router";

import {
  Button,
  Empty,
  ListingPage,
  ListingToolbar,
  Spinner,
  buttonVariants,
  useTopbarSlot,
} from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  parseLoopCategoryFilter,
  parseLoopKindFilter,
  parseLoopStatusFilter,
  type LoopsRouteSearch,
  useLoopsCatalog,
} from "@/hooks/routes/use-loops-catalog";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import { LoopCatalog, LoopCatalogFilters, type LoopStatusFilter } from "@/systems/loops";
import { preloadLoopsRoute } from "./-loops-preload";

function validateLoopsSearch(search: Record<string, unknown>): LoopsRouteSearch {
  return {
    category: parseLoopCategoryFilter(search.category),
    kind: parseLoopKindFilter(search.kind),
    q: normalizeListingSearchValue(search.q),
    status: parseLoopStatusFilter(search.status),
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/loops")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Loops", icon: Repeat2 },
  }),
  validateSearch: validateLoopsSearch,
  loaderDeps: ({ search }) => ({
    category: search.category,
    kind:
      search.kind === "read-only"
        ? ("read_only" as const)
        : search.kind === "workspace"
          ? ("workspace" as const)
          : undefined,
    limit: 50,
    q: search.q,
    sort: "name" as const,
    status: search.status,
  }),
  loader: ({ context, deps, location }) =>
    location.pathname.split("/").filter(Boolean).length === 1
      ? preloadLoopsRoute(context.queryClient, deps)
      : Promise.resolve(),
  component: LoopsRoute,
});

function LoopsRoute() {
  const page = useLoopsCatalog(Route.useSearch());
  const loops = page.loopsQuery.loops;
  const loopCount = page.loopsQuery.total;
  const workspaceLabel = page.activeWorkspace?.name ?? page.activeWorkspace?.id ?? "workspace";

  useTopbarSlot({
    count: page.hasChildMatch || page.workspaceId === "" ? undefined : loopCount,
    actions:
      page.hasChildMatch || page.workspaceId === "" ? undefined : (
        <div className="flex items-center gap-2" data-testid="loops-topbar-actions">
          <Link
            className={buttonVariants({ size: "sm", variant: "ghost" })}
            data-testid="loops-runs-link"
            to="/loop-runs"
          >
            <Activity aria-hidden="true" className="size-3" />
            Runs
          </Link>
          <Button
            data-testid="loops-refresh"
            onClick={page.handleRefresh}
            size="sm"
            type="button"
            variant="ghost"
          >
            <RefreshCw aria-hidden="true" className="size-3" />
            Refresh
          </Button>
        </div>
      ),
  });

  if (page.hasChildMatch) {
    return <Outlet />;
  }
  if (page.workspaceId === "") {
    return (
      <CatalogState
        description="Select a workspace to browse its Loops."
        testId="loops-no-workspace"
        title="No workspace selected"
      />
    );
  }
  if (page.loopsQuery.isLoading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center" data-testid="loops-loading">
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }
  if (page.loopsQuery.error && loops.length === 0) {
    return (
      <CatalogState
        description={page.loopsQuery.error.message ?? "Failed to load loops"}
        icon={AlertCircle}
        testId="loops-error"
        title="Unable to load loops"
      />
    );
  }

  if (loopCount === 0 && !page.hasActiveFilters) {
    return (
      <CatalogState
        description="No Loop definitions are available in this workspace yet."
        testId="loops-empty"
        title="No loops yet"
      />
    );
  }

  return (
    <ListingPage data-testid="loops-catalog">
      <ListingPage.Head
        count={loopCount}
        countTestId="loops-page-count"
        data-testid="loops-page-head"
        meta={
          <>
            <span>Reusable, guardrailed cycles that pursue a goal until it is verified.</span>
            <ListingPage.MetaDot />
            <span>{workspaceLabel}</span>
          </>
        }
        title="Loops"
      />

      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label="Search loops"
            data-testid="loop-search-input"
            onChange={page.setSearchQuery}
            placeholder="Search loops"
            value={page.searchQuery}
          />
          <ListingToolbar.Filters>
            <LoopCatalogFilters
              categories={Object.keys(page.loopsQuery.facets?.categories ?? {})}
              categoryFilter={page.filter.category}
              kindFilter={page.filter.kind}
              onCategoryFilterChange={page.setCategoryFilter}
              onKindFilterChange={page.setKindFilter}
              onStatusFilterChange={page.setStatusFilter}
              statusFilter={page.filter.status}
              statuses={Object.keys(page.loopsQuery.facets?.statuses ?? {}) as LoopStatusFilter[]}
            />
          </ListingToolbar.Filters>
        </ListingToolbar.Leading>
        <ListingToolbar.Trailing>
          <ListingToolbar.ViewToggle onChange={page.setView} value={page.view} />
        </ListingToolbar.Trailing>
      </ListingToolbar>

      <LoopCatalog
        entries={loops}
        errorMessage={page.loopsQuery.error?.message}
        hasActiveFilters={page.hasActiveFilters}
        hasNextPage={page.loopsQuery.hasNextPage}
        isFetchingNextPage={page.loopsQuery.isFetchingNextPage}
        onClearFilters={page.clearFilters}
        onLoadMore={() => void page.loopsQuery.fetchNextPage()}
        onRun={page.handleRun}
        view={page.view}
      />
    </ListingPage>
  );
}

interface CatalogStateProps {
  title: string;
  description: string;
  testId: string;
  icon?: typeof Repeat2;
}

function CatalogState({ title, description, testId, icon = Repeat2 }: CatalogStateProps) {
  return (
    <div
      className="flex min-h-0 flex-1 items-center justify-center px-6 py-10"
      data-testid={testId}
    >
      <Empty className="max-w-md" description={description} icon={icon} title={title} />
    </div>
  );
}
