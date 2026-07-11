import { Zap } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@/types/topbar";
import { AutomationOperationsPage } from "@/systems/automation";
import type { AutomationScopeFilter, AutomationSource } from "@/systems/automation";
import {
  useAutomationTriggersPage,
  type AutomationRouteSearch,
} from "@/hooks/routes/use-automation-page";
import { preloadAutomationTriggersRoute } from "./-automation-preload";

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
    event: parseText(search.event),
    loop: parseText(search.loop),
    q: parseText(search.q),
    scope: parseScope(search.scope),
    source: parseSource(search.source),
  };
}

export const Route = createFileRoute("/_app/triggers")({
  validateSearch: parseCreateSearch,
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Triggers", icon: Zap },
  }),
  loaderDeps: ({ search }) => ({
    enabled: search.enabled,
    event: search.event,
    loop: search.loop,
    q: search.q,
    scope: search.scope,
    source: search.source,
  }),
  loader: ({ context, deps }) => preloadAutomationTriggersRoute(context.queryClient, deps),
  component: TriggersPage,
});

function TriggersPage() {
  const search = Route.useSearch();
  const page = useAutomationTriggersPage(
    search.create === "loop" && search.loop ? { loop: search.loop } : {},
    search
  );

  return (
    <AutomationOperationsPage
      createButtonTestId="create-trigger-btn"
      createLabel="Trigger"
      page={page}
      title="Triggers"
      titlePrefix="triggers"
    />
  );
}
