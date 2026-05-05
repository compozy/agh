import { useEffect, useMemo } from "react";

import type { NetworkConversationMessage, NetworkDirectRoomDetail } from "../types";
import { useLastRead } from "./use-last-read";
import { useNetworkDirectDetail } from "./use-directs";
import { useNetworkMessages } from "./use-messages";
import { useNetworkPresence, type NetworkPresence } from "./use-network-presence";

export interface UseDirectRoomArgs {
  channel: string;
  directId: string;
  selfPeerId?: string;
}

export interface UseDirectRoomResult {
  detail: NetworkDirectRoomDetail | null;
  isDetailLoading: boolean;
  detailError: Error | null;
  messages: NetworkConversationMessage[];
  isMessagesLoading: boolean;
  messagesError: Error | null;
  otherPeerId: string;
  presence: NetworkPresence;
  lastReadIso: string | null;
}

function pickOtherPeerId(detail: NetworkDirectRoomDetail | null, selfPeerId?: string): string {
  if (!detail) {
    return "";
  }
  if (!selfPeerId) {
    return detail.peer_a;
  }
  return detail.peer_a === selfPeerId ? detail.peer_b : detail.peer_a;
}

export function useDirectRoom({
  channel,
  directId,
  selfPeerId,
}: UseDirectRoomArgs): UseDirectRoomResult {
  const detail = useNetworkDirectDetail(channel, directId);
  const messagesQuery = useNetworkMessages({
    channel,
    containerId: directId,
    enabled: Boolean(detail.direct),
    surface: "direct",
  });
  const otherPeerId = useMemo(
    () => pickOtherPeerId(detail.direct, selfPeerId),
    [detail.direct, selfPeerId]
  );
  const presence = useNetworkPresence({ channel, peerId: otherPeerId });
  const { lastReadAt, markRead } = useLastRead();
  const lastReadIso = lastReadAt({ channel, containerId: directId, surface: "direct" });

  useEffect(() => {
    const lastTimestamp = messagesQuery.messages.at(-1)?.timestamp;
    if (!lastTimestamp) {
      return;
    }
    markRead({ channel, containerId: directId, surface: "direct" }, lastTimestamp);
  }, [channel, directId, markRead, messagesQuery.messages]);

  return {
    detail: detail.direct,
    isDetailLoading: detail.isLoading,
    detailError: detail.error,
    messages: messagesQuery.messages,
    isMessagesLoading: messagesQuery.isLoading,
    messagesError: messagesQuery.error,
    otherPeerId,
    presence,
    lastReadIso,
  };
}
