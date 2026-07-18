import { useNavigate } from "@tanstack/react-router";

import type { ListingViewMode } from "@agh/ui";

import { useActiveWorkspace } from "@/systems/workspace";

import { useMarketplaceKind, useMarketplaceSearch } from "../hooks/use-marketplace";
import type { MarketplaceKind, MarketplaceRouteKind } from "../types";
import { MarketplaceKindView } from "./marketplace-kind-view";
import { MarketplaceLanding } from "./marketplace-landing";
import type { MarketplaceViewSort } from "./marketplace-ui";
import { useMarketplaceActionController } from "./use-marketplace-action-controller";

export interface MarketplaceRouteSearch {
  kind?: MarketplaceRouteKind;
  q?: string;
  sort?: MarketplaceViewSort;
  view?: ListingViewMode;
}

export function MarketplaceLandingRouteBody({ search }: { search: MarketplaceRouteSearch }) {
  const navigate = useNavigate({ from: "/marketplace" });
  const { activeWorkspaceId } = useActiveWorkspace();
  const actions = useMarketplaceActionController(activeWorkspaceId);
  const query = useMarketplaceSearch({
    limit: 3,
    q: search.q,
    workspaceId: activeWorkspaceId,
  });
  const updateSearch = (next: Partial<MarketplaceRouteSearch>) => {
    void navigate({
      search: current => ({ ...(current as MarketplaceRouteSearch), ...next }),
      to: "/marketplace",
    });
  };
  const view: ListingViewMode = search.view ?? "rows";
  return (
    <>
      <MarketplaceLanding
        data={query.data}
        error={query.error}
        isFetching={query.isFetching}
        isEntryPending={actions.isEntryPending}
        isLoading={query.isLoading}
        onAction={actions.handleAction}
        onClearSearch={() => updateSearch({ q: undefined })}
        onRetry={() => void query.refetch()}
        onSearchChange={q => updateSearch({ q: q || undefined })}
        onViewChange={nextView =>
          updateSearch({ view: nextView === "rows" ? undefined : nextView })
        }
        query={search.q ?? ""}
        view={view}
      />
      {actions.dialogs}
    </>
  );
}

export function MarketplaceKindRouteBody({
  kind,
  search,
}: {
  kind: MarketplaceKind;
  search: MarketplaceRouteSearch;
}) {
  const navigate = useNavigate({ from: "/marketplace" });
  const { activeWorkspaceId } = useActiveWorkspace();
  const actions = useMarketplaceActionController(activeWorkspaceId);
  const query = useMarketplaceKind({
    kind,
    limit: 100,
    q: search.q,
    workspaceId: activeWorkspaceId,
  });
  const updateSearch = (next: Partial<MarketplaceRouteSearch>) => {
    void navigate({
      search: current => ({ ...(current as MarketplaceRouteSearch), ...next }),
      to: "/marketplace",
    });
  };
  const view: ListingViewMode = search.view ?? "rows";
  return (
    <>
      <MarketplaceKindView
        data={query.data}
        error={query.error}
        isEntryPending={actions.isEntryPending}
        isLoading={query.isLoading}
        kind={kind}
        onAction={actions.handleAction}
        onClearSearch={() => updateSearch({ q: undefined })}
        onRetry={() => void query.refetch()}
        onSearchChange={q => updateSearch({ q: q || undefined })}
        onSortChange={sort => updateSearch({ sort: sort === "relevance" ? undefined : sort })}
        onViewChange={nextView =>
          updateSearch({ view: nextView === "rows" ? undefined : nextView })
        }
        query={search.q ?? ""}
        sort={search.sort ?? "relevance"}
        view={view}
      />
      {actions.dialogs}
    </>
  );
}
