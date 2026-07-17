import { useNavigate } from "@tanstack/react-router";

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
  const setSearch = (q: string) => {
    void navigate({
      search: current => ({ ...(current as MarketplaceRouteSearch), q: q || undefined }),
      to: "/marketplace",
    });
  };
  return (
    <>
      <MarketplaceLanding
        data={query.data}
        error={query.error}
        isFetching={query.isFetching}
        isEntryPending={actions.isEntryPending}
        isLoading={query.isLoading}
        onAction={actions.handleAction}
        onClearSearch={() => setSearch("")}
        onRetry={() => void query.refetch()}
        onSearchChange={setSearch}
        query={search.q ?? ""}
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
        query={search.q ?? ""}
        sort={search.sort ?? "relevance"}
      />
      {actions.dialogs}
    </>
  );
}
