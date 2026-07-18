import { useState } from "react";
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

  const transcriptMessagesChanged = runtimeTailState.transcriptMessages !== transcript.messages;
  const runtimeMessagesChanged = runtimeTailState.runtimeMessages !== runtimeMessages;
  const previousRuntimeCount = runtimeTailState.runtimeMessages?.length ?? 0;
  // Local/fast SSE can finish in the same turn it starts, so we never observe
  // isRunning/isOptimistic mid-flight. Latch the tail when the runtime grows past
  // the durable transcript; a later transcript update clears the latch.
  const runtimeGrewWithUnreconciledMessages =
    runtimeMessagesChanged &&
    runtimeMessages.length > previousRuntimeCount &&
    runtimeMessages.some(
      message => !transcript.messages.some(transcriptMessage => transcriptMessage.id === message.id)
    );
  let hasLocalRuntimeTail = runtimeTailState.hasLocalRuntimeTail;
  if (transcriptMessagesChanged) {
    hasLocalRuntimeTail = runtimeIsRunning;
  } else if (
    runtimeIsRunning ||
    (runtimeMessagesChanged && hasOptimisticRuntimeMessage) ||
    runtimeGrewWithUnreconciledMessages
  ) {
    hasLocalRuntimeTail = true;
  }
  if (runtimeMessages.length === 0) {
    hasLocalRuntimeTail = false;
  }
  const nextRuntimeTailState =
    transcriptMessagesChanged ||
    runtimeMessagesChanged ||
    runtimeTailState.hasLocalRuntimeTail !== hasLocalRuntimeTail
      ? { transcriptMessages: transcript.messages, runtimeMessages, hasLocalRuntimeTail }
      : runtimeTailState;
  if (nextRuntimeTailState !== runtimeTailState) {
    setRuntimeTailState(nextRuntimeTailState);
  }

  const includeRuntimeTail =
    runtimeIsRunning ||
    (runtimeMessages.length > 0 &&
      !transcriptMessagesChanged &&
      (nextRuntimeTailState.hasLocalRuntimeTail ||
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
