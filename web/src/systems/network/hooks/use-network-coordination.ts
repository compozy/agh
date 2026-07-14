import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { useActiveWorkspace } from "@/systems/workspace";

import {
  getNetworkCoordination,
  getNetworkUsage,
  putNetworkCoordination,
  putNetworkCoordinationInvitation,
} from "../adapters/network-coordination-api";
import { networkKeys } from "../lib/query-keys";

export function networkCoordinationOptions(
  workspaceId: string,
  taskId?: string | null,
  enabled = true
) {
  return queryOptions({
    queryKey: networkKeys.coordination(workspaceId, taskId),
    queryFn: ({ signal }) =>
      getNetworkCoordination(workspaceId, {
        taskId: taskId?.trim() || undefined,
        signal,
      }),
    enabled: Boolean(workspaceId) && enabled,
  });
}

export function networkUsageOptions(workspaceId: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.usage(workspaceId),
    queryFn: ({ signal }) => getNetworkUsage(workspaceId, signal),
    enabled: Boolean(workspaceId) && enabled,
  });
}

export function useNetworkCoordination(taskId?: string | null) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  return useQuery(networkCoordinationOptions(workspaceId, taskId, Boolean(workspaceId)));
}

export function useNetworkUsage() {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  return useQuery(networkUsageOptions(workspaceId, Boolean(workspaceId)));
}

export function useAcceptNetworkCoordinationInvitation(taskId?: string | null) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      if (!workspaceId) {
        throw new Error("workspace is required");
      }
      return putNetworkCoordination(
        workspaceId,
        { enabled: true },
        { taskId: taskId ?? undefined }
      );
    },
    onSuccess: async () => {
      if (!workspaceId) return;
      await queryClient.invalidateQueries({
        queryKey: networkKeys.coordination(workspaceId, taskId),
      });
    },
  });
}

export function useDismissNetworkCoordinationInvitation(
  scope: "workspace" | "task",
  taskId?: string
) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      if (!workspaceId) {
        throw new Error("workspace is required");
      }
      return putNetworkCoordinationInvitation(workspaceId, {
        scope,
        task_id: taskId,
        dismissed: true,
      });
    },
    onSuccess: async () => {
      if (!workspaceId) return;
      await queryClient.invalidateQueries({
        queryKey: networkKeys.coordination(workspaceId, taskId),
      });
    },
  });
}
