import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";

import { useActiveWorkspace } from "@/systems/workspace";

import { NetworkApiError, resolveNetworkDirectRoom } from "../adapters/network-api";
import { networkKeys } from "../lib/query-keys";
import type { NetworkResolveDirectRoomResponse } from "../types";
import type { ResolveNetworkDirectRoomInput } from "./network-action-types";

export interface UseResolveNetworkDirectRoomResult {
  resolveRoom: (
    input: ResolveNetworkDirectRoomInput
  ) => Promise<NetworkResolveDirectRoomResponse["direct"]>;
  isResolving: boolean;
  error: Error | null;
}

export function useResolveNetworkDirectRoom(
  options: { workspaceId?: string | null } = {}
): UseResolveNetworkDirectRoomResult {
  const queryClient = useQueryClient();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = options.workspaceId ?? activeWorkspaceId;
  const mutation = useMutation({
    mutationFn: (input: ResolveNetworkDirectRoomInput & { workspaceId: string }) =>
      resolveNetworkDirectRoom(input.workspaceId, input.channel, input.body),
    onSettled: (_data, _error, variables) => {
      void queryClient.invalidateQueries({
        queryKey: networkKeys.directsRoot(variables.workspaceId, variables.channel),
      });
    },
  });
  const resolveRoom = useCallback(
    (input: ResolveNetworkDirectRoomInput) =>
      workspaceId
        ? mutation.mutateAsync({ ...input, workspaceId })
        : Promise.reject(new NetworkApiError("No active workspace selected", 400)),
    [mutation, workspaceId]
  );
  return useMemo(
    () => ({ resolveRoom, isResolving: mutation.isPending, error: mutation.error ?? null }),
    [mutation.error, mutation.isPending, resolveRoom]
  );
}
