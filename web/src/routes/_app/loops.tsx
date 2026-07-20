import { createFileRoute } from "@tanstack/react-router";

import {
  parseLoopCategoryFilter,
  parseLoopKindFilter,
  parseLoopStatusFilter,
  type LoopsRouteSearch,
} from "@/hooks/routes/use-loops-catalog";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import { createOsRouteSync } from "@/systems/os";
import type { TopbarRouteContext } from "@/types/topbar";
import { preloadLoopsRoute } from "./-loops-preload";

function validateLoopsSearch(search: Record<string, unknown>): LoopsRouteSearch {
  return {
    category: parseLoopCategoryFilter(search.category),
    kind: parseLoopKindFilter(search.kind),
    q: normalizeListingSearchValue(search.q),
    status: parseLoopStatusFilter(search.status),
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/loops")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Loops", to: "/loops" } },
  }),
  validateSearch: validateLoopsSearch,
  loaderDeps: ({ search }) => ({
    category: search.category,
    kind:
      search.kind === "read-only"
        ? ("read_only" as const)
        : search.kind === "workspace"
          ? ("workspace" as const)
          : undefined,
    limit: 50,
    q: search.q,
    sort: "name" as const,
    status: search.status,
  }),
  loader: ({ context, deps, location }) =>
    location.pathname.split("/").filter(Boolean).length === 1
      ? preloadLoopsRoute(context.queryClient, deps)
      : Promise.resolve(),
  component: createOsRouteSync("loops"),
});
