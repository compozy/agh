import type { ReactNode } from "react";
import type { ThreadMessage } from "@assistant-ui/react";

import {
  SessionTranscriptThreadContext,
  type SessionTranscriptThreadStatus,
} from "./session-transcript-thread-context-value";

export function SessionTranscriptThreadProvider({
  children,
  messages,
  status,
  isPending,
  isError,
  error,
  retry,
}: {
  children: ReactNode;
  messages: readonly ThreadMessage[];
  status: SessionTranscriptThreadStatus;
  isPending: boolean;
  isError: boolean;
  error: Error | null;
  retry: () => void;
}) {
  return (
    <SessionTranscriptThreadContext.Provider
      value={{ messages, status, isPending, isError, error, retry }}
    >
      {children}
    </SessionTranscriptThreadContext.Provider>
  );
}
