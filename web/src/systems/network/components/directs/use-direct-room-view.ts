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
  workspaceId: string;
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
  workspaceId,
  channel,
  directId,
  selfPeerId: providedSelfPeer,
}: UseDirectRoomViewArgs): UseDirectRoomViewResult {
  const session = useActiveNetworkSession(channel, { workspaceId });
  const selfPeerId = providedSelfPeer ?? session.session?.peerId;
  const room = useDirectRoom({ workspaceId, channel, directId, selfPeerId });
  const openWork = useOpenWork({
    workspaceId,
    channel,
    surface: "direct",
    containerId: directId,
    exactOpenCount: room.detail?.open_work_count,
  });
  const { retry, discard } = useSendNetworkMessage({ workspaceId });

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
