import type { ReactNode } from "react";
import type { ThreadMessage } from "@assistant-ui/react";

import { SessionTranscriptThreadContext } from "./session-transcript-thread-context-value";

export function SessionTranscriptThreadProvider({
  children,
  messages,
}: {
  children: ReactNode;
  messages: readonly ThreadMessage[];
}) {
  return (
    <SessionTranscriptThreadContext.Provider value={messages}>
      {children}
    </SessionTranscriptThreadContext.Provider>
  );
}
