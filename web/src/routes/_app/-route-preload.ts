import type { QueryClient } from "@tanstack/react-query";

import {
  selectActiveWorkspace,
  useActiveWorkspaceStore,
  workspacesListOptions,
} from "@/systems/workspace";

/**
 * Route preloads are opportunistic because their pages already own loading and
 * error presentation through TanStack Query. Rejections stay recorded in that
 * query cache instead of being promoted to the route error boundary.
 */
export async function settleRouteQueries(queries: readonly Promise<unknown>[]): Promise<void> {
  await Promise.allSettled(queries);
}

export async function resolveActiveWorkspaceId(queryClient: QueryClient): Promise<string | null> {
  const [hydrationResult, workspacesResult] = await loadWorkspaceSelection(queryClient);

  if (workspacesResult.status === "rejected") {
    return null;
  }

  const selectedWorkspaceId =
    hydrationResult.status === "fulfilled"
      ? useActiveWorkspaceStore.getState().selectedWorkspaceId
      : null;
  return selectActiveWorkspace(workspacesResult.value, selectedWorkspaceId)?.id ?? null;
}

export async function adoptRouteWorkspaceWhenSelectionInvalid(
  queryClient: QueryClient,
  routeWorkspaceId: string
): Promise<void> {
  const [hydrationResult, workspacesResult] = await loadWorkspaceSelection(queryClient);
  if (hydrationResult.status === "rejected" || workspacesResult.status === "rejected") {
    return;
  }

  const workspaces = workspacesResult.value;
  if (!workspaces.some(workspace => workspace.id === routeWorkspaceId)) {
    return;
  }

  const workspaceStore = useActiveWorkspaceStore.getState();
  const selectedWorkspaceIsValid = workspaces.some(
    workspace => workspace.id === workspaceStore.selectedWorkspaceId
  );
  if (!selectedWorkspaceIsValid) {
    workspaceStore.setSelectedWorkspaceId(routeWorkspaceId);
  }
}

export async function selectRouteWorkspaceForNavigation(
  queryClient: QueryClient,
  routeWorkspaceId: string
): Promise<void> {
  const [hydrationResult, workspacesResult] = await loadWorkspaceSelection(queryClient);
  if (hydrationResult.status === "rejected" || workspacesResult.status === "rejected") {
    return;
  }
  if (!workspacesResult.value.some(workspace => workspace.id === routeWorkspaceId)) {
    return;
  }
  useActiveWorkspaceStore.getState().setSelectedWorkspaceId(routeWorkspaceId);
}

function loadWorkspaceSelection(queryClient: QueryClient) {
  const hydration = useActiveWorkspaceStore.persist.hasHydrated()
    ? Promise.resolve()
    : useActiveWorkspaceStore.persist.rehydrate();

  return Promise.allSettled([
    hydration,
    queryClient.ensureQueryData(workspacesListOptions()),
  ] as const);
}
