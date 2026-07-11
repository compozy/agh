import type { QueryClient } from "@tanstack/react-query";

import { statusOptions } from "@/systems/status";
import { taskInboxOptions, tasksListOptions } from "@/systems/tasks";
import { workspacesListOptions } from "@/systems/workspace";
import { defaultTaskCatalogFilter } from "@/hooks/routes/task-catalog-route-filter";
import { taskScopeForActiveWorkspace } from "@/hooks/routes/workspace-scope-filter";

import { resolveActiveWorkspaceId, settleRouteQueries } from "./-route-preload";

export async function preloadTasksRoute(queryClient: QueryClient): Promise<void> {
  const [workspaceId, statusResult, workspacesResult] = await Promise.all([
    resolveActiveWorkspaceId(queryClient),
    queryClient.ensureQueryData(statusOptions()).catch(() => null),
    queryClient.ensureQueryData(workspacesListOptions()).catch(() => []),
  ]);
  const activeWorkspace = workspacesResult.find(workspace => workspace.id === workspaceId);
  const scope = taskScopeForActiveWorkspace(activeWorkspace, statusResult?.daemon.user_home_dir);
  if (!scope) {
    return;
  }

  await settleRouteQueries([
    queryClient.ensureInfiniteQueryData(tasksListOptions(defaultTaskCatalogFilter(scope))),
    queryClient.ensureInfiniteQueryData(
      taskInboxOptions({ scope: scope.scope, workspace: scope.workspace, limit: 1 })
    ),
  ]);
}
