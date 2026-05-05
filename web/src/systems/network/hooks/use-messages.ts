import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import { networkDirectMessagesOptions, networkThreadMessagesOptions } from "../lib/query-options";
import type {
  NetworkConversationMessage,
  NetworkConversationMessagesQuery,
  NetworkSurface,
} from "../types";

export interface UseNetworkMessagesResult {
  messages: NetworkConversationMessage[];
  isLoading: boolean;
  isFetching: boolean;
  error: Error | null;
}

export interface UseNetworkMessagesArgs {
  channel: string | null | undefined;
  surface: NetworkSurface | null | undefined;
  containerId: string | null | undefined;
  query?: NetworkConversationMessagesQuery;
  enabled?: boolean;
}

export function useNetworkMessages({
  channel,
  surface,
  containerId,
  query = {},
  enabled = true,
}: UseNetworkMessagesArgs): UseNetworkMessagesResult {
  const isReady = Boolean(channel) && Boolean(surface) && Boolean(containerId) && enabled;

  const threadQuery = useQuery(
    networkThreadMessagesOptions(
      channel ?? "",
      containerId ?? "",
      query,
      isReady && surface === "thread"
    )
  );
  const directQuery = useQuery(
    networkDirectMessagesOptions(
      channel ?? "",
      containerId ?? "",
      query,
      isReady && surface === "direct"
    )
  );

  return useMemo(() => {
    const active = surface === "thread" ? threadQuery : directQuery;
    return {
      messages: active.data ?? [],
      isLoading: isReady && active.isLoading,
      isFetching: active.isFetching,
      error: active.error ?? null,
    };
  }, [surface, threadQuery, directQuery, isReady]);
}
