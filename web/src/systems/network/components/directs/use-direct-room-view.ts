import { useCallback } from "react";

import { useActiveNetworkSession, type ActiveNetworkSession } from "../../hooks/use-active-session";
import { useDirectRoom, type UseDirectRoomResult } from "../../hooks/use-direct-room";
import {
  useSendNetworkMessage,
  type SendNetworkMessageDirectInput,
} from "../../hooks/use-network-actions";
import { useOpenWork, type UseOpenWorkResult } from "../../hooks/use-work";
import type { NetworkConversationMessage } from "../../types";

export interface UseDirectRoomViewArgs {
  channel: string;
  directId: string;
  /** Override the local peer id resolution (used for storybook + tests). */
  selfPeerId?: string;
}

export interface UseDirectRoomViewResult {
  room: UseDirectRoomResult;
  session: ActiveNetworkSession | null;
  disabledReason: string | null;
  openWork: UseOpenWorkResult;
  handleRetry: (message: NetworkConversationMessage) => void;
  handleDiscard: (message: NetworkConversationMessage) => void;
}

/**
 * Composition hook for `<DirectRoom>` - keeps the component under the
 * `compozy-react(max-component-complexity)` cap by holding all state, query,
 * and mutation hooks here.
 */
export function useDirectRoomView({
  channel,
  directId,
  selfPeerId: providedSelfPeer,
}: UseDirectRoomViewArgs): UseDirectRoomViewResult {
  const session = useActiveNetworkSession(channel);
  const selfPeerId = providedSelfPeer ?? session.session?.peerId;
  const room = useDirectRoom({ channel, directId, selfPeerId });
  const openWork = useOpenWork({ channel, surface: "direct", containerId: directId });
  const { retry, discard } = useSendNetworkMessage();

  const buildSendInput = useCallback(
    (message: NetworkConversationMessage): SendNetworkMessageDirectInput | null => {
      if (!session.session) {
        return null;
      }
      const text = message.text ?? "";
      return {
        surface: "direct",
        channel,
        directId,
        sessionId: session.session.sessionId,
        peerFrom: session.session.peerId,
        peerTo: room.otherPeerId,
        text,
        displayName: session.session.displayName,
      };
    },
    [channel, directId, room.otherPeerId, session.session]
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
    room,
    session: session.session,
    disabledReason: session.disabledReason,
    openWork,
    handleRetry,
    handleDiscard,
  };
}
