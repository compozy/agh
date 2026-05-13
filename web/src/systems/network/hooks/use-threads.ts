import { useQuery } from "@tanstack/react-query";

import { useActiveWorkspace } from "@/systems/workspace";

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
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const enabled = options.enabled !== false && Boolean(channel) && activeWorkspaceId != null;
  const channelKey = channel ?? "";
  const query = useQuery(
    networkThreadsOptions(
      workspaceId,
      channelKey,
      options.limit ? { limit: options.limit } : {},
      enabled
    )
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
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const enabled =
    options.enabled !== false && Boolean(channel) && Boolean(threadId) && activeWorkspaceId != null;
  const query = useQuery(
    networkThreadDetailOptions(workspaceId, channel ?? "", threadId ?? "", enabled)
  );

  return {
    thread: query.data ?? null,
    isLoading: enabled && query.isLoading,
    error: query.error ?? null,
  };
}
