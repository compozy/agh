import { createFileRoute } from "@tanstack/react-router";

import { type LoopRunsRouteSearch } from "@/hooks/routes/use-loop-runs-route";
import { createOsRouteSync } from "@/systems/os";
import type { TopbarRouteContext } from "@/types/topbar";
import { preloadLoopRunsRoute } from "./-loops-preload";

function validateLoopRunsSearch(search: Record<string, unknown>): LoopRunsRouteSearch {
  const origin =
    search.origin === "catalog" || search.origin === "session" ? search.origin : undefined;
  const originSession =
    typeof search.origin_session === "string" ? search.origin_session.trim() : "";
  return { origin, origin_session: originSession || undefined };
}

export const Route = createFileRoute("/_app/loop-runs")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Runs", to: "/loop-runs" } },
  }),
  loaderDeps: ({ search }) => ({
    origin: search.origin,
    origin_session: search.origin_session,
  }),
  loader: ({ context, deps, location }) =>
    location.pathname.split("/").filter(Boolean).length === 1
      ? preloadLoopRunsRoute(context.queryClient, deps)
      : Promise.resolve(),
  validateSearch: validateLoopRunsSearch,
  component: createOsRouteSync("loops"),
});
