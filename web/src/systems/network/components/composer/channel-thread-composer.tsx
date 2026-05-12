import { useNavigate } from "@tanstack/react-router";

import { useCreateNetworkThread } from "../../hooks/use-network-actions";
import { Composer } from "./composer";

export interface ChannelThreadComposerProps {
  channel: string;
  /** The session id used by the operator to author messages in this channel. */
  sessionId: string;
  /** The local peer id (used for the optimistic root message). */
  peerFrom: string;
  displayName?: string;
  /** Disabled when no session is available; describes why so the placeholder can explain. */
  disabledReason?: string;
}

export function ChannelThreadComposer({
  channel,
  sessionId,
  peerFrom,
  displayName,
  disabledReason,
}: ChannelThreadComposerProps) {
  const navigate = useNavigate();
  const { createThread, isCreating } = useCreateNetworkThread();
  const disabled = disabledReason != null;

  const handleSubmit = async ({ text, reset }: { text: string; reset: () => void }) => {
    try {
      const result = await createThread({
        channel,
        sessionId,
        text,
        peerFrom,
        displayName,
      });
      reset();
      void navigate({
        to: "/network/$channel/threads/$threadId",
        params: { channel, threadId: result.threadId },
      });
    } catch {
      // The hook surfaces a Sonner toast on the second collision. Keep the
      // textarea contents so the user can adjust + retry.
    }
  };

  return (
    <Composer
      disabled={disabled}
      disabledReason={disabledReason}
      isSending={isCreating}
      onSubmit={handleSubmit}
      placeholder="Start a new thread..."
      sendLabel={`Send to #${channel}`}
      testIdSuffix="channel-thread"
    />
  );
}
