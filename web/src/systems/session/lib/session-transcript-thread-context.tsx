import { createContext, type ReactNode } from "react";
import type { ThreadMessage } from "@assistant-ui/react";

export const SessionTranscriptThreadContext = createContext<readonly ThreadMessage[]>([]);

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
