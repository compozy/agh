import { useQuery } from "@tanstack/react-query";

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
  const enabled = options.enabled !== false && Boolean(channel);
  const channelKey = channel ?? "";
  const query = useQuery(
    networkDirectsOptions(
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
  const enabled = options.enabled !== false && Boolean(channel) && Boolean(directId);
  const query = useQuery(networkDirectDetailOptions(channel ?? "", directId ?? "", enabled));

  return {
    direct: query.data ?? null,
    isLoading: enabled && query.isLoading,
    error: query.error ?? null,
  };
}
