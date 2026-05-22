export type WorkspaceScopeFilter = "all" | "global" | "workspace";

export function workspaceFilterForActiveScope(
  scopeFilter: WorkspaceScopeFilter,
  activeWorkspaceId: string | null | undefined
): string | undefined {
  return scopeFilter === "workspace" ? (activeWorkspaceId ?? undefined) : undefined;
}
