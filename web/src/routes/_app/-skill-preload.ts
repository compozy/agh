import type { QueryClient } from "@tanstack/react-query";

import {
  skillContentOptions,
  skillDetailOptions,
  skillShadowsOptions,
  skillsListOptions,
} from "@/systems/skill";

import { resolveActiveWorkspaceId, settleRouteQueries } from "./-route-preload";

export async function preloadSkillsRoute(queryClient: QueryClient): Promise<void> {
  const workspaceId = await resolveActiveWorkspaceId(queryClient);
  if (!workspaceId) {
    return;
  }

  await settleRouteQueries([queryClient.ensureQueryData(skillsListOptions(workspaceId))]);
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
