import { createFileRoute } from "@tanstack/react-router";

import {
  parseBridgePlatformFilter,
  parseBridgeScopeFilter,
  parseBridgeStatusFilter,
  type BridgesRouteSearch,
} from "@/hooks/routes/use-bridges-page";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import { createOsRouteSync } from "@/systems/os";
import type { TopbarRouteContext } from "@/types/topbar";
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
  component: createOsRouteSync("bridges"),
});
