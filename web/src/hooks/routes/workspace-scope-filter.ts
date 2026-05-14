export type WorkspaceScopeFilter = "all" | "global" | "workspace";

export function workspaceFilterForActiveScope(
  scopeFilter: WorkspaceScopeFilter,
  activeWorkspaceId: string | null | undefined
): string | undefined {
  if (scopeFilter === "global") {
    return undefined;
  }

  return activeWorkspaceId ?? undefined;
}
