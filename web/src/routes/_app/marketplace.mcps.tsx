import { createFileRoute } from "@tanstack/react-router";

import {
  MarketplaceKindPage,
  validateMarketplaceKindSearch,
  type MarketplaceKindSearch,
} from "@/systems/marketplace";
import type { TopbarRouteContext } from "@/types/topbar";

export type { MarketplaceKindSearch };

const MARKETPLACE_MCPS_TOPBAR_CONTEXT: { topbar: TopbarRouteContext } = {
  topbar: { crumb: { label: "MCPs" } },
};

export const Route = createFileRoute("/_app/marketplace/mcps")({
  beforeLoad: (): { topbar: TopbarRouteContext } => MARKETPLACE_MCPS_TOPBAR_CONTEXT,
  validateSearch: validateMarketplaceKindSearch,
  component: MarketplaceMcpsRoute,
});

function MarketplaceMcpsRoute() {
  return <MarketplaceKindPage kind="mcp" search={Route.useSearch()} />;
}
