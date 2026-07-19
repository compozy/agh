import { AlertCircle, Plus, RefreshCw, Waypoints } from "lucide-react";
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
  parseBridgePlatformFilter,
  parseBridgeScopeFilter,
  parseBridgeStatusFilter,
  type BridgesRouteSearch,
  useBridgesPage,
} from "@/hooks/routes/use-bridges-page";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import {
  BridgeCreateDialog,
  BridgeEmptyState,
  BridgeListFilters,
  BridgeListPanel,
} from "@/systems/bridges";
import { useActiveWorkspace } from "@/systems/workspace";
import { preloadBridgesRoute } from "./-bridges-preload";

function validateBridgesSearch(search: Record<string, unknown>): BridgesRouteSearch {
  return {
    platform: parseBridgePlatformFilter(search.platform),
    q: normalizeListingSearchValue(search.q),
    scope: parseBridgeScopeFilter(search.scope),
    status: parseBridgeStatusFilter(search.status),
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/bridges")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Bridges", to: "/bridges" } },
  }),
  validateSearch: validateBridgesSearch,
  loaderDeps: ({ search }) => ({
    platform: search.platform,
    q: search.q,
    scope: search.scope ?? "all",
    status: search.status,
  }),
  loader: ({ context, deps }) => preloadBridgesRoute(context.queryClient, deps),
  component: BridgesPage,
});

function BridgesPage() {
  const page = useBridgesPage(Route.useSearch());
  const { activeWorkspace } = useActiveWorkspace();

  // Publish null while a child route is mounted: a non-null slot from this
  // parent would steal the detail route's publish (layout effects run child
  // first, parent last in the same commit).
  useTopbarSlot(
    page.hasChildMatch
      ? null
      : {
          actions: (
            <div className="flex items-center gap-2" data-testid="bridges-topbar-actions">
              <Button
                data-testid="bridges-refresh"
                onClick={page.handleRefresh}
                size="sm"
                type="button"
                variant="ghost"
              >
                <RefreshCw aria-hidden="true" className="size-3" />
                Refresh
              </Button>
              <Button
                data-testid="create-bridge-btn"
                disabled={!page.canCreateBridge}
                onClick={page.openCreateDialog}
                size="sm"
                type="button"
              >
                <Plus aria-hidden="true" className="size-3" />
                Bridge
              </Button>
            </div>
          ),
        }
  );

  if (page.hasChildMatch) {
    return <Outlet />;
  }

  if (page.isInitialLoading) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center"
        data-testid="bridges-loading"
      >
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  }

  if (page.fatalError) {
    return (
      <div
        className="flex min-h-0 flex-1 items-center justify-center py-10"
        data-testid="bridges-error"
      >
        <Empty
          className="max-w-md"
          description={page.fatalError.message ?? "Failed to load bridges"}
          icon={AlertCircle}
          title="Unable to load bridges"
        />
      </div>
    );
  }

  if (page.totalBridgeCount === 0 && !page.hasActiveFilters) {
    return (
      <>
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="bridges-shell">
          <BridgeEmptyState onCreate={page.openCreateDialog} providers={page.providers} />
        </div>
        <BridgeCreateDialog {...page.createDialogProps} />
      </>
    );
  }

  const workspaceLabel = activeWorkspace?.name ?? activeWorkspace?.id ?? "workspace";

  return (
    <>
      <ListingPage
        banner={
          page.backgroundError ? (
            <div className="border-b border-line px-9 py-3">
              <Alert data-testid="bridges-background-error" variant="warning">
                <AlertCircle aria-hidden="true" className="size-4" />
                <AlertTitle>Showing cached bridges</AlertTitle>
                <AlertDescription>
                  {page.backgroundError.message ??
                    "The latest bridge refresh failed. Existing data remains available."}
                </AlertDescription>
              </Alert>
            </div>
          ) : null
        }
        data-testid="bridges-shell"
      >
        <PageHead
          count={page.totalBridgeCount}
          countTestId="bridges-page-count"
          data-testid="bridges-page-head"
          icon={Waypoints}
          meta={
            <>
              <span>Messaging bridges that connect AGH to external platforms.</span>
              <PageHead.MetaDot />
              <span>{workspaceLabel}</span>
            </>
          }
          title="Bridges"
        />

        <ListingToolbar>
          <ListingToolbar.Leading>
            <ListingToolbar.Search
              aria-label="Search bridges"
              data-testid="bridge-search-input"
              onChange={page.setSearchQuery}
              placeholder="Search bridges"
              value={page.searchQuery}
            />
            <ListingToolbar.Filters>
              <BridgeListFilters
                onPlatformFilterChange={page.setPlatformFilter}
                onScopeFilterChange={page.setScopeFilter}
                onStatusFilterChange={page.setStatusFilter}
                platformFilter={page.platformFilter}
                platforms={page.platforms}
                scopeFilter={page.scopeFilter}
                statusFilter={page.statusFilter}
                statuses={page.statuses}
              />
            </ListingToolbar.Filters>
          </ListingToolbar.Leading>
          <ListingToolbar.Trailing>
            <ListingToolbar.ViewToggle onChange={page.setView} value={page.view} />
          </ListingToolbar.Trailing>
        </ListingToolbar>

        <div data-testid="bridge-list-panel">
          <BridgeListPanel
            bridgeHealth={page.bridgeHealth}
            bridges={page.bridges}
            errorMessage={page.backgroundError?.message}
            emptyState={page.hasActiveFilters ? "filtered" : "default"}
            paginationStatus={
              page.isFetchingNextPage ? "loading" : page.hasNextPage ? "available" : undefined
            }
            onClearFilters={page.clearFilters}
            onLoadMore={() => void page.loadMore()}
            view={page.view}
          />
        </div>
      </ListingPage>

      <BridgeCreateDialog {...page.createDialogProps} />
    </>
  );
}
