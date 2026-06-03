import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  deleteWorkspace,
  resolveWorkspace,
  type ResolveWorkspaceParams,
} from "@/systems/workspace/adapters/workspace-api";
import { workspaceKeys } from "@/systems/workspace/lib/query-keys";
import {
  workspaceDetailOptions,
  workspacesListOptions,
} from "@/systems/workspace/lib/query-options";
import type { WorkspacePayload } from "@/systems/workspace/types";

interface UseWorkspaceOptions {
  enabled?: boolean;
}

export function useWorkspaces() {
  return useQuery(workspacesListOptions());
}

export function useWorkspace(workspaceID: string, options?: UseWorkspaceOptions) {
  return useQuery({
    ...workspaceDetailOptions(workspaceID),
    enabled: options?.enabled ?? Boolean(workspaceID),
  });
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

export function useDeleteWorkspace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (workspaceID: string) => deleteWorkspace(workspaceID),
    onSuccess: (_data, workspaceID) => {
      queryClient.setQueryData<WorkspacePayload[]>(workspaceKeys.list(), current =>
        current?.filter(workspace => workspace.id !== workspaceID)
      );
      queryClient.removeQueries({ queryKey: workspaceKeys.detail(workspaceID) });
      return queryClient.invalidateQueries({ queryKey: workspaceKeys.lists() });
    },
  });
}
