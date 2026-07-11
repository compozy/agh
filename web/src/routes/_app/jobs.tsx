import { Clock3 } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { AutomationOperationsPage } from "@/systems/automation";
import type { AutomationScopeFilter, AutomationSource } from "@/systems/automation";
import {
  useAutomationJobsPage,
  type AutomationRouteSearch,
} from "@/hooks/routes/use-automation-page";
import { preloadAutomationJobsRoute } from "./-automation-preload";

/** Optional deep-link that pre-targets the create sheet at a Loop (§9.14 CTA). */
function parseText(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined;
  const normalized = value.trim();
  return normalized === "" ? undefined : normalized;
}

function parseScope(value: unknown): AutomationScopeFilter | undefined {
  return value === "all" || value === "global" || value === "workspace" ? value : undefined;
}

function parseSource(value: unknown): AutomationSource | undefined {
  return value === "config" || value === "package" || value === "dynamic" ? value : undefined;
}

function parseEnabled(value: unknown): boolean | undefined {
  if (value === true || value === "true") return true;
  if (value === false || value === "false") return false;
  return undefined;
}

function parseCreateSearch(search: Record<string, unknown>): AutomationRouteSearch {
  return {
    create: search.create === "loop" ? "loop" : undefined,
    enabled: parseEnabled(search.enabled),
    loop: parseText(search.loop),
    q: parseText(search.q),
    scope: parseScope(search.scope),
    source: parseSource(search.source),
  };
}

export const Route = createFileRoute("/_app/jobs")({
  validateSearch: parseCreateSearch,
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Jobs", icon: Clock3 },
  }),
  loaderDeps: ({ search }) => ({
    enabled: search.enabled,
    loop: search.loop,
    q: search.q,
    scope: search.scope,
    source: search.source,
  }),
  loader: ({ context, deps }) => preloadAutomationJobsRoute(context.queryClient, deps),
  component: JobsPage,
});

function JobsPage() {
  const search = Route.useSearch();
  const page = useAutomationJobsPage(
    search.create === "loop" && search.loop ? { loop: search.loop } : {},
    search
  );

  return (
    <AutomationOperationsPage
      createButtonTestId="create-job-btn"
      createLabel="Job"
      page={page}
      title="Jobs"
      titlePrefix="jobs"
    />
  );
}
