import { normalizeListingSearchValue } from "@/lib/listing-search";

export type MarketplaceKindScope = "market" | "installed";

export interface MarketplaceKindSearch {
  tab?: "installed";
  q?: string;
}

export function validateMarketplaceKindSearch(
  search: Record<string, unknown>
): MarketplaceKindSearch {
  return {
    tab: search.tab === "installed" ? "installed" : undefined,
    q: normalizeListingSearchValue(search.q),
  };
}

export function marketplaceKindScopeFromSearch(
  search: MarketplaceKindSearch
): MarketplaceKindScope {
  return search.tab === "installed" ? "installed" : "market";
}

export function marketplaceKindPath(routeKind: string): `/marketplace/${string}` {
  return `/marketplace/${routeKind}`;
}
