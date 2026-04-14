import { queryOptions } from "@tanstack/react-query";

import {
  getNetworkChannel,
  getNetworkPeer,
  getNetworkStatus,
  listNetworkChannelMessages,
  listNetworkChannels,
  listNetworkPeers,
} from "../adapters/network-api";
import { networkKeys } from "./query-keys";

const DEFAULT_STALE_TIME = 10_000;
const DEFAULT_REFETCH_INTERVAL = 15_000;
const MESSAGES_REFETCH_INTERVAL = 5_000;

export function networkStatusOptions() {
  return queryOptions({
    queryKey: networkKeys.status(),
    queryFn: ({ signal }) => getNetworkStatus(signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
  });
}

export function networkChannelsOptions() {
  return queryOptions({
    queryKey: networkKeys.channels(),
    queryFn: ({ signal }) => listNetworkChannels(signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
  });
}

export function networkChannelDetailOptions(channel: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.channelDetail(channel),
    queryFn: ({ signal }) => getNetworkChannel(channel, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(channel) && enabled,
  });
}

export function networkChannelMessagesOptions(channel: string, limit = 100, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.channelMessages(channel, limit),
    queryFn: ({ signal }) => listNetworkChannelMessages(channel, limit, signal),
    staleTime: 2_000,
    refetchInterval: MESSAGES_REFETCH_INTERVAL,
    enabled: Boolean(channel) && enabled,
  });
}

export function networkPeersOptions(channel?: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.peers(channel),
    queryFn: ({ signal }) => listNetworkPeers(channel, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled,
  });
}

export function networkPeerDetailOptions(peerId: string, enabled = true) {
  return queryOptions({
    queryKey: networkKeys.peerDetail(peerId),
    queryFn: ({ signal }) => getNetworkPeer(peerId, signal),
    staleTime: DEFAULT_STALE_TIME,
    refetchInterval: DEFAULT_REFETCH_INTERVAL,
    enabled: Boolean(peerId) && enabled,
  });
}
