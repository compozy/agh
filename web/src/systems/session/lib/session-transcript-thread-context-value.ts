import { createContext } from "react";
import type { ThreadMessage } from "@assistant-ui/react";

const SessionTranscriptThreadContext = createContext<readonly ThreadMessage[]>([]);

export { SessionTranscriptThreadContext };
