import type { QueryClient } from "@tanstack/react-query";

import {
  skillContentOptions,
  skillDetailOptions,
  skillMarketplaceSearchOptions,
  skillShadowsOptions,
  skillsListOptions,
} from "@/systems/skill";

import { resolveActiveWorkspaceId, settleRouteQueries } from "./-route-preload";

interface SkillsPreloadSearch {
  q?: string;
  tab?: "installed" | "marketplace";
}

export async function preloadSkillsRoute(
  queryClient: QueryClient,
  search: SkillsPreloadSearch
): Promise<void> {
  const workspaceId = await resolveActiveWorkspaceId(queryClient);
  if (!workspaceId) {
    return;
  }

  const queries: Promise<unknown>[] = [queryClient.ensureQueryData(skillsListOptions(workspaceId))];
  if (search.tab === "marketplace" && search.q?.trim()) {
    queries.push(queryClient.ensureQueryData(skillMarketplaceSearchOptions(search.q)));
  }
  await settleRouteQueries(queries);
}

interface SkillDetailPreloadSearch {
  content?: string;
}

export async function preloadSkillDetailRoute(
  queryClient: QueryClient,
  name: string,
  search: SkillDetailPreloadSearch
): Promise<void> {
  const workspaceId = await resolveActiveWorkspaceId(queryClient);
  if (!workspaceId) {
    return;
  }

  const queries: Promise<unknown>[] = [
    queryClient.ensureQueryData(skillsListOptions(workspaceId)),
    queryClient.ensureQueryData(skillDetailOptions(name, workspaceId)),
    queryClient.ensureQueryData(skillShadowsOptions(name, workspaceId)),
  ];
  if (search.content === name) {
    queries.push(queryClient.ensureQueryData(skillContentOptions(name, workspaceId, true)));
  }
  await settleRouteQueries(queries);
}
