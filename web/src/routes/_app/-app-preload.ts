import type { QueryClient } from "@tanstack/react-query";

import { agentCatalogOptions, agentDetailOptions, agentsListOptions } from "@/systems/agent";
import { onboardingStatusOptions } from "@/systems/onboarding";
import { sessionsListOptions } from "@/systems/session";
import { workspaceDetailOptions } from "@/systems/workspace";

import { resolveActiveWorkspaceId, settleRouteQueries } from "./-route-preload";

export async function preloadAppRoute(queryClient: QueryClient): Promise<void> {
  const [[onboardingResult], workspaceId] = await Promise.all([
    Promise.allSettled([queryClient.ensureQueryData(onboardingStatusOptions())] as const),
    resolveActiveWorkspaceId(queryClient),
  ]);

  if (
    onboardingResult.status === "rejected" ||
    onboardingResult.value.completed !== true ||
    !workspaceId
  ) {
    return;
  }

  await settleRouteQueries([
    queryClient.ensureQueryData(agentsListOptions(workspaceId)),
    queryClient.ensureInfiniteQueryData(agentCatalogOptions(workspaceId, { limit: 1 })),
    queryClient.ensureQueryData(workspaceDetailOptions(workspaceId)),
    queryClient.ensureInfiniteQueryData(
      sessionsListOptions({ workspace: workspaceId, state: "active", limit: 1 })
    ),
  ]);
}

export async function preloadHomeRoute(queryClient: QueryClient): Promise<void> {
  const workspaceId = await resolveActiveWorkspaceId(queryClient);
  if (!workspaceId) {
    return;
  }

  await settleRouteQueries([
    queryClient.ensureQueryData(agentsListOptions(workspaceId)),
    queryClient.ensureInfiniteQueryData(
      sessionsListOptions({ workspace: workspaceId, state: "active", type: "user", limit: 1 })
    ),
  ]);
}

export async function preloadAgentDetailRoute(
  queryClient: QueryClient,
  name: string
): Promise<void> {
  const workspaceId = await resolveActiveWorkspaceId(queryClient);
  if (!workspaceId) {
    return;
  }

  await settleRouteQueries([
    queryClient.ensureQueryData(agentDetailOptions(name, workspaceId)),
    queryClient.ensureInfiniteQueryData(
      sessionsListOptions({
        workspace: workspaceId,
        agent: name,
        type: "user",
        sort: "last_activity",
      })
    ),
    queryClient.ensureInfiniteQueryData(
      sessionsListOptions({
        workspace: workspaceId,
        agent: name,
        state: "active",
        type: "user",
        limit: 1,
      })
    ),
    queryClient.ensureInfiniteQueryData(
      sessionsListOptions({
        workspace: workspaceId,
        agent: name,
        type: "user",
        resumable: true,
        limit: 1,
      })
    ),
  ]);
}
