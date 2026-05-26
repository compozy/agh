import { useContext } from "react";
import type { ThreadMessage } from "@assistant-ui/react";

import { SessionTranscriptThreadContext } from "../lib/session-transcript-thread-context";

export function useSessionTranscriptThreadMessages(): readonly ThreadMessage[] {
  return useContext(SessionTranscriptThreadContext);
}
