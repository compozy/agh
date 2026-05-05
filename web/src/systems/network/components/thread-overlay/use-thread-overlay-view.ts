import { useCallback } from "react";

import { useActiveNetworkSession, type ActiveNetworkSession } from "../../hooks/use-active-session";
import {
  useSendNetworkMessage,
  type SendNetworkMessageThreadInput,
} from "../../hooks/use-network-actions";
import { useThreadOverlay, type UseThreadOverlayResult } from "../../hooks/use-thread-overlay";
import { useOpenWork, type UseOpenWorkResult } from "../../hooks/use-work";
import type { NetworkConversationMessage } from "../../types";

export interface UseThreadOverlayViewArgs {
  channel: string;
  threadId: string;
  fullPage: boolean;
}

export interface UseThreadOverlayViewResult {
  overlay: UseThreadOverlayResult;
  session: ActiveNetworkSession | null;
  disabledReason: string | null;
  openWork: UseOpenWorkResult;
  handleRetry: (message: NetworkConversationMessage) => void;
  handleDiscard: (message: NetworkConversationMessage) => void;
}

export function useThreadOverlayView({
  channel,
  threadId,
  fullPage,
}: UseThreadOverlayViewArgs): UseThreadOverlayViewResult {
  const overlay = useThreadOverlay({ channel, fullPage, threadId });
  const session = useActiveNetworkSession(channel);
  const openWork = useOpenWork({ channel, surface: "thread", containerId: threadId });
  const { retry, discard } = useSendNetworkMessage();

  const buildSendInput = useCallback(
    (message: NetworkConversationMessage): SendNetworkMessageThreadInput | null => {
      if (!session.session) {
        return null;
      }
      return {
        surface: "thread",
        channel,
        threadId,
        sessionId: session.session.sessionId,
        peerFrom: session.session.peerId,
        text: message.text ?? "",
        displayName: session.session.displayName,
      };
    },
    [channel, session.session, threadId]
  );

  const handleRetry = useCallback(
    (message: NetworkConversationMessage) => {
      const input = buildSendInput(message);
      if (input == null) {
        return;
      }
      void retry(input, message.message_id);
    },
    [buildSendInput, retry]
  );

  const handleDiscard = useCallback(
    (message: NetworkConversationMessage) => {
      const input = buildSendInput(message);
      if (input == null) {
        return;
      }
      discard(input, message.message_id);
    },
    [buildSendInput, discard]
  );

  return {
    overlay,
    session: session.session,
    disabledReason: session.disabledReason,
    openWork,
    handleRetry,
    handleDiscard,
  };
}
