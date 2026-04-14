import { useQuery } from "@tanstack/react-query";

import {
  networkChannelDetailOptions,
  networkChannelMessagesOptions,
  networkChannelsOptions,
  networkPeerDetailOptions,
  networkPeersOptions,
  networkStatusOptions,
} from "../lib/query-options";

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
  options?: { enabled?: boolean; limit?: number }
) {
  return useQuery(networkChannelMessagesOptions(channel, options?.limit, options?.enabled));
}

export function useNetworkPeers(channel?: string, options?: { enabled?: boolean }) {
  return useQuery(networkPeersOptions(channel, options?.enabled));
}

export function useNetworkPeer(peerId: string, options?: { enabled?: boolean }) {
  return useQuery(networkPeerDetailOptions(peerId, options?.enabled));
}
