import { createFileRoute } from "@tanstack/react-router";

import {
  parseSandboxBackendFilter,
  parseSandboxPersistenceFilter,
  type SandboxRouteSearch,
} from "@/hooks/routes/use-sandbox-page";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import { createOsRouteSync } from "@/systems/os";
import type { TopbarRouteContext } from "@/types/topbar";
import { preloadSandboxRoute } from "./-settings-preload";

function validateSandboxSearch(search: Record<string, unknown>): SandboxRouteSearch {
  return {
    q: normalizeListingSearchValue(search.q),
    backend: parseSandboxBackendFilter(search.backend),
    persistence: parseSandboxPersistenceFilter(search.persistence),
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/sandbox")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Sandbox", to: "/sandbox" } },
  }),
  validateSearch: validateSandboxSearch,
  loader: ({ context }) => preloadSandboxRoute(context.queryClient),
  component: createOsRouteSync("sandbox"),
});
