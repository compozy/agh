import { Link, useChildMatches, useNavigate } from "@tanstack/react-router";

import { Button, RouteNav, useTopbarSlot, type ListingViewMode } from "@agh/ui";
import { parseListingView } from "@/lib/listing-search";

export interface ExtensionsRouteSearch {
  tab?: "bundles";
  view?: ListingViewMode;
}

export function validateExtensionsSearch(search: Record<string, unknown>): ExtensionsRouteSearch {
  return {
    tab: search.tab === "bundles" ? "bundles" : undefined,
    view: parseListingView(search.view),
  };
}

export function useExtensionsRoutePage(search: ExtensionsRouteSearch) {
  const navigate = useNavigate({ from: "/extensions" });
  const child = useChildMatches().length > 0;
  const tab: "extensions" | "bundles" = search.tab ?? "extensions";
  const view: ListingViewMode = search.view ?? "rows";

  const updateSearch = (next: Partial<ExtensionsRouteSearch>) => {
    void navigate({
      search: current => {
        const merged = { ...(current as ExtensionsRouteSearch), ...next };
        return {
          tab: merged.tab,
          view: merged.view === "rows" ? undefined : merged.view,
        };
      },
      to: "/extensions",
    });
  };

  // Publish null while a child route is mounted: a non-null slot from this
  // parent would steal the detail route's publish (layout effects run child
  // first, parent last in the same commit).
  useTopbarSlot(
    child
      ? null
      : {
          routeNav: (
            <RouteNav aria-label="Extensions sections">
              <RouteNav.Link
                render={
                  <Link
                    activeOptions={{ exact: true, includeSearch: true }}
                    search={current => {
                      const { view } = current as ExtensionsRouteSearch;
                      return {
                        tab: undefined,
                        view: view === "rows" ? undefined : view,
                      };
                    }}
                    to="/extensions"
                  />
                }
              >
                Extensions
              </RouteNav.Link>
              <RouteNav.Link
                render={
                  <Link
                    activeOptions={{ exact: true, includeSearch: true }}
                    search={current => {
                      const { view } = current as ExtensionsRouteSearch;
                      return {
                        tab: "bundles" as const,
                        view: view === "rows" ? undefined : view,
                      };
                    }}
                    to="/extensions"
                  />
                }
              >
                Bundles
              </RouteNav.Link>
            </RouteNav>
          ),
          actions: (
            <Button
              render={
                <Link
                  search={{ kind: tab === "bundles" ? "bundles" : "extensions" }}
                  to="/marketplace"
                />
              }
              nativeButton={false}
              size="sm"
              variant="ghost"
            >
              Browse marketplace
            </Button>
          ),
        }
  );

  const setView = (nextView: ListingViewMode) => {
    updateSearch({ view: nextView === "rows" ? undefined : nextView });
  };

  return { child, setView, tab, view };
}
