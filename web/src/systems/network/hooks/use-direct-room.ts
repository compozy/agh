import { useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";

import { toNetworkPresenceState } from "../lib/network-formatters";
import { networkPeersOptions } from "../lib/query-options";
import type {
  NetworkConversationMessage,
  NetworkDirectRoomDetail,
  NetworkPeerSummary,
  NetworkPresence,
} from "../types";
import { useLastRead } from "./use-last-read";
import { useNetworkDirectDetail } from "./use-directs";
import { useNetworkMessages } from "./use-messages";

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

function presenceFromPeer(peer: NetworkPeerSummary | undefined): NetworkPresence {
  return {
    state: toNetworkPresenceState(peer?.presence_state),
    lastSeenAgeSeconds: peer?.last_seen_age_seconds ?? null,
  };
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
  const workspaceId = detail.direct?.workspace_id ?? "";
  const peersQuery = useQuery(
    networkPeersOptions(workspaceId, channel, Boolean(workspaceId && channel && otherPeerId))
  );
  const presence = useMemo(
    () => presenceFromPeer(peersQuery.data?.find(peer => peer.peer_id === otherPeerId)),
    [otherPeerId, peersQuery.data]
  );
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
