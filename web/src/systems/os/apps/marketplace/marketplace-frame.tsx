import { Link } from "@tanstack/react-router";
import { RefreshCw } from "lucide-react";
import type { ReactNode } from "react";

import { Button, RouteNav, Spinner, useTopbarSlot } from "@agh/ui";
import {
  MARKETPLACE_KIND_LABEL,
  MARKETPLACE_KIND_ORDER,
  marketplaceApiKindFor,
  marketplaceRouteKindFor,
  useRefreshMarketplaceCatalog,
  type MarketplaceKind,
} from "@/systems/marketplace";

const MARKETPLACE_NAV_ITEMS = MARKETPLACE_KIND_ORDER.map(kind => ({
  kind,
  routeKind: marketplaceRouteKindFor(kind),
  label: MARKETPLACE_KIND_LABEL[kind],
  to: `/marketplace/${marketplaceRouteKindFor(kind)}` as const,
}));

export function MarketplaceFrame({
  children,
  deep,
  pathname,
}: {
  children: ReactNode;
  deep: boolean;
  pathname: string;
}) {
  const refresh = useRefreshMarketplaceCatalog();
  const activeApiKind = resolveActiveApiKind(pathname);
  const refreshKind = activeApiKind === "bundle" ? undefined : activeApiKind;

  useTopbarSlot(
    deep
      ? null
      : {
          routeNav: (
            <RouteNav aria-label="Marketplace sections" data-testid="marketplace-kind-navigation">
              {MARKETPLACE_NAV_ITEMS.map(item => (
                <RouteNav.Link
                  aria-current={item.kind === activeApiKind ? "page" : undefined}
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

  return children;
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
