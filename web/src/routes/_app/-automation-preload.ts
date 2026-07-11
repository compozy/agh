import type { QueryClient } from "@tanstack/react-query";

import {
  automationJobsListOptions,
  automationTriggersListOptions,
  type AutomationJobStableFilter,
  type AutomationTriggerStableFilter,
} from "@/systems/automation";
import type { AutomationRouteSearch } from "@/hooks/routes/use-automation-page";

import { resolveActiveWorkspaceId, settleRouteQueries } from "./-route-preload";

async function resolveListScope(
  queryClient: QueryClient,
  search: AutomationRouteSearch
): Promise<Pick<AutomationJobStableFilter, "scope" | "workspace_id">> {
  const scope = search.scope === "all" ? undefined : search.scope;
  if (scope !== "workspace") return { scope };
  return { scope, workspace_id: (await resolveActiveWorkspaceId(queryClient)) ?? undefined };
}

export async function preloadAutomationJobsRoute(
  queryClient: QueryClient,
  search: AutomationRouteSearch
): Promise<void> {
  const scope = await resolveListScope(queryClient, search);
  const filters: AutomationJobStableFilter = {
    ...scope,
    enabled: search.enabled,
    limit: 50,
    loop: search.loop,
    q: search.q,
    source: search.source,
  };
  if (scope.scope === "workspace" && !scope.workspace_id) return;
  await settleRouteQueries([
    queryClient.ensureInfiniteQueryData(automationJobsListOptions(filters)),
  ]);
}

export async function preloadAutomationTriggersRoute(
  queryClient: QueryClient,
  search: AutomationRouteSearch
): Promise<void> {
  const scope = await resolveListScope(queryClient, search);
  const filters: AutomationTriggerStableFilter = {
    ...scope,
    enabled: search.enabled,
    event: search.event,
    limit: 50,
    loop: search.loop,
    q: search.q,
    source: search.source,
  };
  if (scope.scope === "workspace" && !scope.workspace_id) return;
  await settleRouteQueries([
    queryClient.ensureInfiniteQueryData(automationTriggersListOptions(filters)),
  ]);
}
