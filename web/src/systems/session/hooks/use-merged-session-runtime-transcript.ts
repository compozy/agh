import { useEffect, useState } from "react";
import { type ThreadMessage, useAuiState } from "@assistant-ui/react";

import { mergeSessionThreadReadModel } from "../lib/session-thread-read-model";
import { useSessionLiveTail, type SessionStreamEventSourceFactory } from "./use-session-live-tail";

interface UseMergedSessionRuntimeTranscriptOptions {
  sessionId: string;
  workspaceId: string;
  eventSourceFactory?: SessionStreamEventSourceFactory;
}

interface RuntimeTailState {
  transcriptMessages: readonly ThreadMessage[] | null;
  runtimeMessages: readonly ThreadMessage[] | null;
  hasLocalRuntimeTail: boolean;
}

export function useMergedSessionRuntimeTranscript({
  sessionId,
  workspaceId,
  eventSourceFactory,
}: UseMergedSessionRuntimeTranscriptOptions) {
  const runtimeMessages = useAuiState(state => state.thread.messages);
  const runtimeIsRunning = useAuiState(state => state.thread.isRunning);
  const transcript = useSessionLiveTail({ sessionId, workspaceId, eventSourceFactory });
  const hasOptimisticRuntimeMessage = runtimeMessages.some(isOptimisticRuntimeMessage);
  const [runtimeTailState, setRuntimeTailState] = useState<RuntimeTailState>({
    transcriptMessages: null,
    runtimeMessages: null,
    hasLocalRuntimeTail: false,
  });

  useEffect(() => {
    setRuntimeTailState(previous => {
      const transcriptMessagesChanged = previous.transcriptMessages !== transcript.messages;
      const runtimeMessagesChanged = previous.runtimeMessages !== runtimeMessages;
      let hasLocalRuntimeTail = previous.hasLocalRuntimeTail;
      if (transcriptMessagesChanged) {
        hasLocalRuntimeTail = runtimeIsRunning;
      } else if (runtimeIsRunning || (runtimeMessagesChanged && hasOptimisticRuntimeMessage)) {
        hasLocalRuntimeTail = true;
      }
      if (runtimeMessages.length === 0) {
        hasLocalRuntimeTail = false;
      }
      if (
        previous.transcriptMessages === transcript.messages &&
        previous.runtimeMessages === runtimeMessages &&
        previous.hasLocalRuntimeTail === hasLocalRuntimeTail
      ) {
        return previous;
      }
      return { transcriptMessages: transcript.messages, runtimeMessages, hasLocalRuntimeTail };
    });
  }, [
    hasOptimisticRuntimeMessage,
    runtimeIsRunning,
    runtimeMessages,
    runtimeMessages.length,
    transcript.messages,
  ]);

  const transcriptMessagesChanged = runtimeTailState.transcriptMessages !== transcript.messages;
  const runtimeMessagesChanged = runtimeTailState.runtimeMessages !== runtimeMessages;
  const includeRuntimeTail =
    runtimeIsRunning ||
    (runtimeMessages.length > 0 &&
      !transcriptMessagesChanged &&
      (runtimeTailState.hasLocalRuntimeTail ||
        (runtimeMessagesChanged && hasOptimisticRuntimeMessage)));
  const messages = mergeSessionThreadReadModel({
    transcriptMessages: transcript.messages,
    runtimeMessages,
    includeRuntimeTail,
  });

  return {
    ...transcript,
    messages,
  };
}

function isOptimisticRuntimeMessage(message: ThreadMessage): boolean {
  return message.metadata?.isOptimistic === true;
}
