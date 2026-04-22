import { useQuery } from "@tanstack/react-query";

import {
  networkChannelDetailOptions,
  networkChannelMessagesOptions,
  networkChannelsOptions,
  networkPeerDetailOptions,
  networkPeerMessagesOptions,
  networkPeersOptions,
  networkStatusOptions,
} from "../lib/query-options";
import type { NetworkChannelMessagesQuery, NetworkPeerMessagesQuery } from "../types";

export function useNetworkStatus() {
  return useQuery(networkStatusOptions());
}

export function useNetworkChannels(options?: { enabled?: boolean }) {
  return useQuery(networkChannelsOptions(options?.enabled));
}

export function useNetworkChannel(channel: string, options?: { enabled?: boolean }) {
  return useQuery(networkChannelDetailOptions(channel, options?.enabled));
}

export function useNetworkChannelMessages(
  channel: string,
  options?: { enabled?: boolean; query?: NetworkChannelMessagesQuery }
) {
  return useQuery(networkChannelMessagesOptions(channel, options?.query, options?.enabled));
}

export function useNetworkPeers(channel?: string, options?: { enabled?: boolean }) {
  return useQuery(networkPeersOptions(channel, options?.enabled));
}

export function useNetworkPeer(peerId: string, options?: { enabled?: boolean }) {
  return useQuery(networkPeerDetailOptions(peerId, options?.enabled));
}

export function useNetworkPeerMessages(
  peerId: string,
  options?: { enabled?: boolean; query?: NetworkPeerMessagesQuery }
) {
  return useQuery(networkPeerMessagesOptions(peerId, options?.query, options?.enabled));
}
