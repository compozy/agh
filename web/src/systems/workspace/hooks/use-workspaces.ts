import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { resolveWorkspace, type ResolveWorkspaceParams } from "../adapters/workspace-api";
import { workspaceKeys } from "../lib/query-keys";
import { workspacesListOptions } from "../lib/query-options";
import type { WorkspacePayload } from "../types";

export function useWorkspaces() {
  return useQuery(workspacesListOptions());
}

export function useResolveWorkspace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: ResolveWorkspaceParams) => resolveWorkspace(params),
    onSuccess: workspace => {
      queryClient.setQueryData<WorkspacePayload[]>(workspaceKeys.list(), current => {
        const existing = current ?? [];
        return [workspace, ...existing.filter(item => item.id !== workspace.id)];
      });
      queryClient.invalidateQueries({ queryKey: workspaceKeys.lists() });
    },
  });
}
