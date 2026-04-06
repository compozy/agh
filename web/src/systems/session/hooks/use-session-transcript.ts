import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";

import { sessionTranscriptOptions } from "../lib/query-options";
import { mapTranscriptToMessages } from "../lib/transcript-mapper";
import type { UIMessage } from "../types";

export interface UseSessionTranscriptReturn {
  transcriptMessages: UIMessage[] | undefined;
  isLoadingTranscript: boolean;
  error: Error | null;
}

export function useSessionTranscript(sessionId: string): UseSessionTranscriptReturn {
  const { data, isLoading, error } = useQuery(sessionTranscriptOptions(sessionId));

  const transcriptMessages = useMemo(
    () => (data ? mapTranscriptToMessages(data) : undefined),
    [data]
  );

  return {
    transcriptMessages,
    isLoadingTranscript: isLoading,
    error: error as Error | null,
  };
}
