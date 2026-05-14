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

export function networkChannelsOptions(workspaceId: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.channels(workspaceId),
    queryFn: ({ signal }) => listNetworkChannels(workspaceId, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: CHANNELS_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && enabled,
  });
}

export function networkChannelDetailOptions(workspaceId: string, channel: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.channelDetail(workspaceId, channel),
    queryFn: ({ signal }) => getNetworkChannel(workspaceId, channel, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && Boolean(channel) && enabled,
  });
}

export function networkThreadsOptions(
  workspaceId: string,
  channel: string,
  query: NetworkThreadsListQuery = {},
  enabled = true
) {
  const normalizedQuery = { ...query, limit: query.limit ?? DEFAULT_LIST_LIMIT };
  return queryOptions({
    queryKey: networkKeys.threadsList(workspaceId, channel, normalizedQuery),
    queryFn: ({ signal }) => listNetworkThreads(workspaceId, channel, normalizedQuery, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && Boolean(channel) && enabled,
  });
}

export function networkThreadDetailOptions(
  workspaceId: string,
  channel: string,
  threadId: string,
  enabled = true
) {
  return queryOptions({
    queryKey: networkKeys.threadDetail(workspaceId, channel, threadId),
    queryFn: ({ signal }) => getNetworkThread(workspaceId, channel, threadId, signal),
    retry: shouldRetryDetailQuery,
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && Boolean(channel) && Boolean(threadId) && enabled,
  });
}

export function networkThreadMessagesOptions(
  workspaceId: string,
  channel: string,
  threadId: string,
  query: NetworkConversationMessagesQuery = {},
  enabled = true
) {
  const normalizedQuery = { ...query, limit: query.limit ?? DEFAULT_TIMELINE_LIMIT };
  return queryOptions({
    queryKey: networkKeys.threadMessages(workspaceId, channel, threadId, normalizedQuery),
    queryFn: ({ signal }) =>
      listNetworkThreadMessages(workspaceId, channel, threadId, normalizedQuery, signal),
    staleTime: MESSAGES_STALE_TIME,
    refetchInterval: MESSAGES_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && Boolean(channel) && Boolean(threadId) && enabled,
  });
}

export function networkDirectsOptions(
  workspaceId: string,
  channel: string,
  query: NetworkDirectsListQuery = {},
  enabled = true
) {
  const normalizedQuery = { ...query, limit: query.limit ?? DEFAULT_LIST_LIMIT };
  return queryOptions({
    queryKey: networkKeys.directsList(workspaceId, channel, normalizedQuery),
    queryFn: ({ signal }) => listNetworkDirectRooms(workspaceId, channel, normalizedQuery, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && Boolean(channel) && enabled,
  });
}

export function networkDirectDetailOptions(
  workspaceId: string,
  channel: string,
  directId: string,
  enabled = true
) {
  return queryOptions({
    queryKey: networkKeys.directDetail(workspaceId, channel, directId),
    queryFn: ({ signal }) => getNetworkDirectRoom(workspaceId, channel, directId, signal),
    retry: shouldRetryDetailQuery,
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && Boolean(channel) && Boolean(directId) && enabled,
  });
}

export function networkDirectMessagesOptions(
  workspaceId: string,
  channel: string,
  directId: string,
  query: NetworkConversationMessagesQuery = {},
  enabled = true
) {
  const normalizedQuery = { ...query, limit: query.limit ?? DEFAULT_TIMELINE_LIMIT };
  return queryOptions({
    queryKey: networkKeys.directMessages(workspaceId, channel, directId, normalizedQuery),
    queryFn: ({ signal }) =>
      listNetworkDirectRoomMessages(workspaceId, channel, directId, normalizedQuery, signal),
    staleTime: MESSAGES_STALE_TIME,
    refetchInterval: MESSAGES_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && Boolean(channel) && Boolean(directId) && enabled,
  });
}

export function networkWorkOptions(workspaceId: string, workId: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.work(workspaceId, workId),
    queryFn: ({ signal }) => getNetworkWork(workspaceId, workId, signal),
    staleTime: MESSAGES_STALE_TIME,
    refetchInterval: WORK_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && Boolean(workId) && enabled,
  });
}

export function networkPeersOptions(workspaceId: string, channel?: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.peers(workspaceId, channel),
    queryFn: ({ signal }) => listNetworkPeers(workspaceId, channel, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && enabled,
  });
}

export function networkPeerDetailOptions(workspaceId: string, peerId: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.peerDetail(workspaceId, peerId),
    queryFn: ({ signal }) => getNetworkPeer(workspaceId, peerId, signal),
    staleTime: LIST_STALE_TIME,
    refetchInterval: LIST_REFETCH_INTERVAL,
    refetchOnWindowFocus: true,
    enabled: Boolean(workspaceId) && Boolean(peerId) && enabled,
  });
}
