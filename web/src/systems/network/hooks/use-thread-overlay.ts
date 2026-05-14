import { useEffect, useMemo } from "react";
import { useNavigate } from "@tanstack/react-router";

import type { NetworkConversationMessage, NetworkThreadDetail } from "../types";
import { useLastRead } from "./use-last-read";
import { useNetworkMessages } from "./use-messages";
import { useNetworkThreadDetail } from "./use-threads";

export interface UseThreadOverlayArgs {
  workspaceId: string;
  channel: string;
  threadId: string;
  fullPage: boolean;
}

export interface UseThreadOverlayResult {
  detail: NetworkThreadDetail | null;
  isDetailLoading: boolean;
  detailError: Error | null;
  rootMessage: NetworkConversationMessage | null;
  replies: NetworkConversationMessage[];
  replyCount: number;
  isMessagesLoading: boolean;
  messagesError: Error | null;
  lastReadIso: string | null;
}

function pickRoot(
  detail: NetworkThreadDetail | null,
  messages: ReadonlyArray<NetworkConversationMessage>
): NetworkConversationMessage | null {
  if (!detail) {
    return null;
  }
  const rootId = detail.root_message_id;
  if (rootId == null) {
    return messages[0] ?? null;
  }
  return messages.find(message => message.message_id === rootId) ?? null;
}

export function useThreadOverlay({
  workspaceId,
  channel,
  threadId,
  fullPage,
}: UseThreadOverlayArgs): UseThreadOverlayResult {
  const navigate = useNavigate();
  const detail = useNetworkThreadDetail(channel, threadId);
  const messagesQuery = useNetworkMessages({
    channel,
    containerId: threadId,
    enabled: Boolean(detail.thread),
    surface: "thread",
  });
  const { lastReadAt, markRead } = useLastRead();
  const lastReadIso = lastReadAt({ channel, containerId: threadId, surface: "thread" });

  const rootMessage = useMemo(
    () => pickRoot(detail.thread, messagesQuery.messages),
    [detail.thread, messagesQuery.messages]
  );
  const replies = useMemo(
    () =>
      rootMessage
        ? messagesQuery.messages.filter(message => message.message_id !== rootMessage.message_id)
        : [...messagesQuery.messages],
    [messagesQuery.messages, rootMessage]
  );
  const replyCount = Math.max(
    0,
    (detail.thread?.message_count ?? messagesQuery.messages.length) - 1
  );

  useEffect(() => {
    if (fullPage) {
      return undefined;
    }
    function handleKey(event: KeyboardEvent) {
      if (event.key !== "Escape") {
        return;
      }
      if (workspaceId) {
        void navigate({
          params: { workspaceId, channel },
          to: "/network/$workspaceId/$channel/threads",
        });
      }
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [channel, fullPage, navigate, workspaceId]);

  useEffect(() => {
    const lastTimestamp = messagesQuery.messages.at(-1)?.timestamp;
    if (!lastTimestamp) {
      return;
    }
    markRead({ channel, containerId: threadId, surface: "thread" }, lastTimestamp);
  }, [channel, markRead, messagesQuery.messages, threadId]);

  return {
    detail: detail.thread,
    isDetailLoading: detail.isLoading,
    detailError: detail.error,
    rootMessage,
    replies,
    replyCount,
    isMessagesLoading: messagesQuery.isLoading,
    messagesError: messagesQuery.error,
    lastReadIso,
  };
}
