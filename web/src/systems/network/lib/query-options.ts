import { queryOptions } from "@tanstack/react-query";

import {
  NetworkApiError,
  getNetworkChannel,
  getNetworkDirectRoom,
  getNetworkPeer,
  getNetworkStatus,
  getNetworkThread,
  getNetworkWork,
  listNetworkChannels,
  listNetworkDirectRooms,
  listNetworkDirectRoomMessages,
  listNetworkPeers,
  listNetworkThreadMessages,
  listNetworkThreads,
  type NetworkDirectsListQuery,
  type NetworkThreadsListQuery,
} from "../adapters/network-api";
import type { NetworkConversationMessagesQuery } from "../types";
import { networkKeys } from "./query-keys";

const STATUS_REFETCH_INTERVAL = 30_000;
const STATUS_STALE_TIME = 10_000;
const CHANNELS_REFETCH_INTERVAL = 30_000;
const LIST_REFETCH_INTERVAL = 15_000;
const LIST_STALE_TIME = 5_000;
const MESSAGES_REFETCH_INTERVAL = 5_000;
const MESSAGES_STALE_TIME = 2_000;
const WORK_REFETCH_INTERVAL = 3_000;
const DEFAULT_TIMELINE_LIMIT = 120;
const DEFAULT_LIST_LIMIT = 50;
const DETAIL_RETRY_LIMIT = 2;

function shouldRetryDetailQuery(failureCount: number, error: Error): boolean {
  if (error instanceof NetworkApiError && error.status >= 400 && error.status < 500) {
    return false;
  }

  return failureCount < DETAIL_RETRY_LIMIT;
}

export function networkStatusOptions() {
  return queryOptions({
    queryKey: networkKeys.status(),
    queryFn: ({ signal }) => getNetworkStatus(signal),
    staleTime: STATUS_STALE_TIME,
    refetchInterval: STATUS_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
  });
}

export function networkChannelsOptions(enabled = true) {
  return queryOptions({
    queryKey: networkKeys.channels(),
    queryFn: ({ signal }) => listNetworkChannels(signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: CHANNELS_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled,
  });
}

export function networkChannelDetailOptions(channel: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.channelDetail(channel),
    queryFn: ({ signal }) => getNetworkChannel(channel, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(channel) && enabled,
  });
}

export function networkThreadsOptions(
  channel: string,
  query: NetworkThreadsListQuery = {},
  enabled = true
) {
  const normalizedQuery = { ...query, limit: query.limit ?? DEFAULT_LIST_LIMIT };
  return queryOptions({
    queryKey: networkKeys.threadsList(channel, normalizedQuery),
    queryFn: ({ signal }) => listNetworkThreads(channel, normalizedQuery, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(channel) && enabled,
  });
}

export function networkThreadDetailOptions(channel: string, threadId: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.threadDetail(channel, threadId),
    queryFn: ({ signal }) => getNetworkThread(channel, threadId, signal),
    retry: shouldRetryDetailQuery,
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(channel) && Boolean(threadId) && enabled,
  });
}

export function networkThreadMessagesOptions(
  channel: string,
  threadId: string,
  query: NetworkConversationMessagesQuery = {},
  enabled = true
) {
  const normalizedQuery = { ...query, limit: query.limit ?? DEFAULT_TIMELINE_LIMIT };
  return queryOptions({
    queryKey: networkKeys.threadMessages(channel, threadId, normalizedQuery),
    queryFn: ({ signal }) => listNetworkThreadMessages(channel, threadId, normalizedQuery, signal),
    staleTime: MESSAGES_STALE_TIME,
    refetchInterval: MESSAGES_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(channel) && Boolean(threadId) && enabled,
  });
}

export function networkDirectsOptions(
  channel: string,
  query: NetworkDirectsListQuery = {},
  enabled = true
) {
  const normalizedQuery = { ...query, limit: query.limit ?? DEFAULT_LIST_LIMIT };
  return queryOptions({
    queryKey: networkKeys.directsList(channel, normalizedQuery),
    queryFn: ({ signal }) => listNetworkDirectRooms(channel, normalizedQuery, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(channel) && enabled,
  });
}

export function networkDirectDetailOptions(channel: string, directId: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.directDetail(channel, directId),
    queryFn: ({ signal }) => getNetworkDirectRoom(channel, directId, signal),
    retry: shouldRetryDetailQuery,
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(channel) && Boolean(directId) && enabled,
  });
}

export function networkDirectMessagesOptions(
  channel: string,
  directId: string,
  query: NetworkConversationMessagesQuery = {},
  enabled = true
) {
  const normalizedQuery = { ...query, limit: query.limit ?? DEFAULT_TIMELINE_LIMIT };
  return queryOptions({
    queryKey: networkKeys.directMessages(channel, directId, normalizedQuery),
    queryFn: ({ signal }) =>
      listNetworkDirectRoomMessages(channel, directId, normalizedQuery, signal),
    staleTime: MESSAGES_STALE_TIME,
    refetchInterval: MESSAGES_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(channel) && Boolean(directId) && enabled,
  });
}

export function networkWorkOptions(workId: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.work(workId),
    queryFn: ({ signal }) => getNetworkWork(workId, signal),
    staleTime: MESSAGES_STALE_TIME,
    refetchInterval: WORK_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workId) && enabled,
  });
}

export function networkPeersOptions(channel?: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.peers(channel),
    queryFn: ({ signal }) => listNetworkPeers(channel, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled,
  });
}

export function networkPeerDetailOptions(peerId: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.peerDetail(peerId),
    queryFn: ({ signal }) => getNetworkPeer(peerId, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(peerId) && enabled,
  });
}
