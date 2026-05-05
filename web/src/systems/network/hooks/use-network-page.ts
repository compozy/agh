import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import { networkStatusOptions } from "../lib/query-options";
import type { NetworkChannelSummary, NetworkRecentEntry, NetworkStatus } from "../types";
import { useNetworkChannels } from "./use-channels";
import { useNetworkRecents } from "./use-recents";

export interface UseNetworkPageResult {
  status: NetworkStatus | null;
  isStatusLoading: boolean;
  statusError: Error | null;
  isNetworkEnabled: boolean;
  isNetworkDisabled: boolean;

  channels: NetworkChannelSummary[];
  pinnedChannels: NetworkChannelSummary[];
  unpinnedChannels: NetworkChannelSummary[];
  pinnedChannelIds: ReadonlyArray<string>;
  isPinned: (channel: string) => boolean;
  togglePinned: (channel: string) => void;
  isChannelsLoading: boolean;
  channelsError: Error | null;

  recents: NetworkRecentEntry[];
  isRecentsLoading: boolean;

  firstVisibleChannel: NetworkChannelSummary | null;
}

export function useNetworkPage(): UseNetworkPageResult {
  const statusQuery = useQuery(networkStatusOptions());
  const status = statusQuery.data ?? null;
  const isNetworkEnabled = status?.enabled === true;
  const isNetworkDisabled = status?.enabled === false;

  const channelsResult = useNetworkChannels({ enabled: isNetworkEnabled });
  const recentsResult = useNetworkRecents(channelsResult.channels, {
    enabled: isNetworkEnabled,
  });

  const firstVisibleChannel = useMemo<NetworkChannelSummary | null>(() => {
    if (channelsResult.pinned.length > 0) {
      return channelsResult.pinned[0] ?? null;
    }
    if (channelsResult.unpinned.length > 0) {
      return channelsResult.unpinned[0] ?? null;
    }
    return null;
  }, [channelsResult.pinned, channelsResult.unpinned]);

  return {
    status,
    isStatusLoading: statusQuery.isLoading && !status,
    statusError: statusQuery.error ?? null,
    isNetworkEnabled,
    isNetworkDisabled,

    channels: channelsResult.channels,
    pinnedChannels: channelsResult.pinned,
    unpinnedChannels: channelsResult.unpinned,
    pinnedChannelIds: channelsResult.pinnedIds,
    isPinned: channelsResult.isPinned,
    togglePinned: channelsResult.togglePinned,
    isChannelsLoading: channelsResult.isLoading,
    channelsError: channelsResult.error,

    recents: recentsResult.recents,
    isRecentsLoading: recentsResult.isLoading,

    firstVisibleChannel,
  };
}
