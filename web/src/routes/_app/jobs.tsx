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
import { preloadAutomationJobsRoute } from "./-automation-preload";

function validateJobsSearch(search: Record<string, unknown>): AutomationRouteSearch {
  return {
    create: search.create === "loop" ? "loop" : undefined,
    enabled: parseAutomationEnabled(search.enabled),
    loop: normalizeListingSearchValue(search.loop),
    q: normalizeListingSearchValue(search.q),
    scope: parseAutomationScope(search.scope),
    source: parseAutomationSource(search.source),
    view: parseListingView(search.view),
  };
}

export const Route = createFileRoute("/_app/jobs")({
  validateSearch: validateJobsSearch,
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { crumb: { label: "Jobs", to: "/jobs" } },
  }),
  loaderDeps: ({ search }) => ({
    enabled: search.enabled,
    loop: automationListLoopFilter(search),
    q: search.q,
    scope: search.scope,
    source: search.source,
  }),
  loader: ({ context, deps }) => preloadAutomationJobsRoute(context.queryClient, deps),
  component: createOsRouteSync("jobs"),
});
