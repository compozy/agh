import { Outlet, createFileRoute, useChildMatches } from "@tanstack/react-router";

import {
  MarketplaceKindPage,
  validateMarketplaceKindSearch,
  type MarketplaceKindSearch,
} from "@/systems/marketplace";
import type { TopbarRouteContext } from "@/types/topbar";

export type { MarketplaceKindSearch };

const MARKETPLACE_BUNDLES_TOPBAR_CONTEXT: { topbar: TopbarRouteContext } = {
  topbar: { crumb: { label: "Bundles" } },
};

export const Route = createFileRoute("/_app/marketplace/bundles")({
  beforeLoad: (): { topbar: TopbarRouteContext } => MARKETPLACE_BUNDLES_TOPBAR_CONTEXT,
  validateSearch: validateMarketplaceKindSearch,
  component: MarketplaceBundlesRoute,
});

function MarketplaceBundlesRoute() {
  const hasChildMatch = useChildMatches().length > 0;
  const search = Route.useSearch();
  if (hasChildMatch) return <Outlet />;
  return <MarketplaceKindPage kind="bundle" search={search} />;
}
