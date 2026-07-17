import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { useActiveWorkspace } from "@/systems/workspace";

import {
  putNetworkCoordination,
  putNetworkCoordinationInvitation,
  type NetworkCoordinationRef,
} from "../adapters/network-coordination-api";
import { networkCoordinationOptions, networkUsageOptions } from "../lib/query-options";
import { networkKeys } from "../lib/query-keys";
import type { NetworkUsageFilters } from "../types";

export function useNetworkCoordination(ref: NetworkCoordinationRef) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  return useQuery(networkCoordinationOptions(workspaceId, ref, Boolean(workspaceId)));
}

export function useNetworkUsage(filters: NetworkUsageFilters = {}) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  return useInfiniteQuery(networkUsageOptions(workspaceId, filters, Boolean(workspaceId)));
}

export function useAcceptNetworkCoordinationInvitation(ref: NetworkCoordinationRef) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (expectedRevision: number) => {
      if (!workspaceId) {
        throw new Error("workspace is required");
      }
      return putNetworkCoordination(workspaceId, {
        scope: ref.scope,
        task_id: ref.taskId,
        run_id: ref.runId,
        enabled: true,
        expected_revision: expectedRevision,
      });
    },
    onSuccess: async coordination => {
      if (!workspaceId) return;
      queryClient.setQueryData(networkKeys.coordination(workspaceId, ref), coordination);
      await queryClient.invalidateQueries({
        queryKey: networkKeys.coordinationRoot(workspaceId),
      });
    },
  });
}

export function useDismissNetworkCoordinationInvitation(ref: NetworkCoordinationRef) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (expectedRevision: number) => {
      if (!workspaceId) {
        throw new Error("workspace is required");
      }
      return putNetworkCoordinationInvitation(workspaceId, {
        scope: ref.scope,
        task_id: ref.taskId,
        run_id: ref.runId,
        dismissed: true,
        expected_revision: expectedRevision,
      });
    },
    onSuccess: async coordination => {
      if (!workspaceId) return;
      queryClient.setQueryData(networkKeys.coordination(workspaceId, ref), coordination);
      await queryClient.invalidateQueries({
        queryKey: networkKeys.coordinationRoot(workspaceId),
      });
    },
  });
}
