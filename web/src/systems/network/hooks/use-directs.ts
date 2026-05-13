import { useQuery } from "@tanstack/react-query";

import { useActiveWorkspace } from "@/systems/workspace";

import { networkDirectDetailOptions, networkDirectsOptions } from "../lib/query-options";
import type { NetworkDirectRoomDetail, NetworkDirectRoomSummary } from "../types";

export interface UseNetworkDirectsResult {
  directs: NetworkDirectRoomSummary[];
  isLoading: boolean;
  error: Error | null;
}

export interface UseNetworkDirectsOptions {
  enabled?: boolean;
  limit?: number;
  peerId?: string;
}

export function useNetworkDirects(
  channel: string | null | undefined,
  options: UseNetworkDirectsOptions = {}
): UseNetworkDirectsResult {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const enabled = options.enabled !== false && Boolean(channel) && activeWorkspaceId != null;
  const channelKey = channel ?? "";
  const query = useQuery(
    networkDirectsOptions(
      workspaceId,
      channelKey,
      {
        ...(options.limit ? { limit: options.limit } : {}),
        ...(options.peerId ? { peer_id: options.peerId } : {}),
      },
      enabled
    )
  );

  return {
    directs: query.data ?? [],
    isLoading: enabled && query.isLoading,
    error: query.error ?? null,
  };
}

export interface UseNetworkDirectDetailResult {
  direct: NetworkDirectRoomDetail | null;
  isLoading: boolean;
  error: Error | null;
}

export function useNetworkDirectDetail(
  channel: string | null | undefined,
  directId: string | null | undefined,
  options: { enabled?: boolean } = {}
): UseNetworkDirectDetailResult {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const enabled =
    options.enabled !== false && Boolean(channel) && Boolean(directId) && activeWorkspaceId != null;
  const query = useQuery(
    networkDirectDetailOptions(workspaceId, channel ?? "", directId ?? "", enabled)
  );

  return {
    direct: query.data ?? null,
    isLoading: enabled && query.isLoading,
    error: query.error ?? null,
  };
}
