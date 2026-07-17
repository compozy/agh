import { Outlet, createFileRoute, useChildMatches, useNavigate } from "@tanstack/react-router";
import { RefreshCw, Store } from "lucide-react";

import { Button, PillGroup, Spinner, useTopbarSlot } from "@agh/ui";
import { normalizeListingSearchValue } from "@/lib/listing-search";
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
  };
}

export const Route = createFileRoute("/_app/marketplace")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Marketplace", icon: Store },
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
  const navigate = useNavigate({ from: "/marketplace" });
  const hasChildMatch = useChildMatches().length > 0;
  const refresh = useRefreshMarketplaceCatalog();
  const activeNav = search.kind ?? "overview";
  const selectedApiKind = search.kind ? marketplaceApiKindFor(search.kind) : undefined;
  const refreshKind = selectedApiKind === "bundle" ? undefined : selectedApiKind;

  useTopbarSlot(
    hasChildMatch
      ? null
      : {
          tabs: (
            <PillGroup
              aria-label="Marketplace section"
              data-testid="marketplace-kind-navigation"
              items={MARKETPLACE_NAV_ITEMS}
              onChange={next => {
                void navigate({
                  search: current => ({
                    ...(current as MarketplaceRouteSearch),
                    kind: next === "overview" ? undefined : next,
                    sort: undefined,
                  }),
                  to: "/marketplace",
                });
              }}
              size="md"
              value={activeNav}
            />
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
