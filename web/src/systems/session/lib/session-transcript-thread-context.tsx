import type { ReactNode } from "react";
import type { ThreadMessage } from "@assistant-ui/react";

import {
  SessionTranscriptThreadContext,
  type SessionTranscriptThreadStatus,
} from "./session-transcript-thread-context-value";

const noop = () => {};

export function SessionTranscriptThreadProvider({
  children,
  messages,
  status,
  isPending,
  isError,
  error,
  hasOlder = false,
  isFetchingOlder = false,
  loadOlder = noop,
  retry,
}: {
  children: ReactNode;
  messages: readonly ThreadMessage[];
  status: SessionTranscriptThreadStatus;
  isPending: boolean;
  isError: boolean;
  error: Error | null;
  hasOlder?: boolean;
  isFetchingOlder?: boolean;
  loadOlder?: () => void;
  retry: () => void;
}) {
  return (
    <SessionTranscriptThreadContext.Provider
      value={{
        messages,
        status,
        isPending,
        isError,
        error,
        hasOlder,
        isFetchingOlder,
        loadOlder,
        retry,
      }}
    >
      {children}
    </SessionTranscriptThreadContext.Provider>
  );
}
