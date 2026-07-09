import { useContext } from "react";
import type { ThreadMessage } from "@assistant-ui/react";

import { SessionTranscriptThreadContext } from "../lib/session-transcript-thread-context-value";
import type { SessionTranscriptThreadState } from "../lib/session-transcript-thread-context-value";

export function useSessionTranscriptThreadMessages(): readonly ThreadMessage[] {
  return useContext(SessionTranscriptThreadContext).messages;
}

export function useSessionTranscriptThreadState(): SessionTranscriptThreadState {
  return useContext(SessionTranscriptThreadContext);
}
