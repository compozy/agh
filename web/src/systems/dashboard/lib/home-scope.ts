import { taskScopeForActiveWorkspace, type ActiveTaskScopeFilter } from "@/systems/tasks";

type ActiveWorkspaceScopeCandidate = Parameters<typeof taskScopeForActiveWorkspace>[0];

export interface HomeScope {
  /** Empty selects the global home scope (whole-system aggregates). */
  workspaceParam: string;
  taskScope: ActiveTaskScopeFilter;
}

/**
 * The home workspace maps to the global scope: the overview endpoint receives
 * no workspace param and aggregates the whole system, mirroring how the tasks
 * surfaces resolve `taskScopeForActiveWorkspace`.
 */
export function homeScopeForActiveWorkspace(
  activeWorkspace: ActiveWorkspaceScopeCandidate | null | undefined,
  userHomeDir: string | undefined
): HomeScope {
  const scope = taskScopeForActiveWorkspace(activeWorkspace, userHomeDir);
  if (!scope || scope.scope === "global") {
    return { workspaceParam: "", taskScope: { scope: "global" } };
  }
  return { workspaceParam: scope.workspace, taskScope: scope };
}
