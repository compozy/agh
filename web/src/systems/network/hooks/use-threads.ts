import { useQuery } from "@tanstack/react-query";

import { networkThreadDetailOptions, networkThreadsOptions } from "../lib/query-options";
import type { NetworkThreadDetail, NetworkThreadSummary } from "../types";

export interface UseNetworkThreadsResult {
  threads: NetworkThreadSummary[];
  isLoading: boolean;
  error: Error | null;
}

export interface UseNetworkThreadsOptions {
  enabled?: boolean;
  limit?: number;
}

export function useNetworkThreads(
  channel: string | null | undefined,
  options: UseNetworkThreadsOptions = {}
): UseNetworkThreadsResult {
  const enabled = options.enabled !== false && Boolean(channel);
  const channelKey = channel ?? "";
  const query = useQuery(
    networkThreadsOptions(channelKey, options.limit ? { limit: options.limit } : {}, enabled)
  );

  return {
    threads: query.data ?? [],
    isLoading: enabled && query.isLoading,
    error: query.error ?? null,
  };
}

export interface UseNetworkThreadDetailResult {
  thread: NetworkThreadDetail | null;
  isLoading: boolean;
  error: Error | null;
}

export function useNetworkThreadDetail(
  channel: string | null | undefined,
  threadId: string | null | undefined,
  options: { enabled?: boolean } = {}
): UseNetworkThreadDetailResult {
  const enabled = options.enabled !== false && Boolean(channel) && Boolean(threadId);
  const query = useQuery(networkThreadDetailOptions(channel ?? "", threadId ?? "", enabled));

  return {
    thread: query.data ?? null,
    isLoading: enabled && query.isLoading,
    error: query.error ?? null,
  };
}
