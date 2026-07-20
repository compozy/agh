import { createFileRoute } from "@tanstack/react-router";

import {
  automationListLoopFilter,
  type AutomationRouteSearch,
} from "@/hooks/routes/use-automation-page";
import { normalizeListingSearchValue, parseListingView } from "@/lib/listing-search";
import {
  parseAutomationEnabled,
  parseAutomationScope,
  parseAutomationSource,
} from "@/systems/automation";
import { createOsRouteSync } from "@/systems/os";
import type { TopbarRouteContext } from "@/types/topbar";
import { preloadAutomationTriggersRoute } from "./-automation-preload";

function validateTriggersSearch(search: Record<string, unknown>): AutomationRouteSearch {
  return {
    create: search.create === "loop" ? "loop" : undefined,
    enabled: parseAutomationEnabled(search.enabled),
    event: normalizeListingSearchValue(search.event),
    loop: normalizeListingSearchValue(search.loop),
    q: normalizeListingSearchValue(search.q),
    scope: parseAutomationScope(search.scope),
    source: parseAutomationSource(search.source),
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/triggers")({
  validateSearch: validateTriggersSearch,
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Triggers", to: "/triggers" } },
  }),
  loaderDeps: ({ search }) => ({
    enabled: search.enabled,
    event: search.event,
    loop: automationListLoopFilter(search),
    q: search.q,
    scope: search.scope,
    source: search.source,
  }),
  loader: ({ context, deps }) => preloadAutomationTriggersRoute(context.queryClient, deps),
  component: createOsRouteSync("triggers"),
});
