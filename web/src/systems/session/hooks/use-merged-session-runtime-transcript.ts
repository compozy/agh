import { useMemo, useRef } from "react";
import { type ThreadMessage, useAuiState } from "@assistant-ui/react";

import { mergeSessionThreadReadModel } from "../lib/session-thread-read-model";
import { useSessionLiveTail, type SessionStreamEventSourceFactory } from "./use-session-live-tail";

interface UseMergedSessionRuntimeTranscriptOptions {
  sessionId: string;
  workspaceId: string;
  eventSourceFactory?: SessionStreamEventSourceFactory;
}

export function useMergedSessionRuntimeTranscript({
  sessionId,
  workspaceId,
  eventSourceFactory,
}: UseMergedSessionRuntimeTranscriptOptions) {
  const runtimeMessages = useAuiState(state => state.thread.messages);
  const runtimeIsRunning = useAuiState(state => state.thread.isRunning);
  const transcript = useSessionLiveTail({ sessionId, workspaceId, eventSourceFactory });
  const transcriptSignatureRef = useRef<string | null>(null);
  const hasLocalRuntimeTailRef = useRef(false);

  const transcriptSignatureChanged = transcriptSignatureRef.current !== transcript.signature;
  if (transcriptSignatureChanged) {
    transcriptSignatureRef.current = transcript.signature;
    hasLocalRuntimeTailRef.current = runtimeIsRunning;
  } else if (runtimeIsRunning || runtimeMessages.some(isOptimisticRuntimeMessage)) {
    hasLocalRuntimeTailRef.current = true;
  }
  if (runtimeMessages.length === 0) {
    hasLocalRuntimeTailRef.current = false;
  }

  const includeRuntimeTail = runtimeIsRunning || hasLocalRuntimeTailRef.current;
  const messages = useMemo(
    () =>
      mergeSessionThreadReadModel({
        transcriptMessages: transcript.messages,
        runtimeMessages,
        includeRuntimeTail,
      }),
    [includeRuntimeTail, runtimeMessages, transcript.messages]
  );

  return {
    ...transcript,
    messages,
  };
}

function isOptimisticRuntimeMessage(message: ThreadMessage): boolean {
  return message.metadata?.isOptimistic === true;
}
