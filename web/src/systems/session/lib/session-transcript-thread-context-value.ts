import { createContext } from "react";
import type { ThreadMessage } from "@assistant-ui/react";

export type SessionTranscriptThreadStatus = "pending" | "error" | "success";

export interface SessionTranscriptThreadState {
  messages: readonly ThreadMessage[];
  status: SessionTranscriptThreadStatus;
  isPending: boolean;
  isError: boolean;
  error: Error | null;
  retry: () => void;
}

const noop = () => {};

const SessionTranscriptThreadContext = createContext<SessionTranscriptThreadState>({
  messages: [],
  status: "success",
  isPending: false,
  isError: false,
  error: null,
  retry: noop,
});

export { SessionTranscriptThreadContext };
