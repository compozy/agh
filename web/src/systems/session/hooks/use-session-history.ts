import type { UIMessage } from "../types";
import { useSessionTranscript, type UseSessionTranscriptReturn } from "./use-session-transcript";

export interface UseSessionHistoryReturn {
  historyMessages: UIMessage[] | undefined;
  isLoadingHistory: boolean;
  error: Error | null;
}

/**
 * Backwards-compatible wrapper around the transcript query.
 * Historical replay now comes from the canonical transcript endpoint.
 */
export function useSessionHistory(sessionId: string): UseSessionHistoryReturn {
  const { transcriptMessages, isLoadingTranscript, error }: UseSessionTranscriptReturn =
    useSessionTranscript(sessionId);

  return {
    historyMessages: transcriptMessages,
    isLoadingHistory: isLoadingTranscript,
    error,
  };
}
