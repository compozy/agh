import { createFileRoute } from "@tanstack/react-router";

import {
  MarketplaceKindPage,
  validateMarketplaceKindSearch,
  type MarketplaceKindSearch,
} from "@/systems/marketplace";
import type { TopbarRouteContext } from "@/types/topbar";

export type { MarketplaceKindSearch };

const MARKETPLACE_EXTENSIONS_TOPBAR_CONTEXT: { topbar: TopbarRouteContext } = {
  topbar: { crumb: { label: "Extensions" } },
};

export const Route = createFileRoute("/_app/marketplace/extensions")({
  beforeLoad: (): { topbar: TopbarRouteContext } => MARKETPLACE_EXTENSIONS_TOPBAR_CONTEXT,
  validateSearch: validateMarketplaceKindSearch,
  component: MarketplaceExtensionsRoute,
});

function MarketplaceExtensionsRoute() {
  return <MarketplaceKindPage kind="extension" search={Route.useSearch()} />;
}
