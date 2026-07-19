import {
  Link,
  Outlet,
  createFileRoute,
  useChildMatches,
  useRouterState,
} from "@tanstack/react-router";
import { RefreshCw } from "lucide-react";

import { Button, RouteNav, Spinner, useTopbarSlot } from "@agh/ui";
import {
  MARKETPLACE_KIND_LABEL,
  MARKETPLACE_KIND_ORDER,
  marketplaceApiKindFor,
  marketplaceRouteKindFor,
  useRefreshMarketplaceCatalog,
  type MarketplaceKind,
} from "@/systems/marketplace";
import type { TopbarRouteContext } from "@/types/topbar";

/**
 * Stable module-level topbar context. `collectCrumbs` folds inherited match
 * contexts by reference identity — a fresh object per `beforeLoad` call makes
 * every marketplace match level contribute a duplicate "Marketplace" crumb
 * (and leaves a stale crumb after navigating to sibling routes).
 */
const MARKETPLACE_TOPBAR_CONTEXT: { topbar: TopbarRouteContext } = {
  topbar: { crumb: { label: "Marketplace", to: "/marketplace" } },
};

export const Route = createFileRoute("/_app/marketplace")({
  beforeLoad: (): { topbar: TopbarRouteContext } => MARKETPLACE_TOPBAR_CONTEXT,
  component: MarketplaceRoute,
});

const MARKETPLACE_NAV_ITEMS = MARKETPLACE_KIND_ORDER.map(kind => ({
  kind,
  routeKind: marketplaceRouteKindFor(kind),
  label: MARKETPLACE_KIND_LABEL[kind],
  to: `/marketplace/${marketplaceRouteKindFor(kind)}` as const,
}));

function MarketplaceRoute() {
  const childMatches = useChildMatches();
  const pathname = useRouterState({ select: state => state.location.pathname });
  const refresh = useRefreshMarketplaceCatalog();

  // Detail (`/$kind/$entryId`) and future activation children own the topbar.
  // Kind pages are direct children and keep RouteNav + Refresh.
  const isDeepChild = childMatches.some(match => {
    const id = match.routeId;
    return id.includes("$") || id.includes("activations");
  });

  const activeApiKind = resolveActiveApiKind(pathname);
  const refreshKind = activeApiKind === "bundle" ? undefined : activeApiKind;

  useTopbarSlot(
    isDeepChild
      ? null
      : {
          routeNav: (
            <RouteNav aria-label="Marketplace sections" data-testid="marketplace-kind-navigation">
              {MARKETPLACE_NAV_ITEMS.map(item => (
                <RouteNav.Link
                  key={item.routeKind}
                  render={
                    <Link activeOptions={{ exact: true, includeSearch: false }} to={item.to} />
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

  return <Outlet />;
}

function resolveActiveApiKind(pathname: string): MarketplaceKind | undefined {
  const segment = pathname.split("/").filter(Boolean)[1];
  if (
    segment === "skills" ||
    segment === "mcps" ||
    segment === "extensions" ||
    segment === "bundles"
  ) {
    return marketplaceApiKindFor(segment);
  }
  return undefined;
}
