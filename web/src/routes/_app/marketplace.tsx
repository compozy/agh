import { Link, Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";
import { RefreshCw } from "lucide-react";

import { Button, RouteNav, Spinner, useTopbarSlot } from "@agh/ui";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import {
  MARKETPLACE_KIND_LABEL,
  MARKETPLACE_KIND_ORDER,
  MarketplaceKindRouteBody,
  MarketplaceLandingRouteBody,
  isMarketplaceRouteKind,
  isMarketplaceViewSort,
  marketplaceApiKindFor,
  marketplaceRouteKindFor,
  useRefreshMarketplaceCatalog,
  type MarketplaceRouteSearch,
} from "@/systems/marketplace";
import type { TopbarRouteContext } from "@/types/topbar";

export type { MarketplaceRouteSearch };

function validateMarketplaceSearch(search: Record<string, unknown>): MarketplaceRouteSearch {
  return {
    kind: isMarketplaceRouteKind(search.kind) ? search.kind : undefined,
    q: normalizeListingSearchValue(search.q),
    sort: isMarketplaceViewSort(search.sort) ? search.sort : undefined,
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/marketplace")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Marketplace", to: "/marketplace" } },
  }),
  validateSearch: validateMarketplaceSearch,
  component: MarketplaceRoute,
});

const MARKETPLACE_NAV_ITEMS = [
  { value: "overview" as const, label: "Overview" },
  ...MARKETPLACE_KIND_ORDER.map(kind => ({
    value: marketplaceRouteKindFor(kind),
    label: kind === "mcp" ? "MCP" : MARKETPLACE_KIND_LABEL[kind],
  })),
];

function MarketplaceRoute() {
  const search = Route.useSearch();
  const hasChildMatch = useChildMatches().length > 0;
  const refresh = useRefreshMarketplaceCatalog();
  const selectedApiKind = search.kind ? marketplaceApiKindFor(search.kind) : undefined;
  const refreshKind = selectedApiKind === "bundle" ? undefined : selectedApiKind;

  useTopbarSlot(
    hasChildMatch
      ? null
      : {
          routeNav: (
            <RouteNav aria-label="Marketplace sections" data-testid="marketplace-kind-navigation">
              {MARKETPLACE_NAV_ITEMS.map(item => (
                <RouteNav.Link
                  key={item.value}
                  render={
                    <Link
                      activeOptions={{ exact: true, includeSearch: true }}
                      search={current => {
                        const { q, view } = current as MarketplaceRouteSearch;
                        return {
                          kind: item.value === "overview" ? undefined : item.value,
                          q,
                          sort: undefined,
                          view,
                        };
                      }}
                      to="/marketplace"
                    />
                  }
                >
                  {item.label}
                </RouteNav.Link>
              ))}
            </RouteNav>
          ),
          actions: (
            <Button
              data-testid="marketplace-refresh"
              disabled={refresh.isPending}
              onClick={() => refresh.mutate(refreshKind)}
              size="sm"
              type="button"
              variant="ghost"
            >
              {refresh.isPending ? (
                <Spinner aria-hidden="true" className="size-3" />
              ) : (
                <RefreshCw aria-hidden="true" className="size-3" />
              )}
              Refresh
            </Button>
          ),
        }
  );

  if (hasChildMatch) return <Outlet />;
  return search.kind ? (
    <MarketplaceKindRouteBody kind={marketplaceApiKindFor(search.kind)} search={search} />
  ) : (
    <MarketplaceLandingRouteBody search={search} />
  );
}
